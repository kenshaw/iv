// Package archives supplies a comic book archive decoder for iv.
//
// See: https://github.com/mholt/archives
package archives

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"sort"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/mholt/archives"
)

func init() {
	decoder.Register(
		"archives",
		decoder.Desc("Comic Book Archives"),
		decoder.MimeTypeExtensionMatch(
			"application/x-7z-compressed", "cb7",
			"application/vnd.rar", "cbr",
			"application/x-rar", "cbr",
			"application/x-rar-compressed", "cbr",
			"application/x-tar", "cbt",
			"application/zip", "cbz",
		),
		decoder.Decoder(decode),
	)
}

// decode lists the renderable images in the archive, handing them back to the
// pipeline for the configured page to be selected.
func decode(ctx context.Context, r io.Reader) (any, error) {
	file, ok := r.(archives.ReaderAtSeeker)
	if !ok {
		return nil, fmt.Errorf("%T: not a ReaderAtSeeker", r)
	}
	fsys, err := archives.FileSystem(ctx, ivctx.PathName(ctx), file)
	if err != nil {
		return nil, err
	}
	var names []string
	err = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !decoder.BuiltinExt(name):
			return nil
		}
		names = append(names, name)
		return nil
	})
	switch {
	case err != nil:
		return nil, err
	case len(names) == 0:
		return nil, fmt.Errorf("no renderable images in archive")
	}
	sort.Strings(names)
	ivctx.Logf(ctx, "archive images: %d", len(names))
	return decoder.FS(fsys, names...), nil
}
