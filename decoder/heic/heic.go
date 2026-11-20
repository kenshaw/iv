// Package heic supplies a heic and heif decoder for iv, in pure Go.
//
// See: https://github.com/gen2brain/h265
package heic

import (
	"image"
	"io"

	"github.com/gen2brain/h265/heic"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"heic",
		decoder.Desc("High Efficiency Image File Format"),
		decoder.Extension("heic", "heif", "hif"),
		decoder.MimeType("image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence"),
		decoder.ImageDecoder(decode),
	)
}

// decode decodes the primary image, turned, mirrored and cropped as the file
// says.
func decode(r io.Reader) (image.Image, error) {
	return heic.Decode(r, heic.Options{AutoRotate: true})
}
