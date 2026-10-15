// Package calibre supplies an ebook decoder for iv, converting books to pdf
// with the `ebook-convert` command of calibre.
//
// See: https://manual.calibre-ebook.com/generated/en/ebook-convert.html
package calibre

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// ebookConvert is the conversion command of calibre.
var ebookConvert = cmd.New("ebook-convert")

func init() {
	decoder.Register(
		"calibre",
		decoder.Desc("Ebooks (via ebook-convert)"),
		decoder.Extension("epub", "mobi", "azw", "azw3", "fb2"),
		decoder.MimeType(
			"application/epub+zip",
			"application/x-mobipocket-ebook",
			"application/vnd.amazon.ebook",
			"text/fb2+xml",
		),
		// an epub can sniff as a plain zip, and an fb2 as plain xml, so the
		// extension decides between them and the other zip and xml formats
		decoder.MimeTypeExtensionMatch(
			"application/zip", "epub",
			"text/xml", "fb2",
			"application/xml", "fb2",
		),
		decoder.Decoder(decode),
	)
}

// exts maps a mime type to the extension that ebook-convert reads the format
// by, for an input whose name has none.
var exts = map[string]string{
	"application/epub+zip":           "epub",
	"application/x-mobipocket-ebook": "mobi",
	"application/vnd.amazon.ebook":   "azw",
	"text/fb2+xml":                   "fb2",
}

// decode converts the book to a pdf, and hands the pdf back to the pipeline.
// The page of the pdf that the configuration names is the page displayed.
//
// ebook-convert reads the format of its input from the extension of the file
// name, so a book that has no extension is spooled with the one that its mime
// type names.
func decode(ctx context.Context, r io.Reader) (any, error) {
	ext := ivctx.FileExt(ivctx.PathName(ctx))
	if ext == "" {
		ext = exts[ivctx.Mime(ctx)]
	}
	src, cleanupSrc, err := cmd.Source(ctx, r, ext)
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()
	dir, cleanupDir, err := cmd.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanupDir()
	pdfName := filepath.Join(dir, "book.pdf")
	if _, err := ebookConvert.CombinedOutput(ctx, src, pdfName); err != nil {
		return nil, err
	}
	buf, err := os.ReadFile(pdfName)
	if err != nil {
		return nil, fmt.Errorf("ebook-convert produced no pdf: %w", err)
	}
	ivctx.Logf(ctx, "ebook-convert: %d bytes of pdf", len(buf))
	return decoder.NewBytes("application/pdf", buf).WithExt("pdf"), nil
}
