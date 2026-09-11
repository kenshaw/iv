package tag

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"math"

	// the cover art is whatever the tagger embedded, in practice always one
	// of these
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// accent is the color the card draws its waveform and artist line in.
type accent struct {
	h, s, v float64
}

// defaultAccent is used when there is no cover art to sample and nothing to
// derive a hue from.
var defaultAccent = accent{h: 205, s: 0.72, v: 0.95}

// accentFor picks the card's accent color from the cover art, falling back to
// a hue derived from the text when there is no art to sample.
func accentFor(art []byte, text string) accent {
	if a, ok := accentFromArt(art); ok {
		return a
	}
	if text == "" {
		return defaultAccent
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(text))
	return accent{h: float64(h.Sum32() % 360), s: 0.68, v: 0.95}
}

// accentFromArt samples the cover art for the hue that carries the most
// colorful weight. Pixels are weighted by saturation so that a small vivid
// detail beats a large grey field, which is what the eye picks out of a cover
// as "its" color.
func accentFromArt(art []byte) (accent, bool) {
	if len(art) == 0 {
		return accent{}, false
	}
	img, _, err := image.Decode(bytes.NewReader(art))
	if err != nil {
		return accent{}, false
	}
	// 24 hue buckets, weighted by saturation and value
	var weight [24]float64
	var sat, val, total float64
	b := img.Bounds()
	step := max(max(b.Dx(), b.Dy())/64, 1)
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			h, s, v := rgbToHSV(img.At(x, y))
			// near-black and near-white carry no usable hue
			if v < 0.15 || s < 0.15 {
				continue
			}
			w := s * v
			weight[int(h/15)%24] += w
			sat, val, total = sat+s*w, val+v*w, total+w
		}
	}
	if total == 0 {
		return accent{}, false
	}
	best := 0
	for i, w := range weight {
		if w > weight[best] {
			best = i
		}
	}
	return accent{
		h: float64(best)*15 + 7.5,
		// pull toward vivid and bright so the accent reads against the dark
		// card whatever the cover looks like
		s: min(max(sat/total, 0.55), 0.85),
		v: min(max(val/total*1.25, 0.80), 1.0),
	}, true
}

// hex returns the accent as a css color, with its saturation and value
// scaled.
func (a accent) hex(sf, vf float64) string {
	return hsvHex(a.h, min(a.s*sf, 1), min(a.v*vf, 1))
}

// rgbToHSV converts a color to hue (degrees), saturation and value.
func rgbToHSV(c color.Color) (float64, float64, float64) {
	ri, gi, bi, _ := c.RGBA()
	r, g, b := float64(ri)/65535, float64(gi)/65535, float64(bi)/65535
	mx, mn := max(r, g, b), min(r, g, b)
	d := mx - mn
	var h float64
	switch {
	case d == 0:
	case mx == r:
		h = math.Mod((g-b)/d, 6)
	case mx == g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	if h *= 60; h < 0 {
		h += 360
	}
	var s float64
	if mx != 0 {
		s = d / mx
	}
	return h, s, mx
}

// hsvHex formats a hue, saturation and value as a css hex color.
func hsvHex(h, s, v float64) string {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return rgbHex(r+m, g+m, b+m)
}

// rgbHex formats unit red, green and blue as a css hex color.
func rgbHex(r, g, b float64) string {
	const hextet = "0123456789abcdef"
	buf := []byte("#000000")
	for i, f := range []float64{r, g, b} {
		n := int(math.Round(min(max(f, 0), 1) * 255))
		buf[1+i*2], buf[2+i*2] = hextet[n>>4], hextet[n&0xf]
	}
	return string(buf)
}
