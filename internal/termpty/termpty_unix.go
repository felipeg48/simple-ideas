//go:build !windows

// Package termpty runs a program under a pseudo-terminal: creack/pty on Unix,
// ConPTY on Windows. It is what the si play terminal drawer and the web demo
// terminal scenes share, and never part of a built deck (CLAUDE.md rule #8).
package termpty

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// Start starts c under a pseudo-terminal of cols×rows (0×0: the system's
// default) and returns the terminal's other end: what c prints is read from
// it, what is written to it is c's input.
func Start(c *exec.Cmd, cols, rows int) (io.ReadWriteCloser, error) {
	if cols <= 0 || rows <= 0 {
		return pty.Start(c)
	}
	return pty.StartWithSize(c, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Resize sets the pseudo-terminal's size.
func Resize(p io.ReadWriteCloser, cols, rows int) {
	if f, ok := p.(*os.File); ok {
		pty.Setsize(f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}
