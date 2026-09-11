// Package fontimg supplies a font preview decoder for iv.
//
// See: https://github.com/kenshaw/fontimg
package fontimg

import (
	"context"
	"io"

	"github.com/kenshaw/fontimg"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	// libmagic identifies bare font files that content sniffing reports as
	// application/octet-stream
	decoder.RegisterMimeType("font/ttf", `^TrueType Font data`, `^OpenType font data`)
	decoder.RegisterMimeType("font/woff", `^Web Open Font Format`)
	decoder.Register(
		"fontimg",
		decoder.Desc("Font previews"),
		decoder.Extension("ttf", "ttc", "otf", "woff", "woff2", "sfnt", "eot", "pfb"),
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

// decode rasterizes a preview of the font.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	c := ivctx.Get(ctx)
	return fontimg.
		New(buf, ivctx.PathName(ctx)).
		Rasterize(
			nil,
			int(c.FontSize),
			c.FontStyle,
			c.FontVariant,
			c.FontFg,
			c.FontBg,
			float64(c.FontDPI),
			float64(c.FontMargin),
		)
}
