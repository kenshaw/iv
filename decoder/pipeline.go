package decoder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"time"

	"github.com/kenshaw/iv/ivctx"
)

// maxDepth caps how many times a decoder may hand its result back to the
// pipeline, bounding decoder cycles.
const maxDepth = 8

var (
	// ErrNotSupported is returned when no registered decoder handles the
	// input.
	ErrNotSupported = errors.New("not supported")
	// ErrUnsupportedFormat is returned when a decoder handles the input's
	// format but the underlying library was not built with support for it.
	ErrUnsupportedFormat = errors.New("format not supported by this build")
	// ErrNotAnimated is returned when an animation was asked for and the
	// input decoded to a still image instead.
	ErrNotAnimated = errors.New("no animation")
)

// stateKey is the context key for the running decoder's initialized state.
type stateKey struct{}

// State returns the value produced by the running decoder's [Init] func. It is
// on the context of a decode func and of a close func alike.
func State(ctx context.Context) any {
	return ctx.Value(stateKey{})
}

// DecodeFile decodes the image from the file, returning the image and the
// detected mime type.
func DecodeFile(ctx context.Context, pathName string) (image.Image, string, error) {
	return asImage(decodeFile(ctx, pathName))
}

// DecodeAnimFile decodes the animation in the file, returning it and the
// detected mime type. Returns [ErrNotAnimated] when the file holds a still
// image, or an animated format that the decoder handling it renders as one.
//
// The config's Animate must be set for this to find an animation at all: it is
// what tells a decoder to produce the frames rather than pick one of them --
// see [ivctx.Animate].
func DecodeAnimFile(ctx context.Context, pathName string) (*Anim, string, error) {
	return asAnim(decodeFile(ctx, pathName))
}

// decodeFile opens the file and runs it through the pipeline.
func decodeFile(ctx context.Context, pathName string) (any, string, error) {
	f, err := os.OpenFile(pathName, os.O_RDONLY, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	ctx = ivctx.WithPathName(ctx, pathName)
	mime, err := Detect(ctx, f)
	if err != nil {
		return nil, "", fmt.Errorf("mime detect: %w", err)
	}
	ivctx.Logf(ctx, "mime: %s", mime)
	return decode(ctx, mime, ivctx.FileExt(pathName), "", f, 0)
}

// DecodeString decodes a bare string argument -- a data: URL, a WIFI: code, a
// http URL -- returning the image and the mime type it was decoded as.
func DecodeString(ctx context.Context, s string) (image.Image, string, error) {
	return asImage(decodeString(ctx, s))
}

// DecodeAnimString decodes the animation named by a bare string argument,
// returning it and the mime type it was decoded as. Returns [ErrNotAnimated]
// when the string decoded to a still image.
func DecodeAnimString(ctx context.Context, s string) (*Anim, string, error) {
	return asAnim(decodeString(ctx, s))
}

// decodeString runs a bare string argument through the pipeline.
func decodeString(ctx context.Context, s string) (any, string, error) {
	matched := MatchString(ctx, s)
	if len(matched) == 0 {
		return nil, "", fmt.Errorf("%q: %w", elide(s), ErrNotSupported)
	}
	var errs []error
	for _, d := range matched {
		state, err := d.init(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: init: %w", d.Name, err))
			continue
		}
		start := time.Now()
		res, err := d.decodeString(context.WithValue(ctx, stateKey{}, state), s)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
			continue
		}
		ivctx.Logf(ctx, "%s decode: %v", d.Name, time.Since(start))
		// a string carries no mime type of its own: a decoder that hands
		// back bytes (data:) supplies one when the pipeline re-enters, and
		// one that builds an image outright (qr) has none to give
		return resolve(ctx, "", res, 0)
	}
	return nil, "", fmt.Errorf("%q: %w", elide(s), errors.Join(errs...))
}

// Decode decodes an image from the reader using the decoders registered for
// the mime type and file extension, returning the image and the mime type it
// was decoded as.
func Decode(ctx context.Context, mime, ext string, r io.Reader) (image.Image, string, error) {
	return asImage(decode(ctx, mime, ext, "", r, 0))
}

// asImage takes the image out of a pipeline result.
func asImage(res any, mime string, err error) (image.Image, string, error) {
	switch img, ok := res.(image.Image); {
	case err != nil:
		return nil, mime, err
	case !ok:
		return nil, mime, fmt.Errorf("mime type %q: decoded no image", mime)
	default:
		return img, mime, nil
	}
}

// asAnim takes the animation out of a pipeline result. A result that is an
// image is a format with no animation in it, or one whose decoder was never
// asked for the animation -- see [ivctx.Animate].
func asAnim(res any, mime string, err error) (*Anim, string, error) {
	switch anim, ok := res.(*Anim); {
	case err != nil:
		return nil, mime, err
	case !ok:
		return nil, mime, fmt.Errorf("mime type %q: %w", mime, ErrNotAnimated)
	default:
		return anim, mime, nil
	}
}

