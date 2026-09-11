// Package vips supplies libvips backed encoders for iv, covering the output
// formats the Go encoders do not handle.
//
// See: https://github.com/cshum/vipsgen
package vips

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/encoder"
	ivvips "github.com/kenshaw/iv/vips"
)

// saveFunc encodes a vips image in a specific format.
//
// The buffer variants are used rather than the target ones: several libvips
// savers seek within their output, which an arbitrary [io.Writer] cannot do.
type saveFunc func(*vips.Image) ([]byte, error)

// format describes a libvips backed encoder.
type format struct {
	name string
	desc string
	ext  string
	save saveFunc
}

// formats are the libvips encoders registered by this package.
var formats = []format{
	{"vips-avif", "AV1 Image File Format", "avif", func(v *vips.Image) ([]byte, error) {
		return v.HeifsaveBuffer(&vips.HeifsaveBufferOptions{Compression: vips.HeifCompressionAv1})
	}},
	{"vips-heif", "High Efficiency Image Format", "heic", func(v *vips.Image) ([]byte, error) {
		return v.HeifsaveBuffer(nil)
	}},
	{"vips-jxl", "JPEG XL", "jxl", func(v *vips.Image) ([]byte, error) {
		return v.JxlsaveBuffer(nil)
	}},
	{"vips-jp2k", "JPEG 2000", "jp2", func(v *vips.Image) ([]byte, error) {
		return v.Jp2ksaveBuffer(nil)
	}},
	{"vips-tiff", "Tag Image File Format", "tiff", func(v *vips.Image) ([]byte, error) {
		return v.TiffsaveBuffer(nil)
	}},
	{"vips-webp", "Web Picture Format", "webp", func(v *vips.Image) ([]byte, error) {
		return v.WebpsaveBuffer(nil)
	}},
	{"vips-gif", "Graphics Interchange Format", "gif", func(v *vips.Image) ([]byte, error) {
		return v.GifsaveBuffer(nil)
	}},
}

func init() {
	for _, f := range formats {
		encoder.Register(
			f.name,
			encoder.Desc(f.desc+" (via libvips)"),
			encoder.Extension(f.ext),
			encoder.Encoder(encodeFunc(f.save)),
		)
	}
}

// encodeFunc builds an encoder that round trips the Go image through libvips.
func encodeFunc(save saveFunc) encoder.EncodeFunc {
	return func(ctx context.Context, w io.Writer, img image.Image) error {
		ivvips.Init(ctx)
		// libvips reads from a source, so hand it a lossless png of the image
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("vips encode: %w", err)
		}
		v, err := vips.NewImageFromBuffer(buf.Bytes(), nil)
		if err != nil {
			return fmt.Errorf("vips encode: load: %w", err)
		}
		out, err := save(v)
		if err != nil {
			return fmt.Errorf("vips encode: save: %w", err)
		}
		_, err = w.Write(out)
		return err
	}
}
