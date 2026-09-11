package decoder

import (
	"context"
	"image"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	defer cleanup(t, "a")
	d := Register("a", Desc("test a"), Extension("aa", "ab"), MimeType("image/a"))
	switch v, ok := Get("a"); {
	case !ok:
		t.Fatal("expected a to be registered")
	case v != d:
		t.Errorf("expected the registered decoder, got %v", v)
	case v.String() != "a (test a)":
		t.Errorf("expected %q, got %q", "a (test a)", v.String())
	}
	if _, ok := Get("nope"); ok {
		t.Error("expected nope to not be registered")
	}
}

func TestRegisterPanics(t *testing.T) {
	defer cleanup(t, "dup")
	Register("dup")
	for _, test := range []struct {
		name string
		f    func()
	}{
		{"empty name", func() { Register("") }},
		{"duplicate name", func() { Register("dup") }},
		{"odd pairs", func() { Register("odd", MimeTypeExtensionMatch("image/x")) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("expected a panic")
				}
			}()
			test.f()
		})
	}
}

func TestMimeMatch(t *testing.T) {
	for _, test := range []struct {
		pattern string
		mime    string
		exp     bool
	}{
		{"image/png", "image/png", true},
		{"image/png", "IMAGE/PNG", true},
		{"image/png", "image/jpeg", false},
		{"video/*", "video/mp4", true},
		{"video/*", "audio/mp4", false},
		{"audio/*", "audio/x-flac", true},
		{"image/svg+xml", "image/svg", true},
		{"image/svg", "image/svg+xml", true},
		{"text/fb2+xml", "text/fb2", true},
		{"image/png", "", false},
	} {
		if got := MimeMatch(test.pattern, test.mime); got != test.exp {
			t.Errorf("MimeMatch(%q, %q) = %v, want %v", test.pattern, test.mime, got, test.exp)
		}
	}
}

func TestMatch(t *testing.T) {
	defer cleanup(t, "m-mime", "m-wild", "m-pair", "m-custom", "m-str")
	Register("m-mime", MimeType("image/png"), Extension("png"), Decoder(nilDecode))
	Register("m-wild", MimeType("video/*"), Extension("mp4"), Decoder(nilDecode))
	Register("m-pair", MimeTypeExtensionMatch("application/zip", "cbz"), Decoder(nilDecode))
	Register("m-custom", Matcher(func(mime, ext string) bool {
		return strings.HasPrefix(mime, "font/")
	}), Decoder(nilDecode))
	Register("m-str", StringMatcher(func(s string) bool {
		return strings.HasPrefix(s, "WIFI:")
	}), StringDecoder(nil))
	ctx := context.Background()
	for _, test := range []struct {
		name string
		mime string
		ext  string
		exp  []string
	}{
		{"exact mime", "image/png", "png", []string{"m-mime"}},
		{"wildcard mime", "video/mp4", "mp4", []string{"m-wild"}},
		{"custom matcher", "font/ttf", "ttf", []string{"m-custom"}},
		{"mime+ext pair", "application/zip", "cbz", []string{"m-pair"}},
		{"pair wrong ext", "application/zip", "docx", nil},
		{"generic falls back to ext", "application/octet-stream", "png", []string{"m-mime"}},
		{"generic unknown ext", "application/octet-stream", "zzz", nil},
		{"no match", "image/gif", "gif", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := names(Match(ctx, test.mime, test.ext)); !slices.Equal(got, test.exp) {
				t.Errorf("Match(%q, %q) = %v, want %v", test.mime, test.ext, got, test.exp)
			}
		})
	}
	// string decoders are never returned by Match, and only they are returned
	// by MatchString
	if got := names(MatchString(ctx, "WIFI:ssid")); !slices.Equal(got, []string{"m-str"}) {
		t.Errorf("MatchString = %v, want [m-str]", got)
	}
	if got := names(MatchString(ctx, "image/png")); got != nil {
		t.Errorf("MatchString = %v, want nil", got)
	}
}

