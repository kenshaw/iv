// Command iv is a command-line image viewer using terminal graphics (Sixel,
// iTerm, Kitty).
package main

import (
	"context"
	"os"

	"github.com/kenshaw/iv/ivcmd"
	"github.com/xo/ox"
	_ "github.com/xo/ox/color"
)

var (
	name    = "iv"
	version = "0.0.0-dev"
)

func main() {
	args := ivcmd.New()
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "the command-line terminal graphics image viewer"),
		ox.VersionString(version),
		ox.Defaults(),
		ox.Exec(args.Run(os.Stdout, os.Stderr)),
		ox.From(args),
	)
}
