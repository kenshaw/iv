package ivctx

import (
	"context"
	"fmt"
	"image"
	"math"
	"slices"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Mode is an image scaling mode: how the configured display size and minimum
// size are applied to an image's own size.
//
// Every measure involved is in pixels. The display size is a ceiling and the
// minimum size a floor, and between them an image is left at its own size --
// so an ordinary image is never resampled, and only one that is too big to
// show or too small to see is touched at all.
type Mode string

func NewMode() (*Mode, error) {
	mode := ModeNone
	return &mode, nil
}

// Scaling modes.
const (
	// ModeBestFit shrinks an image to fit within the display size and grows
	// one smaller than the minimum size up to it, keeping its aspect ratio
	// either way. The default.
	ModeBestFit Mode = "best-fit"
	// ModeNone displays every image at its own size.
	ModeNone Mode = "none"
	// ModeWidth is [ModeBestFit] against the width alone, leaving the height
	// to follow from the aspect ratio however tall that runs.
	ModeWidth Mode = "width"
	// ModeHeight is [ModeBestFit] against the height alone.
	ModeHeight Mode = "height"
	// ModeShrink is [ModeBestFit] without the floor: an image smaller than
	// the display size is left at its own size rather than grown.
	ModeShrink Mode = "shrink"
	// ModeStretch fills the display size exactly, disregarding the aspect
	// ratio. Falls back to [ModeBestFit] when only one of the two is set,
	// there being nothing to stretch to.
	ModeStretch Mode = "stretch"
)

// modes are the scaling modes, in the order they are reported to the user.
var modes = [...]Mode{ModeBestFit, ModeNone, ModeWidth, ModeHeight, ModeShrink, ModeStretch}

// Modes returns the scaling modes.
func Modes() []Mode {
	return slices.Clone(modes[:])
}

// String satisfies the [fmt.Stringer] interface.
func (mode Mode) String() string {
	return string(mode)
}

// Valid reports whether the mode is set and is a known one.
func (mode *Mode) Valid() bool {
	return mode != nil && slices.Contains(modes[:], *mode)
}

// Get returns the mode, which is [ModeBestFit] when one was never set.
func (mode *Mode) Get() Mode {
	if !mode.Valid() {
		return ModeBestFit
	}
	return *mode
}

// MarshalText satisfies the [encoding.TextMarshaler] interface.
func (mode *Mode) MarshalText() ([]byte, error) {
	return []byte(*mode), nil
}

// UnmarshalText satisfies the [encoding.TextUnmarshaler] interface.
func (mode *Mode) UnmarshalText(buf []byte) error {
	v := Mode(strings.ToLower(strings.TrimSpace(string(buf))))
	if !v.Valid() {
		names := make([]string, len(modes))
		for i, m := range modes {
			names[i] = string(m)
		}
		return fmt.Errorf("invalid mode %q (expected one of: %s)", buf, strings.Join(names, ", "))
	}
	*mode = v
	return nil
}

// small is the longest edge below which an image is magnified by whole pixels
// rather than resampled -- see [snap].
const small = 64

// Scale returns the size an image of w x h pixels is displayed at.
func Scale(ctx context.Context, w, h int) (int, int) {
	return Get(ctx).Scale(w, h)
}

// Scale returns the size an image of w x h pixels is displayed at.
func (c *Config) Scale(w, h int) (int, int) {
	if w <= 0 || h <= 0 {
		return w, h
	}
	maxW, maxH := int(c.Width), int(c.Height)
	minW, minH := int(c.MinWidth), int(c.MinHeight)
	switch c.Mode.Get() {
	case ModeNone:
		return w, h
	case ModeStretch:
		// nothing to stretch to unless both are set, so an unset one leaves
		// this behaving as a best fit
		if maxW > 0 && maxH > 0 {
			return maxW, maxH
		}
	case ModeWidth:
		maxH, minH = 0, 0
	case ModeHeight:
		maxW, minW = 0, 0
	case ModeShrink:
		minW, minH = 0, 0
	}
	// an image already inside the display size keeps it, so the shrink never
	// goes above 1
	s := math.Min(1, ceiling(w, h, maxW, maxH))
	s = floor(w, h, s, minW, minH)
	// the display size is the one hard bound: where the two disagree -- a
	// minimum larger than the space there is to show it in -- it wins
	if c := ceiling(w, h, maxW, maxH); s > c {
		s = c
	}
	if n := snap(w, h, s, maxW, maxH); n != 0 {
		return w * n, h * n
	}
	return round(w, s), round(h, s)
}

// ceiling returns the scale at which w x h exactly fills the display size,
// which is +Inf when neither dimension of it is set.
func ceiling(w, h, maxW, maxH int) float64 {
	s := math.Inf(1)
	if maxW > 0 {
		s = math.Min(s, float64(maxW)/float64(w))
	}
	if maxH > 0 {
		s = math.Min(s, float64(maxH)/float64(h))
	}
	return s
}

// floor returns s raised, if it has to be, to the scale at which w x h first
// reaches one of its minimums.
//
// An image that already clears one of them is big enough to see, and stops
// there: a 1000x2 banner grown until its height cleared a 64 pixel floor would
// be 32000 wide, which is not what a floor is for. Only one that clears
// neither is grown, and then only until the first of the two is met.
func floor(w, h int, s float64, minW, minH int) float64 {
	switch {
	case minW <= 0 && minH <= 0,
		minW > 0 && float64(w)*s >= float64(minW),
		minH > 0 && float64(h)*s >= float64(minH):
		return s
	}
	g := math.Inf(1)
	if minW > 0 {
		g = math.Min(g, float64(minW)/(float64(w)*s))
	}
	if minH > 0 {
		g = math.Min(g, float64(minH)/(float64(h)*s))
	}
	return s * g
}

// snap returns the whole number factor to magnify a small image by, or zero
// when the scale is not a magnification of one.
//
// A 16x16 icon resampled up to 61x61 is mush, where the same icon at a whole
// factor is its own pixels drawn larger. Rounding the factor up rather than
// down keeps the result at or above the minimum that asked for it, and a
// factor that would then overrun the display size is refused outright rather
// than returning something that does not fit.
func snap(w, h int, s float64, maxW, maxH int) int {
	if s <= 1 || max(w, h) > small {
		return 0
	}
	n := int(math.Ceil(s - 1e-9))
	switch {
	case n < 2,
		maxW > 0 && w*n > maxW,
		maxH > 0 && h*n > maxH:
		return 0
	}
	return n
}

// round returns v scaled by s, never smaller than a pixel.
func round(v int, s float64) int {
	return max(int(math.Round(float64(v)*s)), 1)
}

// Resize returns src scaled to w x h, and src itself when it is already that
// size.
func Resize(ctx context.Context, src image.Image, w, h int) image.Image {
	b := src.Bounds()
	if w <= 0 || h <= 0 || (b.Dx() == w && b.Dy() == h) {
		return src
	}
	start := time.Now()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	kernel(b.Dx(), b.Dy(), w, h).Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	Logf(ctx, "resize: %dx%d -> %dx%d: %v", b.Dx(), b.Dy(), w, h, time.Since(start))
	return dst
}

// kernel returns the interpolator for scaling sw x sh to w x h.
//
// A whole number magnification of a small image is pixel replication, which is
// what keeps icon art and pixel art crisp -- a smooth kernel blurs it into
// nothing recognizable. Everything else, magnified or reduced, is resampled.
func kernel(sw, sh, w, h int) xdraw.Interpolator {
	if max(sw, sh) <= small && w >= sw && h >= sh &&
		w%sw == 0 && h%sh == 0 && w/sw == h/sh && w/sw >= 2 {
		return xdraw.NearestNeighbor
	}
	return xdraw.CatmullRom
}

// Fit scales src to the size the config displays it at.
func Fit(ctx context.Context, src image.Image) image.Image {
	b := src.Bounds()
	w, h := Scale(ctx, b.Dx(), b.Dy())
	return Resize(ctx, src, w, h)
}
