package qr

import "testing"

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
