// Package png supplies a png encoder for iv.
package png

import (
	"image/png"

	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"png",
		encoder.Desc("Portable Network Graphics"),
		encoder.ImageEncoder(png.Encode),
	)
}
