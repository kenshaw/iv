// Package http supplies a http/https URL decoder for iv, fetching the remote
// resource and handing it back to the decoding pipeline.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// UserAgent is the user agent sent with requests.
var UserAgent = "iv"

func init() {
	decoder.Register(
		"http",
		decoder.Desc("Remote http/https resources"),
		decoder.StringMatcher(func(s string) bool {
			s = strings.ToLower(s)
			return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
		}),
		decoder.StringDecoder(decode),
	)
}

// decode fetches the URL, handing the response body back to the pipeline under
// the mime type the server reported.
func decode(ctx context.Context, urlstr string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlstr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", urlstr, res.Status)
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	mime, _, _ := strings.Cut(res.Header.Get("Content-Type"), ";")
	mime = strings.TrimSpace(mime)
	ivctx.Logf(ctx, "http: %s %d bytes %s", res.Status, len(buf), mime)
	var name string
	if u, err := url.Parse(urlstr); err == nil {
		name = path.Base(u.Path)
	}
	return decoder.NewBytes(mime, buf).WithExt(ivctx.FileExt(name)), nil
}
