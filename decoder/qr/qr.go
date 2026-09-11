// Package qr supplies a QR code decoder for iv, rendering WIFI: codes as
// images.
//
// See: https://github.com/skip2/go-qrcode
package qr

import (
	"context"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/skip2/go-qrcode"
)

// prefix is the string prefix claimed by the qr decoder.
const prefix = "WIFI:"

func init() {
	decoder.Register(
		"qr",
		decoder.Desc("WIFI: QR codes"),
		decoder.StringMatcher(func(s string) bool {
			return strings.HasPrefix(strings.ToUpper(s), prefix)
		}),
		decoder.StringDecoder(decode),
	)
}

// decode renders a WIFI: code as a QR code image.
func decode(ctx context.Context, s string) (any, error) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	c := ivctx.Get(ctx)
	q.ForegroundColor, q.BackgroundColor, q.DisableBorder = c.Fg, c.Bg, true
	return ivctx.AddBorder(ctx, q.Image(-10)), nil
}
