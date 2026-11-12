// Package lottie supplies a lottie animation decoder for iv. It draws the
// frames with the lottie package of okage, in Go (D41).
//
// See: https://github.com/xo/okage
package lottie

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"path"
	"slices"
	"sort"

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
// is put to [header], and a stream that could not be read at all is simply not
// a lottie.
func detect(_ context.Context, r io.Reader) (string, error) {
	buf, _ := bufio.NewReaderSize(r, peek).Peek(peek)
	if !header(buf) {
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
	if isZip(buf) {
		return dotLottie(ctx, buf)
	}
	c, err := lottie.Parse(buf)
	if err != nil {
		return nil, err
	}
	total := int(c.Frames[1] - c.Frames[0])
	if c.Width == 0 || c.Height == 0 || total <= 0 {
		return nil, ErrEmpty
	}
	// a composition has no pixels of its own, so it is drawn at the size that
	// it is shown at, and not resampled to that size after
	w, h := ivctx.Scale(ctx, int(c.Width), int(c.Height))
	if ivctx.Animate(ctx) {
		return animate(ctx, c, total, w, h)
	}
	n := frameIndex(ctx, total)
	ivctx.Logf(ctx, "lottie: %dx%d, %d frames at %g fps, rendering frame %d", w, h, total, c.FrameRate, n)
	return frame(c, n, w, h)
}

// ErrEmpty is the error of a composition with no size or no frames.
var ErrEmpty = errors.New("lottie has no size or no frames")

// frame draws frame n of the composition, counted from its first frame, at
// w x h.
func frame(c *lottie.Composition, n, w, h int) (*image.RGBA, error) {
	if w <= 0 || h <= 0 || w > math.MaxInt32 || h > math.MaxInt32 {
		return nil, fmt.Errorf("lottie frame %dx%d: %w", w, h, ErrEmpty)
	}
	pm, ok := okage.NewPixmap(uint32(w), uint32(h))
	if !ok {
		return nil, fmt.Errorf("lottie frame %dx%d: %w", w, h, ErrEmpty)
	}
	m := pm.AsMut()
	s := lottie.AffineScale(float64(w) / float64(c.Width))
	if err := lottie.RenderFrame(c, c.Frames[0]+float64(n), s, &m); err != nil {
		return nil, fmt.Errorf("lottie frame %d: %w", n, err)
	}
	return pm.AsRGBA(), nil
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
// the pipeline for the configured page to be selected.
//
// The container is a zip. The spec puts the documents under animations/ and
// describes them in a manifest.json, so that is what is looked for first, with
// any other json in the archive taken when there is no such directory.
//
// An archive holding more than one animation spends the page on choosing
// between them, leaving [frameIndex] to fall back to the middle frame. Nearly
// every dotLottie carries a single animation, where the page still selects the
// frame.
func dotLottie(ctx context.Context, buf []byte) (any, error) {
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return nil, fmt.Errorf("dotlottie: %w", err)
	}
	var names, others []string
	for _, f := range zr.File {
		switch {
		case f.FileInfo().IsDir(), path.Ext(f.Name) != ".json":
		case path.Dir(f.Name) == "animations":
			names = append(names, f.Name)
		case path.Base(f.Name) != "manifest.json":
			others = append(others, f.Name)
		}
	}
	if len(names) == 0 {
		names = others
	}
	if len(names) == 0 {
		return nil, errors.New("dotlottie: no animations in the archive")
	}
	sort.Strings(names)
	ivctx.Logf(ctx, "dotlottie animations: %d", len(names))
	return decoder.FS(zr, names...), nil
}

// zipMagic is the local file header signature a zip starts with.
var zipMagic = []byte("PK\x03\x04")

// isZip reports whether buf is a zip, and so a dotLottie rather than a bare
// document.
func isZip(buf []byte) bool {
	return bytes.HasPrefix(buf, zipMagic)
}

// markers are the top level properties that identify a lottie document: the
// frame rate, the in and out points, and the composition size.
//
// Requiring every one of them is what keeps an ordinary .json file from being
// taken for an animation -- no one of these names is rare, but a document
// carrying all five at its top level, each holding a number, is a lottie.
var markers = [...]string{"fr", "ip", "op", "w", "h"}

// allMarkers is the seen set with every marker in it.
const allMarkers = 1<<len(markers) - 1

// header reports whether the head of a document is the head of a lottie.
//
// buf may be cut short mid-document, in which case the walk stops where the
// json does and answers with what it saw up to there. Only the top level is
// read: a "w" belonging to a layer is stepped over rather than mistaken for
// the composition's.
func header(buf []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(buf))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return false
	}
	var seen int
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return false
		}
		key, ok := t.(string)
		if !ok {
			return false
		}
		i := slices.Index(markers[:], key)
		if i == -1 {
			if err := skip(dec); err != nil {
				return false
			}
			continue
		}
		// every identifying property is a number; one that is anything else
		// belongs to some other format that happens to share the name
		if t, err = dec.Token(); err != nil {
			return false
		}
		if _, ok := t.(float64); !ok {
			return false
		}
		if seen |= 1 << i; seen == allMarkers {
			return true
		}
	}
	return false
}

// skip reads past the next value, descending through nested objects and
// arrays.
func skip(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if _, ok := t.(json.Delim); !ok {
		return nil
	}
	for depth := 1; depth > 0; {
		switch t, err := dec.Token(); {
		case err != nil:
			return err
		case t == json.Delim('{'), t == json.Delim('['):
			depth++
		case t == json.Delim('}'), t == json.Delim(']'):
			depth--
		}
	}
	return nil
}
