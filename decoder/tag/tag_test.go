package tag

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestDecodeCard checks that every audio container yields a card svg carrying
// the cover art that was embedded in it, pixel for pixel. The card embeds the
// picture bytes untouched, so this covers the extraction for each container
// as well as the card itself.
func TestDecodeCard(t *testing.T) {
	want := loadImage(t, filepath.Join("..", "..", "testdata", "png", "tux.png"))
	for _, name := range []string{
		"silent.mp3",
		"silent.flac",
		"silent.m4a",
		"silent.ogg",
		"silent.aac",
	} {
		t.Run(name, func(t *testing.T) {
			pathName := filepath.Join("..", "..", "testdata", "tag", name)
			f, err := os.Open(pathName)
			if err != nil {
				t.Skipf("no test data: %v", err)
			}
			defer f.Close()
			res, err := decode(ivctx.WithPathName(context.Background(), pathName), f)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			img, ok := res.(*decoder.Image)
			if !ok {
				t.Fatalf("expected a *decoder.Image, got %T", res)
			}
			if img.Mime != "image/svg+xml" {
				t.Errorf("expected mime %q, got %q", "image/svg+xml", img.Mime)
			}
			buf, err := readAll(img)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			art := embedded(t, buf)
			if got := art.Bounds().Size(); got != want.Bounds().Size() {
				t.Fatalf("expected embedded art %v, got %v", want.Bounds().Size(), got)
			}
			for y := range want.Bounds().Dy() {
				for x := range want.Bounds().Dx() {
					if art.At(x, y) != want.At(x, y) {
						t.Fatalf("embedded art differs at (%d,%d)", x, y)
					}
				}
			}
		})
	}
}

// TestDecodeCardNotSeekable checks the decoder rejects a stream it cannot
// rewind rather than handing back a half read card.
func TestDecodeCardNotSeekable(t *testing.T) {
	if _, err := decode(context.Background(), strings.NewReader("")); err == nil {
		t.Fatal("expected an error")
	}
}

// hrefRE matches the card's embedded cover art.
var hrefRE = regexp.MustCompile(`xlink:href="data:image/[a-z]+;base64,([^"]+)"`)

