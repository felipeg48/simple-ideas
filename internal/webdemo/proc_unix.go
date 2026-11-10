//go:build !windows

package webdemo

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// ownGroup puts a background program in its own process group, so stopping
// it stops what it started too.
func ownGroup(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// signalGroup asks the group led by pid to stop: SIGTERM, and SIGHUP too for
// a terminal (an interactive shell ignores SIGTERM; its programs hang up).
func signalGroup(pid int, hup bool) {
	if hup {
		syscall.Kill(-pid, syscall.SIGHUP)
	}
	syscall.Kill(-pid, syscall.SIGTERM)
}

// killGroup kills every process left in the group led by pid.
func killGroup(pid int) { syscall.Kill(-pid, syscall.SIGKILL) }

// startShell starts the terminal's shell under a pseudo-terminal of cols×rows
// (the shell leads a new session; its programs share its process group, as
// job control is off in si's prompt).
func startShell(c *exec.Cmd, cols, rows int) (io.ReadWriteCloser, bool, error) {
	f, err := pty.StartWithSize(c, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	return f, false, err
}

// resizeShell sets the pseudo-terminal's size.
func resizeShell(f io.ReadWriteCloser, cols, rows int) {
	if p, ok := f.(*os.File); ok {
		pty.Setsize(p, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}
