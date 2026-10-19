// mygou is a desktop client for the KuGou concept edition: a Go backend
// that talks to KuGou and plays the music, and a web frontend drawn in the
// design of YesPlayMusic.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/egoist/mygo"

	"mygou/internal/kugou"
	"mygou/internal/player"
	"mygou/internal/service"
)

//go:embed all:frontend/dist
var dist embed.FS

func main() {
	// Restrict Go runtime heap footprint aggressively and reclaim memory
	debug.SetMemoryLimit(64 * 1024 * 1024)
	debug.SetGCPercent(50)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			debug.FreeOSMemory()
		}
	}()

	dataDir := dataDir()
	cacheDir := cacheDir()

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	client, err := kugou.NewClient(dataDir)
	client.Debug = os.Getenv("MYGOU_HTTP_DEBUG") == "1"
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("MYGOU_DEBUG") == "1" {
		client.Debug = true
	}
	engine := player.New(client, cacheDir, os.Getenv("MYGOU_QUALITY"))
	app := service.New(client, engine, dataDir)

	// A dfid is what /song/url asks for; register in the background so the
	// window opens at once.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := client.EnsureDFID(ctx, dataDir); err != nil {
			log.Println("register device:", err)
		}
	}()

	mygo.Bind(app)

	bar = newShim(app, dataDir)
	service.RegisterStatusBarLyricHook(func(on bool) {
		if bar != nil {
			bar.toggleLyric(on)
		}
	})
	service.RegisterFullScreenHooks(
		func() bool {
			return mainWin != nil && mainWin.IsFullScreen()
		},
		func() bool {
			if mainWin == nil {
				return false
			}
			mainWin.ToggleFullScreen()
			return mainWin.IsFullScreen()
		},
	)

	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		quitting = true
	})

	// Closing the window leaves the app in the menu bar, playing.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			showMain()
		}
	})
	mygo.App.WhenReady(func() {
		mygo.OnMediaCommand(func(c mygo.MediaCommand) {
			if os.Getenv("MYGOU_LOG") == "1" {
				log.Printf("media key: %s", c.Name)
			}
			switch c.Name {
			case "toggle":
				app.Toggle()
			case "play":
				app.Play()
			case "pause":
				app.Pause()
			case "next":
				app.Next()
			case "prev":
				app.Prev()
			case "seek":
				app.Seek(c.Position)
			}
		})
		if mygo.MediaKeysAvailable() {
			log.Printf("media keys: on, targets=%d", mygo.MediaCommandTargets())
		}
		bar.ready()
		openWindow()
		if os.Getenv("MYGOU_PANEL") == "1" {
			time.AfterFunc(time.Second, bar.togglePanel)
		}
		if v := os.Getenv("MYGOU_EXIT"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				time.AfterFunc(time.Duration(secs)*time.Second, mygo.App.Quit)
			}
		}
		if v := os.Getenv("MYGOU_AUTOCLOSE"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				time.AfterFunc(time.Duration(secs)*time.Second, func() {
					if mainWin != nil {
						mainWin.Close()
					}
				})
			}
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

var (
	mainWin  *mygo.Window
	bar      *shim
	quitting bool
)

// showMain opens the window again, or focuses it when it is open.
func showMain() {
	mygo.App.Dock.Show()
	if mainWin != nil {
		mainWin.Show()
		mainWin.Focus()
		return
	}
	openWindow()
}

func openWindow() {
	url := os.Getenv("MYGOU_URL")
	if url == "" {
		url = "/"
	}
	sub, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	mygo.SetFrontend(sub)
	mainWin = mygo.NewWindow(mygo.WindowOptions{
		Title:                "mygou",
		URL:                  url,
		Width:                1280,
		Height:               820,
		MinWidth:             960,
		MinHeight:            620,
		TitleBarStyle:        mygo.TitleBarHiddenInset,
		TrafficLightPosition: &mygo.Point{X: 18, Y: 20},
	})
	mainWin.OnClose(func(e *mygo.CloseEvent) {
		// Scheme 1: Let the window truly close and destroy its WKWebView,
		// freeing mygo://localhost WebContent memory down to 0 MB.
		// Native Go player keeps playing in background menu bar.
		mygo.App.Dock.Hide()
	})
	mainWin.OnClosed(func() {
		mainWin = nil
		debug.FreeOSMemory()
	})
	mainWin.OnEnterFullScreen(func() {
		if mainWin != nil {
			go func() {
				defer func() { _ = recover() }()
				if mainWin != nil && mainWin.Page() != nil {
					mainWin.Page().Eval("window.dispatchEvent(new CustomEvent('mygo:fullscreen', { detail: { fullscreen: true } }))")
				}
			}()
		}
	})
	mainWin.OnLeaveFullScreen(func() {
		if mainWin != nil {
			go func() {
				defer func() { _ = recover() }()
				if mainWin != nil && mainWin.Page() != nil {
					mainWin.Page().Eval("window.dispatchEvent(new CustomEvent('mygo:fullscreen', { detail: { fullscreen: false } }))")
				}
			}()
		}
	})
	mainWin.OnClosed(func() { mainWin = nil })
}

func dataDir() string {
	if d := os.Getenv("MYGOU_DATA"); d != "" {
		return d
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "mygou")
}

func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "mygou")
}
