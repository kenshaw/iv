// Package ivy supplies the document, ebook and page description decoders for
// iv, with xo/ivy, in pure Go (D49).
//
// Each format of ivy is a decoder of its own, named ivy- and the name of the
// format, such as ivy-docx. A format with pages draws the page that --page
// selects, at the size that it is shown at. A format with no pages draws its
// best preview, such as the thumbnail of a Visio drawing.
//
// A spreadsheet whose file gives no orientation is drawn in landscape, on
// US Letter: a csv or tsv table, an Excel workbook, or an OpenDocument
// spreadsheet. A sheet whose file gives portrait stays in portrait (D51).
//
// When the output is a pdf, a format with pages writes its pages with the
// PDF writer of ivy, with their paths and their images as vectors. It writes
// every page, or the page that --page selects (D55).
//
// A document that a password protects, such as a pdf or a docx, opens with
// --password. When that password is missing or wrong, the decoder asks for
// the password at the terminal (D50).
//
// See: https://github.com/xo/ivy
package ivy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log/slog"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/ivy"
	"github.com/xo/ivy/camera"
	"github.com/xo/ivy/chm"
	"github.com/xo/ivy/csv"
	"github.com/xo/ivy/djvu"
	"github.com/xo/ivy/docx"
	"github.com/xo/ivy/emf"
	"github.com/xo/ivy/epub"
	"github.com/xo/ivy/fb2"
	"github.com/xo/ivy/iwork"
	"github.com/xo/ivy/mobi"
	"github.com/xo/ivy/msdoc"
	"github.com/xo/ivy/msppt"
	"github.com/xo/ivy/msxls"
	"github.com/xo/ivy/odf"
	"github.com/xo/ivy/ora"
	"github.com/xo/ivy/pdf"
	"github.com/xo/ivy/pdf/pdfenc"
	"github.com/xo/ivy/pptx"
	"github.com/xo/ivy/ps"
	"github.com/xo/ivy/psd"
	"github.com/xo/ivy/pub"
	"github.com/xo/ivy/rtf"
	"github.com/xo/ivy/vsdx"
	"github.com/xo/ivy/xlsx"
	"github.com/xo/ivy/xps"
	"golang.org/x/term"
)

// formats are the formats of ivy that iv decodes, each with its description.
// blitz draws HTML, Markdown and text, and the archives decoder reads the
// comic archives of ivy and also RAR, so iv does not take those formats of
// ivy.
var formats = []struct {
	format ivy.Format
	desc   string
}{
	{camera.Format, "Camera RAW previews"},
	{chm.Format, "Compiled HTML Help"},
	{csv.Format, "CSV and TSV tables"},
	{djvu.Format, "DjVu documents"},
	{docx.Format, "Word documents"},
	{emf.Format, "EMF and WMF metafiles"},
	{epub.Format, "EPUB ebooks"},
	{fb2.Format, "FictionBook ebooks"},
	{iwork.Format, "Pages, Keynote and Numbers previews"},
	{mobi.Format, "MOBI and Kindle ebooks"},
	{msdoc.Format, "Word 97 to 2003 documents"},
	{msppt.Format, "PowerPoint 97 to 2003 presentations"},
	{msxls.Format, "Excel 97 to 2003 spreadsheets"},
	{odf.Format, "OpenDocument documents"},
	{ora.Format, "OpenRaster and Krita images"},
	{pdf.Format, "PDF documents"},
	{pptx.Format, "PowerPoint presentations"},
	{ps.Format, "EPS and PostScript previews"},
	{psd.Format, "Adobe Photoshop"},
	{pub.Format, "Publisher previews"},
	{rtf.Format, "RTF documents"},
	{vsdx.Format, "Visio previews"},
	{xlsx.Format, "Excel spreadsheets"},
	{xps.Format, "XPS documents"},
}

// rasters are the formats whose pages are raster images, at one point for
// each pixel. A page of one of them draws at 72 dots per inch, at its own
// size, with its transparent pixels, and iv scales it as any other image.
var rasters = []string{"camera", "ora", "psd"}

// generic are the mime types that libmagic gives to a file that it cannot
// name more closely. A decoder that registered one would claim every file
// of that type, so a format takes such a file by its extension only: the
// pipeline routes a file of a generic type by its extension.
var generic = []string{
	"text/plain",
	"text/csv",
	"text/tab-separated-values",
	"text/xml",
	"application/xml",
	"application/zip",
	"application/octet-stream",
}

// containers are the mime types that libmagic gives to the compound files of
// Office 97 to 2003, and the formats with one of them are routed by their
// extension.
var containers = []string{
	"application/x-ole-storage",
	"application/CDFV2",
	"application/vnd.ms-office",
}

// cache shares the fonts and the ICC profiles of the documents.
var cache = ivy.NewCache()

