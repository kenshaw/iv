// Command iv is a command-line image viewer using terminal graphics (Sixel,
// iTerm, Kitty).
package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "github.com/xo/heif"
	"github.com/xo/resvg"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/kenshaw/colors"
	"github.com/kenshaw/rasterm"
	"github.com/spf13/cobra"
)

var (
	name    = "iv"
	version = "0.0.0-dev"
)

func main() {
	if err := run(context.Background(), name, version, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, appName, appVersion string, cliargs []string) error {
	bg := colors.FromColor(color.Transparent)
	c := &cobra.Command{
		Use:     appName + " [flags] <image1> [image2, ..., imageN]",
		Short:   appName + ", a command-line image viewer using terminal graphics",
		Version: appVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return do(os.Stdout, &Params{
				BG:   bg,
				Args: args,
			})
		},
	}
	c.Flags().Var(bg.Pflag(), "bg", "background color")
	c.SetVersionTemplate("{{ .Name }} {{ .Version }}\n")
	c.InitDefaultHelpCmd()
	c.SetArgs(cliargs[1:])
	c.SilenceErrors, c.SilenceUsage = true, false
	return c.ExecuteContext(ctx)
}

type Params struct {
	BG   color.Color
	Args []string
}

// do renders the specified files to w.
func do(w io.Writer, params *Params) error {
	if !rasterm.Available() {
		return rasterm.ErrTermGraphicsNotAvailable
	}
	resvg.WithBackground(params.BG)(resvg.Default)
	// collect files
	var files []string
	for i := 0; i < len(params.Args); i++ {
		v, err := open(params.Args[i])
		if err != nil {
			fmt.Fprintf(w, "error: unable to open arg %d: %v\n", i, err)
		}
		files = append(files, v...)
	}
	return render(w, files)
}

func open(name string) ([]string, error) {
	var v []string
	switch fi, err := os.Stat(name); {
	case err == nil && fi.IsDir():
		entries, err := os.ReadDir(name)
		if err != nil {
			return nil, fmt.Errorf("unable to open directory %q: %v", name, err)
		}
		for _, entry := range entries {
			if s := entry.Name(); !entry.IsDir() && extRE.MatchString(s) {
				v = append(v, filepath.Join(name, s))
			}
		}
		sort.Strings(v)
	case err == nil:
		v = append(v, name)
	default:
		return nil, fmt.Errorf("unable to open %q", name)
	}
	return v, nil
}

var extRE = regexp.MustCompile(`(?i)\.(jpe?g|gif|png|svg|bmp|bitmap|tiff?|hei[vc]|webp)$`)

func render(w io.Writer, files []string) error {
	for i := 0; i < len(files); i++ {
		if err := renderFile(w, files[i]); err != nil {
			fmt.Fprintf(w, "error: unable to render arg %d: %v\n", i, err)
		}
	}
	return nil
}

// doFile renders the specified file to w.
func renderFile(w io.Writer, file string) error {
	fmt.Fprintln(w, file+":")
	f, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("can't open %s: %w", file, err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		defer f.Close()
		return fmt.Errorf("can't decode %s: %w", file, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("can't close %s: %w", file, err)
	}
	return rasterm.Encode(w, img)
}

/*
func init() {
	_ "github.com/jcbritobr/pnm"
	image.RegisterFormat("pbm", "P?", Decode, DecodeConfig)
}
*/
