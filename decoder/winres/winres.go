// Package winres supplies a Windows PE decoder for iv, rendering the icons
// embedded in an executable.
//
// See: https://github.com/tc-hib/winres
package winres

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/tc-hib/winres"
)

func init() {
	decoder.Register(
		"winres",
		decoder.Desc("Windows Portable Executable (icons)"),
		decoder.Extension("exe", "dll", "mui"),
		decoder.MimeType("application/vnd.microsoft.portable-executable"),
		decoder.Decoder(decode),
	)
}

// decode decodes the application icons embedded in a Windows PE file.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("%T: not seekable", r)
	}
	set, err := winres.LoadFromEXE(rs)
	if err != nil {
		return nil, fmt.Errorf("unable to load: %w", err)
	}
	var icons []image.Image
	var walkErr error
	set.Walk(func(typid, id winres.Identifier, lang uint16, data []byte) bool {
		ivctx.Logf(ctx, "resource type: %v res: %v lang: %v len: %d", typid, id, lang, len(data))
		// the resource type says what this is outright. Sniffing the bytes
		// instead only ever worked by accident: an icon directory is a 62
		// byte table of contents that no detector types as an image.
		if typid != winres.RT_GROUP_ICON {
			return true
		}
		icon, err := set.GetIconTranslation(id, lang)
		if err != nil {
			walkErr = fmt.Errorf("resource %v: unable to read icon: %w", id, err)
			return false
		}
		var buf bytes.Buffer
		if err := icon.SaveICO(&buf); err != nil {
			walkErr = fmt.Errorf("resource %v: unable to save icon: %w", id, err)
			return false
		}
		img, _, err := image.Decode(&buf)
		if err != nil {
			walkErr = fmt.Errorf("resource %v: unable to decode icon: %w", id, err)
			return false
		}
		icons = append(icons, img)
		return true
	})
	switch {
	case walkErr != nil:
		return nil, walkErr
	case len(icons) == 0:
		return nil, fmt.Errorf("no icons found")
	}
	return icons, nil
}
