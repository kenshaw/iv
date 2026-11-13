package lottie

import (
	"context"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage"
	"github.com/xo/okage/lottie"
)

// animate plays the timeline of the composition, and draws each frame while
// the frame before it is shown.
//
// The frames are drawn at the size that they are shown at, as a still is. A
// frame that takes longer to draw than the timeline allows is dropped by the
// playback, and does not slow the animation down.
func animate(ctx context.Context, c *lottie.Composition, w, h int) (any, error) {
	total, frame := c.FrameCount(), c.FrameDuration()
	if frame == 0 {
		return nil, okage.ErrLottieEmpty
	}
	// --fps, where it was passed, replaces the rate of the composition
	delay := ivctx.AnimateDelay(ctx, frame)
	ivctx.Logf(ctx, "lottie: %dx%d, %d frames at %g fps", w, h, total, c.FrameRate)
	return &decoder.Anim{
		Frames: decoder.Produce(func(ctx context.Context, s *decoder.Sink) error {
			for i := range total {
				img, err := c.Image(i, w, h)
				if err != nil {
					return err
				}
				if !s.Send(ctx, img, delay) {
					return nil
				}
			}
			return nil
		}),
		Count:    total,
		Duration: time.Duration(total) * delay,
	}, nil
}