func init() {
	for _, f := range formats {
		mimes := slices.DeleteFunc(slices.Clone(f.format.Mimes), func(m string) bool {
			return slices.Contains(generic, m)
		})
		exts := f.format.Exts
		opts := []decoder.Option{
			decoder.Desc(f.desc + " (ivy)"),
			decoder.Extension(exts...),
			decoder.Matcher(func(mime, ext string) bool {
				return slices.Contains(containers, mime) && slices.Contains(exts, ext)
			}),
			// libvips also reads a psd, and is tried after ivy
			decoder.Before("vips"),
			decoder.Decoder(decodeFunc(f.format)),
		}
		if len(mimes) != 0 {
			opts = append(opts, decoder.MimeType(mimes...))
		}
		if f.format.Name == "csv" {
			// libmagic gives text/csv to any text whose lines have the same
			// number of fields, which markdown often has, so a file with
			// another extension is not a table. A table with no extension,
			// as from a pipe, is
			opts = append(opts, decoder.Matcher(func(mime, ext string) bool {
				return ext == "" && (mime == "text/csv" || mime == "text/tab-separated-values")
			}))
		}
		decoder.Register("ivy-"+f.format.Name, opts...)
	}
}

// decodeFunc returns the decode function of the format.
func decodeFunc(f ivy.Format) decoder.DecodeFunc {
	return func(ctx context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		c := ivctx.Get(ctx)
		cfg := ivy.Config{
			Password:     c.Password,
			PasswordFunc: askPassword,
			Logger:       slog.New(slog.NewTextHandler(logWriter{c.Logger}, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: noTime})),
			Cache:        cache,
			// an ebook or a document lays out its text in the fonts that it
			// embeds, then in the fonts of the system
			SystemFonts: true,
			Paper:       paper(f, buf),
		}
		doc, err := f.Open(ctx, bytes.NewReader(buf), cfg)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		describe(ctx, f, doc)
		pager, ok := doc.(ivy.Pager)
		if !ok {
			defer doc.Close()
			img, err := ivy.BestPreview(ctx, doc, displaySize(ctx))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, err)
			}
			return img, nil
		}
		img, n, err := renderPage(ctx, f, pager)
		if err != nil || c.Encoder != "pdf" {
			return img, errors.Join(err, doc.Close())
		}
		// the pdf encoder writes the pages from the document, which stays
		// open until the image is closed
		return &document{Image: img, f: f, doc: doc, cfg: cfg, page: n}, nil
	}
}

// document is a page of a document, drawn as an image, that writes the
// pages of its document as a pdf, as vectors (D55). Close closes the
// document.
type document struct {
	image.Image

	f    ivy.Format
	doc  ivy.Document
	cfg  ivy.Config
	page int
}

// Unwrap returns the image of the page.
func (d *document) Unwrap() image.Image {
	return d.Image
}

// Close satisfies the [io.Closer] interface.
func (d *document) Close() error {
	return d.doc.Close()
}

// WritePDF writes the pages of the document to w as a pdf. It writes every
// page, or the page that --page selects. A page that draws only in part is
// written with the part, and a warning.
func (d *document) WritePDF(ctx context.Context, w io.Writer) error {
	opts := []pdfenc.Option{pdfenc.WithConfig(d.cfg)}
	if ivctx.Get(ctx).Page != 0 {
		opts = append(opts, pdfenc.WithPages(d.page))
	}
	err := pdfenc.Encode(ctx, w, d.doc, opts...)
	var partial *ivy.PartialError
	switch {
	case errors.As(err, &partial):
		ivctx.Warnf(ctx, "%s: drawn in part: %v", d.f.Name, partial)
	case err != nil:
		return fmt.Errorf("%s: %w", d.f.Name, err)
	}
	return nil
}

// sheets are the formats that hold only spreadsheets. The odf format holds
// spreadsheets and other documents.
var sheets = []string{"csv", "msxls", "xlsx"}

// odsMime starts the mime type of an OpenDocument spreadsheet, and of its
// template.
const odsMime = "application/vnd.oasis.opendocument.spreadsheet"

// paper returns the paper for a document whose file gives none. A
// spreadsheet is drawn in landscape, because a table is usually wider than
// it is tall. Any other document keeps the paper of ivy, US Letter in
// portrait (D51).
func paper(f ivy.Format, buf []byte) ivy.Paper {
	if slices.Contains(sheets, f.Name) || (f.Name == "odf" && isODS(buf)) {
		return ivy.Letter.Landscape()
	}
	return ivy.Paper{}
}

// isODS reports whether buf is an OpenDocument spreadsheet. The package of
// an OpenDocument file is a zip whose first entry is mimetype, which is not
// compressed. The local header of an entry is 30 bytes, then the name, the
// extra field and the data.
func isODS(buf []byte) bool {
	if len(buf) < 30 || !bytes.HasPrefix(buf, []byte("PK\x03\x04")) {
		return false
	}
	n := 30 + int(binary.LittleEndian.Uint16(buf[26:]))
	if len(buf) < n || string(buf[30:n]) != "mimetype" {
		return false
	}
	data := buf[min(len(buf), n+int(binary.LittleEndian.Uint16(buf[28:]))):]
	return bytes.HasPrefix(data, []byte(odsMime))
}

