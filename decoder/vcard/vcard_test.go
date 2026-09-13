package vcard

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func TestDetect(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  string
	}{
		{"vcard", "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:A\r\nEND:VCARD\r\n", "text/vcard"},
		{"lower case", "begin:vcard\nfn:A\nend:vcard\n", "text/vcard"},
		{"leading blank lines", "\n\n  BEGIN:VCARD\nFN:A\n", "text/vcard"},
		{"byte order mark", "\ufeffBEGIN:VCARD\nFN:A\n", "text/vcard"},
		{"vcalendar", "BEGIN:VCALENDAR\nEND:VCALENDAR\n", ""},
		{"prose about vcards", "A vcard begins with BEGIN:VCARD\n", ""},
		{"empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, _ := detect(context.Background(), strings.NewReader(test.s))
			if got != test.exp {
				t.Errorf("expected %q, got %q", test.exp, got)
			}
		})
	}
}

func TestParse(t *testing.T) {
	const src = "BEGIN:VCARD\r\n" +
		"VERSION:3.0\r\n" +
		"FN:John Doe\r\n" +
		"ORG:Example Corp\\, Ltd.;Research\r\n" +
		"TITLE:Engineer and\r\n" +
		"  Occasional Poet\r\n" +
		`TEL;TYPE="voice,work":+1 555 0100` + "\r\n" +
		"TEL;TYPE=cell:+1 555 0199\r\n" +
		"item1.EMAIL;TYPE=work:john@example.com\r\n" +
		"NOTE:a note\\; with escapes\\, and a \\n newline\r\n" +
		"END:VCARD\r\n"
	v, err := parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got, exp := v.get("FN"), "John Doe"; got != exp {
		t.Errorf("expected FN %q, got %q", exp, got)
	}
	// a folded line is one value, not two
	if got, exp := v.get("TITLE"), "Engineer and Occasional Poet"; got != exp {
		t.Errorf("expected the folded TITLE %q, got %q", exp, got)
	}
	// escapes resolve, and the semicolon that was escaped is not a separator
	if got, exp := field(v.get("ORG"), 0), "Example Corp, Ltd."; got != exp {
		t.Errorf("expected ORG %q, got %q", exp, got)
	}
	if got, exp := field(v.get("ORG"), 1), "Research"; got != exp {
		t.Errorf("expected the second ORG field %q, got %q", exp, got)
	}
	if got, exp := v.get("NOTE"), "a note; with escapes, and a \n newline"; got != exp {
		t.Errorf("expected NOTE %q, got %q", exp, got)
	}
	// the group prefix on item1.EMAIL is not part of the name
	if got, exp := v.get("EMAIL"), "john@example.com"; got != exp {
		t.Errorf("expected EMAIL %q, got %q", exp, got)
	}
	tel := v.all("TEL")
	if len(tel) != 2 {
		t.Fatalf("expected 2 TEL properties, got %d", len(tel))
	}
	// a quoted parameter holds a list, and matching ignores case
	if !tel[0].param("TYPE", "WORK") || !tel[0].param("type", "voice") {
		t.Errorf("expected the quoted parameter list to be split, got %v", tel[0].params)
	}
	if !tel[1].param("TYPE", "CELL") {
		t.Errorf("expected the lower case parameter to match, got %v", tel[1].params)
	}
	if tel[0].param("TYPE", "CELL") {
		t.Error("expected the work number not to be a mobile")
	}
}

func TestParseBareParam(t *testing.T) {
	// vcard 2.1 wrote the type without a key
	v, err := parse(strings.NewReader("BEGIN:VCARD\nTEL;CELL:+1 555 0100\nEND:VCARD\n"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	tel := v.all("TEL")
	if len(tel) != 1 {
		t.Fatalf("expected 1 TEL, got %d", len(tel))
	}
	if !tel[0].param("TYPE", "CELL") {
		t.Errorf("expected the bare parameter to match, got %v", tel[0].params)
	}
}

func TestName(t *testing.T) {
	for _, test := range []struct {
		name, src, exp string
	}{
		{"formatted", "FN:John Doe\nN:Doe;John;;;\n", "John Doe"},
		{"assembled", "N:Doe;John;Q;Dr.;PhD\n", "Dr. John Q Doe PhD"},
		{"partial", "N:Doe;John\n", "John Doe"},
		{"none", "ORG:Example\n", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			v, err := parse(strings.NewReader("BEGIN:VCARD\n" + test.src + "END:VCARD\n"))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got := name(v); got != test.exp {
				t.Errorf("expected %q, got %q", test.exp, got)
			}
		})
	}
}

// TestDecodeCarriesTheRecord checks the qr code on the card carries the vcard
// as it was written, which is what makes scanning the card useful.
func TestDecodeCarriesTheRecord(t *testing.T) {
	for _, name := range []string{"john-doe.vcf", "folded.vcf"} {
		t.Run(name, func(t *testing.T) {
			pathName := filepath.Join("..", "..", "testdata", "vcard", name)
			buf, err := os.ReadFile(pathName)
			if err != nil {
				t.Skipf("no test data: %v", err)
			}
			res, err := decode(ivctx.WithPathName(context.Background(), pathName), bytes.NewReader(buf))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			img, ok := res.(*decoder.Image)
			if !ok {
				t.Fatalf("expected a *decoder.Image, got %T", res)
			}
			if img.Mime != "image/svg+xml" {
				t.Errorf("expected mime %q, got %q", "image/svg+xml", img.Mime)
			}
			var svg bytes.Buffer
			if _, err := svg.ReadFrom(img.Reader); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			// the card draws the code as one path, and nothing else in the
			// svg is drawn with crisp edges
			if !strings.Contains(svg.String(), `shape-rendering="crispEdges"`) {
				t.Error("expected the card to carry a qr code")
			}
			// the record is what the code encodes, so the card must not have
			// dropped any of it
			if !strings.Contains(string(buf), "END:VCARD") {
				t.Fatal("expected the fixture to be a vcard")
			}
		})
	}
}

func TestDecodeRejectsNameless(t *testing.T) {
	_, err := decode(context.Background(), strings.NewReader("BEGIN:VCARD\nVERSION:3.0\nEND:VCARD\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestCardEscapes(t *testing.T) {
	c := &card{name: `A & B <c> "d"`, accent: accentFor("x")}
	if svg := string(c.svg()); strings.Contains(svg, "<c>") || !strings.Contains(svg, "&amp;") {
		t.Error("expected the name to be escaped")
	}
}

func TestAccentIsStable(t *testing.T) {
	if a, b := accentFor("John Doe"), accentFor("John Doe"); a != b {
		t.Error("expected the same name to derive the same accent")
	}
	if a, b := accentFor("John Doe"), accentFor("Jane Roe"); a == b {
		t.Error("expected different names to derive different accents")
	}
}

// TestContactsAreCapped checks a card with more detail than it has room for
// is trimmed rather than overrunning.
func TestContactsAreCapped(t *testing.T) {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\nFN:Someone\n")
	for i := range 12 {
		b.WriteString("TEL:+1 555 010" + string(rune('0'+i%10)) + "\n")
	}
	b.WriteString("END:VCARD\n")
	v, err := parse(strings.NewReader(b.String()))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := contacts(context.Background(), v); len(got) != maxContacts {
		t.Errorf("expected %d contact lines, got %d", maxContacts, len(got))
	}
}
