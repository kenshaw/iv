// Package binwalk supplies a decoder for iv that finds the images embedded in
// a file that no other decoder recognizes, with the walker of xo/magic, a
// port of binwalk to Go (D47).
//
// See: https://github.com/xo/magic
package binwalk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"testing/fstest"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/magic/walk"
	// registers every format of the walker
	_ "github.com/xo/magic/walk/format/all"
	// registers the stream decoders that the formats of xz, lzma and zstd
	// need, as the binwalk command unpacks them
	_ "github.com/xo/magic/compress/xz"
	_ "github.com/xo/magic/compress/zstd"
)

// depth is how deep the walk goes. At 1, the walker also walks the files in
// each match that it can unpack, such as the files of an archive, as the
// binwalk command does with --extract.
const depth = 1

// maxOutput is the most bytes that the walker unpacks from one match, in
// memory. A compression bomb stops there.
const maxOutput = 256 << 20

// ErrNoImages is the error when the walk finds no image.
var ErrNoImages = errors.New("no embedded images")

func init() {
	decoder.Register(
		"binwalk",
		decoder.Desc("Embedded images in unrecognized files"),
		// a last resort, once every format aware decoder has declined
		decoder.Fallback(),
		decoder.Extension("afdesign", "afphoto", "afpub"),
		decoder.Matcher(func(mime, _ string) bool {
			return mime == "application/octet-stream"
		}),
		decoder.Decoder(decode),
	)
}

// decode walks the file, and hands the images that it finds back to the
// pipeline, in the order of their offsets. An image in a file that the walker
// unpacked, such as a file of an archive, follows the match that holds it.
func decode(ctx context.Context, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	fsys := fstest.MapFS{}
	for e, err := range walk.WalkBytes(ctx, buf, walk.WithDepth(depth), walk.WithMaxOutput(maxOutput)) {
		if err != nil {
			return nil, fmt.Errorf("binwalk: %w", err)
		}
		ext, ok := imageExt(e)
		if !ok {
			continue
		}
		// the walker carves an image as a file of its own, and would find the
		// same image again in it
		e.SkipMembers()
		data, err := io.ReadAll(e.Section())
		if err != nil {
			return nil, fmt.Errorf("binwalk: %s at %d: %w", e.Format, e.Offset, err)
		}
		ivctx.Logf(ctx, "binwalk: %s at %d, %d bytes: %s", e.Format, e.Offset, len(data), e.Description)
		fsys[entryName(e, ext)] = &fstest.MapFile{Data: data}
	}
	if len(fsys) == 0 {
		return nil, ErrNoImages
	}
	names := make([]string, 0, len(fsys))
	for name := range fsys {
		names = append(names, name)
	}
	slices.Sort(names)
	ivctx.Logf(ctx, "binwalk images: %d", len(names))
	return decoder.FS(fsys, names...), nil
}

// imageExt returns the extension of the image that the match is, when a
// builtin decoder of iv reads it. A RIFF match is an image only when it holds
// a WebP, because a WAVE is a RIFF too.
func imageExt(e *walk.Entry) (string, bool) {
	var ext string
	switch e.Format {
	case "png", "gif", "bmp":
		ext = e.Format
	case "jpeg":
		ext = "jpg"
	case "riff":
		if !strings.Contains(e.Description, "WEBP") {
			return "", false
		}
		ext = "webp"
	default:
		return "", false
	}
	return ext, decoder.BuiltinExt("image." + ext)
}

// entryName returns the name of the image of the match in the file set: the
// offset of each match on the path to it, in hex, and the name of the file
// that holds it in the match above it. The offsets have 8 digits, so the
// names sort in the order of the file.
func entryName(e *walk.Entry, ext string) string {
	name := fmt.Sprintf("%08X.%s", e.Offset, ext)
	for p := e; p.Parent != nil; p = p.Parent {
		// the name comes from the file, so it is cleaned into a path that
		// stays in the set
		member := strings.TrimPrefix(path.Clean("/"+p.Member), "/")
		name = path.Join(fmt.Sprintf("%08X", p.Parent.Offset), member, name)
	}
	return name
}
