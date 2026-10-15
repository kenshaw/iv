// Package vips supplies a libvips decoder for iv, covering the image formats
// and pdfs that the Go decoders do not handle.
//
// See: https://github.com/cshum/vipsgen
package vips

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	ivvips "github.com/kenshaw/iv/vips"
	"golang.org/x/term"
)

// maxPasswordAttempts is how many times an encrypted pdf password is prompted
// for before giving up. The load that runs before the first prompt does not
// count against it: it carries whatever password was configured, so an
// unencrypted document never prompts at all.
const maxPasswordAttempts = 3

func init() {
	decoder.Register(
		"vips",
		decoder.Desc("libvips (raster images)"),
		// after the Go decoders, which handle the common formats without the
		// cost of starting vips
		decoder.After("png", "jpeg", "gif", "nativewebp", "tiff", "bmp", "ico", "icns", "netpbm", "resvg"),
		decoder.Extension(
			"avif", "heic", "heif", "jp2", "jpf", "jxl", "jxs", "j2k",
			"exr", "fits", "hdr", "mat", "nii", "pfm", "rad", "v",
		),
		decoder.MatcherContext(match),
		decoder.Decoder(decode),
	)
	decoder.Register(
		"vips-pdf",
		decoder.Desc("libvips (password protected pdf)"),
		decoder.Before("vips"),
		decoder.Extension("pdf"),
		decoder.MimeType("application/pdf"),
		decoder.Decoder(decodePDF),
	)
}

// match reports whether libvips should be tried for the mime type. It claims
// the image types that no more specific decoder registered, minus the ones
// libvips builds do not reliably support.
func match(_ context.Context, mime, _ string) bool {
	switch {
	// the psd decoder hands a file that it cannot read to vips by name
	case mime == "image/vnd.adobe.photoshop", mime == "image/x-photoshop",
		strings.HasPrefix(mime, "image/x-portable-"), // netpbm
		strings.Contains(mime, "jxr"):
		return false
	}
	return strings.HasPrefix(mime, "image/")
}

// loaders maps the mime types iv routes to libvips onto the load operation
// that handles them, for the formats a libvips build can be compiled without.
// Anything missing here is assumed to be supported.
var loaders = map[string]string{
	"image/avif":     "heifload_buffer",
	"image/heic":     "heifload_buffer",
	"image/heif":     "heifload_buffer",
	"image/jp2":      "jp2kload_buffer",
	"image/jpx":      "jp2kload_buffer",
	"image/jxl":      "jxlload_buffer",
	"image/x-exr":    "openexrload",
	"image/fits":     "fitsload",
	"image/x-matlab": "matload",
}

// supported reports whether this libvips build can load the mime type,
// wrapping [decoder.ErrUnsupportedFormat] when it cannot.
func supported(mime string) error {
	if op, ok := loaders[mime]; ok && !vips.HasOperation(op) {
		return fmt.Errorf("%s: %w", op, decoder.ErrUnsupportedFormat)
	}
	return nil
}

// decode decodes an image with libvips.
//
// The image is loaded from the bytes of the input and not from a stream of
// it. libvips decodes the pixels only when something asks for them, which can
// be after the pipeline closed the file, and the image keeps a reference to
// the bytes for as long as it lives.
func decode(ctx context.Context, r io.Reader) (any, error) {
	ivvips.Init(ctx)
	if err := supported(ivctx.Mime(ctx)); err != nil {
		return nil, fmt.Errorf("vips load: %w", err)
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// not every libvips loader accepts every load option -- the jxl loader
	// rejects `unlimited`, the jp2k loader rejects `n` -- so fall back to
	// progressively simpler options when one is refused
	for _, opts := range loadOptions(ivctx.Mime(ctx)) {
		var v *vips.Image
		if v, err = load(ctx, buf, opts); err == nil {
			return ivvips.Open(ctx, v)
		}
		if !isUnsupportedOption(err) {
			break
		}
		ivctx.Logf(ctx, "vips load: retrying with simpler options: %v", err)
	}
	return nil, fmt.Errorf("vips load: %w", err)
}

// loadOptions returns the load options to try, most capable first.
func loadOptions(mime string) []*vips.LoadOptions {
	return []*vips.LoadOptions{
		{
			N:           1,
			FailOnError: true,
			Unlimited:   mime != "image/jxl" && !strings.HasSuffix(mime, "/pdf"),
			Memory:      true,
		},
		{FailOnError: true, Memory: true},
		nil,
	}
}

// load loads the image, applying the configured page when the format is
// paged.
func load(ctx context.Context, buf []byte, opts *vips.LoadOptions) (*vips.Image, error) {
	if page := int(ivctx.Get(ctx).Page); page != 0 && opts != nil {
		// the page count is only known after a load, so load twice when a
		// specific page was requested
		v, err := vips.NewImageFromBuffer(buf, opts)
		if err != nil {
			return nil, err
		}
		if p := page - 1; 0 <= p && p < v.Pages() {
			opts.Page = p
		}
		v.Close()
	}
	return vips.NewImageFromBuffer(buf, opts)
}

// isUnsupportedOption reports whether the error is libvips refusing a load
// option that the format's loader does not have.
func isUnsupportedOption(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no property named")
}

// decodePDF decodes a pdf with libvips, prompting for a password when the
// document is encrypted. It loads from the bytes of the input, as [decode]
// does.
func decodePDF(ctx context.Context, r io.Reader) (any, error) {
	ivvips.Init(ctx)
	if !vips.HasOperation("pdfload_buffer") {
		return nil, fmt.Errorf("vips load: pdfload_buffer: %w", decoder.ErrUnsupportedFormat)
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	pass := ivctx.Get(ctx).Password
	for prompted := 0; ; prompted++ {
		v, err := loadPDF(ctx, buf, pass)
		switch {
		case err == nil:
			return ivvips.Open(ctx, v)
		case !ivvips.IsEncryptedErr(err):
			return nil, fmt.Errorf("vips load: %w", err)
		case prompted == maxPasswordAttempts:
			return nil, errors.New("vips load: invalid password")
		}
		p, err := readPassword()
		if err != nil {
			return nil, fmt.Errorf("vips load: %w", err)
		}
		pass = string(p)
	}
}

// loadPDF loads the pdf with the password, applying the configured page. The
// page count is only known after a load, so a specific page costs two.
func loadPDF(ctx context.Context, buf []byte, pass string) (*vips.Image, error) {
	opts := &vips.PdfloadBufferOptions{
		FailOn:   vips.FailOnError,
		Memory:   true,
		Password: pass,
	}
	if page := int(ivctx.Get(ctx).Page); page != 0 {
		v, err := vips.NewPdfloadBuffer(buf, opts)
		if err != nil {
			return nil, err
		}
		if p := page - 1; 0 <= p && p < v.Pages() {
			opts.Page = p
		}
		v.Close()
	}
	return vips.NewPdfloadBuffer(buf, opts)
}

// readPassword prompts for and reads a password from the terminal.
func readPassword() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("document is encrypted and stdin is not a terminal")
	}
	_, _ = fmt.Fprint(os.Stdout, "Password: ")
	pass, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(os.Stdout)
	return pass, err
}
