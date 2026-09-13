package ivcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/decoder/blitz"
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
		dec  string // decoder the file must route to
		cmd  string // external command the decoder needs
	}{
		{"png", "png/rose.png", "image/png", "png", ""},
		{"bmp", "bmp/rose.bmp", "image/bmp", "bmp", ""},
		{"webp lossless", "nativewebp/rose-lossless.webp", "image/webp", "nativewebp", ""},
		{"webp lossy", "nativewebp/rose-lossy.webp", "image/webp", "nativewebp", ""},
		{"jpeg", "jpeg/precision.jpg", "image/jpeg", "jpeg", ""},
		{"gif", "gif/tux.gif", "image/gif", "gif", ""},
		{"gif animated", "gif/animated.gif", "image/gif", "gif", ""},
		{"tiff uncompressed", "tiff/tux-uncompressed.tiff", "image/tiff", "tiff", ""},
		{"tiff deflate", "tiff/tux-deflate.tiff", "image/tiff", "tiff", ""},
		{"tiff deflate predictor", "tiff/test-deflate-predictor.tiff", "image/tiff", "tiff", ""},
		{"netpbm pbm", "netpbm/test.pbm", "image/x-portable-bitmap", "netpbm", ""},
		{"netpbm pgm", "netpbm/test.pgm", "image/x-portable-graymap", "netpbm", ""},
		{"netpbm ppm", "netpbm/tux.ppm", "image/x-portable-pixmap", "netpbm", ""},
		{"netpbm pam", "netpbm/tux.pam", "image/x-portable-arbitrarymap", "netpbm", ""},
		{"netpbm pbm plain", "netpbm/test-plain.pbm", "image/x-portable-bitmap", "netpbm", ""},
		{"netpbm pgm plain", "netpbm/test-plain.pgm", "image/x-portable-graymap", "netpbm", ""},
		{"netpbm ppm plain", "netpbm/tux-plain.ppm", "image/x-portable-pixmap", "netpbm", ""},
		{"svg", "resvg/rect.svg", "image/svg+xml", "resvg", ""},
		{"svg choropleth", "resvg/choropleth.svg", "image/svg+xml", "resvg", ""},
		{"svgz", "resvg/rect.svgz", "application/gzip", "resvg", ""},
		{"ico", "ico/1.ico", "image/x-icon", "ico", ""},
		{"ico multi", "ico/Mathijssen-Tuxlets-Test-Dummy-Tux.ico", "image/x-icon", "ico", ""},
		{"icns", "icns/test.icns", "image/x-icns", "icns", ""},
		{"dot", "graphviz/booktest_sqlite3.dot", "text/vnd.graphviz", "graphviz", ""},
		{"ttf", "fontimg/Ubuntu-R.ttf", "font/ttf", "fontimg", ""},
		{"jxl", "vips/precision.jxl", "image/jxl", "vips", ""},
		{"heic", "vips/cyberpunk.heic", "image/heic", "vips", ""},
		{"pdf", "vips/file-sample_150kB.pdf", "application/pdf", "vips-pdf", ""},
		{"pdf encrypted", "vips/file-sample_150kB.enc.pdf", "application/pdf", "vips-pdf", ""},
		{"xps", "fitz/example.xps", "application/zip", "fitz", ""},
		{"windows pe", "winres/go-winres.exe", "application/vnd.microsoft.portable-executable", "winres", ""},
		{"markdown", "blitz/sample.md", "text/plain", "blitz", ""},
		{"html", "blitz/sample.html", "text/html", "blitz", ""},
		{"tag mp3", "tag/silent.mp3", "audio/mpeg", "tag", ""},
		{"tag flac", "tag/silent.flac", "audio/flac", "tag", ""},
		{"tag m4a", "tag/silent.m4a", "audio/x-m4a", "tag", ""},
		{"tag ogg", "tag/silent.ogg", "audio/ogg", "tag", ""},
		{"tag aac", "tag/silent.aac", "audio/mpeg", "tag", ""},
		{"docx", "libreoffice/file-sample_100kB.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "libreoffice", "soffice"},
		{"doc", "libreoffice/file-sample_100kB.doc", "application/x-ole-storage", "libreoffice", "soffice"},
		{"odt", "libreoffice/file-sample_100kB.odt", "application/vnd.oasis.opendocument.text", "libreoffice", "soffice"},
		{"mermaid", "mermaid/gantt.mmd", "text/plain", "mermaid", "mmdc"},
		{"video", "ffmpeg/sample_960x540.mp4", "video/mp4", "ffmpeg", "ffmpeg"},
		{"cbz", "archives/science-preview.cbz", "application/zip", "archives", ""},
		{"cbr", "archives/science-preview.cbr", "application/vnd.rar", "archives", ""},
		{"cbt", "archives/science-preview.cbt", "application/x-tar", "archives", ""},
		{"lottie", "lottie/rocket.json", "video/lottie+json", "lottie", ""},
		{"lottie lot", "lottie/star.lot", "video/lottie+json", "lottie", ""},
		{"dotlottie", "lottie/fire.lottie", "application/zip", "lottie", ""},
		{"binwalk", "binwalk/icon.afdesign", "application/octet-stream", "binwalk", "binwalk"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pathName := filepath.Join("..", "testdata", filepath.FromSlash(test.file))
			if _, err := os.Stat(pathName); err != nil {
				t.Skipf("no test data: %v", err)
			}
			ctx := testContext(t)
			if strings.Contains(test.file, ".enc.") {
				// without this the decoder prompts, and a test has no
				// terminal to prompt at
				ivctx.Get(ctx).Password = encPassword
			}
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
			// routing is iv's own work: check the file reaches the decoder
			// it is supposed to, not merely that something decoded it
			matched := decoder.Match(ctx, mime, ivctx.FileExt(pathName))
			var names []string
			for _, d := range matched {
				names = append(names, d.Name)
			}
			if len(matched) == 0 || matched[0].Name != test.dec {
				t.Errorf("expected the %s decoder, got %v", test.dec, names)
			}
			// detection and routing are iv's own work and always run; only
			// the decode needs the external tool
			if test.cmd != "" {
				if err := usable(test.cmd); err != nil {
					t.Skipf("skipping decode: %v", err)
				}
			}
			img, _, err := decoder.DecodeFile(ctx, pathName)
			switch {
			case errors.Is(err, decoder.ErrUnsupportedFormat):
				t.Skipf("this build cannot decode %s: %v", test.name, err)
			case err != nil:
				t.Fatalf("expected no error, got: %v", err)
			}
			assertImage(t, img)
		})
	}
}