// embedded decodes the cover art the card svg embeds.
func embedded(t *testing.T, svg []byte) image.Image {
	t.Helper()
	m := hrefRE.FindSubmatch(svg)
	if m == nil {
		t.Fatal("expected the card to embed the cover art")
	}
	buf, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// loadImage loads an image from the test data.
func loadImage(t *testing.T, pathName string) image.Image {
	t.Helper()
	f, err := os.Open(pathName)
	if err != nil {
		t.Skipf("no test data: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// readAll reads the image's stream.
func readAll(img *decoder.Image) ([]byte, error) {
	var b bytes.Buffer
	_, err := b.ReadFrom(img.Reader)
	return b.Bytes(), err
}

func TestReduce(t *testing.T) {
	// a ramp from silence to full scale
	const n = 4000
	buf := make([]byte, n*2)
	for i := range n {
		v := int16(i * 32767 / n)
		buf[i*2], buf[i*2+1] = byte(v), byte(v>>8)
	}
	peaks := reduce(buf, 8)
	if len(peaks) != 8 {
		t.Fatalf("expected 8 peaks, got %d", len(peaks))
	}
	if peaks[7] != 1 {
		t.Errorf("expected the loudest peak to normalize to 1, got %v", peaks[7])
	}
	for i := 1; i < len(peaks); i++ {
		if peaks[i] <= peaks[i-1] {
			t.Errorf("expected peak %d (%v) above peak %d (%v)", i, peaks[i], i-1, peaks[i-1])
		}
	}
	if reduce(buf, n+1) != nil {
		t.Error("expected no peaks when there are fewer samples than bars")
	}
}

func TestReduceSilence(t *testing.T) {
	peaks := reduce(make([]byte, 2000), 8)
	if len(peaks) != 8 {
		t.Fatalf("expected 8 peaks, got %d", len(peaks))
	}
	for i, v := range peaks {
		if v != 0 {
			t.Errorf("expected peak %d of silence to be 0, got %v", i, v)
		}
	}
}

func TestRate(t *testing.T) {
	for _, test := range []struct {
		dur time.Duration
		exp int
	}{
		{0, maxRate},
		{time.Second, maxRate},
		{30 * time.Second, 1114},
		{time.Minute, minRate},
		{time.Hour, minRate},
	} {
		if got := rate(test.dur, bars); got != test.exp {
			t.Errorf("expected rate(%v) == %d, got %d", test.dur, test.exp, got)
		}
	}
}

func TestFit(t *testing.T) {
	for _, test := range []struct {
		s     string
		width int
		exp   string
	}{
		{"short", 1000, "short"},
		{"", 100, ""},
		{"truncated here", 40, "trunc…"},
		{"ab cdef", 26, "ab…"},
	} {
		if got := fit(test.s, 10, 0.6, test.width); got != test.exp {
			t.Errorf("expected fit(%q, %d) == %q, got %q", test.s, test.width, test.exp, got)
		}
	}
}

func TestShrink(t *testing.T) {
	sizes := []int{40, 20}
	if size, s := shrink("abcde", sizes, 0.5, 100); size != 40 || s != "abcde" {
		t.Errorf("expected the largest size to be kept, got %d %q", size, s)
	}
	if size, s := shrink("abcdefghij", sizes, 0.5, 100); size != 20 || s != "abcdefghij" {
		t.Errorf("expected a step down rather than a truncation, got %d %q", size, s)
	}
	if size, s := shrink(strings.Repeat("a", 40), sizes, 0.5, 100); size != 20 || !strings.HasSuffix(s, "…") {
		t.Errorf("expected a truncation at the smallest size, got %d %q", size, s)
	}
}

func TestClock(t *testing.T) {
	for _, test := range []struct {
		d   time.Duration
		exp string
	}{
		{0, "0:00"},
		{9 * time.Second, "0:09"},
		{212 * time.Second, "3:32"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
	} {
		if got := clock(test.d); got != test.exp {
			t.Errorf("expected clock(%v) == %q, got %q", test.d, test.exp, got)
		}
	}
}

func TestCardWithoutArtOrWaveform(t *testing.T) {
	c := &card{
		title:  "Title",
		artist: "Artist",
		accent: accentFor(nil, "Artist"),
	}
	svg := string(c.svg())
	for _, s := range []string{"Title", "Artist", "<svg", "</svg>", emptyWave} {
		if !strings.Contains(svg, s) {
			t.Errorf("expected the card to contain %q", s)
		}
	}
	if strings.Contains(svg, "base64") {
		t.Error("expected no embedded art")
	}
}

func TestCardEscapes(t *testing.T) {
	c := &card{title: `A & B <c> "d"`, accent: defaultAccent}
	if svg := string(c.svg()); strings.Contains(svg, "<c>") || !strings.Contains(svg, "&amp;") {
		t.Error("expected the title to be escaped")
	}
}

func TestAccentFromArt(t *testing.T) {
	if _, ok := accentFromArt(nil); ok {
		t.Error("expected no accent from no art")
	}
	if _, ok := accentFromArt([]byte("not an image")); ok {
		t.Error("expected no accent from undecodable art")
	}
	// the same text always derives the same accent
	if a, b := accentFor(nil, "x"), accentFor(nil, "x"); a != b {
		t.Errorf("expected a stable accent, got %v and %v", a, b)
	}
	if a, b := accentFor(nil, "x"), accentFor(nil, "y"); a == b {
		t.Error("expected different text to derive different accents")
	}
}

func TestHsvHex(t *testing.T) {
	for _, test := range []struct {
		h, s, v float64
		exp     string
	}{
		{0, 0, 0, "#000000"},
		{0, 0, 1, "#ffffff"},
		{0, 1, 1, "#ff0000"},
		{120, 1, 1, "#00ff00"},
		{240, 1, 1, "#0000ff"},
	} {
		if got := hsvHex(test.h, test.s, test.v); got != test.exp {
			t.Errorf("expected hsvHex(%v, %v, %v) == %q, got %q", test.h, test.s, test.v, test.exp, got)
		}
	}
}

// pngBytes encodes a 1x1 png of the given color.
func pngBytes(t *testing.T, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, c)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return b.Bytes()
}

// write writes a file into dir, returning nothing -- a failure to write test
// data is a test failure.
func write(t *testing.T, dir, name string, buf []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSidecarNone(t *testing.T) {
	if buf, _ := sidecar(context.Background(), ""); buf != nil {
		t.Error("expected no cover without a path")
	}
	dir := t.TempDir()
	write(t, dir, "track.mp3", []byte("audio"))
	if buf, _ := sidecar(context.Background(), filepath.Join(dir, "track.mp3")); buf != nil {
		t.Error("expected no cover in a directory without one")
	}
	if buf, _ := sidecar(context.Background(), filepath.Join(dir, "nope", "track.mp3")); buf != nil {
		t.Error("expected no cover from a directory that does not exist")
	}
}

func TestSidecarFinds(t *testing.T) {
	for _, name := range []string{"cover.png", "Cover.PNG", "folder.png", "FRONT.png", "artwork.png"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			want := pngBytes(t, color.RGBA{1, 2, 3, 255})
			write(t, dir, name, want)
			buf, mime := sidecar(context.Background(), filepath.Join(dir, "track.mp3"))
			if !bytes.Equal(buf, want) {
				t.Errorf("expected %s to be found", name)
			}
			if mime != "image/png" {
				t.Errorf("expected mime %q, got %q", "image/png", mime)
			}
		})
	}
}

func TestSidecarPrefersCover(t *testing.T) {
	dir := t.TempDir()
	want := pngBytes(t, color.RGBA{1, 2, 3, 255})
	write(t, dir, "cover.jpg", want)
	write(t, dir, "folder.png", pngBytes(t, color.RGBA{9, 9, 9, 255}))
	buf, mime := sidecar(context.Background(), filepath.Join(dir, "track.mp3"))
	if !bytes.Equal(buf, want) {
		t.Error("expected cover to be preferred over folder")
	}
	// the name is only a hint about where to look: the content decides
	if mime != "image/png" {
		t.Errorf("expected the sniffed mime %q, got %q", "image/png", mime)
	}
}

func TestSidecarSkipsUnusable(t *testing.T) {
	dir := t.TempDir()
	want := pngBytes(t, color.RGBA{1, 2, 3, 255})
	// a cover.jpg holding something that is not an image at all, and a
	// directory in the way of the next candidate
	write(t, dir, "cover.jpg", []byte("this is not an image"))
	if err := os.Mkdir(filepath.Join(dir, "front.png"), 0o755); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	write(t, dir, "album.png", want)
	buf, _ := sidecar(context.Background(), filepath.Join(dir, "track.mp3"))
	if !bytes.Equal(buf, want) {
		t.Error("expected the unusable candidates to be skipped")
	}
}

func TestSidecarSkipsOversized(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "cover.png", make([]byte, maxCover+1))
	if buf, _ := sidecar(context.Background(), filepath.Join(dir, "track.mp3")); buf != nil {
		t.Error("expected an oversized cover to be skipped")
	}
}

// TestSidecarNotUsedWhenEmbedded checks that a picture in the tags wins over
// a cover sitting next to the file.
func TestSidecarNotUsedWhenEmbedded(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "tag", "silent.flac")
	audio, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("no test data: %v", err)
	}
	write(t, dir, "silent.flac", audio)
	write(t, dir, "cover.png", pngBytes(t, color.RGBA{1, 2, 3, 255}))
	pathName := filepath.Join(dir, "silent.flac")
	f, err := os.Open(pathName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer f.Close()
	res, err := decode(ivctx.WithPathName(context.Background(), pathName), f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	buf, err := readAll(res.(*decoder.Image))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := embedded(t, buf).Bounds().Size(); got == image.Pt(1, 1) {
		t.Error("expected the embedded picture rather than the sidecar cover")
	}
}
