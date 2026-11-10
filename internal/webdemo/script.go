// Package webdemo reads and runs web demo scripts: a `type: demo` page whose
// front matter names an `app:`, with `##` features, `Say:` blocks and steps a
// person would take (go, click, type, expect …). Phase 1 runs them as tests
// over the DevTools protocol (internal/cdp); the spec is
// docs/superpowers/specs/2026-11-10-web-demos.md.
package webdemo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Script is one parsed web demo script.
type Script struct {
	Path      string // the file (absolute)
	Name      string // the file name without extension
	Title     string
	App       string // a URL, or a folder (relative to the script) si serves itself
	Viewport  string // as written
	W, H      int
	Timeout   time.Duration // how long a step waits for its element or condition
	Setup     string        // a shell command run before each feature, in the script's folder
	Starts    []*Program    // start: programs run in the background for the whole run
	Stop      string        // stop: a tear-down command run at the end
	Shell     string        // shell: the shell of terminal steps ("" = $SHELL, else sh)
	Voice     string        // engine:voice as written ("" = the default voice)
	Style     string        // voice-style: a tone for engines that take one (openai)
	Pronounce []Pronounce   // pronounce: word = how to say it
	Features  []*Feature
	Problems  []Problem // the script cannot run
	Warnings  []Problem
}

// Feature is one `##` section: one test case (later, one clip).
type Feature struct {
	Name   string
	What   string // `What it does:`
	Line   int
	Blocks []*Block
}

// Block is one `Say:` line and the steps under it.
type Block struct {
	Say   string
	Line  int
	Steps []*Step
}

// Locator is how a step finds its element: Name (visible text, label, aria
// name, placeholder, title), or a CSS selector (#id, [data-testid=…]),
// optionally narrowed by Role (role=button "Pay").
type Locator struct {
	Name string `json:"name,omitempty"`
	CSS  string `json:"css,omitempty"`
	Role string `json:"role,omitempty"`
}

func (l Locator) String() string {
	switch {
	case l.CSS != "":
		return l.CSS
	case l.Role != "":
		return "role=" + l.Role + " " + strconv.Quote(l.Name)
	}
	return strconv.Quote(l.Name)
}

// Value is a text a step uses: written in the script, or from an
// environment variable (Var, never printed).
type Value struct {
	Text string
	Var  string
}

func (v Value) String() string {
	if v.Var != "" {
		return "$" + v.Var
	}
	return strconv.Quote(v.Text)
}

// Step is one step. Kind is one of the step words (go, click, type, press,
// select, check, uncheck, hover, scroll, wait, waitfor, note, screenshot) or
// expect-text, expect-url, expect-state, expect-errors.
type Step struct {
	Line   int
	Text   string // as written
	Kind   string
	Target Locator
	Value  Value
	State  string        // expect-state: visible, hidden, disabled, enabled, checked, unchecked
	Wait   time.Duration // wait Ns
	For    time.Duration // note … · for 3s
	Pos    string        // note … · position right
	AtLine bool          // note … at line "text": Target.Name is a terminal line's text

	// terminals and background programs
	Name   string            // terminal "Name", start … as NAME, stop NAME, expect output of NAME, expect NAME running
	Opts   map[string]string // terminal options: window, title, theme, backdrop, size
	Block  bool              // run … · wait, stop NAME · wait
	Within time.Duration     // · within 30s
	Ready  Ready             // start …: what to wait for before going on
	Code   int               // expect NAME exited N; run … · exit N (-1: · fails, any non-zero)
}

// Problem is an error or warning at a line of the script.
type Problem struct {
	File string
	Line int
	Msg  string
}

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Msg)
	}
	return fmt.Sprintf("%s: %s", p.File, p.Msg)
}

// Pronounce is one `pronounce: si = ess eye`: what the voice says for a word
// (captions keep the word).
type Pronounce struct{ Word, Say string }

// Viewports are the named viewport sizes.
var Viewports = map[string][2]int{"phone": {400, 800}, "tablet": {820, 1180}, "desktop": {1440, 900}}

// StepWords are the step names, for messages and docs.
var StepWords = []string{"go", "click", "type", "press", "select", "check", "uncheck", "hover", "scroll to", "wait", "wait for", "note", "expect", "screenshot",
	"terminal", "run", "browser", "start", "stop"}

// IsWebDemo reports whether a Markdown or AsciiDoc source is a web demo:
// `type: demo` with an `app:` in its front matter.
func IsWebDemo(src string) bool {
	fm, _ := frontMatter(splitLines(src))
	return fm["type"] == "demo" && fm["app"] != ""
}

