package ivctx

import "os"

// Cell is the assumed size of a terminal character cell in pixels, used when a
// terminal reports its cell grid but not its pixel size.
//
// The terminals that draw graphics on a high density display -- kitty, iTerm2,
// wezterm, ghostty, foot -- all report pixels, so this is only reached on the
// simpler ones, where a cell really is about this big.
const (
	CellWidth  = 8
	CellHeight = 16
)

// Term is the geometry of a terminal, in cells and in pixels.
type Term struct {
	// Cols and Rows are the character cell grid.
	Cols, Rows int
	// Width and Height are the drawable area, in pixels.
	Width, Height int
	// Exact indicates the terminal reported its pixel size itself, rather
	// than it being estimated from the cell grid at [CellWidth] x
	// [CellHeight].
	Exact bool
}

// CellSize returns the size of one character cell, in pixels.
func (t Term) CellSize() (int, int) {
	w, h := CellWidth, CellHeight
	if t.Cols > 0 && t.Rows > 0 && t.Exact {
		w, h = t.Width/t.Cols, t.Height/t.Rows
	}
	return max(w, 1), max(h, 1)
}

// TermSize returns the geometry of the terminal on f, and whether f is a
// terminal that reported one at all.
func TermSize(f *os.File) (Term, bool) {
	if f == nil {
		return Term{}, false
	}
	cols, rows, x, y := termSize(f)
	t := Term{
		Cols:   cols,
		Rows:   rows,
		Width:  x,
		Height: y,
		Exact:  x > 0 && y > 0,
	}
	if !t.Exact {
		t.Width, t.Height = cols*CellWidth, rows*CellHeight
	}
	if t.Width <= 0 || t.Height <= 0 {
		return Term{}, false
	}
	return t, true
}
