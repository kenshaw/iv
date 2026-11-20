// Package tiff supplies a tiff encoder for iv, in Go. It compresses with
// deflate and the horizontal predictor.
package tiff

import (
	"image"
	"io"

	"github.com/kenshaw/iv/encoder"
	"golang.org/x/image/tiff"
)

func init() {
	encoder.Register(
		"tiff",
		encoder.Desc("Tag Image File Format"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
		}),
	)
}
