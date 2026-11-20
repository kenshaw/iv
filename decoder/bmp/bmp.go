// Package bmp supplies a bmp decoder for iv.
//
// See: https://github.com/sergeymakinen/go-bmp
package bmp

import (
	"context"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/sergeymakinen/go-bmp"
)

func init() {
	decoder.RegisterBuiltin(
		"bmp",
		decoder.Desc("Windows Bitmap"),
		decoder.Extension("bmp", "dib"),
		decoder.MimeType("image/bmp", "image/x-bmp", "image/x-ms-bmp"),
		decoder.Decoder(decode),
	)
}

// decode decodes a bmp image.
func decode(_ context.Context, r io.Reader) (any, error) {
	return bmp.Decode(r)
}
