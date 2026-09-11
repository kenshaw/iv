package decoder

import (
	"context"
	"image"
	"io"
)

// Option is a decoder option.
type Option func(*Entry)

// Desc is a decoder option to set the human readable description.
func Desc(desc string) Option {
	return func(d *Entry) {
		d.Desc = desc
	}
}

// Extension is a decoder option to add file extensions, without a leading dot.
func Extension(exts ...string) Option {
	return func(d *Entry) {
		d.Exts = append(d.Exts, exts...)
	}
}

// MimeType is a decoder option to add mime types. A type may end in /* to
// match every subtype.
func MimeType(mimes ...string) Option {
	return func(d *Entry) {
		d.Mimes = append(d.Mimes, mimes...)
	}
}

// MimeTypeExtensionMatch is a decoder option adding a matcher for mime
// type/extension pairs, where neither alone is enough to identify the format.
// Panics when not passed an even number of arguments.
func MimeTypeExtensionMatch(pairs ...string) Option {
	if len(pairs)%2 != 0 {
		panic("decoder: MimeTypeExtensionMatch expects mime/extension pairs")
	}
	return func(d *Entry) {
		for i := 0; i < len(pairs); i += 2 {
			mime, ext := pairs[i], pairs[i+1]
			d.Exts = append(d.Exts, ext)
			d.matchers = append(d.matchers, func(_ context.Context, m, e string) bool {
				return e == ext && MimeMatch(mime, m)
			})
		}
	}
}

// Matcher is a decoder option adding an arbitrary mime type/extension matcher.
func Matcher(f func(mime, ext string) bool) Option {
	return func(d *Entry) {
		d.matchers = append(d.matchers, func(_ context.Context, mime, ext string) bool {
			return f(mime, ext)
		})
	}
}

// MatcherContext is a decoder option adding a context aware matcher.
func MatcherContext(f MatchFunc) Option {
	return func(d *Entry) {
		d.matchers = append(d.matchers, f)
	}
}

// StringMatcher is a decoder option marking the decoder as a string decoder
// and adding a matcher for bare command line arguments.
func StringMatcher(f func(s string) bool) Option {
	return func(d *Entry) {
		d.IsString = true
		d.strMatchers = append(d.strMatchers, func(_ context.Context, s string) bool {
			return f(s)
		})
	}
}

// MimeDetector is a decoder option to add a content sniffer, used when the
// standard mime detection cannot identify the content. The func must not
// consume the reader.
func MimeDetector(f DetectFunc) Option {
	return func(d *Entry) {
		d.detect = f
	}
}

// Decoder is a decoder option to set the decoding func.
func Decoder(f DecodeFunc) Option {
	return func(d *Entry) {
		d.decode = f
	}
}

// ImageDecoder is a decoder option setting the decoding func from a plain
// image decoder, such as [png.Decode].
func ImageDecoder(f func(io.Reader) (image.Image, error)) Option {
	return Decoder(func(_ context.Context, r io.Reader) (any, error) {
		return f(r)
	})
}

// ImagesDecoder is a decoder option setting the decoding func from a
// multi-image decoder, such as [ico.DecodeAll]. The configured page selects
// which image is displayed.
func ImagesDecoder(f func(io.Reader) ([]image.Image, error)) Option {
	return Decoder(func(_ context.Context, r io.Reader) (any, error) {
		return f(r)
	})
}

// StringDecoder is a decoder option marking the decoder as a string decoder
// and setting its decoding func.
func StringDecoder(f StringDecodeFunc) Option {
	return func(d *Entry) {
		d.IsString = true
		d.decodeString = f
	}
}

// Init is a decoder option to set the lazy initialization and close funcs. The
// value returned by the init func is available to the decoder through [State].
func Init(initFunc func(context.Context) (any, error), closeFunc func(context.Context) error) Option {
	return func(d *Entry) {
		d.initFunc, d.closeFunc = initFunc, closeFunc
	}
}

// Before is a decoder option declaring that the decoder must be tried before
// the named decoders.
func Before(names ...string) Option {
	return func(d *Entry) {
		d.before = append(d.before, names...)
	}
}

// After is a decoder option declaring that the decoder must be tried after the
// named decoders.
func After(names ...string) Option {
	return func(d *Entry) {
		d.after = append(d.after, names...)
	}
}

// Builtin is a decoder option marking the decoder's formats as decodable
// through Go's [image.Decode] registry, so that container formats can render
// them directly.
func Builtin() Option {
	return func(d *Entry) {
		d.Builtin = true
	}
}

// Fallback is a decoder option marking the decoder as a last resort, tried
// only after every other decoder has declined or failed.
func Fallback() Option {
	return func(d *Entry) {
		d.Fallback = true
	}
}

// builtin marks the decoder as decoding through Go's [image.Decode] registry.
func builtin() Option {
	return func(d *Entry) {
		d.Builtin, d.decode = true, builtinDecode
	}
}
