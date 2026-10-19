// Package service is what the window calls: search, queue, playback,
// login. It is bound to the frontend with mygo.Bind.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mygou/internal/kugou"
	"mygou/internal/player"
)

// App is the app's state and the frontend's API.
type App struct {
	recent        []kugou.Track
	profileName   string
	profileAvatar string
	vipAt         time.Time
	daily         []kugou.Track
	dailyDay      string

	client *kugou.Client
	engine *player.Engine
	dir    string

	mu       sync.Mutex
	queue    []kugou.Track
	index    int
	lastPlay bool
	mode     string

	stop chan struct{}
}

// Account is the logged in user, as the UI shows it.
type Account struct {
	LoggedIn bool   `json:"loggedIn"`
	UserID   int64  `json:"userID"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
}

type accountFile struct {
	Token  string `json:"token"`
	UserID int64  `json:"userID"`
	Name   string `json:"name"`
}

// New returns the service, loading the account saved in dir.
func New(c *kugou.Client, e *player.Engine, dir string) *App {
	a := &App{client: c, engine: e, dir: dir, stop: make(chan struct{})}
	a.loadAccount()
	a.loadRecent()
	a.loadVIP()
	a.loadQueue()
	go a.watch()
	go a.vipLoop()
	return a
}

func (a *App) loadAccount() {
	if err := a.client.LoadAccount(a.dir); err != nil {
		log.Println("load account:", err)
	}
}

// saveAccount keeps the login across launches.
func (a *App) saveAccount(name string) error {
	f := accountFile{Token: a.client.Token, UserID: a.client.UserID, Name: name}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.dir, "account.json"), data, 0o600)
}

// Account reports the logged in user.
func (a *App) Account() Account {
	a.mu.Lock()
	if a.client.Token == "" {
		a.mu.Unlock()
		return Account{}
	}
	id, name, avatar := a.client.UserID, a.profileName, a.profileAvatar
	a.mu.Unlock()
	if name == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if p, err := a.client.Profile(ctx); err == nil {
			name, avatar = p.Name, p.Avatar
			a.mu.Lock()
			a.profileName, a.profileAvatar = name, avatar
			a.mu.Unlock()
		}
	}
	if name == "" {
		name = "酷狗用户"
	}
	// The page is served over https-capable scheme; keep images off http.
	avatar = strings.Replace(avatar, "http://", "https://", 1)
	if os.Getenv("MYGOU_LOG") == "1" {
		log.Printf("account: name=%q avatar=%q", name, avatar)
	}
	return Account{LoggedIn: true, UserID: id, Name: name, Avatar: avatar}
}

// Logout forgets the login.
func (a *App) Logout() {
	a.client.Token = ""
	a.client.UserID = 0
	a.mu.Lock()
	a.profileName, a.profileAvatar = "", ""
	a.mu.Unlock()
	os.Remove(filepath.Join(a.dir, "account.json"))
}

// Search searches songs.
func (a *App) Search(keyword string) ([]kugou.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return a.client.Search(ctx, keyword, 1, 30)
}

// loadVIP reads when the daily VIP was last claimed.
func (a *App) loadVIP() {
	data, err := os.ReadFile(filepath.Join(a.dir, "vip.json"))
	if err != nil {
		return
	}
	var f struct {
		Last int64 `json:"last"`
	}
	if json.Unmarshal(data, &f) == nil && f.Last > 0 {
		a.vipAt = time.Unix(f.Last, 0)
	}
}

// claimVIP claims the concept edition's daily VIP hours, at most once
// every three hours, as the official client does.
func (a *App) ClaimVIP() (kugou.VIPClaim, error) {
	a.mu.Lock()
	if a.client.Token == "" || time.Since(a.vipAt) < 3*time.Hour {
		a.mu.Unlock()
		return kugou.VIPClaim{}, nil
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	claim, err := a.client.ClaimVIP(ctx)
	if err != nil {
		return claim, err
	}
	a.mu.Lock()
	a.vipAt = time.Now()
	data, _ := json.Marshal(map[string]int64{"last": a.vipAt.Unix()})
	_ = os.WriteFile(filepath.Join(a.dir, "vip.json"), data, 0o600)
	a.mu.Unlock()
	log.Printf("daily vip: %s", claim.String())
	return claim, nil
}

func (a *App) vipLoop() {
	// Claim once at startup, then whenever three hours have passed.
	first := time.AfterFunc(4*time.Second, func() {
		if _, err := a.ClaimVIP(); err != nil {
			log.Println("daily vip:", err)
		}
	})
	defer first.Stop()
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-a.stop:
			return
		case <-t.C:
			if _, err := a.ClaimVIP(); err != nil {
				log.Println("daily vip:", err)
			}
		}
	}
}

// QueueState is the play queue and the index of the current song.
type QueueState struct {
	Tracks []kugou.Track `json:"tracks"`
	Index  int           `json:"index"`
}

// Queue returns the current queue.
func (a *App) Queue() QueueState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return QueueState{
		Tracks: append([]kugou.Track(nil), a.queue...),
		Index:  a.index,
	}
}

// PlayIndex plays the queue entry at index.
func (a *App) PlayIndex(index int) error {
	a.mu.Lock()
	if index < 0 || index >= len(a.queue) {
		a.mu.Unlock()
		return nil
	}
	a.index = index
	t := a.queue[index]
	a.mu.Unlock()
	a.saveQueue()
	a.play(t)
	return nil
}

// CacheStats is the size of the playback cache.
type CacheStats struct {
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
}

func (a *App) cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "mygou")
}

// CacheInfo reports how much disk the playback cache uses.
func (a *App) CacheInfo() (CacheStats, error) {
	dir := a.cacheDir()
	var st CacheStats
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		st.Bytes += info.Size()
		st.Files++
		return nil
	})
	if os.IsNotExist(err) {
		return CacheStats{}, nil
	}
	return st, err
}

// ClearCache deletes the playback cache. The current song keeps playing
// from its open file.
func (a *App) ClearCache() error {
	dir := a.cacheDir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			os.RemoveAll(filepath.Join(dir, e.Name()))
			continue
		}
		os.Remove(filepath.Join(dir, e.Name()))
	}
	return nil
}

// DownloadCurrent copies the playing song into ~/Music/mygou.
func (a *App) DownloadCurrent() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), "Music", "mygou")
	return a.engine.SaveCurrent(dir)
}

// SearchMVs looks music videos up by keyword.
func (a *App) SearchMVs(keyword string) ([]kugou.MV, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.SearchMVs(ctx, keyword, 20)
}

// PlayMV resolves a playable URL for a music video.
func (a *App) PlayMV(id, quality string) (kugou.MV, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return a.client.MVPlay(ctx, id, quality)
}

// CommentPage is one page of song comments.
type CommentPage struct {
	Comments []kugou.Comment `json:"comments"`
	Total    int             `json:"total"`
}

// Comments returns a song's comments by its mix song id.
func (a *App) Comments(mixID string, page int) (CommentPage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	list, total, err := a.client.Comments(ctx, mixID, page, 30)
	if err != nil {
		return CommentPage{}, err
	}
	return CommentPage{Comments: list, Total: total}, nil
}

// PersonalFM returns a fresh batch of private-radio songs.
func (a *App) PersonalFM(count int) ([]kugou.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if count <= 0 {
		count = 20
	}
	return a.client.PersonalFM(ctx, count)
}

// PlaylistTags returns the playlist plaza categories.
func (a *App) PlaylistTags() ([]kugou.Tag, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.PlaylistTags(ctx)
}

// PlaylistPlaza lists public playlists for a category.
func (a *App) PlaylistPlaza(category string, page int) (kugou.PlazaPage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return a.client.PlaylistPlaza(ctx, category, page, 24)
}

// SearchPlaylists looks public playlists up by keyword.
func (a *App) SearchPlaylists(keyword string) ([]kugou.PlaylistHit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.SearchPlaylists(ctx, keyword, 1, 24)
}

// SearchAlbums looks albums up by keyword.
func (a *App) SearchAlbums(keyword string) ([]kugou.AlbumHit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.SearchAlbums(ctx, keyword, 1, 24)
}

// SearchArtists looks singers up by keyword.
func (a *App) SearchArtists(keyword string) ([]kugou.Artist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.SearchArtists(ctx, keyword, 24)
}

// Artist reads a singer's profile.
func (a *App) Artist(id string) (kugou.Artist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.Artist(ctx, id)
}

// FindArtist resolves a singer name to the first matching id.
func (a *App) FindArtist(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	list, err := a.client.SearchArtists(ctx, name, 3)
	if err != nil {
		return "", err
	}
	if len(list) == 0 || list[0].ID == "" {
		return "", errors.New("没有找到这位歌手")
	}
	return list[0].ID, nil
}

// ArtistSongs returns a page of a singer's songs.
func (a *App) ArtistSongs(id string, page int) ([]kugou.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return a.client.ArtistSongs(ctx, id, page, 300)
}

// AlbumSongs returns a page of an album's songs.
func (a *App) AlbumSongs(id string, page int) ([]kugou.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if os.Getenv("MYGOU_LOG") == "1" {
		log.Printf("album songs: id=%q page=%d", id, page)
	}
	tracks, err := a.client.AlbumSongs(ctx, id, page, 100)
	if err != nil {
		log.Println("album songs:", err)
	}
	return tracks, err
}

// Ranks lists the store's charts.
func (a *App) Ranks() ([]kugou.Rank, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.client.Ranks(ctx)
}

// RankSongs returns one page of a chart.
func (a *App) RankSongs(id string, page int) ([]kugou.Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return a.client.RankSongs(ctx, id, page, 100)
}

// Daily returns the account's daily recommendation, cached for the day.
func (a *App) Daily() ([]kugou.Track, error) {
	day := time.Now().Format("2006-01-02")
	a.mu.Lock()
	if a.dailyDay == day && len(a.daily) > 0 {
		out := append([]kugou.Track(nil), a.daily...)
		a.mu.Unlock()
		return out, nil
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tracks, err := a.client.DailyRecommend(ctx)
	if err != nil {
		log.Println("daily:", err)
		return nil, err
	}
	log.Printf("daily: %d tracks", len(tracks))
	a.mu.Lock()
	a.daily, a.dailyDay = tracks, day
	a.mu.Unlock()
	return tracks, nil
}

// PlaylistPage is a list and its songs, as the playlist view needs them.
type PlaylistPage struct {
	Playlist kugou.Playlist `json:"playlist"`
	Tracks   []kugou.Track  `json:"tracks"`
}

// favoriteList returns 我喜欢 and its songs.
func (a *App) favoriteList(ctx context.Context) (kugou.Playlist, []kugou.Track, error) {
	lists, err := a.client.UserPlaylists(ctx)
	if err != nil {
		return kugou.Playlist{}, nil, err
	}
	for _, l := range lists {
		if strings.Contains(l.Name, "喜欢") {
			var listid int64
			parts := strings.Split(l.ID, "_")
			if len(parts) >= 2 {
				listid, _ = strconv.ParseInt(parts[len(parts)-2], 10, 64)
			}
			if listid != 0 {
				if tracks, err := a.client.CloudTracks(ctx, listid); err == nil && len(tracks) > 0 {
					if l.Cover == "" && len(tracks) > 0 {
						l.Cover = tracks[0].Cover
					}
					return l, tracks, nil
				}
			}
			return a.client.PlaylistTracks(ctx, l.ID, 1, 300)
		}
	}
	return kugou.Playlist{}, nil, errors.New("请先登录")
}

// FavoriteState reports whether a song is in 我喜欢.
func (a *App) FavoriteState(hash string) (bool, error) {
	if hash == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, tracks, err := a.favoriteList(ctx)
	if err != nil {
		return false, err
	}
	for _, t := range tracks {
		if t.Hash == hash {
			return true, nil
		}
	}
	return false, nil
}

// ToggleFavorite adds or removes the playing song from 我喜欢.
func (a *App) ToggleFavorite() (bool, error) {
	a.mu.Lock()
	t := a.engine.State().Track
	a.mu.Unlock()
	return a.ToggleFavoriteTrack(t)
}

// ToggleFavoriteTrack toggles any track, not only the playing one.
func (a *App) ToggleFavoriteTrack(t kugou.Track) (bool, error) {
	if t.Hash == "" {
		return false, errors.New("没有在播放的歌")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	list, tracks, err := a.favoriteList(ctx)
	if err != nil {
		return false, err
	}
	var listid int64
	parts := strings.Split(list.ID, "_")
	if len(parts) >= 2 {
		listid, _ = strconv.ParseInt(parts[len(parts)-2], 10, 64)
	}
	if listid == 0 {
		return false, errors.New("找不到我喜欢歌单")
	}
	for _, old := range tracks {
		if old.Hash == t.Hash {
			id, _ := strconv.ParseInt(old.FileID, 10, 64)
			if id == 0 {
				return true, errors.New("这首歌缺少歌单条目 id")
			}
			if _, err := a.client.DeleteSongRaw(ctx, listid, id); err != nil {
				return true, err
			}
			return false, nil
		}
	}
	if err := a.client.AddToPlaylist(ctx, listid, t.Name+" - "+t.Singer, t.Hash, t.AlbumID, t.MixSongID); err != nil {
		return false, err
	}
	return true, nil
}

// Favorite adds the current track to 我喜欢, creating nothing if the
// account has no such list.
func (a *App) Favorite() error {
	a.mu.Lock()
	t := a.engine.State().Track
	a.mu.Unlock()
	if t.Hash == "" {
		return errors.New("没有在播放的歌")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lists, err := a.client.UserPlaylists(ctx)
	if err != nil {
		return err
	}
	for _, l := range lists {
		if strings.Contains(l.Name, "喜欢") {
			// collection_3_<userid>_<listid>_0
			var listid int64
			parts := strings.Split(l.ID, "_")
			if len(parts) >= 2 {
				listid, _ = strconv.ParseInt(parts[len(parts)-2], 10, 64)
			}
			if listid == 0 {
				return errors.New("找不到我喜欢歌单")
			}
			return a.client.AddToPlaylist(ctx, listid, t.Name+" - "+t.Singer, t.Hash, t.AlbumID, t.MixSongID)
		}
	}
	return errors.New("请先登录")
}

// AddTrackToPlaylist adds a track to a specific user playlist by list ID.
func (a *App) AddTrackToPlaylist(listid int64, t kugou.Track) error {
	if t.Hash == "" {
		a.mu.Lock()
		t = a.engine.State().Track
		a.mu.Unlock()
	}
	if t.Hash == "" {
		return errors.New("没有指定歌曲")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	name := t.Name
	if t.Singer != "" {
		name = t.Name + " - " + t.Singer
	}
	return a.client.AddToPlaylist(ctx, listid, name, t.Hash, t.AlbumID, t.MixSongID)
}

// CreatePlaylistAndAdd creates a new playlist and immediately adds a track to it.
func (a *App) CreatePlaylistAndAdd(name string, t kugou.Track) ([]kugou.Playlist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("歌单名不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := a.client.CreatePlaylist(ctx, name); err != nil {
		return nil, err
	}
	lists, err := a.client.UserPlaylists(ctx)
	if err != nil {
		return nil, err
	}
	if t.Hash == "" {
		a.mu.Lock()
		t = a.engine.State().Track
		a.mu.Unlock()
	}
	if t.Hash != "" {
		for _, l := range lists {
			if l.Name == name && l.ListID > 0 {
				songName := t.Name
				if t.Singer != "" {
					songName = t.Name + " - " + t.Singer
				}
				_ = a.client.AddToPlaylist(ctx, l.ListID, songName, t.Hash, t.AlbumID, t.MixSongID)
				break
			}
		}
	}
	return lists, nil
}

// Playlists lists the signed in account's lists.
func (a *App) Playlists() ([]kugou.Playlist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	lists, err := a.client.UserPlaylists(ctx)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		if lists[i].Cover == "" && lists[i].ListID > 0 {
			if tracks, err := a.client.CloudTracks(ctx, lists[i].ListID); err == nil && len(tracks) > 0 {
				lists[i].Cover = tracks[0].Cover
			}
		}
	}
	return lists, nil
}

// PlaylistTracks loads a list's first page of songs.
func (a *App) PlaylistTracks(id string) (PlaylistPage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	list, tracks, err := a.client.PlaylistTracks(ctx, id, 1, 300)
	return PlaylistPage{Playlist: list, Tracks: tracks}, err
}

var statusBarLyricHook func(on bool)

func RegisterStatusBarLyricHook(fn func(on bool)) {
	statusBarLyricHook = fn
}

// StatusBarLyric reports if status bar lyrics are enabled.
func (a *App) StatusBarLyric() bool {
	b, err := os.ReadFile(filepath.Join(a.dir, "lyric_statusbar"))
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(b)) != "0"
}

// SetStatusBarLyric enables or disables status bar lyrics.
func (a *App) SetStatusBarLyric(on bool) {
	val := "1"
	if !on {
		val = "0"
	}
	_ = os.WriteFile(filepath.Join(a.dir, "lyric_statusbar"), []byte(val), 0o644)
	if statusBarLyricHook != nil {
		statusBarLyricHook(on)
	}
}

// Lyric is a song's lyrics as text; KRC marks the timed app format.
type Lyric struct {
	Text string `json:"text"`
	KRC  bool   `json:"krc"`
}

// Lyric finds and downloads the lyrics of a song, prioritizing candidates with translations.
func (a *App) Lyric(hash, name string, duration int) (Lyric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var allCands []kugou.LyricCandidate
	if cands, err := a.client.SearchLyric(ctx, hash, name, duration); err == nil {
		allCands = append(allCands, cands...)
	}
	if name != "" {
		if cands, err := a.client.SearchLyric(ctx, "", name, 0); err == nil {
			allCands = append(allCands, cands...)
		}
	}

	var fallback Lyric
	for _, c := range allCands {
		text, err := a.client.GetLyric(ctx, c.ID, c.AccessKey)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		krc := strings.Contains(text, "[id:$") || strings.Contains(text, "[offset:")
		if strings.Contains(text, "[language:") {
			return Lyric{Text: text, KRC: krc}, nil
		}
		if fallback.Text == "" {
			fallback = Lyric{Text: text, KRC: krc}
		}
	}
	if fallback.Text != "" {
		return fallback, nil
	}
	return Lyric{}, errors.New("没有找到歌词")
}

// PlayQueue plays tracks[index], and remembers the queue for next/prev.
func (a *App) PlayQueue(tracks []kugou.Track, index int) error {
	if len(tracks) == 0 {
		return nil
	}
	if index < 0 || index >= len(tracks) {
		index = 0
	}
	a.mu.Lock()
	a.queue = append([]kugou.Track(nil), tracks...)
	a.index = index
	t := a.queue[index]
	a.mu.Unlock()
	a.saveQueue()
	a.play(t)
	return nil
}

func (a *App) play(t kugou.Track) {
	log.Printf("play %s - %s", t.Name, t.Singer)
	a.mu.Lock()
	a.addRecent(t)
	a.mu.Unlock()
	a.engine.Play(t)
}

// CreatePlaylist makes a new playlist and returns the refreshed list.
func (a *App) CreatePlaylist(name string) ([]kugou.Playlist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := a.client.CreatePlaylist(ctx, name); err != nil {
		return nil, err
	}
	return a.client.UserPlaylists(ctx)
}

// Recent returns the tracks played lately, newest first.
func (a *App) Recent() []kugou.Track {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]kugou.Track(nil), a.recent...)
}

func (a *App) loadRecent() {
	data, err := os.ReadFile(filepath.Join(a.dir, "recent.json"))
	if err != nil {
		return
	}
	var f struct {
		Tracks []kugou.Track `json:"tracks"`
	}
	if json.Unmarshal(data, &f) == nil {
		a.recent = f.Tracks
	}
}

// addRecent records a play, newest first and without duplicates.
func (a *App) addRecent(t kugou.Track) {
	if t.Hash == "" {
		return
	}
	out := []kugou.Track{t}
	for _, r := range a.recent {
		if r.Hash != t.Hash && len(out) < 30 {
			out = append(out, r)
		}
	}
	a.recent = out
	data, err := json.Marshal(map[string]any{"tracks": out})
	if err == nil {
		_ = os.WriteFile(filepath.Join(a.dir, "recent.json"), data, 0o644)
	}
}

type queueFile struct {
	Tracks []kugou.Track `json:"tracks"`
	Index  int           `json:"index"`
	Mode   string        `json:"mode"`
}

func (a *App) loadQueue() {
	data, err := os.ReadFile(filepath.Join(a.dir, "queue.json"))
	if err != nil {
		return
	}
	var q queueFile
	if json.Unmarshal(data, &q) == nil && len(q.Tracks) > 0 {
		a.mu.Lock()
		a.queue = q.Tracks
		if q.Index >= 0 && q.Index < len(q.Tracks) {
			a.index = q.Index
		} else {
			a.index = 0
		}
		if q.Mode != "" {
			a.mode = q.Mode
		}
		track := a.queue[a.index]
		a.mu.Unlock()

		a.engine.SetTrack(track)
	}
}

func (a *App) saveQueue() {
	a.mu.Lock()
	q := queueFile{
		Tracks: a.queue,
		Index:  a.index,
		Mode:   a.mode,
	}
	a.mu.Unlock()

	data, err := json.Marshal(q)
	if err == nil {
		_ = os.WriteFile(filepath.Join(a.dir, "queue.json"), data, 0o644)
	}
}

// Next plays the next track of the queue.
// SetMode switches the play mode: "list" repeats the queue, "one" repeats
// the current song, "shuffle" picks at random.
func (a *App) SetMode(mode string) string {
	switch mode {
	case "one", "shuffle", "list":
	default:
		mode = "list"
	}
	a.mu.Lock()
	a.mode = mode
	a.mu.Unlock()
	a.saveQueue()
	return mode
}

// Mode returns the current play mode.
func (a *App) Mode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mode == "" {
		return "list"
	}
	return a.mode
}

func (a *App) Next() error {
	a.mu.Lock()
	if len(a.queue) == 0 {
		a.mu.Unlock()
		return nil
	}
	switch a.mode {
	case "shuffle":
		if len(a.queue) > 1 {
			n := a.index
			for n == a.index {
				n = rand.Intn(len(a.queue))
			}
			a.index = n
		}
	default:
		a.index = (a.index + 1) % len(a.queue)
	}
	t := a.queue[a.index]
	a.mu.Unlock()
	a.saveQueue()
	a.play(t)
	return nil
}

// autoNext advances when a song finishes: repeat-one replays it.
func (a *App) autoNext() {
	a.mu.Lock()
	if a.mode == "one" && len(a.queue) > 0 {
		t := a.queue[a.index]
		a.mu.Unlock()
		a.play(t)
		return
	}
	a.mu.Unlock()
	a.Next()
}

// Prev plays the previous track of the queue.
func (a *App) Prev() error {
	a.mu.Lock()
	if len(a.queue) == 0 {
		a.mu.Unlock()
		return nil
	}
	a.index = (a.index - 1 + len(a.queue)) % len(a.queue)
	t := a.queue[a.index]
	a.mu.Unlock()
	a.saveQueue()
	a.play(t)
	return nil
}

// Toggle pauses or resumes.
func (a *App) Toggle() {
	if os.Getenv("MYGOU_LOG") == "1" {
		log.Printf("toggle (was playing=%v)", a.engine.State().Playing)
	}
	if a.engine.HasPlayer() {
		a.engine.Toggle()
		return
	}
	// Player not started yet (e.g. freshly launched with restored queue): start playing current track
	a.mu.Lock()
	if len(a.queue) > 0 && a.index >= 0 && a.index < len(a.queue) {
		t := a.queue[a.index]
		a.mu.Unlock()
		a.play(t)
		return
	}
	a.mu.Unlock()
}

// Play resumes if paused; media keys and the panel use it.
func (a *App) Play() {
	if a.engine.HasPlayer() {
		if !a.engine.State().Playing {
			a.engine.Toggle()
		}
		return
	}
	a.Toggle()
}

// Pause pauses if playing.
func (a *App) Pause() {
	if a.engine.State().Playing {
		a.engine.Toggle()
	}
}

// Seek moves the current track, in seconds.
func (a *App) Seek(seconds float64) { a.engine.Seek(seconds) }

// SetVolume sets the gain, 0 to 1, and returns it.
func (a *App) SetVolume(v float64) float64 {
	a.engine.SetVolume(v)
	return a.engine.State().Volume
}

// SetQuality sets the preferred quality: 128, 320 or flac.
func (a *App) SetQuality(q string) string {
	a.engine.SetQuality(q)
	return q
}

// State is the player, for the UI to poll twice a second.
func (a *App) State() player.State { return a.engine.State() }

// LoginQRCode starts a login: draw the URL as a QR code, then poll
// LoginQRCheck with the key.
func (a *App) LoginQRCode() (kugou.QRLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if os.Getenv("MYGOU_LOGIN_APP") == "1" {
		return a.client.QRKeyApp(ctx)
	}
	return a.client.QRKey(ctx)
}

// LoginQRCheck reports whether the phone scanned and confirmed, and logs
// in when it did.
func (a *App) LoginQRCheck(key string) (kugou.QRLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var q kugou.QRLogin
	var err error
	if os.Getenv("MYGOU_LOGIN_APP") == "1" {
		q, err = a.client.QRCheckApp(ctx, key)
	} else {
		q, err = a.client.QRCheck(ctx, key)
	}
	if err != nil {
		return q, err
	}
	if q.Status == 4 && q.Token != "" {
		log.Printf("logged in, userid %d", q.UserID)
		if err := a.saveAccount(""); err != nil {
			return q, err
		}
	}
	return q, nil
}

// watch advances the queue when a track ends. It runs for the app's life.
func (a *App) watch() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-a.stop:
			return
		case <-t.C:
			s := a.engine.State()
			ended := a.lastPlay && !s.Playing && !s.Loading && s.Duration > 0 && s.Position >= s.Duration-1.5
			a.lastPlay = s.Playing || s.Loading
			if ended {
				log.Printf("watch: track ended, next (pos=%.1f dur=%.1f)", s.Position, s.Duration)
				a.autoNext()
			}
		}
	}
}

// Close stops the engine's watching.
func (a *App) Close() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
}

var (
	isFullScreenHook     func() bool
	toggleFullScreenHook func() bool
)

// RegisterFullScreenHooks binds the window's full-screen state to App.
func RegisterFullScreenHooks(isFS func() bool, toggleFS func() bool) {
	isFullScreenHook = isFS
	toggleFullScreenHook = toggleFS
}

// IsFullScreen reports whether the main window is in native full screen.
func (a *App) IsFullScreen() bool {
	if isFullScreenHook != nil {
		return isFullScreenHook()
	}
	return false
}

// ToggleFullScreen enters or exits native full screen and returns the new state.
func (a *App) ToggleFullScreen() bool {
	if toggleFullScreenHook != nil {
		return toggleFullScreenHook()
	}
	return false
}

// ExitFullScreen exits native full screen if currently in full screen.
func (a *App) ExitFullScreen() bool {
	if a.IsFullScreen() {
		return a.ToggleFullScreen()
	}
	return false
}
