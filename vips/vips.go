// Package vips has the shared internals for the vips iv decoder and encoder.
package vips

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/resvg"
)

// maxPdfDimension is the longest edge a rendered pdf page is scaled to.
const maxPdfDimension = 2000

var initOnce sync.Once

// Init starts vips, using the logging and concurrency settings on the context.
// Safe to call repeatedly; only the first call has an effect.
func Init(ctx context.Context) {
	initOnce.Do(func() {
		c := ivctx.Get(ctx)
		start := time.Now()
		level := vips.LogLevelError
		if c.Verbose {
			level = vips.LogLevelDebug
		}
		vips.SetLogging(func(domain string, level vips.LogLevel, msg string) {
			c.Logger("vips %s: %s %s", levelString(level), domain, strings.TrimSpace(msg))
		}, level)
		var config *vips.Config
		if c.VipsConcurrency != 0 {
			config = &vips.Config{
				ConcurrencyLevel: int(c.VipsConcurrency),
			}
		}
		vips.Startup(config)
		c.Logger("vips init: %v", time.Since(start))
	})
}

// Shutdown shuts down vips.
func Shutdown() {
	vips.Shutdown()
}

// Export exports the vips image as a png and decodes it as a Go image.
func Export(ctx context.Context, v *vips.Image) (image.Image, error) {
	if v == nil {
		return nil, fmt.Errorf("vips export: invalid image")
	}
	start := time.Now()
	ext, w, h := strings.TrimPrefix(string(v.Format()), "."), v.Width(), v.Height()
	ivctx.Logf(ctx, "vips format: %s dimensions: %dx%d pages: %d", ext, w, h, v.Pages())
	if ext == "pdf" {
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), maxPdfDimension, maxPdfDimension)
		if scale != 1.0 {
			if err := v.Resize(float64(scale), nil); err != nil {
				return nil, fmt.Errorf("vips unable to scale pdf: %w", err)
			}
			ivctx.Logf(ctx, "vips resize: %v", time.Since(start))
		}
	}
	start = time.Now()
	// the buffer saver is used rather than the target one: several libvips
	// savers seek within their output
	buf, err := v.PngsaveBuffer(nil)
	if err != nil {
		return nil, fmt.Errorf("vips export: %w", err)
	}
	ivctx.Logf(ctx, "vips export: %v", time.Since(start))
	start = time.Now()
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("vips decode: %w", err)
	}
	ivctx.Logf(ctx, "vips decode: %v", time.Since(start))
	return img, nil
}

// IsEncryptedErr reports whether the error is the vips "document is encrypted"
// error, raised for password protected pdfs.
func IsEncryptedErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "document is encrypted")
}

// levelString returns the vips log level as a short string.
func levelString(level vips.LogLevel) string {
	switch level {
	case vips.LogLevelError:
		return "err"
	case vips.LogLevelCritical:
		return "crt"
	case vips.LogLevelWarning:
		return "wrn"
	case vips.LogLevelMessage:
		return "msg"
	case vips.LogLevelInfo:
		return "nfo"
	case vips.LogLevelDebug:
		return "dbg"
	}
	return fmt.Sprintf("(%d)", level)
}
