// Package netpbm supplies a netpbm decoder for iv.
//
// See: https://github.com/spakin/netpbm
package netpbm

import (
	"github.com/kenshaw/iv/decoder"
	_ "github.com/spakin/netpbm"
)

func init() {
	decoder.RegisterBuiltin(
		"netpbm",
		decoder.Desc("Netpbm Portable Bitmap Formats"),
		decoder.Extension("pbm", "pgm", "ppm", "pnm", "pam"),
		decoder.MimeType(
			"image/x-portable-bitmap",
			"image/x-portable-graymap",
			"image/x-portable-pixmap",
			"image/x-portable-anymap",
			"image/x-portable-arbitrarymap",
		),
	)
}