// StringExt is the extension of the files in testdata/strings. Each holds a
// single command line argument that iv accepts in place of a file -- a data:
// URL, a WIFI: code -- with a trailing newline so they stay ordinary text
// files.
const StringExt = ".iv_test_string"

// TestDecodeString decodes every string in testdata/strings. The expectations
// are keyed by file name and checked for completeness, so a string added
// without being described here fails rather than going quietly untested.
func TestDecodeString(t *testing.T) {
	// A zero in size means the dimension is not checked: a rendered web page
	// is only as tall as whatever the site served that minute, and the width
	// is the one part of it iv decides.
	exp := map[string]struct {
		mime string
		size image.Point
	}{
		// a QR code is built outright, so it has no mime type of its own
		"wifi":                {"", image.Pt(350, 350)},
		"data-svg-base64":     {"image/svg+xml", image.Pt(100, 100)},
		"data-svg-urlencoded": {"image/svg+xml", image.Pt(64, 64)},
		"data-png-base64":     {"image/png", image.Pt(1, 1)},
		// a page blitz renders is handed back as an image, so like the QR
		// code it arrives without a mime type
		"yahoo":          {"", image.Pt(pageWidth, 0)},
		"ifconfig-me":    {"", image.Pt(pageWidth, 0)},
		"finance-google": {"", image.Pt(pageWidth, 0)},
		// a url naming an image is still that image: this one is not
		// rendered as a page but handed back to the pipeline, and the mime is
		// what says so -- a rendered page arrives without one
		"microsoft-favicon": {"image/vnd.microsoft.icon", image.Pt(0, 0)},
	}
	dir := filepath.Join("..", "testdata", "strings")
	names, err := filepath.Glob(filepath.Join(dir, "*"+StringExt))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(names) == 0 {
		t.Fatalf("no %s files in %s", StringExt, dir)
	}
	seen := make(map[string]bool, len(names))
	for _, pathName := range names {
		name := strings.TrimSuffix(filepath.Base(pathName), StringExt)
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			want, ok := exp[name]
			if !ok {
				t.Fatalf("%s is not described in the test table", filepath.Base(pathName))
			}
			s, err := readString(pathName)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			img, mime, err := decoder.DecodeString(testContext(t), s)
			switch {
			case errors.Is(err, blitz.ErrFetch):
				// the network, or the site, rather than iv
				t.Skipf("skipping decode: %v", err)
			case err != nil:
				t.Fatalf("expected no error, got: %v", err)
			}
			if mime != want.mime {
				t.Errorf("expected mime %q, got %q", want.mime, mime)
			}
			got := img.Bounds().Size()
			if want.size.X != 0 && got.X != want.size.X {
				t.Errorf("expected width %d, got %d", want.size.X, got.X)
			}
			switch {
			case want.size.Y != 0 && got.Y != want.size.Y:
				t.Errorf("expected height %d, got %d", want.size.Y, got.Y)
			case want.size.Y == 0 && got.Y <= 0:
				t.Errorf("expected a positive height, got %d", got.Y)
			}
		})
	}
	for name := range exp {
		if !seen[name] {
			t.Errorf("%s%s is described but missing from %s", name, StringExt, dir)
		}
	}
}

func TestDecodeStringUnsupported(t *testing.T) {
	for _, s := range []string{
		"ftp://example.com/a.png",
		"not a url at all",
		"",
	} {
		if _, _, err := decoder.DecodeString(testContext(t), s); !errors.Is(err, decoder.ErrNotSupported) {
			t.Errorf("%q: expected ErrNotSupported, got: %v", s, err)
		}
	}
}

