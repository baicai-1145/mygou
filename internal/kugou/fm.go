package kugou

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// PersonalFM returns a fresh batch of songs for the private radio. The
// endpoint hands out a handful at a time, so it is called repeatedly until
// enough distinct tracks are collected.
func (c *Client) PersonalFM(ctx context.Context, want int) ([]Track, error) {
	if want < 1 {
		want = 10
	}
	seen := map[string]bool{}
	var out []Track
	for attempt := 0; attempt < 6 && len(out) < want; attempt++ {
		tracks, err := c.fmBatch(ctx)
		if err != nil {
			if len(out) > 0 {
				break
			}
			return nil, err
		}
		for _, t := range tracks {
			if seen[t.Hash] {
				continue
			}
			seen[t.Hash] = true
			out = append(out, t)
		}
	}
	if len(out) > want {
		out = out[:want]
	}
	return out, nil
}

func (c *Client) fmBatch(ctx context.Context) ([]Track, error) {
	now := time.Now().UnixMilli()
	body := map[string]any{
		"appid":                   AppID,
		"clienttime":              now,
		"mid":                     c.Device.MID,
		"action":                  "play",
		"recommend_source_locked": 0,
		"song_pool_id":            0,
		"callerid":                0,
		"m_type":                  1,
		"platform":                "ios",
		"area_code":               1,
		"remain_songcnt":          0,
		"clientver":               ClientVer,
		"is_overplay":             0,
		"mode":                    "normal",
		"fakem":                   "ca981cfc583a4c37f28d2d49000013c16a0a",
		"key":                     ParamsKey(strconv.FormatInt(now, 10)),
	}
	if c.UserID != 0 {
		body["userid"] = c.UserID
		body["kguid"] = c.UserID
	}
	if c.Token != "" {
		body["token"] = c.Token
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/v2/personal_recommend",
		Data:    string(data),
		Headers: map[string]string{"x-router": "persnfm.service.kugou.com", "Content-Type": "application/json"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Track
	for _, raw := range getSlice(getMap(res.Body, "data"), "song_list") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := Track{
			Hash:      strings.ToLower(getString(m, "hash")),
			Name:      getString(m, "songname"),
			Singer:    getString(m, "author_name"),
			Album:     getString(m, "album_name"),
			AlbumID:   getString(m, "album_id"),
			MixSongID: getString(m, "songid"),
		}
		if singers := getSlice(m, "singerinfo"); len(singers) > 0 {
			if s, ok := singers[0].(map[string]any); ok {
				if t.Singer == "" {
					t.Singer = getString(s, "name", "author_name")
				}
				t.SingerID = getString(s, "id", "author_id")
			}
		}
		if tp := getMap(m, "trans_param"); tp != nil {
			t.Cover = getString(tp, "union_cover")
		}
		if v := intOf(m, "time_length"); v > 0 {
			if v > 10000 { // milliseconds on some responses, seconds on others
				v /= 1000
			}
			t.Duration = int(v)
		}
		if t.Hash == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
