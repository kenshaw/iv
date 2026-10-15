package ivcmd

import (
	"bytes"
	"context"
	"image"
	"io"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestPlay checks that a pass draws every frame of an animation it can keep up
// with, and that each frame is written with the clear and the name above it.
//
// Each frame is up for 50ms. A timer on Windows can fire about 16ms late, and
// with frames of 5ms that made a frame late enough to be dropped.
func TestPlay(t *testing.T) {
	p, buf := testPlayer(t, 0)
	switch n, err := p.play(t.Context(), testAnim(6, 50*time.Millisecond), false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != 6:
		t.Errorf("expected 6 frames drawn, got %d", n)
	case p.dropped != 0:
		t.Errorf("expected no frames dropped, got %d", p.dropped)
	}
	s := buf.String()
	switch {
	case strings.Count(s, cursorHome) != 6:
		t.Errorf("expected every frame to start at the top of the screen, got %d", strings.Count(s, cursorHome))
	case strings.Count(s, "anim.gif:\n"+eraseDown) != 6:
		t.Error("expected the name written above each frame, and the screen cleared under it")
	case strings.Count(s, "<frame>") != 6:
		t.Errorf("expected 6 encoded frames, got %d", strings.Count(s, "<frame>"))
	}
}

// TestPlayDrops checks that an animation drawn more slowly than it plays keeps
// to its own speed by dropping the frames it could not get to in time, rather
// than running late by however long the drawing took.
func TestPlayDrops(t *testing.T) {
	const frames = 20
	p, _ := testPlayer(t, 20*time.Millisecond)
	start := time.Now()
	n, err := p.play(t.Context(), testAnim(frames, time.Millisecond), false)
	elapsed := time.Since(start)
	switch {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n == 0:
		t.Fatal("expected at least one frame drawn")
	case n == frames:
		t.Error("expected frames to be dropped")
	case n+p.dropped != frames:
		t.Errorf("expected %d frames accounted for, got %d drawn and %d dropped", frames, n, p.dropped)
	}
	// the whole animation is 20ms of playback, and a frame takes 20ms to
	// draw, so keeping to the schedule means finishing in about the time one
	// or two frames take rather than the 400ms drawing all of them would
	if elapsed > 200*time.Millisecond {
		t.Errorf("expected the pass to keep to the animation's length, took %v", elapsed)
	}
}

// TestPlayMax checks that an animation longer than a pass plays for is cut off
// rather than holding the screen for the whole of its length.
func TestPlayMax(t *testing.T) {
	p, _ := testPlayer(t, 0)
	// the first frame is drawn, and the second belongs past the end of the
	// pass
	switch n, err := p.play(t.Context(), testAnim(2, ivctx.AnimateMax+time.Second), false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != 1:
		t.Errorf("expected 1 frame drawn, got %d", n)
	}
}

// TestPlayInterrupted checks that a playback whose context is cancelled stops
// where it is, without an error: an interrupt is how a loop is meant to end.
func TestPlayInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	p, _ := testPlayer(t, 0)
	switch n, err := p.play(ctx, testAnim(10, time.Millisecond), false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != 0:
		t.Errorf("expected no frames drawn, got %d", n)
	}
}

// testPlayer returns a player writing to a buffer, through an encoder that
// takes draw to write a frame.
func testPlayer(t *testing.T, draw time.Duration) (*player, *bytes.Buffer) {
	t.Helper()
	name := "test-anim-" + t.Name()
	e := encoder.Register(name, encoder.Term(), encoder.ImageEncoder(func(w io.Writer, _ image.Image) error {
		time.Sleep(draw)
		_, err := io.WriteString(w, "<frame>")
		return err
	}))
	t.Cleanup(func() { encoder.Unregister(name) })
	buf := new(bytes.Buffer)
	return &player{
		w:    buf,
		enc:  e,
		name: "anim.gif",
	}, buf
}

// testAnim returns an animation of n identical frames.
func testAnim(n int, delay time.Duration) *decoder.Anim {
	return &decoder.Anim{
		Frames: func(context.Context) iter.Seq2[decoder.Frame, error] {
			return func(yield func(decoder.Frame, error) bool) {
				for range n {
					frame := decoder.Frame{
						Image: image.NewRGBA(image.Rect(0, 0, 2, 2)),
						Delay: delay,
					}
					if !yield(frame, nil) {
						return
					}
				}
			}
		},
		Count:    n,
		Duration: time.Duration(n) * delay,
	}
}
