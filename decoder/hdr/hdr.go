// Package hdr supplies a Radiance HDR and Portable Float Map decoder for iv,
// in pure Go.
//
// See: https://github.com/mdouchement/hdr
package hdr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/scrgb"
	"github.com/mdouchement/hdr"
	"github.com/mdouchement/hdr/codec/pfm"
	"github.com/mdouchement/hdr/codec/rgbe"
)

func init() {
	decoder.RegisterBuiltin(
		"hdr",
		decoder.Desc("Radiance HDR and Portable Float Map"),
		decoder.Extension("hdr", "rad", "pfm"),
		decoder.MimeType("image/vnd.radiance", "image/x-hdr"),
		decoder.Decoder(decode),
	)
}

// decode decodes a Radiance HDR file, or a Portable Float Map, which starts
// with PF or Pf. Its linear light is clipped and encoded as sRGB, as libvips
// does.
//
// A Portable Float Map stores its rows from the bottom up, and the reader
// returns them in that order, so the decoder turns them over.
func decode(_ context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	dec, flip := rgbe.Decode, false
	if bytes.HasPrefix(buf, []byte("PF")) || bytes.HasPrefix(buf, []byte("Pf")) {
		dec, buf, flip = pfm.Decode, uncomment(buf), true
	}
	img, err := dec(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("hdr: %w", err)
	}
	h, ok := img.(hdr.Image)
	if !ok {
		return nil, fmt.Errorf("hdr: %T: %w", img, decoder.ErrNotSupported)
	}
	return linear(h, flip), nil
}

// uncomment returns a Portable Float Map without the comment lines of its
// header. The format has no comments, but libvips writes a line that starts
// with # after the first, and the reader stops at it. The header is the lines
// of the type, the size and the scale.
func uncomment(buf []byte) []byte {
	var out []byte
	for lines := 0; lines < 3 && len(buf) != 0; {
		line, rest, _ := bytes.Cut(buf, []byte("\n"))
		if !bytes.HasPrefix(line, []byte("#")) {
			out = append(append(out, line...), '\n')
			lines++
		}
		buf = rest
	}
	return append(out, buf...)
}

// linear returns the sRGB image of the linear light of img, with its rows in
// the other order when flip is true.
func linear(img hdr.Image, flip bool) image.Image {
	b := img.Bounds()
	return scrgb.NRGBA64(b.Dx(), b.Dy(), func(x, y int) (float64, float64, float64, float64) {
		if flip {
			y = b.Dy() - 1 - y
		}
		r, g, b, _ := img.HDRAt(b.Min.X+x, b.Min.Y+y).HDRRGBA()
		return r, g, b, 1
	})
}
