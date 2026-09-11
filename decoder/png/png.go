// Package png supplies a png decoder for iv.
//
// See: https://pkg.go.dev/image/png
package png

import (
	_ "image/png"

	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"png",
		decoder.Desc("Portable Network Graphics"),
		decoder.Extension("png"),
		decoder.MimeType("image/png", "image/apng"),
	)
}
