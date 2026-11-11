package zuzu

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/ivctx"
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

// TestDecode draws every file in the test data as an image.
func TestDecode(t *testing.T) {
	for _, name := range []string{
		"booktest_sqlite3.dot",
		"flowchart.mmd",
		"pie.mmd",
		"gantt.mmd",
		"journey.mmd",
		"sankey.mmd",
		"aws.mmd",
	} {
		t.Run(name, func(t *testing.T) {
			b := renderFile(t, testContext(t), name).Bounds()
			if b.Dx() <= 0 || b.Dy() <= 0 {
				t.Errorf("expected a drawn image, got %v", b)
			}
		})
	}
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
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "zuzu", name))
	if err != nil {
		t.Fatalf("no test data: %v", err)
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
