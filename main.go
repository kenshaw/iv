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

	"github.com/cshum/vipsgen/vips"
	"github.com/dhowden/tag"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gen2brain/go-fitz"
	"github.com/kenshaw/colors"
	"github.com/kenshaw/fontimg"
	"github.com/kenshaw/rasterm"
	"github.com/mholt/archives"
	_ "github.com/sergeymakinen/go-ico"
	qrcode "github.com/skip2/go-qrcode"
	_ "github.com/spakin/netpbm"
	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/tc-hib/winres"
	"github.com/tdewolff/canvas"
	"github.com/xo/ox"
	_ "github.com/xo/ox/color"
	"github.com/xo/resvg"
	"github.com/yuin/goldmark"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var (
	name    = "iv"
	version = "0.0.0-dev"
)

func main() {
	args := &Args{
		logger: func(string, ...any) {},
	}
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "the command-line terminal graphics image viewer"),
		ox.VersionString(version),
		ox.Defaults(),
		ox.Exec(run(os.Stdout, args)),
		ox.From(args),
	)
}

type Args struct {
	Verbose         bool               `ox:"enable verbose,short:v"`
	Width           uint               `ox:"display width,short:W"`
	Height          uint               `ox:"display height,short:H"`
	MinWidth        uint               `ox:"minimum width,short:w,default:64"`
	MinHeight       uint               `ox:"minimum height,short:h,default:64"`
	DPI             uint               `ox:"image dpi,default:300,name:dpi"`
	Page            uint               `ox:"page to display,short:p"`
	Fg              *colors.Color      `ox:"foregrond color,default:dimgray"`
	Bg              *colors.Color      `ox:"background color,default:transparent"`
	Border          uint               `ox:"border width,default:30"`
	FontSize        uint               `ox:"font preview size,default:48"`
	FontStyle       canvas.FontStyle   `ox:"font preview style"`
	FontVariant     canvas.FontVariant `ox:"font preview variant"`
	FontFg          *colors.Color      `ox:"font preview foreground color,default:black"`
	FontBg          *colors.Color      `ox:"font preview background color,default:white"`
	FontDPI         uint               `ox:"font preview dpi,default:100,name:font-dpi"`
	FontMargin      uint               `ox:"font preview margin,default:5"`
	TimeCode        time.Duration      `ox:"video time code,short:t"`
	VipsConcurrency uint               `ox:"vips concurrency,default:$NUMCPU"`
	MermaidIcons    []string           `ox:"additional mermaid icon packages"`
	MermaidBg       *colors.Color      `ox:"default mermaid background,default:white"`

	ctx    context.Context
	logger func(string, ...any)

	bgc  *color.NRGBA
	mbgc *color.NRGBA
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
			resvg.WithWidth(max(int(args.Width), int(args.MinWidth)))(resvg.Default)
			resvg.WithHeight(max(int(args.Height), int(args.MinHeight)))(resvg.Default)
		}
		// convert/cache background colors
		if !colors.Is(args.Bg, colors.Transparent) {
			c := args.Bg.NRGBA()
			args.bgc = &c
		}
		if !colors.Is(args.MermaidBg, colors.Transparent) {
			c := args.MermaidBg.NRGBA()
			args.mbgc = &c
		}
		// collect targets
		var targets []target
		for _, pathName := range cliargs {
			if v, err := open(pathName); err == nil {
				targets = append(targets, v...)
			} else {
				fmt.Fprintf(w, "error: unable to open %q: %v\n\n", pathName, err)
			}
		}
		// render
		for _, v := range targets {
			if err := args.render(w, v); err != nil {
				fmt.Fprintf(w, "error: unable to render %q: %v\n\n", v.path, err)
			}
		}
		return nil
	}
}

