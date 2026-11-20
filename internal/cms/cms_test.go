package cms

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/ivy/icc"
)

// TestJPEGProfile checks that the profile of a CMYK JPEG is read from its APP2
// segment, and that a JPEG with no profile has none.
func TestJPEGProfile(t *testing.T) {
	buf := readFile(t, "rose-cmyk-icc.jpg")
	p := JPEGProfile(buf)
	// the signature of a profile is at byte 36
	if len(p) < 40 || string(p[36:40]) != "acsp" {
		t.Fatalf("expected an ICC profile, got %d bytes", len(p))
	}
	if _, err := icc.Parse(p); err != nil {
		t.Errorf("expected the profile to parse, got: %v", err)
	}
	if p := JPEGProfile(readFile(t, "rose-cmyk.jpg")); p != nil {
		t.Errorf("expected no profile, got %d bytes", len(p))
	}
}

// TestJPEGProfileChunks checks that a profile in two APP2 segments, which
// come out of order, is put back together.
func TestJPEGProfileChunks(t *testing.T) {
	app2 := func(seq byte, data string) []byte {
		payload := append([]byte("ICC_PROFILE\x00"), seq, 2)
		payload = append(payload, data...)
		seg := []byte{0xff, 0xe2, 0, 0}
		binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
		return append(seg, payload...)
	}
	buf := []byte{0xff, 0xd8}
	buf = append(buf, app2(2, "second")...)
	buf = append(buf, app2(1, "first ")...)
	buf = append(buf, 0xff, 0xda, 0, 2)
	if got := string(JPEGProfile(buf)); got != "first second" {
		t.Errorf("expected %q, got %q", "first second", got)
	}
	// a missing segment gives no profile
	if got := JPEGProfile(append([]byte{0xff, 0xd8}, app2(1, "first")...)); got != nil {
		t.Errorf("expected no profile, got %q", got)
	}
}

// TestConvert checks which images are converted: a CMYK image with no
// profile, and an image in Display P3. An image in sRGB, with no color space,
// or in HDR is returned as it is.
func TestConvert(t *testing.T) {
	ctx := ivctx.WithConfig(context.Background(), ivctx.New())
	rgb := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range rgb.Pix {
		rgb.Pix[i] = 0xc0
	}
	cmyk := image.NewCMYK(image.Rect(0, 0, 4, 4))
	for i := range cmyk.Pix {
		cmyk.Pix[i] = 0x40
	}
	for _, test := range []struct {
		name    string
		img     image.Image
		cicp    *icc.CICP
		convert bool
	}{
		{"no color space", rgb, nil, false},
		{"srgb", rgb, &icc.CICP{ColorPrimaries: 1, TransferCharacteristics: 13}, false},
		{"unspecified", rgb, &icc.CICP{ColorPrimaries: 2, TransferCharacteristics: 2}, false},
		{"pq", rgb, &icc.CICP{ColorPrimaries: 9, TransferCharacteristics: 16}, false},
		{"display p3", rgb, &icc.CICP{ColorPrimaries: 12, TransferCharacteristics: 13}, true},
		{"cmyk", cmyk, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Convert(ctx, test.img, nil, test.cicp)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			same := got == test.img
			if same == test.convert {
				t.Fatalf("expected converted %t, got %T", test.convert, got)
			}
			if test.convert && got.ColorModel() != color.NRGBAModel {
				t.Errorf("expected an NRGBA image, got %T", got)
			}
		})
	}
}

// TestConvertProfile checks that a CMYK JPEG decodes to the same pixels with
// no profile and with the default CMYK profile of ivy in its file.
func TestConvertProfile(t *testing.T) {
	ctx := ivctx.WithConfig(context.Background(), ivctx.New())
	cmyk := image.NewCMYK(image.Rect(0, 0, 8, 8))
	for i := range cmyk.Pix {
		cmyk.Pix[i] = byte(i * 7)
	}
	a, err := Convert(ctx, cmyk, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	b, err := Convert(ctx, cmyk, icc.DefaultCMYK(), nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !bytes.Equal(a.(*image.NRGBA).Pix, b.(*image.NRGBA).Pix) {
		t.Error("expected the same pixels with the default profile")
	}
	// a profile that is not valid leaves the image as it is
	if got, err := Convert(ctx, cmyk, []byte("not a profile"), nil); err != nil || got != image.Image(cmyk) {
		t.Errorf("expected the image as it is, got %T, %v", got, err)
	}
}

// readFile reads a file of testdata/jpeg.
func readFile(t *testing.T, name string) []byte {
	t.Helper()
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "jpeg", name))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	return buf
}

// TestDescription checks the description of a profile of version 2, from a
// CMYK JPEG, and of the default CMYK profile of ivy.
func TestDescription(t *testing.T) {
	if got, exp := Description(JPEGProfile(readFile(t, "rose-cmyk-icc.jpg"))), "SWOP TR003 Coated"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
	if got := Description(icc.DefaultCMYK()); got == "" {
		t.Error("expected a description of the default CMYK profile")
	} else {
		t.Logf("default CMYK: %q", got)
	}
	if got := Description([]byte("short")); got != "" {
		t.Errorf("expected no description, got %q", got)
	}
}
