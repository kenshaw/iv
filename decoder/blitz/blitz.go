// Package blitz supplies a html and markdown decoder for iv, rendering
// documents and remote pages with the blitz browser engine.
//
// See: https://github.com/xo/blitz
package blitz

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/blitz"
)

// Render geometry. The viewport is a fixed width so a document lays out the
// same whatever the terminal is, and the scale oversamples it so the result
// stays sharp once iv fits it to the terminal. The viewport height is left at
// blitz's browser-like default: the render still grows to fit the document,
// but a page sized in vh units resolves against something sensible.
const (
	width = 1200
	scale = 2
	// maxRows is the most pixel rows that blitz draws. Its renderer keeps the
	// height of an image in 16 bits, and rounds it up to a whole tile of 4
	// rows, so a render of more than 65532 rows fails. A document is drawn
	// whole up to this height, and iv cuts it into pages. At scale 2, 65532
	// rows of 2400 pixels are 600 MB.
	maxRows = 65532
)

// UserAgent is the user agent sent with requests.
var UserAgent = "iv"

func init() {
	decoder.Register(
		"blitz",
		decoder.Desc("Markdown and html documents"),
		// text/plain is shared with mermaid, csv and dot, so this is the last
		// plain text decoder tried
		decoder.After("zuzu", "libreoffice"),
		decoder.Extension("md", "markdown", "mkd", "mdown", "txt", "html", "htm", "xhtml"),
		decoder.MimeType(
			"text/markdown",
			"text/x-markdown",
			"text/html",
			"application/xhtml+xml",
		),
		decoder.Matcher(func(mime, _ string) bool {
			return mime == "text/plain"
		}),
		decoder.Init(open, shut),
		decoder.Decoder(decode),
	)
	decoder.Register(
		"blitz-url",
		decoder.Desc("Remote http/https resources"),
		decoder.StringMatcher(func(s string) bool {
			s = strings.ToLower(s)
			return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
		}),
		decoder.Init(open, shut),
		decoder.StringDecoder(decodeURL),
	)
}

// A blitz context owns worker threads and is expensive to build, so the two
// decoders share one rather than starting an engine each.
var (
	openOnce  sync.Once
	shutOnce  sync.Once
	shared    *blitz.Context
	errShared error
)

// open returns the shared blitz context, building it on first use.
func open(context.Context) (any, error) {
	openOnce.Do(func() {
		shared, errShared = blitz.New(0)
	})
	return shared, errShared
}

// shut closes the shared blitz context. Both decoders register it, so it
// closes on the first call and does nothing on the second.
func shut(context.Context) error {
	var err error
	shutOnce.Do(func() {
		if shared != nil {
			err = shared.Close()
		}
	})
	return err
}

// engine returns the shared blitz context from the decoder state.
func engine(ctx context.Context) (*blitz.Context, error) {
	c, _ := decoder.State(ctx).(*blitz.Context)
	if c == nil {
		return nil, errors.New("blitz not initialized")
	}
	return c, nil
}

// decode renders a markdown or html document.
func decode(ctx context.Context, r io.Reader) (any, error) {
	c, err := engine(ctx)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return render(ctx, c, string(buf), base(ctx), isHTML(ivctx.Mime(ctx), ivctx.PathName(ctx), buf))
}

// render renders the whole source as html or as markdown, and cuts the image
// into pages of the display height.
//
// A document that is taller than blitz draws at scale 2 is drawn again at
// scale 1, which holds a document twice as tall in the same number of rows.
// Only a document that is also taller than that is cut off.
func render(ctx context.Context, c *blitz.Context, src, baseURL string, html bool) (any, error) {
	kind := "markdown"
	if html {
		kind = "html"
	}
	// one highlighter serves both renders, so that a block is highlighted once
	var h blitz.Highlighter
	if !html {
		h = highlighter(ctx)
	}
	img, err := draw(ctx, c, src, baseURL, kind, h, scale)
	if err != nil {
		return nil, err
	}
	if truncated(img, scale) {
		ivctx.Logf(ctx, "blitz %s: document is taller than %d css pixels, drawing it again at scale 1", kind, maxRows/scale)
		if img, err = draw(ctx, c, src, baseURL, kind, h, 1); err != nil {
			return nil, err
		}
		if truncated(img, 1) {
			ivctx.Warnf(ctx, "blitz %s: document is taller than %d css pixels, truncated", kind, maxRows)
		}
	}
	return pages(ctx, img), nil
}

