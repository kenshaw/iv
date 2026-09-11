package tag

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kenshaw/iv/ivctx"
)

// coverNames are the base names, in order of preference, that rippers and
// music players conventionally give the cover art sitting beside the audio.
var coverNames = []string{"cover", "folder", "front", "album", "albumart", "artwork"}

// coverExts are the extensions searched for, in order of preference.
var coverExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

// coverMimes are the image types the card can embed, as reported by
// [http.DetectContentType]. The file's content decides its type: a cover.jpg
// that is really a png is common enough in a music library, and the extension
// is only ever a hint about where to look.
var coverMimes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// maxCover is the largest sidecar cover embedded in a card. The art goes into
// the svg base64 encoded, so this is a guard against a directory holding a
// print resolution scan rather than a thumbnail.
const maxCover = 16 << 20

// sidecar returns cover art found beside the audio file, for the files that
// carry no embedded picture of their own. Returns nil when the audio did not
// come from a directory that has one.
func sidecar(ctx context.Context, pathName string) ([]byte, string) {
	if pathName == "" {
		return nil, ""
	}
	dir := filepath.Dir(pathName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, ""
	}
	// index the directory folded, since whether the file on disk is cover.jpg
	// or Cover.JPG depends on the platform and on whoever ripped the album
	files := make(map[string]os.DirEntry, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files[strings.ToLower(entry.Name())] = entry
		}
	}
	for _, name := range coverNames {
		for _, ext := range coverExts {
			entry, ok := files[name+ext]
			if !ok {
				continue
			}
			buf, mime := readCover(ctx, filepath.Join(dir, entry.Name()), entry)
			if buf != nil {
				ivctx.Logf(ctx, "tag cover: %s %s %d bytes", entry.Name(), mime, len(buf))
				return buf, mime
			}
		}
	}
	return nil, ""
}

// readCover reads a candidate cover, returning it with the mime type sniffed
// from its content. Returns nil when it is too large to embed, unreadable, or
// not an image type the card can carry.
func readCover(ctx context.Context, pathName string, entry os.DirEntry) ([]byte, string) {
	switch fi, err := entry.Info(); {
	case err != nil:
		return nil, ""
	case fi.Size() > maxCover:
		ivctx.Logf(ctx, "tag cover: %s: %d bytes exceeds the %d byte limit", entry.Name(), fi.Size(), maxCover)
		return nil, ""
	}
	buf, err := os.ReadFile(pathName)
	if err != nil {
		ivctx.Logf(ctx, "tag cover: %s: %v", entry.Name(), err)
		return nil, ""
	}
	mime, _, _ := strings.Cut(http.DetectContentType(buf), ";")
	if !coverMimes[mime] {
		ivctx.Logf(ctx, "tag cover: %s: %s cannot be embedded", entry.Name(), mime)
		return nil, ""
	}
	return buf, mime
}
