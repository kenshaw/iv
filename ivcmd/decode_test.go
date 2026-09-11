package ivcmd

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestDecodeFile decodes the test data through the full pipeline, checking
// that each file reaches the decoder it is supposed to and produces a usable
// image.
func TestDecodeFile(t *testing.T) {
	for _, test := range []struct {
		name string
		file string
		mime string
		cmd  string // external command the decoder needs
	}{
		{"png", "png/rose.png", "image/png", ""},
		{"bmp", "bmp/rose.bmp", "image/bmp", ""},
		{"webp lossless", "webp/rose-lossless.webp", "image/webp", ""},
		{"webp lossy", "webp/rose-lossy.webp", "image/webp", ""},
		{"jpeg", "jpeg/precision.jpg", "image/jpeg", ""},
		{"svg", "resvg/rect.svg", "image/svg+xml", ""},
		{"svg choropleth", "resvg/choropleth.svg", "image/svg+xml", ""},
		{"ico", "ico/1.ico", "image/x-icon", ""},
		{"ico multi", "ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico", "image/x-icon", ""},
		{"dot", "graphviz/booktest_sqlite3.dot", "text/vnd.graphviz", ""},
		{"ttf", "fontimg/Ubuntu-R.ttf", "font/ttf", ""},
		{"jxl", "vips/precision.jxl", "image/jxl", ""},
		{"heic", "vips/cyberpunk.heic", "image/heic", ""},
		{"pdf", "vips/file-sample_150kB.pdf", "application/pdf", ""},
		{"xps", "fitz/example.xps", "application/zip", ""},
		{"windows pe", "winres/go-winres.exe", "application/vnd.microsoft.portable-executable", ""},
		{"markdown", "markdown/sample.md", "text/plain", ""},
		{"mermaid", "mermaid/gantt.mmd", "text/plain", "mmdc"},
		{"video", "ffmpeg/sample_960x540.mp4", "video/mp4", "ffmpeg"},
		{"binwalk", "binwalk/icon.afdesign", "application/octet-stream", "binwalk"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.cmd != "" {
				if _, err := exec.LookPath(test.cmd); err != nil {
					t.Skipf("%s not in path", test.cmd)
				}
			}
			pathName := filepath.Join("..", "testdata", filepath.FromSlash(test.file))
			if _, err := os.Stat(pathName); err != nil {
				t.Skipf("no test data: %v", err)
			}
			ctx := testContext(t)
			// the mime type must be detected before any decoder runs
			f, err := os.Open(pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			defer f.Close()
			mime, err := decoder.Detect(ivctx.WithPathName(ctx, pathName), f)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if mime != test.mime {
				t.Errorf("expected mime %q, got %q", test.mime, mime)
			}
			img, _, err := decoder.DecodeFile(ctx, pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			assertImage(t, img)
		})
	}
}

func TestDecodeString(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
	}{
		{"wifi", "WIFI:S:testssid;T:WPA;P:secret;;"},
		{"data svg base64", "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxMDAiIGhlaWdodD0iMTAwIj48Y2lyY2xlIGN4PSI1MCIgY3k9IjUwIiByPSI0MCIgc3Ryb2tlPSJncmVlbiIgc3Ryb2tlLXdpZHRoPSI0IiBmaWxsPSJ5ZWxsb3ciIC8+PC9zdmc+"},
		{"data svg escaped", "data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A//www.w3.org/2000/svg%22%20width%3D%2264px%22%20height%3D%2264px%22%3E%3Crect%20fill%3D%22%2350c848%22%20width%3D%2264%22%20height%3D%2264%22/%3E%3C/svg%3E"},
	} {
		t.Run(test.name, func(t *testing.T) {
			img, _, err := decoder.DecodeString(testContext(t), test.in)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			assertImage(t, img)
		})
	}
}

// TestDecodeComicArchive builds a cbz from the test images and decodes each of
// its pages.
func TestDecodeComicArchive(t *testing.T) {
	names := []string{"png/rose.png", "png/logo.png", "png/card.png"}
	cbz := filepath.Join(t.TempDir(), "test.cbz")
	f, err := os.Create(cbz)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for i, name := range names {
		buf, err := os.ReadFile(filepath.Join("..", "testdata", filepath.FromSlash(name)))
		if err != nil {
			t.Skipf("no test data: %v", err)
		}
		w, err := zw.Create(string(rune('a'+i)) + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(buf); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var sizes []image.Point
	for page := range uint(len(names)) {
		c := ivctx.New()
		c.Page = page + 1
		img, _, err := decoder.DecodeFile(ivctx.WithConfig(context.Background(), c), cbz)
		if err != nil {
			t.Fatalf("page %d: expected no error, got: %v", page+1, err)
		}
		assertImage(t, img)
		sizes = append(sizes, img.Bounds().Size())
	}
	if sizes[0] == sizes[1] && sizes[1] == sizes[2] {
		t.Error("expected the pages to differ; page selection may not be applied")
	}
}

// TestRoundTrip decodes an image and re-encodes it with every registered
// encoder, checking the output is non-empty and, where the format can be read
// back, that it decodes to the same size.
func TestRoundTrip(t *testing.T) {
	ctx := testContext(t)
	img, _, err := decoder.DecodeFile(ctx, filepath.Join("..", "testdata", "png", "rose.png"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, e := range encoder.All() {
		if e.Term {
			continue
		}
		t.Run(e.Name, func(t *testing.T) {
			var buf bytes.Buffer
			switch err := e.Encode(ctx, &buf, img); {
			case errors.Is(err, encoder.ErrUnsupportedFormat):
				t.Skipf("this build cannot encode %s: %v", e.Ext, err)
			case err != nil:
				t.Fatalf("expected no error, got: %v", err)
			}
			if buf.Len() == 0 {
				t.Fatal("expected non-empty output")
			}
			out, _, err := decoder.Decode(ctx, "", e.Ext, bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("expected the encoded image to decode, got: %v", err)
			}
			if got, exp := out.Bounds().Size(), img.Bounds().Size(); got != exp {
				t.Errorf("expected size %v, got %v", exp, got)
			}
		})
	}
}

// testContext returns a context wired to the test log.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	c.Logger = func(s string, v ...any) {
		t.Logf(s, v...)
	}
	return ivctx.WithConfig(context.Background(), c)
}

// assertImage fails the test when the image is missing or empty.
func assertImage(t *testing.T, img image.Image) {
	t.Helper()
	if img == nil {
		t.Fatal("expected an image")
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("expected a non-empty image, got %v", b)
	}
}
