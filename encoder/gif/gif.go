// Package gif supplies a gif encoder for iv, in Go. It reduces an image to
// the 256 colors of the Plan 9 palette, with Floyd-Steinberg dithering.
package gif

import (
	"image"
	"image/gif"
	"io"

	"github.com/kenshaw/iv/encoder"
)

func init() {
	encoder.Register(
		"gif",
		encoder.Desc("Graphics Interchange Format"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return gif.Encode(w, img, nil)
		}),
	)
}
