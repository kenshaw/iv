// Package avif supplies an avif decoder for iv, in pure Go.
//
// See: https://github.com/gen2brain/gav1d
package avif

import (
	"image"
	"io"

	"github.com/gen2brain/gav1d/avif"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"avif",
		decoder.Desc("AV1 Image File Format"),
		decoder.Extension("avif", "avifs"),
		decoder.MimeType("image/avif", "image/avif-sequence"),
		decoder.ImageDecoder(decode),
	)
}

// decode decodes the primary image, turned, mirrored and cropped as the file
// says.
func decode(r io.Reader) (image.Image, error) {
	return avif.Decode(r, avif.Options{AutoRotate: true})
}
