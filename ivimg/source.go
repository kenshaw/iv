package ivimg

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/kenshaw/rasterm"
)

// pngSignature is the first eight bytes of every PNG.
const pngSignature = "\x89PNG\r\n\x1a\n"

// DecodePNG returns an image for the PNG in buf that keeps buf as the
// encoding of the image, so that an encoder can send buf as it is. It reads
// the header, and decodes the pixels only when something asks for them (see
// [Err]).
//
// An animated PNG is decoded at once and keeps no bytes. The decoded image is
// its default image, and a terminal that reads the file can play the
// animation instead of showing that image.
func DecodePNG(buf []byte) (image.Image, error) {
	if pngAnimated(buf) {
		return png.Decode(bytes.NewReader(buf))
	}
	f, err := newFile(buf, png.DecodeConfig, png.Decode)
	if err != nil {
		return nil, err
	}
	return pngFile{f}, nil
}

// ReadPNG reads one PNG from r and decodes it as [DecodePNG] does. It reads
// up to the end of the IEND chunk and no further, so r is then at the start
// of whatever follows the PNG, such as the next PNG in a stream of them.
func ReadPNG(r *bufio.Reader) (image.Image, error) {
	buf := make([]byte, len(pngSignature), 64<<10)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if string(buf) != pngSignature {
		return nil, errors.New("png: invalid format: not a PNG file")
	}
	for {
		// length (4) and type (4), then the data and the CRC (4)
		head, err := r.Peek(8)
		if err != nil {
			return nil, fmt.Errorf("png: reading chunk: %w", noEOF(err))
		}
		n := int64(binary.BigEndian.Uint32(head)) + 12
		typ := string(head[4:8])
		if n > 1<<31 {
			return nil, fmt.Errorf("png: chunk %q of %d bytes", typ, n)
		}
		start := len(buf)
		buf = append(buf, make([]byte, n)...)
		if _, err := io.ReadFull(r, buf[start:]); err != nil {
			return nil, fmt.Errorf("png: reading chunk %q: %w", typ, noEOF(err))
		}
		if typ == "IEND" {
			return DecodePNG(buf)
		}
	}
}

// noEOF returns [io.ErrUnexpectedEOF] for [io.EOF], because a PNG that ends
// before its IEND chunk is cut short.
func noEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

// pngChunks calls f with the type and the data of each chunk of the PNG in
// buf, in order, until f returns false. It stops at a chunk that does not fit
// in buf.
func pngChunks(buf []byte, f func(typ string, data []byte) bool) {
	if len(buf) < len(pngSignature) || string(buf[:len(pngSignature)]) != pngSignature {
		return
	}
	for i := len(pngSignature); i+8 <= len(buf); {
		n := int(binary.BigEndian.Uint32(buf[i:]))
		if n < 0 || i+12+n > len(buf) {
			return
		}
		if !f(string(buf[i+4:i+8]), buf[i+8:i+8+n]) {
			return
		}
		i += 12 + n
	}
}

// pngAnimated returns true when the PNG in buf has an acTL chunk, which makes
// it an animated PNG. The chunk comes before the image data.
func pngAnimated(buf []byte) bool {
	var animated bool
	pngChunks(buf, func(typ string, _ []byte) bool {
		animated = typ == "acTL"
		return !animated && typ != "IDAT"
	})
	return animated
}

// pngOpaque returns true when the PNG in buf has no transparent pixels: its
// color type is gray or truecolor, with no alpha channel, and it has no tRNS
// chunk before its image data.
func pngOpaque(buf []byte) bool {
	opaque := false
	pngChunks(buf, func(typ string, data []byte) bool {
		switch typ {
		case "IHDR":
			// width (4), height (4), bit depth (1), color type (1). Color
			// type 0 is gray and 2 is truecolor.
			if len(data) < 10 {
				return false
			}
			opaque = data[9] == 0 || data[9] == 2
			return opaque
		case "tRNS":
			opaque = false
			return false
		}
		return typ != "IDAT"
	})
	return opaque
}

// pngOf returns the bytes of the PNG file that img holds, when it is a
// [DecodePNG] image. It does not check the bytes. An image that libvips holds
// encodes its PNG when its PNG method is called, so this is only for an image
// that holds the bytes of a file.
func pngOf(img image.Image) ([]byte, bool) {
	switch v := img.(type) {
	case pngFile:
		return v.data, true
	case rasterm.PNG:
		return v.Data, true
	}
	return nil, false
}

