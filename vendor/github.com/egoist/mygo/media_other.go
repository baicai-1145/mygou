//go:build !darwin

package mygo

// NowPlayingInfo is what the system shows while a track plays.
type NowPlayingInfo struct {
	Title    string
	Artist   string
	Album    string
	Duration float64
	Elapsed  float64
	Playing  bool
}

// MediaCommand is a media key press.
type MediaCommand struct {
	Name     string
	Position float64
}

// MediaCommandTargets is zero outside macOS.
func MediaCommandTargets() int { return 0 }

// MediaKeysAvailable is false outside macOS.
func MediaKeysAvailable() bool { return false }

// SetNowPlaying does nothing outside macOS.
func SetNowPlaying(NowPlayingInfo) {}

// OnMediaCommand does nothing outside macOS.
func OnMediaCommand(func(MediaCommand)) {}
