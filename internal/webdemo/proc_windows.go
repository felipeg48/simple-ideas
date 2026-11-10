//go:build windows

package webdemo

import (
	"io"
	"os/exec"
	"strconv"
	"syscall"
)

func ownGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// signalGroup asks the program and its children to stop (taskkill without /F
// sends them a close request).
func signalGroup(pid int, hup bool) {
	exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid)).Run()
}

func killGroup(pid int) { exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run() }

// pipes is a shell's stdin and its stdout and stderr together.
type pipes struct {
	r  *io.PipeReader
	w  *io.PipeWriter
	in io.WriteCloser
}

func (p pipes) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p pipes) Write(b []byte) (int, error) { return p.in.Write(b) }
func (p pipes) Close() error                { p.in.Close(); return p.w.Close() }

// startShell runs the shell on pipes: no pseudo-terminal here, so no
// colours, and si echoes what it types itself (pipes is true). The caller
// waits for the shell and closes the pipes after.
func startShell(c *exec.Cmd, cols, rows int) (io.ReadWriteCloser, bool, error) {
	in, err := c.StdinPipe()
	if err != nil {
		return nil, true, err
	}
	r, w := io.Pipe()
	c.Stdout, c.Stderr = w, w
	if err := c.Start(); err != nil {
		return nil, true, err
	}
	return pipes{r, w, in}, true, nil
}

func resizeShell(f io.ReadWriteCloser, cols, rows int) {}
