package decoder

import (
	"context"
	"errors"
	"image"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

// DefaultDelay is the delay of a frame whose file gives it no delay, as a
// browser shows it.
const DefaultDelay = 100 * time.Millisecond

// ErrNoFrames is the error of an animation with no frames.
var ErrNoFrames = errors.New("no frames")

// Frames returns the result of a decoder for an animation whose frames are
// whole images, with the delay of each (D61).
//
// When the pipeline decodes for playback, the result is an [Anim] that plays
// the frames. Otherwise it is the frames as a set of images, from which the
// pipeline takes the frame that --page selects, or the first. A delay that is
// not positive is [DefaultDelay], and --fps replaces each delay.
func Frames(ctx context.Context, frames []image.Image, delays []time.Duration) (any, error) {
	switch {
	case len(frames) == 0:
		return nil, ErrNoFrames
	case !ivctx.Animate(ctx):
		return frames, nil
	}
	d := make([]time.Duration, len(frames))
	var total time.Duration
	for i := range frames {
		delay := DefaultDelay
		if i < len(delays) && delays[i] > 0 {
			delay = delays[i]
		}
		d[i] = ivctx.AnimateDelay(ctx, delay)
		total += d[i]
	}
	return &Anim{
		Frames: Produce(func(ctx context.Context, s *Sink) error {
			for i, frame := range frames {
				if !s.Send(ctx, frame, d[i]) {
					return nil
				}
			}
			return nil
		}),
		Count:    len(frames),
		Duration: total,
	}, nil
}

// Seconds returns delays in seconds as durations.
func Seconds(delays []float64) []time.Duration {
	d := make([]time.Duration, len(delays))
	for i, v := range delays {
		d[i] = time.Duration(v * float64(time.Second))
	}
	return d
}

// Milliseconds returns delays in milliseconds as durations.
func Milliseconds(delays []int) []time.Duration {
	d := make([]time.Duration, len(delays))
	for i, v := range delays {
		d[i] = time.Duration(v) * time.Millisecond
	}
	return d
}

// Animated reports whether a decoder for a format that can hold frames must
// decode them all: for playback, or for a frame that --page selects.
func Animated(ctx context.Context) bool {
	return ivctx.Animate(ctx) || ivctx.Get(ctx).Page != 0
}
