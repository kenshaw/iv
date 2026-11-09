package okage

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
	"github.com/tdewolff/canvas"
	"github.com/xo/okage/svg"
)

// TestSerifIsLatin checks that text in the serif family draws with the serif
// candidate that is installed, and not with a symbol font. graphviz writes
// its labels as font-family="Times,serif", and resvg drew them as greek
// letters on a system with the urw configs of fontconfig. okage sets serif to
// the first of [svg.SerifCandidates] that is installed.
//
// The test names a family that no system has in front of serif, so that the
// text falls back to serif as "Times" does where it is not installed. macOS
// has a font called Times, which draws "Times,serif" correctly in Times.
//
// It broke once without a sound: resvg v0.12.1 ignored a change to a
// renderer after it was made, and the families were set on resvg.Default.
func TestSerifIsLatin(t *testing.T) {
	var name string
	for _, candidate := range svg.SerifCandidates {
		if _, ok := canvas.FindSystemFont(candidate, canvas.FontRegular); ok {
			name = candidate
			break
		}
	}
	if name == "" {
		t.Fatal("no serif candidate is installed: " + strings.Join(svg.SerifCandidates, ", "))
	}
	ctx := ivctx.WithConfig(context.Background(), ivctx.New())
	state, err := open(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	r := state.(*svg.Renderer)
	got := render(t, r, "No Such Font,serif")
	exp := render(t, r, name)
	if !same(got, exp) {
		t.Errorf("expected the text in No Such Font,serif to draw with %s", name)
	}
}

// render renders a line of text in the font family.
func render(t *testing.T, r *svg.Renderer, family string) *image.RGBA {
	t.Helper()
	data := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="30">`+
		`<text x="2" y="22" font-family="%s" font-size="18">book_id: integer</text></svg>`, family)
	img, err := r.Render([]byte(data))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return img
}

// same returns true when a and b hold the same pixels.
func same(a, b *image.RGBA) bool {
	if a.Rect != b.Rect || len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
