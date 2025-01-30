// Command iv is a command-line image viewer using terminal graphics (Sixel,
// iTerm, Kitty).
package main

import (
	"bytes"
	"context"
	"errors"
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
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	FontDPI         int                `ox:"font dpi,default:100,name:font-dpi"`
	FontMargin      int                `ox:"margin,default:5"`
	TimeCode        time.Duration      `ox:"video time code,short:t"`
	VipsConcurrency int                `ox:"vips concurrency,default:$NUMCPU"`

	ctx    context.Context
	logger func(string, ...any)
	bgc    color.Color
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
			if s := entry.Name(); !entry.IsDir() && extensions[strings.TrimPrefix(filepath.Ext(s), ".")] {
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
	args.logger("out: %v", time.Since(now))
	args.logger("total: %v", time.Since(start))
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
	switch ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")); {
	case typ == "image/svg":
		g = args.renderResvg
	case isImageBuiltin(typ): // builtin
		g = args.renderImage
	case isLibreOffice(typ, ext): // soffice
		g, notStream = args.renderLibreOffice, true
	case isVipsImage(typ): // use vips
		g = args.renderVips
	case typ == "text/plain":
		g = args.renderMarkdown
	case strings.HasPrefix(typ, "font/"):
		g = args.renderFont
	case strings.HasPrefix(typ, "video/"):
		g, notStream = args.renderFfmpeg, true
	default:
		return nil, "", fmt.Errorf("mime type %q not supported", typ)
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
	vipsOnce.Do(vipsInit(args.logger, args.Verbose, args.VipsConcurrency))
	start := time.Now()
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	args.logger("load file: %v", time.Since(start))
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
	args.logger("vips load: %v", time.Since(start))
	return args.vipsExport(v)
}

// renderFfmpeg renders the image using the ffmpeg command.
func (args *Args) renderFfmpeg(_ io.Reader, pathName string) (image.Image, error) {
	var err error
	ffmpegOnce.Do(func() {
		ffprobePath, _ = exec.LookPath("ffprobe")
		ffmpegPath, err = exec.LookPath("ffmpeg")
	})
	switch {
	case err != nil:
		return nil, err
	case ffmpegPath == "":
		return nil, errors.New("ffmpeg not in path")
	}
	tc := args.ffprobeTimecode(pathName)
	args.logger("snapshot at %v", tc)
	params := []string{
		`-hide_banner`,
		`-ss`, tc,
		`-i`, pathName,
		`-vframes`, `1`,
		`-q:v`, `1`,
		`-f`, `apng`,
		`-`,
	}
	args.logger("executing: %s %s", ffmpegPath, strings.Join(params, " "))
	start := time.Now()
	cmd := exec.CommandContext(
		args.ctx,
		ffmpegPath,
		params...,
	)
	var buf, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &stderr
	if err := cmd.Run(); err != nil {
		errstr := stderr.String()
		if len(errstr) > 100 {
			errstr = errstr[:100]
		}
		return nil, fmt.Errorf("%w: %s", err, errstr)
	}
	args.logger("ffmpeg render: %v", time.Since(start))
	return png.Decode(&buf)
}

func (args *Args) ffprobeTimecode(pathName string) string {
	switch {
	case ffprobePath == "":
		return "00:00"
	case args.TimeCode != 0:
		return formatTimecode(args.TimeCode)
	}
	params := []string{
		`-loglevel`, `quiet`,
		`-show_format`,
		pathName,
	}
	args.logger("ffprobe: executing %s %s", ffprobePath, strings.Join(params, " "))
	cmd := exec.CommandContext(args.ctx, ffprobePath, params...)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "00:00"
	}
	m := durationRE.FindStringSubmatch(string(buf))
	if m == nil {
		return "00:00"
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return "00:00"
	}
	dur := time.Duration(f * float64(time.Second))
	args.logger("ffprobe duration: %v / %s", dur, formatTimecode(dur))
	switch {
	case dur >= 1*time.Hour:
		return "10:00"
	case dur >= 30*time.Minute:
		return "05:00"
	case dur >= 15*time.Minute:
		return "03:00"
	case dur >= 5*time.Minute:
		return "02:00"
	case dur > 1*time.Minute:
		return "00:30"
	case dur > 30*time.Second:
		return "00:10"
	case dur > 5*time.Second:
		return "00:02"
	}
	return "00:00"
}

var durationRE = regexp.MustCompile(`(?m)^duration=(.*)$`)

func formatTimecode(d time.Duration) string {
	if d == 0 {
		return "00:00"
	}
	secs := int64(d / time.Minute)
	rem := int64((d % time.Minute) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs, rem)
}

