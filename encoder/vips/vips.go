// Package vips supplies libvips backed encoders for iv, covering the output
// formats the Go encoders do not handle.
//
// See: https://github.com/cshum/vipsgen
package vips

import (
	"context"
	"fmt"
	"image"
	"io"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/encoder"
	ivvips "github.com/kenshaw/iv/vips"
)

// saveFunc encodes a vips image in a specific format.
//
// The buffer variants are used rather than the target ones: several libvips
// savers seek within their output, which an arbitrary [io.Writer] cannot do.
type saveFunc = ivvips.SaveFunc

// format describes a libvips backed encoder.
type format struct {
	name string
	desc string
	ext  string
	// op is the libvips operation the saver dispatches to. Savers are
	// conditionally compiled into libvips, so it may be absent at runtime.
	op   string
	save saveFunc
}

// formats are the libvips encoders registered by this package.
var formats = []format{
	{"vips-avif", "AV1 Image File Format", "avif", "heifsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.HeifsaveBuffer(&vips.HeifsaveBufferOptions{Compression: vips.HeifCompressionAv1})
	}},
	{"vips-heif", "High Efficiency Image Format", "heic", "heifsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.HeifsaveBuffer(nil)
	}},
	{"vips-jxl", "JPEG XL", "jxl", "jxlsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.JxlsaveBuffer(nil)
	}},
	{"vips-jp2k", "JPEG 2000", "jp2", "jp2ksave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.Jp2ksaveBuffer(nil)
	}},
	{"vips-tiff", "Tag Image File Format", "tiff", "tiffsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.TiffsaveBuffer(nil)
	}},
	{"vips-webp", "Web Picture Format", "webp", "webpsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.WebpsaveBuffer(nil)
	}},
	{"vips-gif", "Graphics Interchange Format", "gif", "gifsave_buffer", func(v *vips.Image) ([]byte, error) {
		return v.GifsaveBuffer(nil)
	}},
}

func init() {
	for _, f := range formats {
		encoder.Register(
			f.name,
			encoder.Desc(f.desc+" (via libvips)"),
			encoder.Extension(f.ext),
			encoder.Encoder(encodeFunc(f.op, f.save)),
		)
	}
}

// encodeFunc builds an encoder that saves the Go image through libvips.
func encodeFunc(op string, save saveFunc) encoder.EncodeFunc {
	return func(ctx context.Context, w io.Writer, img image.Image) error {
		ivvips.Init(ctx)
		if !vips.HasOperation(op) {
			return fmt.Errorf("vips encode: %s: %w", op, encoder.ErrUnsupportedFormat)
		}
		out, err := ivvips.Save(ctx, img, save)
		if err != nil {
			return fmt.Errorf("vips encode: %w", err)
		}
		_, err = w.Write(out)
		return err
	}
}
