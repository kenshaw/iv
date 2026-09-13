// Package ivcmd contains the iv command.
package ivcmd

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/decoder"
	_ "github.com/kenshaw/iv/decoder/all"
	"github.com/kenshaw/iv/encoder"
	_ "github.com/kenshaw/iv/encoder/all"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/rasterm"
	"github.com/tdewolff/canvas"
	_ "github.com/xo/ox/color"
	"github.com/xo/resvg"
)

// DefaultEncoder is the encoder used when writing to a terminal.
const DefaultEncoder = "rasterm"

// Args are the iv command arguments.
type Args struct {
	Verbose         bool               `ox:"enable verbose,short:v"`
	Quiet           bool               `ox:"enable quiet,short:q"`
	Mode            *ivctx.Mode        `ox:"scaling mode,short:m,default:best-fit"`
	Width           uint               `ox:"display width in pixels - 0 takes it from the terminal,short:W"`
	Height          uint               `ox:"display height in pixels - 0 takes it from the terminal,short:H"`
	MinWidth        uint               `ox:"minimum width in pixels,short:w,default:64"`
	MinHeight       uint               `ox:"minimum height in pixels,short:h,default:64"`
	DPI             uint               `ox:"image dpi,default:300,name:dpi"`
	Page            uint               `ox:"page to display,short:p"`
	Fg              *colors.Color      `ox:"foreground color,default:dimgray"`
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
	BlitzDark       bool               `ox:"render documents and pages dark"`
	PDFPage         string             `ox:"pdf page size - fit/a4/letter,default:fit,name:pdf-page"`
	PDFMargin       uint               `ox:"pdf page margin in points,default:36,name:pdf-margin"`
	Password        string             `ox:"password for encrypted documents"`
	ForceMime       string             `ox:"force mime type"`
	Out             string             `ox:"write to file instead of the terminal,short:o"`
	Encoder         string             `ox:"output encoder"`
	List            bool               `ox:"list registered decoders and encoders"`
}

// New creates the iv command arguments.
func New() *Args {
	return new(Args)
}

// Config builds the pipeline config from the command arguments.
func (args *Args) Config(stderr io.Writer) *ivctx.Config {
	c := &ivctx.Config{
		Verbose:         args.Verbose,
		Quiet:           args.Quiet,
		Mode:            args.Mode,
		Width:           args.Width,
		Height:          args.Height,
		MinWidth:        args.MinWidth,
		MinHeight:       args.MinHeight,
		DPI:             args.DPI,
		Page:            args.Page,
		Fg:              args.Fg,
		Bg:              args.Bg,
		Border:          args.Border,
		FontSize:        args.FontSize,
		FontStyle:       args.FontStyle,
		FontVariant:     args.FontVariant,
		FontFg:          args.FontFg,
		FontBg:          args.FontBg,
		FontDPI:         args.FontDPI,
		FontMargin:      args.FontMargin,
		TimeCode:        args.TimeCode,
		VipsConcurrency: args.VipsConcurrency,
		MermaidIcons:    args.MermaidIcons,
		MermaidBg:       args.MermaidBg,
		BlitzDark:       args.BlitzDark,
		PDFPage:         args.PDFPage,
		PDFMargin:       args.PDFMargin,
		Password:        args.Password,
		ForceMime:       args.ForceMime,
	}
	if args.Verbose {
		c.Logger = func(s string, v ...any) {
			fmt.Fprintf(stderr, s+"\n", v...)
		}
	}
	if !args.Quiet && !args.Verbose {
		// verbose already puts these on stderr, in sequence
		c.Warner = func(s string, v ...any) {
			fmt.Fprintf(stderr, "warning: "+s+"\n", v...)
		}
	}
	c.Init()
	return c
}

// Run returns the command's exec func.
func (args *Args) Run(stdout, stderr io.Writer) func(context.Context, []string) error {
	return func(ctx context.Context, cliargs []string) error {
		return args.Exec(ctx, stdout, stderr, cliargs)
	}
}

// Exec runs the iv command.
func (args *Args) Exec(ctx context.Context, stdout, stderr io.Writer, cliargs []string) error {
	if args.List {
		return List(stdout)
	}
	enc, err := args.encoder()
	if err != nil {
		return err
	}
	c := args.Config(stderr)
	ctx = ivctx.WithConfig(ctx, c)
	// a decoder holding an engine -- graphviz, lottie, blitz -- releases it
	// here, and a failure to is worth saying rather than dropping
	defer func() {
		if err := decoder.Close(ctx); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
		}
	}()
	args.resolveDisplay(ctx, c, stdout)
	args.configureResvg(c)
	targets, errs := Targets(cliargs...)
	for _, err := range errs {
		fmt.Fprintf(stderr, "error: %v\n", err)
	}
	if len(targets) == 0 {
		return errors.New("no targets to render")
	}
	if args.Out != "" && len(targets) != 1 {
		return fmt.Errorf("--out requires exactly one target, got %d", len(targets))
	}
	var failed bool
	for _, t := range targets {
		if err := args.render(ctx, stdout, enc, t); err != nil {
			failed = true
			fmt.Fprintf(stderr, "error: render %q: %v\n", t.Name, err)
		}
	}
	if failed {
		return errors.New("one or more targets could not be rendered")
	}
	return nil
}

