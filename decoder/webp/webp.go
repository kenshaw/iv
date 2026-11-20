// Package webp supplies a webp decoder for iv, in pure Go. An animated webp
// is played with --animate, and --page selects one of its frames (D61).
//
// See: https://github.com/gen2brain/vpx
package webp

import (
	"bytes"
	"context"

	"github.com/gen2brain/vpx/webp"
	"github.com/kenshaw/iv/decoder"
)

func init() {
	decoder.RegisterBuiltin(
		"webp",
		decoder.Desc("Web Picture Format"),
		decoder.Extension("webp"),
		decoder.MimeType("image/webp"),
		decoder.BytesDecoderContext(decode),
	)
}

// decode decodes the webp in buf: its first frame, or all of its frames when
// they are asked for. vpx composites each frame of an animation.
func decode(ctx context.Context, buf []byte) (any, error) {
	if !decoder.Animated(ctx) {
		return webp.Decode(bytes.NewReader(buf))
	}
	w, err := webp.DecodeAll(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	return decoder.Frames(ctx, w.Image, decoder.Milliseconds(w.Delay))
}
