// Package png supplies a png decoder for iv. An animated png is played with
// --animate, and --page selects one of its frames (D61).
//
// See: https://github.com/gen2brain/pngn
package png

import (
	"bytes"
	"context"
	// registers the png format with image.Decode, which RegisterBuiltin
	// decodes through, and which a comic archive uses for its pages
	_ "image/png"

	"github.com/gen2brain/pngn"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivimg"
)

func init() {
	decoder.RegisterBuiltin(
		"png",
		decoder.Desc("Portable Network Graphics"),
		decoder.Extension("png", "apng"),
		decoder.MimeType("image/png", "image/apng"),
		decoder.BytesDecoderContext(decode),
	)
}

// decode decodes the png in buf. The bytes of a still png stay with the
// image, so that an encoder can send them as they are. An animated png gives
// its frames, when they are asked for.
func decode(ctx context.Context, buf []byte) (any, error) {
	if !ivimg.PNGAnimated(buf) || !decoder.Animated(ctx) {
		return ivimg.DecodePNG(buf)
	}
	p, err := pngn.DecodeAll(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	return decoder.Frames(ctx, p.Image, decoder.Seconds(p.Delay))
}
