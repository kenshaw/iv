package ivctx

import (
	"os"
	"testing"
)

// TestScale checks the sizing policy: an image between the minimum size and
// the display size keeps its own size, one above the display size is shrunk to
// it, and one below the minimum is grown up to it.
func TestScale(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       Mode
		maxW, maxH uint
		minW, minH uint
		w, h       int
		expW, expH int
	}{
		// nothing configured: an image is its own size, whatever it is
		{"unbounded", ModeBestFit, 0, 0, 64, 64, 800, 600, 800, 600},
		{"unbounded tiny", ModeBestFit, 0, 0, 64, 64, 16, 16, 64, 64},

		// between the two, an image is left alone -- no resampling at all
		{"inside the band", ModeBestFit, 1000, 800, 64, 64, 800, 600, 800, 600},
		{"exactly the display size", ModeBestFit, 800, 600, 64, 64, 800, 600, 800, 600},

		// above the display size, shrunk to it, aspect kept
		{"too wide", ModeBestFit, 400, 400, 64, 64, 800, 600, 400, 300},
		{"too tall", ModeBestFit, 400, 400, 64, 64, 600, 800, 300, 400},
		{"huge", ModeBestFit, 1000, 500, 64, 64, 6000, 4000, 750, 500},

		// below the minimum, grown up to it
		{"icon", ModeBestFit, 1000, 500, 64, 64, 16, 16, 64, 64},
		{"one pixel", ModeBestFit, 1000, 500, 64, 64, 1, 1, 64, 64},

		// whichever dimension reaches its minimum first decides: growing this
		// until the height cleared 64 would make it 32000 wide
		{"wide banner", ModeBestFit, 0, 0, 64, 64, 1000, 2, 1000, 2},
		{"narrow banner", ModeBestFit, 0, 0, 64, 64, 2, 1000, 2, 1000},

		// the display size is the hard bound: a minimum larger than the room
		// there is to show it in does not win
		{"minimum over display size", ModeBestFit, 32, 32, 64, 64, 16, 16, 32, 32},

		// no minimum at all, so a small image stays small
		{"no minimum", ModeBestFit, 400, 400, 0, 0, 16, 16, 16, 16},

		// modes
		{"none", ModeNone, 100, 100, 64, 64, 800, 600, 800, 600},
		{"none leaves an icon alone", ModeNone, 100, 100, 64, 64, 16, 16, 16, 16},
		{"width ignores the height", ModeWidth, 400, 100, 64, 64, 800, 600, 400, 300},
		{"height ignores the width", ModeHeight, 100, 400, 64, 64, 800, 600, 533, 400},
		{"shrink has no floor", ModeShrink, 400, 400, 64, 64, 16, 16, 16, 16},
		{"shrink still shrinks", ModeShrink, 400, 400, 64, 64, 800, 600, 400, 300},
		{"stretch fills exactly", ModeStretch, 300, 100, 64, 64, 800, 600, 300, 100},
		// nothing to stretch to with one side unset, so it best fits instead
		{"stretch with one side", ModeStretch, 300, 0, 64, 64, 800, 600, 300, 225},

		// degenerate input is handed back untouched
		{"zero", ModeBestFit, 400, 400, 64, 64, 0, 0, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := New()
			c.Mode, c.Width, c.Height = &test.mode, test.maxW, test.maxH
			c.MinWidth, c.MinHeight = test.minW, test.minH
			w, h := c.Scale(test.w, test.h)
			if w != test.expW || h != test.expH {
				t.Errorf("expected %dx%d, got %dx%d", test.expW, test.expH, w, h)
			}
		})
	}
}

// TestScaleForcedMinimum checks that a minimum given on the command line is a
// hard floor (D37). The image grows until it reaches that minimum, even when
// it already clears the other one, and the display size still bounds it.
func TestScaleForcedMinimum(t *testing.T) {
	for _, test := range []struct {
		name           string
		mode           Mode
		maxW, maxH     uint
		minW, minH     uint
		forceW, forceH bool
		w, h           int
		expW, expH     int
	}{
		// the default minimum height does not stop a given minimum width
		{"width", ModeBestFit, 0, 0, 1000, 64, true, false, 367, 344, 1000, 937},
		{"height", ModeBestFit, 0, 0, 64, 1000, false, true, 367, 344, 1067, 1000},
		// both given, so both are reached
		{"both", ModeBestFit, 0, 0, 1000, 1000, true, true, 367, 344, 1067, 1000},
		// a forced banner grows as far as it was asked to
		{"wide banner", ModeBestFit, 0, 0, 64, 64, false, true, 1000, 2, 32000, 64},
		// an image that already clears a forced minimum is left alone
		{"already wide enough", ModeBestFit, 0, 0, 100, 64, true, false, 367, 344, 367, 344},
		// the display size is still the hard bound
		{"display size wins", ModeBestFit, 800, 600, 1000, 64, true, false, 367, 344, 640, 600},
		// a mode that drops a floor drops a forced one too
		{"shrink", ModeShrink, 0, 0, 1000, 64, true, false, 367, 344, 367, 344},
		{"height mode drops the width", ModeHeight, 0, 0, 1000, 64, true, false, 367, 344, 367, 344},
		// a forced zero is no floor at all
		{"zero", ModeBestFit, 0, 0, 0, 64, true, false, 16, 16, 64, 64},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := New()
			c.Mode, c.Width, c.Height = &test.mode, test.maxW, test.maxH
			c.MinWidth, c.MinHeight = test.minW, test.minH
			c.MinWidthSet, c.MinHeightSet = test.forceW, test.forceH
			w, h := c.Scale(test.w, test.h)
			if w != test.expW || h != test.expH {
				t.Errorf("expected %dx%d, got %dx%d", test.expW, test.expH, w, h)
			}
		})
	}
}

