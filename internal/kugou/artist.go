package kugou

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Artist is a singer as the store describes them.
type Artist struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Intro  string `json:"intro"`
	Fans   int    `json:"fans"`
	Songs  int    `json:"songs"`
	Albums int    `json:"albums"`
}

// Artist reads a singer's profile.
func (c *Client) Artist(ctx context.Context, id string) (Artist, error) {
	body, err := json.Marshal(map[string]any{"author_id": id})
	if err != nil {
		return Artist{}, err
	}
	res, err := c.do(ctx, Request{
		Method:   "POST",
		Path:     "/kmr/v3/author",
		Data:     string(body),
		Headers:  map[string]string{"x-router": "openapi.kugou.com", "kg-tid": "36", "Content-Type": "application/json"},
		CacheTTL: 10 * time.Minute,
	})
	if err != nil {
		return Artist{}, err
	}
	if !res.OK() {
		return Artist{}, res.Err()
	}
	d := getMap(res.Body, "data")
	return Artist{
		ID:     getString(d, "author_id"),
		Name:   getString(d, "author_name"),
		Avatar: getString(d, "sizable_avatar"),
		Intro:  getString(d, "intro"),
		Fans:   int(intOf(d, "fansnums")),
		Songs:  int(intOf(d, "song_count")),
		Albums: int(intOf(d, "album_count")),
	}, nil
}

// SearchArtists looks singers up by name.
func (c *Client) SearchArtists(ctx context.Context, keyword string, limit int) ([]Artist, error) {
	if limit < 1 {
		limit = 5
	}
	res, err := c.do(ctx, Request{
		BaseURL: "https://complexsearch.kugou.com",
		Path:    "/v1/search/author",
		Headers: map[string]string{"x-router": "complexsearch.kugou.com"},
		Params: map[string]string{
			"albumhide":    "0",
			"iscorrection": "1",
			"keyword":      keyword,
			"nocollect":    "0",
			"page":         "1",
			"pagesize":     strconv.Itoa(limit),
			"platform":     "AndroidFilter",
		},
		CacheTTL: 5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Artist
	for _, raw := range getSlice(getMap(res.Body, "data"), "lists") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Artist{
			ID:     getString(m, "AuthorId"),
			Name:   getString(m, "AuthorName"),
			Avatar: getString(m, "Avatar"),
			Fans:   int(intOf(m, "FansNum")),
			Songs:  int(intOf(m, "AudioCount")),
			Albums: int(intOf(m, "AlbumCount")),
		})
	}
	return out, nil
}

// ArtistSongs returns a page of a singer's songs, hottest first.
func (c *Client) ArtistSongs(ctx context.Context, id string, page, size int) ([]Track, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 30
	}
	now := time.Now().Unix()
	body, err := json.Marshal(map[string]any{
		"appid":      AppID,
		"clientver":  ClientVer,
		"mid":        c.Device.MID,
		"clienttime": now,
		"key":        ParamsKey(strconv.FormatInt(now, 10)),
		"author_id":  id,
		"pagesize":   size,
		"page":       page,
		"sort":       1,
		"area_code":  "all",
	})
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/kmr/v1/audio_group/author",
		Data:    string(body),
		Headers: map[string]string{"x-router": "openapi.kugou.com", "kg-tid": "220", "Content-Type": "application/json"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Track
	for _, raw := range getSlice(res.Body, "data") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := Track{
			Hash:         strings.ToLower(getString(m, "hash")),
			Name:         getString(m, "audio_name"),
			Singer:       getString(m, "author_name"),
			SingerID:     getString(m, "author_id"),
			Album:        getString(m, "album_name"),
			AlbumID:      getString(m, "album_id"),
			AlbumAudioID: getString(m, "album_audio_id"),
			MixSongID:    getString(m, "audio_id"),
		}
		if trans := getMap(m, "trans_param"); trans != nil {
			t.Cover = getString(trans, "union_cover")
		}
		if ms := intOf(m, "timelength"); ms > 0 {
			t.Duration = int(ms / 1000)
		}
		if t.Hash == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// AlbumSongs returns a page of an album's songs.
func (c *Client) AlbumSongs(ctx context.Context, id string, page, size int) ([]Track, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 30
	}
	if size > 50 {
		// The endpoint answers "invalid param" above fifty.
		size = 50
	}
	body, err := json.Marshal(map[string]any{
		"album_id": id,
		"is_buy":   "",
		"page":     page,
		"pagesize": size,
	})
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/v1/album_audio/lite",
		Data:    string(body),
		Headers: map[string]string{"x-router": "openapi.kugou.com", "kg-tid": "255", "Content-Type": "application/json"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, fmt.Errorf("%w: %s", res.Err(), snippet(res.Raw))
	}
	var out []Track
	for _, raw := range getSlice(getMap(res.Body, "data"), "songs") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		base := getMap(m, "base")
		info := getMap(m, "audio_info")
		album := getMap(m, "album_info")
		t := Track{
			Hash:         strings.ToLower(getString(info, "hash", "hash_128")),
			Name:         getString(base, "audio_name"),
			Singer:       getString(base, "author_name"),
			Album:        getString(album, "album_name"),
			AlbumID:      getString(base, "album_id"),
			AlbumAudioID: getString(base, "album_audio_id"),
			MixSongID:    getString(base, "audio_id"),
			Cover:        getString(album, "cover"),
		}
		if authors := getSlice(m, "authors"); len(authors) > 0 {
			if a, ok := authors[0].(map[string]any); ok {
				t.SingerID = getString(a, "author_id")
			}
		}
		if ms := intOf(info, "duration"); ms > 0 {
			t.Duration = int(ms / 1000)
		}
		if t.Hash == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
