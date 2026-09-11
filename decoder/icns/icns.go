// Package icns supplies an icns (Apple Icon Image) decoder for iv.
//
// See: https://github.com/jackmordaunt/icns
package icns

import (
	"github.com/jackmordaunt/icns/v3"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"icns",
		decoder.Desc("Apple Icon Image"),
		decoder.Extension("icns"),
		decoder.MimeType("image/x-icns", "image/icns"),
		decoder.ImagesDecoder(icns.DecodeAll),
	)
}