// ParseFile reads a script: Markdown, AsciiDoc (.adoc) or JSON (.json).
func ParseFile(path string) (*Script, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".json":
		return ParseJSON(abs, raw)
	case ".adoc", ".asciidoc":
		return Parse(abs, deck.AdocToMarkdown(string(raw))), nil
	}
	return Parse(abs, string(raw)), nil
}

func splitLines(s string) []string {
	s = strings.TrimPrefix(strings.ReplaceAll(s, "\r\n", "\n"), "\uFEFF")
	return strings.Split(s, "\n")
}

var metaRE = regexp.MustCompile(`^([a-z][a-z-]*)\s*:\s*(.*?)\s*$`)

// frontMatter reads `---` / key: value / `---` at the top; it returns the
// keys (values unquoted, a trailing # comment dropped) and the index of the
// first body line.
func frontMatter(lines []string) (map[string]string, int) {
	fm := map[string]string{}
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return fm, 0
	}
	for i := 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "---" {
			return fm, i + 1
		}
		if m := metaRE.FindStringSubmatch(t); m != nil {
			v := m[2]
			if j := strings.Index(v, " #"); j >= 0 {
				v = strings.TrimSpace(v[:j])
			}
			fm[m[1]] = strings.Trim(v, `"'`)
		}
	}
	return map[string]string{}, 0
}

func newScript(path string) *Script {
	return &Script{Path: path, Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))}
}

// Dir is the script's folder: setup runs there, a folder app is relative to it.
func (s *Script) Dir() string { return filepath.Dir(s.Path) }

// shown is the script's path as messages show it: relative to the working
// folder when it is below it.
func (s *Script) shown() string {
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, s.Path); err == nil && !strings.HasPrefix(r, "..") {
			return r
		}
	}
	return s.Path
}

func (s *Script) problem(line int, format string, a ...any) {
	s.Problems = append(s.Problems, Problem{s.shown(), line, fmt.Sprintf(format, a...)})
}

func (s *Script) warn(line int, format string, a ...any) {
	s.Warnings = append(s.Warnings, Problem{s.shown(), line, fmt.Sprintf(format, a...)})
}

// settings applies the front matter (line: where to report a bad value).
func (s *Script) settings(fm map[string]string, line func(string) int) {
	s.Title, s.App, s.Setup, s.Voice, s.Viewport, s.Style = fm["title"], fm["app"], fm["setup"], fm["voice"], fm["viewport"], fm["voice-style"]
	s.Stop, s.Shell = fm["stop"], fm["shell"]
	if s.App == "" {
		s.problem(line("app"), "a web demo needs app: (a URL, a folder of the app next to the script, or none for a terminal-only demo)")
	}
	s.W, s.H = 1440, 900
	if err := s.SetViewport(s.Viewport); err != nil {
		s.problem(line("viewport"), "%v", err)
	}
	if s.Voice != "" {
		if _, err := ParseVoice(s.Voice); err != nil {
			s.problem(line("voice"), "%v", err)
		}
	}
	s.Timeout = 5 * time.Second
	if v := fm["timeout"]; v != "" {
		if d, ok := duration(v); ok && d > 0 {
			s.Timeout = d
		} else {
			s.problem(line("timeout"), "timeout %q: use seconds (10s) or milliseconds (500ms)", v)
		}
	}
}

// SetViewport sets the size from phone, tablet, desktop or WxH ("" keeps it).
func (s *Script) SetViewport(v string) error {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return nil
	}
	if wh, ok := Viewports[v]; ok {
		s.W, s.H = wh[0], wh[1]
	} else if w, h, ok := strings.Cut(v, "x"); ok && atoi(w) > 0 && atoi(h) > 0 {
		s.W, s.H = atoi(w), atoi(h)
	} else {
		return fmt.Errorf("viewport %q: use WxH (1440x900), phone, tablet or desktop", v)
	}
	return nil
}

