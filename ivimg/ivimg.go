// Package ivimg holds the steps that every decoded image goes through in iv,
// between the decoder and the encoder.
//
// An image moves through the pipeline as an [image.Image], and the value can
// carry more than its pixels:
//
//   - The bytes of the file that it was decoded from, as a [rasterm.PNG] or a
//     [rasterm.JPEG]. [DecodePNG] and [DecodeJPEG] keep them.
//   - The image that libvips holds, which can resize itself, composite itself
//     over a background, and encode itself as a PNG only when an encoder asks
//     for one. See the vips package of iv.
//
// [Prepare] scales an image to the size it is displayed at and composites it
// over the background. If a step does not change the pixels, it returns the
// image as it was, so the bytes of the file stay with it. If a step changes
// the pixels, the result holds no bytes, so an encoder can never send bytes
// that no longer match the pixels.
//
// An encoder decides last how to use what the image holds. The terminal
// encoder sends the bytes of the file when there are some, and raw pixels
// otherwise (see [Raw]). A file encoder writes the bytes as they are when
// they are in its format.
package ivimg

import (
	"context"
	"image"
	"io"
)

// Resizer is an image that can resize itself.
type Resizer interface {
	image.Image
	// Resize returns the image scaled to w x h. The image it is called on
	// does not change.
	Resize(ctx context.Context, w, h int) (image.Image, error)
}

// Flattener is an image that can composite itself over a background color.
type Flattener interface {
	image.Image
	// Flatten returns the image composited over the opaque color bg. The
	// image it is called on does not change.
	Flatten(ctx context.Context, r, g, b uint8) (image.Image, error)
}

// Prepare returns img scaled to the size that the configuration on the
// context displays it at, and composited over the configured background. The
// mime type is the type that img was decoded as.
//
// The decoders of stills and the producers of animation frames use this
// function alike.
func Prepare(ctx context.Context, mime string, img image.Image) (image.Image, error) {
	img, err := Fit(ctx, img)
	if err != nil {
		return nil, err
	}
	return AddBackground(ctx, mime, img)
}

// Unwrap returns the image under each Unwrap method of img, so that a type
// switch on the result finds the concrete type of the pixels. An encoder that
// encodes the pixels itself uses it, because the encoders of the standard
// library are fast only for their own concrete types.
func Unwrap(img image.Image) image.Image {
	for {
		u, ok := img.(interface{ Unwrap() image.Image })
		if !ok {
			return img
		}
		img = u.Unwrap()
	}
}

// Err returns the error that img kept from work that it did after it was
// decoded, such as an image that libvips decodes only when its pixels are
// asked for. It returns nil for any other image. An encoder entry calls it
// after an encode, because an image method such as At has no error to
// return.
func Err(img image.Image) error {
	if e, ok := img.(interface{ Err() error }); ok {
		return e.Err()
	}
	return nil
}

// Close releases what img holds outside of Go memory, such as an image that
// libvips holds. It does nothing for any other image.
func Close(img image.Image) error {
	if c, ok := img.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
