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
		decoder.ImageDecoder(resvg.Decode),
	)
}
