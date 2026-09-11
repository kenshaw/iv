package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

func TestPath(t *testing.T) {
	c := New("definitely-not-a-real-command-xyz")
	if _, err := c.Path(); err == nil {
		t.Fatal("expected an error")
	}
	if c.Available() {
		t.Error("expected the command to be unavailable")
	}
	// the lookup result is cached, so the error must be stable
	if _, err := c.Path(); err == nil || !strings.Contains(err.Error(), "not in path") {
		t.Errorf("expected a stable not-in-path error, got: %v", err)
	}
	sh := New("sh")
	if !sh.Available() {
		t.Skip("sh not in path")
	}
	if sh.Name() != "sh" {
		t.Errorf("expected name sh, got %q", sh.Name())
	}
}

func TestRun(t *testing.T) {
	c := New("sh")
	if !c.Available() {
		t.Skip("sh not in path")
	}
	ctx := context.Background()
	buf, err := c.Run(ctx, "-c", "printf hello")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if string(buf) != "hello" {
		t.Errorf("expected %q, got %q", "hello", string(buf))
	}
	// a failing command surfaces its stderr
	if _, err := c.Run(ctx, "-c", "echo bad things >&2; exit 3"); err == nil {
		t.Error("expected an error")
	} else if !strings.Contains(err.Error(), "bad things") {
		t.Errorf("expected the error to include stderr, got: %v", err)
	}
}

func TestCombinedOutput(t *testing.T) {
	c := New("sh")
	if !c.Available() {
		t.Skip("sh not in path")
	}
	buf, err := c.CombinedOutput(context.Background(), "-c", "printf out; printf err >&2")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, s := range []string{"out", "err"} {
		if !strings.Contains(string(buf), s) {
			t.Errorf("expected the output to contain %q, got %q", s, string(buf))
		}
	}
}

func TestSourceUsesExistingPath(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(name, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := ivctx.WithPathName(context.Background(), name)
	got, cleanup, err := Source(ctx, strings.NewReader("ignored"), "txt")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer cleanup()
	if got != name {
		t.Errorf("expected the existing path %q, got %q", name, got)
	}
}

func TestSourceSpoolsStream(t *testing.T) {
	// no path on the context, so the stream is spooled to a temp file
	got, cleanup, err := Source(context.Background(), strings.NewReader("spooled"), "bin")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if filepath.Ext(got) != ".bin" {
		t.Errorf("expected a .bin extension, got %q", got)
	}
	buf, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if string(buf) != "spooled" {
		t.Errorf("expected %q, got %q", "spooled", string(buf))
	}
	cleanup()
	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Error("expected cleanup to remove the spooled file")
	}
}

func TestSourceSpoolsWhenPathIsMissing(t *testing.T) {
	// a path that is not on disk (an archive entry, say) must still spool
	ctx := ivctx.WithPathName(context.Background(), "inside-an-archive/page.png")
	got, cleanup, err := Source(ctx, strings.NewReader("x"), "png")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer cleanup()
	if got == "inside-an-archive/page.png" {
		t.Error("expected the stream to be spooled")
	}
}

func TestTempDir(t *testing.T) {
	dir, cleanup, err := TempDir(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("expected a directory, got err %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected cleanup to remove the directory")
	}
}

func TestElide(t *testing.T) {
	if got := elide("  short  "); got != "short" {
		t.Errorf("expected %q, got %q", "short", got)
	}
	long := strings.Repeat("x", errLen+50)
	got := elide(long)
	if len(got) != errLen+3 || !strings.HasSuffix(got, "...") {
		t.Errorf("expected the output to be elided, got %d chars", len(got))
	}
}
