// Package fontimg supplies a font preview decoder for iv.
//
// It draws a specimen of the font with okage (D48).
//
// See: https://github.com/xo/okage
package fontimg

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/font"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage"
	"github.com/xo/okage/svg"
)

// The name IDs of the name table that the specimen reads.
const (
	nameSubfamily            = 2
	nameVersion              = 5
	nameTypographicSubfamily = 17
	nameSampleText           = 19
)

// The lines of every specimen, each with the points that it adds to
// --font-size: the characters next to each other at the size, then pangrams
// at growing sizes.
var (
	characters = []sample{
		{"abcdefghijklmnopqrstuvwxyz", 0},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", 0},
		{"0123456789.<:,>;('~\"){!@#$%^&*?`=}[_\\-/+]", 0},
	}
	pangrams = []sample{
		{"The quick brown fox jumps over the lazy dog.", 0},
		{"Pack my box with five dozen liquor jugs.", 6},
		{"Jackdaws love my big sphinx of quartz.", 12},
		{"The five boxing wizards jump quickly.", 18},
	}
)

// sample is a line of every specimen, and the points that it adds to
// --font-size.
type sample struct {
	text string
	size int
}

// emojiLine is the line of the specimen of a font that has color emoji. It
// follows the characters, or, in a font of emoji, takes the place of the
// characters and of the pangrams, at their sizes.
const emojiLine = "😀😂😍🥳😎🤖👍👋❤️🔥🌈⭐🍕🎉🚀🐱🐶🦄"

// emojiKind is the emoji that a face has.
type emojiKind int

// The emoji that a face has.
const (
	// noEmoji is a face with no color emoji.
	noEmoji emojiKind = iota
	// someEmoji is a face with color emoji and letters.
	someEmoji
	// onlyEmoji is a font of emoji: a face with color emoji and no letters.
	onlyEmoji
)

// ErrNoFace is the error when the data holds no face that okage loads.
var ErrNoFace = errors.New("no font face")

// ErrInvalidWeight is the error of a --font-weight outside 1 to 1000.
var ErrInvalidWeight = errors.New("invalid font weight")

func init() {
	// libmagic identifies bare font files that content sniffing reports as
	// application/octet-stream
	decoder.RegisterMimeType("font/ttf", `^TrueType Font data`, `^OpenType font data`)
	decoder.RegisterMimeType("font/woff", `^Web Open Font Format`)
	decoder.Register(
		"fontimg",
		decoder.Desc("Font previews"),
		decoder.Extension("ttf", "ttc", "otf", "otc", "woff", "woff2", "sfnt", "eot"),
		decoder.MimeType(
			"font/*",
			"application/font-sfnt",
			"application/vnd.ms-opentype",   // otf
			"application/vnd.ms-fontobject", // eot
			"application/x-font-ttf",
		),
		decoder.Decoder(decode),
	)
}

// decode draws a specimen of the font: its name, its style and its version,
// then its own sample text, or the alphabet, the digits and pangrams at
// growing sizes.
//
// okage loads TrueType, OpenType, collections, WOFF, WOFF2 and EOT. The
// font goes into a database of its own, so the text draws in it alone, and a
// character that it does not have draws its .notdef glyph. In a collection,
// --page selects the face.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	db := svg.NewFontDatabase()
	ids := db.LoadFontData(buf)
	if len(ids) == 0 {
		return nil, ErrNoFace
	}
	id := ids[ivctx.Page(ctx, len(ids))]
	info, ok := db.Face(id)
	if !ok {
		return nil, ErrNoFace
	}
	c := ivctx.Get(ctx)
	style, err := textStyle(c, info)
	if err != nil {
		return nil, err
	}
	emoji := emojiOf(db, style)
	lines := specimen(db, id, info, ivctx.PathName(ctx), int(c.FontSize), c.FontWeight, emoji)
	ivctx.Logf(ctx, "font: %s, %d faces, weight %d, style %s, emoji: %d", style.Family, len(ids), style.Weight, style.Style, emoji)
	return draw(db, lines, style, c)
}

