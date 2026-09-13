// Package mermaid supplies a mermaid diagram decoder for iv, rendering
// diagrams to svg with the mermaid renderer.
//
// See: https://github.com/xo/mermaid
package mermaid

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/ivctx"
	"github.com/tetratelabs/wazero"
	"github.com/xo/mermaid"
)

func init() {
	decoder.Register(
		"mermaid",
		decoder.Desc("Mermaid diagrams"),
		decoder.Extension("mmd", "mermaid"),
		decoder.MimeTypeExtensionMatch(
			"text/plain", "mmd",
			"text/plain", "mermaid",
		),
		decoder.Init(open, shut),
		decoder.Decoder(decode),
	)
}

// renderer is the decoder's state: the mermaid renderer, and the compilation
// cache standing behind it. Both need closing.
type renderer struct {
	*mermaid.Renderer
	cache wazero.CompilationCache
}

// open builds the renderer.
//
// Compiling the webassembly module is what this costs -- well over a second,
// against a couple of milliseconds to render a diagram -- so the compiled code
// is cached on disk and the next run reads it back instead. Nothing here fails
// for want of a cache: an unwritable one is rendered around, slowly.
func open(ctx context.Context) (any, error) {
	v := new(renderer)
	var opts []mermaid.Option
	if dir, err := cacheDir(); err != nil {
		ivctx.Logf(ctx, "mermaid cache: %v", err)
	} else if v.cache, err = wazero.NewCompilationCacheWithDir(dir); err != nil {
		ivctx.Logf(ctx, "mermaid cache %s: %v", dir, err)
	} else {
		ivctx.Logf(ctx, "mermaid cache: %s", dir)
		opts = append(opts, mermaid.WithCompilationCache(v.cache))
	}
	r, err := mermaid.New(ctx, opts...)
	if err != nil {
		if v.cache != nil {
			v.cache.Close(ctx)
		}
		return nil, err
	}
	v.Renderer = r
	ivctx.Logf(ctx, "mermaid renderer: %s", r.Version())
	return v, nil
}

// shut closes the renderer, releasing the webassembly runtime with it, and the
// cache after it.
func shut(ctx context.Context) error {
	v, _ := decoder.State(ctx).(*renderer)
	if v == nil {
		return nil
	}
	var errs []error
	if v.Renderer != nil {
		errs = append(errs, v.Renderer.Close(ctx))
	}
	if v.cache != nil {
		errs = append(errs, v.cache.Close(ctx))
	}
	return errors.Join(errs...)
}

// cacheDir returns the directory the compiled module is cached in, creating it
// when it is not there.
//
// wazero names what it writes after a hash of the module, its own version and
// the compiler options, so an upgrade lands beside the old entry rather than
// on top of it, and the old one is simply never read again.
func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "iv", "wazero")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// decode renders the diagram as svg, handing it back to the pipeline.
func decode(ctx context.Context, r io.Reader) (any, error) {
	rend, _ := decoder.State(ctx).(*renderer)
	if rend == nil || rend.Renderer == nil {
		return nil, errors.New("renderer not initialized")
	}
	return render(ctx, rend, r)
}

// render renders the diagram with the renderer, which [decode] takes from the
// pipeline and a test supplies for itself.
func render(ctx context.Context, rend *renderer, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// an unknown theme comes back as an error naming it, so there is nothing
	// to check here
	svg, err := rend.RenderWith(ctx, string(buf), mermaid.Options{
		Theme: mermaid.Theme(ivctx.Get(ctx).MermaidTheme),
	})
	if err != nil {
		return nil, err
	}
	ivctx.Logf(ctx, "mermaid: %d bytes of source to %d of svg", len(buf), len(svg))
	return decoder.NewBytes("image/svg+xml", []byte(svg)).WithExt("svg"), nil
}
