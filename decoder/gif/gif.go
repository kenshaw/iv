// Package gif supplies a gif decoder for iv.
//
// See: https://pkg.go.dev/image/gif
package gif

import (
	_ "image/gif"

	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"gif",
		decoder.Desc("Graphics Interchange Format"),
		decoder.Extension("gif"),
		decoder.MimeType("image/gif"),
	)
}
