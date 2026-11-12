package webdemo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/pinned"
)

// The run's life (spec docs/superpowers/specs/2026-11-12-demo-lifecycle.md):
// start:, prepare: and step: in the order written, once; ## Before; for each
// feature its setup: lines, then the feature; ## After; stop: (the programs
// still run); every program stopped; cleanup:. stop: and cleanup: always run
// once the run has begun: after a pass, a failure, a failed prepare:, Ctrl-C
// and a panic.

// Command is one front matter prepare:, step:, setup:, stop: or cleanup:
// line: a shell command, or a source (gist:, github:, a pinned URL) and its
// arguments.
type Command struct {
	Key    string         // as written: prepare, step, setup, stop, cleanup
	Text   string         // the value as written
	Cmds   []string       // what runs, in order: the command, or each fetched script with Args
	Source *pinned.Source // a remote source (nil: a local command)
	Args   string         // the arguments after a source
	Dir    string         // · in <dir>: where it runs, relative to the script ("" = the script's folder)
	Within time.Duration  // · within 5m: how long it may take (0 = the default of its key)
	Line   int
}

// Prep is one line of the run's start, in the order written: a program
// (start:) or a command that must finish (prepare:, step:; a deck's setup:).
type Prep struct {
	Start *Program
	Cmd   *Command
}

// dropComment cuts a trailing ` # comment` outside quotes.
func dropComment(v string) string {
	q := byte(0)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case q != 0:
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			q = c
		case c == '#' && i > 0 && (v[i-1] == ' ' || v[i-1] == '\t'):
			return strings.TrimSpace(v[:i])
		}
	}
	return strings.TrimSpace(v)
}

// lifecycle reads one start:, prepare:, step:, setup:, stop: or cleanup:
// line. In a deck (si play), setup: must finish before the talk, in order
// with start:; in a demo it runs before each feature.
func (s *Script) lifecycle(key, val string, line int, isDeck bool) {
	val = dropComment(val)
	if key == "start" {
		if p := s.start(val, line); p != nil {
			s.Prelude = append(s.Prelude, Prep{Start: p})
		}
		return
	}
	c := s.command(key, val, line)
	if c == nil {
		return
	}
	switch {
	case key == "prepare" || key == "step" || (isDeck && key == "setup"):
		s.Prelude = append(s.Prelude, Prep{Cmd: c})
	case key == "setup":
		s.Setups = append(s.Setups, c)
	case key == "stop":
		s.Stops = append(s.Stops, c)
	case key == "cleanup":
		s.Cleanups = append(s.Cleanups, c)
	}
}

// command reads `<command or source> [args] · within 5m · in dir · sha256 HEX`.
func (s *Script) command(key, val string, line int) *Command {
	parts := clauses(val)
	c := &Command{Key: key, Text: val, Line: line}
	head := strings.TrimSpace(parts[0])
	sum := ""
	for _, p := range parts[1:] {
		w, rest := cutWord(p)
		switch w {
		case "within":
			d, ok := duration(rest)
			if !ok || rest == "" || d <= 0 {
				s.problem(line, "%s: %q: write · within 5m (or 90s)", key, p)
				return nil
			}
			c.Within = d
		case "in":
			if rest == "" {
				s.problem(line, "%s: write · in <folder> (relative to the script)", key)
				return nil
			}
			c.Dir = unquote(rest)
		case "sha256":
			sum = rest
		default:
			s.problem(line, "%s: %q: after ·, write within 5m, in <folder> or sha256 <hex> (a URL's fingerprint)", key, p)
			return nil
		}
	}
	if head == "" {
		s.problem(line, "%s: needs a command (%s: ./seed.sh) or a source (gist:ID@REV, github:owner/repo@REF/path, a URL · sha256 …)", key, key)
		return nil
	}
	src, args, err := source(head, sum)
	if err != nil {
		s.problem(line, "%s: %v", key, err)
		return nil
	}
	if src == nil {
		if sum != "" {
			s.problem(line, "%s: · sha256 is the fingerprint of a URL source", key)
			return nil
		}
		c.Cmds = []string{head}
	}
	c.Source, c.Args = src, args
	return c
}

// source reads a remote source at the start of head (and the arguments after
// it); src is nil for a local command.
func source(head, sum string) (*pinned.Source, string, error) {
	w, args := cutWord(head)
	if !pinned.IsSource(w) {
		return nil, "", nil
	}
	src, err := pinned.Parse(w, sum)
	return src, args, err
}

// Sources are the remote sources the script's lines name, in order.
func (s *Script) Sources() []*pinned.Source {
	var out []*pinned.Source
	for _, p := range s.Prelude {
		if p.Start != nil && p.Start.Source != nil {
			out = append(out, p.Start.Source)
		}
		if p.Cmd != nil && p.Cmd.Source != nil {
			out = append(out, p.Cmd.Source)
		}
	}
	for _, cs := range [][]*Command{s.Setups, s.Stops, s.Cleanups} {
		for _, c := range cs {
			if c.Source != nil {
				out = append(out, c.Source)
			}
		}
	}
	return out
}

