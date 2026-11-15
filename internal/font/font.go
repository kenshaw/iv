// Package font resolves the font iv's generated cards are set in, and
// measures text in it with okage (D48).
//
// Measurement and rendering have to agree: the family named in an svg is the
// one measured here, so fitting a line to a box is arithmetic rather than
// guesswork.
package font

import (
	"strconv"
	"strings"
	"sync"

	"github.com/xo/okage/svg"
)

// families are the families the card is set in, in order of preference. The
// first one installed is both named in the svg and used to measure, so that
// what is measured is what okage goes on to draw.
var families = []string{
	"DejaVu Sans",
	"Helvetica Neue",
	"Helvetica",
	"Arial",
	"Liberation Sans",
	"Noto Sans",
	"FreeSans",
}

// face is the resolved card font: its family, and the fonts of the system,
// which okage shapes the text with to measure it.
type face struct {
	family string
	db     *svg.FontDatabase
}

// resolve finds the first of [families] installed, once. Returns nil when
// none of them are, which leaves the card measuring by estimate and naming
// only the generic family.
var resolve = sync.OnceValue(func() *face {
	for _, family := range families {
		if Installed(family) {
			return &face{family: family, db: systemFonts()}
		}
	}
	return nil
})

// Database returns the fonts of the system, which the family of [Family]
// draws with. It loads them on its first call.
func Database() *svg.FontDatabase {
	return systemFonts()
}

// systemFonts loads the fonts of the system, once.
var systemFonts = sync.OnceValue(func() *svg.FontDatabase {
	// the fonts that the svg.Renderer of okage loads, so that the text is
	// measured in the font that it is drawn in
	db := svg.NewFontDatabase()
	db.LoadOSFonts()
	return db
})

// Installed returns true when a font of the system has the family.
func Installed(family string) bool {
	query := svg.FontQuery{Families: []svg.FontFamily{{Kind: svg.FontFamilyNamed, Name: family}}}
	_, ok := systemFonts().Query(query)
	return ok
}

// Family returns the font stack the svg names. The generic stays last so
// a system without any of the families still gets something proportional.
func Family() string {
	if f := resolve(); f != nil {
		return f.family + ",sans-serif"
	}
	return "sans-serif"
}

// Fallback advances, as a multiple of the font size. Only used when no font
// could be loaded to measure with: deliberate overestimates of a sans-serif's
// average, so a line is trimmed early rather than running past the card.
const (
	boldAdv = 0.60
	regAdv  = 0.55
)

// Measure returns the width s renders to at the size, in the svg's units.
// okage shapes s as a <text> element of the card does, in the same family
// and weight, so the width is the width that the card draws.
func Measure(s string, size int, bold bool) float64 {
	adv := regAdv
	if bold {
		adv = boldAdv
	}
	f := resolve()
	if f == nil {
		return float64(len([]rune(s))) * float64(size) * adv
	}
	style := svg.TextStyle{Family: strconv.Quote(f.family), Size: float32(size), Weight: 400}
	if bold {
		style.Weight = 700
	}
	run, err := svg.ShapeText(s, style, svg.WithFontDatabase(f.db))
	if err != nil {
		return float64(len([]rune(s))) * float64(size) * adv
	}
	return float64(run.Width())
}

// Fit truncates s to what fits in width at the given size, appending an
// ellipsis when it has to cut.
func Fit(s string, size int, bold bool, width int) string {
	w := float64(width)
	if Measure(s, size, bold) <= w {
		return s
	}
	r := []rune(s)
	for n := len(r) - 1; n > 0; n-- {
		if t := strings.TrimRight(string(r[:n]), " ") + "…"; Measure(t, size, bold) <= w {
			return t
		}
	}
	return ""
}

// Shrink returns the largest of the sizes at which s fits in width, and s
// fitted to it -- truncated only when it does not fit even at the smallest.
func Shrink(s string, sizes []int, bold bool, width int) (int, string) {
	for _, size := range sizes {
		if Fit(s, size, bold, width) == s {
			return size, s
		}
	}
	size := sizes[len(sizes)-1]
	return size, Fit(s, size, bold, width)
}
