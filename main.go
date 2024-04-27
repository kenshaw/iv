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
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/kenshaw/colors"
	"github.com/kenshaw/rasterm"
	"github.com/spf13/cobra"
	pdf "github.com/stephenafamo/goldmark-pdf"
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
	if err := run(context.Background(), name, version, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, name, version string, cliargs []string) error {
	bg := colors.FromColor(color.Transparent)
	width := uint(0)
	height := uint(0)
	// sideBySide := false
	// diff := false
	var (
		bashCompletion       bool
		zshCompletion        bool
		fishCompletion       bool
		powershellCompletion bool
		noDescriptions       bool
	)
	c := &cobra.Command{
		Use:           name + " [flags] <image1> [image2, ..., imageN]",
		Short:         name + ", a command-line image viewer using terminal graphics",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  false,
		RunE: func(cmd *cobra.Command, cliargs []string) error {
			// completions and short circuits
			switch {
			case bashCompletion:
				return cmd.GenBashCompletionV2(os.Stdout, !noDescriptions)
			case zshCompletion:
				if noDescriptions {
					return cmd.GenZshCompletionNoDesc(os.Stdout)
				}
				return cmd.GenZshCompletion(os.Stdout)
			case fishCompletion:
				return cmd.GenFishCompletion(os.Stdout, !noDescriptions)
			case powershellCompletion:
				if noDescriptions {
					return cmd.GenPowerShellCompletion(os.Stdout)
				}
				return cmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return do(os.Stdout, bg /*, sideBySide, diff*/, width, height, cliargs)
		},
	}
	c.SetVersionTemplate("{{ .Name }} {{ .Version }}\n")
	c.SetArgs(cliargs[1:])
	// flags
	flags := c.Flags()
	flags.BoolVarP(&verbose, "verbose", "v", verbose, "enable verbose")
	flags.IntVar(&vipsConcurrency, "vips-concurrency", vipsConcurrency, "vips concurrency")
	flags.Lookup("vips-concurrency").Hidden = true
	flags.Var(bg.Pflag(), "bg", "background color")
	// flags.BoolVarP(&sideBySide, "side-by-side", "S", sideBySide, "toggle side-by-side mode")
	// flags.BoolVar(&diff, "diff", diff, "toggle diff mode")
	flags.UintVarP(&width, "width", "W", width, "set width")
	flags.UintVarP(&height, "height", "H", height, "set height")
	// completions
	flags.BoolVar(&bashCompletion, "completion-script-bash", false, "output bash completion script and exit")
	flags.BoolVar(&zshCompletion, "completion-script-zsh", false, "output zsh completion script and exit")
	flags.BoolVar(&fishCompletion, "completion-script-fish", false, "output fish completion script and exit")
	flags.BoolVar(&powershellCompletion, "completion-script-powershell", false, "output powershell completion script and exit")
	flags.BoolVar(&noDescriptions, "no-descriptions", false, "disable descriptions in completion scripts")
	// mark hidden
	for _, name := range []string{
		"completion-script-bash", "completion-script-zsh", "completion-script-fish",
		"completion-script-powershell", "no-descriptions",
	} {
		flags.Lookup(name).Hidden = true
	}
	return c.ExecuteContext(ctx)
}

// do renders the specified files to w.
func do(w io.Writer, bg color.Color /*, sideBySide, diff bool*/, width, height uint, args []string) error {
	if !rasterm.Available() {
		return rasterm.ErrTermGraphicsNotAvailable
	}
	if verbose {
		logger = func(s string, v ...interface{}) {
			fmt.Fprintf(os.Stderr, s+"\n", v...)
		}
	}
	// add background color for svgs
	resvg.WithBackground(bg)(resvg.Default)
	if width != 0 || height != 0 {
		resvg.WithScaleMode(resvg.ScaleBestFit)(resvg.Default)
		resvg.WithWidth(int(width))(resvg.Default)
		resvg.WithHeight(int(height))(resvg.Default)
	}
	// collect files
	var files []string
	for i := 0; i < len(args); i++ {
		v, err := open(args[i])
		if err != nil {
			fmt.Fprintf(w, "error: unable to open arg %d: %v\n", i, err)
		}
		files = append(files, v...)
	}
	return render(w, bg, files)
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

// render renders files to w.
func render(w io.Writer, bg color.Color, files []string) error {
	var c color.Color
	if !colors.Is(bg, colors.Transparent) {
		c = color.NRGBAModel.Convert(bg).(color.NRGBA)
	}
	for i := 0; i < len(files); i++ {
		if err := renderFile(w, c, files[i]); err != nil {
			fmt.Fprintf(w, "error: unable to render arg %d: %v\n", i, err)
		}
	}
	return nil
}

// renderFile renders the specified file to w.
func renderFile(w io.Writer, bg color.Color, name string) error {
	fmt.Fprintln(w, name+":")
	start := time.Now()
	typ, img, err := openImageFile(name)
	if err != nil {
		return err
	}
	logger("open: %v", time.Now().Sub(start))
	if typ != "svg" && bg != nil {
		img = addBackground(bg.(color.NRGBA), img)
		logger("add bg: %v", time.Now().Sub(start))
	}
	now := time.Now()
	if err = rasterm.Encode(w, img); err != nil {
		return err
	}
	logger("out: %v", time.Now().Sub(now))
	logger("total: %v", time.Now().Sub(start))
	return nil
}

// openImageFile opens the image file.
func openImageFile(name string) (string, image.Image, error) {
	f, g := openImage, openVips
	if m := extRE.FindStringSubmatch(name); m != nil {
		switch strings.ToLower(m[1]) {
		case "avif", "heic", "heiv", "heif", "pdf", "jp2", "jxl":
			f, g = g, f
		case "markdown", "md":
			f, g = openMarkdown, nil
		}
	}
	typ, img, err := f(name)
	switch {
	case err == nil:
		return typ, img, nil
	case g == nil:
		return "", nil, err
	}
	logger("initial open failed: %v", err)
	return g(name)
}

// openImage opens the image using Go's standard image.Decode.
func openImage(name string) (string, image.Image, error) {
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
	return typ, img, nil
}

// openVips opens the image using libvips.
func openVips(name string) (string, image.Image, error) {
	initOnce.Do(vipsInit)
	start := time.Now()
	p := vips.NewImportParams()
	p.Density.Set(300)
	v, err := vips.LoadImageFromFile(name, p)
	if err != nil {
		return "", nil, fmt.Errorf("vips can't open %s: %w", name, err)
	}
	logger("vips load: %v", time.Now().Sub(start))
	ext, w, h := strings.TrimPrefix(v.OriginalFormat().FileExt(), "."), v.Width(), v.Height()
	logger("vips format: %s dimensions: %dx%d pages: %d", ext, w, h, v.Pages())
	if ext == "pdf" {
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), 2000, 2000)
		if scale != 1.0 {
			start = time.Now()
			if err := v.Resize(float64(scale), vips.KernelAuto); err != nil {
				return "", nil, fmt.Errorf("vips unable to scale pdf: %w", err)
			}
			logger("vips resize: %v", time.Now().Sub(start))
		}
	}
	return vipsExport(v)
}

