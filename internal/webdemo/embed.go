package webdemo

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// si play's own presenter tools in a demo (RULES §27, issue #13): with app:
// a deck (a .md path: si demo starts si play on it, or a URL si play
// serves), embed-terminal and embed-browser open the play drawer's shell
// tabs and the browser windows over the slides by name, and slide N, next
// and prev move the deck. They go through the play page's public API,
// window.siPlay (internal/assets/web/deck/05-playapi.js), never through si's
// markup.

// embedWindow is the CSS selector of an embed-browser window's page.
func embedWindow(name string) string { return "iframe[data-si-window=" + strconv.Quote(name) + "]" }

var (
	atRE   = regexp.MustCompile(`^(-?\d+)\s*,\s*(-?\d+)$`)
	zoomRE = regexp.MustCompile(`^(\d+)%$`)
)

// embedStep reads embed-terminal and embed-browser (rest: after the word).
func embedStep(st *Step, word, rest string) (*Step, error) {
	parts := clauses(rest)
	st.Kind = word
	if w := strings.TrimSpace(parts[0]); w == "close" || w == "min" || w == "max" || w == "restore" {
		if word == "embed-browser" || len(parts) > 1 {
			return nil, fmt.Errorf(`%q: write embed-terminal close|min|max|restore, or embed-browser "Name" close|min`, st.Text)
		}
		st.State = w
		return st, nil
	}
	name, tail, err := quoted(parts[0])
	if err != nil || name == "" {
		if word == "embed-terminal" {
			return nil, fmt.Errorf(`%q: write embed-terminal "Name" (· split right|down), or embed-terminal close|min|max`, st.Text)
		}
		return nil, fmt.Errorf(`%q: write embed-browser "Name" (a browser: link or a URL; · at 80,60 · size 1100x700 · zoom 125%%)`, st.Text)
	}
	st.Name = name
	if tail != "" {
		if word == "embed-terminal" || (tail != "close" && tail != "min") || len(parts) > 1 {
			return nil, fmt.Errorf(`%q: after the name, embed-browser takes close or min; embed-terminal takes nothing (· split right)`, st.Text)
		}
		st.State = tail
		return st, nil
	}
	st.Opts = map[string]string{}
	for _, c := range parts[1:] {
		k, v := cutWord(c)
		switch {
		case word == "embed-terminal" && k == "split" && (v == "right" || v == "down"):
			st.Value.Text = v
		case word == "embed-browser" && k == "at" && atRE.MatchString(v):
			m := atRE.FindStringSubmatch(v)
			st.Opts["x"], st.Opts["y"] = m[1], m[2]
		case word == "embed-browser" && k == "size" && sizeRE.MatchString(v):
			m := sizeRE.FindStringSubmatch(v)
			st.Opts["w"], st.Opts["h"] = m[1], m[2]
		case word == "embed-browser" && k == "zoom" && zoomRE.MatchString(v) && atoi(strings.TrimSuffix(v, "%")) >= 30 && atoi(strings.TrimSuffix(v, "%")) <= 300:
			st.Opts["zoom"] = strings.TrimSuffix(v, "%")
		case word == "embed-terminal":
			return nil, fmt.Errorf("%q: embed-terminal takes · split right or · split down", st.Text)
		default:
			return nil, fmt.Errorf("%q: embed-browser takes · at X,Y · size WxH · zoom N%% (30%% to 300%%)", st.Text)
		}
	}
	return st, nil
}

// navStep reads slide N, next and prev.
func navStep(st *Step, word, rest string) (*Step, error) {
	st.Kind = word
	if word == "slide" {
		if st.Code = atoi(rest); st.Code < 1 {
			return nil, fmt.Errorf("%q: write slide 4 (the slide's number, from 1)", st.Text)
		}
		return st, nil
	}
	if rest != "" {
		return nil, fmt.Errorf("%q: %s takes nothing (one build step or slide)", st.Text, word)
	}
	return st, nil
}

// embedKinds are the steps that only work on a page si play serves.
var embedKinds = map[string]bool{"embed-terminal": true, "embed-browser": true, "slide": true, "next": true, "prev": true}

// deckApp reports whether app: is a deck file si demo serves with si play.
func deckApp(app string) bool {
	return !strings.Contains(app, "://") && strings.EqualFold(filepath.Ext(app), ".md")
}

// playDeck adds `si play <deck> --addr 127.0.0.1:<free port>` to the front of
// the run's start: programs (stopped with them at the end) and returns the
// deck's URL. si is o.SIBin, else this program.
func (r *runner) playDeck() (string, error) {
	deckPath := r.s.App
	if !filepath.IsAbs(deckPath) {
		deckPath = filepath.Join(r.s.Dir(), deckPath)
	}
	if _, err := os.Stat(deckPath); err != nil {
		return "", fmt.Errorf("app: %s: no such deck next to the script", r.s.App)
	}
	bin := r.o.SIBin
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return "", err
		}
		bin = exe
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	pre := []Prep{{Start: playProgram(bin, deckPath, port, r.s.Shell)}}
	for _, p := range r.s.Prelude {
		if p.Start == nil || p.Start.Name != "si-play" { // a second run of the script: a new port
			pre = append(pre, p)
		}
	}
	r.s.Prelude = pre
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

// playProgram is the start: program of an app: deck (shell: is the drawer's shell).
func playProgram(bin, deckPath string, port int, shell string) *Program {
	cmd := fmt.Sprintf("exec %s play %s --addr 127.0.0.1:%d", shellQuote(bin), shellQuote(deckPath), port)
	if shell != "" {
		cmd = "SHELL=" + shellQuote(shell) + " " + cmd
	}
	return &Program{Name: "si-play", Cmd: cmd, Ready: Ready{Port: port}, Within: time.Minute, Text: "si play " + filepath.Base(deckPath) + " (app:)"}
}

