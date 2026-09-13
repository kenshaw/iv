package vcard

import (
	"bufio"
	"io"
	"strings"
)

// property is one vcard property: a name, its parameters, and its value.
type property struct {
	name   string
	params map[string][]string
	value  string
}

// param returns whether the property carries the parameter value, matched
// without regard to case. A bare parameter -- TEL;CELL: rather than
// TEL;TYPE=CELL: -- counts too, since vcard 2.1 wrote them that way.
func (p property) param(key, value string) bool {
	for k, vs := range p.params {
		if k != "" && !strings.EqualFold(k, key) {
			continue
		}
		for _, v := range vs {
			if strings.EqualFold(v, value) {
				return true
			}
		}
	}
	return false
}

// vcard is a parsed vcard.
type vcard struct {
	props []property
}

// get returns the value of the first property with the name, empty when there
// is none.
func (v *vcard) get(name string) string {
	for _, p := range v.props {
		if strings.EqualFold(p.name, name) && p.value != "" {
			return p.value
		}
	}
	return ""
}

// all returns every property with the name.
func (v *vcard) all(name string) []property {
	var out []property
	for _, p := range v.props {
		if strings.EqualFold(p.name, name) && p.value != "" {
			out = append(out, p)
		}
	}
	return out
}

// parse reads a vcard.
//
// Only the structure is parsed here, not the vocabulary: every property is
// kept as it was written, and what the card draws decides which of them it
// has a use for.
func parse(r io.Reader) (*vcard, error) {
	v := new(vcard)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)
	var line string
	flush := func() {
		if p, ok := parseLine(line); ok {
			v.props = append(v.props, p)
		}
		line = ""
	}
	for sc.Scan() {
		s := strings.TrimRight(sc.Text(), "\r")
		// a line beginning with a space or tab continues the one before it,
		// which is how a long value is written
		if strings.HasPrefix(s, " ") || strings.HasPrefix(s, "\t") {
			line += s[1:]
			continue
		}
		flush()
		line = s
	}
	flush()
	return v, sc.Err()
}

// maxLine bounds a single folded property, so a malformed card cannot make
// the scanner allocate without limit.
const maxLine = 1 << 20

// parseLine parses one unfolded property line.
func parseLine(s string) (property, bool) {
	if s = strings.TrimSpace(s); s == "" {
		return property{}, false
	}
	head, value, ok := strings.Cut(s, ":")
	if !ok {
		return property{}, false
	}
	parts := splitEscaped(head, ';')
	if len(parts) == 0 || parts[0] == "" {
		return property{}, false
	}
	p := property{
		name:   strings.ToUpper(strings.TrimSpace(parts[0])),
		params: make(map[string][]string),
		value:  unescape(value),
	}
	// a property name may carry a group prefix -- item1.TEL -- which nothing
	// here has a use for
	if _, after, ok := strings.Cut(p.name, "."); ok {
		p.name = after
	}
	for _, s := range parts[1:] {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			// a bare parameter, keyed by the empty string
			k, v = "", s
		}
		k = strings.ToUpper(strings.TrimSpace(k))
		for _, v := range splitEscaped(strings.Trim(strings.TrimSpace(v), `"`), ',') {
			p.params[k] = append(p.params[k], v)
		}
	}
	return p, true
}

// splitEscaped splits on sep, honoring the backslash escapes a value may
// carry.
func splitEscaped(s string, sep byte) []string {
	var out []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			b.WriteByte(s[i])
			i++
			b.WriteByte(s[i])
		case s[i] == sep:
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	return append(out, b.String())
}

// unescape resolves the backslash escapes a vcard value carries.
var unescape = strings.NewReplacer(
	`\n`, "\n",
	`\N`, "\n",
	`\,`, ",",
	`\;`, ";",
	`\\`, `\`,
).Replace

// fields splits a structured value -- N, ADR -- into its components.
func fields(s string) []string {
	out := splitEscaped(s, ';')
	for i, v := range out {
		out[i] = strings.TrimSpace(unescape(v))
	}
	return out
}

// field returns the nth component of a structured value, empty when there is
// no such component.
func field(s string, n int) string {
	if f := fields(s); n < len(f) {
		return f[n]
	}
	return ""
}
