// Package jxl supplies a jpeg xl decoder for iv, in pure Go. An animated jpeg
// xl is played with --animate, and --page selects one of its frames (D61).
// The colors are converted to sRGB with the profile or the code points of the
// file (D57).
//
// See: https://github.com/gen2brain/jxl
package jxl

import (
	"context"
	"io"

	"github.com/gen2brain/jxl"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cms"
)

func init() {
	decoder.RegisterBuiltin(
		"jxl",
		decoder.Desc("JPEG XL"),
		decoder.Extension("jxl"),
		decoder.MimeType("image/jxl"),
		decoder.Decoder(decode),
	)
}

// decode decodes the first frame, or all of the frames when they are asked
// for.
func decode(ctx context.Context, r io.Reader) (any, error) {
	if decoder.Animated(ctx) {
		all, err := jxl.DecodeAll(r)
		if err != nil {
			return nil, err
		}
		c := all.Color
		cicp, _ := cms.CICP(c.Primaries, c.Transfer, c.Matrix, true)
		frames, err := cms.ConvertAll(ctx, all.Image, c.ICCP, cicp)
		if err != nil {
			return nil, err
		}
		return decoder.Frames(ctx, frames, decoder.Seconds(all.Delay))
	}
	img, c, err := jxl.DecodeColor(r)
	if err != nil {
		return nil, err
	}
	// a codestream gives no range, and its samples use the full range
	cicp, _ := cms.CICP(c.Primaries, c.Transfer, c.Matrix, true)
	return cms.Convert(ctx, img, c.ICCP, cicp)
}
