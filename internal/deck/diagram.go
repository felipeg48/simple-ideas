package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Delivery diagrams (RULES §38): ```kanban, ```gantt and ```sequence fences,
// written in plain lines, laid out here as elements (boxes, lines, texts)
// with their state after every step. The renderer (render/diagram.go) draws
// the elements in their first state and hands every state to
// web/diagram.js, which tweens between them. No HTML here.

// DiagramKinds are the fence names (and diagram: values) si draws.
var DiagramKinds = []string{"kanban", "gantt", "sequence"}

// IsDiagram reports whether the file is a diagram page (`type: diagram`):
// one frame, landscape (16:9) or portrait (3:4), around its diagrams.
func (d *Deck) IsDiagram() bool { return d.Meta.Type == "diagram" }

// IsDiagramKind reports whether a fence info string names a delivery diagram.
func IsDiagramKind(lang string) bool {
	for _, k := range DiagramKinds {
		if k == lang {
			return true
		}
	}
	return false
}

// Diagram is a laid-out diagram: a viewBox, its elements and their states.
type Diagram struct {
	Kind, Title string
	W, H        float64
	Els         []DEl
	In          []DState   // the entrance: where each element starts before the first state
	States      [][]DState // States[k][i]: element i after step k (k = 0: before any step)
	Steps       []string   // a label per step ("day 2")
	Facts       []string   // what si worked out (the critical path, WIP over the limit), shown under the diagram
}

// DEl is one element. K: box (a rect with up to two lines), bar (a box with
// a progress fill), line (a path drawn by P), ms (a milestone diamond),
// text, head (a column or lifeline header).
type DEl struct {
	K string  `json:"k"`
	C string  `json:"c,omitempty"` // CSS class (si-dg-…)
	D string  `json:"d,omitempty"` // a line's path, relative to its place
	R float64 `json:"r,omitempty"` // corner radius
	// TX, TY place a line's label (from where the line starts)
	TX float64 `json:"tx,omitempty"`
	TY float64 `json:"ty,omitempty"`
}

// DState is an element in one state.
type DState struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w,omitempty"`
	H float64 `json:"h,omitempty"`
	O float64 `json:"o"`           // opacity
	P float64 `json:"p,omitempty"` // a line: how much is drawn (0..1); a bar: its progress
	C string  `json:"c,omitempty"` // classes this state adds (is-over, is-blocked, is-late, is-on)
	T string  `json:"t,omitempty"` // the first line of text
	U string  `json:"u,omitempty"` // the second line
}

// ParseDiagram reads and lays out a diagram fence of kind with its fence
// options; problems name the line.
func ParseDiagram(kind string, attrs map[string]string, body string) (*Diagram, []string) {
	switch kind {
	case "kanban":
		return parseKanban(attrs, body)
	case "gantt":
		return parseGantt(attrs, body)
	case "sequence":
		return parseSequence(attrs, body)
	}
	return &Diagram{Kind: kind, W: 400, H: 100, States: [][]DState{{}}}, []string{fmt.Sprintf("%q is not a diagram (kanban, gantt, sequence)", kind)}
}

// dgWidth is the layout width: the fence's width= (a portrait diagram page
// passes a narrow one), else def.
func dgWidth(attrs map[string]string, def float64) float64 {
	if v, err := strconv.ParseFloat(attrs["width"], 64); err == nil && v >= 300 && v <= 3000 {
		return v
	}
	return def
}

