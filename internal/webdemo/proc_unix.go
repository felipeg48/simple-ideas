//go:build !windows

package webdemo

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/felipeg48/simple-ideas/internal/termpty"
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
	f, err := termpty.Start(c, cols, rows)
	return f, false, err
}

// resizeShell sets the pseudo-terminal's size.
func resizeShell(f io.ReadWriteCloser, cols, rows int) { termpty.Resize(f, cols, rows) }

// DefaultShell is the user's shell: $SHELL, else /bin/sh.
func DefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

// shellCommand is the terminal's shell (shell:, else DefaultShell) with si's
// prompt from rc.
func shellCommand(shell, dir, rc string) *exec.Cmd {
	if shell == "" {
		shell = DefaultShell()
	}
	env := append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "BASH_SILENCE_DEPRECATION_WARNING=1")
	var c *exec.Cmd
	switch strings.TrimSuffix(filepath.Base(shell), ".exe") {
	case "zsh":
		c = exec.Command(shell, "-d", "-i")
		env = append(env, "ZDOTDIR="+rc)
	case "bash":
		c = exec.Command(shell, "--noprofile", "--rcfile", filepath.Join(rc, "bashrc"), "-i")
	default:
		c = exec.Command(shell, "-i")
		env = append(env, "ENV="+filepath.Join(rc, "shrc"))
	}
	c.Dir, c.Env = dir, env
	return c
}
