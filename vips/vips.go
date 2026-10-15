// Package vips has the shared internals for the vips iv decoder and encoder.
package vips

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cshum/vipsgen/vips"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
	"github.com/xo/resvg"
)

// maxPdfDimension is the longest edge a rendered pdf page is capped at. A
// page is rendered at the configured dpi, which at 300 puts an A4 sheet past
// 2400 pixels before anything has asked for it that big.
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

// Image is an image that libvips holds. It satisfies the [image.Image]
// interface, and keeps its pixels in libvips until something asks for them:
//
//   - [Image.Resize] and [Image.Flatten] run in libvips, and return a new
//     Image. The pipeline scales and composites an Image with them.
//   - [Image.PNG] encodes the image as a PNG in libvips, the first time that it
//     is called. The Kitty and iTerm encoders of rasterm send the PNG as it
//     is, so the encode is the last step before the terminal.
//   - [Image.Vips] returns the image to the libvips file encoders.
//   - [Image.At], [Image.ColorModel] and [Image.Unwrap] copy the pixels out of
//     libvips into a Go image, the first time that one of them is called. An
//     encoder that encodes the pixels itself uses the Go image.
//
// libvips decodes the pixels only when one of these asks for them, so a
// decode error can arrive then. [Image.Err] returns it, and the encoder
// entries of iv return it after an encode.
//
// An Image never changes the image that libvips holds, so the methods are
// safe to call from more than one goroutine. Call [Image.Close] to release the
// memory that libvips holds.
type Image struct {
	v *vips.Image

	pixOnce sync.Once
	pix     image.Image

	pngOnce sync.Once
	png     []byte

	mu  sync.Mutex
	err error
}

// Open returns v as an [Image]. A pdf page that is larger than
// maxPdfDimension is scaled down to it first.
func Open(ctx context.Context, v *vips.Image) (*Image, error) {
	if v == nil {
		return nil, errors.New("vips open: invalid image")
	}
	ext, w, h := strings.TrimPrefix(string(v.Format()), "."), v.Width(), v.Height()
	ivctx.Logf(ctx, "vips format: %s dimensions: %dx%d bands: %d pages: %d", ext, w, h, v.Bands(), v.Pages())
	if ext == "pdf" {
		// a cap, not a target: best fit grows as readily as it shrinks, and
		// a page already inside the cap has nothing to gain from being
		// resampled up to it
		_, _, scale, _ := resvg.ScaleBestFit.Scale(uint(w), uint(h), maxPdfDimension, maxPdfDimension)
		if scale = min(scale, 1.0); scale != 1.0 {
			start := time.Now()
			if err := v.Resize(float64(scale), nil); err != nil {
				return nil, fmt.Errorf("vips scaling pdf: %w", err)
			}
			ivctx.Logf(ctx, "vips resize: %v", time.Since(start))
		}
	}
	return newImage(v), nil
}

// newImage returns an [Image] that holds v.
func newImage(v *vips.Image) *Image {
	return &Image{
		v: v,
	}
}

// Err returns the first error that libvips returned for the pixels of the
// image, or nil.
func (im *Image) Err() error {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.err
}

// fail keeps err as the error of the image, when it is the first one.
func (im *Image) fail(err error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.err == nil {
		im.err = err
	}
}

// Bounds satisfies the [image.Image] interface.
func (im *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, im.v.Width(), im.v.Height())
}

// ColorModel satisfies the [image.Image] interface.
func (im *Image) ColorModel() color.Model {
	return im.Unwrap().ColorModel()
}

// At satisfies the [image.Image] interface.
func (im *Image) At(x, y int) color.Color {
	return im.Unwrap().At(x, y)
}

// Opaque returns true when the image has no alpha band. It reads the bands
// and the interpretation, and not the pixels.
func (im *Image) Opaque() bool {
	switch im.v.Bands() {
	case 1, 3:
		return true
	case 4:
		return im.v.Interpretation() == vips.InterpretationCmyk
	}
	return false
}