// pronounce reads `si = ess eye` (several pairs may be split by ;).
func (s *Script) pronounce(v string, line int) {
	for _, p := range strings.Split(v, ";") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		w, say, ok := strings.Cut(p, "=")
		w, say = strings.TrimSpace(strings.Trim(strings.TrimSpace(w), `"'`)), strings.TrimSpace(strings.Trim(strings.TrimSpace(say), `"'`))
		if !ok || w == "" || say == "" {
			s.problem(line, "pronounce %q: write pronounce: word = how to say it", strings.TrimSpace(p))
			continue
		}
		s.Pronounce = append(s.Pronounce, Pronounce{w, say})
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

// duration reads 3s, 1.5s, 500ms or a bare number of seconds.
func duration(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if ms, ok := strings.CutSuffix(v, "ms"); ok {
		f, err := strconv.ParseFloat(ms, 64)
		return time.Duration(f * float64(time.Millisecond)), err == nil && f >= 0
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "s"), 64)
	return time.Duration(f * float64(time.Second)), err == nil && f >= 0
}

var (
	listRE  = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*)$`)
	labelRE = regexp.MustCompile(`^(What it does|Say|Notes|Points)\s*(?:\([^)]*\))?\s*:\s*(.*)$`)
)

// Parse reads a Markdown script (path names it, for messages and relative
// paths).
func Parse(path, src string) *Script {
	s := newScript(path)
	lines := splitLines(src)
	fm, start := frontMatter(lines)
	s.settings(fm, func(key string) int {
		for i := 1; i < start; i++ {
			if m := metaRE.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil && m[1] == key {
				return i + 1
			}
		}
		return 1
	})
	for i := 1; i < start; i++ { // pronounce: and start: may be given more than once
		if m := metaRE.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil && m[1] == "pronounce" {
			s.pronounce(m[2], i+1)
		} else if m != nil && m[1] == "start" {
			s.start(m[2], i+1)
		}
	}
	var f *Feature
	var b *Block
	text := "" // the label whose continuation lines are being read: what, say, skip
	fence := ""
	for i := start; i < len(lines); i++ {
		no, raw := i+1, lines[i]
		t := strings.TrimSpace(raw)
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = t[:3]
			continue
		}
		switch {
		case t == "":
			text = ""
		case strings.HasPrefix(t, "## "):
			f = &Feature{Name: strings.TrimSpace(strings.TrimLeft(t, "# ")), Line: no}
			s.Features = append(s.Features, f)
			b, text = nil, ""
		case strings.HasPrefix(t, "#"):
			text = ""
		case labelRE.MatchString(t):
			m := labelRE.FindStringSubmatch(t)
			switch m[1] {
			case "What it does":
				if f == nil {
					s.problem(no, "What it does: belongs to a feature (a ## heading above it)")
					continue
				}
				f.What, text = m[2], "what"
			case "Say":
				if f == nil {
					s.problem(no, "Say: belongs to a feature (a ## heading above it)")
					continue
				}
				b = &Block{Say: m[2], Line: no}
				f.Blocks = append(f.Blocks, b)
				text = "say"
			default:
				text = "skip" // presenter text, unused by a web demo
			}
		case listRE.MatchString(raw) && text != "skip":
			if f == nil {
				s.problem(no, "a step before the first feature: put it under a ## heading")
				continue
			}
			if b == nil {
				b = &Block{Line: no}
				f.Blocks = append(f.Blocks, b)
			}
			st, err := ParseStep(listRE.FindStringSubmatch(raw)[1])
			if err != nil {
				s.problem(no, "%v", err)
				continue
			}
			st.Line = no
			b.Steps = append(b.Steps, st)
			text = ""
		case text == "what":
			f.What += " " + t
		case text == "say":
			b.Say += " " + t
		}
	}
	s.finish()
	return s
}

// finish checks what a parse cannot see line by line.
func (s *Script) finish() {
	if len(s.Features) == 0 {
		s.problem(0, "no features: each ## heading is a feature with its steps")
	}
	warned := map[string]bool{}
	for _, f := range s.Features {
		n := 0
		for _, b := range f.Blocks {
			n += len(b.Steps)
			for _, st := range b.Steps {
				if st.Value.Var != "" && !warned[st.Value.Var] {
					warned[st.Value.Var] = true
					if _, ok := os.LookupEnv(st.Value.Var); !ok {
						s.warn(st.Line, "$%s is not set in the environment (the step fails until it is)", st.Value.Var)
					}
				}
			}
		}
		if n == 0 {
			s.warn(f.Line, "feature %q has no steps", f.Name)
		}
		s.checkScenes(f)
	}
}

// ParseStep reads one step, as written after the list marker.
func ParseStep(line string) (*Step, error) {
	st := &Step{Text: strings.TrimSpace(line)}
	word, rest := cutWord(st.Text)
	var err error
	bad := func(format string) error { return fmt.Errorf(format, st.Text) }
	switch word {
	case "go":
		if rest == "" {
			return nil, bad("%q: go needs a path (/login) or a URL")
		}
		st.Kind, st.Value.Text = "go", unquote(rest)
	case "click", "hover", "check", "uncheck":
		st.Kind = word
		if st.Target, rest, err = locator(rest); err != nil || rest != "" {
			return nil, bad("%q: " + word + ` needs one element: "text", #id, [data-testid=…] or role=button "Name"`)
		}
	case "scroll":
		to, r := cutWord(rest)
		if to != "to" {
			return nil, bad(`%q: write scroll to "Element"`)
		}
		st.Kind = "scroll"
		if st.Target, rest, err = locator(r); err != nil || rest != "" {
			return nil, bad(`%q: scroll to needs one element`)
		}
	case "type", "select":
		sep := map[string]string{"type": "into", "select": "in"}[word]
		st.Kind = word
		if st.Value, rest, err = value(rest); err != nil {
			return nil, bad("%q: " + word + ` needs a "text" or a $VAR first`)
		}
		if word == "type" && strings.TrimSpace(rest) == "" {
			break // type "text" alone: keystrokes where the focus is, nothing replaced
		}
		w, r := cutWord(rest)
		if w != sep {
			return nil, bad("%q: write " + word + ` "text" ` + sep + ` "Field"`)
		}
		if st.Target, rest, err = locator(r); err != nil || rest != "" {
			return nil, bad("%q: " + word + " needs one field after " + sep)
		}
	case "press":
		if rest == "" {
			return nil, bad("%q: press needs a key (Enter, Tab, cmd+k)")
		}
		st.Kind, st.Value.Text = "press", unquote(rest)
		if _, err := keyChord(st.Value.Text); err != nil {
			return nil, fmt.Errorf("%q: %v", st.Text, err)
		}
	case "wait":
		if f, r := cutWord(rest); f == "for" {
			if o, _ := cutWord(r); o == "output" {
				return outputStep(st, "waitfor-output", r)
			}
			st.Kind = "waitfor"
			if st.Target, rest, err = locator(r); err != nil || rest != "" {
				return nil, bad(`%q: wait for needs one "text" or element`)
			}
			break
		}
		d, ok := duration(rest)
		if !ok || rest == "" {
			return nil, bad(`%q: wait 2s, wait 500ms or wait for "text"`)
		}
		st.Kind, st.Wait = "wait", d
	case "note":
		parts := clauses(rest)
		st.Kind = "note"
		var r string
		if st.Value, r, err = value(parts[0]); err != nil {
			return nil, bad(`%q: write note "text" at "Element"`)
		}
		if at, r2 := cutWord(r); at == "at" {
			if ln, r3 := cutWord(r2); ln == "line" && strings.HasPrefix(r3, `"`) {
				var name string
				if name, r, err = quoted(r3); err != nil || name == "" {
					return nil, bad(`%q: write note "text" at line "text in the terminal"`)
				}
				st.Target, st.AtLine = Locator{Name: name}, true
			} else if st.Target, r, err = locator(r2); err != nil {
				return nil, bad(`%q: note … at needs one element`)
			}
		}
		if strings.TrimSpace(r) != "" {
			return nil, bad(`%q: write note "text" at "Element" (· for 3s · position right)`)
		}
		for _, p := range parts[1:] {
			k, v := cutWord(strings.TrimSpace(p))
			switch k {
			case "for":
				var good bool
				if st.For, good = duration(v); !good {
					return nil, bad("%q: note · for 3s")
				}
			case "position":
				switch v {
				case "top", "right", "bottom", "left":
					st.Pos = v
				default:
					return nil, bad("%q: note · position top, right, bottom or left")
				}
			default:
				return nil, bad("%q: a note takes · for 3s and · position right")
			}
		}
	case "screenshot":
		if rest == "" {
			return nil, bad("%q: screenshot needs a name")
		}
		st.Kind, st.Value.Text = "screenshot", unquote(rest)
	case "expect":
		return expect(st, rest)
	case "terminal", "browser", "run", "start", "stop":
		return termStep(st, word, rest)
	default:
		return nil, fmt.Errorf("unknown step %q (steps: %s)", st.Text, strings.Join(StepWords, ", "))
	}
	return st, nil
}