// open returns the files to open.
func open(pathName string) ([]target, error) {
	switch fi, err := os.Stat(pathName); {
	case err == nil && fi.IsDir():
		entries, err := os.ReadDir(pathName)
		if err != nil {
			return nil, fmt.Errorf("unable to read directory %q: %v", pathName, err)
		}
		var d []target
		for _, entry := range entries {
			if s := entry.Name(); !entry.IsDir() && extensions[fileExt(s)] {
				d = append(d, target{path: filepath.Join(pathName, s)})
			}
		}
		sort.Slice(d, func(i, j int) bool {
			return d[i].path < d[j].path
		})
		return d, nil
	case err == nil:
		return []target{{path: pathName}}, nil
	case strings.Contains(pathName, "://"):
		if _, err := url.Parse(pathName); err == nil {
			return []target{{pathName, true}}, nil
		}
	case strings.HasPrefix(pathName, "WIFI:"):
		return []target{{pathName, true}}, nil
	}
	return nil, fmt.Errorf("unable to open %q", pathName)
}

// targets are either paths that exist on disk, or a url.
type target struct {
	path  string
	isURL bool
}

// render renders the file to w.
func (args *Args) render(w io.Writer, v target) error {
	fmt.Fprintln(w, v.path+":")
	start := time.Now()
	var img image.Image
	var mime string
	var err error
	// render
	if !v.isURL {
		img, mime, err = args.renderFile(v.path)
	} else {
		img, mime, err = args.renderURL(v.path)
	}
	if err != nil {
		return err
	}
	// add background
	img = args.addBackground(mime, img)
	now := time.Now()
	if err = rasterm.Encode(w, img); err != nil {
		return err
	}
	args.logger("encode out: %v", time.Since(now))
	args.logger("total: %v", time.Since(start))
	return nil
}

// renderFile renders the file.
func (args *Args) renderFile(pathName string) (image.Image, string, error) {
	f, err := os.OpenFile(pathName, os.O_RDONLY, 0)
	if err != nil {
		return nil, "", err
	}
	// determine type
	mime, err := mimeDetect(f)
	if err != nil {
		defer f.Close()
		return nil, "", fmt.Errorf("mime detection failed: %v", err)
	}
	args.logger("mime: %s", mime)
	var g func(string, string, io.ReadCloser) (image.Image, error)
	var notStream bool
	switch ext := fileExt(pathName); {
	case mime == "image/svg":
		g = args.decodeResvg
	case isBuiltin(mime): // builtin
		g = args.decodeBuiltin
	case isLibreOffice(mime, ext): // soffice
		g, notStream = args.decodeLibreOffice, true
	case isVips(mime): // use vips
		g = args.decodeVips
	case isFitz(mime, ext):
		g = args.decodeFitz
	case isMermaid(mime, ext):
		g, notStream = args.decodeMermaid, true
	case mime == "text/plain":
		g = args.decodeMarkdown
	case strings.HasPrefix(mime, "font/"):
		g = args.decodeFont
	case strings.HasPrefix(mime, "video/"):
		g, notStream = args.decodeFfmpeg, true
	case strings.HasPrefix(mime, "audio/"):
		g = args.decodeTag
	case isComicArchive(mime, ext):
		g = args.decodeComicArchive
	case isWindowsPE(mime, ext):
		g = args.decodeWindowsPE
	default:
		return nil, "", fmt.Errorf("mime type %q not supported", mime)
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
	img, err := g(pathName, mime, f)
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
	return img, mime, nil
}

// renderURL renders a URL as a QR code.
func (args *Args) renderURL(urlstr string) (image.Image, string, error) {
	q, err := qrcode.New(urlstr, qrcode.Medium)
	if err != nil {
		return nil, "", err
	}
	q.ForegroundColor, q.BackgroundColor, q.DisableBorder = args.Fg, args.Bg, true
	return args.addBorder(q.Image(-10)), "image/bitmap", nil
}

// addBorder adds a border to the image.
func (args *Args) addBorder(src image.Image) image.Image {
	b, w := src.Bounds(), int(args.Border)
	x, y := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, x+(2*w), y+(2*w)))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{args.Bg}, image.Point{}, draw.Src)
	r := image.Rect(w, w, w+b.Dx(), w+b.Dy())
	draw.Draw(dst, r, src, b.Min, draw.Over)
	return dst
}

