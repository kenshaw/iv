package heif

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/kenshaw/iv/ivctx"
	"howett.net/plist"
)

// Desktop is the metadata of a dynamic wallpaper of macOS, which the XMP of a
// heic file holds in the apple_desktop namespace. Each top-level image of the
// file is shown at a time of the day, at a position of the sun, or for the
// light or the dark appearance of macOS. Index is the index of an image among
// the top-level images of the file, from 0.
type Desktop struct {
	// Kind is the property that held the metadata: h24 for the time of the
	// day, solar for the position of the sun, or apr for the appearance only.
	Kind string
	// Light and Dark are the indexes of the images of the two appearances, or
	// -1 when the file names none.
	Light, Dark int
	// Times holds the time of the day of an image, from 0 to 1.
	Times []DesktopTime
	// Suns holds the position of the sun of an image.
	Suns []DesktopSun
}

// DesktopTime is an image that is shown from a time of the day.
type DesktopTime struct {
	Index int     `plist:"i"`
	Time  float64 `plist:"t"`
}

// DesktopSun is an image that is shown at a position of the sun, in degrees.
type DesktopSun struct {
	Index    int     `plist:"i"`
	Altitude float64 `plist:"a"`
	Azimuth  float64 `plist:"z"`
}

// desktopProperty matches a property of the apple_desktop namespace, as an
// attribute or as an element of the XMP.
var desktopProperty = regexp.MustCompile(`apple_desktop:(h24|solar|apr)(?:\s*=\s*"|>)\s*([A-Za-z0-9+/=\s]+?)\s*["<]`)

// ParseDesktop reads the wallpaper metadata in the XMP packet xmp. It returns
// nil when the packet holds none.
func ParseDesktop(xmp []byte) (*Desktop, error) {
	m := desktopProperty.FindSubmatch(xmp)
	if m == nil {
		return nil, nil
	}
	kind := string(m[1])
	data, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(m[2])), ""))
	if err != nil {
		return nil, fmt.Errorf("apple_desktop:%s: %w", kind, err)
	}
	type appearance struct {
		Light *int `plist:"l"`
		Dark  *int `plist:"d"`
	}
	var v struct {
		// apr holds the appearance itself, and h24 and solar hold it under
		// ap
		Light      *int          `plist:"l"`
		Dark       *int          `plist:"d"`
		Appearance *appearance   `plist:"ap"`
		Times      []DesktopTime `plist:"ti"`
		Suns       []DesktopSun  `plist:"si"`
	}
	if _, err := plist.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("apple_desktop:%s: %w", kind, err)
	}
	d := &Desktop{Kind: kind, Light: -1, Dark: -1, Times: v.Times, Suns: v.Suns}
	ap := appearance{Light: v.Light, Dark: v.Dark}
	if v.Appearance != nil {
		ap = *v.Appearance
	}
	if ap.Light != nil {
		d.Light = *ap.Light
	}
	if ap.Dark != nil {
		d.Dark = *ap.Dark
	}
	return d, nil
}

// Describe returns what the metadata says about the image at index, such as
// "the light appearance, from 05:00".
func (d *Desktop) Describe(index int) string {
	var s []string
	switch index {
	case d.Light:
		s = append(s, "the light appearance")
	case d.Dark:
		s = append(s, "the dark appearance")
	}
	for _, t := range d.Times {
		if t.Index == index {
			m := int(math.Round(t.Time*24*60)) % (24 * 60)
			s = append(s, fmt.Sprintf("from %02d:%02d", m/60, m%60))
		}
	}
	for _, sun := range d.Suns {
		if sun.Index == index {
			s = append(s, fmt.Sprintf("with the sun at an altitude of %.1f° and an azimuth of %.1f°", sun.Altitude, sun.Azimuth))
		}
	}
	if len(s) == 0 {
		return "not named by the metadata"
	}
	return strings.Join(s, ", ")
}

// LogDesktop writes to the verbose log what the wallpaper metadata in xmp
// says about the image at index among the count top-level images of the
// file. It writes nothing when xmp holds no such metadata.
func LogDesktop(ctx context.Context, xmp []byte, index, count int) {
	d, err := ParseDesktop(xmp)
	switch {
	case err != nil:
		ivctx.Logf(ctx, "heif: dynamic wallpaper: %v", err)
		return
	case d == nil:
		return
	}
	ivctx.Logf(ctx, "heif: dynamic wallpaper of macOS (apple_desktop:%s) of %d images", d.Kind, count)
	if index >= 0 {
		ivctx.Logf(ctx, "heif: dynamic wallpaper: image %d is %s", index+1, d.Describe(index))
	}
}