// readString reads a string argument from a testdata/strings file. The
// trailing newline is not part of the argument.
func readString(pathName string) (string, error) {
	buf, err := os.ReadFile(pathName)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(buf), "\r\n"), nil
}

// TestDecodeTagCard checks that every audio container renders to the card the
// tag decoder draws -- an svg the decoder hands back to the pipeline, which
// resvg rasterizes, so this covers the round trip through mime detection as
// well. That the card carries the right cover art is checked in the tag
// package, which can read the svg before it is rasterized.
// encPassword opens testdata/vips/file-sample_150kB.enc.pdf.
const encPassword = "password"

const (
	cardWidth  = 1000
	cardHeight = 448
	// pageWidth is the width blitz renders a document at: its viewport width
	// times its scale.
	pageWidth = 2400
)

func TestDecodeTagCard(t *testing.T) {
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
			// the pipeline reports the mime of the card, not the container
			if mime != "image/svg+xml" {
				t.Errorf("expected mime %q, got %q", "image/svg+xml", mime)
			}
			if exp := image.Pt(cardWidth, cardHeight); got.Bounds().Size() != exp {
				t.Errorf("expected card %v, got %v", exp, got.Bounds().Size())
			}
		})
	}
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

// TestDecodeComicArchive decodes the same two page comic from each archive
// format. Nothing but the extension distinguishes a cbz from any other zip,
// so routing is checked as well as the decode.
func TestDecodeComicArchive(t *testing.T) {
	// the archives all hold the same two pages
	sizes := []image.Point{{X: 400, Y: 605}, {X: 400, Y: 604}}
	formats := []struct {
		file string
		mime string
	}{
		{"science-preview.cbz", "application/zip"},
		{"science-preview.cbr", "application/vnd.rar"},
		{"science-preview.cbt", "application/x-tar"},
	}
	ctx := testContext(t)
	for _, f := range formats {
		matched := decoder.Match(ctx, f.mime, ivctx.FileExt(f.file))
		if len(matched) == 0 || matched[0].Name != "archives" {
			var got []string
			for _, d := range matched {
				got = append(got, d.Name)
			}
			t.Errorf("%s: expected the archives decoder, got %v", f.file, got)
		}
	}
	// pages[page][format]
	pages := make([][]image.Image, len(sizes))
	for page := range sizes {
		pages[page] = make([]image.Image, len(formats))
		for i, f := range formats {
			pathName := filepath.Join("..", "testdata", "archives", f.file)
			if _, err := os.Stat(pathName); err != nil {
				t.Skipf("no test data: %v", err)
			}
			c := ivctx.New()
			c.Page = uint(page + 1)
			img, _, err := decoder.DecodeFile(ivctx.WithConfig(context.Background(), c), pathName)
			if err != nil {
				t.Fatalf("%s page %d: expected no error, got: %v", f.file, page+1, err)
			}
			if got := img.Bounds().Size(); got != sizes[page] {
				t.Errorf("%s page %d: expected size %v, got %v", f.file, page+1, sizes[page], got)
			}
			pages[page][i] = img
		}
	}
	// the three formats hold the same images, so each page must come out
	// identical whichever container it was read from
	for page := range pages {
		for i := 1; i < len(formats); i++ {
			if err := sameImage(pages[page][i], pages[page][0]); err != nil {
				t.Errorf("page %d: %s differs from %s: %v", page+1, formats[i].file, formats[0].file, err)
			}
		}
	}
	// and the pages themselves must differ, or the page was never applied
	if err := sameImage(pages[1][0], pages[0][0]); err == nil {
		t.Error("expected the two pages to differ; page selection may not be applied")
	}
	// a page past the end falls back to the first
	c := ivctx.New()
	c.Page = uint(len(sizes)) + 1
	img, _, err := decoder.DecodeFile(ivctx.WithConfig(context.Background(), c),
		filepath.Join("..", "testdata", "archives", "science-preview.cbz"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := img.Bounds().Size(); got != sizes[0] {
		t.Errorf("expected size %v, got %v", sizes[0], got)
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
			switch {
			case errors.Is(err, decoder.ErrUnsupportedFormat):
				// this build can write the format but not read it back --
				// msys2's libvips has no pdf loader, for one
				t.Skipf("this build cannot decode %s: %v", e.Ext, err)
			case err != nil:
				t.Fatalf("expected the encoded image to decode, got: %v", err)
			}
			got, exp := out.Bounds().Size(), img.Bounds().Size()
			if e.Ext == "pdf" {
				// a pdf carries a page, not a raster: what comes back is
				// whatever the reader rasterized it at, so the shape is what
				// survives the trip rather than the pixel count
				if a, b := aspect(got), aspect(exp); math.Abs(a-b) > 0.01*b {
					t.Errorf("expected aspect %.4f, got %.4f (%v from %v)", b, a, got, exp)
				}
				return
			}
			if got != exp {
				t.Errorf("expected size %v, got %v", exp, got)
			}
		})
	}
}

// aspect returns the width over height of a size.
func aspect(p image.Point) float64 {
	return float64(p.X) / float64(p.Y)
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
