// Package jxl supplies a jpeg xl decoder for iv, in pure Go.
//
// See: https://github.com/gen2brain/jxl
package jxl

import (
	"image"
	"io"

	"github.com/gen2brain/jxl"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"jxl",
		decoder.Desc("JPEG XL"),
		decoder.Extension("jxl"),
		decoder.MimeType("image/jxl"),
		decoder.ImageDecoder(decode),
	)
}

// decode decodes the first frame.
func decode(r io.Reader) (image.Image, error) {
	return jxl.Decode(r)
}
