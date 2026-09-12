//go:build unix

package ivctx

import (
	"os"

	"golang.org/x/sys/unix"
)

// termSize returns the terminal's cell grid and pixel size as the kernel
// reports them. The pixel fields come back zero from a terminal that does not
// fill them in, which many do not.
func termSize(f *os.File) (int, int, int, int) {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, 0, 0
	}
	return int(ws.Col), int(ws.Row), int(ws.Xpixel), int(ws.Ypixel)
}
