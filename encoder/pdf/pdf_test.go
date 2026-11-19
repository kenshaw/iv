package pdf

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

// tall is an image several A4 pages long, so that pagination has something to
// divide.
func tall() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 600, 3000))
	for y := range 3000 {
		for x := range 600 {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0x40, 0xff})
		}
	}
	return img
}

// pages counts the page objects in a pdf.
func pages(t *testing.T, buf []byte) int {
	t.Helper()
	return len(regexp.MustCompile(`/Type\s*/Page[^s]`).FindAll(buf, -1))
}

// encodeWith encodes the image with the page size and margin.
func encodeWith(t *testing.T, img image.Image, page string, margin uint) []byte {
	t.Helper()
	c := ivctx.New()
	c.PDFPage, c.PDFMargin = page, margin
	var b bytes.Buffer
	if err := encode(ivctx.WithConfig(context.Background(), c), &b, img); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if b.Len() == 0 {
		t.Fatal("expected non-empty output")
	}
	if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-")) {
		t.Fatal("expected a pdf header")
	}
	return b.Bytes()
}

// TestFitIsOnePage checks the default keeps the image whole: one page its own
// size, however tall it is.
func TestFitIsOnePage(t *testing.T) {
	if n := pages(t, encodeWith(t, tall(), "fit", 36)); n != 1 {
		t.Errorf("expected fit to emit one page, got %d", n)
	}
}

// TestPaginate checks a named page size divides a tall image rather than
// emitting one unreadable page.
func TestPaginate(t *testing.T) {
	for _, size := range []string{"a4", "letter"} {
		t.Run(size, func(t *testing.T) {
			if n := pages(t, encodeWith(t, tall(), size, 36)); n < 2 {
				t.Errorf("expected a tall image to paginate, got %d page(s)", n)
			}
		})
	}
}

// TestPaginateHonorsMargin checks the margin reaches the encoder: more margin
// leaves less room per page, so the same image needs at least as many.
func TestPaginateHonorsMargin(t *testing.T) {
	narrow := pages(t, encodeWith(t, tall(), "a4", 0))
	wide := pages(t, encodeWith(t, tall(), "a4", 72))
	if wide < narrow {
		t.Errorf("expected a larger margin to need at least as many pages, got %d with 72pt against %d with 0", wide, narrow)
	}
}

// TestShortImageIsOnePage checks pagination does not split what already fits.
func TestShortImageIsOnePage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	if n := pages(t, encodeWith(t, img, "a4", 36)); n != 1 {
		t.Errorf("expected an image that fits to stay on one page, got %d", n)
	}
}

// TestCaseInsensitive checks a page size is matched however it is typed.
func TestCaseInsensitive(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for _, size := range []string{"A4", " a4 ", "Letter"} {
		if got := encodeWith(t, img, size, 36); len(got) == 0 {
			t.Errorf("expected %q to be accepted", size)
		}
	}
}

// TestUnknownPageSize checks a bad page size is reported rather than silently
// falling back.
func TestUnknownPageSize(t *testing.T) {
	c := ivctx.New()
	c.PDFPage = "foolscap"
	err := encode(ivctx.WithConfig(context.Background(), c), &bytes.Buffer{}, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, s := range []string{"foolscap", "fit", "a4", "letter"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("expected the error to mention %q, got: %v", s, err)
		}
	}
}

// TestNonRGBA checks an image that is not already an [image.RGBA] is
// converted rather than rejected.
func TestNonRGBA(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 40, 30))
	for y := range 30 {
		for x := range 40 {
			img.SetGray(x, y, color.Gray{uint8(x * y % 256)})
		}
	}
	if got := encodeWith(t, img, "fit", 36); pages(t, got) != 1 {
		t.Error("expected a gray image to encode to one page")
	}
	if rgba := toRGBA(img); rgba.Bounds().Size() != img.Bounds().Size() {
		t.Errorf("expected the conversion to keep the size, got %v", rgba.Bounds().Size())
	}
}

// writer is an image that writes itself as a pdf.
type writer struct {
	image.Image
}

// WritePDF satisfies the [Writer] interface.
func (writer) WritePDF(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, "%PDF-writer")
	return err
}

// TestSource checks that the encoder takes an image that writes itself as a
// pdf before it is prepared, and leaves any other image to encode (D55).
func TestSource(t *testing.T) {
	ctx := ivctx.WithConfig(context.Background(), ivctx.New())
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var b bytes.Buffer
	switch ok, err := source(ctx, &b, writer{img}); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case !ok || b.String() != "%PDF-writer":
		t.Errorf("expected the image to write itself, got %t %q", ok, b.String())
	}
	b.Reset()
	switch ok, err := source(ctx, &b, img); {
	case err != nil:
		t.Fatalf("expected no error, got: %v", err)
	case ok || b.Len() != 0:
		t.Errorf("expected an image to be left to encode, got %t %q", ok, b.String())
	}
}
