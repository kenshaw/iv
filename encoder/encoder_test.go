package encoder

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	defer cleanup(t, "e-a")
	e := Register("e-a", Desc("test a"), ImageEncoder(func(io.Writer, image.Image) error { return nil }))
	switch v, ok := Get("e-a"); {
	case !ok:
		t.Fatal("expected e-a to be registered")
	case v != e:
		t.Error("expected the registered encoder")
	case v.Ext != "e-a":
		t.Errorf("expected the extension to default to the name, got %q", v.Ext)
	case v.String() != "e-a (test a)":
		t.Errorf("expected %q, got %q", "e-a (test a)", v.String())
	}
}

func TestRegisterPanics(t *testing.T) {
	defer cleanup(t, "e-dup")
	Register("e-dup")
	for _, test := range []struct {
		name string
		f    func()
	}{
		{"empty name", func() { Register("") }},
		{"duplicate name", func() { Register("e-dup") }},
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

func TestEncode(t *testing.T) {
	defer cleanup(t, "e-enc")
	Register("e-enc", Encoder(func(_ context.Context, w io.Writer, img image.Image) error {
		_, err := io.WriteString(w, img.Bounds().String())
		return err
	}))
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 3, 4))
	if err := Encode(context.Background(), &buf, "e-enc", img); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got, exp := buf.String(), img.Bounds().String(); got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
}

func TestEncodeNotRegistered(t *testing.T) {
	err := Encode(context.Background(), io.Discard, "nope", image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("expected ErrNotRegistered, got: %v", err)
	}
}

func TestEncodeNoEncodeFunc(t *testing.T) {
	defer cleanup(t, "e-none")
	Register("e-none")
	err := Encode(context.Background(), io.Discard, "e-none", image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err == nil || !strings.Contains(err.Error(), "no encode func") {
		t.Errorf("expected a missing encode func error, got: %v", err)
	}
}

func TestExtensionAndTerm(t *testing.T) {
	defer cleanup(t, "e-jpg", "e-term")
	Register("e-jpg", Extension("jpg"), ImageEncoder(func(io.Writer, image.Image) error { return nil }))
	Register("e-term", Extension("ignored"), Term(), ImageEncoder(func(io.Writer, image.Image) error { return nil }))
	switch e, ok := ForExt("jpg"); {
	case !ok:
		t.Error("expected an encoder for jpg")
	case e.Name != "e-jpg":
		t.Errorf("expected e-jpg, got %s", e.Name)
	}
	if _, ok := ForExt("nope"); ok {
		t.Error("expected no encoder for nope")
	}
	// a terminal encoder has no extension and is never matched by one
	switch e, ok := Get("e-term"); {
	case !ok:
		t.Fatal("expected e-term to be registered")
	case !e.Term:
		t.Error("expected e-term to be a terminal encoder")
	case e.Ext != "":
		t.Errorf("expected no extension, got %q", e.Ext)
	}
	if _, ok := ForExt("ignored"); ok {
		t.Error("expected terminal encoders to be excluded from ForExt")
	}
	if got := names(TermEncoders()); !slices.Contains(got, "e-term") {
		t.Errorf("expected TermEncoders to contain e-term, got %v", got)
	}
	if got := names(TermEncoders()); slices.Contains(got, "e-jpg") {
		t.Errorf("expected TermEncoders to exclude e-jpg, got %v", got)
	}
}

func TestAllIsSorted(t *testing.T) {
	defer cleanup(t, "e-z", "e-a2", "e-m")
	Register("e-z")
	Register("e-a2")
	Register("e-m")
	got := Names()
	if !slices.IsSorted(got) {
		t.Errorf("expected sorted names, got %v", got)
	}
	if len(got) != len(All()) {
		t.Error("expected Names and All to agree")
	}
}

func TestUnregister(t *testing.T) {
	Register("e-u")
	if !Unregister("e-u") {
		t.Error("expected e-u to be unregistered")
	}
	if Unregister("e-u") {
		t.Error("expected the second unregister to report false")
	}
}

// names returns the encoder names.
func names(v []*Entry) []string {
	res := make([]string, len(v))
	for i, e := range v {
		res[i] = e.Name
	}
	return res
}

// cleanup unregisters the named encoders when the test finishes.
func cleanup(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		Unregister(name)
	}
}
