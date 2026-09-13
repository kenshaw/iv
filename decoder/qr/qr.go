// Package qr supplies a QR code decoder for iv, rendering the uris that name
// something to act on rather than something to fetch.
//
// See: https://github.com/skip2/go-qrcode
package qr

import (
	"context"
	"image/color"
	"regexp"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/skip2/go-qrcode"
)

// schemes are the uri schemes rendered as a qr code. All of them name
// something to be acted on -- a call placed, a payment made, an account
// enrolled -- rather than a document to retrieve, which is what makes a qr
// code the useful rendering of them.
var schemes = map[string]bool{
	"bitcoin":   true, // BIP-21
	"dpp":       true, // Wi-Fi Easy Connect
	"ethereum":  true, // EIP-681
	"ftp":       true,
	"ftps":      true,
	"geo":       true, // RFC 5870
	"lightning": true,
	"magnet":    true,
	"mailto":    true, // RFC 6068
	"matrix":    true,
	"otpauth":   true, // Key URI Format
	"sip":       true,
	"sips":      true,
	"sms":       true, // RFC 5724
	"smsto":     true,
	"tel":       true, // RFC 3966
	"wifi":      true,
	"xmpp":      true, // RFC 5122
}

// fetched are the schemes that name a document to retrieve, which iv renders
// rather than encodes. They are excluded so that a page is still a page.
var fetched = map[string]bool{
	"http":  true,
	"https": true,
	"data":  true,
	"file":  true,
}

// schemeRE matches the scheme of a uri.
var schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.\-]*:`)

func init() {
	decoder.Register(
		"qr",
		decoder.Desc("QR codes for mailto:, tel:, geo:, otpauth: and other uris"),
		decoder.StringMatcher(func(s string) bool {
			return match(s)
		}),
		decoder.StringDecoder(decode),
	)
}

// match reports whether the string is a uri to render as a qr code: one of
// the schemes above, or any other scheme with an authority -- the `://` that
// says a bare word followed by a colon was meant as a uri at all.
func match(s string) bool {
	prefix := schemeRE.FindString(s)
	if prefix == "" {
		return false
	}
	scheme := strings.ToLower(strings.TrimSuffix(prefix, ":"))
	switch {
	case fetched[scheme]:
		return false
	case schemes[scheme]:
		return true
	}
	return strings.HasPrefix(s[len(prefix):], "//")
}

// decode renders the uri as a QR code image.
//
// The code is always black on white, with the quiet zone the standard asks
// for, whatever --fg and --bg say. Those are for looking at an image; a qr
// code is for a camera, and one drawn in the default dimgray on a transparent
// ground does not scan at all -- too little contrast, and no light field to
// find the code against.
func decode(ctx context.Context, s string) (any, error) {
	q, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	q.ForegroundColor, q.BackgroundColor = color.Black, color.White
	ivctx.Logf(ctx, "qr: %d bytes, version %d", len(s), q.VersionNumber)
	return q.Image(-10), nil
}
