// Package avif supplies an avif encoder for iv, in pure Go. It writes 8 bits
// for each channel, with alpha.
//
// See: https://github.com/gen2brain/gav1d
package avif

import (
	"image"
	"io"

	"github.com/gen2brain/gav1d/avif"
	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"avif",
		encoder.Desc("AV1 Image File Format"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return avif.Encode(w, img)
		}),
	)
}
