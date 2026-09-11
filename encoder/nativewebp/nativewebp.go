// Package nativewebp supplies a webp encoder for iv.
//
// See: https://github.com/HugoSmits86/nativewebp
package nativewebp

import (
	"image"
	"io"

	"github.com/HugoSmits86/nativewebp"
	"github.com/kenshaw/iv/encoder"
)

// DefaultCompression is the default encode compression level.
var DefaultCompression = nativewebp.DefaultCompression

func init() {
	encoder.Register(
		"nativewebp",
		encoder.Desc("Web Picture Format"),
		encoder.Extension("webp"),
		encoder.ImageEncoder(encode),
	)
}

// encode wraps [nativewebp.Encode].
func encode(w io.Writer, img image.Image) error {
	return nativewebp.Encode(w, img, &nativewebp.Options{
		CompressionLevel: DefaultCompression,
	})
}
