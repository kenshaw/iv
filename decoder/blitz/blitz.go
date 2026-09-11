// Package blitz supplies a html and markdown decoder for iv, rendering
// documents and remote pages with the blitz browser engine.
//
// See: https://github.com/xo/blitz
package blitz

import (
	"context"
	"fmt"
	"image"
	"io"
	"net/url"
	"path/filepath"
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
)

// UserAgent is the user agent sent with requests.
var UserAgent = "iv"

func init() {
	decoder.Register(
		"blitz",
		decoder.Desc("Markdown and html documents"),
		// text/plain is shared with mermaid, csv, and dot, so this is the last
		// plain text decoder tried
		decoder.After("mermaid", "graphviz", "libreoffice"),
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
	sharedErr error
)

// open returns the shared blitz context, building it on first use.
func open(context.Context) (any, error) {
	openOnce.Do(func() {
		shared, sharedErr = blitz.New(0)
	})
	return shared, sharedErr
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
		return nil, fmt.Errorf("blitz not initialized")
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

// render renders the source as html or as markdown.
func render(ctx context.Context, c *blitz.Context, src, baseURL string, html bool) (image.Image, error) {
	opts := options(ctx)
	kind := "markdown"
	if html {
		kind = "html"
	}
	ivctx.Logf(ctx, "blitz %s: %d bytes at %dx%g base %q", kind, len(src), opts.Width, opts.Scale, baseURL)
	render := c.RenderMarkdown
	if html {
		render = c.RenderHTML
	}
	img, err := render(src, baseURL, opts)
	if err != nil {
		return nil, fmt.Errorf("blitz %s: %w", kind, err)
	}
	ivctx.Logf(ctx, "blitz %s: %v", kind, img.Bounds().Size())
	return img, nil
}

// options returns the render options, honoring the configured background when
// one was set -- a rendered document is opaque, so iv's own compositing would
// otherwise never show through it.
func options(ctx context.Context) blitz.Options {
	opts := blitz.DefaultOptions()
	opts.Width, opts.Scale, opts.UserAgent = width, scale, UserAgent
	if bg := ivctx.Get(ctx).Bg; bg != nil && bg.A != 0 {
		c := bg.NRGBA()
		opts.BackgroundRGBA = uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
	}
	return opts
}

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

// htmlExts are the extensions parsed as html rather than markdown.
var htmlExts = map[string]bool{"html": true, "htm": true, "xhtml": true}

// isHTML reports whether the source should be parsed as html. A .md file is
// markdown whatever it contains, so the mime type and extension decide first;
// only a document with neither is judged by what it starts with.
func isHTML(mime, pathName string, buf []byte) bool {
	switch {
	case strings.Contains(mime, "html"):
		return true
	case strings.Contains(mime, "markdown"):
		return false
	}
	if ext := ivctx.FileExt(pathName); ext != "" {
		return htmlExts[ext]
	}
	s := strings.ToLower(strings.TrimSpace(string(buf[:min(len(buf), 256)])))
	return strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html")
}
