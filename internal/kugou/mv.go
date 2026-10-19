package kugou

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MV is a music video: metadata plus, when resolved, a playable file.
type MV struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Singer   string `json:"singer"`
	Cover    string `json:"cover"`
	Duration int    `json:"duration"`
	URL      string `json:"url,omitempty"`
}

// SearchMVs looks music videos up by keyword.
func (c *Client) SearchMVs(ctx context.Context, keyword string, limit int) ([]MV, error) {
	if limit < 1 {
		limit = 20
	}
	res, err := c.do(ctx, Request{
		BaseURL: "https://complexsearch.kugou.com",
		Path:    "/v1/search/mv",
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
	var out []MV
	for _, raw := range getSlice(getMap(res.Body, "data"), "lists") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := getString(m, "MvID", "MvId")
		if id == "" || id == "0" {
			continue
		}
		cover := getString(m, "Pic")
		if cover != "" && !strings.HasPrefix(cover, "http") {
			// Search returns just the file name; the date is its prefix.
			day := ""
			if len(cover) >= 8 {
				day = cover[:8]
			}
			cover = "https://imge.kugou.com/mvhdpic/{size}/" + day + "/" + cover
		}
		out = append(out, MV{
			ID:       id,
			Name:     getString(m, "MvName"),
			Singer:   getString(m, "SingerName"),
			Cover:    cover,
			Duration: int(intOf(m, "Duration")),
		})
	}
	return out, nil
}

// MVPlay resolves a playable URL for a video id.
func (c *Client) MVPlay(ctx context.Context, id, quality string) (MV, error) {
	encode, err := c.mvSlug(ctx, id)
	if err != nil {
		return MV{}, err
	}
	now := time.Now().UnixMilli()
	params := map[string]string{
		"appid":      "1014",
		"clientver":  "11430",
		"clienttime": strconv.FormatInt(now, 10),
		"mid":        c.Device.MID,
		"uuid":       c.Device.MID,
		"dfid":       c.Device.DFID,
		"id":         encode,
		"srcappid":   "2919",
	}
	params["signature"] = SignatureWeb(params, "")
	res, err := c.do(ctx, Request{
		BaseURL:            "https://wwwapi.kugou.com",
		Path:               "/play/mv",
		Params:             params,
		ClearDefaultParams: true,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0",
			"Referer":    "https://www.kugou.com/",
		},
	})
	if err != nil {
		return MV{}, err
	}
	if !res.OK() {
		return MV{}, res.Err()
	}
	info := getMap(getMap(res.Body, "data"), "info")
	base := getMap(info, "base")
	h264 := getMap(info, "h264")
	play := getMap(getMap(res.Body, "data"), "play")
	priv := getMap(getMap(res.Body, "data"), "privilege")

	mv := MV{
		ID:     getString(base, "video_id"),
		Name:   getString(base, "mv_name"),
		Singer: getString(base, "singer"),
		Cover:  getString(base, "hdpic"),
	}
	if ms := intOf(base, "duration"); ms > 0 {
		mv.Duration = int(ms / 1000)
	}
	order := []string{"fhd", "qhd", "hd", "sd", "ld"}
	if at := indexOf(order, quality); at > 0 {
		order = append(append([]string{}, order[at:]...), order[:at]...)
	}
	for _, q := range order {
		hash := getString(h264, q+"_hash")
		if hash == "" {
			continue
		}
		if p := getMap(priv, hash); p != nil && intOf(p, "status") != 1 {
			continue
		}
		entry := getMap(play, hash)
		if entry == nil {
			continue
		}
		url := getString(entry, "downurl")
		if url == "" {
			if list := getSlice(entry, "backupdownurl"); len(list) > 0 {
				url, _ = list[0].(string)
			}
		}
		if url != "" {
			mv.URL = strings.ReplaceAll(url, "http://", "https://")
			break
		}
	}
	if mv.URL == "" {
		return MV{}, errors.New("kugou: mv: no playable file")
	}
	return mv, nil
}

// mvSlug resolves the short id the play endpoint expects. The web page
// redirects /mv_<id>.html to /mv_<slug>/.
func (c *Client) mvSlug(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.kugou.com/mvweb/html/mv_"+id+".html", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	cl := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("kugou: mv: cannot resolve id")
	}
	loc = strings.TrimRight(loc, "/")
	parts := strings.Split(loc, "/")
	slug := parts[len(parts)-1]
	slug = strings.TrimPrefix(slug, "mv_")
	if slug == "" {
		return "", errors.New("kugou: mv: empty slug")
	}
	return slug, nil
}
