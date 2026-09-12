package lottie

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// TestDetect checks what the sniffer claims. A .json file reaches iv with a
// name that says nothing, so the document itself has to be the evidence: the
// documents here are what separates a lottie from the rest of the json in the
// world, and every one of them is something the sniffer could plausibly be
// handed.
func TestDetect(t *testing.T) {
	for _, test := range []struct {
		name string
		src  string
		exp  string
	}{
		{
			"lottie",
			`{"v":"5.8.1","fr":60,"ip":0,"op":40,"w":512,"h":512,"nm":"x","layers":[]}`,
			mime,
		},
		{
			// the identifying properties are not required to come first, or
			// in any particular order
			"lottie reordered",
			`{"nm":"x","assets":[{"id":"a","layers":[]}],"h":512,"w":512,"op":40,"ip":0,"fr":60}`,
			mime,
		},
		{
			// the head of the document is all the sniffer sees, and it is
			// enough as long as the identifying properties are in it
			"lottie truncated after the header",
			`{"v":"5.8.1","fr":60,"ip":0,"op":40,"w":512,"h":512,"nm":"x","layers":[{"ind`,
			mime,
		},
		{
			"package.json",
			`{"name":"x","version":"1.0.0","scripts":{"build":"tsc"},"dependencies":{}}`,
			"",
		},
		{
			// a cut down animation is still not one: every identifying
			// property has to be there
			"some of the properties",
			`{"v":"5.8.1","fr":60,"ip":0,"op":40,"nm":"x","layers":[]}`,
			"",
		},
		{
			// only the top level counts -- a layer's own geometry says
			// nothing about the document holding it
			"properties nested in a layer",
			`{"name":"x","layers":[{"fr":60,"ip":0,"op":40,"w":512,"h":512}]}`,
			"",
		},
		{
			// the names are ordinary enough to turn up together elsewhere,
			// so the values have to be numbers as well
			"the properties are not numbers",
			`{"fr":"60","ip":"0","op":"40","w":"512","h":"512"}`,
			"",
		},
		{"array", `[{"fr":60,"ip":0,"op":40,"w":512,"h":512}]`, ""},
		{"not json", "<svg xmlns=\"http://www.w3.org/2000/svg\"/>", ""},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := detect(context.Background(), bytes.NewReader([]byte(test.src)))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != test.exp {
				t.Errorf("expected mime %q, got %q", test.exp, got)
			}
		})
	}
}

// TestHeaderSize checks the composition size is read from the document, since
// it is what the render is scaled against.
func TestHeaderSize(t *testing.T) {
	w, h, ok := header([]byte(`{"v":"5.8.1","fr":60,"ip":0,"op":40,"w":800,"h":600}`))
	if !ok {
		t.Fatal("expected a lottie")
	}
	if w != 800 || h != 600 {
		t.Errorf("expected 800x600, got %dx%d", w, h)
	}
}

// TestDecode renders each test file, checking that the three ways a lottie
// reaches iv all arrive at an image: a bare document, one behind an extension
// that content sniffing cannot read, and a dotLottie archive.
func TestDecode(t *testing.T) {
	for _, test := range []struct {
		name   string
		file   string
		frames int
	}{
		{"json", "rocket.json", 40},
		{"lot", "star.lot", 146},
		{"lottie", "fire.lottie", 65},
	} {
		t.Run(test.name, func(t *testing.T) {
			img := render(t, testContext(t), test.file)
			if got := img.Bounds().Size(); got != (image.Point{X: 1024, Y: 1024}) {
				t.Errorf("expected 1024x1024, got %v", got)
			}
			if !opaqueSomewhere(img) {
				t.Error("expected the frame to have something drawn on it")
			}
		})
	}
}

