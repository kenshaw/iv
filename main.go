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

func run(ctx context.Context, name, version string, cliargs []string) error {
	bg := colors.FromColor(color.Transparent)
	var (
		bashCompletion       bool
		zshCompletion        bool
		fishCompletion       bool
		powershellCompletion bool
		noDescriptions       bool
	)
	c := &cobra.Command{
		Use:           name + " [flags] <image1> [image2, ..., imageN]",
		Short:         name + ", a command-line image viewer using terminal graphics",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  false,
		RunE: func(cmd *cobra.Command, cliargs []string) error {
			// completions and short circuits
			switch {
			case bashCompletion:
				return cmd.GenBashCompletionV2(os.Stdout, !noDescriptions)
			case zshCompletion:
				if noDescriptions {
					return cmd.GenZshCompletionNoDesc(os.Stdout)
				}
				return cmd.GenZshCompletion(os.Stdout)
			case fishCompletion:
				return cmd.GenFishCompletion(os.Stdout, !noDescriptions)
			case powershellCompletion:
				if noDescriptions {
					return cmd.GenPowerShellCompletion(os.Stdout)
				}
				return cmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return do(os.Stdout, bg, cliargs)
		},
	}
	c.SetVersionTemplate("{{ .Name }} {{ .Version }}\n")
	c.InitDefaultHelpCmd()
	c.SetArgs(cliargs[1:])
	flags := c.Flags()
	flags.Var(bg.Pflag(), "bg", "background color")
	// completions
	flags.BoolVar(&bashCompletion, "completion-script-bash", false, "output bash completion script and exit")
	flags.BoolVar(&zshCompletion, "completion-script-zsh", false, "output zsh completion script and exit")
	flags.BoolVar(&fishCompletion, "completion-script-fish", false, "output fish completion script and exit")
	flags.BoolVar(&powershellCompletion, "completion-script-powershell", false, "output powershell completion script and exit")
	flags.BoolVar(&noDescriptions, "no-descriptions", false, "disable descriptions in completion scripts")
	// mark hidden
	for _, name := range []string{
		"completion-script-bash", "completion-script-zsh", "completion-script-fish",
		"completion-script-powershell", "no-descriptions",
	} {
		flags.Lookup(name).Hidden = true
	}
	return c.ExecuteContext(ctx)
}

// do renders the specified files to w.
func do(w io.Writer, bg color.Color, args []string) error {
	if !rasterm.Available() {
		return rasterm.ErrTermGraphicsNotAvailable
	}
	resvg.WithBackground(bg)(resvg.Default)
	// collect files
	var files []string
	for i := 0; i < len(args); i++ {
		v, err := open(args[i])
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
