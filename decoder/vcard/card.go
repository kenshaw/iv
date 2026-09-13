package vcard

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"github.com/kenshaw/iv/internal/font"
	"github.com/skip2/go-qrcode"
)

// Card geometry, in the proportions of a printed business card: 3.5 by 2
// inches, laid out as type on the left and the code on the right.
const (
	cardW   = 1000
	cardH   = 572
	pad     = 52
	stripeW = 14
	qrSize  = 260
	qrX     = cardW - pad - qrSize
	// the caption below the code belongs to it, so the two are centered
	// together rather than the code alone
	qrCaption = 32
	qrY       = (cardH - qrSize - qrCaption) / 2
	textX     = pad + stripeW + 36
	textW     = qrX - textX - 40
)

// Type sizes.
const (
	nameSize  = 42
	titleSize = 19
	orgSize   = 19
	lineSize  = 17
	labelSize = 11
)

// Card colors. A business card is light, and a qr code has to be dark on
// light to be read at all, so the whole card follows the code.
const (
	cardBg  = "#fdfdfb"
	nameFg  = "#16161a"
	titleFg = "#4a4a55"
	lineFg  = "#2c2c34"
	mutedFg = "#8a8a96"
	ruleFg  = "#e2e2e6"
	codeFg  = "#16161a"
)

// contact is one line of contact detail: an icon, the value, and what kind of
// thing it is.
type contact struct {
	icon  string
	value string
	label string
}

// card is everything the svg draws.
type card struct {
	name     string
	title    string
	org      string
	contacts []contact
	accent   accent
	// raw is the vcard as it was written, which is what the qr code carries:
	// scanning the card hands a phone the whole record rather than a
	// transcription of it.
	raw string
}

// svg renders the card.
func (c *card) svg() []byte {
	a := c.accent
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, cardW, cardH, cardW, cardH)
	b.WriteString(`<defs>`)
	fmt.Fprintf(&b, `<clipPath id="c"><rect width="%d" height="%d" rx="20"/></clipPath>`, cardW, cardH)
	fmt.Fprintf(&b, `<linearGradient id="s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>`, a.hex(0.85, 1), a.hex(1, 0.72))
	b.WriteString(`</defs>`)
	b.WriteString(`<g clip-path="url(#c)">`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, cardW, cardH, cardBg)
	// the stripe down the edge, the one piece of the card that carries the
	// accent at full strength
	fmt.Fprintf(&b, `<rect x="%d" y="0" width="%d" height="%d" fill="url(#s)"/>`, pad, stripeW, cardH)
	c.text(&b)
	c.code(&b)
	fmt.Fprintf(&b, `<rect x="0.5" y="0.5" width="%d" height="%d" rx="20" fill="none" stroke="%s"/>`, cardW-1, cardH-1, ruleFg)
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// Vertical rhythm of the type block, in the order it is stacked.
const (
	titleLead = 34
	orgLead   = 26
	ruleLead  = 22
	lineLead  = 30
)

// height returns how tall the type block stands, so it can be centered on the
// card rather than hung from the top.
func (c *card) height(nameSize int) float64 {
	h := float64(nameSize)
	if c.title != "" {
		h += titleLead
	}
	if c.org != "" {
		h += orgLead
	}
	if len(c.contacts) != 0 {
		h += ruleLead + lineLead*float64(len(c.contacts))
	}
	return h
}

