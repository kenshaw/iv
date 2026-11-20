// Package vipsfile supplies a decoder for iv for the file format of libvips,
// in Go.
//
// A file is a header of 64 bytes, then the pixels, uncompressed, with the
// bands of each pixel side by side, then the metadata in XML. The first 4
// bytes are the magic number, which also gives the byte order of the rest.
//
// See: https://www.libvips.org/API/current/file-format.html
package vipsfile

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"math"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/scrgb"
)

func init() {
	decoder.Register(
		"vipsfile",
		decoder.Desc("libvips image file"),
		decoder.Extension("v", "vips"),
		decoder.Decoder(decode),
	)
}

// The magic numbers of a file, as its first 4 bytes. The rest of the file is
// in little endian order after the first, and in big endian order after the
// second.
var (
	magicIntel = []byte{0xb6, 0xa6, 0xf2, 0x08}
	magicSparc = []byte{0x08, 0xf2, 0xa6, 0xb6}
)

// headerSize is the size of the header, in bytes.
const headerSize = 64

// The band formats of libvips that the decoder reads.
const (
	formatUchar  = 0
	formatUshort = 2
	formatFloat  = 6
	formatDouble = 8
)

// The interpretations of libvips that the decoder reads.
const (
	interpMultiband = 0
	interpBW        = 1
	interpRGB       = 17
	interpSRGB      = 22
	interpRGB16     = 25
	interpGrey16    = 26
	interpSCRGB     = 28
)

// ErrFormat is the error of a file that the decoder does not read, such as
// a CMYK image, a Lab image or a coded image.
var ErrFormat = errors.New("format not supported")

// header is the header of a file, with the byte order of the rest.
type header struct {
	order  binary.ByteOrder
	w, h   int
	bands  int
	format uint32
	interp uint32
	// size is the size of a sample, in bytes
	size int
}

// readHeader reads and checks the header at the start of buf.
func readHeader(buf []byte) (*header, error) {
	if len(buf) < headerSize {
		return nil, io.ErrUnexpectedEOF
	}
	hdr := new(header)
	switch {
	case bytes.HasPrefix(buf, magicIntel):
		hdr.order = binary.LittleEndian
	case bytes.HasPrefix(buf, magicSparc):
		hdr.order = binary.BigEndian
	default:
		return nil, decoder.ErrNotSupported
	}
	field := func(off int) uint32 {
		return hdr.order.Uint32(buf[off:])
	}
	// a field is a signed int of 32 bits, and a negative one is too large
	// as unsigned, so the bounds reject it
	w, h, bands := field(4), field(8), field(12)
	hdr.format, hdr.interp = field(20), field(28)
	switch hdr.format {
	case formatUchar:
		hdr.size = 1
	case formatUshort:
		hdr.size = 2
	case formatFloat:
		hdr.size = 4
	case formatDouble:
		hdr.size = 8
	}
	switch {
	case w == 0 || h == 0 || w > 1<<20 || h > 1<<20:
		return nil, fmt.Errorf("size %dx%d: %w", w, h, ErrFormat)
	case bands < 1 || bands > 4:
		return nil, fmt.Errorf("%d bands: %w", bands, ErrFormat)
	case hdr.size == 0:
		return nil, fmt.Errorf("band format %d: %w", hdr.format, ErrFormat)
	case field(24) != 0:
		return nil, fmt.Errorf("coding %d: %w", field(24), ErrFormat)
	}
	switch hdr.interp {
	case interpMultiband, interpBW, interpRGB, interpSRGB, interpRGB16, interpGrey16, interpSCRGB:
	default:
		return nil, fmt.Errorf("interpretation %d: %w", hdr.interp, ErrFormat)
	}
	hdr.w, hdr.h, hdr.bands = int(w), int(h), int(bands)
	return hdr, nil
}

// sample returns band b of the pixel at x, y of pix, from 0 to 1. A float
// image of scRGB is linear, and any other float image holds the range of 8
// bits.
func (hdr *header) sample(pix []byte, x, y, b int) float64 {
	off := ((y*hdr.w+x)*hdr.bands + b) * hdr.size
	var v float64
	switch hdr.format {
	case formatUchar:
		return float64(pix[off]) / math.MaxUint8
	case formatUshort:
		return float64(hdr.order.Uint16(pix[off:])) / math.MaxUint16
	case formatFloat:
		v = float64(math.Float32frombits(hdr.order.Uint32(pix[off:])))
	default:
		v = math.Float64frombits(hdr.order.Uint64(pix[off:]))
	}
	if hdr.interp != interpSCRGB {
		v /= math.MaxUint8
	}
	return v
}

// color returns the red, green, blue and alpha of the pixel at x, y of pix.
func (hdr *header) color(pix []byte, x, y int) (float64, float64, float64, float64) {
	switch hdr.bands {
	case 1:
		v := hdr.sample(pix, x, y, 0)
		return v, v, v, 1
	case 2:
		v := hdr.sample(pix, x, y, 0)
		return v, v, v, hdr.sample(pix, x, y, 1)
	case 3:
		return hdr.sample(pix, x, y, 0), hdr.sample(pix, x, y, 1), hdr.sample(pix, x, y, 2), 1
	}
	return hdr.sample(pix, x, y, 0), hdr.sample(pix, x, y, 1), hdr.sample(pix, x, y, 2), hdr.sample(pix, x, y, 3)
}

// decode decodes a file of libvips.
func decode(_ context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	hdr, err := readHeader(buf)
	if err != nil {
		return nil, fmt.Errorf("vipsfile: %w", err)
	}
	pix := buf[headerSize:]
	if n := hdr.w * hdr.h * hdr.bands * hdr.size; len(pix) < n {
		return nil, fmt.Errorf("vipsfile: %d bytes of pixels, expected %d: %w", len(pix), n, io.ErrUnexpectedEOF)
	}
	color := func(x, y int) (float64, float64, float64, float64) {
		return hdr.color(pix, x, y)
	}
	if hdr.interp == interpSCRGB {
		return scrgb.NRGBA64(hdr.w, hdr.h, color), nil
	}
	img := image.NewNRGBA64(image.Rect(0, 0, hdr.w, hdr.h))
	for y := range hdr.h {
		for x := range hdr.w {
			r, g, b, a := color(x, y)
			i := img.PixOffset(x, y)
			for c, v := range [4]float64{r, g, b, a} {
				binary.BigEndian.PutUint16(img.Pix[i+2*c:], scrgb.Quantize(v))
			}
		}
	}
	return img, nil
}
