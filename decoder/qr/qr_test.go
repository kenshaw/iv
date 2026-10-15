package qr

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/iv/ivctx"
)

func TestMatch(t *testing.T) {
	for _, test := range []struct {
		s   string
		exp bool
	}{
		// the schemes named outright
		{"mailto:john@example.com", true},
		{"MAILTO:john@example.com", true},
		{"tel:+14155552671", true},
		{"sms:+14155552671?body=hi", true},
		{"geo:37.786971,-122.399677;u=35", true},
		{"xmpp:romeo@montague.net", true},
		{"sip:alice@atlanta.example.com", true},
		{"matrix:r/room:example.org", true},
		{"magnet:?xt=urn:btih:abc", true},
		{"bitcoin:bc1qexample?amount=0.001", true},
		{"ethereum:0xabc@1?value=1e18", true},
		{"lightning:lnbc1example", true},
		{"otpauth://totp/Example:alice?secret=ABC", true},
		{"DPP:C:81/1;M:582b7c1e8a3f;;", true},
		{"WIFI:S:ssid;T:WPA;P:secret;;", true},
		{"ftp://ftp.example.com/pub/x.gz", true},
		// any other scheme, as long as it has an authority
		{"ssh://git@example.com/x.git", true},
		{"irc://irc.example.org/chan", true},
		// the schemes that name a document to fetch, which iv renders
		{"http://example.com", false},
		{"https://example.com/a.png", false},
		{"data:image/png;base64,AAAA", false},
		{"file:///tmp/x.png", false},
		// not uris
		{"nope:whatever", false},
		{"not a url at all", false},
		{"", false},
		{"/some/path/file.png", false},
		{`C:\Users\someone\photo.png`, false},
		{"example.com/a.png", false},
		{":no-scheme", false},
	} {
		if got := match(test.s); got != test.exp {
			t.Errorf("match(%q) = %v, want %v", test.s, got, test.exp)
		}
	}
}

// TestDecodeColors checks that --fg, --bg and --border reach the code, and
// that a transparent background draws white.
func TestDecodeColors(t *testing.T) {
	navy, yellow, none := colors.New(0, 0, 128, 255), colors.New(255, 255, 0, 255), colors.New(0, 0, 0, 0)
	for _, test := range []struct {
		name   string
		fg, bg *colors.Color
		border uint
		expBg  color.RGBA
		expFg  color.RGBA
	}{
		{"default", nil, &none, 40, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255}},
		{"colors", &navy, &yellow, 10, color.RGBA{255, 255, 0, 255}, color.RGBA{0, 0, 128, 255}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := ivctx.New()
			c.Fg, c.Bg, c.Border = test.fg, test.bg, test.border
			res, err := decode(ivctx.WithConfig(context.Background(), c), "WIFI:S:x;;")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			img := res.(image.Image)
			// the corner is the border, and a finder pattern starts right
			// inside it
			b := int(test.border)
			if got := color.RGBAModel.Convert(img.At(0, 0)); got != test.expBg {
				t.Errorf("expected the background %v in the border, got %v", test.expBg, got)
			}
			if got := color.RGBAModel.Convert(img.At(b, b)); got != test.expFg {
				t.Errorf("expected the foreground %v at the edge of the border, got %v", test.expFg, got)
			}
			if got := color.RGBAModel.Convert(img.At(b-1, b-1)); got != test.expBg {
				t.Errorf("expected the border to be %d pixels, got a module at %d", b, b-1)
			}
		})
	}
}
