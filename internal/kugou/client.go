// Package kugou implements the client side of KuGou's music API, ported
// from MakcRe/KuGouMusicApi (MIT) with the "lite" platform, which is the
// one the KuGou concept edition app uses.
package kugou

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The concept edition ("lite") platform.
const (
	AppID     = 3116
	ClientVer = 11440

	androidSalt = "LnT6xpN3khm36zse0QzvmgTZ3waWdRSA"
	webSalt     = "NVPh5oo715z5DIWAeQlhMDsWXXQV4hwt"
	signKeySalt = "185672dd44712f60bb1736df5a377e82"

	pageID     = 967177915
	ppageID    = "356753938,823673182,967485191"
	songClient = 11430

	userAgent = "Android15-1070-11083-46-0-DiscoveryDRADProtocol-wifi"
)

// Client talks to KuGou. It is safe for concurrent use.
type Client struct {
	HTTP   *http.Client
	Device *Device

	// Token and UserID are the logged in account, if any.
	Token  string
	UserID int64

	// Debug logs requests and responses to stderr.
	Debug bool

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	body    map[string]any
	raw     []byte
	expires time.Time
}

// NewClient returns a client whose device is loaded from dir, or created
// there on first use.
func NewClient(dir string) (*Client, error) {
	d, err := LoadDevice(dir)
	if err != nil {
		return nil, err
	}
	return &Client{
		HTTP:   &http.Client{Timeout: 20 * time.Second},
		Device: d,
		cache:  map[string]cached{},
	}, nil
}

// Request describes one API call. EncryptType is "android" (the default),
// "web" or "register"; the signature covers the final parameters.
type Request struct {
	Method      string
	BaseURL     string
	Path        string
	Params      map[string]string
	Data        string
	EncryptType string

	EncryptKey         bool
	ClearDefaultParams bool
	NotSignature       bool
	Headers            map[string]string

	// CacheTTL keeps a successful GET for that long, as the Node API does
	// for two minutes.
	CacheTTL time.Duration
}

// Response is an API answer. Body is the decoded JSON object; APIs such as
// /song/url also answer with top level fields, not only under data.
type Response struct {
	StatusCode int
	Body       map[string]any
	Raw        []byte
	Cookies    []string
}

// OK reports whether the platform accepted the call, as KuGouMusicApi's
// request wrapper does: status 0 and a non-zero error_code mean failure.
func (r *Response) OK() bool {
	if r == nil {
		return false
	}
	if s, ok := toInt(r.Body["status"]); ok && s == 0 {
		return false
	}
	if ec, ok := toInt(r.Body["error_code"]); ok && ec != 0 {
		return false
	}
	return true
}

// Err returns the platform's error text, for logs and the UI.
func (r *Response) Err() error {
	if r == nil {
		return errors.New("kugou: empty response")
	}
	for _, k := range []string{"error", "msg", "message"} {
		if s, ok := r.Body[k].(string); ok && s != "" {
			return errors.New(s)
		}
	}
	if ec, ok := toInt(r.Body["error_code"]); ok && ec != 0 {
		return fmt.Errorf("kugou: error_code %d", ec)
	}
	return fmt.Errorf("kugou: status %v", r.Body["status"])
}

