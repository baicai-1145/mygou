// Package flac decodes FLAC files for package audio, with
// github.com/mewkiz/flac. Import it for its side effect:
//
//	import _ "github.com/egoist/mygo/audio/flac"
package flac

import (
	"io"

	"github.com/egoist/mygo/audio"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
)

func init() {
	audio.RegisterFormat("flac", "fLaC", func(r io.ReadSeeker) (audio.Decoder, error) {
		return NewDecoder(r)
	})
}

// NewDecoder returns a decoder of the FLAC file r, which decodes to the
// file's own sample rate and its first two channels.
func NewDecoder(r io.ReadSeeker) (audio.Decoder, error) {
	stream, err := flac.New(r)
	if err != nil {
		return nil, err
	}
	info := stream.Info
	d := &decoder{
		rs:       r,
		stream:   stream,
		rate:     int(info.SampleRate),
		channels: int(info.NChannels),
		total:    int64(info.NSamples),
	}
	if d.channels > 2 {
		d.channels = 2
	}
	d.scale = float32(1) / float32(int64(1)<<uint(info.BitsPerSample-1))
	return d, nil
}

type decoder struct {
	rs       io.ReadSeeker
	stream   *flac.Stream
	rate     int
	channels int
	total    int64
	scale    float32

	frame *frame.Frame
	at    int   // sample in frame
	pos   int64 // frames delivered from the start
}

func (d *decoder) SampleRate() int { return d.rate }
func (d *decoder) Channels() int   { return d.channels }
func (d *decoder) Len() int64      { return d.total }

// next returns the following frame, or io.EOF.
func (d *decoder) next() error {
	f, err := d.stream.ParseNext()
	if err != nil {
		return err
	}
	d.frame = f
	d.at = 0
	return nil
}

func (d *decoder) Read(buf []float32) (int, error) {
	n := 0
	for n+1 < len(buf) {
		if d.frame == nil || len(d.frame.Subframes) == 0 || d.at >= len(d.frame.Subframes[0].Samples) {
			if err := d.next(); err != nil {
				// The last frame is followed by trailing bytes that are not
				// a frame; once every sample is out, that is the end.
				if d.total > 0 && d.pos >= d.total {
					return n, io.EOF
				}
				return n, err
			}
			if len(d.frame.Subframes) == 0 {
				continue
			}
		}
		left := d.frame.Subframes[0].Samples
		var right []int32
		if d.channels > 1 && len(d.frame.Subframes) > 1 {
			right = d.frame.Subframes[1].Samples
		}
		limit := len(left)
		if right != nil && len(right) < limit {
			limit = len(right)
		}
		for d.at < limit && n+1 < len(buf) {
			buf[n] = float32(left[d.at]) * d.scale
			if right != nil {
				buf[n+1] = float32(right[d.at]) * d.scale
			} else {
				buf[n+1] = buf[n]
			}
			n += 2
			d.at++
			d.pos++
		}
		if limit == 0 {
			d.at = len(left)
		}
	}
	return n, nil
}

// SetPosition seeks by decoding from the start: FLAC frames have no
// random access without an index, and seeking backwards means starting
// over.
func (d *decoder) SetPosition(frame int64) error {
	if frame < 0 {
		frame = 0
	}
	if frame < d.pos {
		if _, err := d.rs.Seek(0, io.SeekStart); err != nil {
			return err
		}
		stream, err := flac.New(d.rs)
		if err != nil {
			return err
		}
		d.stream = stream
		d.frame = nil
		d.at = 0
		d.pos = 0
	}
	d.frame = nil
	for d.pos < frame {
		if d.frame == nil || len(d.frame.Subframes) == 0 || d.at >= len(d.frame.Subframes[0].Samples) {
			if err := d.next(); err != nil {
				return err
			}
			if len(d.frame.Subframes) == 0 {
				continue
			}
		}
		step := int64(len(d.frame.Subframes[0].Samples) - d.at)
		if step == 0 {
			continue
		}
		if d.pos+step > frame {
			step = frame - d.pos
		}
		d.at += int(step)
		d.pos += step
	}
	return nil
}
