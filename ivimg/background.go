package ivimg

import (
	"context"
	"image"
	"image/draw"
	"strings"
	"time"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/ivctx"
)

// AddBackground returns img composited over the background color of the
// configuration on the context. The mime type is the type that img was
// decoded as.
//
// AddBackground returns img as it is in three cases: the background is
// transparent, the image is an svg, which draws its own background, or the
// image has no transparent pixels, which [Opaque] reads from the image
// format. If img is a [Flattener] and the background is opaque, img
// composites itself. Otherwise the pixels are composited into a new
// [image.NRGBA].
func AddBackground(ctx context.Context, mime string, img image.Image) (image.Image, error) {
	bg := ivctx.Get(ctx).Bg
	switch {
	case transparent(bg), strings.HasPrefix(mime, "image/svg"), Opaque(img):
		return img, nil
	}
	start := time.Now()
	if f, ok := img.(Flattener); ok && bg.A == 0xff {
		n := bg.NRGBA()
		dst, err := f.Flatten(ctx, n.R, n.G, n.B)
		if err != nil {
			return nil, err
		}
		ivctx.Logf(ctx, "add bg: %T: %v", img, time.Since(start))
		return dst, nil
	}
	src := Unwrap(img)
	if err := Err(img); err != nil {
		return nil, err
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg.NRGBA()}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	ivctx.Logf(ctx, "add bg: %T: %v", src, time.Since(start))
	return dst, nil
}

// transparent returns true when c is nil or fully transparent.
func transparent(c *colors.Color) bool {
	return c == nil || c.A == 0
}

// Opaque returns true when img has no transparent pixels, as far as its format
// tells without a look at each pixel. It returns false when the format can
// hold transparency, so a false result means only that img can have
// transparent pixels.
//
// The bytes of a PNG name its color type and whether it holds a tRNS chunk.
// An image that has an Opaque method, such as an image that libvips holds,
// answers for itself. The Go image types without an alpha channel are opaque.
func Opaque(img image.Image) bool {
	if opaque, ok := opaqueFile(img); ok {
		return opaque
	}
	for {
		switch v := img.(type) {
		case *image.YCbCr, *image.CMYK, *image.Gray, *image.Gray16:
			return true
		case *image.Paletted:
			// the palette is short, and a pixel can only be as transparent
			// as an entry of it
			for _, c := range v.Palette {
				if _, _, _, a := c.RGBA(); a != 0xffff {
					return false
				}
			}
			return true
		case interface{ Opaque() bool }:
			switch v.(type) {
			// these types scan every pixel to answer, which costs as much
			// as the compositing that the answer can save
			case *image.RGBA, *image.NRGBA, *image.RGBA64, *image.NRGBA64,
				*image.Alpha, *image.Alpha16, *image.NYCbCrA:
				return false
			}
			return v.Opaque()
		case interface{ Unwrap() image.Image }:
			img = v.Unwrap()
		default:
			return false
		}
	}
}