// decodeBuiltin decodes the image from the reader.
func (args *Args) decodeBuiltin(_, _ string, r io.ReadCloser) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	args.logger("dimensions: %dx%d", b.Dx(), b.Dy())
	return img, err
}

// decodeResvg decodes the svg from the reader.
func (args *Args) decodeResvg(_, _ string, r io.ReadCloser) (image.Image, error) {
	img, err := resvg.Decode(r)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	args.logger("dimensions: %dx%d", b.Dx(), b.Dy())
	return img, nil
}

// decodeFont decodes the font from the reader into an image.
func (args *Args) decodeFont(pathName, _ string, r io.ReadCloser) (image.Image, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	font := fontimg.New(buf, pathName)
	img, err := font.Rasterize(
		nil,
		int(args.FontSize),
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
	b := img.Bounds()
	args.logger("dimensions: %dx%d", b.Dx(), b.Dy())
	return img, nil
}

// decodeVips decodes a vips image from the reader.
func (args *Args) decodeVips(pathName, _ string, r io.ReadCloser) (image.Image, error) {
	vipsOnce.Do(vipsInit(args.logger, args.Verbose, int(args.VipsConcurrency)))
	start := time.Now()
	args.logger("load file: %v", time.Since(start))
	start = time.Now()
	opts := &vips.LoadOptions{
		N: 1,
		// Autorotate:  true,
		FailOnError: true,
	}
	if args.Page != 0 {
		v, err := vips.NewImageFromSource(vips.NewSource(r), opts)
		if err != nil {
			return nil, fmt.Errorf("vips can't load %s: %w", pathName, err)
		}
		if page := int(args.Page - 1); 0 <= page && page < v.Pages() {
			opts.Page = page
		}
	}
	v, err := vips.NewImageFromSource(vips.NewSource(r), opts)
	if err != nil {
		return nil, fmt.Errorf("vips can't load %s: %w", pathName, err)
	}
	args.logger("vips load: %v", time.Since(start))
	return args.vipsExport(v)
}

// decodeFitz renders the image using the fitz (mupdf) package.
func (args *Args) decodeFitz(pathName, _ string, r io.ReadCloser) (image.Image, error) {
	start := time.Now()
	// open
	d, err := fitz.NewFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("fitz can't open %s: %w", pathName, err)
	}
	defer d.Close()
	args.logger("fitz load: %v", time.Since(start))
	args.logger("fitz pages: %d", d.NumPage())
	// page
	page := int(args.Page)
	if page != 0 {
		page--
	}
	// render
	var img *image.RGBA
	start = time.Now()
	if args.DPI != 0 {
		img, err = d.ImageDPI(page, float64(args.DPI))
	} else {
		img, err = d.Image(page)
	}
	if err != nil {
		return nil, fmt.Errorf("fitz can't render %s: %w", pathName, err)
	}
	args.logger("fitz render: %v", time.Since(start))
	return img, nil
}

