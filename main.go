// Command iv is a command-line image viewer using terminal graphics (Sixel,
// iTerm, Kitty).
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/gabriel-vasile/mimetype"
	"github.com/kenshaw/colors"
	"github.com/kenshaw/fontimg"
	"github.com/kenshaw/rasterm"
	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/tdewolff/canvas"
	"github.com/xo/ox"
	_ "github.com/xo/ox/color"
	"github.com/xo/resvg"
	"github.com/yuin/goldmark"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var (
	name    = "iv"
	version = "0.0.0-dev"
)

func main() {
	ox.DefaultVersionString = version
	args := &Args{
		logger: func(string, ...any) {},
	}
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "the command-line terminal graphics image viewer"),
		ox.Defaults(),
		ox.Exec(run(os.Stdout, args)),
		ox.From(args),
	)
}

type Args struct {
	Verbose         bool               `ox:"enable verbose,short:v"`
	Width           int                `ox:"display width,short:W"`
	Height          int                `ox:"display height,short:H"`
	DPI             int                `ox:"image dpi,default:300,name:dpi"`
	Page            int                `ox:"page to display,short:p"`
	Bg              *colors.Color      `ox:"image background color,default:transparent"`
	FontSize        int                `ox:"font size,default:48"`
	FontStyle       canvas.FontStyle   `ox:"font style"`
	FontVariant     canvas.FontVariant `ox:"font variant"`
	FontFg          *colors.Color      `ox:"font foreground color,default:black"`
	FontBg          *colors.Color      `ox:"font background color,default:white"`
	FontDPI         int                `ox:"font fpi,default:100,name:font-dpi"`
	FontMargin      int                `ox:"margin,default:5"`
	VipsConcurrency int                `ox:"vips concurrency,default:$NUMCPU"`

	ctx    context.Context
	logger func(string, ...any)
	bgc    color.Color
	once   sync.Once
}

// run renders the specified files to w.
func run(w io.Writer, args *Args) func(context.Context, []string) error {
	return func(ctx context.Context, cliargs []string) error {
		args.ctx = ctx
		if !rasterm.Available() {
			return rasterm.ErrTermGraphicsNotAvailable
		}
		// set verbose logger
		if args.Verbose {
			args.logger = func(s string, v ...any) {
				fmt.Fprintf(os.Stderr, s+"\n", v...)
			}
		}
		// set svg background color and scaling
		resvg.WithBackground(args.Bg)(resvg.Default)
		if args.Width != 0 || args.Height != 0 {
			resvg.WithScaleMode(resvg.ScaleBestFit)(resvg.Default)
			resvg.WithWidth(args.Width)(resvg.Default)
			resvg.WithHeight(args.Height)(resvg.Default)
		}
		// convert the bg color
		if !colors.Is(args.Bg, colors.Transparent) {
			args.bgc = color.NRGBAModel.Convert(args.Bg).(color.NRGBA)
		}
		// collect files
		var files []string
		for i := 0; i < len(cliargs); i++ {
			v, err := open(cliargs[i])
			if err != nil {
				fmt.Fprintf(w, "error: unable to open arg %d: %v\n", i, err)
			}
			files = append(files, v...)
		}
		// render
		for i := 0; i < len(files); i++ {
			if err := args.render(w, files[i]); err != nil {
				fmt.Fprintf(w, "error: unable to render arg %d: %v\n", i, err)
			}
		}
		return nil
	}
}

// open returns the files to open.
func open(name string) ([]string, error) {
	var v []string
	switch fi, err := os.Stat(name); {
	case err == nil && fi.IsDir():
		entries, err := os.ReadDir(name)
		if err != nil {
			return nil, fmt.Errorf("unable to open directory %q: %v", name, err)
		}
		for _, entry := range entries {
			if s := entry.Name(); !entry.IsDir() {
				v = append(v, filepath.Join(name, s))
			}
		}
		sort.Strings(v)
	case err == nil:
		v = append(v, name)
	default:
		return nil, fmt.Errorf("unable to open %q", name)
	}
	return v, nil
}

// render renders the file to w.
func (args *Args) render(w io.Writer, name string) error {
	fmt.Fprintln(w, name+":")
	start := time.Now()
	// render
	img, typ, err := args.renderFile(name)
	if err != nil {
		return err
	}
	// add background
	img = args.addBackground(img, typ)
	now := time.Now()
	if err = rasterm.Encode(w, img); err != nil {
		return err
	}
	args.logger("out: %v", time.Now().Sub(now))
	args.logger("total: %v", time.Now().Sub(start))
	return nil
}

