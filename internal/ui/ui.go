// Package ui prints si's terminal output (si play / si serve startup, si
// build, si check) in one of four looks, with standard-library ANSI only.
// Callers use it only on a terminal (Fancy); piped or NO_COLOR output keeps
// the plain lines it always had.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// View is a look: ledger (the default), badges, checklist or steps.
type View string

const (
	Ledger    View = "ledger"
	Badges    View = "badges"
	Checklist View = "checklist"
	Steps     View = "steps"
)

// Views are the four looks, in the order the docs list them.
var Views = []View{Ledger, Badges, Checklist, Steps}

// ParseView reads a look's name; a wrong one is an error naming the four.
func ParseView(s string) (View, error) {
	v := View(strings.ToLower(strings.TrimSpace(s)))
	for _, k := range Views {
		if v == k {
			return v, nil
		}
	}
	return "", fmt.Errorf("unknown view %q: pick ledger, badges, checklist or steps", s)
}

// Status is what a line means: done, a warning, a problem, or plain news.
type Status int

const (
	OK Status = iota
	Warn
	Fail
	Info
)

// Terminal reports whether w is a terminal (a console on Windows).
func Terminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && isTerminal(f)
}

// Fancy reports whether si may colour its output: stdout and stderr are both
// terminals, NO_COLOR is unset and TERM is not dumb.
func Fancy(stdout, stderr io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !Terminal(stdout) || !Terminal(stderr) {
		return false
	}
	return vt(stdout.(*os.File)) && vt(stderr.(*os.File))
}

const (
	reset   = "\x1b[0m"
	bold    = "1"
	dim     = "2"
	under   = "4"
	red     = "31"
	green   = "32"
	yellow  = "33"
	blue    = "34"
	magenta = "35"
	cyan    = "36"
)

// paint wraps s in an SGR code, or leaves it bare without colour.
func paint(color bool, code, s string) string {
	if !color || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + reset
}

// badge is s as a block: dark text on the colour's background.
func badge(color bool, fg, s string) string {
	if !color {
		return "[" + strings.TrimSpace(s) + "]"
	}
	return "\x1b[1;30;4" + fg[1:] + "m" + s + reset
}

// Width is the visible width of s: ANSI codes left out, a rune a column.
func Width(s string) int {
	n, esc := 0, false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc:
			esc = r != 'm'
		case r == 0xFE0F: // emoji presentation selector: no column of its own
		default:
			n++
		}
	}
	return n
}

// Pad pads s with spaces to n visible columns.
func Pad(s string, n int) string {
	if w := Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func statusColor(st Status) string {
	return [...]string{green, yellow, red, cyan}[st]
}

// Size is a byte count as si says it: 212 KB, 1.9 MB.
func Size(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%d B", n)
}

// Secs is a short duration: 0.4s, 12s.
func Secs(d time.Duration) string {
	if d < 10*time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}

// Printer writes lines in one look. Its zero blank state assumes the line
// before it (the shell prompt) is not blank.
type Printer struct {
	w       io.Writer
	v       View
	color   bool
	blank   bool   // the last line written was blank: no two in a row
	pending string // steps: the last tree line, drawn once the next shows whether it is the last
}

// New is a printer for w in look v; color false writes no escape codes.
func New(w io.Writer, v View, color bool) *Printer { return &Printer{w: w, v: v, color: color} }

// View is the printer's look.
func (p *Printer) View() View { return p.v }

func (p *Printer) c(code, s string) string { return paint(p.color, code, s) }

func (p *Printer) line(s string) {
	if s == "" {
		if p.blank {
			return
		}
		p.blank = true
	} else {
		p.blank = false
	}
	fmt.Fprintln(p.w, s)
}

// tree queues a steps line; the one before it is drawn as a middle branch.
func (p *Printer) tree(s string) {
	p.flush("├─")
	p.pending = s
}

func (p *Printer) flush(branch string) {
	if p.pending != "" {
		p.line(p.c(dim, branch) + " " + p.pending)
		p.pending = ""
	}
}

func (p *Printer) mark(st Status) string {
	return p.c(statusColor(st), [...]string{"✓", "!", "✗", "▶"}[st])
}

func (p *Printer) icon(st Status) string {
	return [...]string{"✅", "⚠️ ", "❌", "👉"}[st]
}

func (p *Printer) value(st Status, s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return p.c(cyan+";"+under, s)
	}
	if st == Warn || st == Fail {
		return p.c(statusColor(st), s)
	}
	return s
}

// Header opens the output: si's name and what it works on. icon is the
// checklist look's mark (🎬 play, 📦 build).
func (p *Printer) Header(icon, title, meta string) {
	p.line("")
	switch p.v {
	case Badges:
		p.line(" " + badge(p.color, magenta, " "+title+" ") + "  " + p.c(dim, meta))
	case Checklist:
		p.line(icon + "  " + p.c(bold, title) + "  " + p.c(dim, meta))
	case Steps:
		p.line(p.c(magenta, "◆") + " " + p.c(bold, title) + " " + p.c(dim, meta))
		p.line(p.c(dim, "│"))
		return
	default:
		p.line(p.c(bold, title) + "  " + p.c(dim, meta))
	}
	p.line("")
}

// Begin opens a quiet command (si check): the steps look draws its root,
// the others a blank line.
func (p *Printer) Begin(title, meta string) {
	p.line("")
	if p.v == Steps {
		p.line(p.c(magenta, "◆") + " " + p.c(bold, title) + " " + p.c(dim, meta))
	}
}

