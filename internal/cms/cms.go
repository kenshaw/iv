// Package cms converts the colors of a decoded image to sRGB, with the icc
// package of ivy (D57).
//
// A decoder gives the ICC profile that its file embeds, or the CICP code
// points of the file. An image that is sRGB already, or whose file gives no
// color space, is returned as it is. An image with the PQ or the HLG transfer
// of HDR is returned as it is too, because the icc package does no tone
// mapping, and the image would look dark. A CMYK image with no profile is
// converted with the default CMYK profile of ivy.
//
// See: https://github.com/xo/ivy
package cms

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"unicode/utf16"

	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/ivy/icc"
)

// The code points of Rec. ITU-T H.273 that the package reads.
const (
	primariesBT709     = 1
	unspecified        = 2
	transferSRGB       = 13
	transferPQ         = 16
	transferHLG        = 18
	maxCodePoint       = 255
	renderingIntent    = icc.RelativeColorimetric
	jpegICCMarker      = 0xe2
	jpegStartOfScan    = 0xda
	jpegICCSignature   = "ICC_PROFILE\x00"
	jpegICCHeaderBytes = len(jpegICCSignature) + 2
)

// CICP returns the code points of a file as an [icc.CICP], and false when one
// of them is not a code point.
func CICP(primaries, transfer, matrix uint16, fullRange bool) (*icc.CICP, bool) {
	if primaries > maxCodePoint || transfer > maxCodePoint || matrix > maxCodePoint {
		return nil, false
	}
	return &icc.CICP{
		ColorPrimaries:          uint8(primaries),
		TransferCharacteristics: uint8(transfer),
		MatrixCoefficients:      uint8(matrix),
		FullRange:               fullRange,
	}, true
}

// Convert returns img in sRGB. profile is the ICC profile that the file of img
// embeds, and c holds the CICP code points of the file, when it has them.
// The profile wins over the code points.
//
// A profile or code points that the icc package cannot use leave img as it
// is, with a warning. Convert returns an error only when ctx is done.
func Convert(ctx context.Context, img image.Image, profile []byte, c *icc.CICP) (image.Image, error) {
	var p *icc.Profile
	var err error
	switch {
	case len(profile) != 0:
		p, err = icc.Parse(profile)
	case img.ColorModel() == color.CMYKModel:
		p, err = icc.Parse(icc.DefaultCMYK())
	case c == nil || isSRGB(c) || isHDR(c):
		return img, nil
	default:
		cc := *c
		if cc.ColorPrimaries == unspecified {
			cc.ColorPrimaries = primariesBT709
		}
		if cc.TransferCharacteristics == unspecified {
			cc.TransferCharacteristics = transferSRGB
		}
		p, err = icc.FromCICP(cc)
	}
	if err != nil {
		ivctx.Warnf(ctx, "color profile: %v", err)
		return img, nil
	}
	out, err := p.ToSRGB(ctx, img, renderingIntent)
	switch {
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case err != nil:
		ivctx.Warnf(ctx, "color profile: %v", err)
		return img, nil
	}
	name := "the default CMYK profile of ivy"
	switch {
	case len(profile) != 0:
		name = fmt.Sprintf("the profile %q", Description(profile))
	case img.ColorModel() != color.CMYKModel:
		name = fmt.Sprintf("the code points %d/%d/%d", c.ColorPrimaries, c.TransferCharacteristics, c.MatrixCoefficients)
	}
	ivctx.Logf(ctx, "color profile: %T converted to sRGB with %s", img, name)
	return out, nil
}

