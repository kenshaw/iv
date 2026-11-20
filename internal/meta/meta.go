// Package meta writes the metadata of an image file to the verbose log of iv
// (D60).
//
// bep/imagemeta reads every Exif, IPTC and XMP tag of a JPEG, TIFF, PNG,
// WebP, HEIC, AVIF or JPEG XL file, and of the camera RAW files that it
// knows, and the size and the count of the frames that the file gives. The
// package adds what imagemeta does not read: the text chunks of a PNG, the
// comments of a JPEG and a GIF, and the description of an ICC profile that
// the file embeds.
//
// See: https://github.com/bep/imagemeta
package meta

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
	"github.com/bep/imagemeta"
	"github.com/kenshaw/iv/internal/cms"
	"github.com/kenshaw/iv/ivctx"
)

// formats are the formats of imagemeta, by mime type.
var formats = map[string]imagemeta.ImageFormat{
	"image/jpeg":           imagemeta.JPEG,
	"image/tiff":           imagemeta.TIFF,
	"image/png":            imagemeta.PNG,
	"image/apng":           imagemeta.PNG,
	"image/webp":           imagemeta.WebP,
	"image/heic":           imagemeta.HEIF,
	"image/heif":           imagemeta.HEIF,
	"image/heic-sequence":  imagemeta.HEIF,
	"image/heif-sequence":  imagemeta.HEIF,
	"image/avif":           imagemeta.AVIF,
	"image/avif-sequence":  imagemeta.AVIF,
	"image/jxl":            imagemeta.JXL,
	"image/gif":            imagemeta.GIF,
	"image/x-adobe-dng":    imagemeta.DNG,
	"image/x-canon-cr2":    imagemeta.CR2,
	"image/x-nikon-nef":    imagemeta.NEF,
	"image/x-sony-arw":     imagemeta.ARW,
	"image/x-pentax-pef":   imagemeta.PEF,
	"image/x-nikon-nrw":    imagemeta.NEF,
	"image/x-canon-cr2raw": imagemeta.CR2,
}

// rawFormats are the formats of the camera RAW files, by extension, for a
// file whose mime type libmagic does not name.
var rawFormats = map[string]imagemeta.ImageFormat{
	"dng": imagemeta.DNG,
	"cr2": imagemeta.CR2,
	"nef": imagemeta.NEF,
	"nrw": imagemeta.NEF,
	"arw": imagemeta.ARW,
	"srf": imagemeta.ARW,
	"sr2": imagemeta.ARW,
	"pef": imagemeta.PEF,
}

// Log writes the metadata of the file pathName, whose mime type is mime, to
// the verbose log. It writes nothing when the log is off, or when the format
// holds no metadata that the package reads.
func Log(ctx context.Context, mime, pathName string) {
	if !ivctx.Get(ctx).Verbose {
		return
	}
	format, ok := formats[mime]
	if !ok {
		format, ok = rawFormats[ivctx.FileExt(pathName)]
	}
	if !ok {
		return
	}
	buf, err := os.ReadFile(pathName)
	if err != nil {
		ivctx.Logf(ctx, "meta: %v", err)
		return
	}
	logTags(ctx, format, buf)
	switch format {
	case imagemeta.JPEG:
		logJPEG(ctx, buf)
	case imagemeta.PNG:
		logPNG(ctx, buf)
	case imagemeta.GIF:
		logGIF(ctx, buf)
	case imagemeta.WebP:
		logWebP(ctx, buf)
	}
}

// logTags writes each tag that imagemeta reads, and the size and the frame
// count of the file.
func logTags(ctx context.Context, format imagemeta.ImageFormat, buf []byte) {
	res, err := imagemeta.Decode(imagemeta.Options{
		R:           bytes.NewReader(buf),
		ImageFormat: format,
		Sources:     imagemeta.EXIF | imagemeta.IPTC | imagemeta.XMP | imagemeta.CONFIG,
		// every tag, and not only those of IFD0
		ShouldHandleTag: func(imagemeta.TagInfo) bool {
			return true
		},
		HandleTag: func(t imagemeta.TagInfo) error {
			ns := t.Namespace
			if ns != "" {
				ns = " " + ns
			}
			ivctx.Logf(ctx, "meta: %s%s %s = %s", t.Source, ns, t.Tag, value(t.Value))
			return nil
		},
		Warnf: func(s string, v ...any) {
			ivctx.Logf(ctx, "meta: warning: "+s, v...)
		},
		// a JPEG XL file can hold its Exif and XMP compressed with brotli
		DecompressBrotli: func(r io.Reader) io.Reader {
			return brotli.NewReader(r)
		},
	})
	if err != nil {
		ivctx.Logf(ctx, "meta: %v", err)
		return
	}
	if c := res.ImageConfig; c.Width != 0 {
		ivctx.Logf(ctx, "meta: %s %dx%d, %d frames", format, c.Width, c.Height, c.FrameCount)
	}
}

// value returns v as text: a string quoted, and bytes in hexadecimal.
func value(v any) string {
	switch v := v.(type) {
	case string:
		return strconv.Quote(v)
	case []byte:
		return fmt.Sprintf("%d bytes: %x", len(v), v)
	case []string:
		s := make([]string, len(v))
		for i, x := range v {
			s[i] = strconv.Quote(x)
		}
		return "[" + strings.Join(s, ", ") + "]"
	}
	return fmt.Sprint(v)
}

// logProfile writes the description of an ICC profile that a file embeds.
func logProfile(ctx context.Context, profile []byte) {
	if len(profile) != 0 {
		ivctx.Logf(ctx, "meta: ICC profile %q, %d bytes", cms.Description(profile), len(profile))
	}
}

// logJPEG writes the comments of a JPEG, and the description of its ICC
// profile. A comment is a COM segment before the first scan.
func logJPEG(ctx context.Context, buf []byte) {
	logProfile(ctx, cms.JPEGProfile(buf))
	if len(buf) < 4 || buf[0] != 0xff || buf[1] != 0xd8 {
		return
	}
	for i := 2; i+4 <= len(buf) && buf[i] == 0xff; {
		m, n := buf[i+1], int(binary.BigEndian.Uint16(buf[i+2:]))
		if m == 0xda || n < 2 || i+2+n > len(buf) {
			return
		}
		if m == 0xfe {
			ivctx.Logf(ctx, "meta: JPEG comment = %s", strconv.Quote(string(buf[i+4:i+2+n])))
		}
		i += 2 + n
	}
}

// pngSkipped are the keywords of the text chunks that imagemeta reads.
var pngSkipped = []string{
	"XML:com.adobe.xmp",
	"Raw profile type exif",
	"Raw profile type APP1",
	"Raw profile type iptc",
	"Raw profile type xmp",
}

// logPNG writes the text chunks of a PNG, and the name and the description
// of its ICC profile.
func logPNG(ctx context.Context, buf []byte) {
	const sig = "\x89PNG\r\n\x1a\n"
	if !bytes.HasPrefix(buf, []byte(sig)) {
		return
	}
	for i := len(sig); i+12 <= len(buf); {
		n := int(binary.BigEndian.Uint32(buf[i:]))
		typ := string(buf[i+4 : i+8])
		if n < 0 || i+12+n > len(buf) || i+12+n < i {
			return
		}
		data := buf[i+8 : i+8+n]
		i += 12 + n
		switch typ {
		case "tEXt", "zTXt", "iTXt":
			key, text, ok := pngText(typ, data)
			switch {
			case !ok:
				ivctx.Logf(ctx, "meta: PNG %s: not valid", typ)
			case !slices.Contains(pngSkipped, key):
				ivctx.Logf(ctx, "meta: PNG %s %s = %s", typ, key, strconv.Quote(text))
			}
		case "iCCP":
			name, rest, ok := bytes.Cut(data, []byte{0})
			if !ok || len(rest) < 1 {
				continue
			}
			profile, _ := inflate(rest[1:])
			ivctx.Logf(ctx, "meta: PNG iCCP %s", strconv.Quote(latin1(name)))
			logProfile(ctx, profile)
		case "IEND":
			return
		}
	}
}

// pngText returns the keyword and the text of a text chunk. A tEXt and a
// zTXt are in Latin-1, and an iTXt is in UTF-8, and can be compressed.
func pngText(typ string, data []byte) (string, string, bool) {
	key, rest, ok := bytes.Cut(data, []byte{0})
	if !ok {
		return "", "", false
	}
	switch typ {
	case "tEXt":
		return latin1(key), latin1(rest), true
	case "zTXt":
		if len(rest) < 1 {
			return "", "", false
		}
		text, err := inflate(rest[1:])
		return latin1(key), latin1(text), err == nil
	}
	// the flag and the method of the compression, the language and the
	// translated keyword, then the text
	if len(rest) < 2 {
		return "", "", false
	}
	compressed := rest[0] == 1
	parts := bytes.SplitN(rest[2:], []byte{0}, 3)
	if len(parts) != 3 {
		return "", "", false
	}
	text := parts[2]
	if compressed {
		var err error
		if text, err = inflate(text); err != nil {
			return "", "", false
		}
	}
	return latin1(key), string(text), true
}

// maxInflate is the most bytes that inflate returns, so that a small chunk
// cannot ask for a large allocation.
const maxInflate = 16 << 20

// inflate returns the data of a zlib stream.
func inflate(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, maxInflate))
}

// latin1 returns the text of b, which the format says is in Latin-1. Many
// programs write UTF-8 there, and text that is valid UTF-8 is almost never
// meant as Latin-1, so such text is taken as UTF-8.
func latin1(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

// logGIF writes the comments of a GIF. A comment is an extension block whose
// label is 0xfe, made of sub-blocks.
func logGIF(ctx context.Context, buf []byte) {
	if len(buf) < 13 || !bytes.HasPrefix(buf, []byte("GIF")) {
		return
	}
	i := 13
	// the global color table follows the screen descriptor
	if buf[10]&0x80 != 0 {
		i += 3 << (buf[10]&7 + 1)
	}
	for i < len(buf) {
		switch buf[i] {
		case 0x21:
			if i+2 > len(buf) {
				return
			}
			label := buf[i+1]
			var text []byte
			i += 2
			for i < len(buf) && buf[i] != 0 {
				n := int(buf[i])
				if i+1+n > len(buf) {
					return
				}
				text = append(text, buf[i+1:i+1+n]...)
				i += 1 + n
			}
			i++
			if label == 0xfe {
				ivctx.Logf(ctx, "meta: GIF comment = %s", strconv.Quote(latin1(text)))
			}
		case 0x2c:
			// an image descriptor, its color table, and its data
			if i+10 > len(buf) {
				return
			}
			flags := buf[i+9]
			i += 10
			if flags&0x80 != 0 {
				i += 3 << (flags&7 + 1)
			}
			// the minimum code size, then the sub-blocks
			i++
			for i < len(buf) && buf[i] != 0 {
				i += 1 + int(buf[i])
			}
			i++
		default:
			return
		}
	}
}

// logWebP writes the description of the ICC profile of a WebP, which an ICCP
// chunk holds.
func logWebP(ctx context.Context, buf []byte) {
	if len(buf) < 12 || string(buf[:4]) != "RIFF" || string(buf[8:12]) != "WEBP" {
		return
	}
	for i := 12; i+8 <= len(buf); {
		n := int(binary.LittleEndian.Uint32(buf[i+4:]))
		if n < 0 || i+8+n > len(buf) || i+8+n < i {
			return
		}
		if string(buf[i:i+4]) == "ICCP" {
			logProfile(ctx, buf[i+8:i+8+n])
			return
		}
		// a chunk is padded to an even size
		i += 8 + n + n&1
	}
}
