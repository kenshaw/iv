package heif

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/gen2brain/h265/heic"
	"howett.net/plist"
)

// TestParseDesktop checks the wallpaper metadata of the two dynamic
// wallpapers of testdata/heic, and what it says about some images.
func TestParseDesktop(t *testing.T) {
	tests := []struct {
		name        string
		light, dark int
		times       int
		describe    map[int]string
	}{
		{"dome.heic", 0, 1, 2, map[int]string{
			0: "the light appearance, from 00:00",
			1: "the dark appearance, from 12:00",
		}},
		{"cyberpunk.heic", 3, 9, 12, map[int]string{
			0:  "from 12:00",
			3:  "the light appearance, from 05:00",
			9:  "the dark appearance, from 19:00",
			12: "not named by the metadata",
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			xmp, err := heic.RawXMP(bytes.NewReader(readFile(t, test.name)))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			d, err := ParseDesktop(xmp)
			switch {
			case err != nil:
				t.Fatalf("expected no error, got: %v", err)
			case d == nil:
				t.Fatal("expected wallpaper metadata")
			}
			if d.Kind != "h24" || d.Light != test.light || d.Dark != test.dark || len(d.Times) != test.times {
				t.Errorf("expected h24, light %d, dark %d and %d times, got %s, %d, %d and %d",
					test.light, test.dark, test.times, d.Kind, d.Light, d.Dark, len(d.Times))
			}
			for index, exp := range test.describe {
				if got := d.Describe(index); got != exp {
					t.Errorf("image %d: expected %q, got %q", index, exp, got)
				}
			}
		})
	}
}

// TestParseDesktopKinds checks the apr and the solar properties, which hold
// the appearance at the top and under ap, and an XMP packet with none.
func TestParseDesktopKinds(t *testing.T) {
	property := func(kind string, v any) []byte {
		buf, err := plist.Marshal(v, plist.BinaryFormat)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		return []byte(`<rdf:Description apple_desktop:` + kind + `="` + base64.StdEncoding.EncodeToString(buf) + `"/>`)
	}
	d, err := ParseDesktop(property("apr", map[string]int{"l": 1, "d": 0}))
	if err != nil || d == nil || d.Light != 1 || d.Dark != 0 {
		t.Fatalf("apr: expected light 1 and dark 0, got %+v, %v", d, err)
	}
	d, err = ParseDesktop(property("solar", map[string]any{
		"ap": map[string]int{"l": 0, "d": 1},
		"si": []map[string]any{{"i": 0, "a": 30.5, "z": 120.0}},
	}))
	if err != nil || d == nil {
		t.Fatalf("solar: expected metadata, got %v", err)
	}
	if got, exp := d.Describe(0), "the light appearance, with the sun at an altitude of 30.5° and an azimuth of 120.0°"; got != exp {
		t.Errorf("solar: expected %q, got %q", exp, got)
	}
	if d, err := ParseDesktop([]byte(`<x:xmpmeta/>`)); d != nil || err != nil {
		t.Errorf("expected no metadata, got %+v, %v", d, err)
	}
}
