package kugou

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Track is a song as search and playlists describe it.
type Track struct {
	Hash         string `json:"hash"`
	Name         string `json:"name"`
	Singer       string `json:"singer"`
	SingerID     string `json:"singerID,omitempty"`
	Album        string `json:"album"`
	AlbumID      string `json:"albumID"`
	AlbumAudioID string `json:"albumAudioID"`
	Cover        string `json:"cover"`
	Duration     int    `json:"duration"`
	MixSongID    string `json:"mixSongID"`
	FileID       string `json:"fileID,omitempty"`
}

// Search searches everything, as the concept edition's search box does.
func (c *Client) Search(ctx context.Context, keyword string, page, size int) ([]Track, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 30
	}
	// /v2/search/song is what the concept edition itself searches with;
	// /v6/search/complex now answers "Parameter Error" to everyone.
	res, err := c.do(ctx, Request{
		BaseURL: "https://complexsearch.kugou.com",
		Path:    "/v2/search/song",
		Headers: map[string]string{"x-router": "complexsearch.kugou.com"},
		Params: map[string]string{
			"platform":        "AndroidFilter",
			"keyword":         keyword,
			"page":            strconv.Itoa(page),
			"pagesize":        strconv.Itoa(size),
			"iscorrection":    "1",
			"privilegefilter": "0",
			"area_code":       "1",
			"dopicfull":       "1",
			"clientver":       "201",
		},
		CacheTTL: time.Minute,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}
	data := getMap(res.Body, "data")
	var tracks []Track
	for _, raw := range getSlice(data, "lists") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t := Track{
			Hash:         strings.ToLower(getString(m, "FileHash", "Hash", "hash")),
			Name:         getString(m, "SongName", "OriSongName", "name"),
			Singer:       getString(m, "SingerName", "singer_name"),
			SingerID:     getString(m, "SingerID", "singerid"),
			Album:        getString(m, "AlbumName", "album_name"),
			AlbumID:      getString(m, "AlbumID", "album_id"),
			AlbumAudioID: getString(m, "AlbumAudioID", "album_audio_id"),
			Cover:        getString(m, "Image", "Cover", "cover"),
			MixSongID:    getString(m, "EMixSongID", "MixSongID", "mixsongid"),
		}
		if d, ok := toInt(m["Duration"]); ok {
			t.Duration = int(d)
		}
		if t.Hash == "" {
			continue
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// Quality is one bitrate of a song and whether the account may play it.
type Quality struct {
	Quality        string `json:"quality"`
	Exists         bool   `json:"exists"`
	Level          int    `json:"level"`
	Bitrate        int    `json:"bitrate"`
	ExtName        string `json:"extName"`
	Hash           string `json:"hash"`
	Size           int64  `json:"size"`
	SizeText       string `json:"sizeText"`
	Privilege      int    `json:"privilege"`
	Status         int    `json:"status"`
	NeedVIP        bool   `json:"needVIP"`
	Obtainable     bool   `json:"obtainable"`
	FreeListen     bool   `json:"freeListen"`
	RequireVIPType string `json:"requireVIPType,omitempty"`
}

// Privilege says which qualities of a song the account may play, and why
// the others are locked.
type Privilege struct {
	Hash            string    `json:"hash"`
	AlbumID         string    `json:"albumID"`
	AlbumAudioID    string    `json:"albumAudioID"`
	Name            string    `json:"name"`
	Singer          string    `json:"singer"`
	Album           string    `json:"album"`
	FreeSong        bool      `json:"freeSong"`
	FreeListen      bool      `json:"freeListen"`
	Qualities       []Quality `json:"qualities"`
	FreeQualities   []string  `json:"freeQualities"`
	VIPQualities    []string  `json:"vipQualities"`
	BestFreeQuality string    `json:"bestFreeQuality"`
}

var qualityOrder = []string{"128", "320", "flac", "high", "super", "viper_clear", "viper_tape", "viper_atmos", "multitrack"}
var upstreamQualities = []string{"128", "320", "flac", "high", "viper_atmos", "viper_tape", "viper_clear", "super", "multitrack"}
var qualityLevel = map[string][]int{"128": {1, 2}, "320": {4}, "flac": {5}, "high": {6}, "super": {7}}
var qualityVIPType = map[string]string{"viper_tape": "suvip", "viper_clear": "suvip", "viper_atmos": "suvip"}

// Privilege asks the media store what the account may play, one entry per
// hash. qualities limits the answer, as the concept edition's quality menu
// does; nil means the usual set.
func (c *Client) Privilege(ctx context.Context, hashes, albumIDs []string, qualities []string) ([]Privilege, error) {
	if len(hashes) == 0 {
		return nil, fmt.Errorf("kugou: privilege: no hash")
	}
	if qualities == nil {
		qualities = []string{"128", "320", "flac", "high", "viper_tape", "viper_clear", "viper_atmos"}
	}
	resource := make([]map[string]any, len(hashes))
	for i, h := range hashes {
		var album any = 0
		if i < len(albumIDs) && albumIDs[i] != "" {
			album = albumIDs[i]
		}
		resource[i] = map[string]any{"type": "audio", "page_id": 0, "hash": strings.ToLower(h), "album_id": album}
	}
	payload := map[string]any{
		"appid":            AppID,
		"area_code":        1,
		"behavior":         "play",
		"clientver":        ClientVer,
		"need_hash_offset": 1,
		"relate":           1,
		"support_verify":   1,
		"resource":         resource,
		"qualities":        upstreamQualities,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res, err := c.do(ctx, Request{
		Method:   "POST",
		Path:     "/v2/get_res_privilege/lite",
		Data:     string(body),
		Headers:  map[string]string{"x-router": "media.store.kugou.com", "Content-Type": "application/json"},
		CacheTTL: 0,
	})
	if err != nil {
		return nil, err
	}
	if !res.OK() {
		return nil, res.Err()
	}

	data := getSlice(res.Body, "data")
	out := make([]Privilege, 0, len(hashes))
	for i, h := range hashes {
		h = strings.ToLower(h)
		item := map[string]any{}
		for _, raw := range data {
			m, ok := raw.(map[string]any)
			if ok && strings.EqualFold(getString(m, "hash"), h) {
				item = m
				break
			}
		}
		if len(item) == 0 && i < len(data) {
			if m, ok := data[i].(map[string]any); ok {
				item = m
			}
		}
		out = append(out, buildPrivilege(h, item, qualities))
	}
	return out, nil
}

func buildPrivilege(hash string, item map[string]any, qualities []string) Privilege {
	goods := getSlice(item, "relate_goods")
	trans := getMap(item, "trans_param")
	p := Privilege{
		Hash:         hash,
		AlbumID:      getString(item, "album_id"),
		AlbumAudioID: getString(item, "album_audio_id"),
		Name:         getString(item, "name"),
		Singer:       getString(item, "singername"),
		Album:        getString(item, "albumname"),
	}
	free := true
	best := ""
	bestLevel := -1
	for _, q := range qualities {
		rel := pickRelate(goods, q)
		exists := rel != nil
		qy := Quality{Quality: q, Exists: exists}
		if exists {
			level, _ := toInt(rel["level"])
			qy.Level = int(level)
			priv, _ := toInt(rel["privilege"])
			qy.Privilege = int(priv)
			status, _ := toInt(rel["status"])
			qy.Status = int(status)
			qy.NeedVIP = priv != 0
			if info := getMap(rel, "info"); info != nil {
				bitrate, _ := toInt(info["bitrate"])
				qy.Bitrate = int(bitrate)
				qy.ExtName = getString(info, "extname")
				size, _ := toInt(info["filesize"])
				qy.Size = size
			}
			qy.Hash = getString(rel, "hash")
			// The concept edition plays the ogg copies of 128 and 320.
			if trans != nil {
				var oh string
				var os int64
				if q == "128" {
					oh, os = getString(trans, "ogg_128_hash"), intOf(trans, "ogg_128_filesize")
				} else if q == "320" {
					oh, os = getString(trans, "ogg_320_hash"), intOf(trans, "ogg_320_filesize")
				}
				if oh != "" {
					qy.Hash, qy.Size, qy.ExtName = oh, os, "ogg"
				}
			}
			qy.FreeListen = q == "128" && qy.NeedVIP
			qy.Obtainable = qy.FreeListen || (!qy.NeedVIP && qy.Status != 0)
			if qy.NeedVIP && !qy.Obtainable {
				qy.RequireVIPType = qualityVIPType[q]
				if qy.RequireVIPType == "" {
					qy.RequireVIPType = "concept"
				}
			}
			if !qy.NeedVIP {
				free = false
			}
			if qy.Obtainable && (qy.Level > bestLevel || qy.Level == bestLevel && indexOf(qualityOrder, q) > indexOf(qualityOrder, best)) {
				best, bestLevel = q, qy.Level
			}
		}
		qy.SizeText = formatSize(qy.Size)
		if qy.Obtainable {
			p.FreeQualities = append(p.FreeQualities, q)
		}
		if exists && qy.NeedVIP && !qy.Obtainable {
			p.VIPQualities = append(p.VIPQualities, q)
		}
		if qy.FreeListen {
			p.FreeListen = true
		}
		p.Qualities = append(p.Qualities, qy)
	}
	p.FreeSong = len(p.Qualities) > 0 && free
	p.BestFreeQuality = best
	return p
}

func pickRelate(goods []any, quality string) map[string]any {
	levels := qualityLevel[quality]
	for _, level := range levels {
		for _, raw := range goods {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			lv, _ := toInt(m["level"])
			if int(lv) == level && getString(m, "quality") == quality {
				return m
			}
		}
	}
	for _, raw := range goods {
		m, ok := raw.(map[string]any)
		if ok && getString(m, "quality") == quality {
			return m
		}
	}
	return nil
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func formatSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1fG", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n > 0:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	}
	return ""
}

func intOf(m map[string]any, key string) int64 {
	n, _ := toInt(m[key])
	return n
}

// SongURL is a playable file: the CDN URLs, their format, and why there is
// none when there is not.
type SongURL struct {
	Status  int      `json:"status"`
	URLs    []string `json:"urls"`
	ExtName string   `json:"extName"`
	Size    int64    `json:"size"`
	Bitrate int      `json:"bitrate"`
	Error   string   `json:"error,omitempty"`
}

// GetSongURL resolves a song to a playable URL, at quality ("128", "320",
// "flac", …). freePart asks for the trial excerpt when the account cannot
// play the whole song.
func (c *Client) GetSongURL(ctx context.Context, hash, albumID, albumAudioID, quality string, freePart bool) (SongURL, error) {
	if quality == "" {
		quality = "128"
	}
	dfid := c.Device.DFID
	if dfid == "" {
		dfid = randomString(24)
	}
	free := "0"
	if freePart {
		free = "1"
	}
	params := map[string]string{
		"album_id":       orZero(albumID),
		"area_code":      "1",
		"hash":           strings.ToLower(hash),
		"ssa_flag":       "is_fromtrack",
		"version":        "11430",
		"page_id":        strconv.Itoa(pageID),
		"quality":        quality,
		"album_audio_id": orZero(albumAudioID),
		"behavior":       "play",
		"pid":            "411",
		"cmd":            "26",
		"pidversion":     "3001",
		"IsFreePart":     free,
		"ppage_id":       ppageID,
		"cdnBackup":      "1",
		"module":         "",
		"clientver":      strconv.Itoa(songClient),
	}
	// The module overrides the request's cookie with a fresh dfid.
	saved := c.Device.DFID
	c.Device.DFID = dfid
	res, err := c.do(ctx, Request{
		Path:       "/v5/url",
		Params:     params,
		EncryptKey: true,
		Headers:    map[string]string{"x-router": "trackercdn.kugou.com"},
	})
	c.Device.DFID = saved
	if err != nil {
		return SongURL{}, err
	}

	body := res.Body
	if data := getMap(body, "data"); data != nil {
		body = data
	}
	out := SongURL{
		Status:  int(intOf(body, "status")),
		ExtName: getString(body, "extName", "extname"),
		Size:    intOf(body, "fileSize") + intOf(body, "file_size"),
		Bitrate: int(intOf(body, "bitrate")),
		Error:   getString(body, "error", "msg"),
	}
	out.Size = intOf(body, "fileSize")
	if out.Size == 0 {
		out.Size = intOf(body, "file_size")
	}
	for _, raw := range getSlice(body, "url") {
		if s := toString(raw); s != "" {
			out.URLs = append(out.URLs, s)
		}
	}
	if len(out.URLs) == 0 {
		for _, raw := range getSlice(body, "backupUrl") {
			if s := toString(raw); s != "" {
				out.URLs = append(out.URLs, s)
			}
		}
	}
	if out.Error == "" && (out.Status != 1 || len(out.URLs) == 0) {
		out.Error = "no url"
	}
	return out, nil
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// LyricCandidate is one lyric the lyrics service offers for a song.
// Profile is the signed in account's public face.
type Profile struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// Profile reads the account's nickname and avatar.
func (c *Client) Profile(ctx context.Context) (Profile, error) {
	if c.Token == "" {
		return Profile{}, errors.New("请先登录")
	}
	now := time.Now().Unix()
	plain, err := json.Marshal(struct {
		Token      string `json:"token"`
		ClientTime int64  `json:"clienttime"`
	}{c.Token, now})
	if err != nil {
		return Profile{}, err
	}
	enc, err := rsaRawHex(plain)
	if err != nil {
		return Profile{}, err
	}
	body, err := json.Marshal(map[string]any{
		"visit_time": now,
		"usertype":   1,
		"p":          strings.ToUpper(enc),
		"userid":     c.UserID,
	})
	if err != nil {
		return Profile{}, err
	}
	res, err := c.do(ctx, Request{
		Method:   "POST",
		Path:     "/v3/get_my_info",
		Params:   map[string]string{"plat": "1"},
		Data:     string(body),
		Headers:  map[string]string{"x-router": "usercenter.kugou.com", "Content-Type": "application/json"},
		CacheTTL: time.Minute,
	})
	if err != nil {
		return Profile{}, err
	}
	if !res.OK() {
		return Profile{}, res.Err()
	}
	data := getMap(res.Body, "data")
	return Profile{
		Name:   getString(data, "nickname", "k_nickname"),
		Avatar: getString(data, "pic", "user_pic", "k_pic"),
	}, nil
}

type LyricCandidate struct {
	ID        string `json:"id"`
	AccessKey string `json:"accessKey"`
	Singer    string `json:"singer"`
	Song      string `json:"song"`
	Duration  int    `json:"duration"`
}

// SearchLyric finds the lyrics of a song by its hash.
func (c *Client) SearchLyric(ctx context.Context, hash, keyword string, duration int) ([]LyricCandidate, error) {
	// The PC endpoints are the ones the service still answers; the app's
	// /v1/search replies 20010 with an empty body.
	res, err := c.do(ctx, Request{
		BaseURL:            "https://lyrics.kugou.com",
		Path:               "/search",
		ClearDefaultParams: true,
		NotSignature:       true,
		Params: map[string]string{
			"ver":      "1",
			"man":      "no",
			"client":   "pc",
			"keyword":  keyword,
			"duration": strconv.Itoa(duration),
			"hash":     strings.ToLower(hash),
		},
		CacheTTL: 10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	var out []LyricCandidate
	for _, raw := range getSlice(res.Body, "candidates") {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, LyricCandidate{
			ID:        getString(m, "id"),
			AccessKey: getString(m, "accesskey"),
			Singer:    getString(m, "singer"),
			Song:      getString(m, "song"),
			Duration:  int(intOf(m, "duration") / 1000),
		})
	}
	return out, nil
}

// GetLyric returns the decoded KRC text of a lyric candidate.
func (c *Client) GetLyric(ctx context.Context, id, accessKey string) (string, error) {
	res, err := c.do(ctx, Request{
		BaseURL: "https://lyrics.kugou.com",
		Path:    "/download",
		Params: map[string]string{
			"ver":       "1",
			"client":    "pc",
			"id":        id,
			"accesskey": accessKey,
			"fmt":       "krc",
			"charset":   "utf8",
		},
	})
	if err != nil {
		return "", err
	}
	content := getString(res.Body, "content")
	if content == "" {
		return "", fmt.Errorf("kugou: lyric: %s", snippet(res.Raw))
	}
	raw, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", err
	}
	ct, _ := toInt(res.Body["contenttype"])
	if ct == 0 {
		return DecodeKRC(raw)
	}
	return string(raw), nil
}

// QRLogin is a login by QR code: the key, the URL to draw, and the token
// once the phone confirmed it.
type QRLogin struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Token  string `json:"token"`
	UserID int64  `json:"userID"`
	// Status: 0 expired, 1 waiting for a scan, 2 scanned and waiting for a
	// confirmation, 4 logged in.
	Status int `json:"status"`
}

// QRKey asks for a login QR code.
func (c *Client) QRKey(ctx context.Context) (QRLogin, error) {
	res, err := c.do(ctx, Request{
		BaseURL:     "https://login-user.kugou.com",
		Path:        "/v2/qrcode",
		EncryptType: "web",
		Params: map[string]string{
			"appid":      "1001",
			"type":       "1",
			"plat":       "4",
			"qrcode_txt": "https://h5.kugou.com/apps/loginQRCode/html/index.html?appid=3116&",
			"srcappid":   "2919",
		},
	})
	if err != nil {
		return QRLogin{}, err
	}
	data := getMap(res.Body, "data")
	q := QRLogin{Key: getString(data, "qrcode", "key")}
	if q.Key == "" {
		return q, fmt.Errorf("kugou: qr key: %s", snippet(res.Raw))
	}
	q.URL = "https://h5.kugou.com/apps/loginQRCode/html/index.html?qrcode=" + q.Key
	return q, nil
}

// QRCheck reports the state of a login QR code, and logs the client in
// when the phone confirmed it.
func (c *Client) QRCheck(ctx context.Context, key string) (QRLogin, error) {
	res, err := c.do(ctx, Request{
		BaseURL:     "https://login-user.kugou.com",
		Path:        "/v2/get_userinfo_qrcode",
		EncryptType: "web",
		Params: map[string]string{
			"plat":     "4",
			"appid":    "3116",
			"srcappid": "2919",
			"qrcode":   key,
			"dev":      c.Device.Dev,
		},
	})
	if err != nil {
		return QRLogin{}, err
	}
	data := getMap(res.Body, "data")
	q := QRLogin{Key: key, Status: int(intOf(data, "status"))}
	if q.Status == 4 {
		q.Token = getString(data, "token")
		q.UserID = intOf(data, "userid")
		if q.Token != "" {
			c.Token = q.Token
			c.UserID = q.UserID
		}
	}
	return q, nil
}

// QRKeyApp asks for a login QR code with the Android app credentials
// (appid 1005 / clientver 20489). Tokens from the lite QR login are refused
// by the cloud list service, so the app flavour is tried separately.
func (c *Client) QRKeyApp(ctx context.Context) (QRLogin, error) {
	res, err := c.do(ctx, Request{
		BaseURL:     "https://login-user.kugou.com",
		Path:        "/v2/qrcode",
		EncryptType: "web",
		Params: map[string]string{
			"appid":      "1005",
			"type":       "1",
			"plat":       "1",
			"qrcode_txt": "https://h5.kugou.com/apps/loginQRCode/html/index.html?appid=1005&",
			"srcappid":   "2919",
			"clientver":  "20489",
		},
	})
	if err != nil {
		return QRLogin{}, err
	}
	data := getMap(res.Body, "data")
	q := QRLogin{Key: getString(data, "qrcode", "key")}
	if q.Key == "" {
		return q, fmt.Errorf("kugou: qr key: %s", snippet(res.Raw))
	}
	q.URL = "https://h5.kugou.com/apps/loginQRCode/html/index.html?qrcode=" + q.Key
	return q, nil
}

// QRCheckApp checks an app QR code and logs in on success.
func (c *Client) QRCheckApp(ctx context.Context, key string) (QRLogin, error) {
	res, err := c.do(ctx, Request{
		BaseURL:     "https://login-user.kugou.com",
		Path:        "/v2/get_userinfo_qrcode",
		EncryptType: "web",
		Params: map[string]string{
			"plat":      "1",
			"appid":     "1005",
			"srcappid":  "2919",
			"clientver": "20489",
			"qrcode":    key,
			"dev":       c.Device.Dev,
		},
	})
	if err != nil {
		return QRLogin{}, err
	}
	data := getMap(res.Body, "data")
	q := QRLogin{Key: key, Status: int(intOf(data, "status"))}
	if q.Status == 4 {
		q.Token = getString(data, "token")
		q.UserID = intOf(data, "userid")
		if q.Token != "" {
			c.Token = q.Token
			c.UserID = q.UserID
		}
	}
	return q, nil
}

var _ = http.MethodPost
