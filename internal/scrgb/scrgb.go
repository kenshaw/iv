// Package scrgb converts the linear light of an HDR image to sRGB, as libvips
// does: each value is clipped to the range from 0 to 1, and then encoded with
// the transfer function of sRGB. A value above 1, which is brighter than the
// white of the display, shows as white.
package scrgb

import (
	"encoding/binary"
	"image"
	"math"
)

// Func returns the linear red, green, blue and alpha of the pixel at x, y,
// from 0 to 1. The color is not premultiplied by the alpha.
type Func func(x, y int) (r, g, b, a float64)

// NRGBA64 returns the image of w x h pixels that f gives, in sRGB at 16 bits
// for each channel.
func NRGBA64(w, h int, f Func) *image.NRGBA64 {
	img := image.NewNRGBA64(image.Rect(0, 0, w, h))
	for y := range h {
		i := y * img.Stride
		for x := range w {
			r, g, b, a := f(x, y)
			// image.NRGBA64 holds each channel high byte first
			binary.BigEndian.PutUint16(img.Pix[i:], encode(r))
			binary.BigEndian.PutUint16(img.Pix[i+2:], encode(g))
			binary.BigEndian.PutUint16(img.Pix[i+4:], encode(b))
			binary.BigEndian.PutUint16(img.Pix[i+6:], Quantize(a))
			i += 8
		}
	}
	return img
}

// Unpremultiply returns the color of a pixel whose color is premultiplied by
// its alpha, as OpenEXR stores it.
func Unpremultiply(r, g, b, a float64) (float64, float64, float64, float64) {
	if a <= 0 || a >= 1 {
		return r, g, b, a
	}
	return r / a, g / a, b / a, a
}

// encode returns the linear value v encoded with the transfer function of
// sRGB, at 16 bits.
func encode(v float64) uint16 {
	switch {
	case !(v > 0):
		// NaN is black, as a negative value is
		return 0
	case v <= 0.0031308:
		v *= 12.92
	default:
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return Quantize(v)
}

// Quantize returns v, from 0 to 1, at 16 bits. A value outside the range is
// clipped to it.
func Quantize(v float64) uint16 {
	switch {
	case !(v > 0):
		return 0
	case v >= 1:
		return math.MaxUint16
	}
	return uint16(v*math.MaxUint16 + 0.5)
}
