package player

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"mygou/internal/kugou"
)

// httpStream turns a CDN URL into an io.ReadSeeker: every Seek drops the
// current response and the next Read asks for the new offset with a Range
// header, so the decoder can start before the file has downloaded.
type httpStream struct {
	url  string
	pos  int64
	size int64
	body io.ReadCloser
}

func newHTTPStream(ctx context.Context, rawURL string) (*httpStream, error) {
	h := &httpStream{url: rawURL}
	if err := h.open(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *httpStream) open(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", h.pos))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return fmt.Errorf("stream: HTTP %d", resp.StatusCode)
	}
	h.body = resp.Body
	if h.size == 0 {
		if resp.StatusCode == http.StatusPartialContent {
			fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %*d-%*d/%d", &h.size)
		} else {
			h.size = resp.ContentLength
		}
	}
	return nil
}

func (h *httpStream) Read(p []byte) (int, error) {
	if h.body == nil {
		if err := h.open(context.Background()); err != nil {
			return 0, err
		}
	}
	n, err := h.body.Read(p)
	h.pos += int64(n)
	return n, err
}

func (h *httpStream) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		h.pos = off
	case io.SeekCurrent:
		h.pos += off
	case io.SeekEnd:
		if h.size == 0 {
			return h.pos, fmt.Errorf("stream: size unknown")
		}
		h.pos = h.size + off
	}
	if h.body != nil {
		h.body.Close()
		h.body = nil
	}
	return h.pos, nil
}

func (h *httpStream) Close() error {
	if h.body != nil {
		err := h.body.Close()
		h.body = nil
		return err
	}
	return nil
}

// resolveTrack mirrors fetch's quality negotiation but stops before the
// download, so playback can start while the file streams.
func (e *Engine) resolveTrack(ctx context.Context, t Track) (kugou.SongURL, string, string, error) {
	e.mu.Lock()
	want := e.quality
	e.mu.Unlock()

	qhash := map[string]string{}
	if priv, err := e.client.Privilege(ctx, []string{t.Hash}, []string{t.AlbumID}, nil); err == nil && len(priv) > 0 {
		for _, q := range priv[0].Qualities {
			if q.Hash != "" {
				qhash[q.Quality] = strings.ToLower(q.Hash)
			}
		}
	}
	hashFor := func(q string) string {
		if h := qhash[q]; h != "" {
			return h
		}
		return t.Hash
	}

	order := []string{want}
	switch want {
	case "high":
		order = append(order, "flac")
		fallthrough
	case "flac":
		order = append(order, "320")
	}
	order = append(order, "128")

	fetchURL := func(h, q string, trial bool) (kugou.SongURL, error) {
		r, err := e.client.GetSongURL(ctx, h, t.AlbumID, t.AlbumAudioID, q, trial)
		if (err != nil || len(r.URLs) == 0) && t.AlbumAudioID != "" && t.AlbumAudioID != "0" {
			if r2, err2 := e.client.GetSongURL(ctx, h, t.AlbumID, "0", q, trial); err2 == nil && len(r2.URLs) > 0 {
				return r2, nil
			}
		}
		if (err != nil || len(r.URLs) == 0) && t.AlbumID != "" && t.AlbumID != "0" {
			if r3, err3 := e.client.GetSongURL(ctx, h, "0", "0", q, trial); err3 == nil && len(r3.URLs) > 0 {
				return r3, nil
			}
		}
		return r, err
	}

	var su kugou.SongURL
	quality := ""
	trial := false
	for _, q := range order {
		r, err := fetchURL(hashFor(q), q, false)
		if err != nil || len(r.URLs) == 0 {
			continue
		}
		su, quality = r, q
		break
	}
	if quality == "" {
		for _, q := range order {
			r, err := fetchURL(hashFor(q), q, true)
			if err != nil || len(r.URLs) == 0 {
				continue
			}
			su, quality, trial = r, q, true
			break
		}
	}
	if quality == "" {
		return su, "", "", fmt.Errorf("no playable url")
	}
	ext := su.ExtName
	if ext == "" {
		ext = "mp3"
	}
	name := hashFor(quality) + "-" + quality + "." + ext
	if trial {
		name = hashFor(quality) + "-" + quality + "-trial." + ext
	}
	return su, quality, filepath.Join(e.dir, name), nil
}