// TestDecodePage checks the page selects the frame, and that the frames it
// selects actually differ -- an animation rendered at a fixed frame whatever
// was asked for would otherwise pass every other test here.
func TestDecodePage(t *testing.T) {
	ctx := testContext(t)
	mid := render(t, ctx, "rocket.json")
	// not the first frame: the rocket's loop is symmetric enough that frame 0
	// and the midpoint come out identical, which would say nothing either way
	ivctx.Get(ctx).Page = 6
	if same(mid, render(t, ctx, "rocket.json")) {
		t.Error("expected the selected frame to differ from the middle one")
	}
	// a page past the end is no page at all, and falls back to the middle
	ivctx.Get(ctx).Page = 1000
	if !same(mid, render(t, ctx, "rocket.json")) {
		t.Error("expected an out of range page to render the middle frame")
	}
}

// TestFit checks the render size follows the configured display size, keeping
// the animation's aspect ratio. A zero leaves the renderer on the
// composition's own size.
func TestFit(t *testing.T) {
	for _, test := range []struct {
		name   string
		w, h   uint
		nw, nh int
		expW   int
		expH   int
	}{
		{"unset", 0, 0, 1024, 1024, 0, 0},
		{"both", 200, 200, 1024, 1024, 200, 200},
		{"wide composition", 400, 400, 800, 400, 400, 200},
		{"tall composition", 400, 400, 400, 800, 200, 400},
		// the other axis is left on its minimum, which then bounds the fit --
		// the same as iv sizes an svg
		{"width only", 200, 0, 1024, 1024, 64, 64},
		{"no composition size", 200, 200, 0, 0, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := ivctx.New()
			c.Width, c.Height = test.w, test.h
			ctx := ivctx.WithConfig(context.Background(), c)
			w, h := fit(ctx, test.nw, test.nh)
			if w != test.expW || h != test.expH {
				t.Errorf("expected %dx%d, got %dx%d", test.expW, test.expH, w, h)
			}
		})
	}
}

// render decodes a test file through the whole pipeline, so that what is
// checked is the file as iv is given it -- sniffed, routed, and rendered --
// rather than this package called directly.
func render(t *testing.T, ctx context.Context, name string) image.Image {
	t.Helper()
	pathName := filepath.Join("..", "..", "testdata", "lottie", name)
	if _, err := os.Stat(pathName); err != nil {
		t.Skipf("no test data: %v", err)
	}
	img, _, err := decoder.DecodeFile(ctx, pathName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// testContext returns a context carrying a config of its own, so that a test
// changing one does not change another's.
func testContext(t *testing.T) context.Context {
	t.Helper()
	c := ivctx.New()
	if testing.Verbose() {
		c.Logger = func(s string, v ...any) { t.Logf(s, v...) }
	}
	return ivctx.WithConfig(context.Background(), c)
}

// opaqueSomewhere reports whether anything was drawn on the frame.
func opaqueSomewhere(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				return true
			}
		}
	}
	return false
}

// same reports whether two frames are identical.
func same(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

// TestMain releases the renderer the way iv does once the tests are done, and
// checks that what was released was the real one.
//
// The pipeline hands a close func its decoder's state -- a decoder handed nil
// instead reports no error here and leaves a working renderer behind, which is
// what the close in this package used to do.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		code = checkClose()
	}
	os.Exit(code)
}

// checkClose builds the renderer through the pipeline, closes it, and checks
// that it is gone: the webassembly runtime goes with it, so nothing decodes
// afterwards.
func checkClose() int {
	ctx := context.Background()
	pathName := filepath.Join("..", "..", "testdata", "lottie", "rocket.json")
	if _, err := os.Stat(pathName); err != nil {
		return 0
	}
	if _, _, err := decoder.DecodeFile(ctx, pathName); err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		return 1
	}
	if err := decoder.Close(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "decoder close: %v\n", err)
		return 1
	}
	if _, _, err := decoder.DecodeFile(ctx, pathName); err == nil {
		fmt.Fprintln(os.Stderr, "decoder close: the renderer still decodes, so it was never closed")
		return 1
	}
	return 0
}
