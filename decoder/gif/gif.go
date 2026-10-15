// Package gif supplies a gif decoder for iv.
//
// See: https://pkg.go.dev/image/gif
package gif

import (
	"context"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.RegisterBuiltin(
		"gif",
		decoder.Desc("Graphics Interchange Format"),
		decoder.Extension("gif"),
		decoder.MimeType("image/gif"),
		decoder.Decoder(decode),
	)
}

// Frame delays.
//
// A gif records its delays in hundredths of a second, and the ones written for
// a browser lean on what browsers do with the short end of that: 0 and 1 mean
// "as fast as you can", which every one of them has drawn at ten frames a
// second since Netscape. An animation timed against that plays at ten times
// the speed it was drawn for without the same floor here.
const (
	tick      = 10 * time.Millisecond
	fastDelay = 10 * time.Millisecond
	slowDelay = 100 * time.Millisecond
)

// decode decodes the gif: the whole animation when one was asked for, and its
// first frame otherwise.
func decode(ctx context.Context, r io.Reader) (any, error) {
	if !ivctx.Animate(ctx) {
		return gif.Decode(r)
	}
	g, err := gif.DecodeAll(r)
	if err != nil {
		return nil, err
	}
	delays := make([]time.Duration, len(g.Image))
	var total time.Duration
	for i := range g.Image {
		d := slowDelay
		if i < len(g.Delay) {
			if v := time.Duration(g.Delay[i]) * tick; v > fastDelay {
				d = v
			}
		}
		// --fps, where it was passed, replaces the timing the file records
		d = ivctx.AnimateDelay(ctx, d)
		delays[i], total = d, total+d
	}
	return &decoder.Anim{
		Frames: decoder.Produce(func(ctx context.Context, s *decoder.Sink) error {
			play(ctx, g, delays, s)
			return nil
		}),
		Count:    len(g.Image),
		Duration: total,
	}, nil
}

// play composites the gif's frames and sends them for display.
//
// A gif frame is a patch rather than a picture: it covers only the part of the
// image that changed, and the disposal method recorded with it says what to do
// with that part before the next one is drawn. Sent as they come, the frames
// would play as a scatter of fragments, so each is drawn onto a canvas carried
// from frame to frame and a copy of the canvas is what is displayed.
func play(ctx context.Context, g *gif.GIF, delays []time.Duration, s *decoder.Sink) {
	canvas := image.NewRGBA(bounds(g))
	var saved *image.RGBA
	for i, frame := range g.Image {
		b := frame.Bounds()
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			saved = clone(canvas)
		}
		draw.Draw(canvas, b, frame, b.Min, draw.Over)
		if !s.Send(ctx, clone(canvas), delays[i]) {
			return
		}
		switch disposal {
		case gif.DisposalBackground:
			// the area the frame covered goes back to showing nothing, which
			// for a gif displayed over anything but its own background means
			// transparent
			draw.Draw(canvas, b, image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = saved
		}
	}
}

// bounds returns the size of the gif's canvas, which is the area its frames
// are patches of.
//
// The logical screen descriptor holds it, and a gif written without one leaves
// it to be worked out from the frames themselves.
func bounds(g *gif.GIF) image.Rectangle {
	if g.Config.Width > 0 && g.Config.Height > 0 {
		return image.Rect(0, 0, g.Config.Width, g.Config.Height)
	}
	var r image.Rectangle
	for _, frame := range g.Image {
		r = r.Union(frame.Bounds())
	}
	return r
}

// clone returns a copy of the canvas, the frames being displayed while the
// canvas moves on to the next of them.
func clone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	return dst
}
