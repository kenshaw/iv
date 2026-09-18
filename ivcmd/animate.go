package ivcmd

import (
	"bytes"
	"context"
	"image"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/rasterm"
)

// The terminal control sequences animated playback is drawn with.
const (
	clearScreen = "\x1b[2J\x1b[H"
	cursorHome  = "\x1b[H"
	eraseDown   = "\x1b[J"
	hideCursor  = "\x1b[?25l"
	showCursor  = "\x1b[?25h"
	// kittyDelete removes every image the terminal is holding. An erase
	// clears the cells a kitty image was drawn in but not the image itself,
	// which would otherwise still be there underneath the frame that follows
	// it. The other protocols draw into the cells and need nothing of the
	// kind.
	kittyDelete = "\x1b_Ga=d\x1b\\"
)

// animate plays the target's animation on the terminal.
//
// The screen is cleared, the name is written at the top of it, and the frames
// are drawn one after another underneath. A pass runs for as long as the
// animation does, up to [ivctx.AnimateMax], and --loop starts it over until
// the playback is interrupted.
func (args *Args) animate(ctx context.Context, stdout io.Writer, enc *encoder.Entry, t Target) (err error) {
	// control-c ends the playback rather than the process, which is what
	// gives the cursor back and leaves the last frame on screen
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	var anim *decoder.Anim
	var mime string
	if t.IsString {
		anim, mime, err = decoder.DecodeAnimString(ctx, t.Name)
	} else {
		anim, mime, err = decoder.DecodeAnimFile(ctx, t.Name)
	}
	if err != nil {
		return err
	}
	// what an animation holds open -- a spooled copy of the source, a loaded
	// animation -- is released here, and a failure to is worth saying rather
	// than dropping
	if anim.Close != nil {
		defer func() {
			if cerr := anim.Close(); err == nil {
				err = cerr
			}
		}()
	}
	// the frames are scaled and composited as they are produced, which is
	// where the mime type is wanted
	ctx = ivctx.WithMime(ctx, mime)
	p := &player{
		w:   stdout,
		enc: enc,
		// the frames are drawn over the name rather than after it, so it
		// stays where it was written
		name: t.Name,
		// the default encoder takes kitty ahead of the others, so kitty being
		// available is kitty being used
		kitty: rasterm.Kitty.Available(),
	}
	if args.Quiet {
		p.name = ""
	}
	if _, err := io.WriteString(stdout, hideCursor+clearScreen); err != nil {
		return err
	}
	defer io.WriteString(stdout, showCursor)
	for {
		start := time.Now()
		n, err := p.play(ctx, anim, args.Loop)
		ivctx.Logf(ctx, "pass: %d of %d frames in %v, %d dropped", n, anim.Count, time.Since(start), p.dropped)
		switch {
		case err != nil:
			return err
		// an animation of a single frame is a still, and looping it would be
		// redrawing the same image until it was interrupted
		case !args.Loop, n < 2, ctx.Err() != nil:
			return nil
		}
	}
}

// player draws an animation's frames to the terminal.
type player struct {
	w       io.Writer
	enc     *encoder.Entry
	name    string
	kitty   bool
	buf     bytes.Buffer
	dropped int
}

// play runs one pass of the animation, returning the number of frames drawn.
// A pass that will be followed by another holds the last frame for as long as
// it is meant to be up, rather than starting over the instant it is drawn.
//
// Each frame has a moment in the pass it belongs at. Drawing one takes as long
// as it takes -- encoding an image for a terminal is not fast, and how slow it
// is varies with the protocol and the size -- so a frame whose moment has gone
// by while the one before it was still being drawn is dropped instead of
// played late, and the animation keeps to its own speed however many of them
// that costs.
func (p *player) play(ctx context.Context, anim *decoder.Anim, hold bool) (int, error) {
	start, sched := time.Now(), time.Duration(0)
	var drawn int
	p.dropped = 0
	for frame, err := range anim.Frames(ctx) {
		switch {
		case err != nil:
			return drawn, err
		case ctx.Err() != nil:
			return drawn, nil
		// an animation longer than a pass plays for is cut off here rather
		// than holding the screen for the length of a feature film
		case sched >= ivctx.AnimateMax:
			return drawn, nil
		}
		at := sched
		sched += frame.Delay
		switch {
		case drawn == 0: // the first frame is never late
		case time.Since(start) >= sched: // its moment has been and gone
			p.dropped++
			continue
		}
		if !wait(ctx, at-time.Since(start)) {
			return drawn, nil
		}
		if err := p.draw(ctx, frame.Image); err != nil {
			return drawn, err
		}
		drawn++
	}
	if hold {
		wait(ctx, sched-time.Since(start))
	}
	return drawn, nil
}

// wait sleeps for d, reporting whether it ran its course rather than the
// playback being interrupted. A duration already past is not waited for: the
// frame it belongs to is late, and making it later helps nothing.
func wait(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// draw writes one frame, clearing what was on the screen and putting the name
// back above it.
//
// The clear, the name, and the image are built up in memory and written out
// together, so that the terminal has them all to work through at once rather
// than showing a blank screen while it waits for the image to arrive.
func (p *player) draw(ctx context.Context, img image.Image) error {
	p.buf.Reset()
	p.buf.WriteString(cursorHome)
	if p.kitty {
		p.buf.WriteString(kittyDelete)
	}
	if p.name != "" {
		p.buf.WriteString(p.name + ":\n")
	}
	p.buf.WriteString(eraseDown)
	if err := p.enc.Encode(ctx, &p.buf, img); err != nil {
		return err
	}
	_, err := p.w.Write(p.buf.Bytes())
	return err
}
