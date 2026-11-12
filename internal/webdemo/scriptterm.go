package webdemo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Program is a command run in the background: a front matter start: (the
// whole run) or a start step (its feature).
type Program struct {
	Cmd    string
	Name   string // as NAME (an unnamed start: is start-<n>); its log is <name>.log
	Ready  Ready
	Within time.Duration // how long Ready may take (0 = 30 s)
	Line   int
}

// Ready is what a started program must show before the demo goes on: a
// text in its output, a TCP port that accepts, or a plain delay.
type Ready struct {
	Text  string
	Port  int
	Delay time.Duration
}

func (r Ready) String() string {
	switch {
	case r.Text != "":
		return "wait for " + strconv.Quote(r.Text)
	case r.Port > 0:
		return fmt.Sprintf("wait for port %d", r.Port)
	case r.Delay > 0:
		return "wait " + r.Delay.String()
	}
	return ""
}

// The terminal window's looks, as in a code window (RULES §25).
var (
	termWindows   = []string{"terminal", "mac", "windows", "none"}
	termThemes    = []string{"auto", "night", "paper", "dusk"}
	termBackdrops = []string{"none", "dusk", "ocean", "mint", "graphite"}
)

var (
	nameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	sizeRE = regexp.MustCompile(`^(\d+)x(\d+)$`)
)

// clauses splits a step at its · separators (not inside quotes).
func clauses(s string) []string {
	var out []string
	q := false
	start := 0
	for i, c := range s {
		switch {
		case c == '"' && (i == 0 || s[i-1] != '\\'):
			q = !q
		case c == '·' && !q:
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + len("·")
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// readyClause reads one `wait for "text"`, `wait for port N` or `wait 3s`
// clause (and `within Ns`); ok is false when the clause is neither.
func readyClause(c string, r *Ready, within *time.Duration) (bool, error) {
	w, rest := cutWord(c)
	switch w {
	case "within":
		d, ok := duration(rest)
		if !ok || rest == "" || d <= 0 {
			return true, fmt.Errorf("within 30s")
		}
		*within = d
		return true, nil
	case "wait":
		if f, r2 := cutWord(rest); f == "for" {
			if p, n := cutWord(r2); p == "port" {
				if r.Port = atoi(n); r.Port <= 0 || r.Port > 65535 {
					return true, fmt.Errorf("wait for port 8080")
				}
				return true, nil
			}
			t, tail, err := quoted(r2)
			if err != nil || t == "" || tail != "" {
				return true, fmt.Errorf(`wait for "text"`)
			}
			r.Text = t
			return true, nil
		}
		d, ok := duration(rest)
		if !ok || rest == "" {
			return true, fmt.Errorf("wait 3s")
		}
		r.Delay = d
		return true, nil
	}
	return false, nil
}

// start reads a front matter `start: <command> [as NAME] · wait for …`.
func (s *Script) start(v string, line int) {
	parts := clauses(v)
	p := &Program{Line: line}
	cmd := parts[0]
	if q, rest, err := quoted(cmd); err == nil {
		cmd = q
		if a, n := cutWord(rest); a == "as" && nameRE.MatchString(n) {
			p.Name = n
		} else if rest != "" {
			s.problem(line, "start: %q: after the quoted command, only as NAME", v)
			return
		}
	} else if f := strings.Fields(cmd); len(f) >= 3 && f[len(f)-2] == "as" && nameRE.MatchString(f[len(f)-1]) {
		p.Name = f[len(f)-1]
		cmd = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(cmd), f[len(f)-1]))
		cmd = strings.TrimSpace(strings.TrimSuffix(cmd, "as"))
	}
	p.Cmd = strings.TrimSpace(cmd)
	if p.Cmd == "" {
		s.problem(line, `start: needs a command (start: ./server · wait for "ready")`)
		return
	}
	for _, c := range parts[1:] {
		if ok, err := readyClause(c, &p.Ready, &p.Within); !ok || err != nil {
			s.problem(line, `start: %q: after ·, write wait for "text", wait for port N, wait 3s or within 60s`, c)
			return
		}
	}
	if p.Name == "" {
		p.Name = fmt.Sprintf("start-%d", len(s.Starts)+1)
	}
	for _, o := range s.Starts {
		if o.Name == p.Name {
			s.problem(line, "start: two programs are named %s", p.Name)
		}
	}
	s.Starts = append(s.Starts, p)
}

// termStep reads terminal, browser, run, start and stop.
func termStep(st *Step, word, rest string) (*Step, error) {
	parts := clauses(rest)
	head := parts[0]
	bad := func(format string) (*Step, error) { return nil, fmt.Errorf(format, st.Text) }
	switch word {
	case "browser":
		if head != "" {
			return bad("%q: browser takes nothing but a transition (it switches back to the app): browser · slide")
		}
		st.Kind = "browser"
		return st, transClauses(st, parts[1:])
	case "terminal":
		name, tail, err := quoted(head)
		if err != nil || name == "" || tail != "" {
			return bad(`%q: write terminal "Name" (· window=terminal title="…" theme=night backdrop=graphite size=100x30)`)
		}
		st.Kind, st.Name, st.Opts = "terminal", name, map[string]string{}
		for _, c := range parts[1:] {
			if !strings.Contains(c, "=") { // terminal "Client" · pop
				if err := transClauses(st, []string{c}); err != nil {
					return nil, err
				}
				continue
			}
			for c != "" {
				k, v, ok := strings.Cut(c, "=")
				if !ok {
					return nil, fmt.Errorf("%q: terminal options are key=value: window, title, theme, backdrop, size", st.Text)
				}
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				if q, r, err := quoted(v); err == nil {
					v, c = q, r
				} else {
					v, c, _ = strings.Cut(v, " ")
					c = strings.TrimSpace(c)
				}
				if err := termOption(k, v); err != nil {
					return nil, fmt.Errorf("%q: %v", st.Text, err)
				}
				st.Opts[k] = v
			}
		}
		return st, nil
	case "run":
		cmd, tail, err := quoted(head)
		if err != nil || cmd == "" || tail != "" {
			return bad(`%q: write run "command" (· wait to wait until it exits)`)
		}
		st.Kind, st.Value.Text = "run", cmd
		for _, c := range parts[1:] {
			if c == "wait" {
				st.Block = true
			} else if c == "fails" {
				st.Block, st.Code = true, -1
			} else if n, ok := strings.CutPrefix(c, "exit "); ok {
				code, err := strconv.Atoi(strings.TrimSpace(n))
				if err != nil || code < 0 {
					return bad(`%q: write · exit 1 (the status the command must end with)`)
				}
				st.Block, st.Code = true, code
			} else if ok, err := readyClause(c, &Ready{}, &st.Within); !ok || err != nil || !strings.HasPrefix(c, "within") {
				return bad(`%q: run takes · wait, · exit 1, · fails and · within 60s`)
			}
		}
		return st, nil
	case "start":
		cmd, tail, err := quoted(head)
		a, n := cutWord(tail)
		if err != nil || cmd == "" || a != "as" || !nameRE.MatchString(n) {
			return bad(`%q: write start "command" as NAME (· wait for "ready" · wait for port 8080 · wait 3s)`)
		}
		st.Kind, st.Value.Text, st.Name = "start", cmd, n
		for _, c := range parts[1:] {
			if ok, err := readyClause(c, &st.Ready, &st.Within); !ok || err != nil {
				return bad(`%q: after ·, write wait for "text", wait for port N, wait 3s or within 60s`)
			}
		}
		return st, nil
	case "stop":
		if !nameRE.MatchString(head) {
			return bad("%q: write stop NAME (· wait to let it finish on its own)")
		}
		st.Kind, st.Name = "stop", head
		for _, c := range parts[1:] {
			if c == "wait" {
				st.Block = true
			} else if ok, err := readyClause(c, &Ready{}, &st.Within); !ok || err != nil || !strings.HasPrefix(c, "within") {
				return bad("%q: stop takes · wait and · within 60s")
			}
		}
		return st, nil
	}
	return nil, fmt.Errorf("unknown step %q", st.Text)
}

func termOption(k, v string) error {
	one := func(list []string) error {
		for _, x := range list {
			if v == x {
				return nil
			}
		}
		return fmt.Errorf("%s=%s: use %s", k, v, strings.Join(list, ", "))
	}
	switch k {
	case "window":
		return one(termWindows)
	case "theme":
		return one(termThemes)
	case "backdrop":
		return one(termBackdrops)
	case "title":
		return nil
	case "size":
		if m := sizeRE.FindStringSubmatch(v); m != nil && atoi(m[1]) >= 20 && atoi(m[2]) >= 5 {
			return nil
		}
		return fmt.Errorf("size=%s: columns x rows, as 100x30", v)
	}
	return fmt.Errorf("unknown terminal option %q (window, title, theme, backdrop, size)", k)
}

// outputStep reads `expect output "text"`, `expect output of NAME "text"`,
// `expect no output "text"` and `wait for output "text"` (rest starts at
// the word output).
func outputStep(st *Step, kind, rest string) (*Step, error) {
	parts := clauses(rest)
	_, r := cutWord(parts[0]) // output
	if of, r2 := cutWord(r); of == "of" && kind == "expect-output" {
		n, r3 := cutWord(r2)
		if !nameRE.MatchString(n) {
			return nil, fmt.Errorf(`%q: write expect output of NAME "text"`, st.Text)
		}
		st.Name, r = n, r3
	}
	t, tail, err := quoted(r)
	if err != nil || t == "" || tail != "" {
		return nil, fmt.Errorf(`%q: the text goes in quotes: %s "text"`, st.Text, map[string]string{"expect-output": "expect output", "expect-no-output": "expect no output", "waitfor-output": "wait for output"}[kind])
	}
	st.Kind, st.Value.Text = kind, t
	for _, c := range parts[1:] {
		if ok, err := readyClause(c, &Ready{}, &st.Within); !ok || err != nil || !strings.HasPrefix(c, "within") || kind == "expect-no-output" {
			return nil, fmt.Errorf("%q: only · within 30s may follow", st.Text)
		}
	}
	return st, nil
}

// procState reads `expect NAME running` and `expect NAME exited 0`; ok is
// false when rest is not one of them (the step is an element's state).
func procState(st *Step, rest string) (bool, error) {
	w, n := cutWord(rest)
	switch {
	case w == "running" && n == "":
		st.Kind, st.State = "expect-proc", "running"
	case w == "exited":
		code, err := strconv.Atoi(n)
		if err != nil {
			return true, fmt.Errorf("%q: write expect NAME exited 0", st.Text)
		}
		st.Kind, st.State, st.Code = "expect-proc", "exited", code
	default:
		return false, nil
	}
	st.Name, _ = cutWord(st.Text[len("expect"):])
	return true, nil
}

// inBrowser are the steps that act on the app's page.
var inBrowser = map[string]bool{"go": true, "click": true, "hover": true, "check": true, "uncheck": true, "scroll": true, "type": true, "select": true,
	"waitfor": true, "expect-text": true, "expect-url": true, "expect-state": true, "expect-errors": true,
	"zoom": true, "pan": true, "spotlight": true, "drag": true}

// inTerminal are the steps that act on the current terminal.
var inTerminal = map[string]bool{"run": true, "expect-output": true, "expect-no-output": true, "waitfor-output": true}

// checkScenes follows a feature's steps through its scenes (the browser, a
// terminal) and its programs: a browser step needs the browser, a terminal
// step a terminal, a program its start.
func (s *Script) checkScenes(f *Feature) {
	known := map[string]bool{}
	for _, p := range s.Starts {
		known[p.Name] = true
	}
	term := "" // the current terminal ("" = the browser)
	if first := firstStep(f); s.App == "none" && first != nil && first.Kind != "terminal" {
		s.problem(first.Line, `with app: none (a terminal-only demo) a feature begins with terminal "Name"`)
	}
	for _, b := range f.Blocks {
		for _, st := range b.Steps {
			switch {
			case s.App == "none" && (st.Kind == "browser" || (inBrowser[st.Kind] && term == "")):
				s.problem(st.Line, "%q: app: none has no browser (a terminal-only demo)", st.Text)
			case st.Kind == "terminal":
				term = st.Name
			case st.Kind == "browser" || st.Kind == "window":
				term = ""
			case inBrowser[st.Kind] && term != "" && st.Opts["off"] == "":
				s.problem(st.Line, "%q acts on the app, but terminal %q is in front: write - browser before it", st.Text, term)
			case (inTerminal[st.Kind] && st.Name == "") || (st.Kind == "note" && st.AtLine):
				if term == "" {
					s.problem(st.Line, `%q needs a terminal: write - terminal "Name" before it`, st.Text)
				}
			case st.Kind == "note" && st.Target != (Locator{}) && term != "":
				s.problem(st.Line, `%q points at the app, but a terminal is in front: in a terminal write note "…" at line "text"`, st.Text)
			case st.Kind == "press" && term != "":
				if _, err := termKey(st.Value.Text); err != nil {
					s.problem(st.Line, "%q: %v", st.Text, err)
				}
			case st.Kind == "start":
				if known[st.Name] {
					s.problem(st.Line, "%q: a program named %s is already started", st.Text, st.Name)
				}
				known[st.Name] = true
			case st.Name != "" && (st.Kind == "stop" || st.Kind == "expect-proc" || st.Kind == "expect-output"):
				if !known[st.Name] {
					s.problem(st.Line, "%q: no program named %s (start \"…\" as %s, or start: … as %s)", st.Text, st.Name, st.Name, st.Name)
				}
			}
		}
	}
}

// termKey is what a key sends to a terminal: Enter, Tab, Escape, Backspace,
// arrows, ctrl+<letter> or one character.
func termKey(k string) (string, error) {
	ch, err := keyChord(k)
	if err != nil {
		return "", err
	}
	seq := map[string]string{"Enter": "\r", "Tab": "\t", "Escape": "\x1b", "Backspace": "\x7f", "Delete": "\x1b[3~", " ": " ",
		"ArrowUp": "\x1b[A", "ArrowDown": "\x1b[B", "ArrowRight": "\x1b[C", "ArrowLeft": "\x1b[D", "Home": "\x1b[H", "End": "\x1b[F"}
	if ch.Mods == 2 && len(ch.Key) == 1 {
		c := strings.ToLower(ch.Key)[0]
		if c >= 'a' && c <= 'z' {
			return string(rune(c & 0x1f)), nil
		}
	}
	if ch.Mods == 0 {
		if s, ok := seq[ch.Key]; ok {
			return s, nil
		}
		if len([]rune(ch.Key)) == 1 {
			return ch.Key, nil
		}
	}
	return "", fmt.Errorf("a terminal takes Enter, Tab, Escape, Backspace, Delete, Space, arrows, Home, End, ctrl+<letter> or one character")
}
