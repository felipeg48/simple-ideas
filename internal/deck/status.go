package deck

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Status boards (`type: status`, alias status-board; RULES.md §36): the
// front matter gives the meta pill row (meta:) and the KPI tiles (tiles:),
// the body one card per `## Area` with `### Done`, `### Missing` and
// `### Improve` lists, a `## Shipped` list (the chips row) and any other
// `## ` section as a note. Every count and bar is computed from the lists.

// ListKeys are the front matter keys whose items are `- …` lines under them.
var ListKeys = map[string]bool{"meta": true, "tiles": true}

// StatusColumns are the three columns of an area, in order.
var StatusColumns = []string{"Done", "Missing", "Improve"}

// StatusTones are the tones of a meta pill (`- git: on main | ok`).
var StatusTones = []string{"ok", "warn", "plain"}

// StatusTileKeys are the keys of a tile line (`- num: 25; lbl: …; sub: …`).
var StatusTileKeys = []string{"num", "gate", "lbl", "sub", "tone"}

// IsStatus reports whether the file is a status board.
func (d *Deck) IsStatus() bool { return d.Meta.Type == "status" }

// IsArchitecture reports whether the file is an architecture page.
func (d *Deck) IsArchitecture() bool { return d.Meta.Type == "architecture" }

// Status is a parsed status board.
type Status struct {
	Pills   []StatusPill
	Tiles   []StatusTile
	Intro   StatusNote // the Markdown before the first ## (Title empty)
	Areas   []StatusArea
	Shipped *StatusArea  // the ## Shipped section: its items are Cols[0]
	Notes   []StatusNote // every other ## section, in order
}

// StatusPill is one pill of the meta row: "label: value" with a tone.
type StatusPill struct{ Label, Value, Tone string }

// StatusTile is one KPI tile; Gate marks the wide highlight tile.
type StatusTile struct {
	Num, Label, Sub, Tone string
	Gate                  bool
}

// StatusNote is a ## section that is neither an area nor Shipped, kept as
// Markdown; Line is the source line of its first body line.
type StatusNote struct {
	Title, Body string
	Line        int
}

// StatusArea is one card: its title, the scope line under it, and the
// items of its three columns (Done, Missing, Improve).
type StatusArea struct {
	Title, Scope string
	Line         int
	Cols         [3][]StatusItem
}

// StatusItem is one list item: its text (inline Markdown) and the marks
// written as trailing code spans.
type StatusItem struct {
	Text  string
	Marks []StatusMark
	Line  int
}

// StatusMark is a trailing code span of an item: Kind is tag (P0-P9),
// part (partial, in progress, wip), gate (gate, gated, blocked), un
// (uncommitted), imp (improvement: a shipped Improve item) or code (a
// version · hash, an id).
type StatusMark struct{ Kind, Text string }

// StatusCounts are the item counts of one area or of the whole board.
type StatusCounts struct{ Done, Missing, Improve int }

// Counts are the area's item counts.
func (a StatusArea) Counts() StatusCounts {
	return StatusCounts{len(a.Cols[0]), len(a.Cols[1]), len(a.Cols[2])}
}

// Total is the overall count of every area.
func (s *Status) Total() StatusCounts {
	var t StatusCounts
	for _, a := range s.Areas {
		c := a.Counts()
		t.Done, t.Missing, t.Improve = t.Done+c.Done, t.Missing+c.Missing, t.Improve+c.Improve
	}
	return t
}

// Items is the number of items.
func (c StatusCounts) Items() int { return c.Done + c.Missing + c.Improve }

// Percent is how much is done: done ÷ (done + missing), rounded. Improve
// items make a thing better; they do not count against it being done. 100
// when nothing is missing.
func (c StatusCounts) Percent() int {
	if c.Done+c.Missing == 0 {
		return 100
	}
	return int(math.Round(float64(c.Done) * 100 / float64(c.Done+c.Missing)))
}

// Shares are the widths of the bar's three segments in percent of every
// item, rounded to two decimals (all zero for an empty area).
func (c StatusCounts) Shares() [3]float64 {
	t := c.Items()
	if t == 0 {
		return [3]float64{}
	}
	var out [3]float64
	for i, n := range []int{c.Done, c.Missing, c.Improve} {
		out[i] = math.Round(float64(n)*10000/float64(t)) / 100
	}
	return out
}

var (
	statusMarkRE = regexp.MustCompile("\\s*`([^`]+)`\\s*$")
	statusPrioRE = regexp.MustCompile(`^P\d$`)
	statusItemRE = regexp.MustCompile(`^\s*[-*+]\s+(.*)$`)
)

