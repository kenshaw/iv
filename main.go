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
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/kenshaw/colors"
	"github.com/kenshaw/rasterm"
	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/xo/ox"
	_ "github.com/xo/ox/color"
	"github.com/xo/resvg"
	"github.com/yookoala/realpath"
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
	args := &Args{
		logger: func(string, ...interface{}) {},
	}
	// flags
	ox.RunContext(
		context.Background(),
		ox.Usage("iv", "the command-line terminal graphics image viewer"),
		ox.Defaults(),
		ox.Exec(run(os.Stdout, args)),
		ox.From(args),
	)
}

type Args struct {
	Verbose         bool          `ox:"enable verbose,short:v"`
	Bg              *colors.Color `ox:"background color,default:transparent"`
	VipsConcurrency int           `ox:"vips concurrency,default:$NUMCPU"`
	Width           int           `ox:"set width"`
	Height          int           `ox:"set height"`
	DPI             int           `ox:"dpi,default:300,name:dpi"`

	logger func(string, ...interface{})
	bgc    color.Color
	once   sync.Once
}

// run renders the specified files to w.

func run(w io.Writer, args *Args) func(context.Context, []string) error {
	return func(ctx context.Context, cliargs []string) error {
		if !rasterm.Available() {
			return rasterm.ErrTermGraphicsNotAvailable
		}
		// set verbose logger
		if args.Verbose {
			args.logger = func(s string, v ...interface{}) {
				fmt.Fprintf(os.Stderr, s+"\n", v...)
			}
		}
		// add background color for svgs
		resvg.WithBackground(args.Bg)(resvg.Default)
		if args.Width != 0 || args.Height != 0 {
			resvg.WithScaleMode(resvg.ScaleBestFit)(resvg.Default)
			resvg.WithWidth(int(args.Width))(resvg.Default)
			resvg.WithHeight(int(args.Height))(resvg.Default)
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
			if s := entry.Name(); !entry.IsDir() && extRE.MatchString(s) {
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
	typ, img, err := args.renderFile(name)
	if err != nil {
		return err
	}
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
func (args *Args) renderFile(name string) (string, image.Image, error) {
	f, g := args.openImage, args.openVips
	if m := extRE.FindStringSubmatch(name); m != nil {
		switch strings.ToLower(m[1]) {
		case "avif", "heic", "heiv", "heif", "pdf", "jp2", "jxl":
			f, g = g, f
		case "markdown", "md":
			f, g = args.openMarkdown, nil
		}
	}
	typ, img, err := f(name)
	switch {
	case err == nil:
		return typ, img, nil
	case g == nil:
		return "", nil, err
	}
	args.logger("initial open failed: %v", err)
	return g(name)
}

// openImage opens the image using Go's standard image.Decode.
func (args *Args) openImage(name string) (string, image.Image, error) {
	start := time.Now()
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return "", nil, fmt.Errorf("can't open %s: %w", name, err)
	}
	img, typ, err := image.Decode(f)
	if err != nil {
		defer f.Close()
		return "", nil, fmt.Errorf("can't decode %s: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return "", nil, fmt.Errorf("can't close %s: %w", name, err)
	}
	args.logger("image open/decode: %v", time.Now().Sub(start))
	return typ, img, nil
}

// openVips opens the image using libvips.
func (args *Args) openVips(name string) (string, image.Image, error) {
	v, err := args.vipsOpenFile(name)
	if err != nil {
		return "", nil, err
	}
	return args.vipsExport(v)
}

// openMarkdown opens the markdown file, rendering it as a pdf then using
// libvips to export it as a standard image.
func (args *Args) openMarkdown(name string) (string, image.Image, error) {
	// read file
	start := time.Now()
	pathstr, err := realpath.Realpath(name)
	if err != nil {
		return "", nil, err
	}
	src, err := os.ReadFile(pathstr)
	if err != nil {
		return "", nil, err
	}
	args.logger("markdown open: %v", time.Now().Sub(start))
	start = time.Now()
	md := goldmark.New(
		goldmark.WithRenderer(
			pdf.New(
				// pdf.WithContext(context.Background()),
				pdf.WithTraceWriter(args),
				pdf.WithImageFS(http.FS(os.DirFS(filepath.Dir(pathstr)))),
				pdf.WithHeadingFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithBodyFont(pdf.GetTextFont("Arial", pdf.FontHelvetica)),
				pdf.WithCodeFont(pdf.GetCodeFont("Arial", pdf.FontHelvetica)),
			),
		),
	)
	buf := new(bytes.Buffer)
	if err := md.Convert(src, buf); err != nil {
		return "", nil, fmt.Errorf("unable to convert markdown to pdf: %w", err)
	}
	args.logger("markdown convert: %v", time.Now().Sub(start))
	start = time.Now()
	v, err := args.vipsOpenReader(buf, name)
	if err != nil {
		return "", nil, fmt.Errorf("vips can't load rendered pdf for %s: %w", name, err)
	}
	args.logger("markdown render: %v", time.Now().Sub(start))
	return args.vipsExport(v)
}

// vipsInit initializes the vip package.
func (args *Args) vipsInit() {
	start := time.Now()
	level := vips.LogLevelError
	if args.Verbose {
		level = vips.LogLevelDebug
	}
	vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
		args.logger("vips %s: %s %s", vipsLevel(level), domain, strings.TrimSpace(msg))
	}, level)
	var config *vips.Config
	if args.VipsConcurrency != 0 {
		config = &vips.Config{
			ConcurrencyLevel: args.VipsConcurrency,
		}
	}
	vips.Startup(config)
	args.logger("vips init: %v", time.Now().Sub(start))
}

// vipsOpenReader opens a vips image from the reader.
func (args *Args) vipsOpenReader(r io.Reader, name string) (*vips.ImageRef, error) {
	args.once.Do(args.vipsInit)
	start := time.Now()
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	args.logger("load file: %v", time.Now().Sub(start))
	start = time.Now()
	p := vips.NewImportParams()
	if args.DPI != 0 {
		p.Density.Set(int(args.DPI))
	}
	v, err := vips.LoadImageFromBuffer(buf, p)
	if err != nil {
		return nil, fmt.Errorf("vips can't load %s: %w", name, err)
	}
	args.logger("vips load: %v", time.Now().Sub(start))
	return v, nil
}

// vipsOpenFile opens the file.
func (args *Args) vipsOpenFile(name string) (*vips.ImageRef, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return args.vipsOpenReader(f, name)
}

// vipsExport exports the vips image as a png image.
func (args *Args) vipsExport(v *vips.ImageRef) (string, image.Image, error) {
	start := time.Now()
	ext, w, h := strings.TrimPrefix(v.OriginalFormat().FileExt(), "."), v.Width(), v.Height()
	args.logger("vips format: %s dimensions: %dx%d pages: %d", ext, w, h, v.Pages())
	if ext == "pdf" {
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), 2000, 2000)
		if scale != 1.0 {
			if err := v.Resize(float64(scale), vips.KernelAuto); err != nil {
				return "", nil, fmt.Errorf("vips unable to scale pdf: %w", err)
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
		return "", nil, fmt.Errorf("vips can't export %s: %w", name, err)
	}
	args.logger("vips export: %v", time.Now().Sub(start))
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return "", nil, fmt.Errorf("can't decode vips image %s: %w", name, err)
	}
	args.logger("image type: %T", img)
	return "png", img, nil
}

// addBackground adds a background to a image.
func (args *Args) addBackground(fg image.Image, typ string) image.Image {
	if args.bgc == nil || typ == "svg" {
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

// Write satisfies the writer interface.
func (args *Args) Write(buf []byte) (int, error) {
	args.logger("md: %s", string(bytes.TrimRightFunc(buf, unicode.IsSpace)))
	return len(buf), nil
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

// extRE matches file extensions.
var extRE = regexp.MustCompile(`(?i)\.(jpe?g|jp2|gif|jxl|png|pdf|svg|bmp|bitmap|tiff?|avif|hei[fvc]|webp|markdown|md)$`)
