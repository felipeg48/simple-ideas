package cdp

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// slots limits how many one-shot headless Chrome runs (a screenshot, a PDF,
// a --dump-dom) this process has at once. On a busy machine (a full
// go test ./..., a site built while a deck films) every Chrome started
// together starves the others of CPU, and their waits run out. A run waits
// for a slot before its timeout starts.
var slots = make(chan struct{}, max(2, runtime.NumCPU()/4))

// Command is headless Chrome with args, in a fresh profile folder of its own
// (two runs never share one), bounded by timeout from when it gets a slot.
// The caller runs it (Output, CombinedOutput) and then calls done, which
// frees the slot and removes the profile.
func Command(timeout time.Duration, chrome string, args ...string) (cmd *exec.Cmd, done func()) {
	slots <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	profile, err := os.MkdirTemp("", "si-chrome-")
	if err == nil && !hasFlag(args, "--user-data-dir") {
		args = append([]string{"--user-data-dir=" + profile}, args...)
	}
	cmd = exec.CommandContext(ctx, chrome, args...)
	cmd.WaitDelay = 5 * time.Second // a child Chrome keeping the pipes open never hangs the caller
	return cmd, func() {
		cancel()
		if profile != "" {
			os.RemoveAll(profile)
		}
		<-slots
	}
}

// Run runs Command and returns its combined output. A Chrome that dies on
// start (an exit status, not the timeout: a busy machine makes it crash now
// and then) gets one more try, in a fresh profile.
func Run(timeout time.Duration, chrome string, args ...string) ([]byte, error) {
	return twice(func() ([]byte, error, bool) {
		cmd, done := Command(timeout, chrome, args...)
		defer done()
		start := time.Now()
		out, err := cmd.CombinedOutput()
		if err != nil && !timedOut(cmd) && wrote(args, start) {
			err = nil // the picture or PDF is written: Chrome crashed only on its way out
		}
		return out, err, cmd.ProcessState != nil && !timedOut(cmd)
	})
}

// wrote reports whether the --screenshot or --print-to-pdf file of args was
// written since start. Under load headless Chrome now and then writes it and
// then dies in teardown ("Teardown watchdog expired", exit status 2).
func wrote(args []string, start time.Time) bool {
	for _, a := range args {
		for _, f := range []string{"--screenshot=", "--print-to-pdf="} {
			if p, ok := strings.CutPrefix(a, f); ok {
				fi, err := os.Stat(p)
				return err == nil && fi.Size() > 0 && !fi.ModTime().Before(start.Add(-time.Second))
			}
		}
	}
	return false
}

func twice(run func() ([]byte, error, bool)) ([]byte, error) {
	out, err, again := run()
	if err != nil && again {
		out, err, _ = run()
	}
	return out, err
}

// timedOut: the run was stopped by its timeout.
func timedOut(cmd *exec.Cmd) bool {
	return cmd.Cancel != nil && cmd.Err == nil && cmd.ProcessState != nil && !cmd.ProcessState.Exited()
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return true
		}
	}
	return false
}

// profileDir is a fresh profile folder for a DevTools Chrome (Launch), or ""
// when the caller named one.
func profileDir(args []string) string {
	if hasFlag(args, "--user-data-dir") {
		return ""
	}
	p, err := os.MkdirTemp("", "si-chrome-")
	if err != nil {
		return ""
	}
	return p
}

// Output runs Command and returns its standard output (a --dump-dom).
func Output(timeout time.Duration, chrome string, args ...string) ([]byte, error) {
	return twice(func() ([]byte, error, bool) {
		cmd, done := Command(timeout, chrome, args...)
		defer done()
		out, err := cmd.Output()
		if err != nil && !timedOut(cmd) && hasFlag(args, "--dump-dom") && bytes.Contains(out, []byte("</html>")) {
			err = nil // the whole DOM is out: Chrome crashed only on its way out
		}
		return out, err, cmd.ProcessState != nil && !timedOut(cmd)
	})
}
