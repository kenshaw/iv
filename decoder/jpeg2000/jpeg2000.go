// Package jpeg2000 supplies a jpeg 2000 decoder for iv, in pure Go. It reads
// a jp2 file and a raw codestream. go-jpeg2000 converts each color space of
// the format to sRGB, and the decoder converts an image whose file embeds an
// ICC profile with the profile (D57).
//
// See: https://github.com/mrjoshuak/go-jpeg2000
package jpeg2000

import (
	"bytes"
	"context"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cms"
	"github.com/mrjoshuak/go-jpeg2000"
)

func init() {
	decoder.RegisterBuiltin(
		"jpeg2000",
		decoder.Desc("JPEG 2000"),
		decoder.Extension("jp2", "jpf", "jpx", "j2k", "j2c", "jpc"),
		decoder.MimeType("image/jp2", "image/jpx", "image/x-jp2-codestream"),
		decoder.Decoder(decode),
	)
}

// decode decodes the image, and converts it with the profile of the file.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	img, err := jpeg2000.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	// a raw codestream has no boxes, and so no profile
	if meta, err := jpeg2000.DecodeMetadata(bytes.NewReader(buf)); err == nil && len(meta.ICCProfile) != 0 {
		return cms.Convert(ctx, img, meta.ICCProfile, nil)
	}
	return img, nil
}
