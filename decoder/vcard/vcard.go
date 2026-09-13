// Package vcard supplies a vcard decoder for iv, drawing a contact as a
// business card with a scannable copy of the record on it.
package vcard

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.Register(
		"vcard",
		decoder.Desc("vCard contacts"),
		decoder.Extension("vcf", "vcard"),
		decoder.MimeType("text/vcard", "text/x-vcard", "text/directory"),
		decoder.MimeDetector(detect),
		decoder.Decoder(decode),
	)
}

// peek is how many leading bytes are sniffed for a vcard.
const peek = 512

// detect sniffs the reader for a vcard. libmagic types these already, but a
// card written without a trailing newline or with an unusual version line
// does not always reach it, and the opening is unmistakable.
func detect(_ context.Context, r io.Reader) (string, error) {
	buf := make([]byte, peek)
	n, err := io.ReadFull(r, buf)
	if n == 0 && err != nil {
		return "", err
	}
	s := strings.ToUpper(strings.TrimLeft(string(buf[:n]), " \t\r\n\ufeff"))
	if strings.HasPrefix(s, "BEGIN:VCARD") {
		return "text/vcard", nil
	}
	return "", nil
}

// decode renders the contact as a business card, handing it back to the
// pipeline as an svg.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	v, err := parse(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("vcard parse: %w", err)
	}
	c := build(ctx, v, string(buf))
	if c.name == "" {
		return nil, fmt.Errorf("vcard: no name")
	}
	ivctx.Logf(ctx, "vcard: %q %q %q, %d contact lines", c.name, c.title, c.org, len(c.contacts))
	return decoder.NewBytes("image/svg+xml", c.svg()).WithExt("svg"), nil
}

// build assembles the card from the parsed record.
func build(ctx context.Context, v *vcard, raw string) *card {
	c := &card{
		name:  name(v),
		title: v.get("TITLE"),
		org:   field(v.get("ORG"), 0),
		raw:   strings.TrimSpace(raw),
	}
	c.accent = accentFor(c.name + c.org)
	c.contacts = contacts(ctx, v)
	return c
}

// name returns the name to print: the formatted one the card supplies, or one
// assembled from its parts when it does not.
func name(v *vcard) string {
	if s := strings.TrimSpace(v.get("FN")); s != "" {
		return s
	}
	// N is family;given;additional;prefix;suffix
	n := fields(v.get("N"))
	for len(n) < 5 {
		n = append(n, "")
	}
	return join(" ", n[3], n[1], n[2], n[0], n[4])
}

// maxContacts is how many contact lines the card has room for.
const maxContacts = 5

// contacts returns the contact lines, in the order a card reads best: how to
// reach the person first, then where they are.
func contacts(ctx context.Context, v *vcard) []contact {
	var out []contact
	add := func(icon, value, label string) {
		if value = strings.TrimSpace(value); value != "" && len(out) < maxContacts {
			out = append(out, contact{icon: icon, value: value, label: label})
		}
	}
	for _, p := range v.all("TEL") {
		icon, label := iconPhone, "phone"
		if p.param("TYPE", "CELL") || p.param("TYPE", "MOBILE") {
			icon, label = iconMobile, "mobile"
		}
		add(icon, p.value, label)
	}
	for _, p := range v.all("EMAIL") {
		add(iconMail, p.value, "email")
	}
	for _, p := range v.all("URL") {
		add(iconGlobe, p.value, "url")
	}
	for _, p := range v.all("ADR") {
		// street;locality;region;postcode -- the parts that name a place,
		// without the post office box and extended address nobody prints
		f := fields(p.value)
		for len(f) < 7 {
			f = append(f, "")
		}
		add(iconPin, join(", ", f[2], f[3], f[4], f[5]), "address")
	}
	if n := len(out); n == maxContacts {
		ivctx.Logf(ctx, "vcard: showing %d contact lines", n)
	}
	return out
}

// join joins the non-empty parts with sep.
func join(sep string, parts ...string) string {
	var out []string
	for _, s := range parts {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, sep)
}

// Icons, drawn as strokes in a 16 unit box on the text's baseline.
const (
	iconPhone  = `<path d="M2.4 3.2a1.2 1.2 0 0 1 1.2-1.2h2a1 1 0 0 1 1 .8l.5 2a1 1 0 0 1-.3 1l-1 .9a10 10 0 0 0 4.2 4.2l.9-1a1 1 0 0 1 1-.3l2 .5a1 1 0 0 1 .8 1v2a1.2 1.2 0 0 1-1.2 1.2A12.6 12.6 0 0 1 2.4 3.2z"/>`
	iconMobile = `<rect x="4.4" y="1.6" width="7.2" height="12.8" rx="1.4"/><path d="M7.2 12.4h1.6"/>`
	iconMail   = `<rect x="1.6" y="3.2" width="12.8" height="9.6" rx="1.4"/><path d="M1.9 4.2 8 8.6l6.1-4.4"/>`
	iconGlobe  = `<circle cx="8" cy="8" r="6.4"/><path d="M1.6 8h12.8M8 1.6a10 10 0 0 1 0 12.8 10 10 0 0 1 0-12.8z"/>`
	iconPin    = `<path d="M8 14.4S13 9.9 13 6.6a5 5 0 0 0-10 0c0 3.3 5 7.8 5 7.8z"/><circle cx="8" cy="6.4" r="1.9"/>`
)
