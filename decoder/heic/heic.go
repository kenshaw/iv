// Package heic supplies a heic and heif decoder for iv, in pure Go. --page
// selects one of the top-level images of a file (D58), or a frame of an image
// sequence, which --animate plays (D61). The colors are converted to sRGB with
// the profile or the code points of the file (D57).
//
// See: https://github.com/gen2brain/h265
package heic

import (
	"bytes"
	"context"

	"github.com/gen2brain/h265/heic"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cms"
	"github.com/kenshaw/iv/internal/heif"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.RegisterBuiltin(
		"heic",
		decoder.Desc("High Efficiency Image File Format"),
		decoder.Extension("heic", "heif", "hif"),
		decoder.MimeType("image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence"),
		decoder.BytesDecoderContext(decode),
	)
}

// decode decodes the image that --page selects, or the primary image, turned,
// mirrored and cropped as the file says.
func decode(ctx context.Context, buf []byte) (any, error) {
	sel, index, count := heif.Select(ctx, buf)
	if ivctx.Get(ctx).Verbose {
		// the metadata of a dynamic wallpaper is in the XMP of the primary
		// image of the file, and names each of its images
		if xmp, err := heic.RawXMP(bytes.NewReader(buf)); err == nil {
			heif.LogDesktop(ctx, xmp, index, count)
		}
	}
	if count <= 1 && decoder.Animated(ctx) {
		// a file with one top-level image can hold an image sequence, whose
		// frames are then its pages
		if all, err := heic.DecodeAll(bytes.NewReader(buf), heic.Options{AutoRotate: true}); err == nil && len(all.Image) > 1 {
			c := all.Color
			cicp, _ := cms.CICP(c.Primaries, c.Transfer, c.Matrix, c.FullRange)
			frames, err := cms.ConvertAll(ctx, all.Image, c.ICCP, cicp)
			if err != nil {
				return nil, err
			}
			return decoder.Frames(ctx, frames, decoder.Seconds(all.Delay))
		}
	}
	img, c, err := heic.DecodeColor(bytes.NewReader(sel), heic.Options{AutoRotate: true})
	if err != nil {
		return nil, err
	}
	cicp, _ := cms.CICP(c.Primaries, c.Transfer, c.Matrix, c.FullRange)
	return cms.Convert(ctx, img, c.ICCP, cicp)
}
