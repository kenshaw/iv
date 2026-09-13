package mermaid

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	// a diagram is rendered to svg and handed back to the pipeline, so the
	// svg decoder has to be in the binary for a decode to finish
	_ "github.com/kenshaw/iv/decoder/resvg"
	"github.com/kenshaw/iv/ivctx"
)

// TestDecode renders every diagram in the test data, checking each one reaches
// the pipeline as an svg carrying the labels it was given.
//
// The corpus is one file per diagram type: what is being covered is the
// renderer understanding each grammar, which is the part that differs between
// them.
func TestDecode(t *testing.T) {
	for _, test := range []struct {
		name   string
		labels []string
	}{
		{"flowchart", []string{"Start", "Great success", "Debug it"}},
		{"sequence", []string{"Alice", "John"}},
		{"class", []string{"Animal", "Duck", "beakColor"}},
		{"state", []string{"Still", "Moving", "Crash"}},
		{"pie", []string{"Dogs", "Cats", "Rats"}},
		{"er", []string{"CUSTOMER", "ORDER"}},
		{"gantt", []string{"A Gantt Diagram"}},
		{"journey", []string{"My working day", "Make tea"}},
		{"mindmap", []string{"mindmap", "Origins"}},
		{"architecture", []string{"API", "Database"}},
		{"sankey", nil},
		{"aws", []string{"Route 53", "CloudFront"}},
		{"aws-architecture", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			svg := renderFile(t, test.name+".mmd")
			if !strings.Contains(svg, "<svg") {
				t.Fatalf("expected an svg document, got %.80q", svg)
			}
			for _, label := range test.labels {
				if !strings.Contains(svg, label) {
					t.Errorf("expected the svg to carry the label %q", label)
				}
			}
		})
	}
}

// TestDecodeSyntaxError checks a diagram that does not parse comes back as an
// error rather than an empty document. The renderer's own diagnostic names the
// offending token, so it is worth passing through untouched.
func TestDecodeSyntaxError(t *testing.T) {
	ctx := testContext(t)
	rend := newRenderer(t, ctx)
	_, err := render(ctx, rend, strings.NewReader("this is not a diagram"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "this is not a diagram") {
		t.Errorf("expected the error to name the offending source, got: %v", err)
	}
}

// TestDecodeTheme checks the configured theme reaches the renderer, and that
// an unknown one is reported rather than quietly ignored.
func TestDecodeTheme(t *testing.T) {
	ctx := testContext(t)
	def := renderFile(t, "flowchart.mmd")
	ivctx.Get(ctx).MermaidTheme = "dark"
	dark := renderFileCtx(t, ctx, "flowchart.mmd")
	if def == dark {
		t.Error("expected the dark theme to render differently from the default")
	}
	ivctx.Get(ctx).MermaidTheme = "chartreuse"
	if _, err := renderFileErr(t, ctx, "flowchart.mmd"); err == nil {
		t.Error("expected an error for an unknown theme")
	} else if !strings.Contains(err.Error(), "chartreuse") {
		t.Errorf("expected the error to name the theme, got: %v", err)
	}
}

// render decodes a test diagram with a config of its own.
func renderFile(t *testing.T, name string) string {
	t.Helper()
	return renderFileCtx(t, testContext(t), name)
}

// renderCtx decodes a test diagram on the given context.
func renderFileCtx(t *testing.T, ctx context.Context, name string) string {
	t.Helper()
	svg, err := renderFileErr(t, ctx, name)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return svg
}

// renderErr decodes a test diagram, returning whatever the decoder said.
func renderFileErr(t *testing.T, ctx context.Context, name string) (string, error) {
	t.Helper()
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mermaid", name))
	if err != nil {
		t.Skipf("no test data: %v", err)
	}
	res, err := render(ctx, newRenderer(t, ctx), bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	img, ok := res.(*decoder.Image)
	if !ok {
		t.Fatalf("expected a *decoder.Image, got %T", res)
	}
	if img.Mime != "image/svg+xml" {
		t.Errorf("expected mime %q, got %q", "image/svg+xml", img.Mime)
	}
	out := new(bytes.Buffer)
	if _, err := out.ReadFrom(img.Reader); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return out.String(), nil
}

// newRenderer builds a renderer for a test, closing it when the test is done.
func newRenderer(t *testing.T, ctx context.Context) *renderer {
	t.Helper()
	state, err := open(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	rend, ok := state.(*renderer)
	if !ok {
		t.Fatalf("expected a *renderer, got %T", state)
	}
	t.Cleanup(func() {
		if err := rend.Renderer.Close(ctx); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if rend.cache != nil {
			rend.cache.Close(ctx)
		}
	})
	return rend
}

// testContext returns a context carrying a config of its own, so that a test
// changing one does not change another's.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	if testing.Verbose() {
		c.Logger = func(s string, v ...any) { t.Logf(s, v...) }
	}
	return ivctx.WithConfig(context.Background(), c)
}

// TestMain releases the renderer the way iv does once the tests are done, and
// checks what was released was the real one -- a decoder handed no state
// reports no error here and leaves a working renderer behind.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		code = checkClose()
	}
	os.Exit(code)
}

// checkClose builds the renderer through the pipeline, closes it, and checks
// it is gone: the webassembly runtime goes with it, so nothing renders
// afterwards.
func checkClose() int {
	ctx := context.Background()
	pathName := filepath.Join("..", "..", "testdata", "mermaid", "flowchart.mmd")
	if _, err := os.Stat(pathName); err != nil {
		return 0
	}
	if _, _, err := decoder.DecodeFile(ctx, pathName); err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		return 1
	}
	if err := decoder.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "decoder close: %v\n", err)
		return 1
	}
	if _, _, err := decoder.DecodeFile(ctx, pathName); err == nil {
		fmt.Fprintln(os.Stderr, "decoder close: the renderer still renders, so it was never closed")
		return 1
	}
	return 0
}
