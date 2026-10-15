// Package resvg supplies a svg decoder for iv.
//
// See: https://github.com/xo/resvg
package resvg

import (
	"context"
	"errors"
	"image/color"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/tdewolff/canvas"
	"github.com/xo/resvg"
)

func init() {
	decoder.RegisterBuiltin(
		"resvg",
		decoder.Desc("Scalable Vector Graphics"),
		decoder.Extension("svg", "svgz"),
		decoder.MimeType("image/svg+xml", "image/svg"),
		// a .svgz is gzipped, so it sniffs as application/gzip and nothing
		// but the extension identifies it -- resvg decompresses it itself
		decoder.MimeTypeExtensionMatch("application/gzip", "svgz"),
		decoder.Init(open, shut),
		decoder.Decoder(decode),
	)
}

// open builds the renderer, with the font families of [families] and the
// configured background.
//
// resvg builds its options when a renderer is made, and a renderer cannot be
// changed afterward, so the options are passed here. A change to
// [resvg.Default] is ignored, which is what made the families of
// [configureFamilies] stop taking effect with resvg v0.12.1.
//
// The display size is not passed on. The best fit of resvg grows an svg to
// fill whatever box it is given, so an svg would arrive at the width of the
// terminal whatever size it declares. An svg is rasterized at the size it
// declares, and ivimg.Fit then applies the same ceiling and floor to it as to
// any other image.
func open(ctx context.Context) (any, error) {
	var bg color.Color = color.Transparent
	if c := ivctx.Get(ctx).Bg; c != nil {
		bg = c
	}
	opts := append(configureFamilies(), resvg.WithBackground(bg))
	return resvg.New(opts...), nil
}

// shut releases the renderer.
func shut(ctx context.Context) error {
	if r, ok := decoder.State(ctx).(*resvg.Resvg); ok {
		return r.Close()
	}
	return nil
}

// decode renders the svg with the renderer that [open] built.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rend, ok := decoder.State(ctx).(*resvg.Resvg)
	if !ok {
		return nil, errors.New("renderer not initialized")
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return rend.Render(buf)
}

// families are the candidate font families for each css generic family, in
// order of preference.
var families = []struct {
	opt        func(string) resvg.Option
	candidates []string
}{
	{resvg.WithSerifFamily, serif},
	{resvg.WithMonospaceFamily, monospace},
	{resvg.WithCursiveFamily, cursive},
	{resvg.WithFantasyFamily, fantasy},
	// the family used when a svg names a font that isn't installed
	{resvg.WithFontFamily, serif},
}

// Candidate font families for the css generic families. There is no
// sans-serif here. resvg drew latin text for sans-serif when the families
// were chosen, and from v0.12.1 [resvg.WithSansSerifFamily] can set it if a
// system needs it.
var (
	serif     = []string{"Times New Roman", "Liberation Serif", "Tinos", "Nimbus Roman", "DejaVu Serif", "Noto Serif", "FreeSerif"}
	monospace = []string{"Courier New", "Liberation Mono", "Cousine", "Nimbus Mono PS", "DejaVu Sans Mono", "Noto Sans Mono", "Menlo", "Consolas", "FreeMono"}
	cursive   = []string{"Comic Sans MS", "Apple Chancery", "Z003", "URW Chancery L", "DejaVu Sans"}
	fantasy   = []string{"Impact", "Papyrus", "P052", "URW Palladio L", "DejaVu Sans"}
)

// configureFamilies returns the options that point the css generic font
// families of resvg at the first candidate family that is installed on the
// system.
//
// resvg resolves the generic families through fontdb's fontconfig parser,
// which takes the first <prefer> family it sees for a generic without
// weighing the coverage or priority that fontconfig itself applies. A system
// with the urw/gsfonts configs installed consequently resolves serif to
// "Standard Symbols PS", rendering latin text as greek letters.
func configureFamilies() []resvg.Option {
	var opts []resvg.Option
	for _, family := range families {
		for _, name := range family.candidates {
			if _, ok := canvas.FindSystemFont(name, canvas.FontRegular); ok {
				opts = append(opts, family.opt(name))
				break
			}
		}
	}
	return opts
}
