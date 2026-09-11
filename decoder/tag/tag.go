// Package tag supplies an audio metadata decoder for iv, rendering embedded
// album art.
//
// See: https://github.com/dhowden/tag
package tag

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/dhowden/tag"
	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

func init() {
	decoder.Register(
		"tag",
		decoder.Desc("Audio metadata (album art)"),
		decoder.Extension("mp3", "m4a", "m4b", "m4p", "flac", "ogg", "oga", "dsf", "aac"),
		decoder.MimeType("audio/*"),
		decoder.Decoder(decode),
	)
}

// decode decodes the embedded picture from audio metadata.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("%T: not seekable", r)
	}
	md, err := tag.ReadFrom(rs)
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "tag format: %s %s - %s", md.Format(), md.Artist(), md.Title())
	pic := md.Picture()
	if pic == nil {
		return nil, errors.New("no embedded picture")
	}
	mime := pic.MIMEType
	if mime == "" {
		mime = "application/octet-stream"
	}
	return decoder.NewBytes(mime, pic.Data), nil
}
