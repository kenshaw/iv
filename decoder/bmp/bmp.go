// Package bmp supplies a bmp decoder for iv.
//
// See: https://github.com/sergeymakinen/go-bmp
package bmp

import (
	"context"
	"errors"
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

// decode decodes a bmp image, handing bitmap variants that the Go decoder does
// not support off to vips.
func decode(_ context.Context, r io.Reader) (any, error) {
	img, err := bmp.Decode(r)
	switch _, ok := errors.AsType[bmp.UnsupportedError](err); {
	case err != nil && ok:
		return decoder.Next("vips"), nil
	case err != nil:
		return nil, err
	}
	return img, nil
}
