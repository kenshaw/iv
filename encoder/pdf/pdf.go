// Package pdf supplies a pdf encoder for iv, wrapping any decoded image in a
// pdf document.
//
// See: https://github.com/xo/blitz
package pdf

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"io"
	"strings"

	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/blitz"
)

func init() {
	encoder.Register(
		"pdf",
		encoder.Desc("Portable Document Format"),
		encoder.Encoder(encode),
	)
}

// pageSizes are the page geometries the encoder accepts, by name. The zero
// value is the default: one page the size of the image, which is what a
// picture wants and what a rendered document wants when it is short.
var pageSizes = map[string]blitz.PageSize{
	"fit":    {},
	"a4":     blitz.A4,
	"letter": blitz.Letter,
}

// PageSizes returns the page geometry names, for error messages.
func PageSizes() []string {
	return []string{"fit", "a4", "letter"}
}

// encode writes the image as a pdf.
func encode(ctx context.Context, w io.Writer, img image.Image) error {
	c := ivctx.Get(ctx)
	page, ok := pageSizes[strings.ToLower(strings.TrimSpace(c.PDFPage))]
	if !ok {
		return fmt.Errorf("pdf encode: unknown page size %q (available: %s)", c.PDFPage, strings.Join(PageSizes(), ", "))
	}
	opts := blitz.PDFOptions{
		Page:     page,
		MarginPt: float64(c.PDFMargin),
		// a named page size is only useful if what does not fit on one goes
		// onto the next; "fit" has nothing to paginate, being one page the
		// size of the image by definition
		Paginate: page != blitz.PageSize{},
	}
	rgba := toRGBA(img)
	ivctx.Logf(ctx, "pdf encode: %v page %+v paginate %t", rgba.Bounds().Size(), opts.Page, opts.Paginate)
	buf, err := blitz.EncodePDF(rgba, opts)
	if err != nil {
		return fmt.Errorf("pdf encode: %w", err)
	}
	_, err = w.Write(buf)
	return err
}

// toRGBA returns the image as a [image.RGBA], converting only when it is not
// one already -- every decoder that renders rather than decodes hands one
// back, so the common path copies nothing.
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}
