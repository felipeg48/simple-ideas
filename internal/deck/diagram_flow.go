package deck

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Flow metrics (RULES §38): ```cfd, ```burndown, ```cycle-time and
// ```throughput, drawn from a Markdown table (or, for cycle-time, one line
// per finished item) on a plot with axes. Steps reveal the data from left
// to right: a curtain (a box in the page's background colour) slides off
// the bands and lines, dots pop in, columns grow. What si works out (the
// throughput, the pace and the finish it points to, the percentiles) is the
// caption and, for cycle-time, the percentile lines.

// plot is the drawing area of a metric.
type plot struct {
	x0, y0, x1, y1 float64 // left, top, right, bottom
	max            float64 // the y axis top
}

func (p plot) sy(v float64) float64 { return p.y1 - (p.y1-p.y0)*v/p.max }

// niceMax rounds v up to 1, 2, 2.5 or 5 × 10^n, and gives a tick step.
func niceMax(v float64) (float64, float64) {
	if v <= 0 {
		return 1, 1
	}
	e := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if m*e >= v {
			return m * e, m * e / 5
		}
	}
	return 10 * e, 2 * e
}

// mdTable reads a Markdown table: its header and its rows of cells.
func mdTable(body string) (head []string, rows [][]string) {
	for _, l := range diagramLines(body) {
		if !strings.HasPrefix(l, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if strings.Trim(strings.Join(cells, ""), "-: ") == "" {
			continue // the separator
		}
		if head == nil {
			head = cells
		} else {
			rows = append(rows, cells)
		}
	}
	return head, rows
}

func numCell(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(s, "d"), "%")), 64)
	return v, err == nil
}

// revealSteps splits n points into the points each step reveals up to: one
// step per point up to 12, else about 8 even steps. The first state shows
// the first point (or two).
func revealSteps(n int) []int {
	var out []int
	if n <= 12 {
		for i := 1; i < n; i++ {
			out = append(out, i)
		}
		return out
	}
	for s := 1; s <= 8; s++ {
		out = append(out, int(math.Round(float64(n-1)*float64(s)/8)))
	}
	return out
}

// metric builds the common parts: the diagram, the title, the plot, the y
// axis and the x labels; add appends an element and its first state.
type metric struct {
	d   *Diagram
	p   plot
	st0 []DState
}

func newMetric(kind string, attrs map[string]string, yMax float64, xLabels []string, xAt func(i int) float64, unit string) *metric {
	m := &metric{d: &Diagram{Kind: kind, Title: attrs["title"]}}
	W := dgWidth(attrs, 1000)
	top := 12.0
	if m.d.Title != "" {
		top += 22
	}
	H := math.Round(W * 0.42)
	mx, step := niceMax(yMax * 1.05)
	m.p = plot{x0: 56, y0: top + 10, x1: W - 16, y1: top + H, max: mx}
	m.d.W, m.d.H = W, m.p.y1+34
	if m.d.Title != "" {
		m.add(DEl{K: "text", C: "si-dg-title"}, DState{X: 12, Y: 20, O: 1, T: strings.ToUpper(m.d.Title)})
	}
	for v := 0.0; v <= mx+1e-9; v += step {
		y := m.p.sy(v)
		m.add(DEl{K: "line", C: "si-dg-grid", D: fmt.Sprintf("M0 0H%g", m.p.x1-m.p.x0)}, DState{X: m.p.x0, Y: y, O: 1, P: 1})
		m.add(DEl{K: "text", C: "si-dg-ylab"}, DState{X: m.p.x0 - 8, Y: y, O: 1, T: trimNum(v)})
	}
	if unit != "" {
		m.add(DEl{K: "text", C: "si-dg-unit"}, DState{X: m.p.x0 - 8, Y: m.p.y0 - 14, O: 1, T: unit})
	}
	every := max(1, int(math.Ceil(float64(len(xLabels))/10)))
	for i, l := range xLabels {
		if i%every != 0 && i != len(xLabels)-1 {
			continue
		}
		m.add(DEl{K: "text", C: "si-dg-xlab"}, DState{X: xAt(i), Y: m.p.y1 + 16, O: 1, T: l})
	}
	return m
}