// Resolve fetches every remote source with get (pinned.Client.Scripts: the
// pin checked, trust asked) and sets what runs: each fetched script, quoted,
// with the arguments.
func (s *Script) Resolve(get func(*pinned.Source) ([]string, error)) error {
	run := func(paths []string, args string) []string {
		var out []string
		for _, p := range paths {
			out = append(out, strings.TrimSpace(shellQuote(p)+" "+args))
		}
		return out
	}
	cmds := append(append(append([]*Command(nil), s.Setups...), s.Stops...), s.Cleanups...)
	for _, p := range s.Prelude {
		if p.Cmd != nil {
			cmds = append(cmds, p.Cmd)
		}
		if st := p.Start; st != nil && st.Source != nil && !st.fetched {
			paths, err := get(st.Source)
			if err != nil {
				return err
			}
			if len(paths) != 1 {
				return fmt.Errorf("start: %s holds %d scripts: a program is one script", st.Source.Key(), len(paths))
			}
			st.Cmd, st.fetched = run(paths, st.Args)[0], true
		}
	}
	for _, c := range cmds {
		if c.Source == nil || c.Cmds != nil {
			continue
		}
		paths, err := get(c.Source)
		if err != nil {
			return err
		}
		c.Cmds = run(paths, c.Args)
	}
	return nil
}

// unresolved names a remote source not fetched yet ("" = none).
func (s *Script) unresolved() string {
	for _, p := range s.Prelude {
		if p.Start != nil && p.Start.Source != nil && !p.Start.fetched {
			return p.Start.Source.Key()
		}
		if p.Cmd != nil && p.Cmd.Cmds == nil {
			return p.Cmd.Source.Key()
		}
	}
	for _, cs := range [][]*Command{s.Setups, s.Stops, s.Cleanups} {
		for _, c := range cs {
			if c.Cmds == nil {
				return c.Source.Key()
			}
		}
	}
	return ""
}

