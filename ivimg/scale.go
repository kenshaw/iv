package ivimg

import (
	"context"
	"image"
	"time"

	"github.com/kenshaw/iv/ivctx"
	xdraw "golang.org/x/image/draw"
)

// Fit returns img scaled to the size that the configuration on the context
// displays it at. See [ivctx.Scale] for the size.
//
// If the size does not change, Fit returns img as it is. If img is a
// [Resizer], it resizes itself. Otherwise the pixels are scaled into a new
// [image.NRGBA].
func Fit(ctx context.Context, img image.Image) (image.Image, error) {
	b := img.Bounds()
	w, h := ivctx.Scale(ctx, b.Dx(), b.Dy())
	return Resize(ctx, img, w, h)
}

// Resize returns img scaled to w x h. If img is already that size, or the size
// is not positive, Resize returns img as it is.
func Resize(ctx context.Context, img image.Image, w, h int) (image.Image, error) {
	b := img.Bounds()
	if w <= 0 || h <= 0 || (b.Dx() == w && b.Dy() == h) {
		return img, nil
	}
	start := time.Now()
	if r, ok := img.(Resizer); ok {
		dst, err := r.Resize(ctx, w, h)
		if err != nil {
			return nil, err
		}
		ivctx.Logf(ctx, "resize: %T %dx%d -> %dx%d: %v", img, b.Dx(), b.Dy(), w, h, time.Since(start))
		return dst, nil
	}
	src := Unwrap(img)
	if err := Err(img); err != nil {
		return nil, err
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	interpolator(b.Dx(), b.Dy(), w, h).Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	ivctx.Logf(ctx, "resize: %T %dx%d -> %dx%d: %v", src, b.Dx(), b.Dy(), w, h, time.Since(start))
	return dst, nil
}

// Nearest returns true when scaling sw x sh to w x h magnifies a small image
// by a whole number. Pixel replication then keeps icon art and pixel art
// crisp, where a smooth kernel blurs it into nothing recognizable. A
// [Resizer] uses it to choose its kernel the same way.
func Nearest(sw, sh, w, h int) bool {
	return max(sw, sh) <= ivctx.Small && w >= sw && h >= sh &&
		w%sw == 0 && h%sh == 0 && w/sw == h/sh && w/sw >= 2
}

// interpolator returns the interpolator for scaling sw x sh to w x h: nearest
// neighbor when [Nearest] is true, and Catmull-Rom for everything else,
// magnified or reduced.
func interpolator(sw, sh, w, h int) xdraw.Interpolator {
	if Nearest(sw, sh, w, h) {
		return xdraw.NearestNeighbor
	}
	return xdraw.CatmullRom
}