// encoder returns the encoder to use, verifying that terminal graphics are
// available when writing to a terminal.
func (args *Args) encoder() (*encoder.Entry, error) {
	switch {
	case args.Encoder != "":
		e, ok := encoder.Get(args.Encoder)
		if !ok {
			return nil, fmt.Errorf("%q: %w (available: %s)", args.Encoder, encoder.ErrNotRegistered, strings.Join(encoder.Names(), ", "))
		}
		return e, nil
	case args.Out != "":
		ext := ivctx.FileExt(args.Out)
		e, ok := encoder.ForExt(ext)
		if !ok {
			return nil, fmt.Errorf("no encoder for extension %q (available: %s)", ext, strings.Join(encoder.Names(), ", "))
		}
		return e, nil
	}
	if !rasterm.Available() {
		return nil, rasterm.ErrTermGraphicsNotAvailable
	}
	e, ok := encoder.Get(DefaultEncoder)
	if !ok {
		return nil, fmt.Errorf("%q: %w", DefaultEncoder, encoder.ErrNotRegistered)
	}
	return e, nil
}

// resolveDisplay fills in the display size from the terminal when none was
// configured, so that the rest of the pipeline works against a size in pixels
// however it was arrived at.
//
// Only terminal output gets a size this way. Writing a file with --out has no
// terminal to fit, and a conversion that silently downscaled to whatever
// terminal happened to be attached would be a surprising thing for a
// conversion to do -- there, an unset display size stays unset, and only an
// explicit --width/--height bounds the result.
func (args *Args) resolveDisplay(ctx context.Context, c *ivctx.Config, stdout io.Writer) {
	if c.Mode.Get() == ivctx.ModeNone || c.Width != 0 || c.Height != 0 || args.Out != "" {
		return
	}
	f, ok := stdout.(*os.File)
	if !ok {
		return
	}
	t, ok := ivctx.TermSize(f)
	if !ok {
		return
	}
	// the name printed above the image and the prompt that follows it both
	// want a line, and an image sized to the last pixel of the screen scrolls
	// the top of itself away
	_, cellHeight := t.CellSize()
	c.Width, c.Height = uint(t.Width), uint(max(t.Height-2*cellHeight, cellHeight))
	how := "estimated"
	if t.Exact {
		how = "reported"
	}
	ivctx.Logf(ctx, "terminal: %dx%d cells, %dx%d pixels (%s), display: %dx%d",
		t.Cols, t.Rows, t.Width, t.Height, how, c.Width, c.Height)
}

// configureResvg applies the background to the svg renderer.
//
// The display size is deliberately not passed on. resvg's best fit grows an
// svg to fill whatever box it is given, so handing it the terminal meant every
// svg arrived at terminal width whatever size it declared -- a 1000 pixel card
// blown up to 1900 and resampled back down again by whatever came next. An
// svg is rasterized at the size it declares, and [ivctx.Fit] then applies the
// same ceiling and floor to it as to any other image.
func (args *Args) configureResvg(c *ivctx.Config) {
	resvg.WithBackground(c.Bg)(resvg.Default)
}

// render decodes the target and writes it with the encoder.
func (args *Args) render(ctx context.Context, stdout io.Writer, enc *encoder.Entry, t Target) error {
	if !args.Quiet && args.Out == "" {
		fmt.Fprintln(stdout, t.Name+":")
	}
	start := time.Now()
	var img image.Image
	var mime string
	var err error
	if t.IsString {
		img, mime, err = decoder.DecodeString(ctx, t.Name)
	} else {
		img, mime, err = decoder.DecodeFile(ctx, t.Name)
	}
	if err != nil {
		return err
	}
	// scaled before the background is composited, so the compositing runs
	// over the pixels that are actually displayed rather than all of them
	img = ivctx.AddBackground(ctx, mime, ivctx.Fit(ctx, img))
	w := stdout
	if args.Out != "" {
		f, err := os.Create(args.Out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	encStart := time.Now()
	if err := enc.Encode(ctx, w, img); err != nil {
		return err
	}
	ivctx.Logf(ctx, "%s encode: %v", enc.Name, time.Since(encStart))
	ivctx.Logf(ctx, "total: %v", time.Since(start))
	return nil
}

// Target is something to render: a path on disk, or a bare string argument
// such as a data: URL.
type Target struct {
	Name     string
	IsString bool
}

// Targets expands the command line arguments into render targets. Directories
// expand to the files within them with a registered extension. Returns the
// targets along with the errors for the arguments that could not be expanded.
func Targets(cliargs ...string) ([]Target, []error) {
	var targets []Target
	var errs []error
	for _, name := range cliargs {
		v, err := expand(name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		targets = append(targets, v...)
	}
	return targets, errs
}

// expand expands a single command line argument into render targets.
func expand(name string) ([]Target, error) {
	switch fi, err := os.Stat(name); {
	case err == nil && fi.IsDir():
		entries, err := os.ReadDir(name)
		if err != nil {
			return nil, err
		}
		var targets []Target
		for _, entry := range entries {
			if s := entry.Name(); !entry.IsDir() && decoder.SupportedExt(s) {
				targets = append(targets, Target{Name: filepath.Join(name, s)})
			}
		}
		if len(targets) == 0 {
			return nil, fmt.Errorf("open %q: no renderable files", name)
		}
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Name < targets[j].Name
		})
		return targets, nil
	case err == nil:
		return []Target{{Name: name}}, nil
	}
	// not on disk; a string decoder may still claim it
	if len(decoder.MatchString(context.Background(), name)) != 0 {
		return []Target{{Name: name, IsString: true}}, nil
	}
	return nil, fmt.Errorf("open %q: %w", name, decoder.ErrNotSupported)
}

// List writes the registered decoders and encoders to w.
func List(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "decoders:"); err != nil {
		return err
	}
	for _, d := range decoder.All() {
		if _, err := fmt.Fprintf(w, "  %-14s %s\n", d.Name, d.Desc); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nencoders:"); err != nil {
		return err
	}
	for _, e := range encoder.All() {
		if _, err := fmt.Fprintf(w, "  %-14s %s\n", e.Name, e.Desc); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\nextensions:\n  %s\n", strings.Join(decoder.Extensions(), " "))
	return err
}
