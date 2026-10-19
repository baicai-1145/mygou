package kugou

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// VIPClaim is the answer of the daily VIP report: watching the concept
// edition's ad earns a few hours of VIP, several times a day.
type VIPClaim struct {
	AwardHours  int    `json:"awardHours"`
	RemainHours int    `json:"remainHours"`
	Done        int    `json:"done"`
	Total       int    `json:"total"`
	Message     string `json:"message"`
}

// ClaimVIP reports a played ad, which grants VIP hours. It needs login.
func (c *Client) ClaimVIP(ctx context.Context) (VIPClaim, error) {
	if c.Token == "" {
		return VIPClaim{}, errors.New("请先登录")
	}
	now := time.Now().UnixMilli()
	body, err := json.Marshal(map[string]any{
		"ad_id":      12307537187,
		"play_start": now - 30_000,
		"play_end":   now,
	})
	if err != nil {
		return VIPClaim{}, err
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/youth/v1/ad/play_report",
		Data:    string(body),
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		return VIPClaim{}, err
	}
	if !res.OK() {
		return VIPClaim{}, res.Err()
	}
	data := getMap(res.Body, "data")
	claim := VIPClaim{
		AwardHours:  int(intOf(data, "award_vip_hour")),
		RemainHours: int(intOf(data, "remain_vip_hour")),
		Done:        int(intOf(data, "done")),
		Total:       int(intOf(data, "total")),
	}
	if msg, ok := data["error_msg"].(string); ok {
		claim.Message = msg
	}
	return claim, nil
}

// DailyRecommend returns the account's songs of the day.
func (c *Client) DailyRecommend(ctx context.Context) ([]Track, error) {
	if c.Token == "" {
		return nil, errors.New("请先登录")
	}
	res, err := c.do(ctx, Request{
		Method:  "POST",
		Path:    "/everyday_song_recommend",
		Params:  map[string]string{"platform": "ios"},
		Headers: map[string]string{"x-router": "everydayrec.service.kugou.com"},
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	data := getMap(res.Body, "data")
	var out []Track
	for _, raw := range getSlice(data, "song_list") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := Track{
			Hash:         strings.ToLower(getString(m, "hash")),
			Name:         getString(m, "songname", "ori_audio_name"),
			Singer:       getString(m, "author_name"),
			Album:        getString(m, "album_name"),
			AlbumID:      getString(m, "album_id"),
			AlbumAudioID: getString(m, "album_audio_id"),
			MixSongID:    getString(m, "mixsongid"),
			Cover:        getString(m, "sizable_cover"),
		}
		if t.Cover == "" {
			if tp := getMap(m, "trans_param"); tp != nil {
				t.Cover = getString(tp, "union_cover")
			}
		}
		if authors := getSlice(m, "authors"); len(authors) > 0 {
			if a, ok := authors[0].(map[string]any); ok {
				t.SingerID = getString(a, "author_id")
			}
		}
		if ms, ok := toInt(m["timelength_320"]); ok && ms > 0 {
			t.Duration = int(ms / 1000)
		} else if ms, ok := toInt(m["timelength"]); ok {
			t.Duration = int(ms / 1000)
		}
		if t.Hash == "" {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// VipDays is a convenience for logs: how much VIP is left after a claim.
func (c VIPClaim) String() string {
	parts := []string{"+" + strconv.Itoa(c.AwardHours) + "h"}
	if c.RemainHours > 0 {
		parts = append(parts, "remain "+strconv.Itoa(c.RemainHours)+"h")
	}
	if c.Total > 0 {
		parts = append(parts, strconv.Itoa(c.Done)+"/"+strconv.Itoa(c.Total))
	}
	return strings.Join(parts, " ")
}
