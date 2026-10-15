package ivimg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/rasterm"
)

// TestPrepareKeepsTheBytes checks that a PNG that is not scaled and has no
// background to composite leaves the pipeline with the bytes of its file.
func TestPrepareKeepsTheBytes(t *testing.T) {
	// inside the display size, and above the minimum size of 64
	buf := encodePNG(t, opaqueNRGBA(80, 80))
	img, err := DecodePNG(buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	got, err := Prepare(testContext(t, 100, 100, nil), "image/png", img)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	data, ok := PNGBytes(got)
	switch {
	case !ok:
		t.Fatalf("expected the bytes of the file, got %T", got)
	case !bytes.Equal(data, buf):
		t.Error("expected the bytes of the file as they are")
	}
}

// TestPrepareDropsTheBytes checks that a scaled PNG holds no bytes, so the
// bytes of the file cannot reach an encoder after the pixels changed.
func TestPrepareDropsTheBytes(t *testing.T) {
	img, err := DecodePNG(encodePNG(t, opaqueNRGBA(200, 100)))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	got, err := Prepare(testContext(t, 100, 100, nil), "image/png", img)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if size := got.Bounds().Size(); size != (image.Point{X: 100, Y: 50}) {
		t.Fatalf("expected 100x50, got %v", size)
	}
	if _, ok := got.(rasterm.PNGImage); ok {
		t.Error("expected no bytes after a resize")
	}
}

// TestResize checks the resize honors the size asked for, and hands back the
// image untouched when there is nothing to do.
func TestResize(t *testing.T) {
	ctx := testContext(t, 0, 0, nil)
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	src.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if got, _ := Resize(ctx, src, 8, 8); got != image.Image(src) {
		t.Error("expected the same image back when it is already that size")
	}
	got, err := Resize(ctx, src, 32, 32)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if size := got.Bounds().Size(); size != (image.Point{X: 32, Y: 32}) {
		t.Fatalf("expected 32x32, got %v", size)
	}
	// a whole number magnification replicates pixels, so the corner the
	// source painted is still that color across the whole 4x4 block
	for _, pt := range []image.Point{{X: 0, Y: 0}, {X: 3, Y: 3}} {
		if r, _, _, a := got.At(pt.X, pt.Y).RGBA(); r>>8 != 255 || a>>8 != 255 {
			t.Errorf("expected the magnified pixel at %v to keep its color", pt)
		}
	}
}

// TestAddBackground checks when the background is composited, and when the
// image is returned as it is.
func TestAddBackground(t *testing.T) {
	red := colors.New(255, 0, 0, 255)
	none := colors.New(0, 0, 0, 0)
	transparent := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	opaque := image.NewYCbCr(image.Rect(0, 0, 2, 2), image.YCbCrSubsampleRatio444)
	opaquePNG, err := DecodePNG(encodePNG(t, opaqueNRGBA(2, 2)))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, test := range []struct {
		name string
		bg   *colors.Color
		mime string
		img  image.Image
		same bool
	}{
		{"transparent background", &none, "image/png", transparent, true},
		{"svg draws its own background", &red, "image/svg+xml", transparent, true},
		{"opaque type", &red, "image/jpeg", opaque, true},
		{"opaque png", &red, "image/png", opaquePNG, true},
		{"transparent image", &red, "image/png", transparent, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := AddBackground(testContext(t, 0, 0, test.bg), test.mime, test.img)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if test.same {
				if !same(got, test.img) {
					t.Fatalf("expected the image as it was, got %T", got)
				}
				return
			}
			if r, g, b, a := got.At(0, 0).RGBA(); r != 0xffff || g != 0 || b != 0 || a != 0xffff {
				t.Errorf("expected red, got %v", got.At(0, 0))
			}
		})
	}
}

