package ivimg

import (
	"image"
	"image/draw"
	"sync"

	"github.com/kenshaw/rasterm"
)

// Raw returns img in the form that the terminal encoder hands to rasterm for
// the protocol typ. The measures behind each choice are in the Performance
// section of the README.
//
// For the Kitty protocol:
//
//   - An image that holds its own encoding as a [rasterm.PNGImage] is
//     returned as it is, and rasterm sends the encoding. An image that libvips
//     holds encodes its PNG then.
//   - A JPEG offers its pixels as RGB, because Kitty cannot take a JPEG.
//   - A tight [image.NRGBA] is sent as RGBA, with nothing copied. It is what
//     the scaler and the compositor of this package leave.
//   - An [image.RGBA], which the renderers of svg, documents and lottie return,
//     is sent as RGBA with the alpha not premultiplied.
//   - Any other image is returned as it is, and rasterm encodes a PNG. A
//     paletted or a gray image makes a small PNG, and the PNG is faster than
//     the conversion to raw pixels.
//
// For the iTerm protocol, an opaque [JPEGEncoder] that does not hold the bytes
// of a file, such as an image that libvips holds, is sent as the JPEG that it
// encodes, at the quality of [rasterm.DefaultJPEGQuality]. rasterm sends a
// JPEG for any image that is not paletted, and for a 896x1080 photo the JPEG of
// libvips took 3.5 ms and 229 KiB, where its PNG took 33 ms and 1498 KiB. Any
// other image is returned as it is.
//
// For the other protocols, img is returned as it is.
//
// The pixels are converted only when the encoder asks for them, and the
// wrappers unwrap to img, so that an encoder that falls back to the pixels
// finds them.
func Raw(typ rasterm.TermType, img image.Image) image.Image {
	switch typ {
	case rasterm.Kitty:
		return kittyRaw(img)
	case rasterm.ITerm:
		if _, ok := pngOf(img); ok {
			return img
		}
		if j, ok := img.(JPEGEncoder); ok && Opaque(img) {
			return &encodedJPEG{Image: img, enc: j}
		}
	}
	return img
}

// kittyRaw returns img in the form that the Kitty encoder of rasterm sends
// fastest. See [Raw].
func kittyRaw(img image.Image) image.Image {
	switch v := img.(type) {
	case rasterm.PNGImage, rasterm.RGBImage, rasterm.RGBAImage:
		return img
	case rasterm.JPEGImage:
		// a JPEG decodes to a type with no alpha channel. A JPEGImage of any
		// other type is left to rasterm.
		if !opaqueType(Unwrap(img)) {
			return img
		}
		return &jpegRGB{rgb: &rgb{Image: img}, data: v.JPEG()}
	case *image.NRGBA:
		if tight(v.Rect, v.Stride, 4) {
			return rasterm.RGBA{Image: v, Data: v.Pix}
		}
		return &rgba{Image: img}
	case *image.RGBA:
		return &rgba{Image: img}
	}
	return img
}

// JPEGEncoder is an image that can encode itself as a JPEG, such as an image
// that libvips holds.
type JPEGEncoder interface {
	image.Image
	// EncodeJPEG returns the image encoded as a JPEG of the quality, from 1
	// to 100.
	EncodeJPEG(quality int) ([]byte, error)
}

// encodedJPEG is an image that encodes itself as a JPEG when it is first asked
// for its bytes. It satisfies the [rasterm.JPEGImage] interface, and has no
// PNG method, so that the iTerm encoder of rasterm sends the JPEG.
type encodedJPEG struct {
	image.Image

	enc  JPEGEncoder
	once sync.Once
	data []byte
}

// JPEG satisfies the [rasterm.JPEGImage] interface. If the encode fails, it
// returns nil, and rasterm encodes the pixels.
func (j *encodedJPEG) JPEG() []byte {
	j.once.Do(func() {
		j.data, _ = j.enc.EncodeJPEG(rasterm.DefaultJPEGQuality)
	})
	return j.data
}

