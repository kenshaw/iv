// Package matlab supplies a MATLAB decoder for iv, in pure Go. It reads a
// MAT-file of version 5. The reader also opens a file of version 7.3, but it
// returns each array with one dimension, so such a file has no image.
//
// See: https://github.com/scigolib/matlab
package matlab

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"math"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/scrgb"
	"github.com/scigolib/matlab"
	"github.com/scigolib/matlab/types"
)

func init() {
	decoder.Register(
		"matlab",
		decoder.Desc("MATLAB arrays"),
		decoder.Extension("mat"),
		decoder.MimeType("application/x-matlab-data"),
		decoder.Decoder(decode),
	)
}

// ErrNoImage is the error of a MAT-file that holds no array that can be shown
// as an image.
var ErrNoImage = errors.New("no array of 2 dimensions, or of 3 dimensions with 1, 3 or 4 planes")

// decode draws the first array of the file that is an image, as libvips
// does: an array of 2 dimensions is gray, and an array of 3 dimensions with
// 3 or 4 planes is color, with alpha in the fourth plane. An array of uint8
// or uint16 keeps its values, and any other array is stretched from its least
// value to its greatest.
func decode(_ context.Context, r io.Reader) (any, error) {
	f, err := matlab.Open(r)
	if err != nil {
		return nil, fmt.Errorf("matlab: %w", err)
	}
	for _, v := range f.Variables {
		if img, ok := toImage(v); ok {
			return img, nil
		}
	}
	return nil, fmt.Errorf("matlab: %w", ErrNoImage)
}

// toImage returns the image of v, when v is an image.
func toImage(v *types.Variable) (image.Image, bool) {
	if v.IsComplex || v.IsSparse || len(v.Dimensions) < 2 || len(v.Dimensions) > 3 {
		return nil, false
	}
	h, w, planes := v.Dimensions[0], v.Dimensions[1], 1
	if len(v.Dimensions) == 3 {
		planes = v.Dimensions[2]
	}
	if w <= 0 || h <= 0 || (planes != 1 && planes != 3 && planes != 4) {
		return nil, false
	}
	vals, scale, ok := values(v.DataType, v.Data)
	if !ok || len(vals) != w*h*planes {
		return nil, false
	}
	// MATLAB stores an array by column: the rows of a column, then the next
	// column, then the next plane
	at := func(x, y, p int) float64 {
		return scale(vals[y+x*h+p*w*h])
	}
	img := image.NewNRGBA64(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			i := img.PixOffset(x, y)
			for c := range 4 {
				var s float64
				switch {
				case c == 3 && planes != 4:
					s = 1
				case planes == 1:
					s = at(x, y, 0)
				default:
					s = at(x, y, c)
				}
				binary.BigEndian.PutUint16(img.Pix[i+2*c:], scrgb.Quantize(s))
			}
		}
	}
	return img, true
}

// values returns the values of a numeric array as float64, with the func that
// maps a value to the range from 0 to 1. The reader gives the values of most
// arrays as float64, so the type of the array decides the range.
func values(typ types.DataType, data any) ([]float64, func(float64) float64, bool) {
	var vals []float64
	switch d := data.(type) {
	case []float64:
		vals = d
	case []float32:
		vals = convert(d)
	case []uint8:
		vals = convert(d)
	case []int8:
		vals = convert(d)
	case []uint16:
		vals = convert(d)
	case []int16:
		vals = convert(d)
	case []uint32:
		vals = convert(d)
	case []int32:
		vals = convert(d)
	case []uint64:
		vals = convert(d)
	case []int64:
		vals = convert(d)
	default:
		return nil, nil, false
	}
	switch typ {
	case types.Uint8:
		return vals, func(v float64) float64 { return v / math.MaxUint8 }, true
	case types.Uint16:
		return vals, func(v float64) float64 { return v / math.MaxUint16 }, true
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range vals {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			lo, hi = min(lo, v), max(hi, v)
		}
	}
	if !(hi > lo) {
		return vals, func(float64) float64 { return 0 }, true
	}
	return vals, func(v float64) float64 { return (v - lo) / (hi - lo) }, true
}

// convert returns the values of s as float64.
func convert[T int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32](s []T) []float64 {
	v := make([]float64, len(s))
	for i, x := range s {
		v[i] = float64(x)
	}
	return v
}
