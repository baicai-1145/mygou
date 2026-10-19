package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mygou/internal/player"
	"mygou/internal/service"
)

type lyricLine struct {
	t    float64
	text string
}

var (
	reKRC = regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	reLRC = regexp.MustCompile(`^\[(\d+):(\d+)(?:[.:](\d+))?\](.*)$`)
	reTag = regexp.MustCompile(`<[^>]*>`)
)

func parseLyrics(raw string) []lyricLine {
	var lines []lyricLine
	for _, l := range strings.Split(raw, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if m := reKRC.FindStringSubmatch(l); len(m) == 4 {
			ms, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			txt := strings.TrimSpace(reTag.ReplaceAllString(m[3], ""))
			if txt != "" {
				lines = append(lines, lyricLine{t: ms / 1000.0, text: txt})
			}
			continue
		}
		if m := reLRC.FindStringSubmatch(l); len(m) == 5 {
			min, _ := strconv.ParseFloat(m[1], 64)
			sec, _ := strconv.ParseFloat(m[2], 64)
			msStr := m[3]
			var frac float64
			if msStr != "" {
				val, _ := strconv.ParseFloat(msStr, 64)
				if len(msStr) == 3 {
					frac = val / 1000.0
				} else {
					frac = val / 100.0
				}
			}
			txt := strings.TrimSpace(m[4])
			if txt != "" {
				lines = append(lines, lyricLine{t: min*60 + sec + frac, text: txt})
			}
			continue
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].t < lines[j].t
	})
	return lines
}

func loadStatusBarLyric(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "lyric_statusbar"))
	if err != nil {
		return true // 默认开启
	}
	return strings.TrimSpace(string(b)) != "0"
}

func saveStatusBarLyric(dir string, on bool) {
	val := "1"
	if !on {
		val = "0"
	}
	_ = os.WriteFile(filepath.Join(dir, "lyric_statusbar"), []byte(val), 0o644)
}

// shim is the menu bar presence: a tray icon whose click opens the mini
// player, a panel of native UI that stays cheap while the main window is
// closed.
type shim struct {
	svc     *service.App
	dataDir string

	mu           sync.Mutex
	state        player.State
	cover        *ui.Bitmap
	coverFor     string
	tray         *mygo.Tray
	panel        *mygo.Window
	lyricOn      bool
	lyricHash    string
	lyrics       []lyricLine
	currentTitle string
}

func newShim(svc *service.App, dataDir string) *shim {
	return &shim{
		svc:     svc,
		dataDir: dataDir,
		lyricOn: loadStatusBarLyric(dataDir),
	}
}

func (s *shim) toggleLyric(on bool) {
	s.mu.Lock()
	s.lyricOn = on
	if !on {
		s.currentTitle = ""
		mygo.RunOnMain(func() {
			if s.tray != nil {
				s.tray.SetTitle("")
			}
		})
	}
	s.mu.Unlock()
	saveStatusBarLyric(s.dataDir, on)
}

func (s *shim) onTrayClick() {
	if mainWin == nil || !mainWin.IsVisible() {
		showMain()
		return
	}
	s.togglePanel()
}

func (s *shim) ready() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "mygou",
		Icon:           trayIcon(),
		IconIsTemplate: runtime.GOOS == "darwin",
		ToolTip:        "mygou",
		Menu:           s.menu(),
	})
	if err != nil {
		log.Printf("tray: %v", err)
		return
	}
	s.tray = tray
	if runtime.GOOS == "linux" {
		tray.SetMenu(s.menu())
	} else {
		tray.OnClick(s.onTrayClick)
		tray.OnRightClick(func() {
			if s.panel != nil {
				s.panel.Close()
			}
			tray.SetMenu(s.menu())
			defer tray.SetMenu(nil)
			tray.PopUpMenu()
		})
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if s.panel != nil {
			s.panel.Close()
		}
		tray.Destroy()
	})
	go s.loop()
}

func (s *shim) menu() *mygo.Menu {
	s.mu.Lock()
	lyricOn := s.lyricOn
	s.mu.Unlock()

	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开 mygou", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { showMain() }},
		mygo.Separator(),
		{
			Label:   "状态栏歌词",
			Type:    mygo.MenuItemCheckbox,
			Checked: lyricOn,
			Click: func(item *mygo.MenuItem, _ *mygo.Window) {
				s.toggleLyric(item.Checked)
			},
		},
		mygo.Separator(),
		{Label: "播放 / 暂停", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { s.svc.Toggle() }},
		{Label: "上一首", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { s.svc.Prev() }},
		{Label: "下一首", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { s.svc.Next() }},
		mygo.Separator(),
		{Label: "退出 mygou", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { mygo.App.Quit() }},
	})
}

func (s *shim) loadLyric(hash, name string, dur int) {
	ly, err := s.svc.Lyric(hash, name, dur)
	if err != nil || ly.Text == "" {
		s.mu.Lock()
		if s.lyricHash == hash {
			s.lyrics = nil
		}
		s.mu.Unlock()
		return
	}
	parsed := parseLyrics(ly.Text)
	s.mu.Lock()
	if s.lyricHash == hash {
		s.lyrics = parsed
	}
	s.mu.Unlock()
}

