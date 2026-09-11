package decoder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kenshaw/iv/ivctx"
)

func TestDecodeImage(t *testing.T) {
	defer cleanup(t, "p-img")
	Register("p-img", MimeType("image/p"), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return solid(len(buf), 1), nil
	}))
	img, mime, err := Decode(context.Background(), "image/p", "", strings.NewReader("abcd"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if mime != "image/p" {
		t.Errorf("expected mime %q, got %q", "image/p", mime)
	}
	if got := img.Bounds().Dx(); got != 4 {
		t.Errorf("expected width 4, got %d", got)
	}
}

func TestDecodeNotSupported(t *testing.T) {
	_, _, err := Decode(context.Background(), "image/nope", "", strings.NewReader("x"))
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got: %v", err)
	}
}

func TestDecodeFallsThroughToNextDecoder(t *testing.T) {
	defer cleanup(t, "p-bad", "p-good")
	var badRead string
	Register("p-bad", MimeType("image/f"), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, _ := io.ReadAll(r)
		badRead = string(buf)
		return nil, errors.New("nope")
	}))
	Register("p-good", After("p-bad"), MimeType("image/f"), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		// the failed decoder consumed the stream, so this only works if the
		// pipeline rewound it
		return solid(len(buf), 1), nil
	}))
	img, _, err := Decode(context.Background(), "image/f", "", strings.NewReader("abcde"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if badRead != "abcde" {
		t.Errorf("expected the first decoder to read the stream, got %q", badRead)
	}
	if got := img.Bounds().Dx(); got != 5 {
		t.Errorf("expected width 5, got %d", got)
	}
}

func TestDecodeAllDecodersFail(t *testing.T) {
	defer cleanup(t, "p-e1", "p-e2")
	Register("p-e1", MimeType("image/e"), Decoder(func(context.Context, io.Reader) (any, error) {
		return nil, errors.New("first failed")
	}))
	Register("p-e2", After("p-e1"), MimeType("image/e"), Decoder(func(context.Context, io.Reader) (any, error) {
		return nil, errors.New("second failed")
	}))
	_, _, err := Decode(context.Background(), "image/e", "", strings.NewReader("x"))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, s := range []string{"first failed", "second failed"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("expected the error to mention %q, got: %v", s, err)
		}
	}
}

func TestDecodeImagesSelectsPage(t *testing.T) {
	defer cleanup(t, "p-multi")
	Register("p-multi", MimeType("image/m"), ImagesDecoder(func(io.Reader) ([]image.Image, error) {
		return []image.Image{solid(1, 1), solid(2, 2), solid(3, 3)}, nil
	}))
	for _, test := range []struct {
		page uint
		exp  int
	}{
		{0, 1}, // unset selects the first
		{1, 1},
		{2, 2},
		{3, 3},
		{9, 1}, // out of range falls back to the first
	} {
		t.Run(fmt.Sprint(test.page), func(t *testing.T) {
			c := ivctx.New()
			c.Page = test.page
			img, _, err := Decode(ivctx.WithConfig(context.Background(), c), "image/m", "", strings.NewReader("x"))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got := img.Bounds().Dx(); got != test.exp {
				t.Errorf("expected width %d, got %d", test.exp, got)
			}
		})
	}
}

func TestDecodeEmptyImages(t *testing.T) {
	defer cleanup(t, "p-empty")
	Register("p-empty", MimeType("image/z"), ImagesDecoder(func(io.Reader) ([]image.Image, error) {
		return nil, nil
	}))
	if _, _, err := Decode(context.Background(), "image/z", "", strings.NewReader("x")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDecodeRetriesWithNewMime(t *testing.T) {
	defer cleanup(t, "p-outer", "p-inner")
	Register("p-outer", MimeType("application/outer"), Decoder(func(context.Context, io.Reader) (any, error) {
		return NewBytes("image/inner", []byte("1234567")).WithExt("inner"), nil
	}))
	var gotExt string
	Register("p-inner", MimeType("image/inner"), MatcherContext(func(_ context.Context, _, ext string) bool {
		gotExt = ext
		return false
	}), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return solid(len(buf), 1), nil
	}))
	img, mime, err := Decode(context.Background(), "application/outer", "", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if mime != "image/inner" {
		t.Errorf("expected the retried mime %q, got %q", "image/inner", mime)
	}
	if gotExt != "inner" {
		t.Errorf("expected the retried ext %q, got %q", "inner", gotExt)
	}
	if got := img.Bounds().Dx(); got != 7 {
		t.Errorf("expected width 7, got %d", got)
	}
}

func TestDecodeHandoff(t *testing.T) {
	defer cleanup(t, "p-from", "p-to")
	Register("p-from", MimeType("image/h"), Decoder(func(context.Context, io.Reader) (any, error) {
		return Next("p-to"), nil
	}))
	// p-to matches nothing, so it is only reachable through the hand off
	Register("p-to", Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return solid(len(buf), 1), nil
	}))
	img, mime, err := Decode(context.Background(), "image/h", "", strings.NewReader("abc"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if mime != "image/h" {
		t.Errorf("expected mime %q, got %q", "image/h", mime)
	}
	if got := img.Bounds().Dx(); got != 3 {
		t.Errorf("expected width 3, got %d (the hand off did not see the original stream)", got)
	}
}

func TestDecodeHandoffUnknown(t *testing.T) {
	defer cleanup(t, "p-lost")
	Register("p-lost", MimeType("image/l"), Decoder(func(context.Context, io.Reader) (any, error) {
		return Next("does-not-exist"), nil
	}))
	_, _, err := Decode(context.Background(), "image/l", "", strings.NewReader("x"))
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected an error naming the missing decoder, got: %v", err)
	}
}

