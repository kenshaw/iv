// Package vpx supplies a lossy webp encoder for iv, in pure Go. The
// nativewebp encoder writes a webp file that has no loss, and --out selects
// it for the webp extension. --encoder vpx selects this one.
//
// See: https://github.com/gen2brain/vpx
package vpx

import (
	"image"
	"io"

	"github.com/gen2brain/vpx/webp"
	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"vpx",
		encoder.Desc("WebP, lossy"),
		encoder.Extension("webp"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return webp.Encode(w, img)
		}),
	)
}
