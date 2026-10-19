package kugou

import (
	"context"
	"strconv"
	"time"
)

// AlbumHit is an album as search returns it.
type AlbumHit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Cover    string `json:"cover"`
	Songs    int    `json:"songs"`
	Publish  string `json:"publish"`
	Language string `json:"language"`
}

// PlaylistHit is a public playlist as search returns it.
type PlaylistHit struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cover   string `json:"cover"`
	Creator string `json:"creator"`
	Count   int    `json:"count"`
	Plays   int    `json:"plays"`
}

// SearchAlbums looks albums up by keyword.
func (c *Client) SearchAlbums(ctx context.Context, keyword string, page, size int) ([]AlbumHit, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	res, err := c.do(ctx, Request{
		BaseURL: "https://complexsearch.kugou.com",
		Path:    "/v1/search/album",
		Headers: map[string]string{"x-router": "complexsearch.kugou.com"},
		Params: map[string]string{
			"keyword":      keyword,
			"page":         strconv.Itoa(page),
			"pagesize":     strconv.Itoa(size),
			"platform":     "AndroidFilter",
			"iscorrection": "1",
			"category":     "1",
			"sorttype":     "0",
			"searchsong":   "0",
		},
		CacheTTL: 5 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []AlbumHit
	for _, raw := range getSlice(getMap(res.Body, "data"), "lists") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := getString(m, "albumid")
		if id == "" || id == "0" {
			continue
		}
		out = append(out, AlbumHit{
			ID:       id,
			Name:     getString(m, "albumname"),
			Artist:   getString(m, "singer"),
			Cover:    getString(m, "img"),
			Songs:    int(intOf(m, "songcount")),
			Publish:  getString(m, "publish_time"),
			Language: getString(m, "language"),
		})
	}
	return out, nil
}

// SearchPlaylists looks public playlists up by keyword.
func (c *Client) SearchPlaylists(ctx context.Context, keyword string, page, size int) ([]PlaylistHit, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	res, err := c.do(ctx, Request{
		BaseURL: "https://complexsearch.kugou.com",
		Path:    "/v1/search/special",
		Headers: map[string]string{"x-router": "complexsearch.kugou.com"},
		Params: map[string]string{
			"albumhide":    "0",
			"iscorrection": "1",
			"keyword":      keyword,
			"nocollect":    "0",
			"page":         strconv.Itoa(page),
			"pagesize":     strconv.Itoa(size),
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
	var out []PlaylistHit
	for _, raw := range getSlice(getMap(res.Body, "data"), "lists") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := getString(m, "gid")
		if id == "" {
			continue
		}
		out = append(out, PlaylistHit{
			ID:      id,
			Name:    getString(m, "specialname"),
			Cover:   getString(m, "img"),
			Creator: getString(m, "nickname"),
			Count:   int(intOf(m, "song_count")),
			Plays:   int(intOf(m, "play_count")),
		})
	}
	return out, nil
}
