package ivcmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
)

// benchWidth and benchHeight are the display size of the benchmarks, in
// pixels. It is about the size of a full screen terminal on a 1920x1200
// display. An image that is larger is scaled down to it, and one that fits is
// left at its own size.
const (
	benchWidth  = 1920
	benchHeight = 1200
)

// benchTargets are the targets that the benchmarks render, one or two for
// each decoder. The libreoffice and calibre decoders are not here, because
// one run of soffice or ebook-convert takes a second or more. A target that
// a version cannot decode is skipped, which is how the same file runs against
// an earlier release.
var benchTargets = []string{
	"archives/science-preview.cbz",
	"binwalk/icon.afdesign",
	"blitz/sample.md",
	"bmp/rose.bmp",
	"ffmpeg/sample_960x540.mp4",
	"fontimg/Ubuntu-R.ttf",
	"gif/tux.gif",
	"graphviz/booktest_sqlite3.dot",
	"gxps/example.xps",
	"icns/test.icns",
	"ico/1.ico",
	"jpeg/precision.jpg",
	"lottie/rocket.json",
	"mermaid/flowchart.mmd",
	"nativewebp/dice-lossless.webp",
	"netpbm/tux.ppm",
	"png/dice.png",
	"png/precision.png",
	"psd/alpha.psd",
	"resvg/choropleth.svg",
	"strings/data-png-base64.iv_test_string",
	"strings/wifi.iv_test_string",
	"tag/silent.mp3",
	"tiff/tux-deflate.tiff",
	"vcard/john-doe.vcf",
	"vips/dome.heic",
	"vips/file-sample_150kB.pdf",
	"vips/precision.avif",
	"vips/precision.jxl",
	"winres/go-winres.exe",
}

// BenchmarkDecode measures the decoders: from the file to a decoded image,
// with nothing scaled or encoded.
func BenchmarkDecode(b *testing.B) {
	ctx := benchContext(b)
	for _, name := range benchTargets {
		t := benchTarget(b, name)
		b.Run(benchName(name), func(b *testing.B) {
			decode := func() error {
				var err error
				if t.IsString {
					_, _, err = decoder.DecodeString(ctx, t.Name)
				} else {
					_, _, err = decoder.DecodeFile(ctx, t.Name)
				}
				return err
			}
			if err := decode(); err != nil {
				b.Skipf("cannot decode: %v", err)
			}
			b.ResetTimer()
			for b.Loop() {
				if err := decode(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRender measures what the command does for each target: it
// decodes, scales to the display size and encodes as terminal graphics.
//
// The protocol is the one that TERM_GRAPHICS selects. Run the benchmark once
// for each of kitty, iterm and sixel. The output is counted and discarded, so
// the time does not include the time that a terminal takes to show the image.
// The out-B/op metric is the size of the output, which is what the terminal
// must read.
func BenchmarkRender(b *testing.B) {
	ctx := benchContext(b)
	enc, ok := encoder.Get(DefaultEncoder)
	if !ok {
		b.Fatalf("encoder %q: not registered", DefaultEncoder)
	}
	args := &Args{Quiet: true}
	for _, name := range benchTargets {
		t := benchTarget(b, name)
		b.Run(benchName(name), func(b *testing.B) {
			var w counter
			if err := args.render(ctx, &w, enc, t); err != nil {
				b.Skipf("cannot render: %v", err)
			}
			size := w.n
			b.ResetTimer()
			for b.Loop() {
				if err := args.render(ctx, io.Discard, enc, t); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(size), "out-B/op")
		})
	}
}

// benchContext returns the context of a benchmark. It holds the default
// configuration of the command, with the display size of the benchmarks.
//
// The decoders are not closed at the end of a benchmark. A closed decoder
// cannot start again, and the benchmarks after it need the same engines.
func benchContext(b *testing.B) context.Context {
	b.Helper()
	c := ivctx.New()
	c.Width, c.Height = benchWidth, benchHeight
	return ivctx.WithConfig(context.Background(), c)
}

// benchTarget returns the target for the file in testdata. A file in
// testdata/strings holds an argument that is not a file.
func benchTarget(b *testing.B, name string) Target {
	b.Helper()
	pathName := filepath.Join("..", "testdata", filepath.FromSlash(name))
	if !strings.HasPrefix(name, "strings/") {
		return Target{Name: pathName}
	}
	buf, err := os.ReadFile(pathName)
	if err != nil {
		b.Fatal(err)
	}
	return Target{Name: strings.TrimSuffix(string(buf), "\n"), IsString: true}
}

// benchName returns the name of the sub-benchmark for the file: its path in
// testdata, with each character that a benchmark name cannot hold changed.
func benchName(name string) string {
	return strings.NewReplacer("/", "_", " ", "_").Replace(name)
}

// counter is a writer that counts the bytes written to it and discards them.
type counter struct {
	n int64
}

// Write satisfies the [io.Writer] interface.
func (c *counter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}
