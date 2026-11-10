//go:build !windows

package main

import (
	"os"
	"os/exec"
)

// noEcho turns the terminal's echo off (stty) and returns what turns it on.
func noEcho(f *os.File) func() {
	c := exec.Command("stty", "-echo")
	c.Stdin = f
	if c.Run() != nil {
		return func() {}
	}
	return func() {
		c := exec.Command("stty", "echo")
		c.Stdin = f
		c.Run()
	}
}