func (m *metric) add(e DEl, s DState) int {
	m.d.Els = append(m.d.Els, e)
	m.st0 = append(m.st0, s)
	return len(m.d.Els) - 1
}

func trimNum(v float64) string {
	if v == math.Trunc(v) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// curtain is the clip of the data (elements with Clip): what is left of x
// is shown.
func (m *metric) curtain() int {
	return m.add(DEl{K: "clip"}, DState{X: m.p.x0 - 4, Y: m.p.y0 - 14, W: 0, H: m.p.y1 - m.p.y0 + 16, O: 1})
}

func (m *metric) openTo(st []DState, c int, x float64) {
	st[c].W = math.Max(0, x-(m.p.x0-4)+1)
}

func polyline(pts [][2]float64) string {
	var b strings.Builder
	for i, p := range pts {
		if i == 0 {
			fmt.Fprintf(&b, "M%.1f %.1f", p[0], p[1])
		} else {
			fmt.Fprintf(&b, "L%.1f %.1f", p[0], p[1])
		}
	}
	return b.String()
}

// ---------- cfd ----------

func parseCFD(attrs map[string]string, body string) (*Diagram, []string) {
	head, rows := mdTable(body)
	if len(head) < 3 || len(rows) < 2 {
		return emptyDiagram("cfd"), []string{"cfd: a table with a first column (the day) and a column per state, Backlog first and Done last, two rows at least"}
	}
	var probs []string
	states := head[1:]
	n := len(rows)
	val := make([][]float64, n)
	var top float64
	var labels []string
	for i, r := range rows {
		labels = append(labels, r[0])
		val[i] = make([]float64, len(states))
		sum := 0.0
		for j := range states {
			if j+1 < len(r) {
				v, ok := numCell(r[j+1])
				if !ok {
					probs = append(probs, fmt.Sprintf("cfd row %q: %q is not a number", r[0], r[j+1]))
				}
				val[i][j] = v
			}
			sum += val[i][j]
		}
		top = math.Max(top, sum)
	}
	xAt := func(i int) float64 { return 56 + (dgWidth(attrs, 1000)-16-56)*float64(i)/float64(n-1) }
	m := newMetric("cfd", attrs, top, labels, xAt, attrs["unit"])
	// bands from the last state (Done) at the bottom up to the first (Backlog)
	cum := func(i, j int) float64 { // the top of band j on row i: states j..last
		s := 0.0
		for k := j; k < len(states); k++ {
			s += val[i][k]
		}
		return s
	}
	for j := range states {
		var pts [][2]float64
		for i := 0; i < n; i++ {
			pts = append(pts, [2]float64{xAt(i), m.p.sy(cum(i, j))})
		}
		for i := n - 1; i >= 0; i-- {
			below := 0.0
			if j+1 < len(states) {
				below = cum(i, j+1)
			}
			pts = append(pts, [2]float64{xAt(i), m.p.sy(below)})
		}
		m.add(DEl{K: "shape", C: "si-dg-band " + bandClass(j, len(states)), D: polyline(pts) + "Z", Clip: true}, DState{O: 1})
	}
	// the legend, right to left in reading order
	lx, ly := m.p.x0, m.p.y1+34
	for j, s := range states {
		m.add(DEl{K: "box", C: "si-dg-key " + bandClass(j, len(states)), R: 2}, DState{X: lx, Y: ly, W: 10, H: 10, O: 1})
		m.add(DEl{K: "text", C: "si-dg-keylab"}, DState{X: lx + 14, Y: ly + 5, O: 1, T: s})
		lx += 26 + 7.5*float64(len(s))
	}
	m.d.H = ly + 18
	c := m.curtain()
	first := 1
	m.openTo(m.st0, c, xAt(first))
	m.d.States = append(m.d.States, m.st0)
	for _, k := range revealSteps(n) {
		if k <= first {
			continue
		}
		st := dsCopy(m.st0)
		m.openTo(st, c, xAt(k))
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, labels[k])
	}
	m.d.In = dsCopy(m.st0)
	m.openTo(m.d.In, c, m.p.x0)
	// what it says: throughput, WIP, and lead time by Little's law
	last := len(states) - 1
	days := float64(n - 1)
	done := val[n-1][last] - val[0][last]
	wip := 0.0
	for j := 1; j < last; j++ {
		wip += val[n-1][j]
	}
	if done > 0 && days > 0 {
		thr := done / days
		r, unit := math.Round(thr*10)/10, orStr(attrs["unit"], "items")
		if r == 1 {
			unit = strings.TrimSuffix(unit, "s")
		}
		m.d.Facts = append(m.d.Facts, fmt.Sprintf("Throughput %s %s a %s", trimNum(r), unit, orStr(attrs["per"], "day")))
		if wip > 0 {
			m.d.Facts = append(m.d.Facts, fmt.Sprintf("In progress now %s, so about %s %ss from start to %s", trimNum(wip), trimNum(math.Round(wip/thr*10)/10), orStr(attrs["per"], "day"), states[last]))
		}
	}
	return m.d, probs
}

