package ivimg

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"sync"
)

// file is an image decoded from the bytes of a PNG or a JPEG file. It reads
// the size and the color model from the header when it is made, and decodes
// the pixels only when something asks for them: a scaler, a compositor, or
// an encoder that encodes the pixels itself. A terminal encoder that sends
// the bytes of the file never decodes them.
//
// A decode error that arrives then is kept for [Err], and the encoder entries
// of iv return it after an encode.
type file struct {
	data   []byte
	config image.Config
	decode func(io.Reader) (image.Image, error)

	once sync.Once
	pix  image.Image

	mu  sync.Mutex
	err error
}

// newFile returns a file for the bytes of buf. It reads the header with
// decodeConfig, and decodes the pixels with decode when they are first asked
// for.
func newFile(buf []byte, decodeConfig func(io.Reader) (image.Config, error), decode func(io.Reader) (image.Image, error)) (*file, error) {
	cfg, err := decodeConfig(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	return &file{
		data:   buf,
		config: cfg,
		decode: decode,
	}, nil
}

// Bounds satisfies the [image.Image] interface. It is read from the header,
// so it does not decode the pixels.
func (f *file) Bounds() image.Rectangle {
	return image.Rect(0, 0, f.config.Width, f.config.Height)
}

// ColorModel satisfies the [image.Image] interface. It is read from the
// header, so it does not decode the pixels.
func (f *file) ColorModel() color.Model {
	return f.config.ColorModel
}

// At satisfies the [image.Image] interface. It decodes the pixels the first
// time that it is called.
func (f *file) At(x, y int) color.Color {
	return f.Unwrap().At(x, y)
}

// Unwrap returns the decoded pixels. It decodes them the first time that it is
// called. If the decode fails, it keeps the error for [file.Err] and returns
// a transparent image of the same size.
func (f *file) Unwrap() image.Image {
	f.once.Do(func() {
		pix, err := f.decode(bytes.NewReader(f.data))
		if err != nil {
			f.mu.Lock()
			f.err = err
			f.mu.Unlock()
			pix = image.NewNRGBA(f.Bounds())
		}
		f.pix = pix
	})
	return f.pix
}

// Err returns the error of the decode of the pixels, or nil when they were
// not decoded or decoded without one.
func (f *file) Err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}

// pngFile is a [file] that holds a PNG. It satisfies the [rasterm.PNGImage]
// interface.
type pngFile struct {
	*file
}

// PNG satisfies the [rasterm.PNGImage] interface.
func (p pngFile) PNG() []byte {
	return p.data
}

// jpegFile is a [file] that holds a JPEG. It satisfies the [rasterm.JPEGImage]
// interface.
type jpegFile struct {
	*file
}

// JPEG satisfies the [rasterm.JPEGImage] interface.
func (j jpegFile) JPEG() []byte {
	return j.data
}

// orient returns img turned as the EXIF orientation o describes, from 1 to 8,
// so that it reads the right way up. An orientation of 1, or one out of
// range, returns img as it is.
//
// Orientations 5 to 8 swap the width and the height.
func orient(img image.Image, o int) image.Image {
	if o < 2 || o > 8 {
		return img
	}
	src := toRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			var dx, dy int
			switch o {
			case 2: // mirrored
				dx, dy = w-1-x, y
			case 3: // turned half way
				dx, dy = w-1-x, h-1-y
			case 4: // flipped top to bottom
				dx, dy = x, h-1-y
			case 5: // mirrored across the diagonal from the top left
				dx, dy = y, x
			case 6: // turned a quarter clockwise
				dx, dy = h-1-y, x
			case 7: // mirrored across the diagonal from the top right
				dx, dy = h-1-y, w-1-x
			case 8: // turned a quarter counterclockwise
				dx, dy = y, w-1-x
			}
			copy(dst.Pix[dst.PixOffset(dx, dy):][:4], src.Pix[src.PixOffset(x, y):][:4])
		}
	}
	return dst
}

// opaqueFile returns whether img, an image that holds the bytes of a PNG or a
// JPEG file, has no transparent pixels, as far as the bytes tell. The second
// result is false when img holds no such bytes.
func opaqueFile(img image.Image) (bool, bool) {
	if data, ok := pngOf(img); ok {
		return pngOpaque(data), true
	}
	switch img.(type) {
	case jpegFile:
		// a CMYK JPEG keeps no bytes (see DecodeJPEG), so a jpegFile is
		// YCbCr or gray, and neither has an alpha channel
		return true, true
	}
	return false, false
}
