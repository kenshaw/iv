// Package webp supplies a webp decoder for iv.
//
// See: https://pkg.go.dev/golang.org/x/image/webp
package webp

import (
	"github.com/kenshaw/iv/decoder"
	_ "golang.org/x/image/webp"
)

func init() {
	decoder.RegisterBuiltin(
		"webp",
		decoder.Desc("Web Picture Format"),
		decoder.Extension("webp"),
		decoder.MimeType("image/webp"),
	)
}
