package blitz

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// ErrFetch is returned when the url could not be retrieved at all, which
// separates a network or site problem from a failure to render what came
// back.
var ErrFetch = errors.New("fetch failed")

// decodeURL fetches the url and either renders it as a page or hands it back
// to the pipeline. A url naming an image or a pdf is still that image or pdf,
// so only the documents blitz is for -- html and markdown -- are rendered
// here; everything else decodes as though it had come off disk.
func decodeURL(ctx context.Context, urlstr string) (any, error) {
	c, err := engine(ctx)
	if err != nil {
		return nil, err
	}
	buf, mime, err := fetch(ctx, urlstr)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.Contains(mime, "html"):
		return render(ctx, c, string(buf), urlstr, true)
	case strings.Contains(mime, "markdown"):
		return render(ctx, c, string(buf), urlstr, false)
	}
	var name string
	if u, err := url.Parse(urlstr); err == nil {
		name = path.Base(u.Path)
	}
	return decoder.NewBytes(mime, buf).WithExt(ivctx.FileExt(name)), nil
}

// fetch retrieves the url, returning the body and the mime type the server
// reported. A server that reports nothing usable is overruled by sniffing the
// body, which is what lets an extensionless url still route.
func fetch(ctx context.Context, urlstr string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlstr, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s: %w", ErrFetch, urlstr, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%w: %s: %s", ErrFetch, urlstr, res.Status)
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s: %w", ErrFetch, urlstr, err)
	}
	mime, _, _ := strings.Cut(res.Header.Get("Content-Type"), ";")
	if mime = strings.TrimSpace(mime); mime == "" || mime == "application/octet-stream" {
		mime, _, _ = strings.Cut(http.DetectContentType(buf), ";")
	}
	ivctx.Logf(ctx, "blitz url: %s %s %d bytes %s", urlstr, res.Status, len(buf), mime)
	return buf, mime, nil
}
