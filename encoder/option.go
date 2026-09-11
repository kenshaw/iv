package encoder

import (
	"context"
	"image"
	"io"
)

// Option is an encoder option.
type Option func(*Entry)

// Desc is an encoder option to set the human readable description.
func Desc(desc string) Option {
	return func(e *Entry) {
		e.Desc = desc
	}
}

// Encoder is an encoder option to set the encoding func.
func Encoder(f EncodeFunc) Option {
	return func(e *Entry) {
		e.encode = f
	}
}

// ImageEncoder is an encoder option setting the encoding func from a plain
// image encoder, such as [png.Encode].
func ImageEncoder(f func(io.Writer, image.Image) error) Option {
	return Encoder(func(_ context.Context, w io.Writer, img image.Image) error {
		return f(w, img)
	})
}

// Extension is an encoder option to set the output file extension, without a
// leading dot.
func Extension(ext string) Option {
	return func(e *Entry) {
		e.Ext = ext
	}
}

// Term is an encoder option marking the encoder as writing terminal graphics.
func Term() Option {
	return func(e *Entry) {
		e.Term, e.Ext = true, ""
	}
}
