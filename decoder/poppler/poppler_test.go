package poppler

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestDecode checks that a plain and an encrypted pdf render, and that a page
// out of range falls back to the first page. The pipeline tries this decoder
// only where libvips has no pdf loader, so the test calls it directly.
func TestDecode(t *testing.T) {
	for _, test := range []struct {
		name string
		file string
		pass string
		page uint
	}{
		{"plain", "file-sample_150kB.pdf", "", 0},
		{"encrypted", "file-sample_150kB.enc.pdf", "password", 0},
		{"page out of range", "file-sample_150kB.pdf", "", 99},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := filepath.Join("..", "..", "testdata", "vips", test.file)
			f, err := os.Open(name)
			if err != nil {
				t.Fatalf("no test data: %v", err)
			}
			defer f.Close()
			c := ivctx.New()
			c.DPI, c.Password, c.Page = 72, test.pass, test.page
			ctx := ivctx.WithPathName(ivctx.WithConfig(context.Background(), c), name)
			res, err := decode(ctx, f)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			img, ok := res.(*decoder.Image)
			if !ok || img.Mime != "image/png" {
				t.Fatalf("expected a png for the pipeline, got %T", res)
			}
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(img.Reader); err != nil {
				t.Fatal(err)
			}
			out, err := png.Decode(&buf)
			if err != nil {
				t.Fatalf("expected a png, got: %v", err)
			}
			// an A4 page at 72 dpi
			if b := out.Bounds(); b.Dx() < 590 || b.Dx() > 600 {
				t.Errorf("expected a page about 595 wide, got %v", b.Size())
			}
		})
	}
}
