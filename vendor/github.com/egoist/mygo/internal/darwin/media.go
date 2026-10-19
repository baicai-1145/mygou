//go:build darwin

package darwin

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Now Playing and the media keys use MediaPlayer's own centres: the system
// shows the track in Control Centre, and routes F7/F8/F9 and the AirPods
// and headphone buttons to the app while it plays.

var (
	mediaOnce   sync.Once
	mediaOK     bool
	mediaKeys   mediaKeySet
	mediaBlocks []objc.Block
	mediaMu     sync.Mutex
	mediaFn     func(cmd string, position float64)
	mediaTarget int

	msgNumberDouble  func(cls id, s objc.SEL, v float64) id
	msgNumberBool    func(cls id, s objc.SEL, v bool) id
	msgEventPosition func(obj id, s objc.SEL) float64
)

type mediaKeySet struct {
	title, artist, album, duration, elapsed, rate id
}

func mediaKey(lib uintptr, name string) id {
	p, err := purego.Dlsym(lib, name)
	if err != nil {
		return 0
	}
	return *(*id)(unsafe.Pointer(p))
}

// loadMedia opens MediaPlayer and caches the now playing keys.
func loadMedia() bool {
	mediaOnce.Do(func() {
		load()
		lib, err := purego.Dlopen(
			"/System/Library/Frameworks/MediaPlayer.framework/MediaPlayer",
			purego.RTLD_GLOBAL|purego.RTLD_NOW,
		)
		if err != nil || objc.GetClass("MPNowPlayingInfoCenter") == 0 {
			return
		}
		purego.RegisterFunc(&msgNumberDouble, msgSendAddr)
		purego.RegisterFunc(&msgNumberBool, msgSendAddr)
		purego.RegisterFunc(&msgEventPosition, msgSendAddr)
		mediaKeys = mediaKeySet{
			title:    mediaKey(lib, "MPMediaItemPropertyTitle"),
			artist:   mediaKey(lib, "MPMediaItemPropertyArtist"),
			album:    mediaKey(lib, "MPMediaItemPropertyAlbumTitle"),
			duration: mediaKey(lib, "MPMediaItemPropertyPlaybackDuration"),
			elapsed:  mediaKey(lib, "MPNowPlayingInfoPropertyElapsedPlaybackTime"),
			rate:     mediaKey(lib, "MPNowPlayingInfoPropertyPlaybackRate"),
		}
		mediaOK = true
	})
	return mediaOK
}

// MediaCommandTargets counts the command handlers that the remote command
// centre accepted, to check that the media keys reach the app.
func MediaCommandTargets() int { return mediaTarget }

// MediaKeysAvailable reports whether the MediaPlayer centres loaded.
func MediaKeysAvailable() bool { return loadMedia() }

// SetNowPlaying publishes the current track to the system.
func SetNowPlaying(title, artist, album string, duration, elapsed float64, playing bool) {
	if !loadMedia() {
		return
	}
	withPool(func() {
		center := send(class("MPNowPlayingInfoCenter"), "defaultCenter")
		dict := autorelease(send(send(class("NSMutableDictionary"), "alloc"), "init"))
		set := func(key, value id) {
			if key != 0 && value != 0 {
				send(dict, "setObject:forKey:", uintptr(value), uintptr(key))
			}
		}
		set(mediaKeys.title, nsString(title))
		set(mediaKeys.artist, nsString(artist))
		set(mediaKeys.album, nsString(album))
		number := func(v float64) id {
			return msgNumberDouble(class("NSNumber"), sel("numberWithDouble:"), v)
		}
		set(mediaKeys.duration, number(duration))
		set(mediaKeys.elapsed, number(elapsed))
		rate := 0.0
		if playing {
			rate = 1
		}
		set(mediaKeys.rate, number(rate))
		send(center, "setNowPlayingInfo:", uintptr(dict))
	})
}

func onMediaCommand(fn func(string, float64)) {
	if !loadMedia() {
		return
	}
	mediaMu.Lock()
	mediaFn = fn
	mediaMu.Unlock()
	withPool(func() {
		center := send(class("MPRemoteCommandCenter"), "sharedCommandCenter")
		add := func(selector, name string, seek bool) {
			cmd := send(center, selector)
			if cmd == 0 {
				return
			}
			send(cmd, "setEnabled:", 1)
			block := objc.NewBlock(func(_ objc.Block, ev id) int64 {
				pos := 0.0
				if seek {
					pos = msgEventPosition(ev, sel("positionTime"))
				}
				mediaMu.Lock()
				f := mediaFn
				mediaMu.Unlock()
				if f != nil {
					f(name, pos)
				}
				return 0
			})
			mediaBlocks = append(mediaBlocks, block)
			if send(cmd, "addTargetWithHandler:", uintptr(block)) != 0 {
				mediaTarget++
			}
		}
		add("playCommand", "play", false)
		add("pauseCommand", "pause", false)
		add("togglePlayPauseCommand", "toggle", false)
		add("nextTrackCommand", "next", false)
		add("previousTrackCommand", "prev", false)
		add("changePlaybackPositionCommand", "seek", true)
	})
	runtime.KeepAlive(mediaBlocks)
}

// OnMediaCommand registers the handler for play/pause/next/prev keys.
func OnMediaCommand(fn func(cmd string, position float64)) {
	onMediaCommand(fn)
}
