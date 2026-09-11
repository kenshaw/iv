package ivctx

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/kenshaw/colors"
)

func TestGetDefaults(t *testing.T) {
	// a context with no config still yields usable defaults
	c := Get(context.Background())
	switch {
	case c == nil:
		t.Fatal("expected a config")
	case c.Logger == nil:
		t.Error("expected a non-nil logger")
	case c.Border != 30:
		t.Errorf("expected border 30, got %d", c.Border)
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
	c.Logger = func(s string, v ...any) {
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

func TestAddBorder(t *testing.T) {
	c := New()
	c.Border = 3
	bg := colors.New(0, 0, 255, 255)
	c.Bg = &bg
	ctx := WithConfig(context.Background(), c)
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for x := range 4 {
		for y := range 2 {
			src.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	dst := AddBorder(ctx, src)
	if got, exp := dst.Bounds(), image.Rect(0, 0, 10, 8); got != exp {
		t.Fatalf("expected bounds %v, got %v", exp, got)
	}
	// the corner is the border color, the middle is the source
	if r, g, b, _ := dst.At(0, 0).RGBA(); r != 0 || g != 0 || b != 0xffff {
		t.Errorf("expected a blue border, got %v", dst.At(0, 0))
	}
	if r, _, b, _ := dst.At(4, 4).RGBA(); r != 0xffff || b != 0 {
		t.Errorf("expected the source pixel, got %v", dst.At(4, 4))
	}
}

func TestAddBackground(t *testing.T) {
	transparent := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for _, test := range []struct {
		name    string
		bg      *colors.Color
		mermaid *colors.Color
		mime    string
		exp     color.NRGBA
		same    bool
	}{
		{
			name: "transparent background is a no-op",
			bg:   ptr(colors.New(0, 0, 0, 0)),
			mime: "image/png",
			same: true,
		},
		{
			name: "svg supplies its own background",
			bg:   ptr(colors.New(255, 0, 0, 255)),
			mime: "image/svg+xml",
			same: true,
		},
		{
			name: "opaque background is composited",
			bg:   ptr(colors.New(255, 0, 0, 255)),
			mime: "image/png",
			exp:  color.NRGBA{R: 255, A: 255},
		},
		{
			name:    "mermaid falls back to the mermaid background",
			bg:      ptr(colors.New(0, 0, 0, 0)),
			mermaid: ptr(colors.New(0, 255, 0, 255)),
			mime:    "text/plain",
			exp:     color.NRGBA{G: 255, A: 255},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := New()
			c.Bg = test.bg
			if test.mermaid != nil {
				c.MermaidBg = test.mermaid
			}
			ctx := WithConfig(context.Background(), c)
			got := AddBackground(ctx, test.mime, transparent)
			if test.same {
				if got != image.Image(transparent) {
					t.Fatal("expected the source image to be returned unchanged")
				}
				return
			}
			if got == image.Image(transparent) {
				t.Fatal("expected a new image")
			}
			r, g, b, a := got.At(0, 0).RGBA()
			e := color.NRGBA(test.exp)
			er, eg, eb, ea := e.RGBA()
			if r != er || g != eg || b != eb || a != ea {
				t.Errorf("expected %v, got %v", test.exp, got.At(0, 0))
			}
		})
	}
}

func TestNopWriteCloser(t *testing.T) {
	w := NopWriteCloser{Writer: discard{}}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

type discard struct{}

func (discard) Write(b []byte) (int, error) { return len(b), nil }

func ptr[T any](v T) *T { return &v }
