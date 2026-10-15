// Package png supplies a png decoder for iv.
//
// See: https://pkg.go.dev/image/png
package png

import (
	// registers the png format with image.Decode, which RegisterBuiltin
	// decodes through, and which a comic archive uses for its pages
	_ "image/png"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivimg"
)

func init() {
	decoder.RegisterBuiltin(
		"png",
		decoder.Desc("Portable Network Graphics"),
		decoder.Extension("png"),
		decoder.MimeType("image/png", "image/apng"),
		// the bytes of the file stay with the image, so that an encoder can
		// send them as they are
		decoder.BytesDecoder(ivimg.DecodePNG),
	)
}
