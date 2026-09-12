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
	"time"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
)

// ErrFetch is returned when the url could not be retrieved at all, which
// separates a network or site problem from a failure to render what came
// back.
var ErrFetch = errors.New("fetch failed")

const (
	// fetchTimeout bounds the whole retrieval. Without one a server that
	// accepts the connection and then stalls leaves iv waiting forever with
	// nothing on screen to say why.
	fetchTimeout = 30 * time.Second
	// maxFetch bounds what is read from a url. Generous for a page or an
	// image, and the difference between a bad url and an out of memory.
	maxFetch = 64 << 20
)

// client is the http client used for every fetch. The default one has no
// timeout at all.
var client = &http.Client{Timeout: fetchTimeout}

// decodeURL fetches the url and either renders it as a page or hands it back
// to the pipeline. A url naming an image or a pdf is still that image or pdf,
// so only the documents blitz is for -- html and markdown -- are rendered
// here; everything else decodes as though it had come off disk.
func decodeURL(ctx context.Context, urlstr string) (any, error) {
	c, err := engine(ctx)
	if err != nil {
		return nil, err
	}
	buf, mime, final, err := fetch(ctx, urlstr)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.Contains(mime, "html"):
		return render(ctx, c, string(buf), final, true)
	case strings.Contains(mime, "markdown"):
		return render(ctx, c, string(buf), final, false)
	}
	var name string
	if u, err := url.Parse(final); err == nil {
		name = path.Base(u.Path)
	}
	return decoder.NewBytes(mime, buf).WithExt(ivctx.FileExt(name)), nil
}

// fetch retrieves the url, returning the body, the mime type the server
// reported, and the url the body actually came from. A server that reports
// nothing usable is overruled by sniffing the body, which is what lets an
// extensionless url still route.
//
// The url followed matters: relative links and sub-resources in a page have
// to resolve against where it was served from, not where the request was
// pointed, and the two differ the moment anything redirects.
func fetch(ctx context.Context, urlstr string) ([]byte, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlstr, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := client.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: %s: %w", ErrFetch, urlstr, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("%w: %s: %s", ErrFetch, urlstr, res.Status)
	}
	// one byte past the limit is read so that hitting it can be told from a
	// body that merely ends there
	buf, err := io.ReadAll(io.LimitReader(res.Body, maxFetch+1))
	switch {
	case err != nil:
		return nil, "", "", fmt.Errorf("%w: %s: %w", ErrFetch, urlstr, err)
	case len(buf) > maxFetch:
		return nil, "", "", fmt.Errorf("%w: %s: larger than the %d byte limit", ErrFetch, urlstr, maxFetch)
	}
	mime, _, _ := strings.Cut(res.Header.Get("Content-Type"), ";")
	if mime = strings.TrimSpace(mime); mime == "" || mime == "application/octet-stream" {
		mime, _, _ = strings.Cut(http.DetectContentType(buf), ";")
	}
	final := urlstr
	if res.Request != nil && res.Request.URL != nil {
		final = res.Request.URL.String()
	}
	if final != urlstr {
		ivctx.Logf(ctx, "blitz url: %s redirected to %s", urlstr, final)
	}
	ivctx.Logf(ctx, "blitz url: %s %s %d bytes %s", final, res.Status, len(buf), mime)
	return buf, mime, final, nil
}
