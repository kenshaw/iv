// Package rasterm supplies the terminal graphics encoder for iv.
//
// See: https://github.com/kenshaw/rasterm
package rasterm

import (
	"context"
	"image"
	"io"

	"github.com/kenshaw/iv/encoder"
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

// encode encodes the image with the protocol that rasterm selects for the
// terminal. The image goes through [ivimg.Raw] first, so that rasterm sends
// the bytes that the image holds, or a form of it that is fast to send, in
// place of an encode.
func encode(_ context.Context, w io.Writer, img image.Image) error {
	return rasterm.Encode(w, ivimg.Raw(protocol(), img))
}

// protocol returns the protocol that the default encoder of rasterm uses,
// when it is Kitty or iTerm, the two protocols that [ivimg.Raw] chooses a
// form for. It returns [rasterm.Default] for any other protocol.
//
// The default encoder of rasterm tries Kitty, then iTerm, then Sixel, and
// uses the first one that is available. Sixel is not asked here, because its
// test writes a query to the terminal.
func protocol() rasterm.TermType {
	switch {
	case rasterm.Kitty.Available():
		return rasterm.Kitty
	case rasterm.ITerm.Available():
		return rasterm.ITerm
	}
	return rasterm.Default
}
