// Package exr supplies an openexr decoder for iv, in pure Go.
//
// See: https://github.com/mrjoshuak/go-openexr
package exr

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/scrgb"
	"github.com/mrjoshuak/go-openexr/exr"
)

func init() {
	decoder.Register(
		"exr",
		decoder.Desc("OpenEXR"),
		decoder.Extension("exr"),
		decoder.MimeType("image/x-exr", "image/aces"),
		decoder.Decoder(decode),
	)
}

// decode decodes the first part of an OpenEXR file. Its linear light is
// clipped and encoded as sRGB, as libvips does.
func decode(_ context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	f, err := exr.OpenReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return nil, fmt.Errorf("exr: %w", err)
	}
	in, err := exr.NewRGBAInputFile(f)
	if err != nil {
		return nil, fmt.Errorf("exr: %w", err)
	}
	defer in.Close()
	img, err := in.ReadRGBA()
	if err != nil {
		return nil, fmt.Errorf("exr: %w", err)
	}
	b := img.Bounds()
	// OpenEXR stores the color premultiplied by the alpha
	return scrgb.NRGBA64(b.Dx(), b.Dy(), func(x, y int) (float64, float64, float64, float64) {
		r, g, b, a := img.RGBA(b.Min.X+x, b.Min.Y+y)
		return scrgb.Unpremultiply(float64(r), float64(g), float64(b), float64(a))
	}), nil
}
