// Package markdown supplies a markdown decoder for iv, rendering markdown to
// pdf and handing it back to the pipeline.
//
// See: https://github.com/stephenafamo/goldmark-pdf
package markdown

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"
)

func init() {
	decoder.Register(
		"markdown",
		decoder.Desc("Markdown documents"),
		// text/plain is shared with mermaid, csv, and dot, so markdown is the
		// last plain text decoder tried
		decoder.After("mermaid", "graphviz", "libreoffice"),
		decoder.Extension("md", "markdown", "mkd", "mdown", "txt"),
		decoder.MimeType("text/markdown", "text/x-markdown"),
		decoder.Matcher(func(mime, _ string) bool {
			return mime == "text/plain"
		}),
		decoder.Decoder(decode),
	)
}

// decode renders the markdown as a pdf, handing it back to the pipeline.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	md := goldmark.New(
		goldmark.WithRenderer(
			pdf.New(
				pdf.WithContext(ctx),
				pdf.WithImageFS(images{ctx: ctx}),
				pdf.WithTraceWriter(logWriter{ctx: ctx}),
				pdf.WithHeadingFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithBodyFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithCodeFont(pdf.GetCodeFont("Arial", pdf.FontHelvetica)),
			),
		),
	)
	buf := new(bytes.Buffer)
	if err := md.Convert(src, buf); err != nil {
		return nil, fmt.Errorf("md convert: %w", err)
	}
	ivctx.Logf(ctx, "md convert: %v", time.Since(start))
	return decoder.NewBytes("application/pdf", buf.Bytes()).WithExt("pdf"), nil
}

// urlRE matches http/s URLs.
var urlRE = regexp.MustCompile(`(?i)^https?://`)

// images resolves the remote images referenced by a markdown document,
// rendering each through the decoding pipeline and re-encoding it as a png
// that goldmark-pdf can embed.
type images struct {
	ctx context.Context
}

// Open satisfies the [http.FileSystem] interface.
func (fsys images) Open(urlstr string) (http.File, error) {
	ivctx.Logf(fsys.ctx, "md open: %s", urlstr)
	if !urlRE.MatchString(urlstr) {
		return nil, fs.ErrNotExist
	}
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	name := path.Base(u.Path)
	req, err := http.NewRequestWithContext(fsys.ctx, http.MethodGet, urlstr, nil)
	if err != nil {
		return nil, fmt.Errorf("md open: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("md open: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("md open: %s: %s", urlstr, res.Status)
	}
	ctx := ivctx.WithPathName(fsys.ctx, name)
	mime, _, _ := strings.Cut(res.Header.Get("Content-Type"), ";")
	img, _, err := decoder.Decode(ctx, strings.TrimSpace(mime), ivctx.FileExt(name), res.Body)
	if err != nil {
		return nil, fmt.Errorf("md open: render: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("md open: encode: %w", err)
	}
	return &file{
		name: name,
		typ:  "image/png",
		r:    bytes.NewReader(buf.Bytes()),
		n:    buf.Len(),
	}, nil
}

// logWriter routes goldmark-pdf's trace output to the verbose logger.
type logWriter struct {
	ctx context.Context
}

// Write satisfies the [io.Writer] interface.
func (w logWriter) Write(buf []byte) (int, error) {
	ivctx.Logf(w.ctx, "md: %s", strings.TrimRight(string(buf), "\r\n\t "))
	return len(buf), nil
}

// file is an in-memory [http.File].
type file struct {
	name string
	typ  string
	r    *bytes.Reader
	n    int
}

func (f *file) MimeType() string                   { return f.typ }
func (f *file) Read(b []byte) (int, error)         { return f.r.Read(b) }
func (f *file) Seek(o int64, w int) (int64, error) { return f.r.Seek(o, w) }
func (f *file) Stat() (fs.FileInfo, error)         { return f, nil }
func (f *file) Readdir(int) ([]fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (*file) Close() error                         { return nil }
func (f *file) Name() string                       { return f.name }
func (f *file) Size() int64                        { return int64(f.n) }
func (f *file) Mode() fs.FileMode                  { return 0o644 }
func (f *file) ModTime() time.Time                 { return time.Time{} }
func (f *file) IsDir() bool                        { return false }
func (f *file) Sys() any                           { return nil }
