package heic

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

// TestPage checks that --page selects a top-level image of a file, and that
// no page, or a page past the last image, gives the primary image (D58).
func TestPage(t *testing.T) {
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "heic", "cyberpunk.heic"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	for _, test := range []struct {
		page uint
		size image.Point
	}{
		{0, image.Pt(5000, 2256)},
		{1, image.Pt(5000, 2256)},
		{3, image.Pt(728, 403)},
		{12, image.Pt(728, 488)},
		{13, image.Pt(5000, 2256)},
	} {
		c := ivctx.New()
		c.Page = test.page
		res, err := decode(ivctx.WithConfig(context.Background(), c), buf)
		if err != nil {
			t.Fatalf("page %d: expected no error, got: %v", test.page, err)
		}
		img, ok := res.(image.Image)
		if !ok {
			t.Fatalf("page %d: expected an image, got %T", test.page, res)
		}
		if got := img.Bounds().Size(); got != test.size {
			t.Errorf("page %d: expected %v, got %v", test.page, test.size, got)
		}
	}
}
