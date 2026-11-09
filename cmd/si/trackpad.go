package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// startTrackpad runs the si-trackpad helper (desktop/trackpad, macOS) against
// this si play: it posts "pad" messages to the loopback relay, and the deck
// page turns them into a pen. The returned func stops it with SIGINT, so the
// helper gives the cursor back.
func startTrackpad(addr net.Addr, stdout io.Writer) (func(), error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("the trackpad pen needs macOS")
	}
	ta, ok := addr.(*net.TCPAddr)
	if !ok || !ta.IP.IsLoopback() {
		return nil, errors.New("use it with si play on a loopback address (the default)")
	}
	bin := ""
	if exe, err := os.Executable(); err == nil {
		// si is often a symlink (~/bin/si -> ~/.bin/si): look beside the link and beside the real file
		dirs := []string{filepath.Dir(exe)}
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			dirs = append(dirs, filepath.Dir(real))
		}
		for _, d := range dirs {
			if p := filepath.Join(d, "si-trackpad"); bin == "" && fileExists(p) {
				bin = p
			}
		}
	}
	if bin == "" {
		if p, err := exec.LookPath("si-trackpad"); err == nil {
			bin = p
		}
	}
	if bin == "" {
		return nil, errors.New("si-trackpad is not installed: build it with make trackpad (or install it next to si)")
	}
	cmd := exec.Command(bin, "--url", "http://"+addr.String())
	cmd.Stdout, cmd.Stderr = io.Discard, os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	fmt.Fprintln(stdout, "  trackpad: the trackpad is the display the deck is on; hold ⌥ Option and draw with one finger (no need to press a)")
	return func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}, nil
}

func fileExists(p string) bool { fi, err := os.Stat(p); return err == nil && !fi.IsDir() }
