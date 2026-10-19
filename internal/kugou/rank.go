package kugou

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// Rank is one of the store's charts.
type Rank struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cover     string `json:"cover"`
	Frequency string `json:"frequency"`
	Intro     string `json:"intro"`
}

// Ranks lists the charts the store publishes.
func (c *Client) Ranks(ctx context.Context) ([]Rank, error) {
	res, err := c.do(ctx, Request{
		Method:   "GET",
		Path:     "/ocean/v6/rank/list",
		Params:   map[string]string{"plat": "2", "withsong": "0", "parentid": "0"},
		CacheTTL: 10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Rank
	for _, raw := range getSlice(getMap(res.Body, "data"), "info") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := getString(m, "rankid")
		if id == "" {
			continue
		}
		out = append(out, Rank{
			ID:        id,
			Name:      getString(m, "rankname"),
			Cover:     getString(m, "imgurl"),
			Frequency: getString(m, "update_frequency"),
			Intro:     getString(m, "intro"),
		})
	}
	return out, nil
}

// RankSongs returns one page of a chart.
func (c *Client) RankSongs(ctx context.Context, id string, page, size int) ([]Track, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 100
	}
	body, err := json.Marshal(map[string]any{
		"show_portrait_mv":        1,
		"show_type_total":         1,
		"filter_original_remarks": 1,
		"area_code":               1,
		"pagesize":                size,
		"rank_cid":                0,
		"type":                    1,
		"page":                    page,
		"rank_id":                 id,
	})
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/openapi/kmr/v2/rank/audio",
		Data:    string(body),
		Headers: map[string]string{"kg-tid": "369", "Content-Type": "application/json"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Track
	for _, raw := range getSlice(getMap(res.Body, "data"), "songlist") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		info := getMap(m, "audio_info")
		album := getMap(m, "album_info")
		trans := getMap(m, "trans_param")
		t := Track{
			Hash:         strings.ToLower(getString(info, "hash_128")),
			Name:         getString(m, "songname"),
			Singer:       getString(m, "author_name"),
			Album:        getString(album, "album_name"),
			AlbumID:      getString(m, "album_id"),
			AlbumAudioID: getString(m, "album_audio_id"),
			MixSongID:    getString(m, "audio_id"),
		}
		if authors := getSlice(m, "authors"); len(authors) > 0 {
			if a, ok := authors[0].(map[string]any); ok {
				t.SingerID = getString(a, "author_id")
			}
		}
		if trans != nil {
			t.Cover = getString(trans, "union_cover")
		}
		if ms := intOf(info, "duration_128"); ms > 0 {
			t.Duration = int(ms / 1000)
		}
		if t.Hash == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
