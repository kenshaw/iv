package ivctx

import (
	"context"
	"testing"
)

func TestGetDefaults(t *testing.T) {
	// a context with no config still yields usable defaults
	c := Get(context.Background())
	switch {
	case c == nil:
		t.Fatal("expected a config")
	case c.Logger == nil:
		t.Error("expected a non-nil logger")
	case c.DPI != 300:
		t.Errorf("expected dpi 300, got %d", c.DPI)
	case c.MinWidth != 64 || c.MinHeight != 64:
		t.Errorf("expected 64x64 minimums, got %dx%d", c.MinWidth, c.MinHeight)
	}
	// the default logger must be safe to call
	c.Logger("test %d", 1)
}

func TestWithConfig(t *testing.T) {
	var logged []string
	c := New()
	c.Page = 7
	c.Logger = func(s string, _ ...any) {
		logged = append(logged, s)
	}
	ctx := WithConfig(context.Background(), c)
	if got := Get(ctx); got != c {
		t.Fatal("expected the config from the context")
	}
	Logf(ctx, "hello")
	if len(logged) != 1 || logged[0] != "hello" {
		t.Errorf("expected the message to reach the logger, got %v", logged)
	}
}

func TestConfigInitSetsLogger(t *testing.T) {
	c := &Config{}
	c.Init()
	if c.Logger == nil {
		t.Fatal("expected Init to set a logger")
	}
	c.Logger("safe to call")
}

func TestWithPathNameAndMime(t *testing.T) {
	ctx := context.Background()
	if got := PathName(ctx); got != "" {
		t.Errorf("expected an empty path name, got %q", got)
	}
	if got := Mime(ctx); got != "" {
		t.Errorf("expected an empty mime, got %q", got)
	}
	ctx = WithPathName(ctx, "a/b.png")
	ctx = WithMime(ctx, "image/png")
	if got := PathName(ctx); got != "a/b.png" {
		t.Errorf("expected %q, got %q", "a/b.png", got)
	}
	if got := Mime(ctx); got != "image/png" {
		t.Errorf("expected %q, got %q", "image/png", got)
	}
}

func TestPage(t *testing.T) {
	for _, test := range []struct {
		page uint
		n    int
		exp  int
	}{
		{0, 5, 0}, // unset
		{1, 5, 0}, // pages are 1-indexed
		{3, 5, 2},
		{5, 5, 4},
		{6, 5, 0}, // past the end
		{1, 0, 0}, // no pages
	} {
		c := New()
		c.Page = test.page
		ctx := WithConfig(context.Background(), c)
		if got := Page(ctx, test.n); got != test.exp {
			t.Errorf("Page(page=%d, n=%d) = %d, want %d", test.page, test.n, got, test.exp)
		}
	}
}

func TestFileExt(t *testing.T) {
	for _, test := range []struct{ in, exp string }{
		{"a.PNG", "png"},
		{"/x/y/z.tar.gz", "gz"},
		{"noext", ""},
		{"", ""},
		{".bashrc", "bashrc"},
	} {
		if got := FileExt(test.in); got != test.exp {
			t.Errorf("FileExt(%q) = %q, want %q", test.in, got, test.exp)
		}
	}
}
