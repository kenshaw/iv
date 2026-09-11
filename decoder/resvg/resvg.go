// Package resvg supplies a svg decoder for iv.
//
// See: https://github.com/xo/resvg
package resvg

import (
	"context"

	"github.com/kenshaw/iv/decoder"
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
		decoder.Init(
			func(_ context.Context) (any, error) {
				configureFamilies(resvg.Default)
				return nil, nil
			},
			nil,
		),
		decoder.ImageDecoder(resvg.Decode),
	)
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

// Candidate font families for the css generic families. There's no
// sans-serif here, as resvg doesn't export an option to set it.
var (
	serif     = []string{"Times New Roman", "Liberation Serif", "Tinos", "Nimbus Roman", "DejaVu Serif", "Noto Serif", "FreeSerif"}
	monospace = []string{"Courier New", "Liberation Mono", "Cousine", "Nimbus Mono PS", "DejaVu Sans Mono", "Noto Sans Mono", "Menlo", "Consolas", "FreeMono"}
	cursive   = []string{"Comic Sans MS", "Apple Chancery", "Z003", "URW Chancery L", "DejaVu Sans"}
	fantasy   = []string{"Impact", "Papyrus", "P052", "URW Palladio L", "DejaVu Sans"}
)

// configureFamilies points resvg's css generic font families at the first
// candidate family actually installed on the system.
//
// resvg resolves the generic families through fontdb's fontconfig parser,
// which takes the first <prefer> family it sees for a generic without
// weighing the coverage or priority that fontconfig itself applies. A system
// with the urw/gsfonts configs installed consequently resolves serif to
// "Standard Symbols PS", rendering latin text as greek letters.
func configureFamilies(r *resvg.Resvg) {
	for _, family := range families {
		for _, name := range family.candidates {
			if _, ok := canvas.FindSystemFont(name, canvas.FontRegular); ok {
				family.opt(name)(r)
				break
			}
		}
	}
}