// Unwrap returns the image that j wraps.
func (j *encodedJPEG) Unwrap() image.Image {
	return j.Image
}

// tight returns true when an image with the bounds r and the stride holds its
// pixels with n bytes for each, row after row from the origin, with nothing
// between the rows.
func tight(r image.Rectangle, stride, n int) bool {
	return r.Min == (image.Point{}) && stride == r.Dx()*n
}

// opaqueType returns true when the type of img has no alpha channel.
func opaqueType(img image.Image) bool {
	switch img.(type) {
	case *image.YCbCr, *image.CMYK, *image.Gray, *image.Gray16:
		return true
	}
	return false
}

// rgb is an image that converts its pixels to packed RGB when it is first
// asked for them. It satisfies the [rasterm.RGBImage] interface.
type rgb struct {
	image.Image

	once sync.Once
	data []byte
}

// RGB satisfies the [rasterm.RGBImage] interface.
func (r *rgb) RGB() []byte {
	r.once.Do(func() {
		src := toRGBA(r.Image)
		data := make([]byte, 0, len(src.Pix)/4*3)
		for i := 0; i < len(src.Pix); i += 4 {
			data = append(data, src.Pix[i], src.Pix[i+1], src.Pix[i+2])
		}
		r.data = data
	})
	return r.data
}

// Unwrap returns the image that r wraps.
func (r *rgb) Unwrap() image.Image {
	return r.Image
}

// jpegRGB is a JPEG that also converts its pixels to packed RGB when it is
// first asked for them. It satisfies the [rasterm.JPEGImage] and the
// [rasterm.RGBImage] interfaces.
type jpegRGB struct {
	*rgb

	data []byte
}

// JPEG satisfies the [rasterm.JPEGImage] interface.
func (j *jpegRGB) JPEG() []byte {
	return j.data
}

// rgba is an image that converts its pixels to packed RGBA, with the alpha not
// premultiplied, when it is first asked for them. It satisfies the
// [rasterm.RGBAImage] interface.
type rgba struct {
	image.Image

	once sync.Once
	data []byte
}

// RGBA satisfies the [rasterm.RGBAImage] interface.
func (r *rgba) RGBA() []byte {
	r.once.Do(func() {
		if n, ok := Unwrap(r.Image).(*image.NRGBA); ok {
			r.data = packNRGBA(n)
			return
		}
		r.data = nrgbaPix(toRGBA(r.Image))
	})
	return r.data
}

// nrgbaPix returns the pixels of src with the alpha not premultiplied, as an
// [image.NRGBA] holds them. [draw.Draw] has no fast path to an [image.NRGBA],
// so the conversion is done here, on the bytes.
func nrgbaPix(src *image.RGBA) []byte {
	pix := make([]byte, len(src.Pix))
	for i := 0; i < len(pix); i += 4 {
		switch a := src.Pix[i+3]; a {
		case 0xff:
			copy(pix[i:i+4], src.Pix[i:i+4])
		case 0:
		default:
			for j := range 3 {
				pix[i+j] = uint8(uint16(src.Pix[i+j]) * 0xff / uint16(a))
			}
			pix[i+3] = a
		}
	}
	return pix
}

// packNRGBA returns the pixels of src with nothing between the rows.
func packNRGBA(src *image.NRGBA) []byte {
	b := src.Bounds()
	pix := make([]byte, 0, b.Dx()*b.Dy()*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		i := src.PixOffset(b.Min.X, y)
		pix = append(pix, src.Pix[i:i+b.Dx()*4]...)
	}
	return pix
}

// Unwrap returns the image that r wraps.
func (r *rgba) Unwrap() image.Image {
	return r.Image
}

// toRGBA returns the pixels of img as a tight [image.RGBA]. [draw.Draw] has a
// fast path to it from the common types, a decoded JPEG among them.
func toRGBA(img image.Image) *image.RGBA {
	src := Unwrap(img)
	if r, ok := src.(*image.RGBA); ok && tight(r.Rect, r.Stride, 4) {
		return r
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}
