package vips

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/ivimg"
	"github.com/kenshaw/rasterm"
)

// TestImage checks that an [Image] resizes, composites and encodes in
// libvips, and that none of it changes the image it started from.
func TestImage(t *testing.T) {
	ctx := testContext(t)
	img := openTestdata(t, "png/tux.png")
	defer img.Close()
	b := img.Bounds()
	if img.Opaque() {
		t.Error("expected an image with an alpha band to be not opaque")
	}
	// resize
	resized, err := img.Resize(ctx, b.Dx()/2, b.Dy()/2)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer ivimg.Close(resized)
	if size := resized.Bounds().Size(); size != (image.Point{X: b.Dx() / 2, Y: b.Dy() / 2}) {
		t.Errorf("expected %dx%d, got %v", b.Dx()/2, b.Dy()/2, size)
	}
	if _, ok := resized.(*Image); !ok {
		t.Errorf("expected the resized image to stay in libvips, got %T", resized)
	}
	if img.Bounds() != b {
		t.Error("expected the image that was resized to keep its size")
	}
	// flatten
	flat, err := img.Flatten(ctx, 255, 0, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer ivimg.Close(flat)
	if !ivimg.Opaque(flat) {
		t.Error("expected a flattened image to be opaque")
	}
	// the corner of tux.png is transparent, so it is the background now
	if r, g, b, a := flat.At(0, 0).RGBA(); r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Errorf("expected the background in the corner, got %v", flat.At(0, 0))
	}
	// png
	data := resized.(*Image).PNG()
	if len(data) == 0 {
		t.Fatal("expected a PNG of the image")
	}
	dec, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if dec.Bounds() != resized.Bounds() {
		t.Errorf("expected a PNG of %v, got %v", resized.Bounds(), dec.Bounds())
	}
	// kitty gets the PNG, and sixel the pixels
	if _, ok := ivimg.Raw(rasterm.Kitty, resized).(rasterm.PNGImage); !ok {
		t.Error("expected the Kitty encoder to get the PNG of libvips")
	}
	// iTerm gets the JPEG of an opaque image
	j, ok := ivimg.Raw(rasterm.ITerm, flat).(rasterm.JPEGImage)
	if !ok {
		t.Fatal("expected the iTerm encoder to get a JPEG of an opaque image")
	}
	if data := j.JPEG(); len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		t.Error("expected the JPEG of libvips")
	}
	if _, ok := ivimg.Raw(rasterm.ITerm, resized).(rasterm.PNGImage); !ok {
		t.Error("expected the iTerm encoder to get the PNG of an image with alpha")
	}
	if _, ok := ivimg.Unwrap(resized).(*image.NRGBA); !ok {
		t.Errorf("expected the pixels as an NRGBA, got %T", ivimg.Unwrap(resized))
	}
}

// TestSaveFromImage checks that a file encoder saves an [Image] from the
// image that libvips holds.
func TestSaveFromImage(t *testing.T) {
	ctx := testContext(t)
	img := openTestdata(t, "png/rose.png")
	defer img.Close()
	var saved *vips.Image
	if _, err := Save(ctx, img, func(v *vips.Image) ([]byte, error) {
		saved = v
		return v.PngsaveBuffer(nil)
	}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if saved != img.Vips() {
		t.Error("expected the save to get the image that libvips holds")
	}
}

// openTestdata opens the file in testdata with libvips.
func openTestdata(t *testing.T, name string) *Image {
	t.Helper()
	ctx := testContext(t)
	Init(ctx)
	buf, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	v, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	img, err := Open(ctx, v)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}
