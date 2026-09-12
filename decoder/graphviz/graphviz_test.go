package graphviz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	// a dot graph is rendered to svg and handed back to the pipeline, so the
	// svg decoder has to be in the binary for a decode to finish
	_ "github.com/kenshaw/iv/decoder/resvg"
)

// dotFile is the dot graph the tests render.
var dotFile = filepath.Join("..", "..", "testdata", "graphviz", "booktest_sqlite3.dot")

// TestDetect checks a dot graph is recognized by its content. Nothing sniffs
// one -- it is plain text to everything else -- so this is what routes a .dot
// file, and what keeps other plain text off this decoder.
func TestDetect(t *testing.T) {
	for _, test := range []struct {
		name string
		src  string
		exp  string
	}{
		{"digraph", "digraph G {\n  a -> b;\n}\n", mime},
		{"graph", "graph G {\n  a -- b;\n}\n", mime},
		{"strict digraph", "strict digraph {\n}\n", mime},
		{"leading blank lines", "\n\n  digraph G {\n}\n", mime},
		{"no brace", "digraph G\n", ""},
		{"prose about graphs", "This digraph G is described below.\n", ""},
		{"markdown", "# A graph\n\nSome text.\n", ""},
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

// TestDecode renders the test graph through the pipeline.
func TestDecode(t *testing.T) {
	if _, err := os.Stat(dotFile); err != nil {
		t.Skipf("no test data: %v", err)
	}
	img, _, err := decoder.DecodeFile(context.Background(), dotFile)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if b := img.Bounds(); b.Dx() <= 0 || b.Dy() <= 0 {
		t.Errorf("expected a drawn image, got %v", b)
	}
}

// TestMain releases the graphviz engine the way iv does once the tests are
// done, and checks that what was released was the real one.
//
// The pipeline hands a close func its decoder's state -- this decoder's close
// reaches for it and returns nil when it is not there, so a broken handover
// shows up as a graph that still renders after [decoder.Close] rather than as
// an error.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		code = checkClose()
	}
	os.Exit(code)
}

// checkClose builds the engine through the pipeline, closes it, and checks it
// is gone: the wasm runtime goes with it, so nothing renders afterwards.
func checkClose() int {
	ctx := context.Background()
	if _, err := os.Stat(dotFile); err != nil {
		return 0
	}
	if _, _, err := decoder.DecodeFile(ctx, dotFile); err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		return 1
	}
	if err := decoder.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "decoder close: %v\n", err)
		return 1
	}
	if _, _, err := decoder.DecodeFile(ctx, dotFile); err == nil {
		fmt.Fprintln(os.Stderr, "decoder close: the engine still renders, so it was never closed")
		return 1
	}
	return 0
}
