package ivcmd

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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
		{"webp lossless", "nativewebp/rose-lossless.webp", "image/webp", ""},
		{"webp lossy", "nativewebp/rose-lossy.webp", "image/webp", ""},
		{"jpeg", "jpeg/precision.jpg", "image/jpeg", ""},
		{"gif", "gif/tux.gif", "image/gif", ""},
		{"gif animated", "gif/animated.gif", "image/gif", ""},
		{"tiff uncompressed", "tiff/tux-uncompressed.tiff", "image/tiff", ""},
		{"tiff deflate", "tiff/tux-deflate.tiff", "image/tiff", ""},
		{"tiff deflate predictor", "tiff/test-deflate-predictor.tiff", "image/tiff", ""},
		{"netpbm pbm", "netpbm/test.pbm", "image/x-portable-bitmap", ""},
		{"netpbm pgm", "netpbm/test.pgm", "image/x-portable-graymap", ""},
		{"netpbm ppm", "netpbm/tux.ppm", "image/x-portable-pixmap", ""},
		{"netpbm pam", "netpbm/tux.pam", "image/x-portable-arbitrarymap", ""},
		{"netpbm pbm plain", "netpbm/test-plain.pbm", "image/x-portable-bitmap", ""},
		{"netpbm pgm plain", "netpbm/test-plain.pgm", "image/x-portable-graymap", ""},
		{"netpbm ppm plain", "netpbm/tux-plain.ppm", "image/x-portable-pixmap", ""},
		{"svg", "resvg/rect.svg", "image/svg+xml", ""},
		{"svg choropleth", "resvg/choropleth.svg", "image/svg+xml", ""},
		{"ico", "ico/1.ico", "image/x-icon", ""},
		{"ico multi", "ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico", "image/x-icon", ""},
		{"icns", "icns/test.icns", "image/x-icns", ""},
		{"dot", "graphviz/booktest_sqlite3.dot", "text/vnd.graphviz", ""},
		{"ttf", "fontimg/Ubuntu-R.ttf", "font/ttf", ""},
		{"jxl", "vips/precision.jxl", "image/jxl", ""},
		{"heic", "vips/cyberpunk.heic", "image/heic", ""},
		{"pdf", "vips/file-sample_150kB.pdf", "application/pdf", ""},
		{"xps", "fitz/example.xps", "application/zip", ""},
		{"windows pe", "winres/go-winres.exe", "application/vnd.microsoft.portable-executable", ""},
		{"markdown", "markdown/sample.md", "text/plain", ""},
		{"tag mp3", "tag/silent.mp3", "audio/mpeg", ""},
		{"tag flac", "tag/silent.flac", "audio/flac", ""},
		{"tag m4a", "tag/silent.m4a", "audio/x-m4a", ""},
		{"tag ogg", "tag/silent.ogg", "audio/ogg", ""},
		{"tag aac", "tag/silent.aac", "audio/mpeg", ""},
		{"docx", "libreoffice/file-sample_100kB.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "soffice"},
		{"doc", "libreoffice/file-sample_100kB.doc", "application/x-ole-storage", "soffice"},
		{"odt", "libreoffice/file-sample_100kB.odt", "application/vnd.oasis.opendocument.text", "soffice"},
		{"mermaid", "mermaid/gantt.mmd", "text/plain", "mmdc"},
		{"video", "ffmpeg/sample_960x540.mp4", "video/mp4", "ffmpeg"},
		{"binwalk", "binwalk/icon.afdesign", "application/octet-stream", "binwalk"},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			// detection is iv's own work and always runs; only the decode
			// needs the external tool
			if test.cmd != "" {
				if err := usable(test.cmd); err != nil {
					t.Skipf("skipping decode: %v", err)
				}
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

// TestDecodeTagArt checks that the album art extracted from each audio
// container is the cover that was embedded, pixel for pixel -- the decoder
// hands the picture back to the pipeline rather than decoding it itself, so
// this covers the round trip through mime detection as well.
func TestDecodeTagArt(t *testing.T) {
	want, err := loadImage(t, filepath.Join("..", "testdata", "png", "tux.png"))
	if err != nil {
		t.Skipf("no cover art: %v", err)
	}
	for _, name := range []string{
		"silent.mp3",
		"silent.flac",
		"silent.m4a",
		"silent.ogg",
		"silent.aac",
	} {
		t.Run(name, func(t *testing.T) {
			pathName := filepath.Join("..", "testdata", "tag", name)
			if _, err := os.Stat(pathName); err != nil {
				t.Skipf("no test data: %v", err)
			}
			got, mime, err := decoder.DecodeFile(testContext(t), pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			// the pipeline reports the mime of the picture, not the container
			if mime != "image/png" {
				t.Errorf("expected mime %q, got %q", "image/png", mime)
			}
			if err := sameImage(got, want); err != nil {
				t.Error(err)
			}
		})
	}
}

// loadImage decodes an image from a file.
func loadImage(t *testing.T, pathName string) (image.Image, error) {
	t.Helper()
	f, err := os.Open(pathName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// sameImage reports whether two images have the same bounds and pixels.
func sameImage(got, want image.Image) error {
	gb, wb := got.Bounds(), want.Bounds()
	if gb.Size() != wb.Size() {
		return fmt.Errorf("expected size %v, got %v", wb.Size(), gb.Size())
	}
	for y := range wb.Dy() {
		for x := range wb.Dx() {
			g, w := got.At(gb.Min.X+x, gb.Min.Y+y), want.At(wb.Min.X+x, wb.Min.Y+y)
			gr, gg, gbl, ga := g.RGBA()
			wr, wg, wbl, wa := w.RGBA()
			if gr != wr || gg != wg || gbl != wbl || ga != wa {
				return fmt.Errorf("pixel (%d,%d): expected %v, got %v", x, y, w, g)
			}
		}
	}
	return nil
}

// TestLibreOfficeRouting checks that every office document in the test data
// is detected and routed to the libreoffice decoder. Routing is the part iv
// owns; the conversion itself is soffice's job, so this needs no soffice.
func TestLibreOfficeRouting(t *testing.T) {
	for _, test := range []struct {
		file string
		mime string
	}{
		{"file-sample_100kB.doc", "application/x-ole-storage"},
		{"file-sample_100kB.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"file-sample_100kB.odt", "application/vnd.oasis.opendocument.text"},
		{"file-sample_100kB.rtf", "text/rtf"},
		{"file_example_XLS_50.xls", "application/x-ole-storage"},
		{"file_example_XLSX_50.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"file_example_ODS_100.ods", "application/vnd.oasis.opendocument.spreadsheet"},
		{"file_example_ODP_200kB.odp", "application/vnd.oasis.opendocument.presentation"},
		{"file_example_PPT_250kB.ppt", "application/x-ole-storage"},
	} {
		t.Run(test.file, func(t *testing.T) {
			pathName := filepath.Join("..", "testdata", "libreoffice", test.file)
			f, err := os.Open(pathName)
			if err != nil {
				t.Skipf("no test data: %v", err)
			}
			defer f.Close()
			ctx := ivctx.WithPathName(testContext(t), pathName)
			mime, err := decoder.Detect(ctx, f)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if mime != test.mime {
				t.Errorf("expected mime %q, got %q", test.mime, mime)
			}
			matched := decoder.Match(ctx, mime, ivctx.FileExt(pathName))
			if len(matched) == 0 || matched[0].Name != "libreoffice" {
				var names []string
				for _, d := range matched {
					names = append(names, d.Name)
				}
				t.Errorf("expected the libreoffice decoder, got %v", names)
			}
		})
	}
}

// TestDecodeIcnsResolutions checks that every resolution in a multi
// resolution icns is decoded, and that the page selects between them.
func TestDecodeIcnsResolutions(t *testing.T) {
	pathName := filepath.Join("..", "testdata", "icns", "test.icns")
	// the entries an icns holds, largest first -- 512 and 256 appear twice,
	// once as a size and once as the @2x variant of the size below it
	exp := []int{1024, 512, 512, 256, 256, 128, 64, 32}
	for i, size := range exp {
		page := uint(i + 1)
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			c := ivctx.New()
			c.Page = page
			img, mime, err := decoder.DecodeFile(ivctx.WithConfig(context.Background(), c), pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if mime != "image/x-icns" {
				t.Errorf("expected mime %q, got %q", "image/x-icns", mime)
			}
			if got, want := img.Bounds().Size(), (image.Point{X: size, Y: size}); got != want {
				t.Errorf("page %d: expected %v, got %v", page, want, got)
			}
		})
	}
	// no page, and a page past the end, both give the first entry
	for _, page := range []uint{0, uint(len(exp)) + 1} {
		c := ivctx.New()
		c.Page = page
		img, _, err := decoder.DecodeFile(ivctx.WithConfig(context.Background(), c), pathName)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got := img.Bounds().Dx(); got != exp[0] {
			t.Errorf("page %d: expected %d, got %d", page, exp[0], got)
		}
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

// usable reports whether an external command the decoders shell out to can
// actually be used.
func usable(name string) error {
	if name == "soffice" {
		return sofficeUsable()
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not in path", name)
	}
	return nil
}

// sofficeUsable reports whether soffice can convert a document. Being in
// $PATH is not enough: LibreOffice also needs a writable user profile and
// somewhere to put its named pipe, neither of which every sandbox and CI
// image provides. Probed once, with a document small enough that the cost is
// one process start.
var sofficeUsable = sync.OnceValue(func() error {
	if _, err := exec.LookPath("soffice"); err != nil {
		return fmt.Errorf("soffice not in path")
	}
	dir, err := os.MkdirTemp("", "iv-soffice-probe.")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "probe.csv")
	if err := os.WriteFile(src, []byte("a,b\n1,2\n"), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", dir, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("soffice cannot run here: %v: %s", err, bytes.TrimSpace(out))
	}
	if _, err := os.Stat(filepath.Join(dir, "probe.pdf")); err != nil {
		return fmt.Errorf("soffice produced no pdf: %s", bytes.TrimSpace(out))
	}
	return nil
})

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
