package blitz

import (
	"context"
	"html"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/transit"
	"github.com/xo/transit/styles"
)

// TestIsHTML checks that the extension decides how a document is parsed
// before the mime type, and that the content decides only when neither does.
func TestIsHTML(t *testing.T) {
	for _, test := range []struct {
		name     string
		mime     string
		pathName string
		src      string
		exp      bool
	}{
		{"markdown that starts with html", "text/html", "README.md", "<div align=\"center\">\n</div>\n\n# Title\n", false},
		{"markdown", "text/x-c", "sample.md", "# Title\n", false},
		{"html", "text/html", "sample.html", "<!doctype html>\n<html></html>\n", true},
		{"html with a markdown mime", "text/markdown", "page.htm", "<p>text</p>\n", true},
		{"html mime", "text/html", "", "<div></div>\n", true},
		{"markdown mime", "text/markdown", "", "# Title\n", false},
		{"text that starts with doctype", "text/plain", "page.txt", "  <!DOCTYPE html>\n<html></html>\n", true},
		{"plain text", "text/plain", "notes.txt", "Some notes.\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isHTML(test.mime, test.pathName, []byte(test.src)); got != test.exp {
				t.Errorf("expected %t, got %t", test.exp, got)
			}
		})
	}
}

// TestPages checks that an image taller than a page is cut into pages of the
// page height, that the rows of each page are the rows of the image, and that
// an image that fits is not cut.
func TestPages(t *testing.T) {
	// each row holds its own number, so that a page shows which rows it got
	img := image.NewRGBA(image.Rect(0, 0, 4, 10))
	for y := range 10 {
		for x := range 4 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(y), A: 255})
		}
	}
	for _, test := range []struct {
		name          string
		width, height uint
		exp           []int
	}{
		{"three pages", 4, 4, []int{4, 4, 2}},
		{"half the width", 2, 2, []int{4, 4, 2}},
		{"exact pages", 4, 5, []int{5, 5}},
		{"fits", 4, 10, nil},
		{"no display height", 4, 0, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := ivctx.New()
			c.Width, c.Height = test.width, test.height
			res := pages(ivctx.WithConfig(context.Background(), c), img)
			if test.exp == nil {
				if res != img {
					t.Fatalf("expected the image, got %T", res)
				}
				return
			}
			v, ok := res.([]image.Image)
			if !ok {
				t.Fatalf("expected pages, got %T", res)
			}
			if len(v) != len(test.exp) {
				t.Fatalf("expected %d pages, got %d", len(test.exp), len(v))
			}
			var y uint8
			for i, page := range v {
				if b := page.Bounds(); b != image.Rect(0, 0, 4, test.exp[i]) {
					t.Fatalf("page %d: expected 4x%d at the origin, got %v", i+1, test.exp[i], b)
				}
				for py := range test.exp[i] {
					if got := page.(*image.RGBA).RGBAAt(3, py).R; got != y {
						t.Fatalf("page %d row %d: expected row %d of the image, got %d", i+1, py, y, got)
					}
					y++
				}
			}
		})
	}
}

// TestHighlight checks that the highlight query of every grammar compiles,
// and that a snippet of each language comes back colored and escaped.
func TestHighlight(t *testing.T) {
	snippets := map[string]string{
		"bash":       "echo \"<$HOME>\"\n",
		"c":          "int main(void) { return 1 < 2; }\n",
		"cpp":        "template <typename T> class A { public: T v; };\n",
		"go":         "func main() { _ = 1 < 2 }\n",
		"html":       "<p class=\"a\">a &amp; b</p>\n",
		"java":       "class A { boolean b = 1 < 2; }\n",
		"javascript": "const a = 1 < 2;\n",
		"json":       "{\"a\": \"<b>\", \"c\": 1}\n",
		"python":     "def f():\n    return 1 < 2\n",
		"ruby":       "def f\n  1 < 2\nend\n",
		"rust":       "fn main() { let a = 1 < 2; }\n",
		"sql":        "select a from b where c < 2;\n",
		"typescript": "const a: number = 1 < 2 ? 1 : 2;\n",
	}
	if len(snippets) != len(grammars) {
		t.Fatalf("expected a snippet for each of the %d grammars, got %d", len(grammars), len(snippets))
	}
	style, ok := styles.Get(lightStyle)
	if !ok {
		t.Fatalf("no style %q", lightStyle)
	}
	ctx := context.Background()
	parser, cursor := transit.NewParser(), transit.NewQueryCursor()
	for name, code := range snippets {
		t.Run(name, func(t *testing.T) {
			s, err := highlight(ctx, parser, cursor, style, name, code)
			switch {
			case err != nil:
				t.Fatalf("expected no error, got: %v", err)
			case !strings.Contains(s, `<span style="color:#`):
				t.Errorf("expected a colored span, got: %s", s)
			case strings.Contains(s, "<2") || strings.Contains(s, "< 2") || strings.Contains(s, `"<b>"`):
				t.Errorf("expected the code to be escaped, got: %s", s)
			}
			if got := spanText(s); got != code {
				t.Errorf("expected the text of the code, got: %q", got)
			}
		})
	}
}

// TestHighlightLanguage checks how the info string of a block names its
// language.
func TestHighlightLanguage(t *testing.T) {
	style, _ := styles.Get(lightStyle)
	ctx := context.Background()
	parser, cursor := transit.NewParser(), transit.NewQueryCursor()
	for _, test := range []struct {
		lang    string
		colored bool
	}{
		{"go", true},
		{"Go", true},
		{"golang", true},
		{"rust,ignore", true},
		{"js{1,3}", true},
		{"sh", true},
		{"postgresql", true},
		{"tsql", true},
		{"usql", true},
		{"clickhouse", false},
		{"neo4j", false},
		{"ydb", false},
		{"brainfuck", false},
		{"", false},
	} {
		t.Run(test.lang, func(t *testing.T) {
			s, err := highlight(ctx, parser, cursor, style, test.lang, "func main() {}\n")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if colored := s != ""; colored != test.colored {
				t.Errorf("expected colored %t, got %q", test.colored, s)
			}
		})
	}
}

// spanText returns the text of highlighted html, with the spans removed and
// the escapes undone.
func spanText(s string) string {
	for {
		i := strings.Index(s, "<span")
		if i < 0 {
			break
		}
		j := strings.Index(s[i:], ">")
		s = s[:i] + s[i+j+1:]
	}
	return html.UnescapeString(strings.ReplaceAll(s, "</span>", ""))
}