// Unwrap returns the pixels of the image as a Go image: an [image.NRGBA], or
// an [image.NRGBA64] for a 16-bit image. It copies them out of libvips the
// first time that it is called. If the copy fails, it keeps the error for
// [Image.Err] and returns a transparent image of the same size.
func (im *Image) Unwrap() image.Image {
	im.pixOnce.Do(func() {
		pix, err := im.export()
		if err != nil {
			im.fail(fmt.Errorf("vips export: %w", err))
			pix = image.NewNRGBA(im.Bounds())
		}
		im.pix = pix
	})
	return im.pix
}

// export copies the pixels out of a copy of the image, because [export]
// changes the image that it is given.
func (im *Image) export() (image.Image, error) {
	c, err := im.v.Copy(nil)
	if err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}
	defer c.Close()
	return export(c)
}

// PNG satisfies the [rasterm.PNGImage] interface. It encodes the image as an
// 8-bit PNG in libvips the first time that it is called, with the
// compression level of [pngCompression] and no metadata. If the encode
// fails, it keeps the error for [Image.Err] and returns nil.
func (im *Image) PNG() []byte {
	im.pngOnce.Do(func() {
		opts := vips.DefaultPngsaveBufferOptions()
		opts.Compression, opts.Filter, opts.Keep = pngCompression, vips.PngFilterSub, vips.KeepNone
		buf, err := im.v.PngsaveBuffer(opts)
		if err != nil {
			im.fail(fmt.Errorf("vips png: %w", err))
			return
		}
		im.png = buf
	})
	return im.png
}

// pngCompression is the zlib compression level of [Image.PNG]. Level 1 takes
// the least time, and rasterm uses the same level when it encodes a PNG. The
// sub filter of PNG took less time than no filter for a photo, and made a
// smaller file: 33 ms and 1498 KiB, against 42 ms and 2151 KiB, for
// testdata/vips/precision.jxl.
const pngCompression = 1

// EncodeJPEG satisfies the JPEGEncoder interface of the ivimg package. It
// encodes the image as a JPEG of the quality in libvips, with no metadata. A
// JPEG has no alpha channel, so it is for an image with no alpha band.
func (im *Image) EncodeJPEG(quality int) ([]byte, error) {
	opts := vips.DefaultJpegsaveBufferOptions()
	opts.Q, opts.Keep = quality, vips.KeepNone
	buf, err := im.v.JpegsaveBuffer(opts)
	if err != nil {
		return nil, fmt.Errorf("vips jpeg: %w", err)
	}
	return buf, nil
}

// Vips returns the image that libvips holds. Do not change it: the [Image]
// and every image made from it read the same pixels.
func (im *Image) Vips() *vips.Image {
	return im.v
}

// Resize satisfies the Resizer interface of the ivimg package. It scales a
// copy of the image in libvips, with the nearest neighbor kernel where
// [ivimg.Nearest] is true and the Lanczos 3 kernel of libvips otherwise.
func (im *Image) Resize(ctx context.Context, w, h int) (image.Image, error) {
	b := im.Bounds()
	c, err := im.v.Copy(nil)
	if err != nil {
		return nil, fmt.Errorf("vips resize: copy: %w", err)
	}
	kernel := vips.KernelLanczos3
	if ivimg.Nearest(b.Dx(), b.Dy(), w, h) {
		kernel = vips.KernelNearest
	}
	if err := c.Resize(float64(w)/float64(b.Dx()), &vips.ResizeOptions{
		Kernel: kernel,
		Vscale: float64(h) / float64(b.Dy()),
	}); err != nil {
		c.Close()
		return nil, fmt.Errorf("vips resize: %w", err)
	}
	if c.Width() != w || c.Height() != h {
		ivctx.Logf(ctx, "vips resize: asked for %dx%d, got %dx%d", w, h, c.Width(), c.Height())
	}
	return newImage(c), nil
}

// Flatten satisfies the Flattener interface of the ivimg package. It
// composites a copy of the image over the color in libvips. The copy is
// converted to 8-bit sRGB first, so that the color is on the scale of the
// pixels.
func (im *Image) Flatten(_ context.Context, r, g, b uint8) (image.Image, error) {
	c, err := im.v.Copy(nil)
	if err != nil {
		return nil, fmt.Errorf("vips flatten: copy: %w", err)
	}
	if err := srgb8(c); err != nil {
		c.Close()
		return nil, fmt.Errorf("vips flatten: %w", err)
	}
	if err := c.Flatten(&vips.FlattenOptions{
		Background: []float64{float64(r), float64(g), float64(b)},
		MaxAlpha:   255,
	}); err != nil {
		c.Close()
		return nil, fmt.Errorf("vips flatten: %w", err)
	}
	return newImage(c), nil
}

