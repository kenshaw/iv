package tag

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// Card geometry. The card is a wide landscape panel: cover art on the left,
// type to its right, and the waveform spanning the full width beneath both.
const (
	cardW   = 1000
	cardH   = 448
	pad     = 40
	artSize = 240
	textX   = pad + artSize + pad
	waveMid = 350
	waveAmp = 50
	bars    = 152
)

// Type sizes, and the factor to estimate a string's rendered width by. Real
// metrics would need the font loaded; these are deliberate overestimates of
// a sans-serif's average advance, so a line is trimmed a little early rather
// than running past the edge of the card.
const (
	artistSize = 24
	albumSize  = 18
	metaSize   = 14
	boldAdv    = 0.60
	regAdv     = 0.55
)

// titleSizes are the sizes a title is set at, largest first. A long title
// steps down through them before it is truncated, so the whole of most
// titles fits.
var titleSizes = []int{40, 35, 30, 26}

// Card colors that don't come from the accent.
const (
	cardBg    = "#0c0c11"
	titleFg   = "#ffffff"
	albumFg   = "#b6b6c2"
	metaFg    = "#7b7b8a"
	scrimFg   = "#0c0c11"
	emptyWave = "#23232d"
)

// card is everything the svg draws.
type card struct {
	title    string
	artist   string
	album    string
	meta     string
	duration time.Duration
	art      []byte
	artMime  string
	accent   accent
	peaks    []float64
}

