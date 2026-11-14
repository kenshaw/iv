package ivcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"iter"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
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

// The values of --animate-mode (D42).
const (
	// animateAuto draws the frames into one Kitty image on a terminal that
	// plays Kitty animations, as [animateStream] does, and draws one image
	// for each frame on any other terminal (D43).
	animateAuto = "auto"
	// animateKitty sends one Kitty animation, which the terminal plays and
	// stores whole, with no check of the name of the terminal.
	animateKitty = "kitty"
	// animateStream always draws the frames into one Kitty image, whose
	// frame the player changes on its own clock.
	animateStream = "stream"
	// animateFrames always draws one image for each frame.
	animateFrames = "frames"
)

// animateMode returns the value of --animate-mode in lower case. An empty
// value is [animateAuto].
func animateMode(s string) (string, error) {
	switch mode := strings.ToLower(strings.TrimSpace(s)); mode {
	case "":
		return animateAuto, nil
	case animateAuto, animateKitty, animateStream, animateFrames:
		return mode, nil
	}
	return "", fmt.Errorf("animate mode %q: %w (available: %s, %s, %s, %s)", s, ErrUnknownAnimateMode, animateAuto, animateKitty, animateStream, animateFrames)
}

// ErrUnknownAnimateMode is the error for a value of --animate-mode that is
// not a mode.
var ErrUnknownAnimateMode = errors.New("unknown animate mode")

// ErrNotKitty is the error for --animate-mode kitty or stream on a terminal
// that does not use the Kitty protocol.
var ErrNotKitty = errors.New("terminal does not use the kitty protocol")

// animate plays the target's animation on the terminal.
//
// The screen is cleared, the name is written at the top of it, and the frames
// are drawn one after another underneath. A pass runs for as long as the
// animation does, up to [ivctx.AnimateMax], and --loop starts it over until
// the playback is interrupted.
//
// A terminal that plays a Kitty animation gets the frames as one animation,
// and plays them itself. See [Args.playKitty].
func (a *Args) animate(ctx context.Context, stdout io.Writer, enc *encoder.Entry, t Target) (err error) {
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
	name := t.Name
	if a.Quiet {
		name = ""
	}
	term := ivctx.Get(ctx).Terminal
	k, isKitty := term.Encoder().(rasterm.KittyEncoder)
	// timed is one Kitty animation that the terminal plays, and stream is
	// one Kitty image whose frame the player changes
	var timed, stream bool
	switch a.AnimateMode {
	case animateAuto:
		// the stream, on a terminal that changes the frame of a Kitty image
		// when a program asks (D43)
		stream = isKitty && kittyPlays(term)
	case animateKitty:
		timed = true
	case animateStream:
		stream = true
	}
	if (timed || stream) && !isKitty {
		return fmt.Errorf("animate mode %q: %w (protocol: %s)", a.AnimateMode, ErrNotKitty, term.Protocol)
	}
	ivctx.Logf(ctx, "animate mode: %s, kitty animation: %t, kitty stream: %t", a.AnimateMode, timed, stream)
	if _, err := io.WriteString(stdout, hideCursor+clearScreen); err != nil {
		return err
	}
	defer io.WriteString(stdout, showCursor)
	if timed {
		return a.playKitty(ctx, stdout, k, anim, name)
	}
	p := &player{
		w:   stdout,
		enc: enc,
		// the frames are drawn over the name rather than after it, so it
		// stays where it was written
		name:  name,
		clear: deleteImages(term),
	}
	if stream {
		p.stream, p.kitty = true, k
	}
	for {
		start := time.Now()
		n, err := p.play(ctx, anim, a.Loop)
		ivctx.Logf(ctx, "pass: %d of %d frames in %v, %d dropped", n, anim.Count, time.Since(start), p.dropped)
		switch {
		case err != nil:
			return err
		// an animation of a single frame is a still, and looping it would be
		// redrawing the same image until it was interrupted
		case !a.Loop, n < 2, ctx.Err() != nil:
			return nil
		}
	}
}

// kittyPlays returns true when the terminal plays a Kitty animation, and
// changes the frame of one when a program asks (D42).
//
// kitty 0.50.0 and later report that they play animations. Older kitty,
// from 0.20.0, and Ghostty 1.4.0 and later play them and do not report it,
// so the name and the version tell. Any other terminal with the Kitty
// protocol, such as WezTerm and Konsole, shows the first frame of an
// animation and nothing after it, so it gets one image for each frame.
func kittyPlays(t rasterm.Terminal) bool {
	switch {
	case t.Protocol != rasterm.Kitty:
		return false
	case t.KittyAnimation, strings.EqualFold(t.Name, "kitty"):
		return true
	case strings.EqualFold(t.Name, "ghostty"):
		return versionAtLeast(t.Version, 1, 4)
	}
	return false
}

// versionAtLeast returns true when the version v, such as "1.4.0" or
// "1.4.0-dev+0e1a2b3", is major.minor or later. A version that does not
// start with two numbers is not.
func versionAtLeast(v string, major, minor int) bool {
	s, rest, _ := strings.Cut(v, ".")
	maj, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	s, _, _ = strings.Cut(rest, ".")
	s = strings.TrimRightFunc(s, func(r rune) bool { return r < '0' || '9' < r })
	mnr, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return maj > major || maj == major && mnr >= minor
}

// deleteImages returns the sequence that removes the images of the terminal
// before a frame is drawn, which only Kitty needs. In tmux the sequence goes
// through the passthrough of tmux, because tmux drops it otherwise.
func deleteImages(t rasterm.Terminal) string {
	switch {
	case t.Protocol != rasterm.Kitty:
		return ""
	case t.Tmux:
		return "\x1bPtmux;" + strings.ReplaceAll(kittyDelete, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return kittyDelete
}

// playKitty sends the frames of the animation to the terminal as one Kitty
// animation, and the terminal plays them, for --animate-mode kitty (D42).
// Each frame is sent once, as it is made, and the terminal keeps the time,
// so no frame is dropped and --loop sends nothing more. The terminal stores
// every frame, so a large animation does not fit, and its last frames do not
// play.
//
// The pass ends as [player.play] ends one: at the start of the last frame,
// or after [ivctx.AnimateMax]. With --loop, the terminal plays the frames
// until the playback is interrupted. An interrupt stops the animation on
// the frame that shows.
func (a *Args) playKitty(ctx context.Context, w io.Writer, k rasterm.KittyEncoder, anim *decoder.Anim, name string) error {
	next, stop := iter.Pull2(anim.Frames(ctx))
	defer stop()
	frame, err, ok := next()
	switch {
	case !ok, ctx.Err() != nil:
		return nil
	case err != nil:
		return err
	}
	if name != "" {
		if _, err := io.WriteString(w, name+":\n"); err != nil {
			return err
		}
	}
	start := time.Now()
	ka, err := k.Animate(w, ivimg.Raw(rasterm.Kitty, frame.Image), frame.Delay)
	if err != nil {
		return err
	}
	// last is when the last frame that was sent starts
	last, sched := time.Duration(0), frame.Delay
	for sched < ivctx.AnimateMax {
		frame, err, ok := next()
		if !ok || ctx.Err() != nil {
			break
		}
		if err != nil {
			return err
		}
		f := rasterm.KittyFrame{
			Delay: frame.Delay,
			// each frame is a whole picture, with its own transparent pixels,
			// and not a patch over the frame before
			Replace: true,
		}
		if _, err := ka.Frame(ivimg.Raw(rasterm.Kitty, frame.Image), f); err != nil {
			return err
		}
		last, sched = sched, sched+frame.Delay
	}
	// the producer is done with
	stop()
	if ctx.Err() != nil {
		return ka.Stop()
	}
	loops := 1
	if a.Loop && ka.Frames() > 1 {
		// the terminal plays the frames until the playback is interrupted
		loops = 0
	}
	if err := ka.End(loops); err != nil {
		return err
	}
	ivctx.Logf(ctx, "kitty animation: %d frames sent in %v", ka.Frames(), time.Since(start))
	if loops == 0 {
		<-ctx.Done()
	} else {
		wait(ctx, last-time.Since(start))
	}
	if ctx.Err() != nil {
		return ka.Stop()
	}
	return nil
}

// slowProducer is how long the player can wait for a frame before the
// producer, and not the drawing, is what made the frame late.
const slowProducer = time.Millisecond

// player draws an animation's frames to the terminal.
type player struct {
	w     io.Writer
	enc   *encoder.Entry
	name  string
	clear string
	// stream draws the frames into one Kitty image with kitty, and ks is
	// that image after the first frame
	stream  bool
	kitty   rasterm.KittyEncoder
	ks      *rasterm.KittyStream
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
//
// A frame that is late because the player waited for the producer to make it
// is drawn, and the rest of the pass moves back by the time it lost. Dropping
// it saves nothing, because the producer makes each frame in turn, and the
// frame after it is as late. The animation then plays slower than its own
// speed, and shows every frame. A video is the exception: it keeps its own
// length, and each late frame is dropped (D43).
func (p *player) play(ctx context.Context, anim *decoder.Anim, hold bool) (int, error) {
	start, sched := time.Now(), time.Duration(0)
	ready := start
	var drawn int
	p.dropped = 0
	for frame, err := range anim.Frames(ctx) {
		// how long the player waited for the producer to make this frame
		waited := time.Since(ready)
		switch {
		case err != nil:
			return drawn, err
		case ctx.Err() != nil:
			return drawn, nil
		// an animation longer than a pass plays for is cut off here. A video
		// plays for its own length (D43)
		case !anim.Video && sched >= ivctx.AnimateMax:
			return drawn, nil
		}
		at := sched
		sched += frame.Delay
		switch late := time.Since(start) - sched; {
		case drawn == 0:
			// the first frame is never late, and the clock starts with it,
			// so that the time the decoder takes to start makes no frame late
			start = time.Now().Add(-at)
		case late < 0: // its moment has not gone by
		case waited >= slowProducer && !anim.Video:
			// the producer made it late, so it is drawn now
			start = start.Add(late + frame.Delay)
		default: // its moment has been and gone
			p.dropped++
			ready = time.Now()
			continue
		}
		if !wait(ctx, at-time.Since(start)) {
			return drawn, nil
		}
		if err := p.draw(ctx, frame.Image); err != nil {
			return drawn, err
		}
		drawn++
		ready = time.Now()
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
	if p.stream {
		return p.drawStream(img)
	}
	p.buf.Reset()
	p.buf.WriteString(cursorHome)
	// the images of the frames before, for Kitty
	p.buf.WriteString(p.clear)
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

// drawStream draws one frame into the Kitty image of the player (D42). The
// first frame places the image under the name. Each later frame changes the
// picture of the image, so nothing is cleared, and the terminal stores two
// frames however long the animation is.
func (p *player) drawStream(img image.Image) error {
	img = ivimg.Raw(rasterm.Kitty, img)
	if p.ks != nil {
		return p.ks.Draw(img)
	}
	if p.name != "" {
		if _, err := io.WriteString(p.w, p.name+":\n"); err != nil {
			return err
		}
	}
	ks, err := p.kitty.Stream(p.w, img)
	if err != nil {
		return err
	}
	p.ks = ks
	return nil
}
