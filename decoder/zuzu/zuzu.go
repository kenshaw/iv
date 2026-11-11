// Package zuzu supplies a DOT graph and Mermaid diagram decoder for iv.
//
// zuzu draws a graph or a diagram in Go, with no external command and no
// WebAssembly (D36).
//
// See: https://github.com/xo/zuzu
package zuzu

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/zuzu"
	"github.com/xo/zuzu/icons/carbon"
	"github.com/xo/zuzu/icons/fa"
	"github.com/xo/zuzu/icons/k8s"
	"github.com/xo/zuzu/icons/mdi"
	"github.com/xo/zuzu/theme"
)

// The mime types of the two dialects.
const (
	dotMime     = "text/vnd.graphviz"
	mermaidMime = "text/vnd.mermaid"
)

// exts are the extensions of the two dialects.
var exts = []string{"gv", "dot", "mmd", "mermaid"}

// peek is how many leading bytes are sniffed. zuzu needs no more than this.
const peek = 512

// scale is the ratio of the pixels of the image to the pixels of the
// diagram. zuzu lays a diagram out for a screen, and at 1 its lines and text
// are blurred in a terminal, which shows each pixel larger. At 2, the layout
// does not change, and the image is twice as wide and twice as high.
const scale = 2

// ErrUnknownTheme is the error of a --mermaid-theme that names no theme.
var ErrUnknownTheme = errors.New("unknown theme")

func init() {
	decoder.Register(
		"zuzu",
		decoder.Desc("DOT graphs and Mermaid diagrams"),
		decoder.MimeType(dotMime, mermaidMime),
		// the detector does not claim a diagram with front matter, or a graph
		// whose first 512 bytes are comments, so the extension routes it.
		// libmagic types such a file as any kind of text, even text/html. A
		// .dot that is not text is a template of Microsoft Word
		decoder.Extension(exts...),
		decoder.Matcher(func(mime, ext string) bool {
			return strings.HasPrefix(mime, "text/") && slices.Contains(exts, ext)
		}),
		decoder.MimeDetector(detect),
		decoder.Decoder(decode),
	)
}

// detect sniffs the reader for a DOT graph or a Mermaid diagram.
//
// It does not claim a source whose first line is ---. zuzu reads that line as
// the front matter of a Mermaid diagram, but a Markdown or a YAML file starts
// the same way, and a claim here would keep it from libmagic and from the
// Markdown decoder.
//
// A short read is not an error here. A stream that cannot be read is not a
// diagram, and the sniffer says so.
func detect(_ context.Context, r io.Reader) (string, error) {
	buf, _ := bufio.NewReaderSize(r, peek).Peek(peek)
	switch d, _ := zuzu.Sniff(buf); d {
	case zuzu.DOT:
		return dotMime, nil
	case zuzu.Mermaid:
		if line, _, _ := strings.Cut(strings.TrimSpace(string(buf)), "\n"); strings.TrimSpace(line) != "---" {
			return mermaidMime, nil
		}
	}
	return "", nil
}

// decode draws the graph or the diagram as an image.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	dialect := dialectOf(ctx, buf)
	opts := []zuzu.RenderOption{
		zuzu.WithScale(scale),
		// the flowcharts of mermaid name the solid icons of font awesome as
		// fa: and as fas:. Each pack is read on its first use
		zuzu.WithIcons("fa", fa.Pack()),
		zuzu.WithIcons("fas", fa.Pack()),
		zuzu.WithIcons("mdi", mdi.Pack()),
		zuzu.WithIcons("carbon", carbon.Pack()),
		zuzu.WithIcons("k8s", k8s.Pack()),
	}
	if dialect == zuzu.Mermaid {
		c := ivctx.Get(ctx)
		mode, err := themeMode(c.MermaidTheme)
		if err != nil {
			return nil, err
		}
		opts = append(opts, zuzu.WithMode(mode))
		if c.MermaidBg != nil {
			opts = append(opts, zuzu.WithBackground(c.MermaidBg))
		}
	}
	chart, report, err := zuzu.Parse(buf, zuzu.WithDialect(dialect))
	logReport(ctx, report)
	if err != nil {
		return nil, err
	}
	img, report, err := chart.Image(ctx, opts...)
	logReport(ctx, report)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// dialectOf returns the dialect of a source. The extension decides, and
// without one, the content does. A source that reached this decoder and is
// not DOT came by the mime type of mermaid.
func dialectOf(ctx context.Context, buf []byte) zuzu.Dialect {
	switch ivctx.FileExt(ivctx.PathName(ctx)) {
	case "gv", "dot":
		return zuzu.DOT
	case "mmd", "mermaid":
		return zuzu.Mermaid
	}
	if d, _ := zuzu.Sniff(buf); d == zuzu.DOT {
		return zuzu.DOT
	}
	return zuzu.Mermaid
}

// themeMode returns the mode of zuzu for a --mermaid-theme. zuzu has a light
// and a dark mode, so each theme of mermaid except dark draws light.
func themeMode(name string) (theme.Mode, error) {
	switch name {
	case "", "default", "forest", "neutral", "modern":
		return theme.Light, nil
	case "dark":
		return theme.Dark, nil
	}
	return theme.Light, fmt.Errorf("mermaid theme %q: %w", name, ErrUnknownTheme)
}

// logReport logs each part of the source that zuzu did not draw.
func logReport(ctx context.Context, report zuzu.Report) {
	for _, issue := range report {
		ivctx.Logf(ctx, "zuzu: %v", issue)
	}
}
