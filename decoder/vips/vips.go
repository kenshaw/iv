// Package vips supplies a libvips decoder for iv, covering the image formats
// and pdfs that the Go decoders do not handle.
//
// See: https://github.com/cshum/vipsgen
package vips

import (
	"context"
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
// for before giving up.
const maxPasswordAttempts = 3

func init() {
	decoder.Register(
		"vips",
		decoder.Desc("libvips (raster images, pdf)"),
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
		decoder.Decoder(DecodePdf),
	)
}

// match reports whether libvips should be tried for the mime type. It claims
// the image types that no more specific decoder registered, minus the ones
// libvips builds do not reliably support.
func match(_ context.Context, mime, _ string) bool {
	switch {
	case mime == "image/vnd.adobe.photoshop", // fitz handles psd
		strings.HasPrefix(mime, "image/x-portable-"), // netpbm
		strings.Contains(mime, "jxr"):
		return false
	}
	return strings.HasPrefix(mime, "image/")
}

// decode decodes an image with libvips.
func decode(ctx context.Context, r io.Reader) (any, error) {
	ivvips.Init(ctx)
	var err error
	// not every libvips loader accepts every load option -- the jxl loader
	// rejects `unlimited`, the jp2k loader rejects `n` -- so fall back to
	// progressively simpler options when one is refused
	for _, opts := range loadOptions(ivctx.Mime(ctx)) {
		var v *vips.Image
		if v, err = load(ctx, r, opts); err == nil {
			return ivvips.Export(ctx, v)
		}
		if !isUnsupportedOption(err) {
			break
		}
		ivctx.Logf(ctx, "vips load: retrying with simpler options: %v", err)
		if err := rewind(r); err != nil {
			return nil, err
		}
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
func load(ctx context.Context, r io.Reader, opts *vips.LoadOptions) (*vips.Image, error) {
	if page := int(ivctx.Get(ctx).Page); page != 0 && opts != nil {
		// the page count is only known after a load, so load twice when a
		// specific page was requested
		v, err := vips.NewImageFromSource(vips.NewSource(io.NopCloser(r)), opts)
		if err != nil {
			return nil, err
		}
		if p := page - 1; 0 <= p && p < v.Pages() {
			opts.Page = p
		}
		if err := rewind(r); err != nil {
			return nil, err
		}
	}
	return vips.NewImageFromSource(vips.NewSource(io.NopCloser(r)), opts)
}

// isUnsupportedOption reports whether the error is libvips refusing a load
// option that the format's loader does not have.
func isUnsupportedOption(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no property named")
}

// DecodePdf decodes a pdf with libvips, prompting for a password when the
// document is encrypted.
func DecodePdf(ctx context.Context, r io.Reader) (any, error) {
	ivvips.Init(ctx)
	var pass []byte
	for i := range maxPasswordAttempts {
		opts := &vips.PdfloadSourceOptions{
			FailOn:   vips.FailOnError,
			Memory:   true,
			Password: string(pass),
		}
		if page := int(ivctx.Get(ctx).Page); page != 0 {
			switch v, err := vips.NewPdfloadSource(vips.NewSource(io.NopCloser(r)), opts); {
			case ivvips.IsEncryptedErr(err):
			case err != nil:
				return nil, fmt.Errorf("vips load: %w", err)
			default:
				if p := page - 1; 0 <= p && p < v.Pages() {
					opts.Page = p
				}
			}
			if err := rewind(r); err != nil {
				return nil, err
			}
		}
		v, err := vips.NewPdfloadSource(vips.NewSource(io.NopCloser(r)), opts)
		switch {
		case err == nil:
			return ivvips.Export(ctx, v)
		case !ivvips.IsEncryptedErr(err):
			return nil, fmt.Errorf("vips load: %w", err)
		case i == maxPasswordAttempts-1:
			return nil, fmt.Errorf("vips load: invalid password")
		}
		if pass, err = readPassword(); err != nil {
			return nil, fmt.Errorf("vips load: %w", err)
		}
		if err := rewind(r); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("vips load: invalid password")
}

// readPassword prompts for and reads a password from the terminal.
func readPassword() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("document is encrypted and stdin is not a terminal")
	}
	_, _ = fmt.Fprint(os.Stdout, "Password: ")
	pass, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(os.Stdout)
	return pass, err
}

// rewind seeks the reader back to its start. The pipeline always hands
// decoders a seekable stream.
func rewind(r io.Reader) error {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("%T: not seekable", r)
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	return nil
}