// draw renders the source at a scale, up to the height that blitz draws. The
// source is Markdown when h is not nil, and h colors its code blocks.
func draw(ctx context.Context, c *blitz.Context, src, baseURL, kind string, h blitz.Highlighter, s uint32) (*image.RGBA, error) {
	opts := options(ctx)
	opts.Scale, opts.MaxHeight = float32(s), maxRows/s
	ivctx.Logf(ctx, "blitz %s: %d bytes at %dx%d base %q", kind, len(src), opts.Width, s, baseURL)
	render := func(src, baseURL string, opts blitz.Options) (*image.RGBA, error) {
		return c.RenderMarkdownHighlighted(src, baseURL, opts, h)
	}
	if h == nil {
		render = c.RenderHTML
	}
	img, err := render(src, baseURL, opts)
	if err != nil {
		return nil, fmt.Errorf("blitz %s: %w", kind, err)
	}
	ivctx.Logf(ctx, "blitz %s: %v", kind, img.Bounds().Size())
	return img, nil
}

// truncated reports whether a render at a scale reached the height that blitz
// draws, so that it holds the top of the document and not all of it.
func truncated(img *image.RGBA, s uint32) bool {
	return img.Bounds().Dy() >= int(maxRows/s*s)
}

// pages cuts the image into pages of the height that [ivctx.PageHeight]
// returns, and the pipeline shows the page that --page selects. An image that
// fits the display is one page and is not cut. Each page shares the pixels
// of the image.
func pages(ctx context.Context, img *image.RGBA) any {
	b := img.Bounds()
	h := ivctx.PageHeight(ctx, b.Dx())
	if h == 0 || b.Dy() <= h {
		return img
	}
	var v []image.Image
	for y := b.Min.Y; y < b.Max.Y; y += h {
		v = append(v, rows(img, y, min(y+h, b.Max.Y)))
	}
	ivctx.Logf(ctx, "blitz: %d pages of %d pixels", len(v), h)
	return v
}

// rows returns the rows y0 to y1 of the image as an image of its own, at the
// origin, that shares the pixels of the image.
func rows(img *image.RGBA, y0, y1 int) *image.RGBA {
	b := img.Bounds()
	start := img.PixOffset(b.Min.X, y0)
	end := img.PixOffset(b.Min.X, y1-1) + 4*b.Dx()
	return &image.RGBA{
		Pix:    img.Pix[start:end:end],
		Stride: img.Stride,
		Rect:   image.Rect(0, 0, b.Dx(), y1-y0),
	}
}

// options returns the render options, honoring the configured background when
// one was set -- a rendered document is opaque, so iv's own compositing would
// otherwise never show through it.
func options(ctx context.Context) blitz.Options {
	c := ivctx.Get(ctx)
	opts := blitz.DefaultOptions()
	opts.Width, opts.UserAgent = width, UserAgent
	if c.BlitzDark {
		// a document is laid out light by default, so this is what makes a
		// page's own dark styling apply
		opts.ColorScheme = blitz.Dark
		opts.BackgroundRGBA = darkBg
	}
	if bg := c.Bg; bg != nil && bg.A != 0 {
		n := bg.NRGBA()
		opts.BackgroundRGBA = uint32(n.R)<<24 | uint32(n.G)<<16 | uint32(n.B)<<8 | uint32(n.A)
	}
	return opts
}

// darkBg is the backdrop painted under a document rendered dark, for the
// documents that style no background of their own.
const darkBg = 0x16161aff

// base returns the base url that relative links in a local document resolve
// against: the directory holding it.
func base(ctx context.Context) string {
	pathName := ivctx.PathName(ctx)
	if pathName == "" {
		return ""
	}
	abs, err := filepath.Abs(pathName)
	if err != nil {
		return ""
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Dir(abs)) + "/"}).String()
}

// The extensions that decide how a document is parsed.
var (
	htmlExts     = []string{"html", "htm", "xhtml"}
	markdownExts = []string{"md", "markdown", "mkd", "mdown"}
)

// isHTML reports whether the source must be parsed as html.
//
// The extension decides first. A Markdown file that starts with html, such as
// a README that starts with a centered logo in a <div>, is typed text/html by
// libmagic, and is still Markdown. The mime type decides next, and only a
// document with neither is judged by what it starts with.
func isHTML(mime, pathName string, buf []byte) bool {
	switch ext := ivctx.FileExt(pathName); {
	case slices.Contains(htmlExts, ext):
		return true
	case slices.Contains(markdownExts, ext):
		return false
	}
	switch {
	case strings.Contains(mime, "html"):
		return true
	case strings.Contains(mime, "markdown"):
		return false
	}
	s := strings.ToLower(strings.TrimSpace(string(buf[:min(len(buf), 256)])))
	return strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html")
}
