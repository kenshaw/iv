// Package lottie supplies a lottie animation decoder for iv, rasterizing
// frames with the thorvg vector engine.
//
// See: https://github.com/xo/lottie
package lottie

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"sort"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/lottie"
)

// mime is the lottie mime type.
const mime = "video/lottie+json"

func init() {
	decoder.Register(
		"lottie",
		decoder.Desc("Lottie animations"),
		decoder.Extension("json"),
		decoder.MimeType(mime, "application/lottie+json"),
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
		decoder.Init(open, shut),
		decoder.Decoder(decode),
	)
}

// open builds the renderer. Compiling the thorvg webassembly module is the
// expensive part of it, and the pipeline runs this once however many
// animations are rendered.
func open(ctx context.Context) (any, error) {
	r, err := lottie.New(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// shut closes the renderer, releasing the webassembly runtime with it.
func shut(ctx context.Context) error {
	r, _ := decoder.State(ctx).(*lottie.Renderer)
	if r == nil {
		return nil
	}
	return r.Close(ctx)
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
	if _, _, ok := header(buf); !ok {
		return "", nil
	}
	return mime, nil
}

// decode renders a single frame of the animation.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if isZip(buf) {
		return dotLottie(ctx, buf)
	}
	rend, _ := decoder.State(ctx).(*lottie.Renderer)
	if rend == nil {
		return nil, errors.New("renderer not initialized")
	}
	// only the size is taken from the document here, and a document this
	// cannot read is still handed to thorvg to judge: a file reaching this
	// point named .lot or .lottie was routed by that name rather than by the
	// sniffer, and thorvg has the final say on what it can parse
	nw, nh, _ := header(buf)
	w, h := ivctx.Scale(ctx, nw, nh)
	// thorvg draws to whatever size it is handed, so the animation is
	// rasterized at the displayed size rather than resampled to it after: a
	// composition has no pixels of its own to preserve. A zero size, from a
	// document whose own size could not be read, leaves it on thorvg's.
	//
	// the errors this package returns already name themselves, and the
	// pipeline prefixes the decoder on top, so nothing is added here
	a, err := rend.Load(ctx, buf, lottie.Options{Width: w, Height: h})
	if err != nil {
		return nil, err
	}
	defer a.Close(ctx)
	w, h = a.Size()
	total := int(a.TotalFrames())
	frame := frameIndex(ctx, total)
	ivctx.Logf(ctx, "lottie: %dx%d, %d frames at %g fps, rendering frame %d", w, h, total, a.FPS(), frame)
	img, err := a.Frame(ctx, float32(frame))
	if err != nil {
		return nil, err
	}
	return img, nil
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

// header reads the composition size out of the head of a lottie document,
// reporting whether the document is one at all.
//
// buf may be cut short mid-document, in which case the walk stops where the
// json does and answers with what it saw up to there. Only the top level is
// read: a "w" belonging to a layer is stepped over rather than mistaken for
// the composition's.
func header(buf []byte) (int, int, bool) {
	dec := json.NewDecoder(bytes.NewReader(buf))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return 0, 0, false
	}
	var w, h float64
	var seen int
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return 0, 0, false
		}
		key, ok := t.(string)
		if !ok {
			return 0, 0, false
		}
		i := slices.Index(markers[:], key)
		if i == -1 {
			if err := skip(dec); err != nil {
				return 0, 0, false
			}
			continue
		}
		// every identifying property is a number; one that is anything else
		// belongs to some other format that happens to share the name
		if t, err = dec.Token(); err != nil {
			return 0, 0, false
		}
		v, ok := t.(float64)
		if !ok {
			return 0, 0, false
		}
		switch key {
		case "w":
			w = v
		case "h":
			h = v
		}
		if seen |= 1 << i; seen == allMarkers {
			return int(w), int(h), true
		}
	}
	return 0, 0, false
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
