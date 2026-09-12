package vips

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/ivctx"
)

// TestRoundTrip checks a Go image survives a trip through libvips and back
// unchanged, pixel for pixel.
//
// This is what the memory conversion has to earn: it replaced a png encode and
// decode on each side, which were slow but obviously lossless. A band in the
// wrong order, a premultiplied alpha, or a row stride read as pixels would all
// show up here and nowhere else.
func TestRoundTrip(t *testing.T) {
	ctx := testContext(t)
	for _, name := range []string{
		"png/tux.png",  // alpha, including partly transparent edges
		"png/rose.png", // opaque photographic
		"png/precision.png",
	} {
		t.Run(name, func(t *testing.T) {
			want := loadNRGBA(t, name)
			assertSame(t, want, roundTrip(t, ctx, want), 0)
		})
	}
}

// TestImportConverts checks the images that are not already an NRGBA are
// converted rather than handed over as whatever memory they happen to hold --
// an [image.RGBA] is premultiplied, which libvips would read as unpremultiplied
// and wash out every translucent pixel.
func TestImportConverts(t *testing.T) {
	ctx := testContext(t)
	src := loadNRGBA(t, "png/tux.png")
	// the same picture, premultiplied
	rgba := image.NewRGBA(src.Bounds())
	for y := range src.Bounds().Dy() {
		for x := range src.Bounds().Dx() {
			rgba.Set(x, y, src.At(x, y))
		}
	}
	got := roundTrip(t, ctx, rgba)
	// an [image.RGBA] stores its colors premultiplied, which is lossy in
	// itself, so the trip out through unpremultiplied samples and back cannot
	// land exactly. What is being checked is that the conversion happened at
	// all: reading premultiplied samples as unpremultiplied would wash every
	// translucent pixel out, not move it by a level or two.
	assertSame(t, rgba, got, 0x200)
}

// TestImportSubImage checks a sub image is repacked. Its rows are spaced by
// the width of the image it was cut from, which libvips reads as pixels.
func TestImportSubImage(t *testing.T) {
	ctx := testContext(t)
	full := loadNRGBA(t, "png/tux.png")
	b := full.Bounds()
	sub := full.SubImage(image.Rect(b.Min.X+10, b.Min.Y+10, b.Min.X+40, b.Min.Y+40))
	got := roundTrip(t, ctx, sub)
	if size := got.Bounds().Size(); size != (image.Point{X: 30, Y: 30}) {
		t.Fatalf("expected 30x30, got %v", size)
	}
	assertSame(t, sub, got, 0)
}

// TestExportNoAlpha checks an image libvips holds without an alpha band comes
// back opaque rather than transparent -- the band count is what says whether
// there is one, and reading it wrong makes every pixel vanish.
func TestExportNoAlpha(t *testing.T) {
	ctx := testContext(t)
	Init(ctx)
	// three bands of colour and nothing else, which is what an opaque jpeg or
	// heic arrives as
	const w, h = 4, 3
	pix := make([]byte, w*h*3)
	for i := range pix {
		pix[i] = byte(i * 7)
	}
	v, err := vips.NewImageFromMemory(pix, w, h, 3)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	got, err := Export(ctx, v)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if size := got.Bounds().Size(); size != (image.Point{X: w, Y: h}) {
		t.Fatalf("expected %dx%d, got %v", w, h, size)
	}
	for y := range h {
		for x := range w {
			i := (y*w + x) * 3
			r, g, b, a := got.At(x, y).RGBA()
			exp := [4]uint32{uint32(pix[i]) * 257, uint32(pix[i+1]) * 257, uint32(pix[i+2]) * 257, 0xffff}
			if got := [4]uint32{r, g, b, a}; got != exp {
				t.Errorf("pixel (%d,%d): expected %v, got %v", x, y, exp, got)
			}
		}
	}
}

// roundTrip sends a Go image through libvips and back, as a lossless png so
// that any difference in the result is the conversion's and not the format's.
//
// Going out through [Save] is the point: the pixels of an NRGBA reach libvips
// by reference, and it is the encode inside Save that reads them. A round trip
// that stopped at the image would never touch the memory that matters.
func roundTrip(t *testing.T, ctx context.Context, img image.Image) image.Image {
	t.Helper()
	buf, err := Save(ctx, img, func(v *vips.Image) ([]byte, error) {
		return v.PngsaveBuffer(nil)
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	v, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	got, err := Export(ctx, v)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return got
}

// assertSame checks two images hold the same pixels, comparing the
// premultiplied samples within tol -- which is where the error of a conversion
// stays bounded whatever the alpha, unlike the unpremultiplied ones, where a
// nearly invisible pixel can swing the whole range.
func assertSame(t *testing.T, want, got image.Image, tol uint32) {
	t.Helper()
	wb, gb := want.Bounds(), got.Bounds()
	if wb.Dx() != gb.Dx() || wb.Dy() != gb.Dy() {
		t.Fatalf("expected %v, got %v", wb.Size(), gb.Size())
	}
	near := func(a, b uint32) bool {
		if a > b {
			a, b = b, a
		}
		return b-a <= tol
	}
	for y := range wb.Dy() {
		for x := range wb.Dx() {
			w, g := want.At(wb.Min.X+x, wb.Min.Y+y), got.At(gb.Min.X+x, gb.Min.Y+y)
			wr, wg, wbl, wa := w.RGBA()
			gr, gg, gbl, ga := g.RGBA()
			if !near(wr, gr) || !near(wg, gg) || !near(wbl, gbl) || !near(wa, ga) {
				t.Fatalf("pixel (%d,%d): expected %v, got %v", x, y, w, g)
			}
		}
	}
}

// loadNRGBA reads a test image.
func loadNRGBA(t *testing.T, name string) *image.NRGBA {
	t.Helper()
	pathName := filepath.Join("..", "testdata", filepath.FromSlash(name))
	f, err := os.Open(pathName)
	if err != nil {
		t.Skipf("no test data: %v", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return nrgba(img)
}

// testContext returns a context carrying a config of its own.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	if testing.Verbose() {
		c.Logger = func(s string, v ...any) { t.Logf(s, v...) }
	}
	return ivctx.WithConfig(context.Background(), c)
}
