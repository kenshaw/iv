// Package jpeg supplies a jpeg decoder for iv.
//
// See: https://pkg.go.dev/image/jpeg
package jpeg

import (
	// registers the jpeg format with image.Decode, which RegisterBuiltin
	// decodes through, and which a comic archive uses for its pages
	_ "image/jpeg"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivimg"
)

func init() {
	decoder.RegisterBuiltin(
		"jpeg",
		decoder.Desc("Joint Photographic Experts Group"),
		decoder.Extension("jpg", "jpeg", "jpe", "jif", "jfif", "jfi"),
		decoder.MimeType("image/jpeg"),
		// the bytes of the file stay with the image, so that an encoder can
		// send them as they are
		decoder.BytesDecoder(ivimg.DecodeJPEG),
	)
}
