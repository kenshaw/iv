package ivcmd

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/rasterm"
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

// TestPlaySlowProducer checks that an animation whose frames are made more
// slowly than it plays draws every frame, and does not drop them. A dropped
// frame was already made, so dropping it saves nothing, and a pass that
// dropped every late frame showed only the first one.
func TestPlaySlowProducer(t *testing.T) {
	const frames = 6
	p, _ := testPlayer(t, 0)
	anim := testAnim(frames, 5*time.Millisecond)
	anim.Frames = slowFrames(anim.Frames, 30*time.Millisecond)
	switch n, err := p.play(t.Context(), anim, false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != frames:
		t.Errorf("expected %d frames drawn, got %d with %d dropped", frames, n, p.dropped)
	}
}

// TestPlayVideo checks that a video keeps its own length: a frame that the
// decoder made late is dropped, where any other animation draws it (D43).
func TestPlayVideo(t *testing.T) {
	const frames = 6
	p, _ := testPlayer(t, 0)
	anim := testAnim(frames, 5*time.Millisecond)
	anim.Video = true
	anim.Frames = slowFrames(anim.Frames, 30*time.Millisecond)
	switch n, err := p.play(t.Context(), anim, false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case p.dropped == 0:
		t.Error("expected late frames dropped")
	case n+p.dropped != frames:
		t.Errorf("expected %d frames accounted for, got %d drawn and %d dropped", frames, n, p.dropped)
	}
}

// TestPlayFirstFrame checks that the clock starts with the first frame, so
// that a decoder that is slow to start, as ffmpeg is, makes no frame late.
func TestPlayFirstFrame(t *testing.T) {
	const frames = 4
	p, _ := testPlayer(t, 0)
	anim := testAnim(frames, 50*time.Millisecond)
	anim.Video = true
	next := anim.Frames
	anim.Frames = func(ctx context.Context) iter.Seq2[decoder.Frame, error] {
		time.Sleep(100 * time.Millisecond)
		return next(ctx)
	}
	switch n, err := p.play(t.Context(), anim, false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != frames:
		t.Errorf("expected %d frames drawn, got %d with %d dropped", frames, n, p.dropped)
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

// TestKittyPlays checks which terminals get the frames as one Kitty
// animation: the ones that report it, kitty, and Ghostty 1.4.0 and later.
func TestKittyPlays(t *testing.T) {
	for _, test := range []struct {
		name string
		term rasterm.Terminal
		exp  bool
	}{
		{"reported", rasterm.Terminal{Protocol: rasterm.Kitty, KittyAnimation: true}, true},
		{"kitty", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "kitty", Version: "0.42.2"}, true},
		{"ghostty", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "ghostty", Version: "1.4.0"}, true},
		{"ghostty dev", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "ghostty", Version: "1.4.0-dev+0e1a2b3"}, true},
		{"ghostty 2", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "ghostty", Version: "2.0.1"}, true},
		{"old ghostty", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "ghostty", Version: "1.3.1"}, false},
		{"ghostty with no version", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "ghostty"}, false},
		{"wezterm", rasterm.Terminal{Protocol: rasterm.Kitty, Name: "WezTerm"}, false},
		{"forced", rasterm.Terminal{Protocol: rasterm.Kitty}, false},
		{"iterm", rasterm.Terminal{Protocol: rasterm.ITerm, Name: "kitty", KittyAnimation: true}, false},
		{"none", rasterm.Terminal{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if ok := kittyPlays(test.term); ok != test.exp {
				t.Errorf("expected %t, got %t", test.exp, ok)
			}
		})
	}
}