// ConvertAll converts each frame of an animation, as [Convert] does.
func ConvertAll(ctx context.Context, frames []image.Image, profile []byte, c *icc.CICP) ([]image.Image, error) {
	out := make([]image.Image, len(frames))
	for i, frame := range frames {
		var err error
		if out[i], err = Convert(ctx, frame, profile, c); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// isSRGB reports whether c names sRGB, or leaves the color space unspecified,
// which a reader takes as sRGB.
func isSRGB(c *icc.CICP) bool {
	p, t := c.ColorPrimaries, c.TransferCharacteristics
	return (p == primariesBT709 || p == unspecified) && (t == transferSRGB || t == unspecified)
}

// isHDR reports whether c names a transfer of HDR.
func isHDR(c *icc.CICP) bool {
	return c.TransferCharacteristics == transferPQ || c.TransferCharacteristics == transferHLG
}

// JPEGProfile returns the ICC profile that the JPEG in buf embeds, or nil. A
// profile is stored in APP2 segments that start with ICC_PROFILE, then the
// number of the segment and the count of the segments, each from 1. The
// segments come before the first scan.
func JPEGProfile(buf []byte) []byte {
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return nil
	}
	var chunks [][]byte
	count := 0
	for i := 2; i+4 <= len(buf); {
		if buf[i] != 0xff {
			return nil
		}
		m := buf[i+1]
		if m == jpegStartOfScan {
			break
		}
		n := int(binary.BigEndian.Uint16(buf[i+2:]))
		if n < 2 || i+2+n > len(buf) {
			return nil
		}
		seg := buf[i+4 : i+2+n]
		if m == jpegICCMarker && len(seg) > jpegICCHeaderBytes && bytes.HasPrefix(seg, []byte(jpegICCSignature)) {
			seq, total := int(seg[len(jpegICCSignature)]), int(seg[len(jpegICCSignature)+1])
			if total == 0 || seq == 0 || seq > total || (count != 0 && total != count) {
				return nil
			}
			if chunks == nil {
				chunks, count = make([][]byte, total), total
			}
			chunks[seq-1] = seg[jpegICCHeaderBytes:]
		}
		i += 2 + n
	}
	var profile []byte
	for _, chunk := range chunks {
		if chunk == nil {
			return nil
		}
		profile = append(profile, chunk...)
	}
	return profile
}

// Description returns the description of the ICC profile, such as "Display
// P3", or an empty string when the profile has none. A profile of version 2
// holds it as a textDescriptionType, in ASCII, and a profile of version 4 as
// a multiLocalizedUnicodeType, in UTF-16, of which the first record is taken.
func Description(profile []byte) string {
	const tagTable = 128
	if len(profile) < tagTable+4 {
		return ""
	}
	n := int(binary.BigEndian.Uint32(profile[tagTable:]))
	for i := range min(n, (len(profile)-tagTable-4)/12) {
		e := profile[tagTable+4+12*i:]
		if string(e[:4]) != "desc" {
			continue
		}
		off, size := int(binary.BigEndian.Uint32(e[4:])), int(binary.BigEndian.Uint32(e[8:]))
		if off < 0 || size < 12 || off+size > len(profile) || off+size < off {
			return ""
		}
		return description(profile[off : off+size])
	}
	return ""
}

// description returns the text of the data of a desc tag.
func description(data []byte) string {
	switch string(data[:4]) {
	case "desc":
		// the type, 4 reserved bytes, the length with its NUL, the text
		n := int(binary.BigEndian.Uint32(data[8:]))
		if n <= 0 || 12+n > len(data) {
			return ""
		}
		return string(bytes.TrimRight(data[12:12+n], "\x00"))
	case "mluc":
		// the type, 4 reserved bytes, the count of records and their size,
		// then each record: the language, the country, the length and the
		// offset of its text
		if len(data) < 28 || binary.BigEndian.Uint32(data[8:]) == 0 {
			return ""
		}
		n, off := int(binary.BigEndian.Uint32(data[20:])), int(binary.BigEndian.Uint32(data[24:]))
		if n < 0 || off < 0 || off+n > len(data) || off+n < off {
			return ""
		}
		u := make([]uint16, n/2)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(data[off+2*i:])
		}
		return string(utf16.Decode(u))
	}
	return ""
}