// PNGBytes returns the bytes of the PNG file that img was decoded from, when
// img holds them: it is a [rasterm.PNG], and the bytes start with the PNG
// signature and an IHDR chunk that holds the width and height of the bounds
// of img. An image that encodes its PNG when asked, such as an image that
// libvips holds, is not one.
func PNGBytes(img image.Image) ([]byte, bool) {
	data, ok := pngOf(img)
	if !ok {
		return nil, false
	}
	if len(data) < 24 || string(data[:8]) != pngSignature || string(data[12:16]) != "IHDR" {
		return nil, false
	}
	b := img.Bounds()
	w, h := binary.BigEndian.Uint32(data[16:20]), binary.BigEndian.Uint32(data[20:24])
	return data, int64(w) == int64(b.Dx()) && int64(h) == int64(b.Dy())
}

// DecodeJPEG returns an image for the JPEG in buf that keeps buf as the
// encoding of the image, so that an encoder can send buf as it is. It reads
// the header, and decodes the pixels only when something asks for them (see
// [Err]).
//
// Two kinds of JPEG are decoded at once and keep no bytes, because a terminal
// can show them differently from the decoded image:
//
//   - A CMYK or YCCK JPEG. The color conversion of the terminal is not the
//     one of the Go decoder, and some terminals show the colors inverted.
//   - A JPEG whose EXIF orientation is not 1. The Go decoder does not apply
//     the orientation, so the image is turned here, which makes new pixels.
func DecodeJPEG(buf []byte) (image.Image, error) {
	f, err := newFile(buf, jpeg.DecodeConfig, jpeg.Decode)
	if err != nil {
		return nil, err
	}
	o := jpegOrientation(buf)
	if f.config.ColorModel != color.CMYKModel && (o == 0 || o == 1) {
		return jpegFile{f}, nil
	}
	img, err := jpeg.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	return orient(img, o), nil
}

// JPEGBytes returns the encoding of img as a JPEG when img holds one: it is a
// [rasterm.JPEGImage] and the bytes start with the JPEG start of image
// marker. An encoder that writes them does not need the size check of
// [PNGBytes], because a [rasterm.JPEG] is never resized in place.
func JPEGBytes(img image.Image) ([]byte, bool) {
	j, ok := img.(rasterm.JPEGImage)
	if !ok {
		return nil, false
	}
	data := j.JPEG()
	return data, len(data) > 2 && data[0] == 0xff && data[1] == 0xd8
}

// jpegOrientation returns the EXIF orientation of the JPEG in buf, from 1 to
// 8, or 0 when it has none. It reads the APP1 segments that come before the
// first frame header.
func jpegOrientation(buf []byte) int {
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return 0
	}
	for i := 2; i+4 <= len(buf); {
		if buf[i] != 0xff {
			return 0
		}
		m := buf[i+1]
		switch {
		case m == 0xff: // fill byte
			i++
			continue
		case m == 0x01, 0xd0 <= m && m <= 0xd7: // a marker with no length
			i += 2
			continue
		case m == 0xd9, m == 0xda, 0xc0 <= m && m <= 0xcf && m != 0xc4 && m != 0xc8 && m != 0xcc:
			// end of image, start of scan or start of frame: the EXIF data
			// comes before all of them
			return 0
		}
		n := int(binary.BigEndian.Uint16(buf[i+2:]))
		if n < 2 || i+2+n > len(buf) {
			return 0
		}
		if seg := buf[i+4 : i+2+n]; m == 0xe1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return exifOrientation(seg[6:])
		}
		i += 2 + n
	}
	return 0
}

// exifOrientation returns the orientation tag (0x0112) of the first image
// file directory of the TIFF structure in buf, or 0 when it has none.
func exifOrientation(buf []byte) int {
	if len(buf) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(buf[:4]) {
	case "II*\x00":
		order = binary.LittleEndian
	case "MM\x00*":
		order = binary.BigEndian
	default:
		return 0
	}
	ifd := int(order.Uint32(buf[4:]))
	if ifd < 8 || ifd+2 > len(buf) {
		return 0
	}
	// each entry is a tag (2), a type (2), a count (4) and a value (4)
	for i, n := ifd+2, int(order.Uint16(buf[ifd:])); n > 0 && i+12 <= len(buf); i, n = i+12, n-1 {
		if order.Uint16(buf[i:]) == 0x0112 {
			// a SHORT, which is in the first 2 bytes of the value
			return int(order.Uint16(buf[i+8:]))
		}
	}
	return 0
}
