// Package psd supplies a Photoshop decoder for iv.
//
// See: https://github.com/oov/psd
package psd

import (
	"context"
	"image"
	"image/draw"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/oov/psd"
)

func init() {
	decoder.RegisterBuiltin(
		"psd",
		decoder.Desc("Adobe Photoshop"),
		decoder.Extension("psd", "psb"),
		decoder.MimeType("image/vnd.adobe.photoshop", "image/x-photoshop"),
		decoder.Decoder(decode),
	)
}

// decode decodes the merged image that a Photoshop file holds next to its
// layers.
//
// oov/psd decodes it in Go, and it matched the source image pixel for pixel
// for an 8-bit RGB, an RGB with alpha and a CMYK file. It fails on some 16-bit
// files, and a file that it cannot read is handed to the vips decoder, which
// reads it through ImageMagick where libvips is built with it.
func decode(ctx context.Context, r io.Reader) (any, error) {
	img, _, err := psd.Decode(r, nil)
	if err != nil {
		ivctx.Logf(ctx, "psd decode: %v, trying vips", err)
		return decoder.Next("vips"), nil
	}
	// the picker converts each pixel when it is asked for, which the scaler
	// and the encoders would do many times over, so the pixels are converted
	// once here
	b := img.Picker.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img.Picker, b.Min, draw.Src)
	return dst, nil
}