// renderFile renders the file.
func (args *Args) renderFile(name string) (image.Image, string, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, "", err
	}
	// determine type
	typ, err := mimeDetect(f)
	if err != nil {
		defer f.Close()
		return nil, "", fmt.Errorf("mime detection failed: %v", err)
	}
	args.logger("mime: %s", typ)
	var g func(io.Reader, string) (image.Image, error)
	var notStream bool
	switch {
	case typ == "image/svg":
		g = args.renderResvg
	case typ == "text/plain":
		g = args.renderMarkdown
	case isImageBuiltin(typ):
		g = args.renderImage
	case strings.HasPrefix(typ, "font/"):
		g = args.renderFont
	case strings.HasPrefix(typ, "image/"): // use vips
		g = args.renderVips
	case strings.HasPrefix(typ, "video/"):
		g, notStream = args.renderAstiav, true
	}
	if notStream {
		if err := f.Close(); err != nil {
			return nil, "", fmt.Errorf("could not close file: %w", err)
		}
	} else {
		// reset reader
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			defer f.Close()
			return nil, "", fmt.Errorf("could not seek start: %w", err)
		}
	}
	img, err := g(f, name)
	if err != nil {
		if !notStream {
			defer f.Close()
		}
		return nil, "", err
	}
	if !notStream {
		if err := f.Close(); err != nil {
			return nil, "", fmt.Errorf("could not close file: %w", err)
		}
	}
	return img, typ, nil
}

// renderImage decodes the image from the reader.
func (*Args) renderImage(r io.Reader, _ string) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// renderResvg decodes the svg from the reader.
func (args *Args) renderResvg(r io.Reader, _ string) (image.Image, error) {
	return resvg.Decode(r)
}

// renderFont decodes the font from the reader into an image.
func (args *Args) renderFont(r io.Reader, name string) (image.Image, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	font := fontimg.New(buf, name)
	img, err := font.Rasterize(
		nil,
		args.FontSize,
		args.FontStyle,
		args.FontVariant,
		args.FontFg,
		args.FontBg,
		float64(args.FontDPI),
		float64(args.FontMargin),
	)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// renderVips opens a vips image from the reader.
func (args *Args) renderVips(r io.Reader, name string) (image.Image, error) {
	args.once.Do(vipsInit(args.logger, args.Verbose, args.VipsConcurrency))
	start := time.Now()
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	args.logger("load file: %v", time.Now().Sub(start))
	start = time.Now()
	p := vips.NewImportParams()
	if args.DPI != 0 {
		p.Density.Set(args.DPI)
	}
	if args.Page != 0 {
		v, err := vips.LoadImageFromBuffer(buf, vips.NewImportParams())
		if err != nil {
			return nil, fmt.Errorf("vips can't load %s: %w", name, err)
		}
		if page := args.Page - 1; 0 <= page && page < v.Metadata().Pages {
			p.Page.Set(page)
		}
	}
	v, err := vips.LoadImageFromBuffer(buf, p)
	if err != nil {
		return nil, fmt.Errorf("vips can't load %s: %w", name, err)
	}
	args.logger("vips load: %v", time.Now().Sub(start))
	return args.vipsExport(v)
}

func (args *Args) renderAstiav(r io.Reader, name string) (image.Image, error) {
	return nil, fmt.Errorf("not supported! %q", name)
}

// vipsExport exports the vips image as a png image.
func (args *Args) vipsExport(v *vips.ImageRef) (image.Image, error) {
	start := time.Now()
	ext, w, h := strings.TrimPrefix(v.OriginalFormat().FileExt(), "."), v.Width(), v.Height()
	args.logger("vips format: %s dimensions: %dx%d pages: %d", ext, w, h, v.Pages())
	if ext == "pdf" {
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), 2000, 2000)
		if scale != 1.0 {
			if err := v.Resize(float64(scale), vips.KernelAuto); err != nil {
				return nil, fmt.Errorf("vips unable to scale pdf: %w", err)
			}
			args.logger("vips resize: %v", time.Now().Sub(start))
		}
	}
	start = time.Now()
	buf, _, err := v.ExportPng(&vips.PngExportParams{
		Filter:    vips.PngFilterNone,
		Interlace: false,
		Palette:   true,
		// Bitdepth:  4,
	})
	if err != nil {
		return nil, fmt.Errorf("vips can't export %s: %w", name, err)
	}
	args.logger("vips export: %v", time.Now().Sub(start))
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("can't decode vips image %s: %w", name, err)
	}
	args.logger("image type: %T", img)
	return img, nil
}

/*
// vipsOpen opens the file.
func (args *Args) vipsOpen(name string) (image.Image, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return args.renderVips(f, name)
}
*/

// addBackground adds a background to a image.
func (args *Args) addBackground(fg image.Image, typ string) image.Image {
	if args.bgc == nil || typ == "image/svg" {
		return fg
	}
	start := time.Now()
	b, c := fg.Bounds(), args.bgc.(color.NRGBA)
	img := image.NewNRGBA(b)
	for i := 0; i < b.Dx(); i++ {
		for j := 0; j < b.Dy(); j++ {
			img.SetNRGBA(i, j, c)
		}
	}
	draw.Draw(img, b, fg, image.Point{}, draw.Over)
	args.logger("add bg: %v", time.Now().Sub(start))
	return img
}