// decode runs one pass of the decoding pipeline. When name is non-empty only
// that decoder is tried.
func decode(ctx context.Context, mime, ext, name string, r io.Reader, depth int) (any, string, error) {
	if depth >= maxDepth {
		return nil, mime, fmt.Errorf("mime type %q: exceeded %d decode retries", mime, maxDepth)
	}
	rs, err := readSeeker(r)
	if err != nil {
		return nil, mime, err
	}
	ctx = ivctx.WithMime(ctx, mime)
	matched, err := candidates(ctx, mime, ext, name)
	if err != nil {
		return nil, mime, err
	}
	var errs []error
	for _, d := range matched {
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, mime, fmt.Errorf("seek: %w", err)
		}
		state, err := d.init(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: init: %w", d.Name, err))
			continue
		}
		start := time.Now()
		res, err := d.decode(context.WithValue(ctx, stateKey{}, state), rs)
		if err != nil {
			ivctx.Logf(ctx, "%s decode: %v", d.Name, err)
			errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
			continue
		}
		ivctx.Logf(ctx, "%s decode: %v", d.Name, time.Since(start))
		// a hand off replays the same stream through another decoder, so it
		// is resolved here where the stream is still in hand
		if h, ok := res.(*Handoff); ok {
			if _, err := rs.Seek(0, io.SeekStart); err != nil {
				return nil, mime, fmt.Errorf("seek: %w", err)
			}
			return decode(ctx, mime, ext, h.Name, rs, depth+1)
		}
		v, resMime, err := resolve(ctx, mime, res, depth)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
			continue
		}
		return v, resMime, nil
	}
	return nil, mime, fmt.Errorf("mime type %q: %w", mime, errors.Join(errs...))
}

// candidates returns the decoders to try for the mime type and extension, or
// just the named decoder when name is non-empty.
func candidates(ctx context.Context, mime, ext, name string) ([]*Entry, error) {
	if name != "" {
		d, ok := Get(name)
		switch {
		case !ok:
			return nil, fmt.Errorf("decoder %q: not registered", name)
		case d.decode == nil:
			return nil, fmt.Errorf("decoder %q: has no stream decoder", name)
		}
		return []*Entry{d}, nil
	}
	matched := Match(ctx, mime, ext)
	matched = slicesFilter(matched, func(d *Entry) bool { return d.decode != nil })
	if len(matched) == 0 {
		return nil, fmt.Errorf("mime type %q: %w", mime, ErrNotSupported)
	}
	return matched, nil
}

// resolve turns a decoder result into a final image -- or, for a decode that
// asked for one, an animation -- recursing back into the pipeline as needed.
func resolve(ctx context.Context, mime string, res any, depth int) (any, string, error) {
	switch v := res.(type) {
	case nil:
		return nil, mime, errors.New("decoded no image")
	case *Anim:
		ivctx.Logf(ctx, "animation: %d frames, %v", v.Count, v.Duration)
		return v, mime, nil
	case image.Image:
		b := v.Bounds()
		ivctx.Logf(ctx, "dimensions: %dx%d", b.Dx(), b.Dy())
		return v, mime, nil
	case []image.Image:
		if len(v) == 0 {
			return nil, mime, errors.New("decoded no images")
		}
		ivctx.Logf(ctx, "images: %d", len(v))
		return resolve(ctx, mime, v[ivctx.Page(ctx, len(v))], depth)
	case *Image:
		return decode(ctx, normalizeMime(v.Mime), v.Ext, "", v.Reader, depth+1)
	case *Handoff:
		return nil, mime, fmt.Errorf("decoder %q: hand off requires a stream", v.Name)
	case *Files:
		return resolveFiles(ctx, v, depth)
	}
	return nil, mime, fmt.Errorf("decoder result %T: %w", res, ErrNotSupported)
}

// resolveFiles decodes the configured page of a file set.
func resolveFiles(ctx context.Context, files *Files, depth int) (any, string, error) {
	if len(files.Names) == 0 {
		return nil, "", errors.New("no files to decode")
	}
	name := files.Names[ivctx.Page(ctx, len(files.Names))]
	ivctx.Logf(ctx, "file: %s", name)
	f, err := files.FS.Open(name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	rs, err := readSeeker(f)
	if err != nil {
		return nil, "", err
	}
	ctx = ivctx.WithPathName(ctx, name)
	mime, err := Detect(ctx, rs)
	if err != nil {
		return nil, "", fmt.Errorf("mime detect: %w", err)
	}
	return decode(ctx, mime, ivctx.FileExt(name), "", rs, depth+1)
}

// readSeeker returns r as a [io.ReadSeeker], buffering it in memory when it
// cannot seek. The pipeline rewinds between decoders, so every stream it holds
// must be re-readable.
func readSeeker(r io.Reader) (io.ReadSeeker, error) {
	if rs, ok := r.(io.ReadSeeker); ok {
		if _, err := rs.Seek(0, io.SeekStart); err == nil {
			return rs, nil
		}
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf), nil
}

// slicesFilter returns the elements of s satisfying f.
func slicesFilter[S ~[]E, E any](s S, f func(E) bool) S {
	var res S
	for _, v := range s {
		if f(v) {
			res = append(res, v)
		}
	}
	return res
}

// elide shortens a string for use in error messages.
func elide(s string) string {
	if len(s) > 64 {
		return s[:61] + "..."
	}
	return s
}