func (c *Client) do(ctx context.Context, r Request) (*Response, error) {
	if r.Method == "" {
		r.Method = http.MethodGet
	}
	if r.BaseURL == "" {
		r.BaseURL = "https://gateway.kugou.com"
	}
	if r.EncryptType == "" {
		r.EncryptType = "android"
	}

	// Device identifiers, as KuGouMusicApi's createRequest derives them.
	dfid := c.Device.DFID
	if dfid == "" {
		dfid = "-"
	}
	mid := c.Device.MID
	clienttime := strconv.FormatInt(time.Now().Unix(), 10)

	params := map[string]string{}
	if !r.ClearDefaultParams {
		params["dfid"] = dfid
		params["mid"] = mid
		params["uuid"] = "-"
		params["appid"] = strconv.Itoa(AppID)
		params["clientver"] = strconv.Itoa(ClientVer)
		params["clienttime"] = clienttime
		if c.Token != "" {
			params["token"] = c.Token
		}
		if c.UserID != 0 {
			params["userid"] = strconv.FormatInt(c.UserID, 10)
		}
	}
	for k, v := range r.Params {
		params[k] = v
	}
	if _, ok := params["clienttime"]; !ok {
		params["clienttime"] = clienttime
	}

	if r.EncryptKey {
		appid := params["appid"]
		if appid == "" {
			appid = strconv.Itoa(AppID)
		}
		var uid int64
		if s, ok := params["userid"]; ok {
			uid, _ = strconv.ParseInt(s, 10, 64)
		}
		params["key"] = SignKey(params["hash"], params["mid"], uid, appid)
	}

	if !r.NotSignature && params["signature"] == "" {
		switch r.EncryptType {
		case "register":
			params["signature"] = SignatureRegister(params)
		case "web":
			params["signature"] = SignatureWeb(params, r.Data)
		default:
			params["signature"] = SignatureAndroid(params, r.Data)
		}
	}

	u, err := url.Parse(r.BaseURL + r.Path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	var body io.Reader
	if r.Data != "" {
		body = strings.NewReader(r.Data)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("dfid", dfid)
	req.Header.Set("clienttime", params["clienttime"])
	req.Header.Set("mid", mid)
	req.Header.Set("kg-rc", "1")
	req.Header.Set("kg-thash", "5d816a0")
	req.Header.Set("kg-rec", "1")
	req.Header.Set("kg-rf", "B9EDA08A64250DEFFBCADDEE00F8F25F")
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	if r.Data != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "text/plain;charset=utf-8")
	}

	if r.Method == http.MethodGet && r.CacheTTL > 0 {
		key := cacheKey(u.String())
		c.mu.Lock()
		if e, ok := c.cache[key]; ok && time.Now().Before(e.expires) {
			c.mu.Unlock()
			return &Response{StatusCode: 200, Body: e.body, Raw: e.raw}, nil
		}
		c.mu.Unlock()
	}

	if c.Debug {
		fmt.Fprintf(os.Stderr, "kugou: %s %s\n", r.Method, u.Redacted())
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Fprintf(os.Stderr, "kugou: %d %s\n", res.StatusCode, snippet(raw))
	}
	out := &Response{StatusCode: res.StatusCode, Raw: raw}
	for _, ck := range res.Header.Values("Set-Cookie") {
		out.Cookies = append(out.Cookies, cleanCookie(ck))
	}
	var decoded map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err == nil {
		out.Body = decoded
	}
	if res.StatusCode >= 400 {
		return out, fmt.Errorf("kugou: %s %s: HTTP %d: %s", r.Method, r.Path, res.StatusCode, snippet(raw))
	}

	if r.Method == http.MethodGet && r.CacheTTL > 0 && out.OK() {
		key := cacheKey(u.String())
		c.mu.Lock()
		c.cache[key] = cached{body: out.Body, raw: out.Raw, expires: time.Now().Add(r.CacheTTL)}
		c.mu.Unlock()
	}
	return out, nil
}

func cacheKey(u string) string {
	sum := md5.Sum([]byte(u))
	return hex.EncodeToString(sum[:])
}

func cleanCookie(s string) string {
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// sortedParams renders parameters as KuGou signs them: keys sorted, each
// as key=value, joined without separators.
func sortedParams(p map[string]string) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(p[k])
	}
	return b.String()
}

// toInt reads a JSON number or numeric string.
func toInt(v any) (int64, bool) {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		return n, err == nil
	case float64:
		return int64(x), true
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	case int:
		return int64(x), true
	case int64:
		return x, true
	}
	return 0, false
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func getString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := toString(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func getSlice(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}
