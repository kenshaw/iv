// Package vips supplies a libvips decoder for iv, covering the image formats
// that the Go decoders do not handle.
//
// See: https://github.com/cshum/vipsgen
package vips

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cshum/vipsgen/v2/vips"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	ivvips "github.com/kenshaw/iv/vips"
)

func init() {
	decoder.Register(
		"vips",
		decoder.Desc("libvips (raster images)"),
		// after the Go decoders, which handle the common formats without the
		// cost of starting vips
		decoder.After("png", "jpeg", "gif", "nativewebp", "tiff", "bmp", "ico", "icns", "netpbm", "okage"),
		decoder.Extension(
			"avif", "heic", "heif", "jp2", "jpf", "jxl", "jxs", "j2k",
			"exr", "fits", "hdr", "mat", "nii", "pfm", "rad", "v",
		),
		decoder.MatcherContext(match),
		decoder.Decoder(decode),
	)
}

// match reports whether libvips should be tried for the mime type. It claims
// the image types that no more specific decoder registered, minus the ones
// libvips builds do not reliably support.
func match(_ context.Context, mime, _ string) bool {
	switch {
	// the psd decoder hands a file that it cannot read to vips by name
	case mime == "image/vnd.adobe.photoshop", mime == "image/x-photoshop",
		strings.HasPrefix(mime, "image/x-portable-"), // netpbm
		strings.Contains(mime, "jxr"):
		return false
	}
	return strings.HasPrefix(mime, "image/")
}

// loaders maps the mime types iv routes to libvips onto the load operation
// that handles them, for the formats a libvips build can be compiled without.
// Anything missing here is assumed to be supported.
var loaders = map[string]string{
	"image/avif":     "heifload_buffer",
	"image/heic":     "heifload_buffer",
	"image/heif":     "heifload_buffer",
	"image/jp2":      "jp2kload_buffer",
	"image/jpx":      "jp2kload_buffer",
	"image/jxl":      "jxlload_buffer",
	"image/x-exr":    "openexrload",
	"image/fits":     "fitsload",
	"image/x-matlab": "matload",
}

// supported reports whether this libvips build can load the mime type,
// wrapping [decoder.ErrUnsupportedFormat] when it cannot.
func supported(mime string) error {
	if op, ok := loaders[mime]; ok && !vips.HasOperation(op) {
		return fmt.Errorf("%s: %w", op, decoder.ErrUnsupportedFormat)
	}
	return nil
}

// decode decodes an image with libvips.
//
// The image is loaded from the bytes of the input and not from a stream of
// it. libvips decodes the pixels only when something asks for them, which can
// be after the pipeline closed the file, and the image keeps a reference to
// the bytes for as long as it lives.
func decode(ctx context.Context, r io.Reader) (any, error) {
	ivvips.Init(ctx)
	if err := supported(ivctx.Mime(ctx)); err != nil {
		return nil, fmt.Errorf("vips load: %w", err)
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// not every libvips loader accepts every load option -- the jxl loader
	// rejects `unlimited`, the jp2k loader rejects `n` -- so fall back to
	// progressively simpler options when one is refused
	for _, opts := range loadOptions(ivctx.Mime(ctx)) {
		var v *vips.Image
		if v, err = load(ctx, buf, opts); err == nil {
			return ivvips.Open(ctx, v)
		}
		if !isUnsupportedOption(err) {
			break
		}
		ivctx.Logf(ctx, "vips load: retrying with simpler options: %v", err)
	}
	return nil, fmt.Errorf("vips load: %w", err)
}

// loadOptions returns the load options to try, most capable first.
func loadOptions(mime string) []*vips.LoadOptions {
	return []*vips.LoadOptions{
		{
			N:           1,
			FailOnError: true,
			Unlimited:   mime != "image/jxl",
			Memory:      true,
		},
		{FailOnError: true, Memory: true},
		nil,
	}
}

// load loads the image, applying the configured page when the format is
// paged.
func load(ctx context.Context, buf []byte, opts *vips.LoadOptions) (*vips.Image, error) {
	if page := int(ivctx.Get(ctx).Page); page != 0 && opts != nil {
		// the page count is only known after a load, so load twice when a
		// specific page was requested
		v, err := vips.NewImageFromBuffer(buf, opts)
		if err != nil {
			return nil, err
		}
		if p := page - 1; 0 <= p && p < v.Pages() {
			opts.Page = p
		}
		v.Close()
	}
	return vips.NewImageFromBuffer(buf, opts)
}

// isUnsupportedOption reports whether the error is libvips refusing a load
// option that the format's loader does not have.
func isUnsupportedOption(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no property named")
}
