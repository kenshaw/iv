package tag

import (
	"strings"
	"sync"

	"github.com/tdewolff/canvas"
)

// families are the families the card is set in, in order of preference. The
// first one installed is both named in the svg and used to measure, so that
// what is measured is what resvg goes on to draw.
var families = []string{
	"DejaVu Sans",
	"Helvetica Neue",
	"Helvetica",
	"Arial",
	"Liberation Sans",
	"Noto Sans",
	"FreeSans",
}

// face is the resolved card font.
type face struct {
	family        string
	regular, bold *canvas.Font
}

// resolve finds the first of [families] installed, once. Returns nil when
// none of them are, which leaves the card measuring by estimate and naming
// only the generic family.
var resolve = sync.OnceValue(func() *face {
	for _, family := range families {
		regular, err := canvas.LoadSystemFont(family, canvas.FontRegular)
		if err != nil {
			continue
		}
		bold, err := canvas.LoadSystemFont(family, canvas.FontBold)
		if err != nil {
			bold = regular
		}
		return &face{family: family, regular: regular, bold: bold}
	}
	return nil
})

// fontFamily returns the font stack the svg names. The generic stays last so
// a system without any of the families still gets something proportional.
func fontFamily() string {
	if f := resolve(); f != nil {
		return f.family + ",sans-serif"
	}
	return "sans-serif"
}

// mmPerPt converts the millimetres canvas measures in back to points.
const mmPerPt = 25.4 / 72.0

// Fallback advances, as a multiple of the font size. Only used when no font
// could be loaded to measure with: deliberate overestimates of a sans-serif's
// average, so a line is trimmed early rather than running past the card.
const (
	boldAdv = 0.60
	regAdv  = 0.55
)

// measure returns the width s renders to at the size, in the svg's units.
// Canvas sizes a face in points and answers in millimetres, and one svg user
// unit is one point's worth of em at the same number, so the conversions
// cancel and what comes back is directly comparable to a width in the card's
// geometry.
func measure(s string, size int, bold bool) float64 {
	f := resolve()
	if f == nil {
		adv := regAdv
		if bold {
			adv = boldAdv
		}
		return float64(len([]rune(s))) * float64(size) * adv
	}
	font := f.regular
	if bold {
		font = f.bold
	}
	return font.Face(float64(size), canvas.Black).TextWidth(s) / mmPerPt
}

// fit truncates s to what fits in width at the given size, appending an
// ellipsis when it has to cut.
func fit(s string, size int, bold bool, width int) string {
	w := float64(width)
	if measure(s, size, bold) <= w {
		return s
	}
	r := []rune(s)
	for n := len(r) - 1; n > 0; n-- {
		if t := strings.TrimRight(string(r[:n]), " ") + "…"; measure(t, size, bold) <= w {
			return t
		}
	}
	return ""
}

// shrink returns the largest of the sizes at which s fits in width, and s
// fitted to it -- truncated only when it does not fit even at the smallest.
func shrink(s string, sizes []int, bold bool, width int) (int, string) {
	for _, size := range sizes {
		if fit(s, size, bold, width) == s {
			return size, s
		}
	}
	size := sizes[len(sizes)-1]
	return size, fit(s, size, bold, width)
}
