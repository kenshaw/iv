// Package heic supplies a heic encoder for iv, in pure Go.
//
// See: https://github.com/gen2brain/h265
package heic

import (
	"image"
	"io"

	"github.com/gen2brain/h265/heic"
	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"heic",
		encoder.Desc("High Efficiency Image File Format"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return heic.Encode(w, img)
		}),
	)
}