func expect(st *Step, rest string) (*Step, error) {
	w, r := cutWord(rest)
	var err error
	switch w {
	case "text":
		st.Kind = "expect-text"
		if st.Value, r, err = value(r); err != nil || r != "" {
			return nil, fmt.Errorf(`%q: write expect text "words"`, st.Text)
		}
	case "url":
		if r == "" {
			return nil, fmt.Errorf("%q: write expect url /path (or a full URL)", st.Text)
		}
		st.Kind, st.Value.Text = "expect-url", unquote(r)
	case "no":
		if o, _ := cutWord(r); o == "output" {
			return outputStep(st, "expect-no-output", r)
		}
		if r != "errors" {
			return nil, fmt.Errorf(`%q: write expect no errors (or expect no output "text")`, st.Text)
		}
		st.Kind = "expect-errors"
	case "output":
		return outputStep(st, "expect-output", rest)
	default:
		if nameRE.MatchString(w) {
			if ok, err := procState(st, r); ok {
				return st, err
			}
		}
		st.Kind = "expect-state"
		if st.Target, r, err = locator(rest); err != nil {
			return nil, fmt.Errorf(`%q: expect text "…", expect url /path, expect "Element" visible|hidden|disabled|enabled|checked|unchecked, or expect no errors`, st.Text)
		}
		switch r {
		case "visible", "hidden", "disabled", "enabled", "checked", "unchecked":
			st.State = r
		default:
			return nil, fmt.Errorf("%q: the state is visible, hidden, disabled, enabled, checked or unchecked", st.Text)
		}
	}
	return st, nil
}

