// Package tiff supplies a tiff decoder for iv.
//
// See: https://pkg.go.dev/golang.org/x/image/tiff
package tiff

import (
	"github.com/kenshaw/iv/decoder"
	_ "golang.org/x/image/tiff"
)

func init() {
	decoder.RegisterBuiltin(
		"tiff",
		decoder.Desc("Tag Image File Format"),
		decoder.Extension("tif", "tiff"),
		decoder.MimeType("image/tiff", "image/tiff-fx"),
	)
}
