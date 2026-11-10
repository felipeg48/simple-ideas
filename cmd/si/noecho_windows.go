//go:build windows

package main

import (
	"os"
	"syscall"
)

// noEcho turns the console's echo off and returns what turns it on.
func noEcho(f *os.File) func() {
	h := syscall.Handle(f.Fd())
	var mode uint32
	if syscall.GetConsoleMode(h, &mode) != nil {
		return func() {}
	}
	set := syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")
	const echoInput = 0x0004 // ENABLE_ECHO_INPUT
	set.Call(uintptr(h), uintptr(mode&^echoInput))
	return func() { set.Call(uintptr(h), uintptr(mode)) }
}