// line is a line of the specimen, and its size in points. A line of the
// system is drawn in the font of the cards of iv, and not in the font.
type line struct {
	text   string
	size   int
	system bool
}

// specimen returns the lines of the specimen of the face (D48). The first
// names the family, the style and the version of the face, and the weight
// that --font-weight sets in place of the weight of the face. The sample text
// of the face follows, when it has one. Then come the characters, a line of
// emoji when the face draws them in color, and the pangrams at growing sizes.
//
// A font of emoji cannot draw its own name, so the first line is in the font
// of the system, and the line of emoji follows at the sizes of the pangrams.
//
// The family is the typographic family, such as "Figtree", so the style is
// the typographic subfamily, such as "Light Italic", when the face has one.
func specimen(db *svg.FontDatabase, id svg.FontID, info *svg.FaceInfo, pathName string, size int, weight uint, emoji emojiKind) []line {
	head := familyName(info, pathName)
	if s := subfamily(db, id); s != "" {
		head += ", " + s
	}
	if s := version(db, id); s != "" {
		head += ", v" + s
	}
	if weight != 0 {
		head += ", weight " + strconv.FormatUint(uint64(weight), 10)
	}
	if emoji == onlyEmoji {
		lines := []line{{head, size + 2, true}}
		for _, l := range pangrams {
			lines = append(lines, line{emojiLine, size + l.size, false})
		}
		return lines
	}
	lines := []line{{head, size + 2, false}}
	if s, ok := db.FaceName(id, nameSampleText); ok {
		for t := range strings.SplitSeq(s, "\n") {
			if t = strings.TrimSpace(t); t != "" {
				lines = append(lines, line{t, size, false})
			}
		}
	}
	for _, l := range characters {
		lines = append(lines, line{l.text, size + l.size, false})
	}
	if emoji == someEmoji {
		lines = append(lines, line{emojiLine, size, false})
	}
	for _, l := range pangrams {
		lines = append(lines, line{l.text, size + l.size, false})
	}
	return lines
}

// emojiOf returns the emoji that the face has. A face has color emoji when it
// draws the line of emoji in color. A face with no emoji draws its .notdef
// glyph for each one, with no color. A font of emoji draws no glyph of the
// lowercase letters.
func emojiOf(db *svg.FontDatabase, style svg.TextStyle) emojiKind {
	style.Size = 16
	run, err := svg.ShapeText(emojiLine, style, svg.WithFontDatabase(db))
	if err != nil || !run.HasColorGlyphs() {
		return noEmoji
	}
	if _, err := svg.ShapeText(characters[0].text, style, svg.WithFontDatabase(db)); errors.Is(err, okage.ErrSVGTextNoFont) {
		return onlyEmoji
	}
	return someEmoji
}

// version returns the version of the face, without the word "Version" and
// without what follows a semicolon. A font often adds the data of its build
// there, as in "Version 2.057;GOOG;noto-emoji:20260911:fc4ca365e7c2".
func version(db *svg.FontDatabase, id svg.FontID) string {
	s, _ := db.FaceName(id, nameVersion)
	s, _, _ = strings.Cut(s, ";")
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "Version "))
}

// subfamily returns the typographic subfamily of the face, or its subfamily
// when it has none, or "".
func subfamily(db *svg.FontDatabase, id svg.FontID) string {
	for _, nameID := range []uint16{nameTypographicSubfamily, nameSubfamily} {
		if s, ok := db.FaceName(id, nameID); ok && s != "" {
			return s
		}
	}
	return ""
}

// familyName returns the family of the face, or the name of the file
// without its extension when the face names none.
func familyName(info *svg.FaceInfo, pathName string) string {
	if len(info.Families) != 0 && info.Families[0].Name != "" {
		return info.Families[0].Name
	}
	return strings.TrimSuffix(filepath.Base(pathName), filepath.Ext(pathName))
}

