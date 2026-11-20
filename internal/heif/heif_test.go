package heif

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gen2brain/h265/heic"
)

// TestParse checks the top-level images of each heic file against what
// libheif 1.21 lists for it with heif-info.
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		primary uint32
		images  []uint32
	}{
		{"cyberpunk.heic", 51, []uint32{51, 53, 54, 95, 144, 199, 216, 277, 310, 327, 368, 369}},
		{"dome.heic", 145, []uint32{145, 291}},
		{"alpha.heic", 1, []uint32{1}},
		{"grid_alpha.heic", 7, []uint32{7}},
		{"sample1.heif", 1, []uint32{1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(readFile(t, test.name))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if f.Primary != test.primary {
				t.Errorf("expected primary %d, got %d", test.primary, f.Primary)
			}
			if !slices.Equal(f.Images, test.images) {
				t.Errorf("expected images %v, got %v", test.images, f.Images)
			}
		})
	}
}

// TestWithPrimary checks that the decoder decodes the image that the pitm box
// names, and that a tile of a grid cannot be named.
func TestWithPrimary(t *testing.T) {
	f, err := Parse(readFile(t, "cyberpunk.heic"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	buf, err := f.WithPrimary(54)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	img, err := heic.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := img.Bounds().Size(); got.X != 728 || got.Y != 403 {
		t.Errorf("expected 728x403, got %v", got)
	}
	// item 60 is a tile of the grid of the primary image
	if _, err := f.WithPrimary(60); err == nil {
		t.Error("expected an error for a tile")
	}
}

// readFile reads a file of testdata/heic.
func readFile(t *testing.T, name string) []byte {
	t.Helper()
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "heic", name))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	return buf
}
