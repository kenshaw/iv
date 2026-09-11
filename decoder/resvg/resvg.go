// Package resvg supplies a svg decoder for iv.
//
// See: https://github.com/xo/resvg
package resvg

import (
	"github.com/kenshaw/iv/decoder"
	"github.com/xo/resvg"
)

func init() {
	decoder.RegisterBuiltin(
		"resvg",
		decoder.Desc("Scalable Vector Graphics"),
		decoder.Extension("svg", "svgz"),
		decoder.MimeType("image/svg+xml", "image/svg"),
		// a .svgz is gzipped, so it sniffs as application/gzip and nothing
		// but the extension identifies it -- resvg decompresses it itself
		decoder.MimeTypeExtensionMatch("application/gzip", "svgz"),
		decoder.ImageDecoder(resvg.Decode),
	)
}
