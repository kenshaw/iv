// Package jpeg2000 supplies a jpeg 2000 encoder for iv, in pure Go. It writes
// a jp2 file.
//
// See: https://github.com/mrjoshuak/go-jpeg2000
package jpeg2000

import (
	"image"
	"io"

	"github.com/kenshaw/iv/encoder"
	"github.com/mrjoshuak/go-jpeg2000"
)

func init() {
	encoder.Register(
		"jpeg2000",
		encoder.Desc("JPEG 2000"),
		encoder.Extension("jp2"),
		encoder.ImageEncoder(func(w io.Writer, img image.Image) error {
			return jpeg2000.Encode(w, img, jpeg2000.DefaultOptions())
		}),
	)
}
