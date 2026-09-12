//go:build !unix

package ivctx

import (
	"os"

	"golang.org/x/term"
)

// termSize returns the terminal's cell grid. There is no TIOCGWINSZ here and
// no pixel geometry to go with it, so the size is always estimated from the
// grid.
func termSize(f *os.File) (int, int, int, int) {
	cols, rows, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0, 0, 0, 0
	}
	return cols, rows, 0, 0
}
