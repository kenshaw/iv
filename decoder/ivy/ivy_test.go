package ivy

import (
	"bytes"
	"context"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage"
)

// TestDecode draws each file in testdata/ivy with the decoder that the
// pipeline routes it to, and checks that it is an ivy decoder (D49).
func TestDecode(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "testdata", "ivy"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			ctx := testContext(t)
			pathName := filepath.Join("..", "..", "testdata", "ivy", name)
			img, mime, err := decoder.DecodeFile(ctx, pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			d := decoder.Match(ctx, mime, ivctx.FileExt(name))
			if len(d) == 0 || !isIvy(d[0].Name) {
				t.Errorf("expected an ivy decoder first, got %v", names(d))
			}
			if b := img.Bounds(); b.Dx() < 16 || b.Dy() < 16 {
				t.Errorf("expected an image, got %v", b)
			}
		})
	}
}

// TestPage checks that --page selects a page of a document with pages, and
// that a page past the last one is the first.
func TestPage(t *testing.T) {
	var pages [][]byte
	for _, page := range []uint{1, 2, 99} {
		ctx := testContext(t)
		ivctx.Get(ctx).Page = page
		img := testDecode(t, ctx, "sample.epub")
		pages = append(pages, rgba(img))
	}
	switch {
	case bytes.Equal(pages[0], pages[1]):
		t.Error("expected page 2 to differ from page 1")
	case !bytes.Equal(pages[0], pages[2]):
		t.Error("expected page 99 to be page 1")
	}
}

// TestPageDPI checks that a page draws at --dpi, or at the lower resolution
// that draws it at the size that it is shown at.
func TestPageDPI(t *testing.T) {
	// an A4 page, 595 x 842 points, is 2480 x 3509 pixels at 300 dpi
	page := testPage{595, 842}
	for _, test := range []struct {
		name          string
		width, height uint
		exp           float32
	}{
		{"no display", 0, 0, 300},
		{"display larger than the page", 4000, 4000, 300},
		{"display half the height", 4000, 1754, 150},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := ivctx.New()
			c.Width, c.Height = test.width, test.height
			dpi := pageDPI(ivctx.WithConfig(t.Context(), c), page)
			if dpi < test.exp-1 || dpi > test.exp+1 {
				t.Errorf("expected about %g dpi, got %g", test.exp, dpi)
			}
		})
	}
}

// TestRouting checks the rules of routing: an Office 97 to 2003 file, which
// libmagic types as a compound file, goes by its extension, and text that
// libmagic takes for csv is a table only with the extension of a table, or
// with no extension.
func TestRouting(t *testing.T) {
	ctx := testContext(t)
	for _, test := range []struct {
		mime, ext string
		exp       string
	}{
		{"application/x-ole-storage", "doc", "ivy-msdoc"},
		{"application/x-ole-storage", "ppt", "ivy-msppt"},
		{"application/x-ole-storage", "xls", "ivy-msxls"},
		{"text/csv", "csv", "ivy-csv"},
		{"text/csv", "", "ivy-csv"},
		{"text/csv", "md", ""},
		{"text/plain", "txt", ""},
	} {
		t.Run(test.mime+" "+test.ext, func(t *testing.T) {
			got := names(decoder.Match(ctx, test.mime, test.ext))
			switch {
			case test.exp == "" && slices.ContainsFunc(got, isIvy):
				t.Errorf("expected no ivy decoder, got %v", got)
			case test.exp != "" && (len(got) == 0 || got[0] != test.exp):
				t.Errorf("expected %s first, got %v", test.exp, got)
			}
		})
	}
}

// testPage is a page of a size, for pageDPI.
type testPage struct {
	w, h float32
}

// Size satisfies the ivy.Page interface.
func (p testPage) Size() (float32, float32) {
	return p.w, p.h
}

// Render satisfies the ivy.Page interface.
func (p testPage) Render(context.Context, *okage.Pixmap, okage.Transform) error {
	return nil
}

// testDecode decodes the file in testdata/ivy.
func testDecode(t *testing.T, ctx context.Context, name string) image.Image {
	t.Helper()
	img, _, err := decoder.DecodeFile(ctx, filepath.Join("..", "..", "testdata", "ivy", name))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// rgba returns the pixels of img, to compare.
func rgba(img image.Image) []byte {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, img.At(x, y))
		}
	}
	return dst.Pix
}

// names returns the names of the decoders.
func names(d []*decoder.Entry) []string {
	v := make([]string, len(d))
	for i, e := range d {
		v[i] = e.Name
	}
	return v
}

// isIvy returns true for the name of an ivy decoder.
func isIvy(name string) bool {
	return len(name) > 4 && name[:4] == "ivy-"
}

// testContext returns the context of a test, with the default configuration
// of iv.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	if testing.Verbose() {
		c.Logger = func(s string, v ...any) { t.Logf(s, v...) }
	}
	return ivctx.WithConfig(t.Context(), c)
}