// renderLibreOffice renders the image using the `soffice` command.
func (args *Args) renderLibreOffice(_ io.Reader, pathName string) (image.Image, error) {
	var err error
	sofficeOnce.Do(func() {
		sofficePath, err = exec.LookPath("soffice")
	})
	switch {
	case err != nil:
		return nil, err
	case sofficePath == "":
		return nil, errors.New("soffice not in path")
	}
	tmpDir, err := os.MkdirTemp("", name+".")
	if err != nil {
		return nil, err
	}
	args.logger("temp dir: %s", tmpDir)
	params := []string{
		`--headless`,
		`--convert-to`, `pdf`,
		`--outdir`, tmpDir,
		pathName,
	}
	args.logger("executing: %s %s", sofficePath, strings.Join(params, " "))
	start := time.Now()
	cmd := exec.CommandContext(
		args.ctx,
		sofficePath,
		params...,
	)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		if len(buf) > 100 {
			buf = buf[:100]
		}
		return nil, fmt.Errorf("%w: %s", err, string(buf))
	}
	args.logger("soffice render: %v", time.Since(start))
	pdf := filepath.Join(
		tmpDir,
		strings.TrimSuffix(filepath.Base(pathName), filepath.Ext(pathName))+".pdf",
	)
	args.logger("rendering soffice output: %q", pdf)
	f, err := os.OpenFile(pdf, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	img, err := args.renderVips(f, pdf)
	if err != nil {
		defer f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	args.logger("removing: %s", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		return nil, err
	}
	return img, nil
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
			args.logger("vips resize: %v", time.Since(start))
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
	args.logger("vips export: %v", time.Since(start))
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("can't decode vips image %s: %w", name, err)
	}
	args.logger("image type: %T", img)
	return img, nil
}

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
	args.logger("add bg: %v", time.Since(start))
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
	args.logger("markdown convert: %v", time.Since(start))
	start = time.Now()
	pdf, err := args.renderVips(buf, name)
	if err != nil {
		return nil, fmt.Errorf("vips can't load rendered pdf for %s: %w", name, err)
	}
	args.logger("markdown render: %v", time.Since(start))
	return pdf, nil
}

// Write satisfies the writer interface.
func (args *Args) Write(buf []byte) (int, error) {
	args.logger("md: %s", string(bytes.TrimRightFunc(buf, unicode.IsSpace)))
	return len(buf), nil
}

// vipsInit initializes the vips package.
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
		logger("vips init: %v", time.Since(start))
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

// isVipsImage returns true if the mime type is supported by libvips.
func isVipsImage(typ string) bool {
	switch typ {
	case "application/pdf":
		return true
	}
	return strings.HasPrefix(typ, "image/")
}

// isLibreOffice returns true if the mime type is supported by the `soffice`
// command.
func isLibreOffice(typ, ext string) bool {
	switch {
	case
		strings.HasPrefix(typ, "application/vnd.openxmlformats-officedocument."), // pptx, xlsx, ...
		strings.HasPrefix(typ, "application/vnd.ms-"),                            // ppt, xls, ...
		strings.HasPrefix(typ, "application/vnd.oasis.opendocument."),            // otp, otp, odg, ...
		typ == "text/rtf",
		typ == "text/csv",
		typ == "text/tab-separated-values",
		typ == "text/plain" && (ext == "csv" || ext == "tsv"):
		return true
	}
	return false
}

var urlRE = regexp.MustCompile(`(?i)^https?://`)

var (
	vipsOnce    sync.Once
	sofficeOnce sync.Once
	ffmpegOnce  sync.Once
)

var (
	sofficePath string
	ffprobePath string
	ffmpegPath  string
)

// extensions are the extensions to check for directories.
var extensions = map[string]bool{
	"3g2":      true,
	"3gp":      true,
	"asf":      true,
	"avif":     true,
	"avi":      true,
	"bmp":      true,
	"bpg":      true,
	"csv":      true,
	"doc":      true,
	"docx":     true,
	"dvb":      true,
	"dwg":      true,
	"eot":      true,
	"flv":      true,
	"gif":      true,
	"heic":     true,
	"heif":     true,
	"ico":      true,
	"jp2":      true,
	"jpeg":     true,
	"jpf":      true,
	"jpg":      true,
	"jxl":      true,
	"jxs":      true,
	"m4v":      true,
	"markdown": true,
	"md":       true,
	"mj2":      true,
	"mkv":      true,
	"mov":      true,
	"mp4":      true,
	"mpeg":     true,
	"mpg":      true,
	"odc":      true,
	"odf":      true,
	"odg":      true,
	"odp":      true,
	"ods":      true,
	"odt":      true,
	"otf":      true,
	"otg":      true,
	"otp":      true,
	"ots":      true,
	"ott":      true,
	"pdf":      true,
	"png":      true,
	"ppt":      true,
	"pptx":     true,
	"pub":      true,
	"rtf":      true,
	"svg":      true,
	"tiff":     true,
	"tsv":      true,
	"ttc":      true,
	"ttf":      true,
	"txt":      true,
	"webm":     true,
	"webp":     true,
	"woff2":    true,
	"woff":     true,
	"xls":      true,
	"xlsx":     true,
	"xpm":      true,
}
