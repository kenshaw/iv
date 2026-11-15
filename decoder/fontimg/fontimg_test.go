package fontimg

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/okage/svg"
	"github.com/xo/zuzu/text/emoji"
)

// TestDecode draws a specimen of each font in testdata/fontimg. The WOFF and
// the WOFF2 of Noto Mono draw the same pixels as its TrueType (D48), and
// IBMPlexSans.woff2 is a collection in WOFF2.
func TestDecode(t *testing.T) {
	ttf := testDecode(t, testContext(t), "NotoMono-Regular.ttf")
	for _, name := range []string{"Figtree-Italic[wght].ttf", "Ubuntu-R.ttf", "NotoMono-Regular.woff", "NotoMono-Regular.woff2", "IBMPlexSans.woff2"} {
		t.Run(name, func(t *testing.T) {
			img := testDecode(t, testContext(t), name)
			if strings.HasPrefix(name, "NotoMono") && !bytes.Equal(img.Pix, ttf.Pix) {
				t.Error("expected the pixels of NotoMono-Regular.ttf")
			}
		})
	}
}

// TestCollection checks that --page selects a face of a collection, in a
// WOFF2 that holds the six faces of IBM Plex Sans, and that each face draws
// differently.
func TestCollection(t *testing.T) {
	exp := []string{
		"IBM Plex Sans, Regular, v3.005",
		"IBM Plex Sans, Medium, v3.005",
		"IBM Plex Sans, SemiBold, v3.005",
		"IBM Plex Sans, Italic, v3.005",
		"IBM Plex Sans, Medium Italic, v3.005",
		"IBM Plex Sans, SemiBold Italic, v3.005",
	}
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fontimg", "IBMPlexSans.woff2"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	db := svg.NewFontDatabase()
	ids := db.LoadFontData(buf)
	if len(ids) != len(exp) {
		t.Fatalf("expected %d faces, got %d", len(exp), len(ids))
	}
	seen := make(map[string]bool)
	for i, id := range ids {
		info, _ := db.Face(id)
		if head := specimen(db, id, info, "", 48, 0, noEmoji)[0].text; head != exp[i] {
			t.Errorf("face %d: expected %q, got %q", i+1, exp[i], head)
		}
		ctx := testContext(t)
		ivctx.Get(ctx).Page = uint(i + 1)
		img := testDecode(t, ctx, "IBMPlexSans.woff2")
		if seen[string(img.Pix)] {
			t.Errorf("face %d: expected a face of its own, got the pixels of another", i+1)
		}
		seen[string(img.Pix)] = true
	}
}

// TestSpecimen checks the lines of the specimen: the family, the style, the
// version and the weight of --font-weight, then the sample text of the font,
// the characters, and the pangrams at growing sizes. A font with a sample
// text of its own, as IBM Plex Sans has, gets every line too (D48).
func TestSpecimen(t *testing.T) {
	for _, test := range []struct {
		name   string
		weight uint
		head   string
		sample string
	}{
		{"Ubuntu-R.ttf", 0, "Ubuntu, Regular, v0.83", ""},
		{"Figtree-Italic[wght].ttf", 0, "Figtree, Light Italic, v2.002", ""},
		{"Figtree-Italic[wght].ttf", 900, "Figtree, Light Italic, v2.002, weight 900", ""},
		{"IBMPlexSans.woff2", 0, "IBM Plex Sans, Regular, v3.005", "How razorback-jumping frogs can level six piqued gymnasts!"},
	} {
		t.Run(test.head, func(t *testing.T) {
			db, id, info := testFace(t, test.name)
			exp := []line{{test.head, 50, false}}
			if test.sample != "" {
				exp = append(exp, line{test.sample, 48, false})
			}
			for _, l := range slices.Concat(characters, pangrams) {
				exp = append(exp, line{l.text, 48 + l.size, false})
			}
			if lines := specimen(db, id, info, test.name, 48, test.weight, noEmoji); !slices.Equal(lines, exp) {
				t.Errorf("expected %v, got %v", exp, lines)
			}
		})
	}
}