// bandClass colours a cfd state: the first (the backlog) muted, the last
// (done) green, the one before it the accent, then the second accent …
func bandClass(j, n int) string {
	if j == 0 && n > 2 {
		return "is-cb"
	}
	return fmt.Sprintf("is-c%d", (n-1-j)%6)
}

func emptyDiagram(kind string) *Diagram {
	return &Diagram{Kind: kind, W: 400, H: 60, States: [][]DState{{}}}
}

// ---------- burndown ----------

func parseBurndown(attrs map[string]string, body string) (*Diagram, []string) {
	head, rows := mdTable(body)
	if len(head) < 2 || len(rows) < 2 {
		return emptyDiagram("burndown"), []string{"burndown: a table | Day | Remaining |, two rows at least (a third column: the scope)"}
	}
	var probs []string
	n := len(rows)
	var rem, scope []float64
	var labels []string
	top := 0.0
	for _, r := range rows {
		labels = append(labels, r[0])
		v, ok := numCell(r[1])
		if !ok {
			probs = append(probs, fmt.Sprintf("burndown row %q: %q is not a number", r[0], r[1]))
		}
		rem = append(rem, v)
		top = math.Max(top, v)
		if len(r) > 2 && len(head) > 2 {
			if s, ok := numCell(r[2]); ok {
				scope = append(scope, s)
				top = math.Max(top, s)
			}
		}
	}
	// the planned end: end= rows (the sprint's length), else the table's
	total := n
	if v, err := strconv.Atoi(attrs["days"]); err == nil && v >= n {
		total = v
	}
	W := dgWidth(attrs, 1000)
	xAt := func(i int) float64 { return 56 + (W-16-56)*float64(i)/float64(total-1) }
	all := append([]string(nil), labels...)
	last, numeric := 0, true
	if v, err := strconv.Atoi(labels[n-1]); err == nil {
		last = v
	} else {
		numeric = false
	}
	for len(all) < total { // the planned days still to come
		last++
		if numeric {
			all = append(all, strconv.Itoa(last))
		} else {
			all = append(all, "")
		}
	}
	m := newMetric("burndown", attrs, top, all, xAt, attrs["unit"])
	ideal := m.add(DEl{K: "line", C: "si-dg-ideal", D: fmt.Sprintf("M0 0L%.1f %.1f", xAt(total-1)-xAt(0), m.p.sy(0)-m.p.sy(rem[0]))}, DState{X: xAt(0), Y: m.p.sy(rem[0]), O: 1, P: 1})
	_ = ideal
	var pts, spts [][2]float64
	for i, v := range rem {
		pts = append(pts, [2]float64{xAt(i), m.p.sy(v)})
	}
	for i, v := range scope {
		spts = append(spts, [2]float64{xAt(i), m.p.sy(v)})
	}
	if len(spts) > 1 {
		m.add(DEl{K: "shape", C: "si-dg-scope", D: polyline(spts), Clip: true}, DState{O: 1})
	}
	m.add(DEl{K: "shape", C: "si-dg-actual", D: polyline(pts), Clip: true}, DState{O: 1})
	c := m.curtain()
	m.openTo(m.st0, c, xAt(0)+3)
	m.d.States = append(m.d.States, m.st0)
	for _, k := range revealSteps(n) {
		st := dsCopy(m.st0)
		m.openTo(st, c, xAt(k)+3)
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, labels[k])
	}
	m.d.In = dsCopy(m.st0)
	// the pace and where it points
	burned := rem[0] - rem[n-1]
	if burned > 0 {
		pace := burned / float64(n-1)
		at := float64(n-1) + rem[n-1]/pace
		f := fmt.Sprintf("Burned %s of %s in %d %ss; at this pace done by %s %d", trimNum(burned), trimNum(rem[0]), n-1, orStr(attrs["per"], "day"), orStr(attrs["per"], "day"), int(math.Ceil(at))+1)
		if total > n || attrs["days"] != "" {
			f += fmt.Sprintf(" (plan: %d)", total)
		}
		m.d.Facts = append(m.d.Facts, f)
	} else {
		m.d.Facts = append(m.d.Facts, "Nothing burned yet")
	}
	return m.d, probs
}

