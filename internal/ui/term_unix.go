//go:build unix

package ui

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func vt(*os.File) bool { return true }

// Keys puts the terminal in f into cbreak mode (keys arrive one at a time,
// not echoed; Ctrl-C still interrupts) and returns what restores it.
func Keys(f *os.File) func() {
	save := exec.Command("stty", "-g")
	save.Stdin = f
	was, err := save.Output()
	if err != nil {
		return func() {}
	}
	set := exec.Command("stty", "-icanon", "-echo", "min", "1", "time", "0")
	set.Stdin = f
	if set.Run() != nil {
		return func() {}
	}
	return func() {
		c := exec.Command("stty", strings.TrimSpace(string(was)))
		c.Stdin = f
		c.Run()
	}
}

// CPUTime is the CPU time this process has used, user and system.
func CPUTime() time.Duration {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
