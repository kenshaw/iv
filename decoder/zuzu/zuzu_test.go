package zuzu

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/zuzu"
)

// TestDetect checks that a graph or a diagram is recognized by its content,
// and that Markdown and YAML are not.
func TestDetect(t *testing.T) {
	for _, test := range []struct {
		name string
		src  string
		exp  string
	}{
		{"digraph", "digraph G {\n  a -> b;\n}\n", dotMime},
		{"graph", "graph G {\n  a -- b;\n}\n", dotMime},
		{"strict digraph", "strict digraph {\n}\n", dotMime},
		{"leading blank lines", "\n\n  digraph G {\n}\n", dotMime},
		{"flowchart", "flowchart LR\n  a --> b\n", mermaidMime},
		{"mermaid graph", "graph TD\n  a --> b\n", mermaidMime},
		{"mermaid comment", "%% a comment\nsequenceDiagram\n  a->>b: hi\n", mermaidMime},
		{"front matter", "---\ntitle: A diagram\n---\nflowchart LR\n  a --> b\n", ""},
		{"markdown front matter", "---\ntitle: A page\n---\n# A page\n", ""},
		{"yaml", "---\nkey: value\n", ""},
		{"prose about graphs", "This digraph G is described below.\n", ""},
		{"markdown", "# A graph\n\nSome text.\n", ""},
		{"vega lite", `{"$schema": "https://vega.github.io/schema/vega-lite/v6.json"}`, ""},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := detect(context.Background(), strings.NewReader(test.src))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != test.exp {
				t.Errorf("expected mime %q, got %q", test.exp, got)
			}
		})
	}
}

// TestDetectVega checks that a file whose name ends in .vg.json is claimed as
// a Vega spec, and in .vl.json as a Vega-Lite spec, and that a json file with
// another name is not, even with the $schema of Vega.
func TestDetectVega(t *testing.T) {
	const spec = `{"$schema": "https://vega.github.io/schema/vega/v5.json"}`
	for _, test := range []struct {
		name string
		exp  string
	}{
		{"chart.vg.json", vegaMime},
		{"dir/CHART.VG.JSON", vegaMime},
		{"chart.vl.json", vegaLiteMime},
		{"dir/CHART.VL.JSON", vegaLiteMime},
		{"chart.json", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := detect(ivctx.WithPathName(context.Background(), test.name), strings.NewReader(spec))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != test.exp {
				t.Errorf("expected mime %q, got %q", test.exp, got)
			}
		})
	}
}

// TestDecodeVega draws the Vega and the Vega-Lite spec in testdata/zuzu,
// which read their data from data/sales.csv, a directory under them (D44,
// D46).
func TestDecodeVega(t *testing.T) {
	for _, name := range vegaSpecs {
		t.Run(name, func(t *testing.T) {
			img, err := decodeSpec(t, filepath.Join("..", "..", "testdata", "zuzu", name), true, "")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if b := img.Bounds(); b.Dx() <= 0 || b.Dy() <= 0 {
				t.Errorf("expected a drawn image, got %v", b)
			}
		})
	}
}

// vegaSpecs are the Vega and the Vega-Lite spec in testdata/zuzu. Each reads
// its data with "url": "data/sales.csv".
var vegaSpecs = []string{"sales.vg.json", "sales.vl.json"}

// TestVegaData checks that a Vega spec reads data only from its directory and
// the directories under it (D44), or from the directory of --zuzu-data and
// the directories under it (D45). A path that leaves the directory, by .. or
// by a symbolic link, and a url with a scheme are refused. Without
// --zuzu-data, any url of a spec that is not a file on disk is refused. A
// Vega-Lite spec reads its data the same way (D46).
func TestVegaData(t *testing.T) {
	for _, spec := range vegaSpecs {
		t.Run(spec, func(t *testing.T) {
			testVegaData(t, spec)
		})
	}
}

// testVegaData runs the cases of [TestVegaData] for the spec in
// testdata/zuzu.
func testVegaData(t *testing.T, spec string) {
	t.Helper()
	dir := t.TempDir()
	csv, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zuzu", "data", "sales.csv"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zuzu", spec))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	specDir := filepath.Join(dir, "spec")
	for _, name := range []string{filepath.Join(specDir, "data", "in.csv"), filepath.Join(dir, "outside.csv")} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, csv, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dataDir := filepath.Join(specDir, "data")
	type vegaTest struct {
		name string
		url  string
		file bool
		data string
		exp  bool
	}
	tests := []vegaTest{
		{"subdirectory", "data/in.csv", true, "", true},
		{"leading slash", "/data/in.csv", true, "", true},
		{"parent", "../outside.csv", true, "", false},
		{"scheme", "https://example.com/sales.csv", true, "", false},
		{"not a file", "data/in.csv", false, "", false},
		{"zuzu data", "in.csv", true, dataDir, true},
		{"zuzu data parent", "../data/in.csv", true, dataDir, false},
		{"zuzu data not a file", "in.csv", false, dataDir, true},
		{"zuzu data missing", "in.csv", true, filepath.Join(dir, "missing"), false},
	}
	// windows needs a privilege to make a symbolic link
	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join("..", "outside.csv"), filepath.Join(specDir, "link.csv")); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, vegaTest{"link out", "link.csv", true, "", false})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := filepath.Join(specDir, strings.ReplaceAll(test.name, " ", "-")+filepath.Ext(strings.TrimSuffix(spec, ".json"))+".json")
			spec := bytes.Replace(src, []byte(`"url": "data/sales.csv"`), []byte(`"url": "`+test.url+`"`), 1)
			if err := os.WriteFile(name, spec, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := decodeSpec(t, name, test.file, test.data)
			switch {
			case test.exp && err != nil:
				t.Errorf("expected no error, got: %v", err)
			case !test.exp && err == nil:
				t.Errorf("expected %q refused", test.url)
			case !test.file && test.data == "" && !errors.Is(err, zuzu.ErrNoLoader):
				t.Errorf("expected %v, got: %v", zuzu.ErrNoLoader, err)
			}
		})
	}
}