// TestPageHeight checks the height of a page: the height that the display
// height allows at the scale that fits the width, so that each page is shown
// at the full width.
func TestPageHeight(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       Mode
		maxW, maxH uint
		w          int
		exp        int
	}{
		{"wider than the display", ModeBestFit, 1200, 800, 2400, 1600},
		{"narrower than the display", ModeBestFit, 3000, 800, 2400, 800},
		{"no display width", ModeBestFit, 0, 800, 2400, 800},
		{"no display height", ModeBestFit, 1200, 0, 2400, 0},
		{"no display size", ModeBestFit, 0, 0, 2400, 0},
		{"shrink", ModeShrink, 1200, 800, 2400, 1600},
		{"stretch", ModeStretch, 1200, 800, 2400, 1600},
		{"height mode ignores the width", ModeHeight, 1200, 800, 2400, 800},
		{"width mode has no pages", ModeWidth, 1200, 800, 2400, 0},
		{"none has no pages", ModeNone, 1200, 800, 2400, 0},
		{"no image", ModeBestFit, 1200, 800, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := New()
			c.Mode, c.Width, c.Height = &test.mode, test.maxW, test.maxH
			got := c.PageHeight(test.w)
			if got != test.exp {
				t.Fatalf("expected %d, got %d", test.exp, got)
			}
			// a page of this height is shown at the full width that fits
			if got == 0 || test.mode == ModeHeight || test.mode == ModeStretch {
				return
			}
			fitW, _ := c.Scale(test.w, 1)
			if w, h := c.Scale(test.w, got); w != fitW || h > int(test.maxH) {
				t.Errorf("expected a page shown at %dx%d or less, got %dx%d", fitW, test.maxH, w, h)
			}
		})
	}
}

// TestScaleSnapsSmallImages checks a small image is magnified by a whole
// number of pixels. A 16x16 icon resampled up to 61x61 is mush, where the same
// icon at 4x is its own pixels drawn larger.
func TestScaleSnapsSmallImages(t *testing.T) {
	c := New()
	c.MinWidth, c.MinHeight = 61, 61
	w, h := c.Scale(16, 16)
	if w != 64 || h != 64 {
		t.Errorf("expected the factor rounded up to 4x, 64x64, got %dx%d", w, h)
	}
	// unless the whole factor would not fit, where the exact size wins over a
	// crisper one that does not go on the screen
	c.Width, c.Height = 62, 62
	if w, h = c.Scale(16, 16); w != 61 || h != 61 {
		t.Errorf("expected 61x61, got %dx%d", w, h)
	}
}

// TestScaleIsIdempotent checks a size that came out of the scale goes back in
// unchanged. A renderer that rasterizes at the displayed size hands its result
// to the same scale on the way to the encoder, and a second pass that moved it
// again would resample for nothing.
func TestScaleIsIdempotent(t *testing.T) {
	for _, mode := range Modes() {
		t.Run(string(mode), func(t *testing.T) {
			c := New()
			c.Mode, c.Width, c.Height = &mode, 640, 480
			for _, size := range [][2]int{{6000, 4000}, {16, 16}, {800, 600}, {1, 1}, {1000, 2}} {
				w, h := c.Scale(size[0], size[1])
				w2, h2 := c.Scale(w, h)
				if w != w2 || h != h2 {
					t.Errorf("%dx%d scaled to %dx%d, then to %dx%d", size[0], size[1], w, h, w2, h2)
				}
			}
		})
	}
}

// TestMode checks the mode parses from and prints back as its name, which is
// how it reaches iv from the command line.
func TestMode(t *testing.T) {
	for _, mode := range Modes() {
		var got Mode
		if err := got.UnmarshalText([]byte(string(mode))); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != mode {
			t.Errorf("expected %q, got %q", mode, got)
		}
		buf, err := mode.MarshalText()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if string(buf) != string(mode) {
			t.Errorf("expected %q, got %q", mode, buf)
		}
	}
	// case and surrounding space are not the user's problem
	var got Mode
	if err := got.UnmarshalText([]byte("  Best-Fit ")); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != ModeBestFit {
		t.Errorf("expected %q, got %q", ModeBestFit, got)
	}
	if err := got.UnmarshalText([]byte("enormous")); err == nil {
		t.Error("expected an error")
	} else if got != ModeBestFit {
		t.Errorf("expected the mode left alone on an error, got %q", got)
	}
}

// TestTermCellSize checks the reported cell size, and the fallback for a
// terminal that reports a grid but no pixels.
func TestTermCellSize(t *testing.T) {
	for _, test := range []struct {
		name string
		term Term
		expW int
		expH int
	}{
		{"reported", Term{Cols: 100, Rows: 50, Width: 1000, Height: 1000, Exact: true}, 10, 20},
		{"estimated", Term{Cols: 100, Rows: 50, Width: 800, Height: 800}, CellWidth, CellHeight},
		{"empty", Term{}, CellWidth, CellHeight},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, h := test.term.CellSize()
			if w != test.expW || h != test.expH {
				t.Errorf("expected %dx%d, got %dx%d", test.expW, test.expH, w, h)
			}
		})
	}
}

// TestTermSizeNotATerminal checks a file that is not a terminal reports no
// geometry, rather than a zero one that would scale everything to nothing.
func TestTermSizeNotATerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "iv")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer f.Close()
	if _, ok := TermSize(f); ok {
		t.Error("expected a regular file to report no terminal geometry")
	}
	if _, ok := TermSize(nil); ok {
		t.Error("expected a nil file to report no terminal geometry")
	}
}
