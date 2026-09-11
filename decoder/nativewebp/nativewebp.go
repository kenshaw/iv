// Package nativewebp supplies a webp decoder for iv.
//
// The same package supplies the webp encoder, see [encoder/nativewebp].
//
// See: https://github.com/HugoSmits86/nativewebp
//
// [encoder/nativewebp]: https://pkg.go.dev/github.com/kenshaw/iv/encoder/nativewebp
package nativewebp

import (
	"github.com/HugoSmits86/nativewebp"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"nativewebp",
		decoder.Desc("Web Picture Format"),
		decoder.Extension("webp"),
		decoder.MimeType("image/webp"),
		// DecodeIgnoreAlphaFlag rather than Decode: a VP8L image carries its
		// own transparency but the spec still requires the VP8X alpha flag,
		// and the underlying decoder rejects that combination unless the flag
		// is cleared first
		decoder.ImageDecoder(nativewebp.DecodeIgnoreAlphaFlag),
	)
}