// renderPage draws the page that --page selects, and returns its number,
// from 0.
//
// A page of a raster image draws at its own size. Any other page draws at
// --dpi, or at a lower resolution when that is larger than the size that the
// page is shown at, so that it is drawn at the size that it is shown at and
// is not scaled after.
func renderPage(ctx context.Context, f ivy.Format, pager ivy.Pager) (image.Image, int, error) {
	count, err := pager.PageCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", f.Name, err)
	}
	if count == 0 {
		return nil, 0, fmt.Errorf("%s: %w", f.Name, ErrNoPages)
	}
	n := ivctx.Page(ctx, count)
	page, err := pager.Page(ctx, n)
	if err != nil {
		return nil, 0, fmt.Errorf("%s page %d: %w", f.Name, n+1, err)
	}
	// a page of a document is paper, and is white under its content. A
	// raster image keeps its transparent pixels, as any other image does, so
	// that --bg applies to it
	dpi, bg := float32(72), color.Color(nil)
	if !slices.Contains(rasters, f.Name) {
		dpi, bg = pageDPI(ctx, page), color.White
	}
	ivctx.Logf(ctx, "%s: page %d of %d at %g dpi", f.Name, n+1, count, dpi)
	img, err := ivy.RenderImage(ctx, page, dpi, bg)
	var partial *ivy.PartialError
	switch {
	case errors.As(err, &partial):
		ivctx.Warnf(ctx, "%s page %d: drawn in part: %v", f.Name, n+1, partial.Err)
	case err != nil:
		return nil, 0, fmt.Errorf("%s page %d: %w", f.Name, n+1, err)
	}
	return img, n, nil
}

// ErrNoPages is the error of a document with no pages.
var ErrNoPages = errors.New("no pages")

// maxPasswordAttempts is the number of times that the decoder asks for the
// password of a document. The password of --password is tried first, and is
// not one of them.
const maxPasswordAttempts = 3

// ErrPasswordAttempts is the error when no password that the decoder asked
// for opens the document.
var ErrPasswordAttempts = errors.New("too many wrong passwords")

// ErrNoTerminal is the error when the decoder must ask for a password, and
// stdin is not a terminal.
var ErrNoTerminal = errors.New("stdin is not a terminal, so --password must give the password")

// readPassword reads a password. A test replaces it.
var readPassword = terminalPassword

// askPassword asks for the password of a document, at most
// maxPasswordAttempts times. ivy calls it while the password is missing or
// wrong.
func askPassword(_ context.Context, attempt int) (string, error) {
	if attempt > maxPasswordAttempts {
		return "", fmt.Errorf("%d attempts: %w", maxPasswordAttempts, ErrPasswordAttempts)
	}
	return readPassword()
}

// terminalPassword reads a password at the terminal, and does not show it.
func terminalPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", ErrNoTerminal
	}
	_, _ = fmt.Fprint(os.Stdout, "Password: ")
	buf, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(os.Stdout)
	if err != nil {
		return "", fmt.Errorf("password: %w", err)
	}
	return string(buf), nil
}

// pageDPI returns the resolution that draws the page at the size that it is
// shown at, and at most at --dpi.
func pageDPI(ctx context.Context, page ivy.Page) float32 {
	dpi := float32(ivctx.Get(ctx).DPI)
	if dpi <= 0 {
		dpi = 300
	}
	w, h := page.Size()
	if w <= 0 || h <= 0 {
		return dpi
	}
	pw, ph := int(math.Ceil(float64(w*dpi/72))), int(math.Ceil(float64(h*dpi/72)))
	dw, dh := ivctx.Scale(ctx, pw, ph)
	if dw >= pw && dh >= ph {
		return dpi
	}
	return dpi * min(float32(dw)/float32(pw), float32(dh)/float32(ph))
}

// displaySize returns the size that the display gives, as the size of the
// preview to choose, or zero for the largest preview.
func displaySize(ctx context.Context) image.Point {
	c := ivctx.Get(ctx)
	return image.Pt(int(c.Width), int(c.Height))
}

// describe logs what the document says about itself.
func describe(ctx context.Context, f ivy.Format, doc ivy.Document) {
	d, ok := doc.(ivy.Describer)
	if !ok {
		return
	}
	info, err := d.Describe(ctx)
	if err != nil {
		ivctx.Logf(ctx, "%s: describe: %v", f.Name, err)
		return
	}
	ivctx.Logf(ctx, "%s: title %q, authors %q, producer %q", f.Name, info.Title, strings.Join(info.Authors, ", "), info.Producer)
}

// logWriter writes each line of the log of ivy to the log of iv.
type logWriter struct {
	logf func(string, ...any)
}

// Write satisfies the [io.Writer] interface.
func (w logWriter) Write(p []byte) (int, error) {
	for line := range strings.SplitSeq(strings.TrimSpace(string(p)), "\n") {
		w.logf("ivy: %s", line)
	}
	return len(p), nil
}

// noTime drops the time from a line of the log of ivy, which the log of iv
// does not have.
func noTime(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
