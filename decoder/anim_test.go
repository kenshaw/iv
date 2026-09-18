package decoder

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

func TestProduce(t *testing.T) {
	frames := Produce(func(_ context.Context, s *Sink) error {
		for i := range 4 {
			if !s.Send(image.NewRGBA(image.Rect(0, 0, 1+i, 1)), time.Duration(i)*time.Second) {
				return errors.New("send refused")
			}
		}
		return nil
	})
	var widths []int
	var delays []time.Duration
	for frame, err := range frames(animCtx(t, ivctx.ModeNone)) {
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		widths = append(widths, frame.Image.Bounds().Dx())
		delays = append(delays, frame.Delay)
	}
	switch {
	case len(widths) != 4:
		t.Fatalf("expected 4 frames, got %d", len(widths))
	case widths[0] != 1 || widths[3] != 4:
		t.Errorf("expected the frames in order, got %v", widths)
	case delays[3] != 3*time.Second:
		t.Errorf("expected the delay to be carried with the frame, got %v", delays[3])
	}
}

// TestProduceStops checks that a playback that gives up part way through takes
// the producer with it, rather than leaving it making frames for a screen that
// has moved on.
func TestProduceStops(t *testing.T) {
	stopped := make(chan struct{})
	frames := Produce(func(ctx context.Context, s *Sink) error {
		defer close(stopped)
		for {
			if !s.Send(image.NewRGBA(image.Rect(0, 0, 1, 1)), 0) {
				return ctx.Err()
			}
		}
	})
	var n int
	for range frames(animCtx(t, ivctx.ModeNone)) {
		if n++; n == 3 {
			break
		}
	}
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("expected the producer to stop with the playback")
	}
	if n != 3 {
		t.Errorf("expected 3 frames, got %d", n)
	}
}

// TestProduceError checks that a producer's error reaches the playback after
// the frames it managed to send.
func TestProduceError(t *testing.T) {
	errExpected := errors.New("expected")
	frames := Produce(func(_ context.Context, s *Sink) error {
		s.Send(image.NewRGBA(image.Rect(0, 0, 1, 1)), 0)
		return errExpected
	})
	var n int
	var err error
	for _, v := range frames(animCtx(t, ivctx.ModeNone)) {
		if err = v; err != nil {
			break
		}
		n++
	}
	switch {
	case n != 1:
		t.Errorf("expected 1 frame ahead of the error, got %d", n)
	case !errors.Is(err, errExpected):
		t.Errorf("expected the producer's error, got: %v", err)
	}
}

// TestSinkScales checks that a frame is scaled to the display size as it is
// produced, that being work the playback would otherwise be doing between one
// frame and the next.
func TestSinkScales(t *testing.T) {
	frames := Produce(func(_ context.Context, s *Sink) error {
		s.Send(image.NewRGBA(image.Rect(0, 0, 400, 200)), 0)
		return nil
	})
	ctx := animCtx(t, ivctx.ModeBestFit)
	ivctx.Get(ctx).Width, ivctx.Get(ctx).Height = 100, 100
	for frame := range frames(ctx) {
		if b := frame.Image.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
			t.Errorf("expected the frame scaled to 100x50, got %s", b)
		}
	}
}

// animCtx returns a context configured the way the pipeline configures one for
// playback.
func animCtx(t *testing.T, mode ivctx.Mode) context.Context {
	t.Helper()
	c := ivctx.New()
	c.Mode, c.Animate = &mode, true
	return ivctx.WithConfig(t.Context(), c)
}
