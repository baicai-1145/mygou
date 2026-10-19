package kugou

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Playlist is a user's collection: 我喜欢, 默认收藏, or one they made.
type Playlist struct {
	ID      string `json:"id"`
	ListID  int64  `json:"listid,omitempty"`
	Name    string `json:"name"`
	Cover   string `json:"cover"`
	Count   int    `json:"count"`
	Creator string `json:"creator"`
}

// UserPlaylists lists the collection lists of the signed in account.
func (c *Client) UserPlaylists(ctx context.Context) ([]Playlist, error) {
	if c.Token == "" || c.UserID == 0 {
		return nil, errors.New("请先登录")
	}
	userid := strconv.FormatInt(c.UserID, 10)
	body, err := json.Marshal(map[string]any{
		"userid":    c.UserID,
		"token":     c.Token,
		"total_ver": 979,
		"type":      2,
		"page":      1,
		"pagesize":  50,
	})
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/v7/get_all_list",
		Params:  map[string]string{"plat": "1", "userid": userid, "token": c.Token},
		Data:    string(body),
		Headers: map[string]string{"x-router": "cloudlist.service.kugou.com", "Content-Type": "application/json"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	data := getMap(res.Body, "data")
	var out []Playlist
	for _, raw := range getSlice(data, "info") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idStr := getString(m, "global_collection_id", "list_create_gid")
		listID := intOf(m, "listid")
		if listID == 0 {
			parts := strings.Split(idStr, "_")
			if len(parts) >= 2 {
				listID, _ = strconv.ParseInt(parts[len(parts)-2], 10, 64)
			}
		}
		p := Playlist{
			ID:      idStr,
			ListID:  listID,
			Name:    getString(m, "name"),
			Cover:   getString(m, "pic", "cover"),
			Creator: getString(m, "list_create_username"),
		}
		if n, ok := toInt(m["count"]); ok {
			p.Count = int(n)
		}
		if p.ID == "" {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// CreatePlaylist makes a new, empty cloud playlist.
func (c *Client) CreatePlaylist(ctx context.Context, name string) error {
	if c.Token == "" || c.UserID == 0 {
		return errors.New("请先登录")
	}
	body, err := json.Marshal(map[string]any{
		"userid": c.UserID, "token": c.Token, "total_ver": 0,
		"name": name, "type": 0, "source": 1, "is_pri": 0,
		"list_create_userid": 0, "list_create_listid": 0,
		"list_create_gid": "", "from_shupinmv": 0,
	})
	if err != nil {
		return err
	}
	res, err := c.do(ctx, Request{
		Method: "POST",
		Path:   "/v5/add_list",
		Params: map[string]string{
			"last_time": strconv.FormatInt(time.Now().Unix(), 10),
			"last_area": "gztx",
			"userid":    strconv.FormatInt(c.UserID, 10),
			"token":     c.Token,
		},
		Data:    string(body),
		Headers: map[string]string{"Content-Type": "application/json", "x-router": "cloudlist.service.kugou.com"},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return res.Err()
	}
	return nil
}

// AddToPlaylist adds a song to one of the account's lists, by listid.
func (c *Client) AddToPlaylist(ctx context.Context, listid int64, name, hash, albumID, mixSongID string) error {
	if c.Token == "" || c.UserID == 0 {
		return errors.New("请先登录")
	}
	userid := strconv.FormatInt(c.UserID, 10)
	var album, mix int64
	album, _ = strconv.ParseInt(albumID, 10, 64)
	mix, _ = strconv.ParseInt(mixSongID, 10, 64)
	body, err := json.Marshal(map[string]any{
		"userid":      c.UserID,
		"token":       c.Token,
		"listid":      listid,
		"list_ver":    0,
		"type":        0,
		"slow_upload": 1,
		"scene":       "false;null",
		"data": []map[string]any{{
			"number":    1,
			"name":      name,
			"hash":      hash,
			"size":      0,
			"sort":      0,
			"timelen":   0,
			"bitrate":   0,
			"album_id":  album,
			"mixsongid": mix,
		}},
	})
	if err != nil {
		return err
	}
	res, err := c.do(ctx, Request{
		Method: "POST",
		Path:   "/v6/add_song",
		Params: map[string]string{
			"last_time": strconv.FormatInt(time.Now().Unix(), 10),
			"last_area": "gztx",
			"userid":    userid,
			"token":     c.Token,
		},
		Data:    string(body),
		Headers: map[string]string{"Content-Type": "application/json", "x-router": "cloudlist.service.kugou.com"},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return res.Err()
	}
	return nil
}

// CloudTracks lists a cloud playlist with the per-entry fileid that
// delete_songs needs; the public list API does not expose it.
func (c *Client) CloudTracks(ctx context.Context, listid int64) ([]Track, error) {
	raw, err := c.CloudFileIDs(ctx, listid)
	if err != nil {
		return nil, err
	}
	var d struct {
		Data struct {
			Info []struct {
				FileID  int64  `json:"fileid"`
				AudioID int64  `json:"audio_id"`
				Hash    string `json:"hash"`
				Name    string `json:"name"`
				Timelen int    `json:"timelen"`
				Cover   string `json:"cover"`
				AlbumID any    `json:"album_id"`
				Singer  []struct {
					Name string `json:"name"`
				} `json:"singerinfo"`
				Album struct {
					ID   any    `json:"album_id"`
					Name string `json:"album_name"`
				} `json:"albuminfo"`
				TransParam struct {
					UnionCover string `json:"union_cover"`
				} `json:"trans_param"`
			} `json:"info"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, err
	}
	var tracks []Track
	albumIDs := map[string]bool{}
	for _, it := range d.Data.Info {
		name, singer := it.Name, ""
		if i := strings.LastIndex(name, "."); i > 0 {
			name = name[:i]
		}
		if parts := strings.SplitN(name, " - ", 2); len(parts) == 2 {
			singer, name = parts[0], parts[1]
		}
		if singer == "" && len(it.Singer) > 0 {
			singer = it.Singer[0].Name
		}
		albumID := it.Album.ID
		if albumID == nil {
			albumID = it.AlbumID
		}
		albStr := fmt.Sprint(albumID)
		if albStr != "" && albStr != "0" {
			albumIDs[albStr] = true
		}
		cov := it.Cover
		if cov == "" {
			cov = it.TransParam.UnionCover
		}
		tracks = append(tracks, Track{
			Hash:         strings.ToLower(it.Hash),
			Name:         name,
			Singer:       singer,
			Album:        it.Album.Name,
			AlbumID:      albStr,
			AlbumAudioID: "0",
			Cover:        cov,
			MixSongID:    strconv.FormatInt(it.AudioID, 10),
			FileID:       strconv.FormatInt(it.FileID, 10),
			Duration:     it.Timelen / 1000,
		})
	}

	missing := 0
	for _, t := range tracks {
		if t.Cover == "" && t.AlbumID != "" && t.AlbumID != "0" {
			missing++
		}
	}
	if missing > 0 && len(albumIDs) > 0 {
		metas := c.albums(ctx, albumIDs)
		for i := range tracks {
			if tracks[i].Cover == "" {
				if meta, ok := metas[tracks[i].AlbumID]; ok && meta.Cover != "" {
					tracks[i].Cover = meta.Cover
					if tracks[i].Album == "" {
						tracks[i].Album = meta.Name
					}
					if tracks[i].Singer == "" {
						tracks[i].Singer = meta.Artist
					}
				}
			}
		}
	}
	// Invert so newest added tracks appear first, matching public list order.
	for i, j := 0, len(tracks)-1; i < j; i, j = i+1, j-1 {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	}
	return tracks, nil
}

// DeleteSongRaw removes one entry from a cloud list and returns the raw
// response, so callers can inspect error_code instead of guessing.
func (c *Client) DeleteSongRaw(ctx context.Context, listid, fileid int64) (string, error) {
	body, err := json.Marshal(map[string]any{
		"listid": listid, "userid": c.UserID, "type": 0, "token": c.Token,
		"list_ver": 0, "data": []map[string]any{{"fileid": fileid}},
	})
	if err != nil {
		return "", err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/cloudlist.service/v4/delete_songs",
		Data:    string(body),
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		return "", err
	}
	return string(res.Raw), nil
}

// CloudFileIDs calls the app cloud list endpoint, which (unlike the public
// list API) is expected to return the per-entry fileid used by delete_songs.
func (c *Client) CloudFileIDs(ctx context.Context, listid int64) (string, error) {
	body, err := json.Marshal(map[string]any{
		"listid": listid, "userid": c.UserID, "type": 0, "page": 1,
		"pagesize": 30, "area_code": 1, "allplatform": 1,
		"show_cover": 1, "token": c.Token,
	})
	if err != nil {
		return "", err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/cloudlist.service/v4/get_list_all_file_v3",
		Data:    string(body),
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		return "", err
	}
	return string(res.Raw), nil
}

// PlaylistTracks returns one page of a list's songs, with album covers and
// names filled in from the album service: the list API itself sends neither.
func (c *Client) PlaylistTracks(ctx context.Context, id string, page, size int) (Playlist, []Track, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 60
	}
	res, err := c.do(ctx, Request{
		Method: "GET",
		Path:   "/pubsongs/v2/get_other_list_file_nofilt",
		Params: map[string]string{
			"area_code":            "1",
			"begin_idx":            strconv.Itoa((page - 1) * size),
			"plat":                 "1",
			"type":                 "1",
			"mode":                 "1",
			"personal_switch":      "1",
			"extend_fields":        "abtags,hot_cmt,popularization",
			"pagesize":             strconv.Itoa(size),
			"global_collection_id": id,
		},
		CacheTTL: time.Minute,
	})
	if err != nil {
		return Playlist{}, nil, err
	}
	if !res.OK() {
		return Playlist{}, nil, res.Err()
	}
	data := getMap(res.Body, "data")

	list := Playlist{ID: id}
	if info := getMap(data, "list_info"); info != nil {
		list.Name = getString(info, "name")
		if strings.HasSuffix(list.Name, "喜欢的音乐") {
			list.Name = "我喜欢"
		}
		list.Cover = getString(info, "pic", "cover")
		if n, ok := toInt(info["count"]); ok {
			list.Count = int(n)
		}
	}

	var tracks []Track
	albumIDs := map[string]bool{}
	for _, raw := range getSlice(data, "songs") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cov := getString(m, "cover", "pic", "img")
		if cov == "" {
			if tp, ok := m["trans_param"].(map[string]any); ok {
				cov = getString(tp, "union_cover")
			}
		}
		if cov == "" {
			if singers := getSlice(m, "singerinfo"); len(singers) > 0 {
				if sm, ok := singers[0].(map[string]any); ok {
					cov = getString(sm, "avatar")
				}
			}
		}
		t := Track{
			Hash:         strings.ToLower(getString(m, "hash")),
			Name:         getString(m, "name", "remark"),
			AlbumID:      getString(m, "album_id"),
			AlbumAudioID: getString(m, "album_audio_id"),
			MixSongID:    getString(m, "add_mixsongid", "mixsongid"),
			Cover:        cov,
		}
		if ms, ok := toInt(m["timelen"]); ok {
			t.Duration = int(ms / 1000)
		}
		// The list sends "Singer - Title" as the name, singer included.
		if parts := strings.SplitN(t.Name, " - ", 2); len(parts) == 2 {
			t.Singer, t.Name = parts[0], parts[1]
		}
		if t.Hash == "" {
			continue
		}
		if t.AlbumID != "" {
			albumIDs[t.AlbumID] = true
		}
		tracks = append(tracks, t)
	}

	if len(tracks) > 0 {
		metas := c.albums(ctx, albumIDs)
		for i := range tracks {
			if meta, ok := metas[tracks[i].AlbumID]; ok {
				if tracks[i].Cover == "" && meta.Cover != "" {
					tracks[i].Cover = meta.Cover
				}
				if tracks[i].Album == "" {
					tracks[i].Album = meta.Name
				}
				if tracks[i].Singer == "" {
					tracks[i].Singer = meta.Artist
				}
			}
		}
		if list.Cover == "" && len(tracks) > 0 {
			list.Cover = tracks[0].Cover
		}
	}
	return list, tracks, nil
}

type albumMeta struct {
	Name   string
	Cover  string
	Artist string
}

// albums fetches covers and names for up to twenty albums at a time.
func (c *Client) albums(ctx context.Context, ids map[string]bool) map[string]albumMeta {
	out := map[string]albumMeta{}
	var batch []string
	flush := func() {
		if len(batch) == 0 {
			return
		}
		var data []map[string]any
		for _, id := range batch {
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				data = append(data, map[string]any{"album_id": n})
			}
		}
		batch = nil
		body, _ := json.Marshal(map[string]any{
			"data":   data,
			"is_buy": 0,
			"fields": "album_id,album_name,sizable_cover,author_name",
		})
		res, err := c.do(ctx, Request{
			Method:   "POST",
			Path:     "/kmr/v2/albums",
			Data:     string(body),
			Headers:  map[string]string{"x-router": "openapi.kugou.com", "kg-tid": "255", "Content-Type": "application/json"},
			CacheTTL: time.Hour,
		})
		if err != nil || !res.OK() {
			return
		}
		for _, raw := range getSlice(res.Body, "data") {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			out[getString(m, "album_id")] = albumMeta{
				Name:   getString(m, "album_name"),
				Cover:  getString(m, "sizable_cover"),
				Artist: getString(m, "author_name"),
			}
		}
	}
	for id := range ids {
		batch = append(batch, id)
		if len(batch) == 20 {
			flush()
		}
	}
	flush()
	return out
}
