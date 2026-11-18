package ivy

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/ivy"
	"github.com/xo/okage"
)

// encPassword opens testdata/ivy/file-sample_150kB.enc.pdf.
const encPassword = "password"

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
			answer(t)
			if strings.Contains(name, ".enc.") {
				ivctx.Get(ctx).Password = encPassword
			}
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

// TestPaper checks that a spreadsheet whose file gives no orientation is
// drawn in landscape, and that a sheet whose file gives portrait, and any
// other document, is drawn in portrait (D51).
func TestPaper(t *testing.T) {
	tests := []struct {
		name      string
		landscape bool
	}{
		{"sales.csv", true},
		{"spreadsheet.xlsx", true},
		// the file gives A4 in portrait
		{"file_example_XLSX_50.xlsx", false},
		{"file_example_ODS_100.ods", false},
		{"file-sample_100kB.docx", false},
		{"file-sample_100kB.odt", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := testDecode(t, testContext(t), test.name).Bounds()
			if landscape := b.Dx() > b.Dy(); landscape != test.landscape {
				t.Errorf("expected landscape %t, got %v", test.landscape, b.Size())
			}
		})
	}
}

// TestIsODS checks that an OpenDocument spreadsheet is told from the other
// OpenDocument files by the first entry of its zip.
func TestIsODS(t *testing.T) {
	tests := []struct {
		name string
		exp  bool
	}{
		{"file_example_ODS_100.ods", true},
		{"file-sample_100kB.odt", false},
		{"file_example_ODP_200kB.odp", false},
		{"file_example_XLSX_50.xlsx", false},
		{"sales.csv", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ivy", test.name))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got := isODS(buf); got != test.exp {
				t.Errorf("expected %t, got %t", test.exp, got)
			}
			// a file cut short is not a spreadsheet
			for _, n := range []int{0, 29, 37, 40} {
				if isODS(buf[:min(n, len(buf))]) {
					t.Errorf("expected false for the first %d bytes", n)
				}
			}
		})
	}
}

// TestPassword checks that a pdf that a password protects opens with
// --password, or with a password that the decoder asks for when --password
// is missing or wrong, and that the decoder asks at most
// maxPasswordAttempts times (D50).
func TestPassword(t *testing.T) {
	for _, test := range []struct {
		name     string
		password string
		answers  []string
		asked    int
		exp      error
	}{
		{"with password", encPassword, nil, 0, nil},
		{"asked", "", []string{"wrong", encPassword}, 2, nil},
		{"wrong password", "wrong", []string{encPassword}, 1, nil},
		{"too many attempts", "", []string{"a", "b", "c", encPassword}, 3, ErrPasswordAttempts},
		{"no terminal", "", nil, 1, ErrNoTerminal},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := testContext(t)
			ivctx.Get(ctx).Password = test.password
			asked := answer(t, test.answers...)
			img, _, err := decoder.DecodeFile(ctx, filepath.Join("..", "..", "testdata", "ivy", "file-sample_150kB.enc.pdf"))
			switch {
			case *asked != test.asked:
				t.Errorf("expected to be asked %d times, got %d", test.asked, *asked)
			case test.exp == nil && err != nil:
				t.Errorf("expected no error, got: %v", err)
			case test.exp == nil && img.Bounds().Dx() < 16:
				t.Errorf("expected an image, got %v", img.Bounds())
			case test.exp != nil && (!errors.Is(err, test.exp) || !errors.Is(err, ivy.ErrPassword)):
				t.Errorf("expected %v and %v, got: %v", test.exp, ivy.ErrPassword, err)
			}
		})
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

// answer makes the decoder read the answers in order when it asks for a
// password, and then return ErrNoTerminal, and returns the number of times
// that it asked. A test never reads a password at the terminal.
func answer(t *testing.T, answers ...string) *int {
	t.Helper()
	asked, prev := new(int), readPassword
	readPassword = func() (string, error) {
		*asked++
		if *asked > len(answers) {
			return "", ErrNoTerminal
		}
		return answers[*asked-1], nil
	}
	t.Cleanup(func() { readPassword = prev })
	return asked
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
