//go:build !386 && !arm

// Package fitz supplies a mupdf (fitz) decoder for iv, covering document
// formats such as epub, xps, mobi, and fb2.
//
// See: https://github.com/gen2brain/go-fitz
package fitz

import (
	"context"
	"fmt"
	"image"
	"io"

	"github.com/gen2brain/go-fitz"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.Register(
		"fitz",
		decoder.Desc("mupdf (epub, xps, mobi, fb2, psd)"),
		// not cbz: mupdf opens comic archives too, but the archives decoder
		// owns them, and a .cbz sniffs as a generic application/zip which
		// would otherwise let this one claim it by extension first
		decoder.Extension("epub", "xps", "oxps", "mobi", "fb2", "psd"),
		decoder.MimeType(
			"application/epub+zip",
			"application/x-mobipocket-ebook",
			"application/oxps",
			"application/vnd.ms-xpsdocument",
			"text/fb2+xml",
			"image/vnd.adobe.photoshop",
		),
		decoder.MimeTypeExtensionMatch(
			"text/xml", "fb2",
			"application/zip", "xps",
			"application/zip", "oxps",
		),
		decoder.Decoder(decode),
	)
}

// decode renders a page of the document with mupdf.
func decode(ctx context.Context, r io.Reader) (any, error) {
	d, err := fitz.NewFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("fitz load: %w", err)
	}
	defer d.Close()
	n := d.NumPage()
	ivctx.Logf(ctx, "fitz pages: %d", n)
	if n == 0 {
		return nil, fmt.Errorf("fitz: document has no pages")
	}
	page := ivctx.Page(ctx, n)
	var img *image.RGBA
	if dpi := ivctx.Get(ctx).DPI; dpi != 0 {
		img, err = d.ImageDPI(page, float64(dpi))
	} else {
		img, err = d.Image(page)
	}
	if err != nil {
		return nil, fmt.Errorf("fitz render: %w", err)
	}
	return img, nil
}