// Close releases the memory that libvips holds for the image. Each [Image]
// made from it holds its own reference, which its own Close releases.
func (im *Image) Close() error {
	im.v.Close()
	return nil
}

// srgb8 converts v to 8-bit sRGB, in place.
func srgb8(v *vips.Image) error {
	if v.Interpretation() != vips.InterpretationSrgb {
		if err := v.Colourspace(vips.InterpretationSrgb, nil); err != nil {
			return fmt.Errorf("colourspace: %w", err)
		}
	}
	if v.BandFormat() != vips.BandFormatUchar {
		if err := v.Cast(vips.BandFormatUchar, nil); err != nil {
			return fmt.Errorf("cast: %w", err)
		}
	}
	return nil
}

// export converts the vips image to the Go image with the same memory layout.
//
// libvips writes its pixels densely -- band order, no padding between rows --
// which is exactly what [image.NRGBA] and [image.NRGBA64] hold, so the work is
// normalizing the image to four bands at the right depth and then taking the
// buffer. libvips keeps alpha unpremultiplied, which is what the N in those
// two type names means.
func export(v *vips.Image) (image.Image, error) {
	// a 16 bit image keeps its depth. Everything else comes out 8 bit, the
	// float formats included, where the colourspace conversion is what brings
	// the values into range.
	deep := v.BandFormat() == vips.BandFormatUshort
	space, format := vips.InterpretationSrgb, vips.BandFormatUchar
	if deep {
		space, format = vips.InterpretationRgb16, vips.BandFormatUshort
	}
	// bring grayscale, cmyk and the rest to colour, which is also what maps a
	// float image's values into range
	if v.Interpretation() != space {
		if err := v.Colourspace(space, nil); err != nil {
			return nil, fmt.Errorf("colourspace: %w", err)
		}
	}
	// the band count is what says whether there is an alpha band, not
	// [vips.Image.HasAlpha], which answers from the interpretation and so gets
	// it wrong for an image that has none
	switch b := v.Bands(); {
	case b == 3:
		if err := v.Addalpha(); err != nil {
			return nil, fmt.Errorf("addalpha: %w", err)
		}
	case b > 4:
		// colour, alpha, and whatever else the source was carrying
		if err := v.ExtractBand(0, &vips.ExtractBandOptions{N: 4}); err != nil {
			return nil, fmt.Errorf("extract band: %w", err)
		}
	}
	// the colourspace conversion can change the depth on its way through
	if v.BandFormat() != format {
		if err := v.Cast(format, nil); err != nil {
			return nil, fmt.Errorf("cast: %w", err)
		}
	}
	buf, err := v.WriteToMemory()
	if err != nil {
		return nil, err
	}
	w, h, depth := v.Width(), v.Height(), 4
	if deep {
		depth = 8
	}
	if n := w * h * depth; len(buf) != n {
		return nil, fmt.Errorf("expected %d bytes for a %dx%d image, got %d", n, w, h, len(buf))
	}
	rect := image.Rect(0, 0, w, h)
	if !deep {
		return &image.NRGBA{Pix: buf, Stride: w * 4, Rect: rect}, nil
	}
	// an [image.NRGBA64] holds its samples big endian, where libvips holds
	// them in the machine's own order
	pix := make([]byte, len(buf))
	for i := 0; i+1 < len(buf); i += 2 {
		binary.BigEndian.PutUint16(pix[i:], binary.NativeEndian.Uint16(buf[i:]))
	}
	return &image.NRGBA64{Pix: pix, Stride: w * 8, Rect: rect}, nil
}

// SaveFunc encodes a vips image in a specific format.
type SaveFunc func(*vips.Image) ([]byte, error)

