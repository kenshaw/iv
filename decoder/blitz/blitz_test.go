package blitz

import "testing"

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
