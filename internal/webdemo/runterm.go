package webdemo

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// quietCheck is an expect no output, checked when its block ends.
type quietCheck struct {
	t    *term
	from int // the plain text's offset where the checked output begins
	text string
	idx  int // the step's result in the feature
}

// commonVars are environment variables a command may use that are no
// secret: their values are not masked.
var commonVars = map[string]bool{"HOME": true, "PATH": true, "USER": true, "PWD": true, "SHELL": true, "TMPDIR": true, "LANG": true, "TERM": true}

var cmdVarRE = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)`)

// commandSecrets are the values of the $VARs the script's commands use:
// the shell expands them, so they may show in an output.
func commandSecrets(s *Script) []string {
	var cmds []string
	for _, p := range s.Starts {
		cmds = append(cmds, p.Cmd)
	}
	for _, f := range s.Features {
		for _, b := range f.Blocks {
			for _, st := range b.Steps {
				if st.Kind == "run" || st.Kind == "start" {
					cmds = append(cmds, st.Value.Text)
				}
			}
		}
	}
	var out []string
	for _, c := range cmds {
		for _, m := range cmdVarRE.FindAllStringSubmatch(c, -1) {
			if v := os.Getenv(m[1]); v != "" && !commonVars[m[1]] {
				out = append(out, v)
			}
		}
	}
	return out
}

// startProgram starts a program in the background and waits for it to be
// ready.
func (r *runner) startProgram(name, cmd string, rd Ready, within time.Duration, local bool) error {
	t := time.Now()
	r.pm.Lock() // held while it starts: a shutdown never misses it
	if r.stopping {
		r.pm.Unlock()
		return errStopping
	}
	p, err := startProc(name, cmd, r.s.Dir(), logPath(r.o.Out, name), r.mask)
	if err != nil {
		r.pm.Unlock()
		return &failure{expected: name + " starts", found: err.Error()}
	}
	r.procs[name] = p
	r.all = append(r.all, p)
	if local {
		r.local = append(r.local, p)
	}
	r.pm.Unlock()
	if rd != (Ready{}) {
		if err := p.ready(rd, within); err != nil {
			return err
		}
		fmt.Fprintf(r.o.Log, "  %s ready (%s) in %.1fs\n", name, rd, time.Since(t).Seconds())
	}
	return nil
}

// shutdown runs stop: (while the programs still run, so it can talk to
// them), then stops everything the run started: terminals and programs
// (SIGTERM, SIGKILL after stopGrace, all at once). It runs once: at the end,
// on a failure, on Ctrl-C and on a panic.
func (r *runner) shutdown() {
	r.down.Do(func() {
		if r.s.Stop != "" {
			t := time.Now()
			out, err := shell(r.s.Stop, r.s.Dir())
			sr := StepResult{Text: "stop: " + r.s.Stop, Status: "passed", Seconds: time.Since(t).Seconds()}
			if err != nil {
				sr.Status, sr.Found = "failed", r.mask(err.Error()+" "+out)
			}
			r.line(sr)
		}
		r.pm.Lock()
		r.stopping = true
		terms, procs, srv, rc := r.allTerms, r.all, r.srv, r.rc
		r.pm.Unlock()
		var wg sync.WaitGroup
		for _, t := range terms {
			wg.Add(1)
			go func() { defer wg.Done(); t.close() }()
		}
		for _, p := range procs {
			wg.Add(1)
			go func() { defer wg.Done(); p.stop() }()
		}
		wg.Wait()
		if srv != nil {
			srv.close()
		}
		if rc != "" {
			os.RemoveAll(rc)
		}
	})
}

// endFeature closes the feature's terminals and stops its start steps'
// programs.
func (r *runner) endFeature() {
	if r.app != "" {
		r.c.Session = r.app
	}
	r.pm.Lock()
	terms, local := r.terms, r.local
	r.terms, r.local, r.cur, r.quiet = map[string]*term{}, nil, nil, nil
	r.pm.Unlock()
	var wg sync.WaitGroup
	for _, t := range terms {
		wg.Add(1)
		go func() { defer wg.Done(); t.close() }()
	}
	for _, p := range local {
		wg.Add(1)
		go func() { defer wg.Done(); p.stop() }()
	}
	wg.Wait()
}

// show puts a page in front: the app's or a terminal's. In a take the
// screencast follows it, and the notes of the page left come down.
func (r *runner) show(session string) error {
	if r.c.Session == session {
		return nil
	}
	if r.tk != nil {
		r.js(`window.__siDemo&&__siDemo.clearNotes()`)
		r.tk.notes = nil
	}
	r.c.Session = session
	r.c.Call("Page.bringToFront", nil, nil)
	if r.tk == nil || !r.tk.on {
		return nil
	}
	r.mu.Lock()
	old := r.film
	r.film = session
	r.mu.Unlock()
	r.c.CallIn(old, "Page.stopScreencast", nil, nil)
	return r.screencast()
}

func (r *runner) screencast() error {
	return r.c.Call("Page.startScreencast", map[string]any{"format": "jpeg", "quality": 90, "maxWidth": r.s.W, "maxHeight": r.s.H, "everyNthFrame": 1}, nil)
}

var errStopping = fmt.Errorf("the run is stopping")

// openTerm starts a terminal's shell (and, the first time, the page server
// and si's prompt files); the lock keeps a shutdown from missing it.
func (r *runner) openTerm(st *Step) (*term, error) {
	r.pm.Lock()
	defer r.pm.Unlock()
	if r.stopping {
		return nil, errStopping
	}
	if r.srv == nil {
		dir, err := os.MkdirTemp("", "si-shell-")
		if err != nil {
			return nil, err
		}
		r.rc = dir
		if err := shellRC(dir); err != nil {
			return nil, err
		}
		if r.srv, err = newTermServer(); err != nil {
			return nil, err
		}
	}
	t, err := openShell(st.Name, r.s.Shell, r.s.Dir(), r.rc, r.mask)
	if err != nil {
		return nil, err
	}
	t.opts = st.Opts
	r.terms[st.Name] = t
	r.allTerms = append(r.allTerms, t)
	r.srv.add(t)
	return t, nil
}

// terminal opens a terminal (or brings it back): a shell, and its page in a
// tab of its own.
func (r *runner) terminal(st *Step) error {
	t := r.terms[st.Name]
	if t == nil {
		var err error
		if t, err = r.openTerm(st); err != nil {
			return err
		}
		prev := r.c.Session
		if _, err := r.c.OpenPage(r.ctx, r.s.W, r.s.H); err != nil {
			return err
		}
		t.session = r.c.Session
		err = r.loadTerm(t)
		r.c.Session = prev
		if prev != "" && r.tk != nil && r.tk.on {
			r.c.Call("Page.bringToFront", nil, nil) // the scene filmed now stays in front until the switch
		}
		if err != nil {
			return err
		}
		if err := t.prompt(15 * time.Second); err != nil {
			return err
		}
	}
	if err := r.show(t.session); err != nil {
		return err
	}
	r.cur = t
	if r.tk != nil {
		if !r.tk.on {
			if err := r.roll(); err != nil {
				return err
			}
			time.Sleep(800 * time.Millisecond)
		}
		if v, err := r.c.Eval(`window.__siTermSpot&&__siTermSpot()`); err == nil {
			var xy [2]float64
			if json.Unmarshal(v, &xy) == nil && xy[0] > 0 {
				r.glide(xy[0], xy[1])
				r.ripple(xy[0], xy[1])
			}
		}
	}
	return nil
}

// loadTerm loads the terminal page in the current session and waits until
// it shows the shell.
func (r *runner) loadTerm(t *term) error {
	for _, m := range []string{"Page.enable", "Runtime.enable"} {
		if err := r.c.Call(m, nil, nil); err != nil {
			return err
		}
	}
	if r.tk != nil {
		if err := r.c.Call("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": overlaySource()}, nil); err != nil {
			return err
		}
	}
	if err := r.navigate(r.srv.url(t)); err != nil {
		return err
	}
	end := time.Now().Add(10 * time.Second)
	for {
		if v, err := r.c.Eval(`!!window.__siTermReady`); err == nil && string(v) == "true" {
			return nil
		}
		if time.Now().After(end) {
			return &failure{expected: "terminal " + strconv.Quote(t.name) + " opens", found: "its page did not start"}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// browser brings the app back in front (loading it the first time).
func (r *runner) browser() error {
	if err := r.show(r.app); err != nil {
		return err
	}
	r.cur = nil
	if !r.home {
		r.home = true
		return r.navigate(r.start)
	}
	return nil
}

// typeIn types text into the terminal (at a person's pace in a take) and
// presses Enter.
func (r *runner) typeIn(t *term, text string) error {
	for _, ch := range text {
		if t.pipes { // no echo on pipes: si shows what it types
			t.out.Write([]byte(string(ch)))
		}
		if err := t.send(string(ch)); err != nil {
			return err
		}
		if r.tk != nil {
			time.Sleep(55 * time.Millisecond)
		}
	}
	if r.tk != nil {
		time.Sleep(250 * time.Millisecond) // a look at the line before Enter
	}
	if t.pipes {
		t.out.Write([]byte("\r\n"))
		return t.send("\r\n")
	}
	return t.send("\r")
}

// within is d, or def when the step gave none.
func within(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// termStep runs the steps of terminals and programs.
func (r *runner) termStep(st *Step) error {
	r.pm.Lock()
	p := r.procs[st.Name]
	r.pm.Unlock()
	t := r.cur
	switch st.Kind {
	case "terminal":
		return r.terminal(st)
	case "browser":
		return r.browser()
	case "run":
		at, n := t.out.mark()
		t.runAt, t.ran = at, true
		if t.runs == 0 {
			t.base = n
		}
		t.runs++
		if !t.pipes {
			// each command ends with one prompt mark; an earlier run not waited
			// for may not have printed its own yet, so count, don't look
			// ponytail: a Ctrl-C at an idle prompt adds a mark and shifts this count
			n = t.base + t.runs - 1
		}
		if err := r.typeIn(t, st.Value.Text); err != nil {
			return err
		}
		if !st.Block {
			return nil
		}
		if t.pipes {
			t.send("echo \x1b]133;D;%errorlevel%\x07\r\n")
		}
		d := within(st.Within, time.Minute)
		end := time.Now().Add(d)
		for {
			if code, ok := t.out.code(n); ok {
				switch {
				case st.Code < 0 && code == 0:
					return &failure{expected: "the command fails (a non-zero exit)", found: "exit status 0", output: t.out.tail(40)}
				case st.Code >= 0 && code != st.Code:
					return &failure{expected: fmt.Sprintf("the command exits %d", st.Code), found: fmt.Sprintf("exit status %d", code), output: t.out.tail(40)}
				}
				return nil
			}
			if time.Now().After(end) {
				return &failure{expected: "the command ends", found: "still running after " + d.String(), output: t.out.tail(40)}
			}
			time.Sleep(50 * time.Millisecond)
		}
	case "press":
		k, err := termKey(st.Value.Text)
		if err != nil {
			return err
		}
		return t.send(k)
	case "expect-output", "waitfor-output":
		text, where, tail := func() string { return t.since() }, "terminal "+strconv.Quote(t.nameOr()), func() string { return t.out.tail(40) }
		if st.Name != "" {
			text, where, tail = func() string { return p.out.text(0) }, st.Name, func() string { return p.out.tail(40) }
		}
		d := within(st.Within, r.s.Timeout)
		return r.untilFor(d, func() (bool, error) {
			if shows(text(), st.Value.Text) {
				return true, nil
			}
			return false, &failure{expected: where + " shows " + strconv.Quote(st.Value.Text), found: "not within " + d.String(), output: tail()}
		})
	case "expect-no-output":
		from := t.runAt
		if t.ran {
			s := t.out.text(t.runAt)
			from += len(s)
			for i := 0; i < len(s); i++ {
				if s[i] == '\n' {
					from = t.runAt + i + 1
					break
				}
			}
		}
		r.quiet = append(r.quiet, quietCheck{t: t, from: from, text: st.Value.Text, idx: -1})
		return nil
	case "start":
		return r.startProgram(st.Name, st.Value.Text, st.Ready, st.Within, true)
	case "stop":
		if !st.Block {
			p.stop()
			return nil
		}
		d := within(st.Within, time.Minute)
		select {
		case <-p.done:
			return nil
		case <-time.After(d):
			return &failure{expected: st.Name + " exits on its own", found: "still running after " + d.String(), output: p.out.tail(40)}
		}
	case "expect-proc":
		return r.untilFor(r.s.Timeout, func() (bool, error) {
			found := "it is running"
			if p.exited() {
				found = fmt.Sprintf("it exited with status %d", p.code)
			}
			switch {
			case st.State == "running" && !p.exited():
				return true, nil
			case st.State == "exited" && p.exited() && p.code == st.Code:
				return true, nil
			case st.State == "running":
				return true, &failure{expected: st.Name + " running", found: found, output: p.out.tail(40)}
			}
			return false, &failure{expected: fmt.Sprintf("%s exited %d", st.Name, st.Code), found: found, output: p.out.tail(40)}
		})
	case "note":
		arg, _ := json.Marshal(st.Target.Name)
		err := r.untilFor(r.s.Timeout, func() (bool, error) {
			if v, err := r.c.Eval(`window.__siTermLine(` + string(arg) + `)`); err == nil && string(v) == "true" {
				return true, nil
			}
			return false, &failure{expected: "a line in terminal " + strconv.Quote(t.nameOr()) + " shows " + strconv.Quote(st.Target.Name), found: "no line on screen shows it", output: t.out.tail(40)}
		})
		if err == nil {
			r.note(st)
		}
		return err
	}
	return fmt.Errorf("step %q is not known to the runner", st.Kind)
}

func (t *term) nameOr() string {
	if t == nil {
		return ""
	}
	return t.name
}

// checkQuiet runs the block's expect no output checks; it returns the
// failed one's index and its failure.
func (r *runner) checkQuiet() (int, *failure) {
	defer func() { r.quiet = nil }()
	for _, q := range r.quiet {
		if shows(q.t.out.text(q.from), q.text) {
			return q.idx, &failure{expected: "terminal " + strconv.Quote(q.t.name) + " never shows " + strconv.Quote(q.text), found: "it did", output: q.t.out.tail(40)}
		}
	}
	return -1, nil
}