func cutWord(s string) (string, string) {
	s = strings.TrimSpace(s)
	w, r, _ := strings.Cut(s, " ")
	return w, strings.TrimSpace(r)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if q, _, err := quoted(s); err == nil {
		return q
	}
	return s
}

// quoted reads a "double-quoted" string (\" and \\ escaped) at the start of s.
func quoted(s string) (string, string, error) {
	if !strings.HasPrefix(s, `"`) {
		return "", s, fmt.Errorf("not quoted")
	}
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			// only \" and \\ are escapes; any other backslash is kept, so a
			// shell command's \n or \t reaches the shell as written
			if i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
				i++
			}
			b.WriteByte(s[i])
		case '"':
			return b.String(), strings.TrimSpace(s[i+1:]), nil
		default:
			b.WriteByte(c)
		}
	}
	return "", s, fmt.Errorf("unclosed quote")
}

var varRE = regexp.MustCompile(`^\$([A-Za-z_][A-Za-z0-9_]*)`)

func value(s string) (Value, string, error) {
	s = strings.TrimSpace(s)
	if m := varRE.FindStringSubmatch(s); m != nil {
		return Value{Var: m[1]}, strings.TrimSpace(s[len(m[0]):]), nil
	}
	q, rest, err := quoted(s)
	return Value{Text: q}, rest, err
}

var roleRE = regexp.MustCompile(`^role=([a-z]+)\s*`)

// locator reads an element at the start of s: "text", #id, [attr=…] (one CSS
// token: up to the next space), or role=name "text".
func locator(s string) (Locator, string, error) {
	s = strings.TrimSpace(s)
	if m := roleRE.FindStringSubmatch(s); m != nil {
		name, rest, err := quoted(s[len(m[0]):])
		if err != nil {
			return Locator{}, s, err
		}
		return Locator{Role: m[1], Name: name}, rest, nil
	}
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "[") || strings.HasPrefix(s, ".") {
		end := len(s)
		depth := 0
		for i, c := range s {
			if c == '[' {
				depth++
			} else if c == ']' {
				depth--
			} else if c == ' ' && depth == 0 {
				end = i
				break
			}
		}
		return Locator{CSS: s[:end]}, strings.TrimSpace(s[end:]), nil
	}
	name, rest, err := quoted(s)
	if err == nil && name == "" {
		err = fmt.Errorf("empty name")
	}
	return Locator{Name: name}, rest, err
}

// Chord is a key with its modifiers (alt 1, ctrl 2, meta 4, shift 8).
type Chord struct {
	Key  string
	Mods int
}

// keyChord reads Enter, Tab, cmd+k, ctrl+shift+p.
func keyChord(s string) (Chord, error) {
	parts := strings.Split(s, "+")
	var c Chord
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if i < len(parts)-1 {
			switch strings.ToLower(p) {
			case "alt", "option", "opt":
				c.Mods |= 1
			case "ctrl", "control":
				c.Mods |= 2
			case "cmd", "meta", "command", "super":
				c.Mods |= 4
			case "shift":
				c.Mods |= 8
			default:
				return c, fmt.Errorf("unknown modifier %q (alt, ctrl, cmd, shift)", p)
			}
			continue
		}
		if k, ok := namedKey(p); ok {
			c.Key = k.key
		} else if len([]rune(p)) == 1 {
			c.Key = p
		} else {
			return c, fmt.Errorf("unknown key %q (Enter, Tab, Escape, Backspace, Delete, Space, Arrow…, Home, End, PageUp, PageDown, F1–F12 or one character)", p)
		}
	}
	return c, nil
}

