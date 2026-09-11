// Package jpeg supplies a jpeg encoder for iv.
package jpeg

import (
	"image"
	"image/jpeg"
	"io"

	"github.com/kenshaw/iv/encoder"
)

// DefaultQuality is the default jpeg encode quality.
var DefaultQuality = jpeg.DefaultQuality

func init() {
	encoder.Register(
		"jpeg",
		encoder.Desc("Joint Photographic Experts Group"),
		encoder.Extension("jpg"),
		encoder.ImageEncoder(encode),
	)
}

// encode wraps [jpeg.Encode].
func encode(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, &jpeg.Options{
		Quality: DefaultQuality,
	})
}