// decodeFfmpeg decodes the image using the ffmpeg command.
func (args *Args) decodeFfmpeg(pathName, _ string, _ io.ReadCloser) (image.Image, error) {
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

// formatTimecode formats a duration in ffmpeg's timecode format.
func formatTimecode(d time.Duration) string {
	if d == 0 {
		return "00:00"
	}
	secs := int64(d / time.Minute)
	rem := int64((d % time.Minute) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs, rem)
}

// decodeLibreOffice renders the image using the `soffice` command.
func (args *Args) decodeLibreOffice(pathName, _ string, _ io.ReadCloser) (image.Image, error) {
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
	pdfName := filepath.Join(
		tmpDir,
		strings.TrimSuffix(filepath.Base(pathName), filepath.Ext(pathName))+".pdf",
	)
	args.logger("rendering soffice output: %q", pdfName)
	f, err := os.OpenFile(pdfName, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	img, err := args.decodeVips(pdfName, "", f)
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

// decodeMermaid renders the image using the `mmdc` command.
func (args *Args) decodeMermaid(pathName, _ string, _ io.ReadCloser) (image.Image, error) {
	var err error
	mmdcOnce.Do(func() {
		mmdcPath, err = exec.LookPath("mmdc")
	})
	switch {
	case err != nil:
		return nil, err
	case mmdcPath == "":
		return nil, errors.New("mmdc not in path")
	}
	params := []string{
		`--outputFormat`, `svg`,
		`--input`, pathName,
		`--output`, `-`,
		`--iconPacks`, "@iconify-json/logos",
	}
	params = append(params, args.MermaidIcons...)
	args.logger("executing: %s %s", mmdcPath, strings.Join(params, " "))
	start := time.Now()
	cmd := exec.CommandContext(
		args.ctx,
		mmdcPath,
		params...,
	)
	var buf, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	for s := range strings.SplitSeq(strings.TrimSpace(stderr.String()), "\n") {
		args.logger("mmdc: %s", s)
	}
	args.logger("mmdc render: %v", time.Since(start))
	return args.decodeResvg("", "", io.NopCloser(&buf))
}

// vipsExport exports the vips image as a png image.
func (args *Args) vipsExport(v *vips.Image) (image.Image, error) {
	start := time.Now()
	ext, w, h := strings.TrimPrefix(string(v.Format()), "."), v.Width(), v.Height()
	args.logger("vips format: %s dimensions: %dx%d pages: %d", ext, w, h, v.Pages())
	if ext == "pdf" {
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), 2000, 2000)
		if scale != 1.0 {
			if err := v.Resize(float64(scale), nil); err != nil {
				return nil, fmt.Errorf("vips unable to scale pdf: %w", err)
			}
			args.logger("vips resize: %v", time.Since(start))
		}
	}
	start = time.Now()
	buf := new(bytes.Buffer)
	target := vips.NewTarget(nopWriteCloser{buf})
	if err := v.PngsaveTarget(target, nil); err != nil {
		return nil, fmt.Errorf("vips can't export %s: %w", name, err)
	}
	args.logger("vips export: %v", time.Since(start))
	start = time.Now()
	img, _, err := image.Decode(buf)
	if err != nil {
		return nil, fmt.Errorf("can't decode vips image %s: %w", name, err)
	}
	args.logger("go vips decode: %v", time.Since(start))
	args.logger("image type: %T", img)
	return img, nil
}

// decodeTag renders the embedded picture from music metadata (ie, album art).
func (args *Args) decodeTag(_, _ string, r io.ReadCloser) (image.Image, error) {
	f, ok := r.(*os.File)
	if !ok {
		return nil, fmt.Errorf("%T not supported (*os.File only)", r)
	}
	md, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	pic := md.Picture()
	if pic == nil {
		return nil, errors.New("no embedded picture")
	}
	img, _, err := image.Decode(bytes.NewReader(pic.Data))
	return img, err
}

// decodeComicArchive renders the first file in the comic archive with integer
// suffix.
func (args *Args) decodeComicArchive(pathName, mime string, r io.ReadCloser) (image.Image, error) {
	file, ok := r.(*os.File)
	if !ok {
		return nil, fmt.Errorf("%T not supported (*os.File only)", r)
	}
	fsys, err := archives.FileSystem(args.ctx, pathName, file)
	if err != nil {
		return nil, err
	}
	var files []string
	err = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !comicExtensions[fileExt(name)]:
			return nil
		}
		args.logger("file %q", name)
		files = append(files, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	i := 0
	if page := int(args.Page - 1); 0 <= page && page < len(files) {
		i = page
	}
	f, err := fsys.Open(files[i])
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// decodeWindowsPE decodes embedded application icons from the windows PE (exe)
// file.
func (args *Args) decodeWindowsPE(pathName, mime string, r io.ReadCloser) (image.Image, error) {
	file, ok := r.(*os.File)
	if !ok {
		return nil, fmt.Errorf("%T not supported (*os.File only)", r)
	}
	rs, err := winres.LoadFromEXE(file)
	if err != nil {
		return nil, fmt.Errorf("unable to load resources: %w", err)
	}
	var icons []image.Image
	var count int
	rs.Walk(func(typid, id winres.Identifier, lang uint16, data []byte) bool {
		mime := mimetype.Detect(data)
		args.logger("resource %d: type: %v res: %v lang: %v len: %d mime: %v", count, typid, id, lang, len(data), mime)
		if mime.String() != "image/x-icon" {
			return true
		}
		icon, err := rs.GetIconTranslation(id, lang)
		if err != nil {
			args.logger("resource %d: unable to read icon: %v", err)
			return false
		}
		var buf bytes.Buffer
		if err := icon.SaveICO(&buf); err != nil {
			args.logger("resource %d: unable to save icon: %v", err)
			return false
		}
		img, _, err := image.Decode(&buf)
		if err != nil {
			args.logger("resource %d: unable to decode icon: %v", err)
			return false
		}
		b := img.Bounds()
		args.logger("icon %d: dimensions: %dx%d", len(icons)+1, b.Dx(), b.Dy())
		icons = append(icons, img)
		count++
		return true
	})
	if len(icons) == 0 {
		return nil, fmt.Errorf("no icons found")
	}
	page := 0
	if args.Page != 0 {
		if p := int(args.Page - 1); 0 <= p && p < len(icons) {
			page = p
		}
	}
	return icons[page], nil
}

// addBackground adds a background to a image.
func (args *Args) addBackground(mime string, src image.Image) image.Image {
	bg := args.bgc
	switch {
	case bg == nil && mime == "text/plain": // mermaid
		bg = args.mbgc
	}
	switch {
	case bg == nil, mime == "image/svg":
		return src
	}
	start := time.Now()
	b, c := src.Bounds(), *bg
	img := image.NewNRGBA(b)
	for i := range b.Dx() {
		for j := range b.Dy() {
			img.SetNRGBA(i, j, c)
		}
	}
	draw.Draw(img, b, src, image.Point{}, draw.Over)
	args.logger("add bg: %v", time.Since(start))
	return img
}

// decodeMarkdown renders the markdown file, rendering it as a pdf then using
// libvips to export it as a standard image.
func (args *Args) decodeMarkdown(pathName, _ string, r io.ReadCloser) (image.Image, error) {
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
	pdf, err := args.decodeVips(pathName, "", io.NopCloser(buf))
	if err != nil {
		return nil, fmt.Errorf("vips can't load rendered pdf for %s: %w", pathName, err)
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
		vips.SetLogging(func(domain string, level vips.LogLevel, msg string) {
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
		return "err"
	case vips.LogLevelCritical:
		return "crt"
	case vips.LogLevelWarning:
		return "wrn"
	case vips.LogLevelMessage:
		return "mes"
	case vips.LogLevelInfo:
		return "nfo"
	case vips.LogLevelDebug:
		return "dbg"
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
	pathName := path.Base(u.Path)
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
	img, err := fs.args.decodeVips(pathName, "", res.Body)
	if err != nil {
		return nil, fmt.Errorf("md open: render: %w", err)
	}
	fs.args.logger("md open: %s", pathName)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("md open: encode: %w", err)
	}
	b := buf.Bytes()
	return &file{
		name: pathName,
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

type Image struct {
	img image.Image
}

func (img *Image) Image() image.Image {
	return img
}

// RawPNG returns a raw PNG of the image, for use by encoders.
func (img *Image) RawPNG() ([]byte, error) {
	return nil, nil
}

// ColorModel satisfies the [image.Image] interface.
func (img *Image) ColorModel() color.Model {
	return nil
}

// Bounds satisfies the [image.Image] interface.
func (img *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, 0, 0)
}

// At satisfies the [image.Image] interface.
func (img *Image) At(x, y int) color.Color {
	return nil
}

// nopWriteCloser wraps a writer with a noop close method.
type nopWriteCloser struct {
	io.Writer
}

// Close satisfies the io.Closer interface.
func (nopWriteCloser) Close() error {
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

// isBuiltin returns true if the mime type is a supported, builtin Go image
// type.
func isBuiltin(typ string) bool {
	switch typ {
	case
		"image/bmp",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/tiff",
		"image/x-icon":
		return true
	case "image/x-portable-floatmap":
		return false
	}
	return strings.HasPrefix(typ, "image/x-portable-")
}

// isVips returns true if the mime type is supported by libvips.
func isVips(typ string) bool {
	switch typ {
	case "application/pdf":
		return true
	case "image/vnd.adobe.photoshop":
		return false
	}
	return strings.HasPrefix(typ, "image/") &&
		!strings.HasPrefix(typ, "image/x-portable-") &&
		!strings.Contains(typ, "jxr")
}

// isFitz returns true if the mime type is supported by fitz (mupdf).
//
// epub, xps, mobi, fb2
func isFitz(typ, ext string) bool {
	switch {
	case
		typ == "application/epub+zip",           // epub
		typ == "application/x-mobipocket-ebook", // mobi
		typ == "text/fb2+xml",                   // fb2
		typ == "text/xml" && ext == "fb2",
		typ == "image/vnd.adobe.photoshop",       // psd
		typ == "application/zip" && ext == "xps", // xps
		strings.HasPrefix(typ, "image/x-portable-") && typ != "image/x-portable-floatmap":
		return true
	}
	return false
}

// isMermaid returns true if the mime type is supported by the `mmdc` command.
func isMermaid(typ, ext string) bool {
	return typ == "text/plain" && ext == "mmd"
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

// isComicArchive returns true if the type and extension match for a comic
// archive.
func isComicArchive(typ, ext string) bool {
	switch {
	case
		typ == "application/x-7z-compressed" && ext == "cb7", // 7z
		// typ == "application" && ext == "cba",           // ACE -- no support for the compression format
		typ == "application/x-rar-compressed" && ext == "cbr", // rar
		typ == "application/x-tar" && ext == "cbt",            // tar
		typ == "application/zip" && ext == "cbz":              // zip
		return true
	}
	return false
}

// isWindowsPE returns true if the mime type is supported
func isWindowsPE(typ, ext string) bool {
	return typ == "application/vnd.microsoft.portable-executable"
}

// fileExt returns the lower case file extension.
func fileExt(s string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(s), "."))
}

// urlRE matches http/s URLs.
var urlRE = regexp.MustCompile(`(?i)^https?://`)

var (
	vipsOnce    sync.Once
	sofficeOnce sync.Once
	ffmpegOnce  sync.Once
	mmdcOnce    sync.Once
)

var (
	sofficePath string
	ffprobePath string
	ffmpegPath  string
	mmdcPath    string
)

// extensions are the extensions to check for directories.
var extensions = map[string]bool{
	"3g2":      true,
	"3gp":      true,
	"aac":      true,
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
	"flac":     true,
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
	"m4a":      true,
	"m4v":      true,
	"markdown": true,
	"md":       true,
	"mj2":      true,
	"mkv":      true,
	"mov":      true,
	"mp3":      true,
	"mp4":      true,
	"mpeg3":    true,
	"mpeg":     true,
	"mpg":      true,
	"odc":      true,
	"odf":      true,
	"odg":      true,
	"odp":      true,
	"ods":      true,
	"odt":      true,
	"oga":      true,
	"ogg":      true,
	"ogv":      true,
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
	// comic archives
	"cb7": true,
	"cba": true,
	"cbr": true,
	"cbt": true,
	"cbz": true,
	// fitz (mupdf)
	"xps":  true,
	"epub": true,
	"mobi": true,
	"fb2":  true,
	// mermaid
	"mmd": true,
	// windows pe
	"exe": true,
}

// comicExtensions are the extensions of files within a comic archive to
// consider for display.
var comicExtensions = map[string]bool{
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
	"bmp":  true,
	"png":  true,
	"webp": true,
	"tiff": true,
	"tif":  true,
}
