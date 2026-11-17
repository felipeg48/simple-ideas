package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/serve"
	"github.com/felipeg48/simple-ideas/internal/ui"
)

// uiView is the look of this run's terminal output (setView).
var uiView = ui.Ledger

// setView picks the look for play, serve, build and check: --view, else
// SI_VIEW, else ui.view: in the project's si.conf, else the user's, else
// ledger. A wrong name is a usage error (bad true).
func setView(cmd string, rest []string, o *options, stderr io.Writer) (int, bool) {
	target := ""
	switch cmd {
	case "play", "serve", "build", "check":
		if len(rest) == 1 {
			target = rest[0]
		}
	default:
		if len(rest) == 0 { // si <dir>: a build
			target = cmd
		}
	}
	if target == "" && o.view == "" {
		return 0, false
	}
	v, err := resolveView(o.view, os.Getenv("SI_VIEW"), target, o.conf)
	if err != nil {
		return usageErr(stderr, err.Error()), true
	}
	uiView = v
	return 0, false
}

func resolveView(flagV, envV, target, conf string) (ui.View, error) {
	if flagV != "" {
		v, err := ui.ParseView(flagV)
		if err != nil {
			return "", fmt.Errorf("--view: %v", err)
		}
		return v, nil
	}
	if envV != "" {
		v, err := ui.ParseView(envV)
		if err != nil {
			return "", fmt.Errorf("SI_VIEW: %v", err)
		}
		return v, nil
	}
	dir := target
	if st, err := os.Stat(target); err == nil && !st.IsDir() {
		dir = filepath.Dir(target)
	}
	if c := config.Load(conf, filepath.Join(dir, "si.conf")); c.UIView != "" { // project over user
		return ui.View(c.UIView), nil
	}
	return ui.Ledger, nil
}

// printer is the look's printer when stdout and stderr are a terminal (and
// NO_COLOR is unset), else nil: the plain lines si always printed.
func printer(stdout, stderr io.Writer) *ui.Printer {
	if !ui.Fancy(stdout, stderr) {
		return nil
	}
	return ui.New(stdout, uiView, true)
}

// shown is a path as the reader typed it: relative to here when inside, a
// folder with its slash (the current one by its name).
func shown(p string) string {
	abs, _ := filepath.Abs(p)
	out := filepath.ToSlash(p)
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, abs); err == nil && !strings.HasPrefix(r, "..") {
			out = filepath.ToSlash(r)
		}
	}
	if out == "." {
		out = filepath.Base(abs)
	}
	if st, err := os.Stat(p); err == nil && st.IsDir() && !strings.HasSuffix(out, "/") {
		out += "/"
	}
	return out
}

// problems prints deck problems (Fail) or warnings (Warn) through p.
func problems(p *ui.Printer, st ui.Status, ps []deck.Problem) {
	for _, x := range ps {
		p.Problem(st, shown(x.File), x.Line, x.Msg)
	}
}

// kindOf names what a page is: deck, article, lab, mind map.
func kindOf(d *deck.Deck) string {
	switch k := d.Meta.Type; {
	case !d.IsDoc() && k == "":
		return "deck"
	case k == "mindmap":
		return "mind map"
	default:
		return k
	}
}

// treeSize is the bytes and HTML pages under dir.
func treeSize(dir string) (size int64, pages int) {
	filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			if info, err := e.Info(); err == nil {
				size += info.Size()
			}
			if strings.HasSuffix(p, ".html") {
				pages++
			}
		}
		return nil
	})
	return size, pages
}

func plural(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}

// playStart is si play's and si serve's first screen in the chosen look.
func playStart(p *ui.Printer, srv *serve.Server, src, cmd, open, phone string, port int, keys bool, o *options, stdout io.Writer) {
	var d *deck.Deck
	dir := src
	if st, err := os.Stat(src); err == nil && !st.IsDir() {
		d, _ = deck.Parse(src)
		dir = filepath.Dir(src)
	}
	theme := o.theme
	if theme == "" && d != nil {
		theme = d.Meta.Theme
	}
	if theme == "" {
		theme = config.Load(o.conf, filepath.Join(dir, "si.conf")).Theme
	}
	meta := "serving " + shown(src) // "serving": the word scripts and demos wait for
	if theme != "" {
		meta += " · theme " + filepath.Base(theme)
	}
	p.Header("🎬", cmd+" "+version, meta)
	switch {
	case d != nil && d.IsDoc():
		p.Item(ui.OK, kindOf(d), d.Title(), "")
	case d != nil:
		p.Item(ui.OK, kindOf(d), d.Title(), plural(len(d.Slides), "slide"))
	default:
		p.Item(ui.OK, "pages", plural(len(srv.RemoteDecks()), "page"), "and the guides beside them")
	}
	p.Item(ui.OK, "live", "live reload on", plural(srv.Live().Files, "file")+" watched")
	if srv.Terminal {
		p.Item(ui.OK, "drawer", "press ` on the deck", "local commands, loopback only")
	}
	p.Break()
	if phone != "" {
		p.Item(ui.Info, "phone", phone, "same network")
	} else if srv.Terminal {
		p.Item(ui.Info, "phone", fmt.Sprintf("si play %s --addr 0.0.0.0:%d", src, port), "to open it on a phone")
	}
	p.Item(ui.Info, "open", open, "")
	if srv.Mon.Web != "" {
		p.Item(ui.Info, "monitor", srv.Mon.Web, "the control room, this machine")
	}
	if keys {
		p.Hint("m monitor · p codes · w web monitor · o open · q quit")
	} else {
		p.Hint("Ctrl+C stops")
	}
	if phone != "" {
		printQR(stdout, phone)
	}
}

