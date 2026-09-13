// Package netpbm supplies a netpbm decoder for iv.
//
// See: https://github.com/spakin/netpbm
package netpbm

import (
	"github.com/kenshaw/iv/decoder"
	_ "github.com/spakin/netpbm"
)

func init() {
	// content sniffing only recognizes the plain (ASCII) variants and raw
	// ppm, so let libmagic identify the rest by its description
	decoder.RegisterMimeType("image/x-portable-arbitrarymap", `^Netpbm PAM image file`)
	decoder.RegisterMimeType("image/x-portable-bitmap", `(?s)^Netpbm image data.*\bbitmap$`)
	decoder.RegisterMimeType("image/x-portable-graymap", `(?s)^Netpbm image data.*\bgreymap$`)
	decoder.RegisterMimeType("image/x-portable-pixmap", `(?s)^Netpbm image data.*\bpixmap$`)
	decoder.RegisterBuiltin(
		"netpbm",
		decoder.Desc("Netpbm Portable Bitmap Formats"),
		decoder.Extension("pbm", "pgm", "ppm", "pnm", "pam"),
		decoder.MimeType(
			"image/x-portable-bitmap",
			"image/x-portable-graymap",
			// libmagic's spelling
			"image/x-portable-greymap",
			"image/x-portable-pixmap",
			"image/x-portable-anymap",
			"image/x-portable-arbitrarymap",
		),
	)
}
