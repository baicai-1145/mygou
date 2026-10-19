package kugou

import (
	"context"
	"strconv"
	"time"
)

// Comment is one song comment.
type Comment struct {
	ID       string `json:"id"`
	User     string `json:"user"`
	Avatar   string `json:"avatar"`
	Content  string `json:"content"`
	Time     string `json:"time"`
	Location string `json:"location"`
	Likes    int    `json:"likes"`
	Replies  int    `json:"replies"`
}

// Comments returns a page of comments for a song, by its mix song id.
func (c *Client) Comments(ctx context.Context, mixID string, page, size int) ([]Comment, int, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 30
	}
	res, err := c.do(ctx, Request{
		Method: "POST",
		Path:   "/mcomment/v1/cmtlist",
		Params: map[string]string{
			"ver":               "6",
			"mixsongid":         mixID,
			"need_show_image":   "1",
			"p":                 strconv.Itoa(page),
			"pagesize":          strconv.Itoa(size),
			"show_classify":     "1",
			"show_hotword_list": "1",
			"extdata":           "0",
			"code":              "fc4be23b4e972707f36b8a828a93ba8a",
		},
		CacheTTL: time.Minute,
	})
	if err != nil {
		return nil, 0, err
	}
	if !res.OK() {
		return nil, 0, res.Err()
	}
	total := int(intOf(res.Body, "combine_count"))
	if total == 0 {
		total = int(intOf(res.Body, "count"))
	}
	var out []Comment
	for _, raw := range getSlice(res.Body, "list") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cm := Comment{
			ID:      getString(m, "id"),
			User:    getString(m, "user_name"),
			Avatar:  getString(m, "user_pic"),
			Content: getString(m, "content"),
			Time:    getString(m, "addtime"),
			Likes:   int(intOf(getMap(m, "like"), "count")),
			Replies: int(intOf(m, "reply_num")),
		}
		if loc := getString(m, "location"); loc != "" {
			cm.Location = loc
		}
		if cm.Content == "" {
			continue
		}
		out = append(out, cm)
	}
	return out, total, nil
}