// textStyle returns the style that selects the face: its family, and its
// weight and style, or those that --font-weight and --font-italic give. On a
// variable font, they set the axes wght, ital and slnt.
func textStyle(c *ivctx.Config, info *svg.FaceInfo) (svg.TextStyle, error) {
	style := svg.TextStyle{
		Family: strconv.Quote(familyName(info, "")),
		Weight: info.Weight,
		Style:  info.Style,
	}
	switch w := c.FontWeight; {
	case w > 1000:
		return style, fmt.Errorf("font weight %d: %w (1 to 1000)", w, ErrInvalidWeight)
	case w != 0:
		style.Weight = uint16(w)
	}
	if c.FontItalic {
		style.Style = svg.FontStyleItalic
	}
	return style, nil
}

// draw draws the lines one under the other, in the color --font-fg on
// --font-bg, with --font-margin around them. A line that no font draws is
// left out. As in the fontimg package that
// came before, a size is in points and the margin is in millimetres, at
// --font-dpi.
func draw(db *svg.FontDatabase, lines []line, style svg.TextStyle, c *ivctx.Config) (*image.RGBA, error) {
	dpi := float32(c.FontDPI)
	if dpi <= 0 {
		dpi = 100
	}
	pt, margin := dpi/72, float32(c.FontMargin)*dpi/25.4
	var runs []*svg.TextRun
	var width, height float32
	for _, l := range lines {
		s, ldb := style, db
		if l.system {
			s, ldb = svg.TextStyle{Family: font.Family(), Weight: 400}, font.Database()
		}
		s.Size = float32(l.size) * pt
		run, err := svg.ShapeText(l.text, s, svg.WithFontDatabase(ldb))
		switch {
		case errors.Is(err, okage.ErrSVGTextNoFont):
			// a line that no font draws, as the name of a font of emoji on
			// a system with none of the fonts of the cards, is left out
			continue
		case err != nil:
			return nil, fmt.Errorf("font: %w", err)
		}
		runs = append(runs, run)
		width, height = max(width, run.Width()), height+run.Bounds().Height()
	}
	if len(runs) == 0 {
		return nil, ErrNoFace
	}
	w, h := math.Ceil(float64(width+2*margin)), math.Ceil(float64(height+2*margin))
	pixmap, ok := okage.NewPixmap(uint32(w), uint32(h))
	if !ok {
		return nil, fmt.Errorf("font: no pixmap of %vx%v", w, h)
	}
	pixmap.Fill(okageColor(c.FontBg, color.White))
	paint := okage.DefaultPaint()
	fg := okageColor(c.FontFg, color.Black)
	paint.Shader = okage.SolidColor(fg)
	mut := pixmap.AsMut()
	y := margin
	for _, run := range runs {
		b := run.Bounds()
		// the run starts on its baseline, and its bounds start at its ascent
		baseline := y - b.Top()
		pb := okage.NewPathBuilder()
		run.AppendPath(pb, margin, baseline)
		if p, ok := pb.Finish(); ok {
			pixmap.FillPath(p, &paint, okage.FillRuleWinding, okage.IdentityTransform(), nil)
		}
		if run.HasColorGlyphs() {
			run.RenderColorGlyphs(&mut, okage.TransformFromTranslate(margin, baseline))
		}
		y += b.Height()
	}
	return pixmap.AsRGBA(), nil
}

// okageColor returns c as a color of okage, or def when c is nil.
func okageColor(c *colors.Color, def color.Color) okage.Color {
	v := def
	if c != nil {
		v = c
	}
	n, _ := color.NRGBAModel.Convert(v).(color.NRGBA)
	return okage.ColorFromRGBA8(n.R, n.G, n.B, n.A)
}
