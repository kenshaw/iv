// Package jpeg supplies a jpeg decoder for iv.
//
// See: https://pkg.go.dev/image/jpeg
package jpeg

import (
	_ "image/jpeg"

	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"jpeg",
		decoder.Desc("Joint Photographic Experts Group"),
		decoder.Extension("jpg", "jpeg", "jpe", "jif", "jfif", "jfi"),
		decoder.MimeType("image/jpeg"),
	)
}
