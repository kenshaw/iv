// Package jxl supplies a jpeg xl encoder for iv, in pure Go. The encoder
// writes no alpha, and treats each image as sRGB.
//
// See: https://github.com/gen2brain/jxl
package jxl

import (
	"image"
	"io"

	"github.com/gen2brain/jxl"
	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"jxl",
		encoder.Desc("JPEG XL"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return jxl.Encode(w, img)
		}),
	)
}