// openMarkdown opens the markdown file, rendering it as a pdf then using
// libvips to export it as a standard image.
func openMarkdown(name string) (string, image.Image, error) {
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
	logger("markdown open: %v", time.Now().Sub(start))
	start = time.Now()
	md := goldmark.New(
		goldmark.WithRenderer(
			pdf.New(
				// pdf.WithContext(context.Background()),
				pdf.WithTraceWriter(loggerWriter{}),
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
	if err := os.WriteFile("blah.pdf", buf.Bytes(), 0o644); err != nil {
		panic(err)
	}
	logger("markdown convert: %v", time.Now().Sub(start))
	initOnce.Do(vipsInit)
	start = time.Now()
	v, err := vips.NewImageFromReader(buf)
	if err != nil {
		return "", nil, fmt.Errorf("vips can't load rendered pdf for %s: %w", name, err)
	}
	logger("markdown render: %v", time.Now().Sub(start))
	return vipsExport(v)
}

// addBackground adds a background to the image.
func addBackground(bg color.NRGBA, fg image.Image) image.Image {
	b := fg.Bounds()
	img := image.NewNRGBA(b)
	for i := 0; i < b.Dx(); i++ {
		for j := 0; j < b.Dy(); j++ {
			img.SetNRGBA(i, j, bg)
		}
	}
	draw.Draw(img, b, fg, image.Point{}, draw.Over)
	return img
}

type loggerWriter struct{}

func (loggerWriter) Write(buf []byte) (int, error) {
	logger("md: %s", string(bytes.TrimRightFunc(buf, unicode.IsSpace)))
	return len(buf), nil
}

// vipsExport exports the vips image as a png image.
func vipsExport(v *vips.ImageRef) (string, image.Image, error) {
	start := time.Now()
	img, err := v.ToImage(&vips.ExportParams{
		Format: vips.ImageTypePNG,
	})
	if err != nil {
		return "", nil, fmt.Errorf("vips can't export %s: %w", name, err)
	}
	logger("vips export: %v", time.Now().Sub(start))
	return "png", img, nil
}

// global vars.
var (
	logger          = func(string, ...interface{}) {}
	verbose         = false
	vipsConcurrency = runtime.NumCPU()
)

// initOnce is the init guard.
var initOnce sync.Once

// vipsInit initializes the vip package.
func vipsInit() {
	level := vips.LogLevelError
	if verbose {
		level = vips.LogLevelDebug
	}
	vips.LoggingSettings(func(domain string, level vips.LogLevel, msg string) {
		logger("vips %s: %s %s", vipsLevel(level), domain, strings.TrimSpace(msg))
	}, level)
	vips.Startup(&vips.Config{
		ConcurrencyLevel: vipsConcurrency,
	})
}

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