// TestEmoji checks that a font of emoji, Noto Color Emoji of zuzu, gets its
// name in the font of the system, then the line of emoji at the sizes of the
// pangrams, drawn in color. It has no letters, so it gets no other line, and
// IBM Plex Sans, which has no color emoji, gets no line of emoji.
func TestEmoji(t *testing.T) {
	db := svg.NewFontDatabase()
	ids := db.LoadFontData(emoji.Font().Data())
	if len(ids) == 0 {
		t.Fatal("no face in Noto Color Emoji")
	}
	info, _ := db.Face(ids[0])
	style, err := textStyle(ivctx.New(), info)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if kind := emojiOf(db, style); kind != onlyEmoji {
		t.Fatalf("expected Noto Color Emoji to be a font of emoji, got %d", kind)
	}
	lines := specimen(db, ids[0], info, "", 48, 0, onlyEmoji)
	exp := []line{{"Noto Color Emoji, Regular, v2.057", 50, true}}
	for _, l := range pangrams {
		exp = append(exp, line{emojiLine, 48 + l.size, false})
	}
	if !slices.Equal(lines, exp) {
		t.Errorf("expected %v, got %v", exp, lines)
	}
	res, err := decode(testContext(t), bytes.NewReader(emoji.Font().Data()))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	img, _ := res.(*image.RGBA)
	switch {
	case !hasColor(img):
		t.Error("expected pixels in color")
	case img.Bounds().Dx() > 2500:
		t.Errorf("expected about the width of the line of emoji at 66 points, got %d", img.Bounds().Dx())
	}
	plex, _, plexInfo := testFace(t, "IBMPlexSans.woff2")
	plexStyle, _ := textStyle(ivctx.New(), plexInfo)
	if kind := emojiOf(plex, plexStyle); kind != noEmoji {
		t.Errorf("expected no color emoji in IBM Plex Sans, got %d", kind)
	}
}

// hasColor returns true when a pixel of img is not gray.
func hasColor(img *image.RGBA) bool {
	for i := 0; i+2 < len(img.Pix); i += 4 {
		r, g, b := int(img.Pix[i]), int(img.Pix[i+1]), int(img.Pix[i+2])
		if max(r, g, b)-min(r, g, b) > 64 {
			return true
		}
	}
	return false
}

// TestWeight checks that --font-weight sets the wght axis of a variable font,
// so that a heavier weight draws wider.
func TestWeight(t *testing.T) {
	var widths []int
	for _, weight := range []uint{100, 900} {
		ctx := testContext(t)
		ivctx.Get(ctx).FontWeight = weight
		widths = append(widths, testDecode(t, ctx, "Figtree-Italic[wght].ttf").Bounds().Dx())
	}
	if widths[0] >= widths[1] {
		t.Errorf("expected weight 900 wider than weight 100, got %v", widths)
	}
}

// TestErrors checks that data with no font and a weight above 1000 are
// errors.
func TestErrors(t *testing.T) {
	if _, err := decode(testContext(t), bytes.NewReader(make([]byte, 64))); !errors.Is(err, ErrNoFace) {
		t.Errorf("expected %v, got: %v", ErrNoFace, err)
	}
	ctx := testContext(t)
	ivctx.Get(ctx).FontWeight = 1001
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fontimg", "Ubuntu-R.ttf"))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	if _, err := decode(ctx, bytes.NewReader(buf)); !errors.Is(err, ErrInvalidWeight) {
		t.Errorf("expected %v, got: %v", ErrInvalidWeight, err)
	}
}

// testDecode draws the font in testdata/fontimg.
func testDecode(t *testing.T, ctx context.Context, name string) *image.RGBA {
	t.Helper()
	pathName := filepath.Join("..", "..", "testdata", "fontimg", name)
	buf, err := os.ReadFile(pathName)
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	res, err := decode(ivctx.WithPathName(ctx, pathName), bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	img, ok := res.(*image.RGBA)
	if !ok {
		t.Fatalf("expected *image.RGBA, got %T", res)
	}
	if b := img.Bounds(); b.Dx() < 100 || b.Dy() < 100 {
		t.Errorf("expected a specimen, got %v", b)
	}
	return img
}

// testFace loads the font in testdata/fontimg into a database of its own.
func testFace(t *testing.T, name string) (*svg.FontDatabase, svg.FontID, *svg.FaceInfo) {
	t.Helper()
	buf, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fontimg", name))
	if err != nil {
		t.Fatalf("no test data: %v", err)
	}
	db := svg.NewFontDatabase()
	ids := db.LoadFontData(buf)
	if len(ids) == 0 {
		t.Fatalf("no face in %s", name)
	}
	info, _ := db.Face(ids[0])
	return db, ids[0], info
}

// testContext returns the context of a test, with the default configuration
// of iv.
func testContext(t *testing.T) context.Context {
	t.Helper()
	return ivctx.WithConfig(t.Context(), ivctx.New())
}
