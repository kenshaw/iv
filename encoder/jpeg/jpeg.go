// Package jpeg supplies a jpeg encoder for iv.
package jpeg

import (
	"context"
	"image"
	"image/jpeg"
	"io"

	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
)

// DefaultQuality is the default jpeg encode quality.
var DefaultQuality = jpeg.DefaultQuality

func init() {
	encoder.Register(
		"jpeg",
		encoder.Desc("Joint Photographic Experts Group"),
		encoder.Extension("jpg"),
		encoder.Encoder(encode),
	)
}

// encode writes the image as a JPEG with [DefaultQuality]. An image that holds
// its own encoding as a JPEG, such as a JPEG file that was not scaled, is
// written as it is. An image that can encode itself, such as an image that
// libvips holds, encodes itself.
func encode(ctx context.Context, w io.Writer, img image.Image) error {
	if buf, ok := ivimg.JPEGBytes(img); ok {
		ivctx.Logf(ctx, "jpeg encode: %d bytes as they are", len(buf))
		_, err := w.Write(buf)
		return err
	}
	if j, ok := img.(ivimg.JPEGEncoder); ok {
		buf, err := j.EncodeJPEG(DefaultQuality)
		if err != nil {
			return err
		}
		ivctx.Logf(ctx, "jpeg encode: %T encoded itself", img)
		_, err = w.Write(buf)
		return err
	}
	return jpeg.Encode(w, ivimg.Unwrap(img), &jpeg.Options{
		Quality: DefaultQuality,
	})
}
