// Package poppler supplies a pdf decoder for iv, rendering a page to png with
// the `pdftoppm` command of poppler. It is the fallback for a libvips that is
// built without a pdf loader, such as the libvips of MSYS2.
//
// See: https://poppler.freedesktop.org
package poppler

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// pdftoppm is the raster renderer of poppler.
var pdftoppm = cmd.New("pdftoppm")

func init() {
	decoder.Register(
		"poppler",
		decoder.Desc("pdf documents (via pdftoppm)"),
		// after vips-pdf, which renders a pdf in process wherever libvips
		// has a pdf loader
		decoder.After("vips-pdf"),
		decoder.Extension("pdf"),
		decoder.MimeType("application/pdf"),
		decoder.Decoder(decode),
	)
}

// decode renders the configured page of the pdf at the configured dpi, with
// the configured password, and hands the png back to the pipeline. pdftoppm
// refuses a page that the document does not have, so a page that fails to
// render falls back to the first page.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanupSrc, err := cmd.Source(ctx, r, "pdf")
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()
	page := max(int(ivctx.Get(ctx).Page), 1)
	buf, err := render(ctx, src, page)
	if err != nil && page != 1 {
		ivctx.Logf(ctx, "pdftoppm: page %d: %v", page, err)
		buf, err = render(ctx, src, 1)
	}
	if err != nil {
		return nil, err
	}
	return decoder.NewBytes("image/png", buf).WithExt("png"), nil
}

// render renders one page of the pdf to png, and returns the png.
func render(ctx context.Context, src string, page int) ([]byte, error) {
	dir, cleanup, err := cmd.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	c := ivctx.Get(ctx)
	params := []string{"-png", "-singlefile", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page)}
	if c.DPI != 0 {
		params = append(params, "-r", strconv.Itoa(int(c.DPI)))
	}
	if c.Password != "" {
		params = append(params, "-upw", c.Password)
	}
	out := filepath.Join(dir, "page")
	if _, err := pdftoppm.CombinedOutput(ctx, append(params, src, out)...); err != nil {
		return nil, err
	}
	// -singlefile names the file for the prefix alone
	return os.ReadFile(out + ".png")
}