// siPlay runs a window.siPlay call in the app's page and returns its value.
func (r *runner) siPlay(call string, a ...any) (json.RawMessage, error) {
	v, err := r.evalIn(r.app, 0, `(async function(){if(!window.siPlay)throw new Error("the page has no window.siPlay: app: is not a deck si play serves on a loopback address");return await siPlay.`+fmt.Sprintf(call, a...)+`})()`)
	if err != nil {
		return nil, &failure{expected: "siPlay." + strings.SplitN(call, "(", 2)[0] + " works", found: err.Error()}
	}
	return v, nil
}

func js(v any) string { b, _ := json.Marshal(v); return string(b) }

// embedStep runs a step on si play's tools: embed-terminal, embed-browser,
// slide, next, prev, and run, type, press, expect output and wait for output
// on the embed-terminal tab in front.
func (r *runner) embedStep(st *Step) error {
	pause := func() { // the drawer slides, a window opens, the slide moves: on film, it is seen
		if r.tk != nil {
			time.Sleep(600 * time.Millisecond)
		}
	}
	typing := 0
	if r.tk != nil {
		typing = 55
	}
	switch st.Kind {
	case "embed-terminal":
		if st.State != "" {
			if st.State == "close" || st.State == "min" {
				r.embed = ""
			}
			_, err := r.siPlay("terminal.state(%s)", js(st.State))
			pause()
			return err
		}
		o := map[string]string{}
		if st.Value.Text != "" {
			o["split"] = st.Value.Text
		}
		r.embed = st.Name
		_, err := r.siPlay("terminal.open(%s,%s)", js(st.Name), js(o))
		pause()
		return err
	case "embed-browser":
		var err error
		switch st.State {
		case "close":
			_, err = r.siPlay("browser.close(%s)", js(st.Name))
		case "min":
			_, err = r.siPlay("browser.min(%s)", js(st.Name))
		default:
			o := map[string]float64{}
			for k, v := range st.Opts {
				f, _ := strconv.ParseFloat(v, 64)
				if k == "zoom" {
					f /= 100
				}
				o[k] = f
			}
			_, err = r.siPlay("browser.open(%s,%s)", js(st.Name), js(o))
		}
		pause()
		return err
	case "slide", "next", "prev":
		call := st.Kind + "()"
		if st.Kind == "slide" {
			call = fmt.Sprintf("slide(%d)", st.Code)
		}
		v, err := r.siPlay("%s", call)
		if err == nil && st.Kind == "slide" {
			var at struct{ Slide, Slides int }
			json.Unmarshal(v, &at)
			if at.Slide != st.Code {
				return &failure{expected: fmt.Sprintf("slide %d", st.Code), found: fmt.Sprintf("the deck has %d slide(s)", at.Slides)}
			}
		}
		time.Sleep(250 * time.Millisecond) // the transition
		pause()
		return err
	case "run":
		// expect output looks after the command's own line: what the tab showed
		// before it, and its echo, are not its output
		before, _ := r.embedOutput()
		if r.embedRan == nil {
			r.embedRan = map[string][2]string{}
		}
		r.embedRan[r.embed] = [2]string{before, st.Value.Text}
		_, err := r.siPlay("terminal.run(%s,%s,{typing:%d})", js(r.embed), js(st.Value.Text), typing)
		return err
	case "type", "press":
		text := st.Value.Text
		if st.Kind == "press" {
			var err error
			if text, err = termKey(text); err != nil {
				return err
			}
			typing = 0
		} else if t, err := r.value(st.Value); err != nil {
			return err
		} else {
			text = t
		}
		_, err := r.siPlay("terminal.type(%s,%s,{typing:%d})", js(r.embed), js(text), typing)
		return err
	case "expect-output", "waitfor-output":
		d := within(st.Within, r.s.Timeout)
		var out string
		return r.untilFor(d, func() (bool, error) {
			var err error
			if out, err = r.embedOutput(); err != nil {
				return true, err
			}
			if shows(sinceRun(out, r.embedRan[r.embed]), st.Value.Text) {
				return true, nil
			}
			return false, &failure{expected: "drawer tab " + strconv.Quote(r.embed) + " shows " + strconv.Quote(st.Value.Text), found: "not within " + d.String(), output: lastLines(out, 40)}
		})
	}
	return fmt.Errorf("step %q is not an embed step", st.Kind)
}

// embedOutput is the text of the embed-terminal tab in front.
func (r *runner) embedOutput() (string, error) {
	v, err := r.siPlay("terminal.output(%s)", js(r.embed))
	var out string
	json.Unmarshal(v, &out)
	return out, err
}

// sinceRun is out after the last run's command line (ran: the output
// before it, the command), else all of it.
func sinceRun(out string, ran [2]string) string {
	if ran[1] == "" {
		return out
	}
	rest := strings.TrimPrefix(out, strings.TrimRight(ran[0], " \n"))
	if i := strings.Index(rest, ran[1]); i >= 0 {
		return rest[i+len(ran[1]):]
	}
	return rest
}

// toEmbed reports whether a step goes to si play's tools (embedStep).
func (r *runner) toEmbed(st *Step) bool {
	if r.cur != nil { // a terminal of the demo's own is in front
		return false
	}
	switch st.Kind {
	case "embed-terminal", "embed-browser", "slide", "next", "prev":
		return true
	case "run", "expect-output", "waitfor-output", "press":
		return r.embed != "" && st.Name == ""
	case "type":
		return r.embed != "" && st.Target == (Locator{})
	}
	return false
}
