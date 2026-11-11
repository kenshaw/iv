package blitz

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

// TestIsHTML checks that the extension decides how a document is parsed
// before the mime type, and that the content decides only when neither does.
func TestIsHTML(t *testing.T) {
	for _, test := range []struct {
		name     string
		mime     string
		pathName string
		src      string
		exp      bool
	}{
		{"markdown that starts with html", "text/html", "README.md", "<div align=\"center\">\n</div>\n\n# Title\n", false},
		{"markdown", "text/x-c", "sample.md", "# Title\n", false},
		{"html", "text/html", "sample.html", "<!doctype html>\n<html></html>\n", true},
		{"html with a markdown mime", "text/markdown", "page.htm", "<p>text</p>\n", true},
		{"html mime", "text/html", "", "<div></div>\n", true},
		{"markdown mime", "text/markdown", "", "# Title\n", false},
		{"text that starts with doctype", "text/plain", "page.txt", "  <!DOCTYPE html>\n<html></html>\n", true},
		{"plain text", "text/plain", "notes.txt", "Some notes.\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isHTML(test.mime, test.pathName, []byte(test.src)); got != test.exp {
				t.Errorf("expected %t, got %t", test.exp, got)
			}
		})
	}
}

// TestPages checks that an image taller than a page is cut into pages of the
// page height, that the rows of each page are the rows of the image, and that
// an image that fits is not cut.
func TestPages(t *testing.T) {
	// each row holds its own number, so that a page shows which rows it got
	img := image.NewRGBA(image.Rect(0, 0, 4, 10))
	for y := range 10 {
		for x := range 4 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(y), A: 255})
		}
	}
	for _, test := range []struct {
		name          string
		width, height uint
		exp           []int
	}{
		{"three pages", 4, 4, []int{4, 4, 2}},
		{"half the width", 2, 2, []int{4, 4, 2}},
		{"exact pages", 4, 5, []int{5, 5}},
		{"fits", 4, 10, nil},
		{"no display height", 4, 0, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := ivctx.New()
			c.Width, c.Height = test.width, test.height
			res := pages(ivctx.WithConfig(context.Background(), c), img)
			if test.exp == nil {
				if res != img {
					t.Fatalf("expected the image, got %T", res)
				}
				return
			}
			v, ok := res.([]image.Image)
			if !ok {
				t.Fatalf("expected pages, got %T", res)
			}
			if len(v) != len(test.exp) {
				t.Fatalf("expected %d pages, got %d", len(test.exp), len(v))
			}
			var y uint8
			for i, page := range v {
				if b := page.Bounds(); b != image.Rect(0, 0, 4, test.exp[i]) {
					t.Fatalf("page %d: expected 4x%d at the origin, got %v", i+1, test.exp[i], b)
				}
				for py := range test.exp[i] {
					if got := page.(*image.RGBA).RGBAAt(3, py).R; got != y {
						t.Fatalf("page %d row %d: expected row %d of the image, got %d", i+1, py, y, got)
					}
					y++
				}
			}
		})
	}
}
