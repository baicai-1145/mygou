// Command smoke exercises the KuGou client against the real service:
// device registration, search, privilege, song URL and lyrics.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/mp3"
	_ "github.com/egoist/mygo/audio/vorbis"

	"mygou/internal/kugou"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dir := os.Getenv("MYGOU_DATA")
	if dir == "" {
		dir = filepath.Join(".tmp", "data")
	}
	c, err := kugou.NewClient(dir)
	if err != nil {
		fatal(err)
	}
	c.Debug = os.Getenv("MYGOU_DEBUG") == "1"
	fmt.Printf("device: guid=%s mid=%s dev=%s dfid=%q\n", c.Device.GUID, c.Device.MID, c.Device.Dev, c.Device.DFID)
	if err := c.EnsureDFID(ctx, dir); err != nil {
		fatal(err)
	}
	if err := c.LoadAccount(dir); err != nil {
		fatal(err)
	}
	fmt.Printf("account: loggedIn=%v userid=%d\n", c.Token != "", c.UserID)
	fmt.Printf("dfid: %s\n", c.Device.DFID)

	keyword := "夜航"
	if len(os.Args) > 1 {
		keyword = os.Args[1]
	}
	tracks, err := c.Search(ctx, keyword, 1, 5)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("search %q: %d tracks\n", keyword, len(tracks))
	for i, t := range tracks {
		fmt.Printf("  %d. %-28s %-16s hash=%s album=%s albumAudio=%s mix=%s dur=%d cover=%s\n",
			i+1, t.Name, t.Singer, t.Hash, t.AlbumID, t.AlbumAudioID, t.MixSongID, t.Duration, t.Cover)
	}
	if len(tracks) == 0 {
		return
	}
	t := tracks[0]
	priv, err := c.Privilege(ctx, []string{t.Hash}, []string{t.AlbumID}, nil)
	if err != nil {
		fatal(err)
	}
	if len(priv) > 0 {
		p := priv[0]
		fmt.Printf("privilege: freeSong=%v freeListen=%v best=%q free=%v vip=%v\n", p.FreeSong, p.FreeListen, p.BestFreeQuality, p.FreeQualities, p.VIPQualities)
		for _, q := range p.Qualities {
			fmt.Printf("  %-12s exists=%-5v level=%d ext=%-5s size=%-8s needVIP=%-5v obtainable=%-5v req=%s\n",
				q.Quality, q.Exists, q.Level, q.ExtName, q.SizeText, q.NeedVIP, q.Obtainable, q.RequireVIPType)
		}
	}

	quality := "128"
	if len(priv) > 0 && priv[0].BestFreeQuality != "" {
		quality = priv[0].BestFreeQuality
	}
	su, err := c.GetSongURL(ctx, t.Hash, t.AlbumID, t.AlbumAudioID, quality, false)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("song url (%s): status=%d ext=%s size=%d bitrate=%d err=%q urls=%d\n", quality, su.Status, su.ExtName, su.Size, su.Bitrate, su.Error, len(su.URLs))
	for _, u := range su.URLs {
		fmt.Printf("  %s\n", u)
	}
	if len(su.URLs) > 0 {
		probe(su.URLs[0])
		full, err := http.Get(su.URLs[0])
		if err == nil {
			data, _ := io.ReadAll(full.Body)
			full.Body.Close()
			fmt.Printf("downloaded %d bytes\n", len(data))
			os.MkdirAll(".tmp", 0o755)
			os.WriteFile(".tmp/probe-audio", data, 0o644)
			decode(data)
		}
	}

	for _, tr := range tracks {
		lyrics, err := c.SearchLyric(ctx, tr.Hash, tr.Name, tr.Duration)
		if err != nil {
			fmt.Println("lyric search:", err)
			continue
		}
		if len(lyrics) == 0 {
			fmt.Printf("no lyric for %s\n", tr.Name)
			continue
		}
		fmt.Printf("lyric candidate for %q: id=%s accesskey=%s song=%s\n", tr.Name, lyrics[0].ID, lyrics[0].AccessKey, lyrics[0].Song)
		text, err := c.GetLyric(ctx, lyrics[0].ID, lyrics[0].AccessKey)
		if err != nil {
			fmt.Println("lyric:", err)
			continue
		}
		if len(text) > 240 {
			text = text[:240]
		}
		fmt.Printf("lyric head:\n%s\n", text)
		break
	}

	q, err := c.QRKey(ctx)
	if err != nil {
		fmt.Println("qr key:", err)
	} else {
		fmt.Printf("qr key ok: %s\n", q.URL)
	}
}

func probe(u string) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		fmt.Println("probe:", err)
		return
	}
	req.Header.Set("Range", "bytes=0-262143")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("probe:", err)
		return
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 262144))
	fmt.Printf("probe: HTTP %d type=%s bytes=%d magic=% x\n", res.StatusCode, res.Header.Get("Content-Type"), len(data), data[:min(8, len(data))])
	if len(data) > 4 {
		switch {
		case string(data[:3]) == "ID3":
			fmt.Println("  looks like MP3 (ID3)")
		case string(data[:4]) == "OggS":
			fmt.Println("  looks like OGG")
		case string(data[:4]) == "fLaC":
			fmt.Println("  looks like FLAC")
		}
	}
}

func decode(data []byte) {
	d, name, err := audio.NewDecoder(bytesReader(data))
	if err != nil {
		fmt.Println("decode:", err)
		return
	}
	fmt.Printf("decoder: %s rate=%d channels=%d frames=%d\n", name, d.SampleRate(), d.Channels(), d.Len())
	buf := make([]float32, 4096)
	var sum float64
	var n int
	for n < d.SampleRate()*d.Channels()*2 { // two seconds
		read, err := d.Read(buf)
		if read > 0 {
			for _, v := range buf[:read] {
				sum += float64(v * v)
			}
			n += read
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("decode read:", err)
			break
		}
	}
	if n > 0 {
		fmt.Printf("decoded %d samples, rms=%.4f\n", n, math.Sqrt(sum/float64(n)))
	}
}

func bytesReader(b []byte) io.ReadSeeker { return bytes.NewReader(b) }

func fatal(err error) {
	b, _ := json.MarshalIndent(err, "", "  ")
	fmt.Fprintln(os.Stderr, string(b))
	os.Exit(1)
}
