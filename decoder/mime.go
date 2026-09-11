package decoder

import (
	"context"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/gabriel-vasile/mimetype"
	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/magic"
)

// magicPeek is how many leading bytes are handed to libmagic.
const magicPeek = 64 * 1024

// Detect determines the mime type of the reader.
//
// The forced mime type wins, then any decoder supplied [MimeDetector], then
// content sniffing, and finally libmagic's description of the content matched
// against the patterns registered with [RegisterMimeType].
//
// The reader is rewound to its start before returning.
func Detect(ctx context.Context, rs io.ReadSeeker) (string, error) {
	if mime := ivctx.Get(ctx).ForceMime; mime != "" {
		return normalizeMime(mime), nil
	}
	defer rs.Seek(0, io.SeekStart)
	for _, d := range All() {
		if d.detect == nil {
			continue
		}
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		switch mime, err := d.detect(ctx, rs); {
		case err != nil:
			ivctx.Logf(ctx, "mime detect %s: %v", d.Name, err)
		case mime != "":
			return normalizeMime(mime), nil
		}
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	m, err := mimetype.DetectReader(rs)
	if err != nil {
		return "", err
	}
	mime := normalizeMime(m.String())
	if !isUnidentified(mime) {
		return mime, nil
	}
	// the sniffed type says nothing useful; ask libmagic
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return mime, nil
	}
	buf := make([]byte, magicPeek)
	n, err := io.ReadFull(rs, buf)
	if n == 0 && err != nil {
		return mime, nil
	}
	if desc, err := Describe(buf[:n]); err == nil && desc != "" {
		ivctx.Logf(ctx, "libmagic: %s", desc)
		if v, ok := lookupDescription(desc); ok {
			return v, nil
		}
	}
	return mime, nil
}

// isUnidentified reports whether content sniffing failed to identify the
// content at all.
//
// Only such content is handed to libmagic. Text that sniffed as text/plain is
// deliberately excluded: the registered [MimeDetector] funcs already cover the
// text formats iv cares about, and libmagic's text analysis has been observed
// to read out of bounds and crash the process on ordinary input (a mermaid
// diagram and a graphviz dot file in testdata both trigger it).
func isUnidentified(mime string) bool {
	return mime == "" || mime == "application/octet-stream"
}

// normalizeMime strips any parameters and lower cases the mime type.
func normalizeMime(mime string) string {
	mime, _, _ = strings.Cut(mime, ";")
	return strings.ToLower(strings.TrimSpace(mime))
}

var (
	magicMu   sync.Mutex
	magicOnce sync.Once
	magicDB   *magic.Magic
	magicErr  error
)

// Describe returns libmagic's textual description of the buffer.
func Describe(buf []byte) (string, error) {
	magicOnce.Do(func() {
		magicDB, magicErr = magic.New(magic.None)
	})
	if magicErr != nil {
		return "", magicErr
	}
	// a magic_t cookie is not safe for concurrent use
	magicMu.Lock()
	defer magicMu.Unlock()
	return magicDB.Buffer(buf)
}

// descriptions maps libmagic description patterns to mime types.
var (
	descMu       sync.RWMutex
	descriptions []description
)

// description is a libmagic description pattern and the mime type it implies.
type description struct {
	re   *regexp.Regexp
	mime string
}

// RegisterMimeType registers regular expressions matched against libmagic's
// description of content whose mime type could not otherwise be determined.
// The first matching pattern wins.
func RegisterMimeType(mime string, patterns ...string) {
	descMu.Lock()
	defer descMu.Unlock()
	for _, pattern := range patterns {
		descriptions = append(descriptions, description{
			re:   regexp.MustCompile(`(?i)` + pattern),
			mime: mime,
		})
	}
}

// lookupDescription returns the mime type for a libmagic description.
func lookupDescription(desc string) (string, bool) {
	descMu.RLock()
	defer descMu.RUnlock()
	for _, d := range descriptions {
		if d.re.MatchString(desc) {
			return d.mime, true
		}
	}
	return "", false
}
