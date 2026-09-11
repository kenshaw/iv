// Package binwalk supplies a decoder for iv that extracts images embedded in
// otherwise unrecognized files using the `binwalk` command.
//
// See: https://github.com/ReFirmLabs/binwalk
package binwalk

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing/fstest"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// binwalk is the binwalk command.
var binwalk = cmd.New("binwalk")

func init() {
	decoder.Register(
		"binwalk",
		decoder.Desc("Embedded images in unrecognized files (via binwalk)"),
		// a last resort, once every format aware decoder has declined
		decoder.Fallback(),
		decoder.Extension("afdesign", "afphoto", "afpub"),
		decoder.Matcher(func(mime, _ string) bool {
			return mime == "application/octet-stream"
		}),
		decoder.Decoder(decode),
	)
}

// decode extracts the file with binwalk, handing the extracted images back to
// the pipeline. The extracted files are read into memory so the extraction
// directory can be removed before the pipeline decodes them.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanupSrc, err := cmd.Source(ctx, r, ivctx.FileExt(ivctx.PathName(ctx)))
	if err != nil {
		return nil, err
	}
	defer cleanupSrc()
	dir, cleanupDir, err := cmd.TempDir(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanupDir()
	if _, err := binwalk.Run(ctx, "--extract", "--quiet", "--directory", dir, src); err != nil {
		return nil, err
	}
	fsys := fstest.MapFS{}
	err = filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir(), !decoder.BuiltinExt(name):
			return nil
		}
		buf, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		fsys[filepath.ToSlash(rel)] = &fstest.MapFile{Data: buf}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(fsys) == 0 {
		return nil, fmt.Errorf("binwalk extracted no images")
	}
	names := make([]string, 0, len(fsys))
	for name := range fsys {
		names = append(names, name)
	}
	sort.Strings(names)
	ivctx.Logf(ctx, "binwalk images: %d", len(names))
	return decoder.FS(fsys, names...), nil
}