func TestOrder(t *testing.T) {
	for _, test := range []struct {
		name string
		in   []*Entry
		exp  []string
	}{
		{
			"registration order",
			[]*Entry{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			[]string{"a", "b", "c"},
		},
		{
			"before",
			[]*Entry{{Name: "a"}, {Name: "b"}, {Name: "c", before: []string{"a"}}},
			[]string{"c", "a", "b"},
		},
		{
			"after",
			[]*Entry{{Name: "a", after: []string{"c"}}, {Name: "b"}, {Name: "c"}},
			[]string{"c", "a", "b"},
		},
		{
			"chained",
			[]*Entry{{Name: "a", after: []string{"b"}}, {Name: "b", after: []string{"c"}}, {Name: "c"}},
			[]string{"c", "b", "a"},
		},
		{
			"unknown names ignored",
			[]*Entry{{Name: "a", after: []string{"zzz"}}, {Name: "b", before: []string{"yyy"}}},
			[]string{"a", "b"},
		},
		{
			"cycle drops one constraint",
			[]*Entry{{Name: "a", after: []string{"b"}}, {Name: "b", after: []string{"a"}}},
			[]string{"b", "a"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := names(order(test.in)); !slices.Equal(got, test.exp) {
				t.Errorf("order = %v, want %v", got, test.exp)
			}
		})
	}
}

func TestOrderIsStable(t *testing.T) {
	in := []*Entry{{Name: "a"}, {Name: "b"}, {Name: "c", before: []string{"b"}}, {Name: "d"}}
	exp := names(order(in))
	for range 10 {
		if got := names(order(in)); !slices.Equal(got, exp) {
			t.Fatalf("order is not deterministic: %v != %v", got, exp)
		}
	}
}

func TestExtensions(t *testing.T) {
	defer cleanup(t, "e-one", "e-two")
	RegisterBuiltin("e-one", Extension("zzb", "zza"), Decoder(nilDecode))
	Register("e-two", Extension("zza", "zzc"), Decoder(nilDecode))
	if got, exp := Extensions(), []string{"zza", "zzb", "zzc"}; !slices.Equal(got, exp) {
		t.Errorf("Extensions = %v, want %v", got, exp)
	}
	for _, test := range []struct {
		name      string
		supported bool
		builtin   bool
	}{
		{"x/y/file.zza", true, true},
		{"file.ZZB", true, true},
		{"file.zzc", true, false},
		{"file.zzz", false, false},
		{"noext", false, false},
	} {
		if got := SupportedExt(test.name); got != test.supported {
			t.Errorf("SupportedExt(%q) = %v, want %v", test.name, got, test.supported)
		}
		if got := BuiltinExt(test.name); got != test.builtin {
			t.Errorf("BuiltinExt(%q) = %v, want %v", test.name, got, test.builtin)
		}
	}
}

func TestInitClose(t *testing.T) {
	defer cleanup(t, "ic")
	var inits, closes int
	Register("ic",
		MimeType("image/ic"),
		Init(
			func(context.Context) (any, error) {
				inits++
				return "state", nil
			},
			func(context.Context) error {
				closes++
				return nil
			},
		),
		Decoder(func(ctx context.Context, _ io.Reader) (any, error) {
			if s, _ := State(ctx).(string); s != "state" {
				t.Errorf("expected the init state on the context, got %q", s)
			}
			return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
		}),
	)
	ctx := context.Background()
	for range 3 {
		if _, _, err := Decode(ctx, "image/ic", "", strings.NewReader("x")); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	}
	if inits != 1 {
		t.Errorf("expected the init func to run once, ran %d times", inits)
	}
	if err := Close(ctx); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// closing twice must not run the close func again
	if err := Close(ctx); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if closes != 1 {
		t.Errorf("expected the close func to run once, ran %d times", closes)
	}
}

func TestRegisterBuiltin(t *testing.T) {
	defer cleanup(t, "b-default", "b-custom")
	// with no decode func, a builtin decoder goes through image.Decode
	d := RegisterBuiltin("b-default", MimeType("image/bd"))
	switch {
	case !d.Builtin:
		t.Error("expected the decoder to be marked builtin")
	case d.decode == nil:
		t.Fatal("expected a decode func")
	}
	if _, _, err := Decode(context.Background(), "image/bd", "", strings.NewReader("not an image")); err == nil {
		t.Error("expected image.Decode to reject the input")
	}
	// an explicit decode func wins over the image.Decode default: the
	// builtin mark says the format is in the registry, not that the decoder
	// has to use it
	var called bool
	d = RegisterBuiltin("b-custom",
		MimeType("image/bc"),
		Decoder(func(context.Context, io.Reader) (any, error) {
			called = true
			return image.NewRGBA(image.Rect(0, 0, 2, 2)), nil
		}),
	)
	if !d.Builtin {
		t.Error("expected the decoder to be marked builtin")
	}
	img, _, err := Decode(context.Background(), "image/bc", "", strings.NewReader("not an image"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Error("expected the registered decode func to be used")
	}
	if got := img.Bounds().Dx(); got != 2 {
		t.Errorf("expected width 2, got %d", got)
	}
}

func TestUnregister(t *testing.T) {
	Register("u-one")
	Register("u-two")
	if !Unregister("u-one") {
		t.Error("expected u-one to be unregistered")
	}
	if Unregister("u-one") {
		t.Error("expected the second unregister to report false")
	}
	if _, ok := Get("u-one"); ok {
		t.Error("expected u-one to be gone")
	}
	if _, ok := Get("u-two"); !ok {
		t.Error("expected u-two to remain")
	}
	Unregister("u-two")
}

// nilDecode is a decode func that decodes nothing.
func nilDecode(context.Context, io.Reader) (any, error) {
	return nil, nil
}

// names returns the decoder names.
func names(v []*Entry) []string {
	var res []string
	for _, d := range v {
		res = append(res, d.Name)
	}
	return res
}

// cleanup unregisters the named decoders when the test finishes.
func cleanup(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		Unregister(name)
	}
}
