// Package rasterm supplies the terminal graphics encoder for iv.
//
// See: https://github.com/kenshaw/rasterm
package rasterm

import (
	"github.com/kenshaw/iv/encoder"
	"github.com/kenshaw/rasterm"
)

func init() {
	encoder.Register(
		"rasterm",
		encoder.Desc("Terminal graphics (Kitty, iTerm, Sixel)"),
		encoder.ImageEncoder(rasterm.Encode),
		encoder.Term(),
	)
}