// ParseStatusItem splits an item's text from its trailing code-span marks.
func ParseStatusItem(s string) StatusItem {
	it := StatusItem{Text: strings.TrimSpace(s)}
	for {
		m := statusMarkRE.FindStringSubmatchIndex(it.Text)
		if m == nil || m[0] == 0 { // an item that is only a code span keeps it as its text
			break
		}
		v := strings.TrimSpace(it.Text[m[2]:m[3]])
		it.Text = strings.TrimSpace(it.Text[:m[0]])
		it.Marks = append([]StatusMark{{statusMarkKind(v), v}}, it.Marks...)
	}
	return it
}

func statusMarkKind(v string) string {
	switch strings.ToLower(v) {
	case "partial", "in progress", "wip":
		return "part"
	case "gate", "gated", "blocked":
		return "gate"
	case "uncommitted":
		return "un"
	case "improvement":
		return "imp"
	}
	if statusPrioRE.MatchString(v) {
		return "tag"
	}
	return "code"
}

// statusMeta reads meta:, tiles: and updated: (and warns when a page that
// is not a status board sets them, or a page that is not an architecture
// page sets number:).
func (d *Deck) statusMeta(file string, keyLine map[string]int) {
	if !d.IsStatus() {
		for _, k := range []string{"meta", "tiles", "updated"} {
			if ln, ok := keyLine[k]; ok {
				d.Warnings = append(d.Warnings, Problem{File: rel(file), Line: ln, Msg: fmt.Sprintf("front matter %q is for status boards (type: status)", k)})
			}
		}
	}
	if ln, ok := keyLine["number"]; ok && !d.IsArchitecture() {
		d.Warnings = append(d.Warnings, Problem{File: rel(file), Line: ln, Msg: `front matter "number" is for architecture pages (type: architecture)`})
	}
	if !d.IsStatus() {
		return
	}
	st := &Status{}
	for _, l := range d.Meta.Lists {
		if l.Key == "meta" {
			st.Pills = append(st.Pills, parsePill(l.Val))
			if t := st.Pills[len(st.Pills)-1].Tone; !contains(StatusTones, t) {
				d.problem(file, l.Line, "meta: unknown tone %q (use %s)", t, strings.Join(StatusTones, ", "))
			}
			continue
		}
		t, msg := parseTile(l.Val)
		if msg != "" {
			d.problem(file, l.Line, "tiles: %s", msg)
		}
		st.Tiles = append(st.Tiles, t)
	}
	d.Status = st
}

// parsePill reads "label: value | tone"; without a label the whole text
// is the value, without a tone it is plain.
func parsePill(s string) StatusPill {
	p := StatusPill{Tone: "plain"}
	if i := strings.LastIndex(s, "|"); i >= 0 {
		p.Tone, s = strings.TrimSpace(s[i+1:]), s[:i]
	}
	s = strings.TrimSpace(s)
	if l, v, ok := strings.Cut(s, ": "); ok {
		p.Label, p.Value = strings.TrimSpace(l), strings.TrimSpace(v)
	} else {
		p.Value = s
	}
	return p
}

// parseTile reads "num: 25; lbl: Sessions; sub: …" (or gate: in place of
// num:, the wide highlight tile); the message says what is wrong.
func parseTile(s string) (StatusTile, string) {
	var t StatusTile
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, ":")
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if !ok {
			return t, fmt.Sprintf("%q is not key: value (use %s, separated by ;)", part, strings.Join(StatusTileKeys, ", "))
		}
		switch k {
		case "num":
			t.Num = v
		case "gate":
			t.Num, t.Gate = v, true
		case "lbl", "label":
			t.Label = v
		case "sub":
			t.Sub = v
		case "tone":
			if !contains(StatusTones, v) {
				return t, fmt.Sprintf("unknown tone %q (use %s)", v, strings.Join(StatusTones, ", "))
			}
			t.Tone = v
		default:
			return t, fmt.Sprintf("unknown key %q (use %s)", k, strings.Join(StatusTileKeys, ", "))
		}
	}
	if t.Num == "" {
		return t, "a tile needs num: (or gate:)"
	}
	return t, ""
}