// text draws the name, what the person does, and the contact lines.
func (c *card) text(b *strings.Builder) {
	size, name := font.Shrink(c.name, []int{nameSize, 36, 30, 25}, true, textW)
	y := (cardH-c.height(size))/2 + float64(size)*0.78
	fmt.Fprintf(b, `<text x="%d" y="%.0f" font-family="%s" font-size="%d" font-weight="700" fill="%s">%s</text>`, textX, y, font.Family(), size, nameFg, esc(name))
	if c.title != "" {
		y += titleLead
		fmt.Fprintf(b, `<text x="%d" y="%.0f" font-family="%s" font-size="%d" font-weight="600" fill="%s" letter-spacing="0.4">%s</text>`, textX, y, font.Family(), titleSize, c.accent.hex(1, 0.62), esc(font.Fit(c.title, titleSize, false, textW)))
	}
	if c.org != "" {
		y += orgLead
		fmt.Fprintf(b, `<text x="%d" y="%.0f" font-family="%s" font-size="%d" fill="%s">%s</text>`, textX, y, font.Family(), orgSize, titleFg, esc(font.Fit(c.org, orgSize, false, textW)))
	}
	if len(c.contacts) == 0 {
		return
	}
	y += ruleLead
	fmt.Fprintf(b, `<rect x="%d" y="%.0f" width="%d" height="1" fill="%s"/>`, textX, y, min(textW, 220), ruleFg)
	for _, v := range c.contacts {
		y += lineLead
		// the icon sits on the text's baseline, drawn in a 16 unit box
		fmt.Fprintf(b, `<g transform="translate(%d,%.0f)" fill="none" stroke="%s" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">%s</g>`, textX, y-12, c.accent.hex(1, 0.6), v.icon)
		fmt.Fprintf(b, `<text x="%d" y="%.0f" font-family="%s" font-size="%d" fill="%s">%s</text>`, textX+28, y, font.Family(), lineSize, lineFg, esc(font.Fit(v.value, lineSize, false, textW-28)))
	}
}

// code draws the qr code carrying the whole vcard, module by module rather
// than as an embedded raster, so it stays sharp at whatever size the card is
// shown at.
func (c *card) code(b *strings.Builder) {
	q, err := qrcode.New(c.raw, qrcode.Medium)
	if err != nil {
		return
	}
	bitmap := q.Bitmap()
	n := len(bitmap)
	if n == 0 {
		return
	}
	// the bitmap comes with a four module quiet zone already around it, which
	// is drawn as the code's own light panel
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="#ffffff"/>`, qrX-10, qrY-10, qrSize+20, qrSize+20)
	m := float64(qrSize) / float64(n)
	var d strings.Builder
	for y, row := range bitmap {
		for x, on := range row {
			if !on {
				continue
			}
			// each module is a rect in one path, which keeps the svg to a
			// single element rather than a thousand
			fmt.Fprintf(&d, "M%.2f %.2fh%.2fv%.2fh-%.2fz",
				float64(qrX)+float64(x)*m, float64(qrY)+float64(y)*m, m, m, m)
		}
	}
	fmt.Fprintf(b, `<path d="%s" fill="%s" shape-rendering="crispEdges"/>`, d.String(), codeFg)
	fmt.Fprintf(b, `<text x="%d" y="%d" font-family="%s" font-size="%d" fill="%s" text-anchor="middle" letter-spacing="1.2">SCAN FOR CONTACT</text>`,
		qrX+qrSize/2, qrY+qrSize+32, font.Family(), labelSize, mutedFg)
}

// accent is the color the card's stripe and icons are drawn in, derived from
// the name so that a person's card is always the same color.
type accent struct {
	h, s, v float64
}

// accentFor derives the accent from the text.
func accentFor(text string) accent {
	if text == "" {
		return accent{h: 205, s: 0.70, v: 0.86}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(text))
	return accent{h: float64(h.Sum32() % 360), s: 0.62, v: 0.80}
}

// hex returns the accent as a css color, with its saturation and value
// scaled.
func (a accent) hex(sf, vf float64) string {
	return hsvHex(a.h, min(a.s*sf, 1), min(a.v*vf, 1))
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
	const hextet = "0123456789abcdef"
	buf := []byte("#000000")
	for i, f := range []float64{r + m, g + m, b + m} {
		n := int(math.Round(min(max(f, 0), 1) * 255))
		buf[1+i*2], buf[2+i*2] = hextet[n>>4], hextet[n&0xf]
	}
	return string(buf)
}

// esc escapes text for inclusion in the svg.
var esc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace
