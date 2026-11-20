package meta

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

// TestLog checks that the log holds tags of each source of a file, and the
// text chunks of a PNG.
func TestLog(t *testing.T) {
	tests := []struct {
		file, mime string
		exp        []string
	}{
		{"jpeg/sunrise.jpg", "image/jpeg", []string{
			`meta: EXIF IFD0 Model = "PENTAX K-3 II"`,
			`meta: EXIF IFD0/GPSInfoIFD GPSLongitude = 4.50846`,
			`meta: IPTC IPTCApplication Keywords = ["Malaga", "Torremolinos"]`,
			`meta: XMP http://ns.adobe.com/photoshop/1.0/ State = "Andalucía"`,
			`meta: JPEG 1024x640, 1 frames`,
		}},
		{"png/sunrise.png", "image/png", []string{
			`meta: EXIF IFD0 Model = "PENTAX K-3 II"`,
			`meta: PNG tEXt exif:Artist = "Bjørn Erik Pedersen"`,
		}},
		{"avif/sunrise.avif", "image/avif", []string{
			`meta: EXIF IFD0 Model = "PENTAX K-3 II"`,
		}},
		{"jxl/sunrise.jxl", "image/jxl", []string{
			`meta: EXIF IFD0 Model = "PENTAX K-3 II"`,
		}},
		{"webp/sunrise.webp", "image/webp", []string{
			`meta: EXIF IFD0 Model = "PENTAX K-3 II"`,
			`meta: ICC profile "sRGB IEC61966-2.1"`,
		}},
		{"jpeg/rose-cmyk-icc.jpg", "image/jpeg", []string{
			`meta: ICC profile "SWOP TR003 Coated", 122072 bytes`,
		}},
		{"gif/animated.gif", "image/gif", []string{
			`meta: GIF 128x128, 6 frames`,
		}},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			ctx, lines := testContext(true)
			Log(ctx, test.mime, filepath.Join("..", "..", "testdata", filepath.FromSlash(test.file)))
			log := strings.Join(*lines, "\n")
			for _, exp := range test.exp {
				if !strings.Contains(log, exp) {
					t.Errorf("expected the log to hold %q", exp)
				}
			}
		})
	}
	// without -v, nothing is read
	ctx, lines := testContext(false)
	Log(ctx, "image/jpeg", filepath.Join("..", "..", "testdata", "jpeg", "sunrise.jpg"))
	if len(*lines) != 0 {
		t.Errorf("expected no log, got %d lines", len(*lines))
	}
}

// TestComments checks the comments of a JPEG and a GIF, which the Go
// encoders do not write, so they are put in the encoded files here.
func TestComments(t *testing.T) {
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), []color.Color{color.Black, color.White})
	var j bytes.Buffer
	if err := jpeg.Encode(&j, img, nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// a COM segment after the start of the image
	com := []byte("a comment")
	seg := append([]byte{0xff, 0xfe, 0, byte(len(com) + 2)}, com...)
	buf := append(append(append([]byte{}, j.Bytes()[:2]...), seg...), j.Bytes()[2:]...)
	ctx, lines := testContext(true)
	logJPEG(ctx, buf)
	if exp := `meta: JPEG comment = "a comment"`; !slices.Contains(*lines, exp) {
		t.Errorf("expected %q, got %q", exp, *lines)
	}
	var g bytes.Buffer
	if err := gif.Encode(&g, img, nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// a comment extension after the screen descriptor and its color table of
	// two colors
	head := 13 + 3*2
	ext := append([]byte{0x21, 0xfe, byte(len(com))}, com...)
	ext = append(ext, 0)
	buf = append(append(append([]byte{}, g.Bytes()[:head]...), ext...), g.Bytes()[head:]...)
	ctx, lines = testContext(true)
	logGIF(ctx, buf)
	if exp := `meta: GIF comment = "a comment"`; !slices.Contains(*lines, exp) {
		t.Errorf("expected %q, got %q", exp, *lines)
	}
}

// TestPNGText checks a text chunk of each kind, plain and compressed.
func TestPNGText(t *testing.T) {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write([]byte("compressed text"))
	_ = w.Close()
	tests := []struct {
		typ  string
		data []byte
		key  string
		text string
	}{
		{"tEXt", []byte("Title\x00a title"), "Title", "a title"},
		{"tEXt", []byte("Author\x00caf\xe9"), "Author", "café"},
		{"zTXt", append([]byte("Comment\x00\x00"), z.Bytes()...), "Comment", "compressed text"},
		{"iTXt", []byte("Title\x00\x00\x00en\x00Titel\x00a title"), "Title", "a title"},
		{"iTXt", append([]byte("parameters\x00\x01\x00\x00\x00"), z.Bytes()...), "parameters", "compressed text"},
	}
	for _, test := range tests {
		key, text, ok := pngText(test.typ, test.data)
		if !ok || key != test.key || text != test.text {
			t.Errorf("%s: expected %q = %q, got %q = %q (%t)", test.typ, test.key, test.text, key, text, ok)
		}
	}
}

// testContext returns a context with -v set to verbose, and the lines of
// its log.
func testContext(verbose bool) (context.Context, *[]string) {
	c := ivctx.New()
	c.Verbose = verbose
	lines := new([]string)
	c.Logger = func(s string, v ...any) {
		*lines = append(*lines, fmt.Sprintf(s, v...))
	}
	return ivctx.WithConfig(context.Background(), c), lines
}
