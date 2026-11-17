package cdp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Command is headless Chrome with args, in a fresh profile folder of its own
// (two runs never share one), bounded by timeout from when it gets one of
// the machine-wide Slots (slots.go): on a busy machine (a full
// go test ./..., a site built while a deck films) every Chrome started
// together starves the others of CPU, and their waits run out. The caller
// runs it (Output, CombinedOutput) and then calls done, which frees the slot
// and removes the profile.
func Command(timeout time.Duration, chrome string, args ...string) (cmd *exec.Cmd, done func()) {
	cmd, done, _ = command(timeout, chrome, args...)
	return cmd, done
}

func command(timeout time.Duration, chrome string, args ...string) (cmd *exec.Cmd, done func(), wait time.Duration) {
	s, wait := acquire()
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
		s.release()
	}, wait
}

// Run runs Command and returns its combined output. A Chrome that dies on
// start (an exit status, not the timeout: a busy machine makes it crash now
// and then) gets one more try, in a fresh profile; so does a run stopped by
// its timeout before it wrote its file, when the machine is loaded.
func Run(timeout time.Duration, chrome string, args ...string) ([]byte, error) {
	return twice(timeout, func() try {
		cmd, done, wait := command(timeout, chrome, args...)
		defer done()
		start := time.Now()
		out, err := cmd.CombinedOutput()
		if err != nil && !timedOut(cmd) && wrote(args, start) {
			err = nil // the picture or PDF is written: Chrome crashed only on its way out
		}
		return try{out, err, cmd.ProcessState != nil && !timedOut(cmd), timedOut(cmd), wait}
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

// try is one Chrome run: its output and error, whether it crashed (an exit
// status) or was stopped by its timeout, and how long it waited for a slot.
type try struct {
	out               []byte
	err               error
	crashed, timedOut bool
	wait              time.Duration
}

// loaded reports whether the machine is busy with other Chromes: this run
// waited for its slot, or other runs hold slots now. A timeout then is the
// load, not the page, and earns one more try.
var loaded = func(wait time.Duration) bool { return wait > time.Second || busySlots() > 0 }

func twice(timeout time.Duration, run func() try) ([]byte, error) {
	t := run()
	tries, wait := 1, t.wait
	if t.err != nil && (t.crashed || t.timedOut && loaded(t.wait)) {
		t = run()
		tries, wait = 2, wait+t.wait
	}
	if t.err != nil && t.timedOut {
		return t.out, fmt.Errorf("headless Chrome took longer than %s (%d tries, %s waited for one of the %d Chrome slots on this machine): %w", timeout, tries, wait.Round(time.Millisecond), Slots, t.err)
	}
	if t.err != nil && wait > time.Second {
		return t.out, fmt.Errorf("%w (after %s waiting for a Chrome slot)", t.err, wait.Round(time.Millisecond))
	}
	return t.out, t.err
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
	return twice(timeout, func() try {
		cmd, done, wait := command(timeout, chrome, args...)
		defer done()
		out, err := cmd.Output()
		if err != nil && !timedOut(cmd) && hasFlag(args, "--dump-dom") && bytes.Contains(out, []byte("</html>")) {
			err = nil // the whole DOM is out: Chrome crashed only on its way out
		}
		return try{out, err, cmd.ProcessState != nil && !timedOut(cmd), timedOut(cmd), wait}
	})
}
