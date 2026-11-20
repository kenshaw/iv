// Package webp supplies the webp encoders for iv, in pure Go (D63). The webp
// encoder writes a webp with no loss, and --out selects it for the webp
// extension. --encoder webp-lossy writes a lossy webp.
//
// See: https://github.com/gen2brain/vpx
package webp

import (
	"image"
	"io"

	"github.com/gen2brain/vpx/webp"
	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"webp",
		encoder.Desc("WebP, with no loss"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			// a negative method is the default method of vpx
			return webp.Encode(w, img, webp.EncodeOptions{Lossless: true, Method: -1})
		}),
	)
	encoder.Register(
		"webp-lossy",
		encoder.Desc("WebP, lossy"),
		encoder.Extension("webp"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return webp.Encode(w, img)
		}),
	)
}
