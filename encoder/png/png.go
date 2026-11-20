// Package png supplies a png encoder for iv.
package png

import (
	"context"
	"image"
	"image/png"
	"io"

	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
)

func init() {
	encoder.Register(
		"png",
		encoder.Desc("Portable Network Graphics"),
		encoder.Encoder(encode),
	)
}

// encode writes the image as a PNG. An image that holds the bytes of a PNG
// file, such as a PNG file that was not scaled, is written as it is. Any other
// image is encoded by Go at its default level, from the pixels under each
// wrapper. The Go encoder keeps the depth of a 16-bit image.
func encode(ctx context.Context, w io.Writer, img image.Image) error {
	if buf, ok := ivimg.PNGBytes(img); ok {
		ivctx.Logf(ctx, "png encode: %d bytes as they are", len(buf))
		_, err := w.Write(buf)
		return err
	}
	return png.Encode(w, ivimg.Unwrap(img))
}
