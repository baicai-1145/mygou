package kugou

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

// Tag is a playlist category, with its children.
type Tag struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Children []Tag  `json:"children,omitempty"`
}

// PlazaPage is one page of the playlist plaza.
type PlazaPage struct {
	Playlists []PlaylistHit `json:"playlists"`
	HasNext   bool          `json:"hasNext"`
}

// PlaylistTags returns the playlist categories: parents with their children.
func (c *Client) PlaylistTags(ctx context.Context) ([]Tag, error) {
	body, _ := json.Marshal(map[string]any{
		"tag_type": "collection",
		"tag_id":   0,
		"source":   3,
	})
	res, err := c.do(ctx, Request{
		Method:   "POST",
		Path:     "/pubsongs/v1/get_tags_by_type",
		Data:     string(body),
		Headers:  map[string]string{"Content-Type": "application/json"},
		CacheTTL: 30 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	var out []Tag
	for _, raw := range getSlice(res.Body, "data") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := Tag{ID: getString(m, "tag_id"), Name: getString(m, "tag_name")}
		for _, craw := range getSlice(m, "son") {
			cm, ok := craw.(map[string]any)
			if !ok {
				continue
			}
			t.Children = append(t.Children, Tag{
				ID:   getString(cm, "tag_id"),
				Name: getString(cm, "tag_name"),
			})
		}
		out = append(out, t)
	}
	return out, nil
}

// PlaylistPlaza lists public playlists, optionally for one category tag.
func (c *Client) PlaylistPlaza(ctx context.Context, category string, page, size int) (PlazaPage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 24
	}
	cat, _ := strconv.Atoi(category)
	now := time.Now().Unix()
	ct := strconv.FormatInt(now, 10)
	body, err := json.Marshal(map[string]any{
		"appid":               AppID,
		"mid":                 c.Device.MID,
		"clientver":           ClientVer,
		"platform":            "android",
		"clienttime":          ct,
		"userid":              c.UserID,
		"module_id":           1,
		"page":                page,
		"pagesize":            size,
		"key":                 ParamsKey(ct),
		"req_multi":           1,
		"retrun_min":          5,
		"return_special_falg": 1,
		"special_recommend": map[string]any{
			"withtag":       1,
			"withsong":      1,
			"sort":          1,
			"ugc":           1,
			"is_selected":   0,
			"withrecommend": 1,
			"area_code":     1,
			"categoryid":    cat,
		},
	})
	if err != nil {
		return PlazaPage{}, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/v2/special_recommend",
		Data:    string(body),
		Headers: map[string]string{"x-router": "specialrec.service.kugou.com", "Content-Type": "application/json"},
	})
	if err != nil {
		return PlazaPage{}, err
	}
	if !res.OK() {
		return PlazaPage{}, res.Err()
	}
	data := getMap(res.Body, "data")
	page2 := PlazaPage{HasNext: intOf(data, "has_next") != 0}
	for _, raw := range getSlice(data, "special_list") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := getString(m, "global_collection_id")
		if id == "" {
			continue
		}
		cover := getString(m, "imgurl")
		if cover == "" {
			cover = getString(m, "pic")
		}
		count := int(intOf(m, "percount"))
		if count == 0 {
			count = int(intOf(m, "song_count"))
		}
		if count == 0 {
			count = len(getSlice(m, "songs"))
		}
		page2.Playlists = append(page2.Playlists, PlaylistHit{
			ID:      id,
			Name:    getString(m, "specialname"),
			Cover:   cover,
			Creator: getString(m, "nickname"),
			Count:   count,
			Plays:   int(intOf(m, "play_count")),
		})
	}
	return page2, nil
}
