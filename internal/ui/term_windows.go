//go:build windows

package ui

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// si writes UTF-8 (— ✓ … in its plain lines too); a console left on an OEM
// code page would show them as garbage, so the console is told to read UTF-8.
func init() {
	if isTerminal(os.Stdout) || isTerminal(os.Stderr) {
		windows.SetConsoleOutputCP(65001) // CP_UTF8
	}
}

func isTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

// vt turns on ANSI escape processing for a console output handle.
func vt(f *os.File) bool {
	h := windows.Handle(f.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return false
	}
	return windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}

// Keys turns line input and echo off on the console in f (Ctrl-C still
// interrupts) and returns what restores it.
func Keys(f *os.File) func() {
	h := windows.Handle(f.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return func() {}
	}
	windows.SetConsoleMode(h, mode&^(windows.ENABLE_LINE_INPUT|windows.ENABLE_ECHO_INPUT))
	return func() { windows.SetConsoleMode(h, mode) }
}

// CPUTime is the CPU time this process has used, user and kernel.
func CPUTime() time.Duration {
	var c, e, k, u windows.Filetime
	if windows.GetProcessTimes(windows.CurrentProcess(), &c, &e, &k, &u) != nil {
		return 0
	}
	ticks := func(ft windows.Filetime) int64 { return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime) }
	return time.Duration((ticks(k) + ticks(u)) * 100) // 100 ns ticks
}