// renderMarkdown renders the markdown file, rendering it as a pdf then using
// libvips to export it as a standard image.
func (args *Args) renderMarkdown(r io.Reader, name string) (image.Image, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// read file
	start := time.Now()
	md := goldmark.New(
		goldmark.WithRenderer(
			pdf.New(
				pdf.WithContext(args.ctx),
				pdf.WithImageFS(files{args}),
				pdf.WithTraceWriter(args),
				pdf.WithHeadingFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithBodyFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithCodeFont(pdf.GetCodeFont("Arial", pdf.FontHelvetica)),
			),
		),
	)
	buf := new(bytes.Buffer)
	if err := md.Convert(src, buf); err != nil {
		return nil, fmt.Errorf("unable to convert markdown to pdf: %w", err)
	}
	args.logger("markdown convert: %v", time.Now().Sub(start))
	start = time.Now()
	pdf, err := args.renderVips(buf, name)
	if err != nil {
		return nil, fmt.Errorf("vips can't load rendered pdf for %s: %w", name, err)
	}
	args.logger("markdown render: %v", time.Now().Sub(start))
	return pdf, nil
}

// Write satisfies the writer interface.
func (args *Args) Write(buf []byte) (int, error) {
	args.logger("md: %s", string(bytes.TrimRightFunc(buf, unicode.IsSpace)))
	return len(buf), nil
}

// vipsInit initializes the vip package.
func vipsInit(logger func(string, ...any), verbose bool, concurrency int) func() {
	return func() {
		start := time.Now()
		level := vips.LogLevelError
		if verbose {
			level = vips.LogLevelDebug
		}
		vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
			logger("vips %s: %s %s", vipsLevel(level), domain, strings.TrimSpace(msg))
		}, level)
		var config *vips.Config
		if concurrency != 0 {
			config = &vips.Config{
				ConcurrencyLevel: concurrency,
			}
		}
		vips.Startup(config)
		logger("vips init: %v", time.Now().Sub(start))
	}
}

// vipsLevel returns the vips level as a string.
func vipsLevel(level vips.LogLevel) string {
	switch level {
	case vips.LogLevelError:
		return "error"
	case vips.LogLevelCritical:
		return "critical"
	case vips.LogLevelWarning:
		return "warning"
	case vips.LogLevelMessage:
		return "message"
	case vips.LogLevelInfo:
		return "info"
	case vips.LogLevelDebug:
		return "debug"
	}
	return fmt.Sprintf("(%d)", level)
}

type files struct {
	args *Args
}

func (fs files) Open(urlstr string) (http.File, error) {
	fs.args.logger("md open: %s", urlstr)
	if !urlRE.MatchString(urlstr) {
		return nil, os.ErrNotExist
	}
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	name := path.Base(u.Path)
	req, err := http.NewRequestWithContext(fs.args.ctx, "GET", urlstr, nil)
	if err != nil {
		return nil, fmt.Errorf("md open: %w", err)
	}
	cl := &http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("md open: do: %w", err)
	}
	defer res.Body.Close()
	img, err := fs.args.renderVips(res.Body, name)
	if err != nil {
		return nil, fmt.Errorf("md open: render: %w", err)
	}
	fs.args.logger("md open: %s", name)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("md open: encode: %w", err)
	}
	b := buf.Bytes()
	return &file{
		name: name,
		typ:  "image/png",
		r:    bytes.NewReader(b),
		n:    len(b),
	}, nil
}

type file struct {
	name string
	typ  string
	r    *bytes.Reader
	n    int
}

func (f *file) MimeType() string {
	return f.typ
}

func (f *file) Read(b []byte) (int, error) {
	return f.r.Read(b)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return f.r.Seek(offset, whence)
}

func (f *file) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f *file) Readdir(int) ([]fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (*file) Close() error {
	return nil
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return int64(f.n)
}

func (f *file) Mode() fs.FileMode {
	return 0o644
}

func (f *file) ModTime() time.Time {
	return time.Now()
}

func (f *file) IsDir() bool {
	return false
}

func (f *file) Sys() any {
	return nil
}

// mimeDetect determines the mime type for the reader.
func mimeDetect(r io.Reader) (string, error) {
	mime, err := mimetype.DetectReader(r)
	if err != nil {
		return "", err
	}
	typ := mime.String()
	if i := strings.Index(typ, ";"); i != -1 {
		typ = typ[:i]
	}
	return strings.TrimSuffix(typ, "+xml"), nil
}

// isImageBuiltin returns true if the mime type is a supported, builtin Go
// image type.
func isImageBuiltin(typ string) bool {
	switch typ {
	case
		"image/bmp",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/tiff":
		return true
	}
	return false
}

var urlRE = regexp.MustCompile(`(?i)^https?`)
