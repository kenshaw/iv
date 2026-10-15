package resvg

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
	"github.com/tdewolff/canvas"
	"github.com/xo/resvg"
)

// TestSerifIsLatin checks that text in the serif family draws with the serif
// candidate that is installed, and not with a symbol font. graphviz writes
// its labels as font-family="Times,serif", and a system with the urw configs
// of fontconfig drew them as greek letters (see configureFamilies).
//
// It broke once without a sound: resvg v0.12.1 ignores a change to a
// renderer after it is made, and the families were set on [resvg.Default].
func TestSerifIsLatin(t *testing.T) {
	var name string
	for _, candidate := range serif {
		if _, ok := canvas.FindSystemFont(candidate, canvas.FontRegular); ok {
			name = candidate
			break
		}
	}
	if name == "" {
		t.Fatal("no serif candidate is installed: " + strings.Join(serif, ", "))
	}
	ctx := ivctx.WithConfig(context.Background(), ivctx.New())
	state, err := open(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	r := state.(*resvg.Resvg)
	defer r.Close()
	got := render(t, r, "Times,serif")
	exp := render(t, r, name)
	if !same(got, exp) {
		t.Errorf("expected the text in Times,serif to draw with %s", name)
	}
}

// render renders a line of text in the font family.
func render(t *testing.T, r *resvg.Resvg, family string) *image.RGBA {
	t.Helper()
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="30">`+
		`<text x="2" y="22" font-family="%s" font-size="18">book_id: integer</text></svg>`, family)
	img, err := r.Render([]byte(svg))
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
