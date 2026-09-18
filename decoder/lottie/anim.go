package lottie

import (
	"context"
	"errors"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/lottie"
)

// animate plays the animation's own timeline, rasterizing each frame as the
// one before it is displayed.
//
// A composition has no pixels of its own, so the frames are drawn at the size
// they are shown at rather than resampled to it, the same as a still is -- and
// a frame that turns out to take longer to draw than the timeline allows for
// is dropped by the playback rather than slowing the animation down.
func animate(ctx context.Context, rend *lottie.Renderer, buf []byte, w, h int) (any, error) {
	a, err := rend.Load(ctx, buf, lottie.Options{Width: w, Height: h})
	if err != nil {
		return nil, err
	}
	total := int(a.TotalFrames())
	fps := a.FPS()
	if total <= 0 || fps <= 0 {
		return nil, errors.Join(errors.New("animation has no timeline"), a.Close(ctx))
	}
	w, h = a.Size()
	// --fps, where it was passed, replaces the rate the composition declares
	delay := ivctx.AnimateDelay(ctx, time.Duration(float32(time.Second)/fps))
	ivctx.Logf(ctx, "lottie: %dx%d, %d frames at %g fps", w, h, total, fps)
	return &decoder.Anim{
		Frames: decoder.Produce(func(ctx context.Context, s *decoder.Sink) error {
			for i := range total {
				img, err := a.Frame(ctx, float32(i))
				if err != nil {
					return err
				}
				if !s.Send(img, delay) {
					return nil
				}
			}
			return nil
		}),
		Count:    total,
		Duration: time.Duration(total) * delay,
		// the animation is loaded for as long as it is played, a looping
		// playback rendering its frames again rather than keeping them; the
		// renderer behind it belongs to the decoder and outlives it
		Close: func() error {
			return a.Close(context.WithoutCancel(ctx))
		},
	}, nil
}
