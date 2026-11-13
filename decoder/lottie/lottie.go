// Package lottie supplies a lottie animation decoder for iv. It draws the
// frames with the lottie package of okage, in Go (D41).
//
// See: https://github.com/xo/okage
package lottie

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage"
	"github.com/xo/okage/lottie"
)

// mime is the lottie mime type.
const mime = "video/lottie+json"

func init() {
	decoder.Register(
		"lottie",
		decoder.Desc("Lottie animations"),
		decoder.Extension("json"),
		decoder.MimeType(mime, "application/lottie+json"),
		// the mime type of a lottie is video/lottie+json, which the ffmpeg
		// decoder also claims with video/*. Without this, the order of the
		// two depends on the order in which their packages are initialized.
		decoder.Before("ffmpeg"),
		// a .lot or .lottie holding a bare document sniffs as
		// application/json, which says nothing about what the json is, so the
		// two unambiguous extensions are paired with it here. A .json is not:
		// the name is no evidence at all, and a lottie carrying it is
		// recognized by [detect] reading the document instead.
		//
		// A dotLottie archive sniffs as application/zip, which is generic
		// enough that the extension alone routes it -- see isGeneric in the
		// decoder package.
		decoder.MimeTypeExtensionMatch(
			"application/json", "lot",
			"application/json", "lottie",
		),
		decoder.MimeDetector(detect),
		decoder.Decoder(decode),
	)
}

// peek is how many leading bytes are sniffed for a lottie document.
//
// The properties that identify one are written ahead of the artwork --
// bodymovin emits v, fr, ip, op, w and h before the assets and layers -- so
// this has to cover the head of the object rather than the megabytes of paths
// that can follow it. Across the 879 animations of the noto emoji set the
// deepest of those headers ends at byte 52, so this is generous; it is sniffed
// against every file iv opens, which is what keeps it from being larger still.
const peek = 16 * 1024

// detect sniffs the reader for a lottie animation.
//
// A short read is not an error here, only less to look at: whatever came back
// is put to [lottie.Detect], and a stream that could not be read at all is
// simply not a lottie.
func detect(_ context.Context, r io.Reader) (string, error) {
	buf, _ := bufio.NewReaderSize(r, peek).Peek(peek)
	if !lottie.Detect(buf) {
		return "", nil
	}
	return mime, nil
}

// decode renders the animation: the whole timeline when one was asked for --
// see [animate] -- and a single frame of it otherwise.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if lottie.IsDotLottie(buf) {
		return dotLottie(ctx, buf)
	}
	c, err := lottie.Parse(buf)
	if err != nil {
		return nil, err
	}
	total := c.FrameCount()
	if total == 0 {
		return nil, okage.ErrLottieEmpty
	}
	// a composition has no pixels of its own, so it is drawn at the size that
	// it is shown at, and not resampled to that size after
	w, h := ivctx.Scale(ctx, int(c.Width), int(c.Height))
	if ivctx.Animate(ctx) {
		return animate(ctx, c, w, h)
	}
	n := frameIndex(ctx, total)
	ivctx.Logf(ctx, "lottie: %dx%d, %d frames at %g fps, rendering frame %d", w, h, total, c.FrameRate, n)
	return c.Image(n, w, h)
}

// frameIndex returns the frame to render: the configured page when it names
// one, and the middle of the timeline otherwise.
//
// A lottie commonly opens on an empty stage -- the artwork flies in -- so its
// first frame makes a poor still, where the midpoint shows the animation with
// everything on screen.
func frameIndex(ctx context.Context, total int) int {
	if page := int(ivctx.Get(ctx).Page) - 1; 0 <= page && page < total {
		return page
	}
	return total / 2
}

// dotLottie lists the animations in a dotLottie archive, handing them back to
// the pipeline for the configured page to be selected. [lottie.OpenDotLottie]
// decides which files of the archive are animations.
//
// An archive holding more than one animation spends the page on choosing
// between them, leaving [frameIndex] to fall back to the middle frame. Nearly
// every dotLottie carries a single animation, where the page still selects the
// frame.
func dotLottie(ctx context.Context, buf []byte) (any, error) {
	d, err := lottie.OpenDotLottie(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "dotlottie animations: %d", len(d.Animations))
	return decoder.FS(d.FS(), d.Animations...), nil
}
