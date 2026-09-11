// Package ico supplies an ico decoder for iv.
//
// See: https://github.com/sergeymakinen/go-ico
package ico

import (
	"github.com/kenshaw/iv/decoder"
	"github.com/sergeymakinen/go-ico"
)

func init() {
	decoder.Register(
		"ico",
		decoder.Desc("Windows Icon"),
		decoder.Builtin(),
		decoder.Extension("ico", "cur"),
		decoder.MimeType("image/ico", "image/x-icon", "image/vnd.microsoft.icon"),
		decoder.ImagesDecoder(ico.DecodeAll),
	)
}