func TestDecodeFiles(t *testing.T) {
	defer cleanup(t, "p-container", "p-leaf")
	fsys := fstest.MapFS{
		"a.leaf": &fstest.MapFile{Data: []byte("1")},
		"b.leaf": &fstest.MapFile{Data: []byte("22")},
		"c.leaf": &fstest.MapFile{Data: []byte("333")},
	}
	Register("p-container", MimeType("application/container"), Decoder(func(context.Context, io.Reader) (any, error) {
		return FS(fsys, "a.leaf", "b.leaf", "c.leaf"), nil
	}))
	Register("p-leaf", Extension("leaf"), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return solid(len(buf), 1), nil
	}))
	for _, test := range []struct {
		page uint
		exp  int
	}{{0, 1}, {2, 2}, {3, 3}, {99, 1}} {
		t.Run(fmt.Sprint(test.page), func(t *testing.T) {
			c := ivctx.New()
			c.Page = test.page
			img, _, err := Decode(ivctx.WithConfig(context.Background(), c), "application/container", "", strings.NewReader("x"))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got := img.Bounds().Dx(); got != test.exp {
				t.Errorf("expected width %d, got %d", test.exp, got)
			}
		})
	}
}

func TestDecodeEmptyFiles(t *testing.T) {
	defer cleanup(t, "p-nofiles")
	Register("p-nofiles", MimeType("application/nofiles"), Decoder(func(context.Context, io.Reader) (any, error) {
		return FS(fstest.MapFS{}), nil
	}))
	if _, _, err := Decode(context.Background(), "application/nofiles", "", strings.NewReader("x")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDecodeRetryLimit(t *testing.T) {
	defer cleanup(t, "p-loop")
	Register("p-loop", MimeType("application/loop"), Decoder(func(context.Context, io.Reader) (any, error) {
		return NewBytes("application/loop", []byte("x")), nil
	}))
	_, _, err := Decode(context.Background(), "application/loop", "", strings.NewReader("x"))
	if err == nil || !strings.Contains(err.Error(), "retries") {
		t.Fatalf("expected a retry limit error, got: %v", err)
	}
}

func TestDecodeUnknownResult(t *testing.T) {
	defer cleanup(t, "p-weird", "p-nil")
	Register("p-weird", MimeType("application/weird"), Decoder(func(context.Context, io.Reader) (any, error) {
		return 42, nil
	}))
	Register("p-nil", MimeType("application/nil"), Decoder(func(context.Context, io.Reader) (any, error) {
		return nil, nil
	}))
	for _, mime := range []string{"application/weird", "application/nil"} {
		if _, _, err := Decode(context.Background(), mime, "", strings.NewReader("x")); err == nil {
			t.Errorf("%s: expected an error", mime)
		}
	}
}

func TestDecodeString(t *testing.T) {
	defer cleanup(t, "p-str")
	Register("p-str",
		StringMatcher(func(s string) bool { return strings.HasPrefix(s, "TEST:") }),
		StringDecoder(func(_ context.Context, s string) (any, error) {
			return solid(len(s), 1), nil
		}),
	)
	img, _, err := DecodeString(context.Background(), "TEST:abc")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := img.Bounds().Dx(); got != 8 {
		t.Errorf("expected width 8, got %d", got)
	}
	if _, _, err := DecodeString(context.Background(), "OTHER:abc"); !errors.Is(err, ErrNotSupported) {
		t.Errorf("expected ErrNotSupported, got: %v", err)
	}
}

func TestDecodeStringRetriesThroughPipeline(t *testing.T) {
	defer cleanup(t, "p-str2", "p-target")
	Register("p-str2",
		StringMatcher(func(s string) bool { return strings.HasPrefix(s, "WRAP:") }),
		StringDecoder(func(_ context.Context, s string) (any, error) {
			return NewBytes("image/target", []byte(strings.TrimPrefix(s, "WRAP:"))), nil
		}),
	)
	Register("p-target", MimeType("image/target"), Decoder(func(_ context.Context, r io.Reader) (any, error) {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return solid(len(buf), 1), nil
	}))
	img, mime, err := DecodeString(context.Background(), "WRAP:abcd")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if mime != "image/target" {
		t.Errorf("expected mime %q, got %q", "image/target", mime)
	}
	if got := img.Bounds().Dx(); got != 4 {
		t.Errorf("expected width 4, got %d", got)
	}
}

func TestReadSeeker(t *testing.T) {
	// a plain reader is buffered so the pipeline can rewind it
	rs, err := readSeeker(onlyReader{strings.NewReader("hello")})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for range 2 {
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		buf, err := io.ReadAll(rs)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if string(buf) != "hello" {
			t.Errorf("expected %q, got %q", "hello", string(buf))
		}
	}
	// an already seekable reader is returned as-is, rewound
	sr := bytes.NewReader([]byte("world"))
	_, _ = io.ReadAll(sr)
	got, err := readSeeker(sr)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != io.ReadSeeker(sr) {
		t.Error("expected the original reader to be returned")
	}
	buf, _ := io.ReadAll(got)
	if string(buf) != "world" {
		t.Errorf("expected %q, got %q", "world", string(buf))
	}
}

// solid returns a solid white image of the given size.
func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.White)
		}
	}
	return img
}

// onlyReader hides the Seek method of the wrapped reader.
type onlyReader struct {
	io.Reader
}