// TestPlayKitty checks that an animation goes to a Kitty terminal as one
// animation: the root frame, each other frame once, and the end, which plays
// the frames once. rasterm sends a frame in chunks, and a=f,f= starts the
// first chunk of each frame.
func TestPlayKitty(t *testing.T) {
	a, buf := &Args{}, new(bytes.Buffer)
	if err := a.playKitty(t.Context(), buf, rasterm.KittyEncoder{}, testAnim(3, 10*time.Millisecond), "anim.gif"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	s := buf.String()
	switch {
	case !strings.HasPrefix(s, "anim.gif:\n"):
		t.Error("expected the name written above the animation")
	case strings.Count(s, "a=T,") != 1:
		t.Errorf("expected 1 root frame, got %d", strings.Count(s, "a=T,"))
	case strings.Count(s, "a=f,f=") != 2:
		t.Errorf("expected 2 frames, got %d", strings.Count(s, "a=f,f="))
	case !strings.Contains(s, ",s=3,v=2\x1b"):
		t.Error("expected the frames played once")
	case strings.Contains(s, ",s=1\x1b"):
		t.Error("expected the animation not stopped")
	}
}

// TestPlayKittyLoop checks that --loop has the terminal play the frames
// until the playback is interrupted, and that the interrupt stops the
// animation on the frame that shows.
func TestPlayKittyLoop(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	a, buf := &Args{Loop: true}, new(bytes.Buffer)
	if err := a.playKitty(ctx, buf, rasterm.KittyEncoder{}, testAnim(3, time.Millisecond), ""); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	s := buf.String()
	end, stop := strings.Index(s, ",s=3,v=1\x1b"), strings.Index(s, ",s=1\x1b")
	switch {
	case end < 0:
		t.Error("expected the frames played until interrupted")
	case stop < end:
		t.Error("expected the animation stopped after the interrupt")
	}
}

// TestPlayKittyMax checks that an animation longer than a pass plays for is
// cut off, as [player.play] cuts it off.
func TestPlayKittyMax(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	a, buf := &Args{}, new(bytes.Buffer)
	if err := a.playKitty(ctx, buf, rasterm.KittyEncoder{}, testAnim(3, ivctx.AnimateMax/2+time.Second), ""); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// the third frame starts after the end of the pass
	if n := strings.Count(buf.String(), "a=f,f="); n != 1 {
		t.Errorf("expected 1 frame after the root frame, got %d", n)
	}
}

// TestPlayStream checks that the stream draws the frames into one Kitty
// image: the name and the image once, then each frame written into the frame
// that is not shown, and shown, with nothing cleared.
func TestPlayStream(t *testing.T) {
	buf := new(bytes.Buffer)
	p := &player{w: buf, name: "anim.gif", stream: true, kitty: rasterm.KittyEncoder{}}
	switch n, err := p.play(t.Context(), testAnim(3, 20*time.Millisecond), false); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case n != 3:
		t.Errorf("expected 3 frames drawn, got %d", n)
	}
	s := buf.String()
	switch {
	case !strings.HasPrefix(s, "anim.gif:\n") || strings.Count(s, "anim.gif:") != 1:
		t.Error("expected the name written once, above the image")
	case strings.Count(s, "a=T,") != 1:
		t.Errorf("expected 1 image, got %d", strings.Count(s, "a=T,"))
	case strings.Count(s, "a=f,f=") != 2:
		t.Errorf("expected 2 frames written, got %d", strings.Count(s, "a=f,f="))
	case !strings.Contains(s, ",c=2\x1b") || !strings.Contains(s, ",r=1,X=1,"):
		t.Error("expected the second frame shown, then the first frame written again and shown")
	case strings.Contains(s, cursorHome), strings.Contains(s, kittyDelete):
		t.Error("expected nothing cleared")
	}
}

// TestAnimateMode checks the values of --animate-mode: case and space do not
// matter, empty is auto, and any other value is an error.
func TestAnimateMode(t *testing.T) {
	for _, test := range []struct {
		s, exp string
	}{
		{"", animateAuto},
		{"auto", animateAuto},
		{" Kitty ", animateKitty},
		{"FRAMES", animateFrames},
		{"stream", animateStream},
	} {
		got, err := animateMode(test.s)
		switch {
		case err != nil:
			t.Errorf("%q: expected no error, got: %v", test.s, err)
		case got != test.exp:
			t.Errorf("%q: expected %q, got %q", test.s, test.exp, got)
		}
	}
	if _, err := animateMode("sixel"); !errors.Is(err, ErrUnknownAnimateMode) {
		t.Errorf("expected %v, got: %v", ErrUnknownAnimateMode, err)
	}
}

// TestDeleteImages checks that only Kitty deletes the images before a frame,
// and that the delete goes through the passthrough of tmux in tmux.
func TestDeleteImages(t *testing.T) {
	for _, test := range []struct {
		name string
		term rasterm.Terminal
		exp  string
	}{
		{"kitty", rasterm.Terminal{Protocol: rasterm.Kitty}, kittyDelete},
		{"tmux", rasterm.Terminal{Protocol: rasterm.Kitty, Tmux: true}, "\x1bPtmux;\x1b\x1b_Ga=d\x1b\x1b\\\x1b\\"},
		{"sixel", rasterm.Terminal{Protocol: rasterm.Sixel}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := deleteImages(test.term); got != test.exp {
				t.Errorf("expected %q, got %q", test.exp, got)
			}
		})
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

// slowFrames returns frames that each take d to make, after the first.
func slowFrames(frames func(context.Context) iter.Seq2[decoder.Frame, error], d time.Duration) func(context.Context) iter.Seq2[decoder.Frame, error] {
	return func(ctx context.Context) iter.Seq2[decoder.Frame, error] {
		return func(yield func(decoder.Frame, error) bool) {
			first := true
			for frame, err := range frames(ctx) {
				if !first {
					time.Sleep(d)
				}
				first = false
				if !yield(frame, err) {
					return
				}
			}
		}
	}
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