// buildStatus reads the body into areas, the Shipped list and notes, and
// reports what does not fit: an unknown ### column, an item outside a
// column, a column written twice, a board without areas.
func (d *Deck) buildStatus(lines []srcLine) {
	st := d.Status
	if st == nil {
		st = &Status{}
		d.Status = st
	}
	type section struct {
		title string
		line  int
		body  []srcLine
	}
	var intro []srcLine
	var secs []section
	var ft fenceTracker
	for _, l := range lines {
		t := strings.TrimSpace(l.text)
		if !ft.inFence(l.text) && strings.HasPrefix(t, "## ") {
			secs = append(secs, section{strings.TrimSpace(t[3:]), l.no, nil})
			continue
		}
		if len(secs) == 0 {
			intro = append(intro, l)
		} else {
			secs[len(secs)-1].body = append(secs[len(secs)-1].body, l)
		}
	}
	st.Intro = noteOf("", intro, d.Title())
	for _, s := range secs {
		if strings.HasPrefix(strings.ToLower(s.title), "shipped") {
			a := &StatusArea{Title: s.title, Line: s.line}
			a.Cols[0] = statusItems(s.body)
			st.Shipped = a
			continue
		}
		hasCol := false
		for _, l := range s.body {
			if strings.HasPrefix(strings.TrimSpace(l.text), "### ") {
				hasCol = true
				break
			}
		}
		if !hasCol { // a ## section without columns: a note (a list in it is not counted)
			st.Notes = append(st.Notes, noteOf(s.title, s.body, ""))
			continue
		}
		a := StatusArea{Title: s.title, Line: s.line}
		col := -1
		seen := [3]bool{}
		var scope []string
		var ft fenceTracker
		var cur []srcLine
		flush := func() {
			if col >= 0 {
				a.Cols[col] = append(a.Cols[col], statusItems(cur)...)
			}
			cur = nil
		}
		for _, l := range s.body {
			t := strings.TrimSpace(l.text)
			if ft.inFence(l.text) {
				continue
			}
			if h, ok := strings.CutPrefix(t, "### "); ok {
				flush()
				col = -1
				for i, c := range StatusColumns {
					if strings.EqualFold(strings.TrimSpace(h), c) {
						col = i
					}
				}
				if col < 0 {
					d.problem(l.file, l.no, "status area %q: unknown column ### %s (use ### Done, ### Missing or ### Improve)", s.title, strings.TrimSpace(h))
				} else if seen[col] {
					d.problem(l.file, l.no, "status area %q: ### %s is written twice", s.title, StatusColumns[col])
				} else {
					seen[col] = true
				}
				continue
			}
			if col < 0 {
				if statusItemRE.MatchString(l.text) {
					d.problem(l.file, l.no, "status area %q: a list item before ### Done, ### Missing or ### Improve", s.title)
				} else if t != "" {
					scope = append(scope, t)
				}
				continue
			}
			if t != "" && !statusItemRE.MatchString(l.text) && !strings.HasPrefix(l.text, " ") && !strings.HasPrefix(l.text, "\t") {
				d.Warnings = append(d.Warnings, Problem{File: rel(l.file), Line: l.no, Msg: fmt.Sprintf("status area %q: only list items count in ### %s; this line is left out", s.title, StatusColumns[col])})
				continue
			}
			cur = append(cur, l)
		}
		flush()
		a.Scope = strings.Join(scope, " ")
		if a.Counts().Items() == 0 {
			d.Warnings = append(d.Warnings, Problem{File: rel(d.Path), Line: s.line, Msg: fmt.Sprintf("status area %q has no items", s.title)})
		}
		st.Areas = append(st.Areas, a)
	}
	if len(st.Areas) == 0 {
		d.problem(d.Path, 0, "a status board needs at least one ## area with ### Done, ### Missing and ### Improve lists")
	}
}

// statusItems are the list items of lines; an indented line continues the
// item above it.
func statusItems(lines []srcLine) []StatusItem {
	var out []StatusItem
	var text []string
	line := 0
	flush := func() {
		if len(text) > 0 {
			it := ParseStatusItem(strings.Join(text, " "))
			it.Line = line
			out = append(out, it)
		}
		text = nil
	}
	for _, l := range lines {
		if m := statusItemRE.FindStringSubmatch(l.text); m != nil && !strings.HasPrefix(l.text, "  ") && !strings.HasPrefix(l.text, "\t") {
			flush()
			text, line = []string{m[1]}, l.no
		} else if t := strings.TrimSpace(l.text); t != "" && len(text) > 0 {
			text = append(text, strings.TrimLeft(strings.TrimSpace(t), "-*+ "))
		} else if t == "" {
			flush()
		}
	}
	flush()
	return out
}

// noteOf keeps a section's lines as Markdown; a first # line equal to the
// title (the intro's) is dropped, the header shows it.
func noteOf(title string, lines []srcLine, docTitle string) StatusNote {
	n := StatusNote{Title: title}
	var body []string
	for _, l := range lines {
		if len(body) == 0 && strings.TrimSpace(l.text) == "" {
			continue
		}
		if len(body) == 0 && docTitle != "" && strings.TrimSpace(l.text) == "# "+docTitle {
			continue
		}
		if n.Line == 0 {
			n.Line = l.no
		}
		body = append(body, l.text)
	}
	n.Body = strings.TrimSpace(strings.Join(body, "\n"))
	return n
}
