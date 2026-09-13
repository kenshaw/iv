package decoder

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
)

func TestNormalizeMime(t *testing.T) {
	for _, test := range []struct{ in, exp string }{
		{"image/png", "image/png"},
		{"IMAGE/PNG", "image/png"},
		{"text/plain; charset=utf-8", "text/plain"},
		{"  text/html ;x=1", "text/html"},
		{"", ""},
	} {
		if got := normalizeMime(test.in); got != test.exp {
			t.Errorf("normalizeMime(%q) = %q, want %q", test.in, got, test.exp)
		}
	}
}

func TestIsGeneric(t *testing.T) {
	for _, test := range []struct {
		mime string
		exp  bool
	}{
		{"", true},
		{"application/octet-stream", true},
		{"text/plain", true},
		{"application/zip", true},
		{"image/png", false},
		{"font/ttf", false},
		// libmagic's guesses over plain text
		{"text/csv", true},
		{"text/tab-separated-values", true},
		{"text/x-c", true},
		{"text/x-ruby", true},
		{"text/x-shellscript", true},
		// not a guess: a format it read
		{"text/html", false},
		{"text/rtf", false},
	} {
		if got := isGeneric(test.mime); got != test.exp {
			t.Errorf("isGeneric(%q) = %v, want %v", test.mime, got, test.exp)
		}
	}
}

func TestDetectForceMime(t *testing.T) {
	c := ivctx.New()
	c.ForceMime = "IMAGE/Custom; charset=utf-8"
	ctx := ivctx.WithConfig(context.Background(), c)
	got, err := Detect(ctx, strings.NewReader("not really an image"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "image/custom"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
}

func TestDetectSniffs(t *testing.T) {
	// a 1x1 png
	buf, err := os.ReadFile("../testdata/png/rose.png")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	rs := bytes.NewReader(buf)
	got, err := Detect(context.Background(), rs)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "image/png"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
	// detection must leave the reader at its start
	if pos, err := rs.Seek(0, io.SeekCurrent); err != nil || pos != 0 {
		t.Errorf("expected the reader to be rewound, got pos %d err %v", pos, err)
	}
}

func TestDetectCustomDetector(t *testing.T) {
	defer cleanup(t, "d-custom")
	Register("d-custom", MimeDetector(func(_ context.Context, r io.Reader) (string, error) {
		buf := make([]byte, 5)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		if string(buf) == "MAGIC" {
			return "application/x-magic", nil
		}
		return "", nil
	}))
	got, err := Detect(context.Background(), strings.NewReader("MAGIC and then some"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "application/x-magic"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
	// a detector that declines leaves detection to the sniffer
	got, err = Detect(context.Background(), strings.NewReader("plain old text"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "text/plain"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
}

func TestDetectCustomDetectorError(t *testing.T) {
	defer cleanup(t, "d-err")
	Register("d-err", MimeDetector(func(context.Context, io.Reader) (string, error) {
		return "", errors.New("detector failed")
	}))
	// a failing detector must not fail detection
	got, err := Detect(context.Background(), strings.NewReader("plain old text"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "text/plain"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
}

func TestRegisterMimeType(t *testing.T) {
	// libmagic describes a bare truetype font as "TrueType Font data, ..."
	RegisterMimeType("font/test-ttf", `^TrueType Font data`)
	t.Cleanup(func() {
		descMu.Lock()
		defer descMu.Unlock()
		descriptions = nil
	})
	for _, test := range []struct {
		desc string
		exp  string
		ok   bool
	}{
		{"TrueType Font data, 18 tables", "font/test-ttf", true},
		{"truetype font data, digitally signed", "font/test-ttf", true},
		{"PNG image data, 200 x 200", "", false},
	} {
		got, ok := lookupDescription(test.desc)
		if ok != test.ok || got != test.exp {
			t.Errorf("lookupDescription(%q) = %q, %v; want %q, %v", test.desc, got, ok, test.exp, test.ok)
		}
	}
}

func TestDescribe(t *testing.T) {
	buf, err := os.ReadFile("../testdata/png/rose.png")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	desc, err := Describe(buf)
	if err != nil {
		t.Skipf("libmagic unavailable: %v", err)
	}
	if !strings.Contains(strings.ToLower(desc), "png") {
		t.Errorf("expected libmagic to describe a png, got %q", desc)
	}
}

// TestDetectFallsBackToDescription covers the second stage: libmagic has no
// mime type for a font, reporting application/octet-stream, but it does
// describe one -- which is what the registered patterns match against.
func TestDetectFallsBackToDescription(t *testing.T) {
	buf, err := os.ReadFile("../testdata/fontimg/Ubuntu-R.ttf")
	if err != nil {
		t.Skipf("no font test data: %v", err)
	}
	if _, err := Describe(buf); err != nil {
		t.Skipf("libmagic unavailable: %v", err)
	}
	RegisterMimeType("font/ttf", `^TrueType Font data`, `^OpenType font data`)
	t.Cleanup(func() {
		descMu.Lock()
		defer descMu.Unlock()
		descriptions = nil
	})
	got, err := Detect(context.Background(), bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if exp := "font/ttf"; got != exp {
		t.Errorf("expected %q, got %q", exp, got)
	}
}

func TestIsUnidentified(t *testing.T) {
	for _, test := range []struct {
		mime string
		exp  bool
	}{
		{"", true},
		{"application/octet-stream", true},
		{"text/plain", false},
		{"application/zip", false},
		{"image/png", false},
	} {
		if got := isUnidentified(test.mime); got != test.exp {
			t.Errorf("isUnidentified(%q) = %v, want %v", test.mime, got, test.exp)
		}
	}
}

// TestDetectText guards the decision in isGeneric: libmagic types plain text
// by what it looks like, so markdown comes back as text/x-c and a mermaid
// diagram as text/x-ruby. Whatever it decides, the result has to stay a text
// type, and a guess at a language has to stay generic so the extension is
// still what routes the file.
func TestDetectText(t *testing.T) {
	for _, name := range []string{
		"../testdata/mermaid/aws.mmd",
		"../testdata/graphviz/booktest_sqlite3.dot",
		"../testdata/blitz/sample.md",
	} {
		t.Run(name, func(t *testing.T) {
			buf, err := os.ReadFile(name)
			if err != nil {
				t.Skipf("no test data: %v", err)
			}
			got, err := Detect(context.Background(), bytes.NewReader(buf))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if !strings.HasPrefix(got, "text/") {
				t.Errorf("expected a text mime type, got %q", got)
			}
			if strings.HasPrefix(got, "text/x-") && !isGeneric(got) {
				t.Errorf("expected the language guess %q to be generic", got)
			}
		})
	}
}
