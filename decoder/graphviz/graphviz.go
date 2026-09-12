// Package graphviz supplies a graphviz (dot) decoder for iv.
//
// See: https://github.com/goccy/go-graphviz
package graphviz

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/kenshaw/iv/decoder"
)

// mime is the graphviz mime type.
const mime = "text/vnd.graphviz"

// graphRE matches the opening of a dot graph.
var graphRE = regexp.MustCompile(`(?is)^\s*(strict\s+)?(di)?graph\b[^{]*\{`)

// peek is how many leading bytes are sniffed for a dot graph.
const peek = 512

func init() {
	decoder.Register(
		"graphviz",
		decoder.Desc("Graphviz graph description language"),
		decoder.Extension("gv", "dot"),
		decoder.MimeType(mime),
		decoder.MimeDetector(detect),
		decoder.Init(
			func(ctx context.Context) (any, error) {
				return graphviz.New(ctx)
			},
			func(ctx context.Context) error {
				gv, _ := decoder.State(ctx).(*graphviz.Graphviz)
				if gv == nil {
					return nil
				}
				return gv.Close()
			},
		),
		decoder.Decoder(decode),
	)
}

// detect sniffs the reader for a dot graph.
//
// A short read is not an error here, only less to look at: a stream that could
// not be read at all is simply not a dot graph, and saying so is what the
// sniffer is for.
func detect(_ context.Context, r io.Reader) (string, error) {
	buf, _ := bufio.NewReaderSize(r, peek).Peek(peek)
	if graphRE.Match(buf) {
		return mime, nil
	}
	return "", nil
}

// decode renders the dot graph as svg, handing it back to the pipeline.
func decode(ctx context.Context, r io.Reader) (any, error) {
	gv, _ := decoder.State(ctx).(*graphviz.Graphviz)
	if gv == nil {
		return nil, fmt.Errorf("graphviz not initialized")
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	g, err := cgraph.ParseBytes(buf)
	if err != nil {
		return nil, err
	}
	defer g.Close()
	out := new(bytes.Buffer)
	if err := gv.Render(ctx, g, graphviz.SVG, out); err != nil {
		return nil, err
	}
	return decoder.NewImage("image/svg+xml", out).WithExt("svg"), nil
}
