// Package ivctx provides the configuration, context plumbing, and shared
// helpers used by the iv decoder and encoder pipelines.
package ivctx

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/kenshaw/colors"
	"github.com/tdewolff/canvas"
	"github.com/xo/ox"
)

func init() {
	ox.RegisterTypeName(ox.Type("mode"), "*ivctx.Mode")
	ox.RegisterTextType(NewMode)
}

// Config is the iv pipeline configuration. A nil or absent config behaves as
// [New], so that decoders can always be exercised without a command line.
type Config struct {
	Verbose         bool
	Quiet           bool
	Mode            *Mode
	Width           uint
	Height          uint
	MinWidth        uint
	MinHeight       uint
	DPI             uint
	Page            uint
	Fg              *colors.Color
	Bg              *colors.Color
	Border          uint
	FontSize        uint
	FontStyle       canvas.FontStyle
	FontVariant     canvas.FontVariant
	FontFg          *colors.Color
	FontBg          *colors.Color
	FontDPI         uint
	FontMargin      uint
	TimeCode        time.Duration
	VipsConcurrency uint
	MermaidTheme    string
	MermaidBg       *colors.Color
	BlitzDark       bool
	PDFPage         string
	PDFMargin       uint
	Password        string
	ForceMime       string

	// Logger is the verbose logger. Always non-nil after [Config.Init].
	Logger func(string, ...any)
	// Warner reports something the user should see whether or not they asked
	// for verbose output -- a page truncated, a format degraded. Always
	// non-nil after [Config.Init].
	Warner func(string, ...any)
}

// New creates a config with the same defaults as the iv command.
func New() *Config {
	mode := ModeBestFit
	c := &Config{
		Mode:       &mode,
		MinWidth:   64,
		MinHeight:  64,
		DPI:        300,
		Fg:         named(colors.Dimgray),
		Bg:         named(colors.Transparent),
		Border:     30,
		FontSize:   48,
		FontFg:     named(colors.Black),
		FontBg:     named(colors.White),
		FontDPI:    100,
		FontMargin: 5,
		MermaidBg:  named(colors.White),
		PDFPage:    "fit",
		PDFMargin:  36,
	}
	c.Init()
	return c
}

// named returns the named color as a [colors.Color].
func named(n colors.NamedColor) *colors.Color {
	c := n.Color()
	return &c
}

// Init normalizes the config, ensuring the scaling mode is set and the
// loggers are non-nil.
func (c *Config) Init() {
	if !c.Mode.Valid() {
		mode := ModeBestFit
		c.Mode = &mode
	}
	if c.Logger == nil {
		c.Logger = func(string, ...any) {}
	}
	if c.Warner == nil {
		c.Warner = func(string, ...any) {}
	}
}

// contextKey is the context key type.
type contextKey int

const (
	configKey contextKey = iota
	pathNameKey
	mimeKey
)

// WithConfig adds the config to the context.
func WithConfig(ctx context.Context, c *Config) context.Context {
	c.Init()
	return context.WithValue(ctx, configKey, c)
}

// Get returns the config on the context, returning a default config when not
// present.
func Get(ctx context.Context) *Config {
	if ctx != nil {
		if c, ok := ctx.Value(configKey).(*Config); ok && c != nil {
			return c
		}
	}
	return New()
}

// WithPathName adds the path name of the target being decoded to the context.
func WithPathName(ctx context.Context, pathName string) context.Context {
	return context.WithValue(ctx, pathNameKey, pathName)
}

// PathName returns the path name of the target being decoded.
func PathName(ctx context.Context) string {
	s, _ := ctx.Value(pathNameKey).(string)
	return s
}

// WithMime adds the mime type of the content being decoded to the context.
func WithMime(ctx context.Context, mime string) context.Context {
	return context.WithValue(ctx, mimeKey, mime)
}

// Mime returns the mime type of the content being decoded.
func Mime(ctx context.Context) string {
	s, _ := ctx.Value(mimeKey).(string)
	return s
}

// Logf writes a verbose log message.
func Logf(ctx context.Context, s string, v ...any) {
	Get(ctx).Logger(s, v...)
}

// Warnf reports something the user should see whether or not they asked for
// verbose output. The message is also logged, so a verbose run keeps it in
// sequence with everything around it.
func Warnf(ctx context.Context, s string, v ...any) {
	c := Get(ctx)
	c.Logger(s, v...)
	c.Warner(s, v...)
}

// Page returns the zero-indexed page to display, clamped to [0, n). Returns 0
// when no page was requested or when the requested page is out of range.
func Page(ctx context.Context, n int) int {
	if page := int(Get(ctx).Page) - 1; 0 <= page && page < n {
		return page
	}
	return 0
}

// AddBorder adds a border of the configured width and background color to src.
func AddBorder(ctx context.Context, src image.Image) image.Image {
	c := Get(ctx)
	b, w := src.Bounds(), int(c.Border)
	x, y := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, x+2*w, y+2*w))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bgColor(c.Bg)}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(w, w, w+x, w+y), src, b.Min, draw.Over)
	return dst
}

// AddBackground composites src over the configured background color. Images
// that supply their own background (svg) and fully transparent backgrounds are
// returned as-is.
func AddBackground(ctx context.Context, mime string, src image.Image) image.Image {
	c := Get(ctx)
	bg := c.Bg
	if isTransparent(bg) && mime == "text/plain" { // mermaid
		bg = c.MermaidBg
	}
	switch {
	case isTransparent(bg), strings.HasPrefix(mime, "image/svg"):
		return src
	}
	start := time.Now()
	b, n := src.Bounds(), bg.NRGBA()
	dst := image.NewNRGBA(b)
	for i := range b.Dx() {
		for j := range b.Dy() {
			dst.SetNRGBA(b.Min.X+i, b.Min.Y+j, n)
		}
	}
	draw.Draw(dst, b, src, b.Min, draw.Over)
	c.Logger("add bg: %v", time.Since(start))
	return dst
}

// isTransparent returns true when c is nil or fully transparent.
func isTransparent(c *colors.Color) bool {
	return c == nil || c.A == 0
}

// bgColor returns c as a [color.Color], defaulting to transparent.
func bgColor(c *colors.Color) color.Color {
	if c == nil {
		return color.Transparent
	}
	return *c
}

// FileExt returns the lower case file extension of s, without the leading dot.
func FileExt(s string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(s), "."))
}

// NopWriteCloser wraps a writer with a no-op close method.
type NopWriteCloser struct {
	io.Writer
}

// Close satisfies the [io.Closer] interface.
func (NopWriteCloser) Close() error {
	return nil
}