// Save encodes an image with libvips.
//
// An [Image] is saved from the image that libvips holds, with nothing
// copied. An [image.NRGBA] is handed to libvips as memory with nothing
// copied, which is what [Image.Unwrap] produces and what the scaler and the
// compositor of the ivimg package leave behind. Anything else is
// converted to one first, in a single pass. A 16 bit image is the exception:
// libvips only takes 8 bit pixels from memory, so that one goes through a
// lossless png rather than lose half its depth on the way in.
//
// The save happens here rather than in the caller because libvips is reading
// pixels it does not own: [vips.Image.Copy] hands back an image holding no
// reference to the Go buffer behind it, so nothing but the call below keeps
// that buffer from being collected while libvips is still reading it.
func Save(ctx context.Context, img image.Image, save SaveFunc) ([]byte, error) {
	if img == nil {
		return nil, errors.New("vips save: invalid image")
	}
	Init(ctx)
	if v, ok := img.(*Image); ok {
		ivctx.Logf(ctx, "vips save: from the vips image")
		buf, err := save(v.Vips())
		if err != nil {
			return nil, fmt.Errorf("vips save: %w", err)
		}
		return buf, nil
	}
	img = ivimg.Unwrap(img)
	start := time.Now()
	v, keep, how, err := importImage(img)
	if err != nil {
		return nil, fmt.Errorf("vips import: %w", err)
	}
	// the deferred call is what spans the save: libvips reads the pixels for
	// as long as it is encoding them
	defer runtime.KeepAlive(keep)
	ivctx.Logf(ctx, "vips import: %T via %s: %v", img, how, time.Since(start))
	buf, err := save(v)
	if err != nil {
		return nil, fmt.Errorf("vips save: %w", err)
	}
	return buf, nil
}

// importImage converts the Go image to a vips image, returning the value that
// has to stay alive for as long as libvips is reading it, and the route the
// conversion took.
func importImage(img image.Image) (*vips.Image, any, string, error) {
	if deepImage(img) {
		// a png carries its own pixels across, so nothing of ours outlives
		// this call
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			return nil, nil, "", err
		}
		v, err := vips.NewImageFromBuffer(buf.Bytes(), nil)
		return v, nil, "png", err
	}
	n := nrgba(img)
	b := n.Bounds()
	v, err := vips.NewImageFromMemory(n.Pix, b.Dx(), b.Dy(), 4)
	if err != nil {
		return nil, nil, "", err
	}
	defer v.Close()
	out, err := tag(v)
	if err != nil {
		return nil, nil, "", err
	}
	return out, n, "memory", nil
}

// tag returns the image with its bands named as sRGB.
//
// libvips takes raw memory as untagged bands -- multiband, in its terms -- and
// leaves the savers to guess what the numbers mean. Most of them guess right;
// the jxl one refuses outright with a JxlEncoderSetBasicInfo error, and none
// of them should have to guess. A copy is what sets the interpretation, and it
// moves no pixels: every other field is carried across as it was.
func tag(v *vips.Image) (*vips.Image, error) {
	out, err := v.Copy(&vips.CopyOptions{
		Width:          v.Width(),
		Height:         v.Height(),
		Bands:          v.Bands(),
		Format:         v.BandFormat(),
		Coding:         v.Coding(),
		Interpretation: vips.InterpretationSrgb,
		Xres:           v.ResX(),
		Yres:           v.ResY(),
		Xoffset:        v.OffsetX(),
		Yoffset:        v.OffsetY(),
	})
	if err != nil {
		return nil, fmt.Errorf("copy: %w", err)
	}
	return out, nil
}

// deepImage reports whether the image holds more than 8 bits a sample, and so
// has something to lose by being handed over as memory.
//
// [image.CMYK] is not one of these: its samples are 8 bits, and converting it
// to rgba costs a pass rather than a depth.
func deepImage(img image.Image) bool {
	switch img.(type) {
	case *image.NRGBA64, *image.RGBA64, *image.Gray16:
		return true
	}
	return false
}

// nrgba returns the image as a tightly packed [image.NRGBA], which is the one
// Go layout libvips can read as it stands: four 8 bit bands in order, rows
// flush against each other, and alpha unpremultiplied.
//
// An image that is already one is returned as it is. A sub image is not --
// its rows are spaced by the parent's width, which libvips would read as
// pixels.
func nrgba(img image.Image) *image.NRGBA {
	b := img.Bounds()
	if n, ok := img.(*image.NRGBA); ok && n.Stride == b.Dx()*4 && b.Min == (image.Point{}) {
		return n
	}
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
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
