// Package rasterm supplies the terminal graphics encoder for iv.
//
// See: https://github.com/kenshaw/rasterm
package rasterm

import (
	"context"
	"image"
	"io"

	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/kenshaw/iv/ivimg"
	"github.com/kenshaw/rasterm"
)

func init() {
	encoder.Register(
		"rasterm",
		encoder.Desc("Terminal graphics (Kitty, iTerm, Sixel)"),
		encoder.Encoder(encode),
		encoder.Term(),
	)
}

// encode encodes the image with the protocol of the terminal in the
// configuration, which [rasterm.Detect] found (D42). The image goes through
// [ivimg.Raw] first, so that rasterm sends the bytes that the image holds,
// or a form of it that is fast to send, in place of an encode.
func encode(ctx context.Context, w io.Writer, img image.Image) error {
	t := ivctx.Get(ctx).Terminal
	enc := t.Encoder()
	if enc == nil {
		return rasterm.ErrTermGraphicsNotAvailable
	}
	return enc.Encode(w, ivimg.Raw(t.Protocol, img))
}
