// Package data supplies an inline data: URL decoder for iv.
package data

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/kenshaw/iv/decoder"
)

// prefix is the string prefix claimed by the data decoder.
const prefix = "data:"

func init() {
	decoder.Register(
		"data",
		decoder.Desc("inline data: URLs"),
		decoder.StringMatcher(func(s string) bool {
			return strings.HasPrefix(strings.ToLower(s), prefix)
		}),
		decoder.StringDecoder(decode),
	)
}

// decode decodes an inline data: URL, handing the embedded content back to the
// pipeline under its declared mime type.
func decode(_ context.Context, s string) (any, error) {
	mime, data, ok := strings.Cut(s[len(prefix):], ",")
	if !ok {
		return nil, fmt.Errorf("malformed data: missing %q", ',')
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	var err error
	var buf []byte
	if s, ok := strings.CutSuffix(mime, ";base64"); ok {
		mime = s
		buf, err = base64.StdEncoding.AppendDecode(buf, []byte(data))
	} else {
		var v string
		v, err = url.QueryUnescape(data)
		buf = []byte(v)
	}
	if err != nil {
		return nil, fmt.Errorf("malformed data: %w", err)
	}
	if mime == "" {
		mime = "text/plain"
	}
	return decoder.NewBytes(mime, buf), nil
}