// ---------- cycle-time ----------

var ctLineRE = regexp.MustCompile(`^(.+?)\s*·\s*(\d{4}-\d{2}-\d{2})\s*·\s*(\d+(?:\.\d+)?)\s*d?$`)

func parseCycleTime(attrs map[string]string, body string) (*Diagram, []string) {
	type item struct {
		id   string
		done time.Time
		days float64
	}
	var items []item
	var probs []string
	head, rows := mdTable(body)
	if head != nil {
		for _, r := range rows {
			if len(r) < 3 {
				continue
			}
			t, err1 := time.Parse("2006-01-02", r[1])
			v, ok := numCell(r[2])
			if err1 != nil || !ok {
				probs = append(probs, fmt.Sprintf("cycle-time row %q: want | Item | Finished YYYY-MM-DD | Days |", r[0]))
				continue
			}
			items = append(items, item{r[0], t, v})
		}
	} else {
		for _, l := range diagramLines(body) {
			mm := ctLineRE.FindStringSubmatch(l)
			if mm == nil {
				probs = append(probs, fmt.Sprintf("cycle-time %q: write ID · finished YYYY-MM-DD · days (LOGIN-12 · 2026-12-03 · 4d)", l))
				continue
			}
			t, _ := time.Parse("2006-01-02", mm[2])
			v, _ := strconv.ParseFloat(mm[3], 64)
			items = append(items, item{mm[1], t, v})
		}
	}
	if len(items) < 2 {
		return emptyDiagram("cycle-time"), append(probs, "cycle-time: two finished items at least")
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].done.Before(items[b].done) })
	first, last := items[0].done, items[len(items)-1].done
	span := math.Max(1, last.Sub(first).Hours()/24)
	top := 0.0
	var days []float64
	for _, it := range items {
		top = math.Max(top, it.days)
		days = append(days, it.days)
	}
	W := dgWidth(attrs, 1000)
	xOf := func(t time.Time) float64 { return 56 + 14 + (W-16-56-28)*t.Sub(first).Hours()/24/span }
	var labels []string
	var at []time.Time
	for d := 0.0; d <= span; d += math.Max(1, math.Ceil(span/8)) {
		t := first.AddDate(0, 0, int(d))
		labels, at = append(labels, t.Format("Jan 2")), append(at, t)
	}
	m := newMetric("cycle-time", attrs, top, labels, func(i int) float64 { return xOf(at[i]) }, "days")
	// percentiles (nearest rank)
	sorted := append([]float64(nil), days...)
	sort.Float64s(sorted)
	pct := func(p float64) float64 { return sorted[max(0, int(math.Ceil(p/100*float64(len(sorted))))-1)] }
	var lines []int
	for _, p := range []float64{50, 85, 95} {
		v := pct(p)
		lines = append(lines, m.add(DEl{K: "line", C: "si-dg-pct", D: fmt.Sprintf("M0 0H%g", m.p.x1-m.p.x0), TX: m.p.x1 - m.p.x0, TY: -8}, DState{X: m.p.x0, Y: m.p.sy(v), T: fmt.Sprintf("%g%% · %s d", p, trimNum(v))}))
	}
	dot0 := len(m.d.Els)
	for _, it := range items {
		cls := "si-dg-dot"
		if it.days > pct(85) {
			cls += " is-late"
		}
		m.add(DEl{K: "dot", C: cls}, DState{X: xOf(it.done), Y: m.p.sy(it.days), O: 0, W: 0})
	}
	// steps: the dots by finish date, then the percentile lines
	groups := revealSteps(len(items) + 1)
	show := func(st []DState, upto int) {
		for i := range items {
			if i < upto {
				st[dot0+i].O, st[dot0+i].W = 1, 6
			}
		}
	}
	show(m.st0, 1)
	m.d.States = append(m.d.States, m.st0)
	for _, g := range groups {
		st := dsCopy(m.st0)
		show(st, g)
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, "to "+items[min(g, len(items))-1].done.Format("Jan 2"))
	}
	st := dsCopy(m.d.States[len(m.d.States)-1])
	show(st, len(items))
	for _, l := range lines {
		st[l].O, st[l].P = 1, 1
	}
	m.d.States = append(m.d.States, st)
	m.d.Steps = append(m.d.Steps, "percentiles")
	m.d.In = dsCopy(m.st0)
	m.d.Facts = append(m.d.Facts, fmt.Sprintf("85%% of %d items finished within %s days; half within %s", len(items), trimNum(pct(85)), trimNum(pct(50))))
	return m.d, probs
}

