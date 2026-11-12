package pinned

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Trust decides whether code from elsewhere may run: a pin trusted before
// (with the same contents) runs; else a person at a terminal is shown the
// source and asked once, and a yes is remembered; --trust says yes for one
// run (CI); without a terminal and without --trust, si refuses and says why.
type Trust struct {
	Store string    // the file of trusted pins (~/.si/trusted): "<pin> <sha256>" per line
	Yes   bool      // --trust: yes for this run, remembered nowhere
	Ask   bool      // a person can answer (stdin is a terminal)
	In    io.Reader // where the answer comes from
	Out   io.Writer // where the source is shown

	mu sync.Mutex
	rd *bufio.Reader
}

// Terminal reports whether stdin is a terminal a person can answer at (not
// a pipe, a file or /dev/null, a character device too).
func Terminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if null, err := os.Stat(os.DevNull); err == nil && os.SameFile(fi, null) {
		return false
	}
	return true
}

func (t *Trust) load() map[string]string {
	known := map[string]string{}
	b, err := os.ReadFile(t.Store)
	if err != nil {
		return known
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if i := strings.LastIndexByte(ln, ' '); i > 0 && !strings.HasPrefix(ln, "#") {
			known[ln[:i]] = ln[i+1:]
		}
	}
	return known
}

// Check lets the code at key, whose contents hash to sum, run, or says why
// not. show writes what the person is asked about: where it comes from, its
// size, its first lines.
func (t *Trust) Check(key, sum string, show func(io.Writer)) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if was, ok := t.load()[key]; ok {
		if was == sum {
			return nil
		}
		return fmt.Errorf("%s changed since it was trusted (sha256 %s… then, %s… now): the pin no longer matches, so it does not run; pin the new revision to review it again", key, short(was), short(sum))
	}
	if t.Yes {
		return nil
	}
	if !t.Ask {
		return fmt.Errorf("%s is code from elsewhere that was never trusted on this machine, and there is no terminal to ask in: run it once in a terminal to review it, or pass --trust (CI)", key)
	}
	out := t.Out
	if out == nil {
		out = io.Discard
	}
	fmt.Fprintf(out, "\nsi runs code from elsewhere:\n")
	if show != nil {
		show(out)
	}
	fmt.Fprintf(out, "Trust %s and run it (remembered for this pin)? [y/N] ", key)
	if t.rd == nil {
		in := t.In
		if in == nil {
			in = os.Stdin
		}
		t.rd = bufio.NewReader(in)
	}
	ans, _ := t.rd.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "y", "yes":
	default:
		return fmt.Errorf("%s: not trusted, so it does not run", key)
	}
	if err := os.MkdirAll(filepath.Dir(t.Store), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(t.Store, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s\n", key, sum)
	return err
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// Preview writes a file for the trust question: its name, its size and its
// first n lines.
func Preview(w io.Writer, name string, b []byte, n int) {
	fmt.Fprintf(w, "  %s (%d bytes)\n", name, len(b))
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	for i, ln := range lines {
		if i == n {
			fmt.Fprintf(w, "    … %d more line(s)\n", len(lines)-n)
			break
		}
		if len(ln) > 120 {
			ln = ln[:120] + "…"
		}
		fmt.Fprintf(w, "    │ %s\n", ln)
	}
}
