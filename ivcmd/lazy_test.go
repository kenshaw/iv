package ivcmd

import (
	"bytes"
	"image"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
)

// TestEncodeAfterDecode checks that each file that libvips decodes still
// encodes after the pipeline closed the file.
//
// libvips decodes the pixels only when an encoder asks for them (D22). It
// loaded from a stream of the file once, and a JPEG XL then failed in the
// encoder, with a warning and a transparent image, because the file was
// closed. The PNG of the encoder must show the image.
func TestEncodeAfterDecode(t *testing.T) {
	enc, ok := encoder.Get("png")
	if !ok {
		t.Fatal("expected the png encoder")
	}
	names, err := filepath.Glob(filepath.Join("..", "testdata", "vips", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		t.Run(filepath.Base(name), func(t *testing.T) {
			ctx := testContext(t)
			if strings.Contains(name, ".enc.") {
				ivctx.Get(ctx).Password = encPassword
			}
			img, mime, err := decoder.DecodeFile(ctx, name)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			defer ivimg.Close(img)
			prepared, err := ivimg.Prepare(ctx, mime, img)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			defer ivimg.Close(prepared)
			var buf bytes.Buffer
			if err := enc.Encode(ctx, &buf, prepared); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			out, err := png.Decode(&buf)
			if err != nil {
				t.Fatalf("expected a png, got: %v", err)
			}
			if blank(out) {
				t.Error("expected the image, got a blank one")
			}
		})
	}
}

// blank returns true when every pixel of img is the same.
func blank(img image.Image) bool {
	b := img.Bounds()
	first := img.At(b.Min.X, b.Min.Y)
	for y := b.Min.Y; y < b.Max.Y; y += max(1, b.Dy()/64) {
		for x := b.Min.X; x < b.Max.X; x += max(1, b.Dx()/64) {
			if img.At(x, y) != first {
				return false
			}
		}
	}
	return true
}