// ---------- throughput ----------

func parseThroughput(attrs map[string]string, body string) (*Diagram, []string) {
	head, rows := mdTable(body)
	if len(head) < 2 || len(rows) < 1 {
		return emptyDiagram("throughput"), []string{"throughput: a table | Week | Done |"}
	}
	var probs []string
	var vals []float64
	var labels []string
	top, sum := 0.0, 0.0
	for _, r := range rows {
		v, ok := numCell(r[1])
		if !ok {
			probs = append(probs, fmt.Sprintf("throughput row %q: %q is not a number", r[0], r[1]))
		}
		labels, vals = append(labels, r[0]), append(vals, v)
		top, sum = math.Max(top, v), sum+v
	}
	n := len(vals)
	W := dgWidth(attrs, 1000)
	slot := (W - 16 - 56) / float64(n)
	xAt := func(i int) float64 { return 56 + slot*(float64(i)+.5) }
	m := newMetric("throughput", attrs, top, labels, xAt, attrs["unit"])
	bw := math.Min(slot*.62, 70)
	col0 := len(m.d.Els)
	for i, v := range vals {
		m.add(DEl{K: "box", C: "si-dg-col", R: 3}, DState{X: xAt(i) - bw/2, Y: m.p.y1, W: bw, H: 0, O: 1, T: ""})
		_ = v
	}
	avg := sum / float64(n)
	avgEl := m.add(DEl{K: "line", C: "si-dg-avg", D: fmt.Sprintf("M0 0H%g", m.p.x1-m.p.x0), TX: m.p.x1 - m.p.x0, TY: -8}, DState{X: m.p.x0, Y: m.p.sy(avg), T: "average " + trimNum(math.Round(avg*10)/10)})
	grow := func(st []DState, upto int) {
		for i := 0; i < upto; i++ {
			h := m.p.y1 - m.p.sy(vals[i])
			st[col0+i].Y, st[col0+i].H = m.p.y1-h, h
		}
	}
	grow(m.st0, 1)
	m.d.States = append(m.d.States, m.st0)
	for i := 2; i <= n; i++ {
		st := dsCopy(m.st0)
		grow(st, i)
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, labels[i-1])
	}
	st := dsCopy(m.d.States[len(m.d.States)-1])
	st[avgEl].O, st[avgEl].P = 1, 1
	m.d.States = append(m.d.States, st)
	m.d.Steps = append(m.d.Steps, "average")
	m.d.In = dsCopy(m.st0)
	m.d.In[col0].H, m.d.In[col0].Y = 0, m.p.y1
	m.d.Facts = append(m.d.Facts, fmt.Sprintf("%s finished in %d %s, %s on average", trimNum(sum), n, orStr(attrs["per"], "weeks"), trimNum(math.Round(avg*10)/10)))
	return m.d, probs
}

// orStr is the first of its arguments that is not empty.
func orStr(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
