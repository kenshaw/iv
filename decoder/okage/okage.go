// Package okage supplies a svg decoder for iv.
//
// okage is a port of resvg to Go.
//
// See: https://github.com/xo/okage
package okage

import (
	"context"
	"errors"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage/svg"
)

func init() {
	decoder.RegisterBuiltin(
		"okage",
		decoder.Desc("Scalable Vector Graphics"),
		decoder.Extension("svg", "svgz"),
		decoder.MimeType("image/svg+xml", "image/svg"),
		// a .svgz is gzipped, so it sniffs as application/gzip and nothing
		// but the extension identifies it -- okage decompresses it itself
		decoder.MimeTypeExtensionMatch("application/gzip", "svgz"),
		decoder.Init(open, nil),
		decoder.Decoder(decode),
	)
}

// open makes the renderer, with the configured background. The renderer
// loads the fonts of the system on its first render, and sets each css
// generic family to the first of its candidates that is installed.
//
// The display size is not passed on. An svg is rasterized at the size it
// declares, and ivimg.Fit then applies the same ceiling and floor to it as to
// any other image.
func open(ctx context.Context) (any, error) {
	var opts []svg.Option
	if bg := ivctx.Get(ctx).Bg; bg != nil {
		opts = append(opts, svg.WithBackground(bg))
	}
	return svg.New(opts...), nil
}

// decode renders the svg with the renderer that [open] made.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rend, ok := decoder.State(ctx).(*svg.Renderer)
	if !ok {
		return nil, errors.New("renderer not initialized")
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return rend.Render(buf)
}
