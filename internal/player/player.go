// Package player turns a track into sound: it resolves a playable URL,
// keeps the file in a disk cache, and decodes it with package audio.
package player

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/flac"   // FLAC, the lossless tier
	_ "github.com/egoist/mygo/audio/mp3"    // MP3
	_ "github.com/egoist/mygo/audio/vorbis" // Ogg Vorbis, the concept edition's 128/320

	"mygou/internal/kugou"
)

// Track is what the UI shows and what the engine plays.
type Track = kugou.Track

// State is a snapshot of the engine for the UI: one call per second is
// enough, the UI interpolates the position itself.
type State struct {
	Track    Track   `json:"track"`
	Playing  bool    `json:"playing"`
	Loading  bool    `json:"loading"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   float64 `json:"volume"`
	Quality  string  `json:"quality"`
	Error    string  `json:"error,omitempty"`
}

// Engine owns the one player of the app. It is safe for concurrent use.
type Engine struct {
	client  *kugou.Client
	dir     string
	quality string
	http    *http.Client

	mu         sync.Mutex
	player     *audio.Player
	track      Track
	playing    bool
	loading    bool
	volume     float64
	qualityNow string
	err        string
	// generation counts Play calls, so a slow download of a track the user
	// skipped does not start playing over the new one.
	gen    int
	cancel context.CancelFunc
	path   string
	stream io.Closer
}

// New returns an engine caching tracks in dir.
func New(c *kugou.Client, dir string, quality string) *Engine {
	if quality == "" {
		quality = "320"
	}
	return &Engine{
		client:  c,
		dir:     filepath.Join(dir, "tracks"),
		quality: quality,
		http:    &http.Client{Timeout: 60 * time.Second},
		volume:  1,
	}
}

// SetQuality sets the quality the engine asks for: "128", "320", "flac".
func (e *Engine) SetQuality(q string) {
	e.mu.Lock()
	e.quality = q
	e.mu.Unlock()
}

// Play resolves and plays t, in the background. The call returns at once;
// the previous track's download, if still running, is canceled.
func (e *Engine) Play(t Track) {
	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
	}
	e.cancel = cancel
	e.gen++
	gen := e.gen
	e.track = t
	e.loading = true
	e.err = ""
	if e.player != nil {
		e.player.Close()
		e.player = nil
	}
	if e.stream != nil {
		e.stream.Close()
		e.stream = nil
	}
	e.playing = false
	e.mu.Unlock()

	go e.play(ctx, t, gen)
}

func (e *Engine) play(ctx context.Context, t Track, gen int) {
	var (
		r       io.ReadSeeker
		closer  io.Closer
		label   string
		quality string
		path    string
	)
	if os.Getenv("MYGOU_STREAM") != "0" {
		if su, q, p, err := e.resolveTrack(ctx, t); err == nil {
			if st, err := os.Stat(p); err == nil && st.Size() > 0 {
				f, err := os.Open(p)
				if err != nil {
					e.fail(gen, err)
					return
				}
				r, closer, label, quality, path = f, f, filepath.Base(p), q, p
			} else if hs, err := newHTTPStream(ctx, su.URLs[0]); err == nil {
				r, closer, label, quality = hs, hs, filepath.Base(p), q
				go func(dst string, urls []string) {
					_ = os.MkdirAll(e.dir, 0o755)
					tmp := dst + ".part"
					if err := e.download(context.Background(), urls, tmp); err == nil {
						_ = os.Rename(tmp, dst)
					}
				}(p, su.URLs)
			}
		}
	}
	if r == nil {
		p, q, err := e.fetch(ctx, t, gen)
		if err != nil {
			e.fail(gen, err)
			return
		}
		f, err := os.Open(p)
		if err != nil {
			e.fail(gen, err)
			return
		}
		r, closer, label, quality, path = f, f, filepath.Base(p), q, p
	}

	e.mu.Lock()
	if gen != e.gen {
		e.mu.Unlock()
		closer.Close()
		return
	}
	volume := e.volume
	e.mu.Unlock()

	log.Printf("player: open %s (%s)", label, quality)
	p, err := audio.OpenStream(r)
	if err != nil {
		closer.Close()
		e.fail(gen, fmt.Errorf("decode %s: %w", label, err))
		return
	}
	p.SetVolume(volume)

	e.mu.Lock()
	if gen != e.gen {
		e.mu.Unlock()
		p.Close()
		closer.Close()
		return
	}
	if e.stream != nil {
		e.stream.Close()
	}
	e.player = p
	e.stream = closer
	e.path = path
	e.loading = false
	e.playing = true
	e.qualityNow = quality
	e.mu.Unlock()
	p.Play()
}

func (e *Engine) fail(gen int, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if gen != e.gen {
		return
	}
	e.loading = false
	e.playing = false
	e.err = err.Error()
}

// fetch returns a cached file, or resolves and downloads one. The quality
// asked for is used when the account may play it, the best free one
// otherwise, so a free account still hears something.
func (e *Engine) fetch(ctx context.Context, t Track, gen int) (string, string, error) {
	e.mu.Lock()
	want := e.quality
	e.mu.Unlock()

	// Each quality has its own hash, and the URL endpoint only returns the
	// file for that hash; the plain hash is the 128 kbps copy.
	qhash := map[string]string{}
	if priv, err := e.client.Privilege(ctx, []string{t.Hash}, []string{t.AlbumID}, nil); err == nil && len(priv) > 0 {
		for _, q := range priv[0].Qualities {
			if q.Hash != "" {
				qhash[q.Quality] = strings.ToLower(q.Hash)
			}
		}
	}
	hashFor := func(q string) string {
		if h := qhash[q]; h != "" {
			return h
		}
		return t.Hash
	}

	// Ask for the wanted quality first and fall back down the tiers: the
	// privilege API under-reports what a paid account may play, while the
	// URL endpoint is authoritative.
	order := []string{want}
	switch want {
	case "high":
		order = append(order, "flac")
		fallthrough
	case "flac":
		order = append(order, "320")
	}
	order = append(order, "128")

	fetchURL := func(h, q string, trial bool) (kugou.SongURL, error) {
		r, err := e.client.GetSongURL(ctx, h, t.AlbumID, t.AlbumAudioID, q, trial)
		if (err != nil || len(r.URLs) == 0) && t.AlbumAudioID != "" && t.AlbumAudioID != "0" {
			if r2, err2 := e.client.GetSongURL(ctx, h, t.AlbumID, "0", q, trial); err2 == nil && len(r2.URLs) > 0 {
				return r2, nil
			}
		}
		if (err != nil || len(r.URLs) == 0) && t.AlbumID != "" && t.AlbumID != "0" {
			if r3, err3 := e.client.GetSongURL(ctx, h, "0", "0", q, trial); err3 == nil && len(r3.URLs) > 0 {
				return r3, nil
			}
		}
		return r, err
	}

	var su kugou.SongURL
	quality := ""
	trial := false
	for _, q := range order {
		r, err := fetchURL(hashFor(q), q, false)
		if err != nil || len(r.URLs) == 0 {
			continue
		}
		su, quality = r, q
		break
	}
	if quality == "" {
		// A paid song the account may not play whole: ask for the trial
		// excerpt, as the concept edition itself does when logged out.
		for _, q := range order {
			r, err := fetchURL(hashFor(q), q, true)
			if err != nil || len(r.URLs) == 0 {
				continue
			}
			su, quality, trial = r, q, true
			break
		}
	}
	if quality == "" {
		return "", "", errors.New("no playable url")
	}
	ext := su.ExtName
	if ext == "" {
		ext = "mp3"
	}
	name := hashFor(quality) + "-" + quality + "." + ext
	if trial {
		// A trial excerpt and the whole song differ: never share a file.
		name = hashFor(quality) + "-" + quality + "-trial." + ext
	}
	path := filepath.Join(e.dir, name)
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		// A file of another size is a stale one, e.g. a trial excerpt kept
		// under the whole song's name before that distinction existed.
		if su.Size == 0 || abs64(st.Size()-su.Size) <= su.Size/20 {
			return path, quality, nil
		}
		os.Remove(path)
	}
	if err := os.MkdirAll(e.dir, 0o755); err != nil {
		return "", "", err
	}
	tmp := path + ".part"
	if err := e.download(ctx, su.URLs, tmp); err != nil {
		os.Remove(tmp)
		return "", "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", "", err
	}
	e.pruneCache(path)
	return path, quality, nil
}

// cacheLimit caps the playback cache; older files are dropped first.
const cacheLimit = 1 << 30 // 1 GiB

func (e *Engine) pruneCache(keep string) {
	entries, err := os.ReadDir(e.dir)
	if err != nil {
		return
	}
	type entry struct {
		path string
		size int64
		mod  time.Time
	}
	var files []entry
	var total int64
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		files = append(files, entry{filepath.Join(e.dir, ent.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	limit := int64(cacheLimit)
	if v := os.Getenv("MYGOU_CACHE_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int64(n) << 20
		}
	}
	if total <= limit {
		return
	}
	e.mu.Lock()
	current := e.path
	e.mu.Unlock()
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for _, f := range files {
		if total <= limit*8/10 {
			break
		}
		if f.path == current || f.path == keep {
			continue
		}
		if err := os.Remove(f.path); err == nil {
			total -= f.size
		}
	}
	log.Printf("player: cache pruned to %d MB", total>>20)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (e *Engine) download(ctx context.Context, urls []string, path string) error {
	var lastErr error
	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		res, err := e.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
			res.Body.Close()
			lastErr = fmt.Errorf("download: HTTP %d", res.StatusCode)
			continue
		}
		f, err := os.Create(path)
		if err != nil {
			res.Body.Close()
			return err
		}
		_, err = io.Copy(f, res.Body)
		f.Close()
		res.Body.Close()
		if err != nil {
			lastErr = err
			os.Remove(path)
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("download failed")
	}
	return lastErr
}

// Toggle pauses or resumes.
// SaveCurrent copies the playing file into dir and returns its path. The
// file is already in the disk cache, so this is a local copy.
func (e *Engine) SaveCurrent(dir string) (string, error) {
	e.mu.Lock()
	path := e.path
	t := e.track
	e.mu.Unlock()
	if path == "" {
		return "", errors.New("没有正在播放的歌曲")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := strings.TrimSpace(t.Singer + " - " + t.Name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	if name == "" {
		name = "track"
	}
	dest := filepath.Join(dir, name+filepath.Ext(path))
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return dest, nil
}

// SetTrack populates the current track metadata without starting playback.
func (e *Engine) SetTrack(t Track) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.track = t
	e.playing = false
	e.loading = false
}

// HasPlayer reports whether audio playback has been started.
func (e *Engine) HasPlayer() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.player != nil
}

func (e *Engine) Toggle() {
	e.mu.Lock()
	p := e.player
	playing := e.playing
	e.mu.Unlock()
	if p == nil {
		return
	}
	if playing {
		p.Pause()
	} else {
		p.Play()
	}
	e.mu.Lock()
	e.playing = !playing
	e.mu.Unlock()
}

// Seek moves the current track.
func (e *Engine) Seek(seconds float64) {
	e.mu.Lock()
	p := e.player
	e.mu.Unlock()
	if p != nil && seconds >= 0 {
		p.Seek(time.Duration(seconds * float64(time.Second)))
	}
}

// SetVolume sets the gain, 0 to 1.
func (e *Engine) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	e.mu.Lock()
	e.volume = v
	p := e.player
	e.mu.Unlock()
	if p != nil {
		p.SetVolume(v)
	}
}

// State returns a snapshot for the UI.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := State{
		Track:   e.track,
		Playing: e.playing,
		Loading: e.loading,
		Volume:  e.volume,
		Quality: e.qualityNow,
		Error:   e.err,
	}
	if e.player != nil {
		s.Position = e.player.Position().Seconds()
		s.Duration = e.player.Duration().Seconds()
		// A stream that ended is no longer playing.
		if e.playing && !e.player.Playing() {
			e.playing = false
			s.Playing = false
		}
	}
	if s.Duration == 0 {
		s.Duration = float64(e.track.Duration)
	}
	return s
}
