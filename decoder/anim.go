package decoder

import (
	"context"
	"image"
	"iter"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

// lookahead is how many frames a producer may run ahead of the display.
//
// Making a frame -- seeking a video, compositing a gif, scaling either one to
// the terminal -- happens while the frames before it are on screen, and this
// is how much of that head start is worth keeping: enough to ride out a frame
// that takes unusually long to produce, and not so much that an animation cut
// short has spent its time making frames nobody saw.
const lookahead = 8

// Frame is one frame of an animation, scaled to the size it is displayed at,
// with the time it is shown for.
type Frame struct {
	Image image.Image
	Delay time.Duration
}

// Anim is a decoded animation, returned by a decoder in place of an image when
// the pipeline is decoding for playback -- see [ivctx.Animate].
type Anim struct {
	// Frames returns the animation's frames, in order. It is called once for
	// each pass, so a looping playback produces the frames again rather than
	// holding every one of them in memory: an animation is as many as
	// [ivctx.AnimateMax] leaves room for, each the size of the terminal,
	// which is more than there is any need to keep.
	Frames func(context.Context) iter.Seq2[Frame, error]
	// Count is how many frames a pass yields, 0 when that is not known ahead
	// of time.
	Count int
	// Duration is how long a pass runs for, 0 when not known.
	Duration time.Duration
	// Close releases whatever produced the frames -- a spooled copy of the
	// source, a loaded animation -- once the playback has finished with it.
	// Nil when there is nothing to release.
	Close func() error
}

// Sink is what an animation's producer sends its frames to.
type Sink struct {
	ctx context.Context
	ch  chan Frame
}

// Context returns the producer's context, which is cancelled once the playback
// has ended.
func (s *Sink) Context() context.Context {
	return s.ctx
}

// Send hands a frame to the playback, blocking while the playback is still
// working through the frames already sent.
//
// The frame is scaled and composited here, in the producer's goroutine, so
// that the work of preparing one frame overlaps the drawing of the one before
// it.
//
// Reports whether the frame was taken. A false return means the playback has
// ended -- interrupted, or out of time -- and the producer should stop.
func (s *Sink) Send(img image.Image, delay time.Duration) bool {
	frame := Frame{
		Image: ivctx.AddBackground(s.ctx, ivctx.Mime(s.ctx), ivctx.Fit(s.ctx, img)),
		Delay: delay,
	}
	select {
	case s.ch <- frame:
		return true
	case <-s.ctx.Done():
		return false
	}
}

// Produce returns a frame func for [Anim.Frames] that runs f in a goroutine of
// its own, feeding the frames it sends to the playback as they are made.
//
// f is started afresh for every pass and is handed a context that is cancelled
// when the pass ends, so that a producer blocked on [Sink.Send] -- and the
// external commands it is running -- stop along with the playback rather than
// running on to the end of an animation that is no longer being watched.
func Produce(f func(context.Context, *Sink) error) func(context.Context) iter.Seq2[Frame, error] {
	return func(ctx context.Context) iter.Seq2[Frame, error] {
		return func(yield func(Frame, error) bool) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			s := &Sink{
				ctx: ctx,
				ch:  make(chan Frame, lookahead),
			}
			errch := make(chan error, 1)
			go func() {
				defer close(s.ch)
				errch <- f(ctx, s)
			}()
			for frame := range s.ch {
				if !yield(frame, nil) {
					return
				}
			}
			// a producer that stopped because the playback did has nothing to
			// report but that
			if err := <-errch; err != nil && ctx.Err() == nil {
				yield(Frame{}, err)
			}
		}
	}
}
