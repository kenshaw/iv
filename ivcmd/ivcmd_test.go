package ivcmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/decoder"
)

func TestTargets(t *testing.T) {
	dir := t.TempDir()
	// a directory holding renderable and unrenderable files
	for _, name := range []string{"b.png", "a.png", "notes.unknownext"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	file := filepath.Join(dir, "a.png")
	for _, test := range []struct {
		name    string
		args    []string
		exp     []Target
		wantErr bool
	}{
		{
			name: "file",
			args: []string{file},
			exp:  []Target{{Name: file}},
		},
		{
			name: "directory is sorted and filtered",
			args: []string{dir},
			exp: []Target{
				{Name: filepath.Join(dir, "a.png")},
				{Name: filepath.Join(dir, "b.png")},
			},
		},
		{
			name:    "empty directory",
			args:    []string{empty},
			wantErr: true,
		},
		{
			name: "data url",
			args: []string{"data:image/png;base64,AAAA"},
			exp:  []Target{{Name: "data:image/png;base64,AAAA", IsString: true}},
		},
		{
			name: "wifi code",
			args: []string{"WIFI:ssid"},
			exp:  []Target{{Name: "WIFI:ssid", IsString: true}},
		},
		{
			name: "http url",
			args: []string{"https://example.com/a.png"},
			exp:  []Target{{Name: "https://example.com/a.png", IsString: true}},
		},
		{
			name:    "unsupported",
			args:    []string{"ftp://example.com/a.png"},
			wantErr: true,
		},
		{
			name:    "missing file",
			args:    []string{filepath.Join(dir, "nope.png")},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, errs := Targets(test.args...)
			if test.wantErr {
				if len(errs) == 0 {
					t.Fatalf("expected an error, got targets %v", got)
				}
				return
			}
			if len(errs) != 0 {
				t.Fatalf("expected no errors, got: %v", errs)
			}
			if !slices.Equal(got, test.exp) {
				t.Errorf("expected %v, got %v", test.exp, got)
			}
		})
	}
}

func TestTargetsCollectsErrorsAndTargets(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "a.png")
	if err := os.WriteFile(good, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	targets, errs := Targets(good, "ftp://nope", good)
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestEncoderSelection(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    *Args
		exp     string
		wantErr bool
	}{
		{"explicit name", &Args{Encoder: "png"}, "png", false},
		{"explicit name wins over out", &Args{Encoder: "png", Out: "x.jpg"}, "png", false},
		{"from out extension", &Args{Out: "x.jpg"}, "jpeg", false},
		{"from out webp extension", &Args{Out: "x.webp"}, "nativewebp", false},
		{"unknown encoder", &Args{Encoder: "nope"}, "", true},
		{"unknown out extension", &Args{Out: "x.zzz"}, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, err := test.args.encoder()
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %v", e)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if e.Name != test.exp {
				t.Errorf("expected %q, got %q", test.exp, e.Name)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	fg, bg := colors.New(1, 2, 3, 4), colors.New(5, 6, 7, 8)
	args := &Args{
		Verbose:   true,
		Page:      3,
		Border:    12,
		Fg:        &fg,
		Bg:        &bg,
		ForceMime: "image/png",
	}
	var buf bytes.Buffer
	c := args.Config(&buf)
	switch {
	case c.Page != 3:
		t.Errorf("expected page 3, got %d", c.Page)
	case c.Border != 12:
		t.Errorf("expected border 12, got %d", c.Border)
	case c.Fg != &fg || c.Bg != &bg:
		t.Error("expected the colors to carry over")
	case c.ForceMime != "image/png":
		t.Errorf("expected the forced mime to carry over, got %q", c.ForceMime)
	}
	c.Logger("hello %s", "world")
	if got := buf.String(); got != "hello world\n" {
		t.Errorf("expected the verbose logger to write to stderr, got %q", got)
	}
	// without verbose, the logger discards
	buf.Reset()
	args.Verbose = false
	args.Config(&buf).Logger("hello")
	if buf.Len() != 0 {
		t.Errorf("expected no output without verbose, got %q", buf.String())
	}
}

func TestExecList(t *testing.T) {
	var buf bytes.Buffer
	args := &Args{List: true}
	if err := args.Exec(context.Background(), &buf, &buf, nil); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	for _, s := range []string{"decoders:", "encoders:", "extensions:", "png", "rasterm"} {
		if !strings.Contains(out, s) {
			t.Errorf("expected the listing to mention %q", s)
		}
	}
}

func TestExecNoTargets(t *testing.T) {
	var buf bytes.Buffer
	args := &Args{Encoder: "png"}
	if err := args.Exec(context.Background(), &buf, &buf, nil); err == nil {
		t.Fatal("expected an error")
	}
}

func TestExecOutRequiresSingleTarget(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.png", "b.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	args := &Args{Out: filepath.Join(dir, "out.png"), Quiet: true}
	err := args.Exec(context.Background(), &buf, &buf, []string{dir})
	if err == nil || !strings.Contains(err.Error(), "exactly one target") {
		t.Fatalf("expected a single target error, got: %v", err)
	}
}

func TestExecWritesOutputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")
	var stdout, stderr bytes.Buffer
	args := &Args{Out: out, Quiet: true}
	if err := args.Exec(context.Background(), &stdout, &stderr, []string{"../testdata/png/rose.png"}); err != nil {
		t.Fatalf("expected no error, got: %v (%s)", err, stderr.String())
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if fi.Size() == 0 {
		t.Error("expected a non-empty output file")
	}
	if stdout.Len() != 0 {
		t.Errorf("expected nothing on stdout when writing to a file, got %q", stdout.String())
	}
}

func TestExecReportsRenderErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	// no decoder claims the forced mime type
	args := &Args{Encoder: "png", Quiet: true, ForceMime: "application/x-not-supported"}
	err := args.Exec(context.Background(), &stdout, &stderr, []string{bad})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(stderr.String(), "bad.png") {
		t.Errorf("expected the failing target to be named on stderr, got %q", stderr.String())
	}
}

func TestDecoderRegistryIsPopulated(t *testing.T) {
	// importing ivcmd must register the full decoder and encoder sets
	for _, name := range []string{"png", "jpeg", "vips", "resvg", "markdown", "qr", "data", "http"} {
		if _, ok := decoder.Get(name); !ok {
			t.Errorf("expected the %q decoder to be registered", name)
		}
	}
	if !errors.Is(func() error { _, _, err := decoder.DecodeString(context.Background(), "nope://x"); return err }(), decoder.ErrNotSupported) {
		t.Error("expected an unsupported string to be reported as such")
	}
}