// recent keeps the server's log lines for the monitors; while the terminal
// monitor is on they do not go to out.
type recent struct {
	mu    sync.Mutex
	out   io.Writer
	quiet bool // the terminal monitor is on: keep, do not print
	lines []ui.LogLine
}

func (r *recent) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.quiet {
		r.out.Write(b)
	}
	text := strings.TrimSpace(string(b))
	if len(text) > 20 && text[4] == '/' && text[7] == '/' { // the log package's date and time
		text = text[20:]
	}
	text = strings.TrimPrefix(strings.TrimPrefix(text, "si play: "), "si: ")
	st := ui.Warn
	if strings.Contains(text, "change detected") {
		text, st = "reloaded", ui.OK
	}
	r.lines = append(r.lines, ui.LogLine{At: time.Now(), St: st, Text: text})
	if len(r.lines) > 6 {
		r.lines = r.lines[len(r.lines)-6:]
	}
	return len(b), nil
}

// snapshot is the recent lines (serve.MonitorInfo.Log).
func (r *recent) snapshot() []ui.LogLine {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ui.LogLine(nil), r.lines...)
}

func (r *recent) hush(quiet bool) {
	r.mu.Lock()
	r.quiet = quiet
	r.mu.Unlock()
}

// codes is the p screen: the network links as terminal QR codes, the LAN
// address and the audience's (its public one with --tunnel).
func codes(srv *serve.Server, phone string) string {
	var b strings.Builder
	b.WriteString("si play · the links as QR codes\n\n")
	n := 0
	show := func(name, u string) {
		fmt.Fprintf(&b, "  %s: %s\n", name, u)
		printQR(&b, u)
		n++
	}
	if phone != "" {
		show("on your network", phone)
	}
	if srv.RemoteQR != nil && srv.RemoteQR.Audience != nil {
		show("audience", srv.RemoteQR.Audience.URL)
	}
	if n == 0 {
		b.WriteString("  No network links: si play talk.md --addr 0.0.0.0 (your network) or --audience (the room's phones).\n")
	}
	b.WriteString("\n  q / Esc back\n")
	return b.String()
}

// monitor runs si play's keys and live monitor until done: m shows or hides
// the monitor (the terminal's alternate screen, redrawn every second), p the
// links as QR codes, w opens the web monitor, o the deck, q quits (or, on
// the codes, goes back). It returns what restores the terminal.
func monitor(srv *serve.Server, logs *recent, quit func(), done <-chan struct{}, stdout io.Writer) func() {
	restore := ui.Keys(os.Stdin)
	keys := make(chan byte)
	go func() {
		b := make([]byte, 1)
		for {
			if n, err := os.Stdin.Read(b); err != nil {
				return
			} else if n == 1 {
				keys <- b[0]
			}
		}
	}()
	view, back := "", "" // "", "mon" or "qr"; back: what Esc on the codes returns to
	show := func(text string) {
		fmt.Fprint(stdout, "\x1b[H\x1b[2J"+strings.ReplaceAll(text, "\n", "\r\n"))
	}
	draw := func() {
		st := srv.Stats() // the web monitor's numbers too
		st.Keys = "q quit · m hide · p codes · w web · o open"
		if view == "mon" {
			show(ui.Monitor(uiView, st, true))
		}
	}
	enter := func(v string) {
		if view == "" {
			fmt.Fprint(stdout, "\x1b[?1049h\x1b[?25l") // the alternate screen, no cursor
			logs.hush(true)
		}
		view = v
	}
	leave := func() {
		if view != "" {
			view = ""
			logs.hush(false)
			fmt.Fprint(stdout, "\x1b[?25h\x1b[?1049l") // the cursor back, the main screen back
		}
	}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				leave()
				return
			case <-t.C:
				draw()
			case k := <-keys:
				switch {
				case view == "qr" && (k == 'q' || k == 'Q' || k == 0x1b):
					if back == "mon" {
						view = "mon"
						draw()
					} else {
						leave()
					}
				case k == 'q' || k == 'Q':
					leave()
					quit()
				case k == 'o' || k == 'O':
					openBrowser(srv.Mon.Open)
				case k == 'w' || k == 'W':
					openBrowser(srv.Mon.Web)
				case k == 'p' || k == 'P':
					back = view
					enter("qr")
					show(codes(srv, srv.Mon.Phone))
				case k == 'm' || k == 'M':
					if view == "mon" {
						leave()
					} else {
						enter("mon")
						draw()
					}
				}
			}
		}
	}()
	return func() {
		<-finished
		restore()
	}
}

// summary is a build's last line: outputs, bytes, time, and what to open.
func summary(p *ui.Printer, ok bool, n int, size int64, took time.Duration, index string) {
	text := plural(n, "output") + " · " + ui.Size(size) + " · " + ui.Secs(took)
	if !ok {
		p.Summary(ui.Fail, "Failed", text, "")
		return
	}
	p.Summary(ui.OK, "Done", text, "open "+shown(index))
}

// tally is si check's last line: warnings and problems, and what was checked.
func tally(p *ui.Printer, problems, warnings int, what string) int {
	st := ui.OK
	if warnings > 0 {
		st = ui.Warn
	}
	if problems > 0 {
		st = ui.Fail
	}
	text := plural(warnings, "warning") + " · " + plural(problems, "problem") + " · " + what
	if st == ui.OK {
		text = what + " OK · no warnings, no problems" // "OK": as the plain line says it
	}
	p.Summary(st, "", text, "")
	if problems > 0 {
		return exitProblems
	}
	return exitOK
}
