// Package mermaid supplies a mermaid diagram decoder for iv, rendering
// diagrams with the mermaid cli (mmdc).
//
// See: https://github.com/mermaid-js/mermaid-cli
package mermaid

import (
	"bytes"
	"context"
	"io"

	"github.com/kenshaw/iv/decoder"
	"github.com/kenshaw/iv/internal/cmd"
	"github.com/kenshaw/iv/ivctx"
)

// mmdc is the mermaid cli command.
var mmdc = cmd.New("mmdc")

func init() {
	decoder.Register(
		"mermaid",
		decoder.Desc("Mermaid diagrams"),
		decoder.Extension("mmd", "mermaid"),
		decoder.MimeTypeExtensionMatch(
			"text/plain", "mmd",
			"text/plain", "mermaid",
		),
		decoder.Decoder(decode),
	)
}

// decode renders a mermaid diagram as svg, handing it back to the pipeline.
func decode(ctx context.Context, r io.Reader) (any, error) {
	src, cleanup, err := cmd.Source(ctx, r, "mmd")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	params := []string{
		"--outputFormat", "svg",
		"--input", src,
		"--output", "-",
		"--iconPacks", "@iconify-json/logos",
	}
	params = append(params, ivctx.Get(ctx).MermaidIcons...)
	buf, err := mmdc.Run(ctx, params...)
	if err != nil {
		return nil, err
	}
	return decoder.NewImage("image/svg+xml", bytes.NewReader(buf)).WithExt("svg"), nil
}