// Item is one fact: a status, a short label, its value and dim detail.
func (p *Printer) Item(st Status, label, value, meta string) {
	switch p.v {
	case Badges:
		s := " " + badge(p.color, statusColor(st), " "+Pad(strings.ToUpper(label), 6)+" ") + "  " + p.value(st, value)
		if meta != "" {
			s += p.c(dim, " · "+meta)
		}
		p.line(s)
	case Checklist:
		s := p.icon(st) + "  " + Pad(strings.ToUpper(label[:1])+label[1:], 8) + " " + p.value(st, value)
		if meta != "" {
			s += "  " + p.c(dim, "("+meta+")")
		}
		p.line(s)
	case Steps:
		s := p.mark(st) + " " + Pad(label+" "+p.value(st, value), 34)
		if meta != "" {
			s += " " + p.c(dim, meta)
		}
		p.tree(s)
	default:
		s := "  " + p.mark(st) + " " + p.c(dim, Pad(label, 10)) + " " + p.value(st, value)
		if meta != "" {
			s += "  " + p.c(dim, meta)
		}
		p.line(s)
	}
}

// Output is one written file of a build: its kind (deck, lab, guide), name,
// size and the time it took.
func (p *Printer) Output(st Status, kind, name string, size int64, took time.Duration) {
	sz, t := Size(size), Secs(took)
	switch p.v {
	case Badges:
		p.line(" " + badge(p.color, statusColor(st), map[Status]string{OK: "  OK  ", Warn: " WARN ", Fail: " FAIL "}[st]) + "  " + Pad(name, 22) + " " + p.c(dim, sz+" · "+t))
	case Checklist:
		p.line(p.icon(st) + "  " + Pad(name, 24) + " " + p.c(dim, Pad(sz, 7)+"  "+t))
	case Steps:
		n := int(took / (100 * time.Millisecond))
		n = max(1, min(n, 8))
		p.tree(p.mark(st) + " " + Pad(name, 22) + " " + p.c(dim, Pad(sz, 7)) + " " + p.c(statusColor(st), Pad(strings.Repeat("▇", n), 8)) + " " + p.c(dim, t))
	default:
		p.line("  " + p.mark(st) + " " + p.c(dim, Pad(kind, 10)) + " " + Pad(name, 24) + " " + p.c(dim, Pad(sz, 7)+"  "+t))
	}
}

// Problem is a warning or a problem at file:line (line 0: the file).
func (p *Printer) Problem(st Status, file string, line int, msg string) {
	loc := file
	if line > 0 {
		loc = fmt.Sprintf("%s:%d", file, line)
	}
	switch p.v {
	case Badges:
		p.line(" " + badge(p.color, statusColor(st), map[Status]string{Warn: " WARN ", Fail: " FAIL ", OK: "  OK  ", Info: " INFO "}[st]) + "  " + loc)
		p.line("          " + msg)
	case Checklist:
		at := file
		if line > 0 {
			at = fmt.Sprintf("%s line %d", file, line)
		}
		p.line(p.icon(st) + "  " + p.c(statusColor(st), at))
		p.line("    " + msg)
	case Steps:
		at := file
		if line > 0 {
			at += p.c(dim, fmt.Sprintf(":%d", line))
		}
		p.tree(p.mark(st) + " " + at + "  " + msg)
	default:
		p.line("  " + p.mark(st) + " " + p.c(dim, loc) + "  " + msg)
	}
}

// Break separates groups: a blank line, or the tree's trunk in steps.
func (p *Printer) Break() {
	if p.v == Steps {
		p.flush("├─")
		p.line(p.c(dim, "│"))
		return
	}
	p.line("")
}

// Summary closes the output: head (Done, Failed; "" colours the whole text)
// and text, then next (what to open) in the looks that show it.
func (p *Printer) Summary(st Status, head, text, next string) {
	col := statusColor(st)
	if head == "" {
		head, text = text, ""
	}
	join := func(sep string) string {
		if text == "" {
			return p.c(col, head)
		}
		return p.c(col, head) + sep + text
	}
	switch p.v {
	case Badges:
		tag, rest := strings.ToUpper(head), text
		if text == "" {
			tag, rest = [...]string{"OK", "WARN", "FAIL", "INFO"}[st], p.c(col, head)
		}
		p.line(" " + badge(p.color, col, " "+tag+" ") + "  " + rest)
	case Checklist:
		p.line([...]string{"🎉", "⚠️ ", "❌", "👉"}[st] + "  " + join(" "))
		if next != "" {
			p.line("    " + next)
		}
	case Steps:
		p.flush("├─")
		if text != "" {
			head = strings.ToLower(head) // └─ done
		}
		p.line(p.c(dim, "└─") + " " + join(" "))
	default:
		p.line(join("  "))
		if next != "" {
			p.line(p.c(dim, "  "+next))
		}
	}
}

// End closes a steps tree that has no summary (si play's links).
func (p *Printer) End() { p.flush("└─") }

// Hint is a dim footer line: the keys.
func (p *Printer) Hint(text string) {
	p.End()
	p.line("")
	if p.v == Ledger {
		text = "  " + text
	} else if p.v == Badges {
		text = " " + text
	}
	p.line(p.c(dim, text))
}

// Text is a plain line in the printer's stream (not part of the tree).
func (p *Printer) Text(s string) {
	p.End()
	p.line(s)
}