type keyDef struct {
	key, code, text string
	vk              int
}

var namedKeys = map[string]keyDef{
	"enter": {"Enter", "Enter", "\r", 13}, "return": {"Enter", "Enter", "\r", 13},
	"tab": {"Tab", "Tab", "", 9}, "escape": {"Escape", "Escape", "", 27}, "esc": {"Escape", "Escape", "", 27},
	"backspace": {"Backspace", "Backspace", "", 8}, "delete": {"Delete", "Delete", "", 46},
	"space": {" ", "Space", " ", 32}, "arrowup": {"ArrowUp", "ArrowUp", "", 38}, "arrowdown": {"ArrowDown", "ArrowDown", "", 40},
	"arrowleft": {"ArrowLeft", "ArrowLeft", "", 37}, "arrowright": {"ArrowRight", "ArrowRight", "", 39},
	"up": {"ArrowUp", "ArrowUp", "", 38}, "down": {"ArrowDown", "ArrowDown", "", 40}, "left": {"ArrowLeft", "ArrowLeft", "", 37}, "right": {"ArrowRight", "ArrowRight", "", 39},
	"home": {"Home", "Home", "", 36}, "end": {"End", "End", "", 35}, "pageup": {"PageUp", "PageUp", "", 33}, "pagedown": {"PageDown", "PageDown", "", 34},
}

func init() {
	for i := 1; i <= 12; i++ {
		n := fmt.Sprintf("F%d", i)
		namedKeys[strings.ToLower(n)] = keyDef{n, n, "", 111 + i}
	}
}

func namedKey(p string) (keyDef, bool) {
	k, ok := namedKeys[strings.ToLower(p)]
	return k, ok
}

// jsonScript is the JSON form: the front matter keys, then features with
// blocks of steps written as in Markdown.
type jsonScript struct {
	Type, Title, App, Viewport, Timeout, Setup, Voice string
	Stop, Shell                                       string
	Style                                             string   `json:"voice-style"`
	Pronounce                                         []string `json:"pronounce"`
	Start                                             []string `json:"start"`
	Features                                          []struct {
		Name   string `json:"name"`
		What   string `json:"what"`
		Blocks []struct {
			Say   string   `json:"say"`
			Steps []string `json:"steps"`
		} `json:"blocks"`
	} `json:"features"`
}

// ParseJSON reads the JSON form of a script. Lines point at where each step's
// string sits in the file.
func ParseJSON(path string, raw []byte) (*Script, error) {
	var j jsonScript
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	s := newScript(path)
	src := string(raw)
	pos := 0
	lineOf := func(needle string) int {
		b, _ := json.Marshal(needle)
		if k := strings.Index(src[pos:], string(b)); k >= 0 {
			pos += k
			return strings.Count(src[:pos], "\n") + 1
		}
		return 0
	}
	fm := map[string]string{"title": j.Title, "app": j.App, "viewport": j.Viewport, "timeout": j.Timeout, "setup": j.Setup, "voice": j.Voice, "voice-style": j.Style, "stop": j.Stop, "shell": j.Shell}
	s.settings(fm, func(key string) int {
		if k := strings.Index(src, `"`+key+`"`); k >= 0 {
			return strings.Count(src[:k], "\n") + 1
		}
		return 0
	})
	for _, p := range j.Pronounce {
		s.pronounce(p, lineOf(p))
	}
	for _, p := range j.Start {
		s.start(p, lineOf(p))
	}
	pos = 0
	for _, jf := range j.Features {
		f := &Feature{Name: jf.Name, What: jf.What, Line: lineOf(jf.Name)}
		s.Features = append(s.Features, f)
		for _, jb := range jf.Blocks {
			b := &Block{Say: jb.Say}
			if jb.Say != "" {
				b.Line = lineOf(jb.Say)
			}
			f.Blocks = append(f.Blocks, b)
			for _, txt := range jb.Steps {
				no := lineOf(txt)
				st, err := ParseStep(txt)
				if err != nil {
					s.problem(no, "%v", err)
					continue
				}
				st.Line = no
				b.Steps = append(b.Steps, st)
			}
		}
	}
	s.finish()
	return s, nil
}