// svg renders the card.
func (c *card) svg() []byte {
	a := c.accent
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" viewBox="0 0 %d %d">`, cardW, cardH, cardW, cardH)
	b.WriteString(`<defs>`)
	fmt.Fprintf(&b, `<clipPath id="c"><rect width="%d" height="%d" rx="18"/></clipPath>`, cardW, cardH)
	fmt.Fprintf(&b, `<clipPath id="a"><rect x="%d" y="%d" width="%d" height="%d" rx="14"/></clipPath>`, pad, pad, artSize, artSize)
	fmt.Fprintf(&b, `<linearGradient id="s" x1="0" y1="0" x2="0.3" y2="1"><stop offset="0" stop-color="%s" stop-opacity="0.62"/><stop offset="0.6" stop-color="%s" stop-opacity="0.90"/><stop offset="1" stop-color="%s" stop-opacity="0.99"/></linearGradient>`, scrimFg, scrimFg, scrimFg)
	fmt.Fprintf(&b, `<linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="%s"/><stop offset="0.5" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>`, a.hex(0.85, 1.0), a.hex(1.0, 0.92), a.hex(0.9, 0.62))
	b.WriteString(`<filter id="b" x="-25%" y="-25%" width="150%" height="150%"><feGaussianBlur stdDeviation="56"/></filter>`)
	b.WriteString(`</defs>`)
	b.WriteString(`<g clip-path="url(#c)">`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, cardW, cardH, cardBg)
	if href := c.href(); href != "" {
		// the cover art, blown up and blurred, is the card's backdrop -- it
		// is what ties the card to the record without reading as an image
		fmt.Fprintf(&b, `<image x="%d" y="%d" width="%d" height="%d" preserveAspectRatio="xMidYMid slice" filter="url(#b)" opacity="0.85" xlink:href="%s"/>`, -cardW/4, -cardH, cardW+cardW/2, cardH*3, href)
		fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="url(#s)"/>`, cardW, cardH)
		fmt.Fprintf(&b, `<image x="%d" y="%d" width="%d" height="%d" preserveAspectRatio="xMidYMid slice" clip-path="url(#a)" xlink:href="%s"/>`, pad, pad, artSize, artSize, href)
	} else {
		fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="url(#s)"/>`, cardW, cardH)
		c.placeholder(&b)
	}
	fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" rx="14" fill="none" stroke="#ffffff" stroke-opacity="0.10"/>`, pad, pad, artSize, artSize)
	c.text(&b)
	c.wave(&b)
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// href returns the cover art as a data url, empty when there is no art.
func (c *card) href() string {
	if len(c.art) == 0 {
		return ""
	}
	mime := c.artMime
	if mime == "" {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(c.art)
}

// placeholder draws a note glyph in place of missing cover art.
func (c *card) placeholder(b *strings.Builder) {
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="14" fill="%s" fill-opacity="0.14"/>`, pad, pad, artSize, artSize, c.accent.hex(1, 1))
	fmt.Fprintf(b, `<g transform="translate(%d,%d)" fill="%s" fill-opacity="0.55">`, pad+artSize/2, pad+artSize/2, c.accent.hex(1, 1))
	b.WriteString(`<ellipse cx="-26" cy="34" rx="30" ry="21" transform="rotate(-20 -26 34)"/>`)
	b.WriteString(`<path d="M-2,34 L-2,-54 L10,-58 L10,34 Z"/>`)
	b.WriteString(`<path d="M10,-58 C46,-46 56,-22 48,2 C52,-22 34,-34 10,-28 Z"/>`)
	b.WriteString(`</g>`)
}

// line is one line of the card's type block.
type line struct {
	text    string
	size    int
	weight  int
	fill    string
	spacing string
	// lead is the vertical space the line occupies, as a multiple of its
	// size. The block is stacked and centered, so a missing tag closes its
	// own gap rather than leaving a hole.
	lead float64
}

// text stacks the title, artist, album and technical lines, centered against
// the cover art.
func (c *card) text(b *strings.Builder) {
	w := cardW - textX - pad
	size, title := shrink(c.title, titleSizes, boldAdv, w)
	lines := []line{{title, size, 700, titleFg, "0", 1.5}}
	if c.artist != "" {
		lines = append(lines, line{fit(c.artist, artistSize, regAdv, w), artistSize, 600, c.accent.hex(0.8, 1), "0", 1.55})
	}
	if c.album != "" {
		lines = append(lines, line{fit(c.album, albumSize, regAdv, w), albumSize, 400, albumFg, "0", 1.9})
	}
	if c.meta != "" {
		lines = append(lines, line{fit(c.meta, metaSize, regAdv, w), metaSize, 400, metaFg, "0.6", 1.4})
	}
	var total float64
	for _, l := range lines {
		total += float64(l.size) * l.lead
	}
	// the block is centered on the art, and its first baseline sits a cap
	// height below the top of the block
	y := float64(pad+artSize/2) - total/2 + float64(lines[0].size)*0.78
	for _, l := range lines {
		fmt.Fprintf(b, `<text x="%d" y="%.1f" font-family="%s" font-size="%d" font-weight="%d" letter-spacing="%s" fill="%s">%s</text>`, textX, y, fontFamily, l.size, l.weight, l.spacing, l.fill, esc(l.text))
		y += float64(l.size) * l.lead
	}
}

// wave draws the waveform as mirrored bars, with the track's start and end
// times beneath its ends.
func (c *card) wave(b *strings.Builder) {
	const (
		width = cardW - 2*pad
		pitch = float64(width) / bars
	)
	bw := pitch * 0.56
	fmt.Fprintf(b, `<rect x="%d" y="%.1f" width="%d" height="1" fill="%s" fill-opacity="0.35"/>`, pad, float64(waveMid)-0.5, width, c.accent.hex(1, 1))
	fill, empty := "url(#w)", c.peaks == nil
	if empty {
		fill = emptyWave
	}
	fmt.Fprintf(b, `<g fill="%s">`, fill)
	for i := range bars {
		// with no waveform the bars collapse to a flat rule, so the card
		// still reads as an audio card rather than looking broken
		h := 3.0
		if !empty {
			h = max(c.peaks[i]*waveAmp*2, 3)
		}
		x := float64(pad) + float64(i)*pitch + (pitch-bw)/2
		fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f"/>`, x, float64(waveMid)-h/2, bw, h, bw/2)
	}
	b.WriteString(`</g>`)
	fmt.Fprintf(b, `<text x="%d" y="428" font-family="%s" font-size="13" fill="%s">0:00</text>`, pad, fontFamily, metaFg)
	if c.duration > 0 {
		fmt.Fprintf(b, `<text x="%d" y="428" text-anchor="end" font-family="%s" font-size="13" fill="%s">%s</text>`, cardW-pad, fontFamily, metaFg, clock(c.duration))
	}
}

// fontFamily is the card's font stack. The generic stays last so a system
// without any of the named families still gets something proportional.
const fontFamily = "DejaVu Sans,Helvetica Neue,Helvetica,Arial,sans-serif"

// shrink returns the largest of the sizes at which s fits in width, and s
// fitted to it -- truncated only when it does not fit even at the smallest.
func shrink(s string, sizes []int, adv float64, width int) (int, string) {
	for _, size := range sizes {
		if fit(s, size, adv, width) == s {
			return size, s
		}
	}
	size := sizes[len(sizes)-1]
	return size, fit(s, size, adv, width)
}

// fit truncates s to what fits in width at the given size, appending an
// ellipsis when it has to cut.
func fit(s string, size int, adv float64, width int) string {
	per := float64(size) * adv
	n := int(float64(width) / per)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 2 {
		return ""
	}
	return strings.TrimRight(string(r[:n-1]), " ") + "…"
}

// clock formats a duration as m:ss, or h:mm:ss past an hour.
func clock(d time.Duration) string {
	d = d.Round(time.Second)
	h, m, s := int(d/time.Hour), int(d/time.Minute)%60, int(d/time.Second)%60
	if h != 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// esc escapes text for inclusion in the svg.
var esc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace
