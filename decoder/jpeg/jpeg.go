// Package jpeg supplies a jpeg decoder for iv, with jpegn (D62). A CMYK or
// YCCK jpeg is converted to sRGB with the profile of the file, or with the
// default CMYK profile of ivy when it has none (D57).
//
// See: https://github.com/gen2brain/jpegn
package jpeg

import (
	"context"
	"image/color"

	// registers the jpeg format with image.Decode, which RegisterBuiltin
	// decodes through, and which a comic archive uses for its pages
	_ "image/jpeg"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cms"
	"github.com/kenshaw/iv/ivimg"
)

func init() {
	decoder.RegisterBuiltin(
		"jpeg",
		decoder.Desc("Joint Photographic Experts Group"),
		decoder.Extension("jpg", "jpeg", "jpe", "jif", "jfif", "jfi"),
		decoder.MimeType("image/jpeg"),
		// the bytes of the file stay with the image, so that an encoder can
		// send them as they are
		decoder.BytesDecoderContext(decode),
	)
}

// decode decodes the jpeg in buf, and keeps buf with the image. A CMYK jpeg
// keeps no bytes (see [ivimg.DecodeJPEG]), and is converted to sRGB.
func decode(ctx context.Context, buf []byte) (any, error) {
	img, err := ivimg.DecodeJPEG(buf)
	if err != nil || img.ColorModel() != color.CMYKModel {
		return img, err
	}
	return cms.Convert(ctx, img, cms.JPEGProfile(buf), nil)
}
