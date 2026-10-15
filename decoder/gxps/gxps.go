// Package gxps supplies an XPS decoder for iv, rendering a page to png with
// the `xpstopng` command of libgxps.
//
// See: https://gitlab.gnome.org/GNOME/libgxps
package gxps

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// xpstopng is the png renderer of libgxps.
var xpstopng = cmd.New("xpstopng")

func init() {
	decoder.Register(
		"gxps",
		decoder.Desc("XPS documents (via xpstopng)"),
		decoder.Extension("xps", "oxps"),
		decoder.MimeType(
			"application/oxps",
			"application/vnd.ms-xpsdocument",
		),
		// an xps document is a zip, and only the extension tells it from
		// the other zip formats
		decoder.MimeTypeExtensionMatch(
			"application/zip", "xps",
			"application/zip", "oxps",
		),
		decoder.Decoder(decode),
	)
}

// decode renders the configured page of the document at the configured dpi,
// and hands the png back to the pipeline.
//
// xpstopng refuses a page that the document does not have, and the number of
// pages is not known before a render, so a page that fails to render falls
// back to the first page, as a page out of range does for the other paged
// formats.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanupSrc, err := cmd.Source(ctx, r, "xps")
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()
	page := max(int(ivctx.Get(ctx).Page), 1)
	buf, err := render(ctx, src, page)
	if err != nil && page != 1 {
		ivctx.Logf(ctx, "xpstopng: page %d: %v", page, err)
		buf, err = render(ctx, src, 1)
	}
	if err != nil {
		return nil, err
	}
	return decoder.NewBytes("image/png", buf).WithExt("png"), nil
}

// render renders one page of the document to png, and returns the png.
func render(ctx context.Context, src string, page int) ([]byte, error) {
	dir, cleanup, err := cmd.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	params := []string{"-f", strconv.Itoa(page), "-l", strconv.Itoa(page)}
	if dpi := ivctx.Get(ctx).DPI; dpi != 0 {
		params = append(params, "-r", strconv.Itoa(int(dpi)))
	}
	prefix := filepath.Join(dir, "page")
	if _, err := xpstopng.CombinedOutput(ctx, append(params, src, prefix)...); err != nil {
		return nil, err
	}
	// xpstopng names the file for the prefix and the page number
	names, err := filepath.Glob(prefix + "-*.png")
	switch {
	case err != nil:
		return nil, err
	case len(names) != 1:
		return nil, fmt.Errorf("xpstopng: expected one png for page %d, got %d", page, len(names))
	}
	return os.ReadFile(names[0])
}