// shellQuote quotes a path for sh -c (cmd /C on Windows).
func shellQuote(p string) string {
	if runtime.GOOS == "windows" {
		return `"` + p + `"`
	}
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

// dirOf is where a command runs: the script's folder, or · in <dir> below it.
func (s *Script) dirOf(d string) string {
	if d == "" {
		return s.Dir()
	}
	if filepath.IsAbs(d) {
		return d
	}
	return filepath.Join(s.Dir(), d)
}

// lockedBuf is a buffer two streams may write.
type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// runTimed runs a command in its own process group and waits up to d for
// it; past d the group is stopped. Output still held open by something it
// left in the background is let go 2 s after it exits.
func runTimed(cmd, dir string, d time.Duration, abort <-chan struct{}) (string, error) {
	c := shellCmd(cmd, dir)
	var out lockedBuf
	c.Stdout, c.Stderr = &out, &out
	c.WaitDelay = 2 * time.Second
	ownGroup(c)
	if err := c.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	timer := time.NewTimer(d)
	defer timer.Stop()
	why := fmt.Errorf("still running after %s (· within 5m gives it longer)", d)
	select {
	case err := <-done:
		return out.String(), err
	case <-abort:
		why = fmt.Errorf("stopped: the run is stopping")
	case <-timer.C:
	}
	{
		signalGroup(c.Process.Pid, false)
		select {
		case <-done:
		case <-time.After(stopGrace):
			killGroup(c.Process.Pid)
			<-done
		}
		return out.String(), why
	}
}

// lastLines is the last n lines of s.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// defaultWithin is how long a command may take when its line gives no
// · within: prepare: and step: the script's timeout:, the others 10 minutes.
func (r *runner) defaultWithin(c *Command) time.Duration {
	if c.Within > 0 {
		return c.Within
	}
	if c.Key == "prepare" || c.Key == "step" {
		return r.s.Timeout
	}
	return 10 * time.Minute
}

// command runs a front matter command (each of its scripts, in order) and
// gives its result; its output goes to lifecycle.log beside the report.
func (r *runner) command(c *Command) StepResult {
	t := time.Now()
	sr := StepResult{Line: c.Line, Text: c.Key + ": " + c.Text, Kind: c.Key, Status: "passed"}
	d := r.defaultWithin(c)
	var abort <-chan struct{}
	if c.Key != "stop" && c.Key != "cleanup" {
		abort = r.abort
	}
	for _, cmd := range c.Cmds {
		out, err := runTimed(cmd, r.s.dirOf(c.Dir), d, abort)
		r.logLife(sr.Text, cmd, out)
		if err != nil {
			sr.Status, sr.Expected, sr.Found, sr.Output = "failed", c.Key+": exits 0", r.mask(strings.TrimSpace(err.Error())), r.mask(lastLines(out, 40))
			break
		}
	}
	sr.Seconds = time.Since(t).Seconds()
	r.line(sr)
	if r.o.Progress != nil {
		r.o.Progress(sr)
	}
	return sr
}

// logLife appends a command's output to lifecycle.log (masked).
func (r *runner) logLife(what, cmd, out string) {
	if r.o.Out == "" {
		return
	}
	f, err := os.OpenFile(filepath.Join(r.o.Out, "lifecycle.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "── %s (%s)\n%s", r.mask(what), time.Now().Format(time.TimeOnly), r.mask(out))
	if out != "" && !strings.HasSuffix(out, "\n") {
		fmt.Fprintln(f)
	}
}

// prelude runs start:, prepare: and step: in the order written; it returns
// the first that failed (nil: all passed). Its results go to r.pre.
func (r *runner) prelude() *StepResult {
	r.pm.Lock()
	r.began = true
	r.pm.Unlock()
	for _, p := range r.s.Prelude {
		var sr StepResult
		if p.Cmd != nil {
			sr = r.command(p.Cmd)
		} else {
			sr = r.startResult(p.Start)
		}
		r.pre = append(r.pre, sr)
		if sr.Status == "failed" {
			return &sr
		}
	}
	return nil
}

// startResult starts a start: program and gives its result.
func (r *runner) startResult(p *Program) StepResult {
	fmt.Fprintf(r.o.Log, "start: %s\n", r.mask(p.Cmd))
	t := time.Now()
	sr := StepResult{Line: p.Line, Text: "start: " + firstOf(p.Text, p.Cmd), Kind: "start", Status: "passed"}
	if err := r.startProgram(p.Name, p.Cmd, p.Ready, p.Within, false, r.s.dirOf(p.Dir)); err != nil {
		sr.Status, sr.Found = "failed", r.mask(err.Error())
		if fe, ok := err.(*failure); ok {
			sr.Expected, sr.Found, sr.Output = r.mask(fe.expected), r.mask(fe.found), fe.output
		}
		r.line(sr)
	}
	sr.Seconds = time.Since(t).Seconds()
	if r.o.Progress != nil {
		r.o.Progress(sr)
	}
	return sr
}

// Stage is a deck's own setup: si play and si serve run its setup: and
// start: lines (in the order written) before the talk, keep its programs
// running while they serve, and run stop:, stop the programs and run
// cleanup: when they quit. si build never runs them.
type Stage struct {
	r *runner
}

// ParseStage reads a deck's setup:, start:, stop: and cleanup: lines
// (deck.Meta.Lifecycle); path is the deck, its folder where they run. The
// result's Problems are si check problems.
func ParseStage(path string, lines []deck.MetaLine) *Script {
	abs, _ := filepath.Abs(path)
	s := newScript(abs)
	s.Timeout = 10 * time.Minute // a deck's setup: may take its time; · within for less
	for _, l := range lines {
		s.lifecycle(l.Key, l.Val, l.Line, true)
	}
	return s
}

// Empty reports whether the deck sets nothing up.
func (s *Script) Empty() bool {
	return len(s.Prelude) == 0 && len(s.Stops) == 0 && len(s.Cleanups) == 0
}

// NewStage prepares a deck's stage; logs (each program's <name>.log and
// lifecycle.log) go to out, a line per step to log, each step's result to
// progress as it ends.
func NewStage(s *Script, out string, log io.Writer, progress func(StepResult)) *Stage {
	r := &runner{s: s, o: Options{Out: out, Log: log, Progress: progress}, procs: map[string]*proc{}, terms: map[string]*term{}, abort: make(chan struct{})}
	if r.o.Log == nil {
		r.o.Log = io.Discard
	}
	r.secrets = commandSecrets(s)
	return &Stage{r: r}
}

// Up runs the setup: and start: lines in order; it stops at the first that
// fails and returns it.
func (st *Stage) Up() error {
	if k := st.r.s.unresolved(); k != "" {
		return fmt.Errorf("%s was not fetched", k)
	}
	if f := st.r.prelude(); f != nil {
		msg := f.Found
		if f.Output != "" {
			msg += "\n" + f.Output
		}
		return fmt.Errorf("%s: %s", f.Text, msg)
	}
	return nil
}

// Down runs stop:, stops the programs and runs cleanup: (once; safe to call
// on any path out) and gives their results.
func (st *Stage) Down() []StepResult {
	st.r.shutdown()
	return st.r.post
}

// Programs are the started programs' names and their logs.
func (st *Stage) Programs() map[string]string {
	out := map[string]string{}
	for _, p := range st.r.s.Prelude {
		if p.Start != nil {
			out[p.Start.Name] = logPath(st.r.o.Out, p.Start.Name)
		}
	}
	return out
}
