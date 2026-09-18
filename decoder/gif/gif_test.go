package gif

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestDecodeAnim decodes an animated gif and checks that what comes back is
// the animation whole: every frame the file holds, each a picture of the
// canvas rather than the patch of it the file stores, and a delay for each.
func TestDecodeAnim(t *testing.T) {
	ctx := animCtx(t)
	f, err := os.Open("../../testdata/gif/animated.gif")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	res, err := decode(ctx, f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	anim, ok := res.(*decoder.Anim)
	if !ok {
		t.Fatalf("expected an animation, got %T", res)
	}
	if anim.Count < 2 {
		t.Fatalf("expected more than one frame, got %d", anim.Count)
	}
	var n int
	var total time.Duration
	var bounds string
	for frame, err := range anim.Frames(ctx) {
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if frame.Delay <= 0 {
			t.Errorf("frame %d: expected a delay, got %v", n, frame.Delay)
		}
		if b := frame.Image.Bounds().String(); n == 0 {
			bounds = b
		} else if b != bounds {
			t.Errorf("frame %d: expected the canvas %s, got %s", n, bounds, b)
		}
		n, total = n+1, total+frame.Delay
	}
	switch {
	case n != anim.Count:
		t.Errorf("expected %d frames, got %d", anim.Count, n)
	case total != anim.Duration:
		t.Errorf("expected the frames to add up to %v, got %v", anim.Duration, total)
	}
}

// TestDecodeStill checks that a gif decoded without an animation having been
// asked for is still a single image.
func TestDecodeStill(t *testing.T) {
	f, err := os.Open("../../testdata/gif/animated.gif")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	switch res, err := decode(t.Context(), f); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case res == nil:
		t.Fatal("expected an image")
	}
}

// animCtx returns a context configured the way the pipeline configures one for
// playback, with no scaling so that the frames keep the size the file gives
// them.
func animCtx(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	mode := ivctx.ModeNone
	c.Mode, c.Animate = &mode, true
	return ivctx.WithConfig(t.Context(), c)
}
