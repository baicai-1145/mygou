//go:build darwin

package mygo

import "github.com/egoist/mygo/internal/darwin"

// NowPlayingInfo is what the system shows in Control Centre and the lock
// screen while a track plays.
type NowPlayingInfo struct {
	Title    string
	Artist   string
	Album    string
	Duration float64
	Elapsed  float64
	Playing  bool
}

// MediaCommand is a media key press: Play, Pause, Toggle, Next, Prev or
// Seek, whose Position is the new time in seconds.
type MediaCommand struct {
	Name     string
	Position float64
}

// MediaKeysAvailable reports whether this system routes media keys to the
// app (macOS 10.12 and later).
func MediaKeysAvailable() bool { return darwin.MediaKeysAvailable() }

// MediaCommandTargets reports how many remote command handlers are
// registered, for diagnostics.
func MediaCommandTargets() int { return darwin.MediaCommandTargets() }

// SetNowPlaying publishes the current track to the system.
func SetNowPlaying(info NowPlayingInfo) {
	darwin.SetNowPlaying(info.Title, info.Artist, info.Album, info.Duration, info.Elapsed, info.Playing)
}

// OnMediaCommand handles the media keys and headphone buttons while the
// app plays. Call it once, on the main thread.
func OnMediaCommand(fn func(MediaCommand)) {
	darwin.OnMediaCommand(func(name string, position float64) {
		fn(MediaCommand{Name: name, Position: position})
	})
}
