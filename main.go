package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	_ "github.com/jcbritobr/pnm"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/kenshaw/rasterm"
)

func main() {
	if err := run(os.Stdout, os.Args[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

/*
func init() {
	"github.com/klippa-app/go-libheif"
	"github.com/klippa-app/go-libheif/library"
	err := libheif.Init(libheif.Config{LibraryConfig: library.Config{
		Command: library.Command{
			BinPath: "go",
			Args:    []string{"run", "library/worker_example/main.go"},
		},
	}})
	if err != nil {
		panic(fmt.Sprintf("could not start libheif worker: %v", err))
	}
}
*/

/*
func init() {
	image.RegisterFormat("pbm", "P?", Decode, DecodeConfig)
}
*/

func run(w io.Writer, files ...string) error {
	for _, file := range files {
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
		fmt.Fprintln(w, file+":")
		if err := rasterm.Encode(w, img); err != nil {
			return err
		}
	}
	return nil
}