// TestOpaque checks the formats that [Opaque] reads without a look at each
// pixel.
func TestOpaque(t *testing.T) {
	rgbaPNG := encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 2, 2)))
	grayPNG := encodePNG(t, image.NewGray(image.Rect(0, 0, 2, 2)))
	for _, test := range []struct {
		name string
		img  image.Image
		exp  bool
	}{
		{"ycbcr", image.NewYCbCr(image.Rect(0, 0, 1, 1), image.YCbCrSubsampleRatio420), true},
		{"gray", image.NewGray(image.Rect(0, 0, 1, 1)), true},
		{"nrgba is not scanned", opaqueNRGBA(1, 1), false},
		{"opaque palette", image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}), true},
		{"transparent palette", image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Transparent}), false},
		{"rgba png", rasterm.PNG{Image: image.NewNRGBA(image.Rect(0, 0, 2, 2)), Data: rgbaPNG}, false},
		{"gray png", rasterm.PNG{Image: image.NewGray(image.Rect(0, 0, 2, 2)), Data: grayPNG}, true},
		{"gray png with trns", rasterm.PNG{Image: image.NewGray(image.Rect(0, 0, 2, 2)), Data: insertChunk(t, grayPNG, "tRNS", []byte{0, 0})}, false},
	} {
		if got := Opaque(test.img); got != test.exp {
			t.Errorf("%s: expected %t, got %t", test.name, test.exp, got)
		}
	}
}