// decodeSpec draws the Vega spec at name, with data as --zuzu-data. When file
// is false, the spec is not a file on disk, as a file in an archive is not.
func decodeSpec(t *testing.T, name string, file bool, data string) (*image.RGBA, error) {
	t.Helper()
	buf, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	ctx := testContext(t)
	ivctx.Get(ctx).ZuzuData = data
	ctx = ivctx.WithPathName(ctx, name)
	if file {
		ctx = ivctx.WithFile(ctx, name)
	}
	res, err := decode(ctx, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	img, ok := res.(*image.RGBA)
	if !ok {
		t.Fatalf("expected an *image.RGBA, got %T", res)
	}
	return img, nil
}

// TestDecode draws every graph and diagram in testdata/zuzu as an image.
func TestDecode(t *testing.T) {
	for _, name := range testFiles(t) {
		t.Run(name, func(t *testing.T) {
			b := renderFile(t, testContext(t), name).Bounds()
			if b.Dx() <= 0 || b.Dy() <= 0 {
				t.Errorf("expected a drawn image, got %v", b)
			}
		})
	}
}

// testFiles returns the names of the graphs and diagrams in testdata/zuzu.
func testFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("..", "..", "testdata", "zuzu"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && slices.Contains(exts, ivctx.FileExt(entry.Name())) {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("no test data in testdata/zuzu")
	}
	return names
}

// TestDecodeSyntaxError checks that a diagram that does not parse comes back
// as an error, and not as an empty image.
func TestDecodeSyntaxError(t *testing.T) {
	ctx := testContext(t)
	if _, err := decode(ctx, strings.NewReader("this is not a diagram")); err == nil {
		t.Fatal("expected an error")
	}
}

// TestDecodeTheme checks that the theme reaches zuzu, and that an unknown
// one is an error that names it.
func TestDecodeTheme(t *testing.T) {
	ctx := testContext(t)
	def := renderFile(t, ctx, "flowchart.mmd")
	ivctx.Get(ctx).MermaidTheme = "dark"
	if dark := renderFile(t, ctx, "flowchart.mmd"); bytes.Equal(def.Pix, dark.Pix) {
		t.Error("expected the dark theme to draw differently from the default")
	}
	ivctx.Get(ctx).MermaidTheme = "chartreuse"
	if _, err := renderFileErr(t, ctx, "flowchart.mmd"); err == nil {
		t.Error("expected an error for an unknown theme")
	} else if !strings.Contains(err.Error(), "chartreuse") {
		t.Errorf("expected the error to name the theme, got: %v", err)
	}
}

// TestDecodeBackground checks that --mermaid-bg sets the background of a
// diagram that zuzu draws, and leaves a DOT graph alone.
func TestDecodeBackground(t *testing.T) {
	red := colors.New(255, 0, 0, 255)
	for _, test := range []struct {
		name string
		exp  color.RGBA
	}{
		{"flowchart.mmd", color.RGBA{R: 255, A: 255}},
		{"booktest_sqlite3.dot", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := testContext(t)
			ivctx.Get(ctx).MermaidBg = &red
			if got := renderFile(t, ctx, test.name).RGBAAt(0, 0); got != test.exp {
				t.Errorf("expected the corner to be %v, got %v", test.exp, got)
			}
		})
	}
}

// TestDecodeEmoji checks that an emoji draws, in the font of Noto Color
// Emoji. zuzu reports a character that no font has, and draws nothing for it.
func TestDecodeEmoji(t *testing.T) {
	ctx := testContext(t)
	var missing []error
	ivctx.Get(ctx).Logger = func(_ string, v ...any) {
		for _, arg := range v {
			if err, ok := arg.(error); ok && errors.Is(err, zuzu.ErrNoGlyph) {
				missing = append(missing, err)
			}
		}
	}
	renderFile(t, ctx, "emoji.mmd")
	for _, err := range missing {
		t.Errorf("expected every emoji to draw, got: %v", err)
	}
}

// renderFile draws a test file on the context.
func renderFile(t *testing.T, ctx context.Context, name string) *image.RGBA {
	t.Helper()
	img, err := renderFileErr(t, ctx, name)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// renderFileErr draws a test file, and returns the image or the error of the
// decoder.
func renderFileErr(t *testing.T, ctx context.Context, name string) (*image.RGBA, error) {
	t.Helper()
	pathName := filepath.Join("..", "..", "testdata", "zuzu", name)
	buf, err := os.ReadFile(pathName)
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	res, err := decode(ivctx.WithPathName(ctx, pathName), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	img, ok := res.(*image.RGBA)
	if !ok {
		t.Fatalf("expected an *image.RGBA, got %T", res)
	}
	return img, nil
}

// testContext returns a context with a configuration of its own, so that a
// test that changes it does not change another test.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	if testing.Verbose() {
		c.Logger = func(s string, v ...any) { t.Logf(s, v...) }
	}
	return ivctx.WithConfig(context.Background(), c)
}