// diagramLines is the fence body as trimmed lines without comments (//).
func diagramLines(body string) []string {
	var out []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// dotFields splits "a · b · c" into trimmed, non-empty fields.
func dotFields(s string) []string {
	var out []string
	for _, f := range strings.Split(s, "·") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// arrowRE finds →, ->, -->, ⇢.
var arrowRE = regexp.MustCompile(`\s*(-->|->|→|⇢)\s*`)

// stepRE starts a step: day 2, step, sprint 3, week 2, iteration 4.
var stepRE = regexp.MustCompile(`(?i)^(day|step|sprint|week|iteration)\b(.*)$`)

func dsCopy(s []DState) []DState { return append([]DState(nil), s...) }

// ---------- kanban ----------

type kCard struct {
	id, title, who, tag string
	col                 int
	blocked             string
	order               int // when it arrived in its column
	shown               bool
}

var cardIDRE = regexp.MustCompile(`^[A-Z][A-Z0-9]*-\d+$`)

const (
	kPad, kGap, kHead, kCardH, kCardGap = 12.0, 12.0, 40.0, 50.0, 8.0
)

func parseKanban(attrs map[string]string, body string) (*Diagram, []string) {
	d := &Diagram{Kind: "kanban", Title: attrs["title"]}
	kW := dgWidth(attrs, 1000)
	var probs []string
	var cols []string
	wip := map[int]int{}
	cards := map[string]*kCard{}
	var keys []string
	seq := 0
	colOf := func(name string) int {
		for i, c := range cols {
			if strings.EqualFold(c, strings.TrimSpace(name)) {
				return i
			}
		}
		return -1
	}
	readWIP := func(v string) {
		for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' }) {
			name, n, ok := strings.Cut(strings.TrimSpace(p), ":")
			if !ok {
				f := strings.Fields(p)
				if len(f) < 2 {
					continue
				}
				name, n = strings.Join(f[:len(f)-1], " "), f[len(f)-1]
			}
			i, err := strconv.Atoi(strings.TrimSpace(n))
			if c := colOf(name); c >= 0 && err == nil && i > 0 {
				wip[c] = i
			} else {
				probs = append(probs, fmt.Sprintf("kanban wip %q: want Column:limit, for a column of the board", strings.TrimSpace(p)))
			}
		}
	}
	type kStep struct {
		label string
		lines []string
	}
	var steps []kStep
	var base []string
	wipLine := attrs["wip"]
	for _, l := range diagramLines(body) {
		low := strings.ToLower(l)
		switch {
		case strings.HasPrefix(low, "columns:"):
			for _, c := range strings.Split(l[len("columns:"):], ",") {
				if c = strings.TrimSpace(c); c != "" {
					cols = append(cols, c)
				}
			}
		case strings.HasPrefix(low, "wip:"):
			wipLine = l[len("wip:"):] // read once the columns are known
		case stepRE.MatchString(l):
			steps = append(steps, kStep{label: l})
		case len(steps) > 0:
			steps[len(steps)-1].lines = append(steps[len(steps)-1].lines, l)
		default:
			base = append(base, l)
		}
	}
	if len(cols) == 0 {
		return &Diagram{Kind: "kanban", W: 400, H: 60, States: [][]DState{{}}}, []string{"kanban: name the columns first: columns: Backlog, Doing, Done"}
	}
	if wipLine != "" {
		readWIP(wipLine)
	}
	addCard := func(l string) {
		f := dotFields(l)
		if len(f) == 0 {
			return
		}
		c := &kCard{col: -1}
		head := f[0]
		if sp := strings.IndexByte(head, ' '); sp > 0 && cardIDRE.MatchString(head[:sp]) {
			c.id, c.title = head[:sp], strings.TrimSpace(head[sp:])
		} else if cardIDRE.MatchString(head) {
			c.id = head
		} else {
			c.id, c.title = head, ""
		}
		for _, o := range f[1:] {
			switch {
			case colOf(o) >= 0 && c.col < 0:
				c.col = colOf(o)
			case strings.HasPrefix(o, "@"):
				c.who = o
			case strings.HasPrefix(strings.ToLower(o), "blocked"):
				c.blocked = strings.TrimSpace(strings.TrimLeft(o[len("blocked"):], ": "))
				if c.blocked == "" {
					c.blocked = "blocked"
				}
			default:
				c.tag = o
			}
		}
		if c.col < 0 {
			probs = append(probs, fmt.Sprintf("kanban card %q: give it a column (%s)", c.id, strings.Join(cols, ", ")))
			c.col = 0
		}
		if cards[c.id] != nil {
			probs = append(probs, fmt.Sprintf("kanban card %q twice", c.id))
			return
		}
		seq++
		c.order, c.shown = seq, true
		cards[c.id] = c
		keys = append(keys, c.id)
	}
	for _, l := range base {
		addCard(l)
	}
	// every card that will ever exist gets an element: cards added in a step
	// are hidden before it
	snap := func() map[string]kCard {
		m := map[string]kCard{}
		for k, c := range cards {
			m[k] = *c
		}
		return m
	}
	snaps := []map[string]kCard{snap()}
	for _, st := range steps {
		for _, l := range st.lines {
			if strings.HasPrefix(l, "+") {
				addCard(strings.TrimSpace(l[1:]))
				continue
			}
			key, rest, _ := strings.Cut(l, " ")
			c := cards[key]
			if c == nil {
				probs = append(probs, fmt.Sprintf("kanban %s: %q is not a card on the board (add one with + ID Title · Column)", st.label, key))
				continue
			}
			rest = strings.TrimSpace(rest)
			switch {
			case arrowRE.MatchString(l):
				to := arrowRE.Split(l, 2)[1]
				if i := colOf(to); i >= 0 {
					if i != c.col {
						seq++
						c.col, c.order = i, seq
					}
				} else {
					probs = append(probs, fmt.Sprintf("kanban %s: %q is not a column (%s)", st.label, to, strings.Join(cols, ", ")))
				}
			case strings.HasPrefix(strings.ToLower(rest), "unblocked"):
				c.blocked = ""
			case strings.HasPrefix(strings.ToLower(rest), "blocked"):
				c.blocked = strings.TrimSpace(strings.TrimLeft(rest[len("blocked"):], ": "))
				if c.blocked == "" {
					c.blocked = "blocked"
				}
			case strings.EqualFold(rest, "done"):
				seq++
				c.col, c.order, c.blocked = len(cols)-1, seq, ""
			default:
				probs = append(probs, fmt.Sprintf("kanban %s: %q: write ID → Column, ID blocked: why, ID unblocked or ID done", st.label, l))
			}
		}
		snaps = append(snaps, snap())
		d.Steps = append(d.Steps, st.label)
	}
	// layout: the tallest column over every state sets the height
	n := len(cols)
	colW := (kW - 2*kPad - float64(n-1)*kGap) / float64(n)
	colX := func(i int) float64 { return kPad + float64(i)*(colW+kGap) }
	rows := 1
	place := func(s map[string]kCard) (map[string]int, []int) {
		slot := map[string]int{}
		count := make([]int, n)
		ordered := append([]string(nil), keys...)
		// by arrival in the column
		for i := 1; i < len(ordered); i++ {
			for j := i; j > 0 && s[ordered[j]].order < s[ordered[j-1]].order; j-- {
				ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
			}
		}
		for _, k := range ordered {
			c, ok := s[k]
			if !ok {
				continue
			}
			slot[k] = count[c.col]
			count[c.col]++
		}
		return slot, count
	}
	for _, s := range snaps {
		_, count := place(s)
		for _, c := range count {
			rows = max(rows, c)
		}
	}
	top := kPad
	if d.Title != "" {
		top += 22
	}
	d.W = kW
	d.H = top + kHead + float64(rows)*(kCardH+kCardGap) + kPad
	// elements: the column lanes, their headers, then the cards
	for range cols {
		d.Els = append(d.Els, DEl{K: "box", C: "si-dg-lane", R: 10})
	}
	for range cols {
		d.Els = append(d.Els, DEl{K: "box", C: "si-dg-colhead"})
	}
	for range keys {
		d.Els = append(d.Els, DEl{K: "box", C: "si-dg-card", R: 6})
	}
	over := map[string]bool{}
	for k, s := range snaps {
		slot, count := place(s)
		st := make([]DState, 0, len(d.Els))
		for i := range cols {
			c := ""
			if wip[i] > 0 && count[i] > wip[i] {
				c = "is-over"
				over[fmt.Sprintf("%s in %s", cols[i], labelOr(d.Steps, k))] = true
			}
			st = append(st, DState{X: colX(i), Y: top, W: colW, H: d.H - top - kPad, O: 1, C: c})
		}
		for i, name := range cols {
			t := fmt.Sprintf("%s  %d", name, count[i])
			if wip[i] > 0 {
				t = fmt.Sprintf("%s  %d/%d", name, count[i], wip[i])
			}
			c := ""
			if wip[i] > 0 && count[i] > wip[i] {
				c = "is-over"
			}
			st = append(st, DState{X: colX(i), Y: top, W: colW, H: kHead, O: 1, C: c, T: t})
		}
		for _, key := range keys {
			c, ok := s[key]
			if !ok {
				st = append(st, DState{X: colX(0) + 6, Y: top + kHead, W: colW - 12, H: kCardH, O: 0, T: cardTitle(cards[key])})
				continue
			}
			cls, sub := "", strings.TrimSpace(c.who+" "+c.tag)
			if c.blocked != "" {
				cls, sub = "is-blocked", "⚠ "+c.blocked
			}
			if c.col == n-1 {
				cls = strings.TrimSpace(cls + " is-done")
			}
			if k > 0 && snaps[k-1][key].col != c.col {
				cls = strings.TrimSpace(cls + " is-on")
			}
			st = append(st, DState{X: colX(c.col) + 6, Y: top + kHead + float64(slot[key])*(kCardH+kCardGap), W: colW - 12, H: kCardH, O: 1, C: cls, T: cardTitle(&c), U: sub})
		}
		d.States = append(d.States, st)
	}
	d.In = dsCopy(d.States[0])
	for i := 2 * n; i < len(d.In); i++ {
		d.In[i].O, d.In[i].Y = 0, d.In[i].Y+14
	}
	for k := range over {
		d.Facts = append(d.Facts, "Over its WIP limit: "+k)
	}
	sortStrings(d.Facts)
	if d.Title != "" {
		d.Els = append(d.Els, DEl{K: "text", C: "si-dg-title"})
		for k := range d.States {
			d.States[k] = append(d.States[k], DState{X: kPad, Y: kPad + 8, O: 1, T: strings.ToUpper(d.Title)})
		}
		d.In = append(d.In, d.States[0][len(d.States[0])-1])
	}
	return d, probs
}

func labelOr(steps []string, k int) string {
	if k == 0 {
		return "the first state"
	}
	return steps[k-1]
}

func cardTitle(c *kCard) string {
	if c.title == "" {
		return c.id
	}
	if c.id == "" || !cardIDRE.MatchString(c.id) {
		return c.title
	}
	return c.id + " " + c.title
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