// loop keeps the panel, menu, and status bar lyrics in step with the player.
func (s *shim) loop() {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	tickCount := 0
	for range t.C {
		tickCount++
		st := s.svc.State()

		s.mu.Lock()
		s.state = st
		if st.Track.Hash != "" && st.Track.Hash != s.coverFor {
			s.coverFor = st.Track.Hash
			go s.refreshCover(st.Track.Hash, st.Track.Cover)
		}
		if st.Track.Hash != "" && st.Track.Hash != s.lyricHash {
			s.lyricHash = st.Track.Hash
			s.lyrics = nil
			go s.loadLyric(st.Track.Hash, st.Track.Name, int(st.Duration))
		}

		var targetTitle string
		if s.lyricOn && st.Playing {
			pos := st.Position
			var currentLine string
			for _, l := range s.lyrics {
				if l.t <= pos+0.25 {
					currentLine = l.text
				} else {
					break
				}
			}
			if currentLine != "" {
				targetTitle = currentLine
			} else if st.Track.Name != "" {
				targetTitle = st.Track.Name + " - " + st.Track.Singer
			}
		}

		if targetTitle != "" {
			runes := []rune(targetTitle)
			if len(runes) > 28 {
				targetTitle = string(runes[:26]) + "…"
			}
			targetTitle = "  " + targetTitle
		}

		if targetTitle != s.currentTitle {
			s.currentTitle = targetTitle
			title := targetTitle
			mygo.RunOnMain(func() {
				if s.tray != nil {
					s.tray.SetTitle(title)
				}
			})
		}
		s.mu.Unlock()

		if tickCount%4 == 0 {
			mygo.RunOnMain(func() {
				mygo.SetNowPlaying(mygo.NowPlayingInfo{
					Title:    st.Track.Name,
					Artist:   st.Track.Singer,
					Album:    st.Track.Album,
					Duration: st.Duration,
					Elapsed:  st.Position,
					Playing:  st.Playing,
				})
			})
			if s.panel != nil {
				s.panel.Update(func() {})
			}
		}
	}
}

// refreshCover fetches the cover of a new track, then redraws the panel.
func (s *shim) refreshCover(hash, url string) {
	bmp := s.loadCover(url)
	s.mu.Lock()
	if s.coverFor == hash {
		s.cover = bmp
	}
	panel := s.panel
	s.mu.Unlock()
	if panel != nil {
		panel.Update(func() {})
	}
}

// loadCover fetches the cover of the current track at 200 pixels, once.
func (s *shim) loadCover(url string) *ui.Bitmap {
	if url == "" {
		return nil
	}
	url = strings.ReplaceAll(url, "{size}", "200")
	url = strings.Replace(url, "http://", "https://", 1)
	res, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		return nil
	}
	return bmp
}

func (s *shim) togglePanel() {
	if s.panel != nil {
		s.panel.Close()
		return
	}
	s.panel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      s.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     300,
		Height:    210,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(s.view),
	})
	s.panel.OnClosed(func() { s.panel = nil })
}

func (s *shim) view(c *ui.Context) {
	t := c.Theme()
	box := ui.Column(c).Fill()
	if !c.Vibrancy() {
		box.Background(t.Surface).Radius(12)
	}
	box.Padding(18).Gap(14).Children(func() {
		s.mu.Lock()
		st := s.state
		cover := s.cover
		s.mu.Unlock()

		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			if cover != nil {
				ui.Image(c, cover).Size(56, 56).Fit(ui.Cover).Radius(8)
			} else {
				ui.Row(c).Size(56, 56).Radius(8).Background(t.Surface)
			}
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				name := "未在播放"
				singer := "搜索一首歌开始"
				if st.Track.Hash != "" {
					name = st.Track.Name
					singer = st.Track.Singer
				}
				ui.Text(c, name).FontSize(13).Bold().SingleLine()
				ui.Text(c, singer).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
		})

		// Progress: two boxes, a track and its filled part.
		pos, dur := st.Position, st.Duration
		ratio := float32(0)
		if dur > 0 {
			ratio = float32(pos / dur)
			if ratio > 1 {
				ratio = 1
			}
		}
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).Height(4).Radius(2).Background(t.Surface).Children(func() {
				if ratio > 0 {
					ui.Row(c).Height(4).Radius(2).Background(t.Accent).Width(float32(264) * ratio)
				}
			})
			ui.Row(c).Children(func() {
				ui.Spacer(c)
				ui.Text(c, fmt.Sprintf("%s / %s", clock(pos), clock(dur))).FontSize(10).TextColor(t.TextMuted)
			})
		})

		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c.Key("prev"), "⏮").Clicked() {
				s.svc.Prev()
			}
			label := "▶"
			if st.Playing {
				label = "⏸"
			}
			if ui.PrimaryButton(c.Key("toggle"), label).Clicked() {
				s.svc.Toggle()
			}
			if ui.Button(c.Key("next"), "⏭").Clicked() {
				s.svc.Next()
			}
			ui.Spacer(c)
			if ui.Button(c.Key("open"), "打开主窗口").Clicked() {
				showMain()
				if s.panel != nil {
					s.panel.Close()
				}
			}
		})
	})
}

func clock(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func trayIcon() []byte {
	const size = 18
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := color.RGBA{0, 0, 0, 255}

	for y := 2; y < 14; y++ {
		img.Set(3, y, c)
		img.Set(4, y, c)
		img.Set(13, y, c)
		img.Set(14, y, c)
	}
	for x := 3; x <= 14; x++ {
		for y := 2; y <= 5; y++ {
			img.Set(x, y, c)
		}
	}
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			if dx*dx+dy*dy <= 9 {
				img.Set(3+dx, 13+dy, c)
				img.Set(13+dx, 13+dy, c)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