// TestDecodePNGAnimated checks that an animated PNG keeps no bytes.
func TestDecodePNGAnimated(t *testing.T) {
	buf := insertChunk(t, encodePNG(t, opaqueNRGBA(2, 2)), "acTL", make([]byte, 8))
	img, err := DecodePNG(buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := img.(rasterm.PNGImage); ok {
		t.Error("expected no bytes for an animated PNG")
	}
}

// TestReadPNG checks that a stream of PNGs reads one at a time, each with its
// own bytes.
func TestReadPNG(t *testing.T) {
	a, b := encodePNG(t, opaqueNRGBA(2, 2)), encodePNG(t, opaqueNRGBA(3, 1))
	r := bufio.NewReader(bytes.NewReader(append(append([]byte{}, a...), b...)))
	for i, exp := range [][]byte{a, b} {
		img, err := ReadPNG(r)
		if err != nil {
			t.Fatalf("png %d: expected no error, got: %v", i, err)
		}
		if data, ok := PNGBytes(img); !ok || !bytes.Equal(data, exp) {
			t.Errorf("png %d: expected its own bytes", i)
		}
	}
	if _, err := ReadPNG(r); err == nil {
		t.Error("expected an error at the end of the stream")
	}
}

// TestDecodeLazily checks that a PNG and a JPEG decode their pixels only when
// they are asked for, and that a decode error then reaches [Err] and the
// steps of the pipeline.
func TestDecodeLazily(t *testing.T) {
	buf := encodePNG(t, opaqueNRGBA(80, 60))
	img, err := DecodePNG(buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	f := img.(pngFile)
	if f.Bounds() != image.Rect(0, 0, 80, 60) || f.pix != nil {
		t.Fatalf("expected the size from the header and no pixels, got %v and %T", f.Bounds(), f.pix)
	}
	if pix := Unwrap(img); pix == nil || pix.Bounds() != img.Bounds() || Err(img) != nil {
		t.Errorf("expected the pixels on Unwrap, got %T and %v", pix, Err(img))
	}
	// a good header in front of bad data: the header reads, and the pixels
	// do not
	bad := append(append([]byte{}, buf[:33]...), make([]byte, 64)...)
	img, err = DecodePNG(bad)
	if err != nil {
		t.Fatalf("expected the header to read, got: %v", err)
	}
	if _, err := Resize(testContext(t, 0, 0, nil), img, 40, 30); err == nil {
		t.Error("expected the resize to return the decode error")
	}
	if Err(img) == nil {
		t.Error("expected Err to return the decode error")
	}
}

// TestOrient checks that a JPEG with an EXIF orientation is turned, and keeps
// no bytes.
func TestOrient(t *testing.T) {
	var buf bytes.Buffer
	src := opaqueNRGBA(16, 8)
	// the top right quarter is red. A block is used, because a JPEG blurs the
	// color of a single pixel into its neighbors
	for y := range 4 {
		for x := 8; x < 16; x++ {
			src.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	img, err := DecodeJPEG(withOrientation(buf.Bytes(), 6))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := JPEGBytes(img); ok {
		t.Error("expected no bytes for a turned JPEG")
	}
	if size := img.Bounds().Size(); size != (image.Point{X: 8, Y: 16}) {
		t.Fatalf("expected 8x16 after a quarter turn, got %v", size)
	}
	// a quarter turn clockwise takes the top right corner to the bottom
	// right, and the top left to the top right
	if r, g, _, _ := img.At(6, 14).RGBA(); r>>8 < 200 || g>>8 > 60 {
		t.Errorf("expected red in the bottom right corner, got %v", img.At(6, 14))
	}
	if r, g, _, _ := img.At(6, 1).RGBA(); r>>8 > 160 || g>>8 < 100 {
		t.Errorf("expected gray in the top right corner, got %v", img.At(6, 1))
	}
}

// TestDecodeJPEG checks which JPEGs keep their bytes.
func TestDecodeJPEG(t *testing.T) {
	buf := encodeJPEG(t)
	for _, test := range []struct {
		name string
		buf  []byte
		keep bool
	}{
		{"plain", buf, true},
		{"orientation 1", withOrientation(buf, 1), true},
		{"orientation 6", withOrientation(buf, 6), false},
	} {
		img, err := DecodeJPEG(test.buf)
		if err != nil {
			t.Fatalf("%s: expected no error, got: %v", test.name, err)
		}
		if _, ok := JPEGBytes(img); ok != test.keep {
			t.Errorf("%s: expected bytes kept %t, got %t", test.name, test.keep, ok)
		}
	}
}

// TestRaw checks the form in which each kind of image reaches rasterm.
func TestRaw(t *testing.T) {
	nrgba := opaqueNRGBA(2, 2)
	if r, ok := Raw(rasterm.Kitty, nrgba).(rasterm.RGBA); !ok || &r.Data[0] != &nrgba.Pix[0] {
		t.Error("expected a tight NRGBA as RGBA, with nothing copied")
	}
	// a paletted image makes a small PNG, so rasterm encodes it
	paletted := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black})
	if got := Raw(rasterm.Kitty, paletted); got != image.Image(paletted) {
		t.Errorf("expected a paletted image as it is, got %T", got)
	}
	// the other protocols get the image as it is
	if got := Raw(rasterm.Sixel, nrgba); got != image.Image(nrgba) {
		t.Errorf("expected an image as it is for sixel, got %T", got)
	}
	img, err := DecodeJPEG(encodeJPEG(t))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	switch r := Raw(rasterm.Kitty, img).(type) {
	case interface {
		rasterm.JPEGImage
		rasterm.RGBImage
	}:
		if len(r.JPEG()) == 0 || len(r.RGB()) != 8*8*3 {
			t.Error("expected the bytes of the JPEG and its pixels as RGB")
		}
	default:
		t.Errorf("expected a JPEG that is also RGB, got %T", r)
	}
	// half of the red, at half the alpha, is all of the red once the alpha
	// is not premultiplied
	rgba := image.NewRGBA(image.Rect(0, 0, 1, 1))
	rgba.Pix = []byte{0x80, 0, 0, 0x80}
	if r, ok := Raw(rasterm.Kitty, rgba).(rasterm.RGBAImage); !ok || !bytes.Equal(r.RGBA(), []byte{0xff, 0, 0, 0x80}) {
		t.Error("expected an RGBA with the alpha not premultiplied")
	}
	png, err := DecodePNG(encodePNG(t, nrgba))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !same(Raw(rasterm.Kitty, png), png) {
		t.Error("expected a PNG as it is")
	}
}

// testContext returns a context with the display size and the background.
func testContext(t *testing.T, w, h uint, bg *colors.Color) context.Context {
	t.Helper()
	c := ivctx.New()
	c.Width, c.Height = w, h
	if bg != nil {
		c.Bg = bg
	}
	return ivctx.WithConfig(t.Context(), c)
}

// same returns true when a and b are the same image. Two values of
// [rasterm.PNG] are compared by their bytes, because a struct that holds a
// slice cannot be compared with ==.
func same(a, b image.Image) bool {
	if pa, ok := a.(rasterm.PNG); ok {
		pb, ok := b.(rasterm.PNG)
		return ok && bytes.Equal(pa.Data, pb.Data) && pa.Image == pb.Image
	}
	return a == b
}

// opaqueNRGBA returns an opaque gray image of w x h.
func opaqueNRGBA(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0x80
		if i%4 == 3 {
			img.Pix[i] = 0xff
		}
	}
	return img
}

// encodePNG returns img encoded as a PNG.
func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// insertChunk returns the PNG in buf with a chunk of the type and the data
// inserted after its IHDR chunk.
func insertChunk(t *testing.T, buf []byte, typ string, data []byte) []byte {
	t.Helper()
	// the signature (8) and the IHDR chunk (25)
	const at = 8 + 25
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	chunk = append(chunk, typ...)
	chunk = append(chunk, data...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(append([]byte(typ), data...)))
	out := append(append([]byte{}, buf[:at]...), chunk...)
	return append(out, buf[at:]...)
}

// encodeJPEG returns an 8x8 image encoded as a JPEG.
func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, opaqueNRGBA(8, 8), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation returns the JPEG in buf with an APP1 segment after its start
// of image marker that holds an EXIF orientation tag.
func withOrientation(buf []byte, orientation uint16) []byte {
	// a big endian TIFF header, and one directory with one entry: the tag,
	// the type SHORT (3), the count 1 and the value
	tiff := []byte("MM\x00*\x00\x00\x00\x08\x00\x01")
	tiff = binary.BigEndian.AppendUint16(tiff, 0x0112)
	tiff = binary.BigEndian.AppendUint16(tiff, 3)
	tiff = binary.BigEndian.AppendUint32(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0, 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xff, 0xe1}
	app1 = binary.BigEndian.AppendUint16(app1, uint16(len(seg)+2))
	app1 = append(app1, seg...)
	out := append(append([]byte{}, buf[:2]...), app1...)
	return append(out, buf[2:]...)
}

// TestRawITerm checks that an opaque image that can encode itself goes to iTerm
// as its JPEG, and one that holds the bytes of a PNG as the PNG.
func TestRawITerm(t *testing.T) {
	img := &testEncoder{NRGBA: opaqueNRGBA(2, 2)}
	got := Raw(rasterm.ITerm, img)
	if _, ok := got.(rasterm.PNGImage); ok {
		t.Fatal("expected no PNG method, so that iTerm sends the JPEG")
	}
	j, ok := got.(rasterm.JPEGImage)
	if !ok || string(j.JPEG()) != "jpeg" {
		t.Fatalf("expected the JPEG of the image, got %T", got)
	}
	if img.quality != rasterm.DefaultJPEGQuality {
		t.Errorf("expected quality %d, got %d", rasterm.DefaultJPEGQuality, img.quality)
	}
	png, err := DecodePNG(encodePNG(t, opaqueNRGBA(2, 2)))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !same(Raw(rasterm.ITerm, png), png) {
		t.Error("expected a PNG as it is")
	}
}

// testEncoder is an opaque image that encodes itself as a JPEG.
type testEncoder struct {
	*image.NRGBA

	quality int
}

// Opaque returns true.
func (e *testEncoder) Opaque() bool {
	return true
}

// EncodeJPEG satisfies the [JPEGEncoder] interface.
func (e *testEncoder) EncodeJPEG(quality int) ([]byte, error) {
	e.quality = quality
	return []byte("jpeg"), nil
}
