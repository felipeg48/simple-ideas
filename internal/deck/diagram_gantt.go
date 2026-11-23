package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ```gantt: tasks with lengths and what they wait for, scheduled here
// (calendar days), the critical path worked out; `today DATE · Task 60%`
// lines are steps that move the today line and fill the bars.

type gTask struct {
	name, section string
	dur           int
	after         []string
	from          time.Time
	ms            bool
	es, ef        int // earliest start and finish, days from the start
	crit          bool
	prog          float64
	line          string
}

var (
	gDurRE  = regexp.MustCompile(`^(\d+)\s*(d|w|day|days|week|weeks)$`)
	gProgRE = regexp.MustCompile(`^(.+?)\s+(\d{1,3})\s*%$`)
)

const (
	gPad, gLabelW, gAxis, gRow, gSecRow, gBarH = 12.0, 210.0, 30.0, 32.0, 26.0, 18.0
)

func parseGantt(attrs map[string]string, body string) (*Diagram, []string) {
	d := &Diagram{Kind: "gantt", Title: attrs["title"]}
	gW := dgWidth(attrs, 1000)
	var probs []string
	start, err := time.Parse("2006-01-02", attrs["start"])
	if err != nil {
		probs = append(probs, "gantt: give the first day: ```gantt start=2026-12-01")
		start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	var tasks []*gTask
	byName := map[string]*gTask{}
	type gStep struct {
		label string
		today time.Time
		prog  map[string]float64
	}
	var steps []gStep
	section := ""
	var rows []any // *gTask or a section name (string)
	setProg := func(m map[string]float64, spec, where string) {
		for _, p := range strings.Split(spec, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			name, v := p, 1.0
			if mm := gProgRE.FindStringSubmatch(p); mm != nil {
				n, _ := strconv.Atoi(mm[2])
				name, v = strings.TrimSpace(mm[1]), min(float64(n), 100)/100
			} else if strings.HasSuffix(strings.ToLower(p), " done") {
				name = strings.TrimSpace(p[:len(p)-5])
			} else {
				probs = append(probs, fmt.Sprintf("gantt %s: %q: write Task 60%% or Task done", where, p))
				continue
			}
			if byName[strings.ToLower(name)] == nil {
				probs = append(probs, fmt.Sprintf("gantt %s: %q is not a task", where, name))
				continue
			}
			m[strings.ToLower(name)] = v
		}
	}
	type later struct{ spec, where string }
	var pending []later // progress lines read once every task is known
	for _, l := range diagramLines(body) {
		low := strings.ToLower(l)
		switch {
		case strings.HasPrefix(low, "section "):
			section = strings.TrimSpace(l[len("section "):])
			rows = append(rows, section)
		case strings.HasPrefix(low, "today "):
			f := dotFields(l[len("today "):])
			t, err := time.Parse("2006-01-02", strings.TrimSpace(f[0]))
			if err != nil {
				probs = append(probs, fmt.Sprintf("gantt %q: today YYYY-MM-DD", l))
				continue
			}
			steps = append(steps, gStep{label: "today " + f[0], today: t, prog: map[string]float64{}})
			if len(f) > 1 {
				pending = append(pending, later{strings.Join(f[1:], ","), fmt.Sprintf("step %d", len(steps))})
			}
		case len(steps) > 0:
			pending = append(pending, later{l, fmt.Sprintf("step %d", len(steps))})
		default:
			f := dotFields(l)
			t := &gTask{name: f[0], section: section, line: l}
			for _, o := range f[1:] {
				lo := strings.ToLower(o)
				switch {
				case lo == "milestone":
					t.ms = true
				case gDurRE.MatchString(lo):
					m := gDurRE.FindStringSubmatch(lo)
					n, _ := strconv.Atoi(m[1])
					if m[2][0] == 'w' {
						n *= 7
					}
					t.dur = n
				case strings.HasPrefix(lo, "after "):
					for _, a := range strings.Split(o[len("after "):], ",") {
						if a = strings.TrimSpace(a); a != "" {
							t.after = append(t.after, a)
						}
					}
				case strings.HasPrefix(lo, "from "):
					if v, err := time.Parse("2006-01-02", strings.TrimSpace(o[5:])); err == nil {
						t.from = v
					} else {
						probs = append(probs, fmt.Sprintf("gantt task %q: from YYYY-MM-DD", t.name))
					}
				case strings.HasSuffix(lo, "%"):
					n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(lo, "%")))
					if err == nil {
						t.prog = min(float64(n), 100) / 100
					}
				default:
					probs = append(probs, fmt.Sprintf("gantt task %q: %q is not a length (5d, 2w), after …, from …, milestone or a %%", t.name, o))
				}
			}
			if t.dur == 0 && !t.ms {
				probs = append(probs, fmt.Sprintf("gantt task %q: give it a length (5d, 2w) or say milestone", t.name))
				t.dur = 1
			}
			if byName[strings.ToLower(t.name)] != nil {
				probs = append(probs, fmt.Sprintf("gantt task %q twice", t.name))
				continue
			}
			byName[strings.ToLower(t.name)] = t
			tasks = append(tasks, t)
			rows = append(rows, t)
		}
	}
	if len(tasks) == 0 {
		return &Diagram{Kind: "gantt", W: 400, H: 60, States: [][]DState{{}}}, append(probs, "gantt: no tasks (Design · 5d)")
	}
	for _, p := range pending {
		var k int
		fmt.Sscanf(p.where, "step %d", &k)
		setProg(steps[k-1].prog, p.spec, p.where)
	}
	// schedule: in written order, a task after the ones it waits for
	day := func(t time.Time) int { return int(t.Sub(start).Hours() / 24) }
	done := map[*gTask]bool{}
	var visit func(t *gTask, path map[*gTask]bool)
	visit = func(t *gTask, path map[*gTask]bool) {
		if done[t] {
			return
		}
		if path[t] {
			probs = append(probs, fmt.Sprintf("gantt task %q waits for itself", t.name))
			return
		}
		path[t] = true
		es := 0
		if !t.from.IsZero() {
			es = day(t.from)
		}
		for _, a := range t.after {
			p := byName[strings.ToLower(a)]
			if p == nil {
				probs = append(probs, fmt.Sprintf("gantt task %q: after %q, which is not a task", t.name, a))
				continue
			}
			visit(p, path)
			es = max(es, p.ef)
		}
		t.es, t.ef = es, es+t.dur
		done[t] = true
	}
	for _, t := range tasks {
		visit(t, map[*gTask]bool{})
	}
	end := 0
	for _, t := range tasks {
		end = max(end, t.ef)
	}
	// the critical path: no slack between a task and the end
	lf := map[*gTask]int{}
	for i := len(tasks) - 1; i >= 0; i-- {
		t := tasks[i]
		v := end
		for _, s := range tasks {
			for _, a := range s.after {
				if byName[strings.ToLower(a)] == t {
					v = min(v, lf[s]-s.dur)
				}
			}
		}
		lf[t] = v
	}
	var crit []string
	for _, t := range tasks {
		if lf[t]-t.dur == t.es {
			t.crit = true
			crit = append(crit, t.name)
		}
	}
	span := max(end, 1)
	for _, s := range steps {
		span = max(span, day(s.today)+1)
	}
	// layout
	top := gPad
	if d.Title != "" {
		top += 22
	}
	x0, x1 := gPad+gLabelW, gW-gPad
	sc := (x1 - x0) / float64(span)
	y := top + gAxis
	rowY := map[any]float64{}
	for _, r := range rows {
		rowY[r] = y
		if _, ok := r.(string); ok {
			y += gSecRow
		} else {
			y += gRow
		}
	}
	d.W, d.H = gW, y+gPad
	every := 1
	for _, e := range []int{1, 2, 7, 14, 28, 56} {
		every = e
		if span/e <= 10 {
			break
		}
	}
	var st0 []DState
	add := func(e DEl, s DState) { d.Els = append(d.Els, e); st0 = append(st0, s) }
	for k := 0; k <= span; k += every {
		x := x0 + float64(k)*sc
		add(DEl{K: "line", C: "si-dg-grid", D: fmt.Sprintf("M0 0V%g", d.H-gPad-top-gAxis+6)}, DState{X: x, Y: top + gAxis - 6, O: 1, P: 1})
		add(DEl{K: "text", C: "si-dg-tick"}, DState{X: x, Y: top + gAxis - 14, O: 1, T: start.AddDate(0, 0, k).Format("Jan 2")})
	}
	for _, r := range rows {
		switch v := r.(type) {
		case string:
			add(DEl{K: "text", C: "si-dg-section"}, DState{X: gPad, Y: rowY[r] + gSecRow/2 + 2, O: 1, T: strings.ToUpper(v)})
		case *gTask:
			add(DEl{K: "text", C: "si-dg-task"}, DState{X: gPad + 6, Y: rowY[r] + gRow/2, O: 1, T: v.name})
		}
	}
	barAt := len(d.Els)
	for _, t := range tasks {
		cls := ""
		if t.crit {
			cls = "is-crit"
		}
		if t.ms {
			add(DEl{K: "ms", C: "si-dg-ms " + cls}, DState{X: x0 + float64(t.es)*sc, Y: rowY[t] + gRow/2, O: 1})
			continue
		}
		add(DEl{K: "bar", C: "si-dg-bar " + cls, R: 4}, DState{X: x0 + float64(t.es)*sc, Y: rowY[t] + (gRow-gBarH)/2, W: float64(t.dur) * sc, H: gBarH, O: 1, P: t.prog})
	}
	todayAt := len(d.Els)
	todayLen := d.H - gPad - top - gAxis + 10
	add(DEl{K: "line", C: "si-dg-today", D: fmt.Sprintf("M0 0V%g", todayLen), TY: todayLen + 10}, DState{X: x0, Y: top + gAxis - 8, O: 0, P: 1})
	if d.Title != "" {
		add(DEl{K: "text", C: "si-dg-title"}, DState{X: gPad, Y: gPad + 8, O: 1, T: strings.ToUpper(d.Title)})
	}
	d.States = append(d.States, st0)
	prog := map[string]float64{}
	for _, t := range tasks {
		prog[strings.ToLower(t.name)] = t.prog
	}
	for _, s := range steps {
		st := dsCopy(st0)
		for k, v := range s.prog {
			prog[k] = v
		}
		td := day(s.today)
		for i, t := range tasks {
			e := &st[barAt+i]
			p := prog[strings.ToLower(t.name)]
			e.P = p
			var cls []string
			if p >= 1 {
				cls = append(cls, "is-done")
			} else if td >= t.ef && !t.ms {
				cls = append(cls, "is-late")
			}
			e.C = strings.Join(cls, " ")
		}
		st[todayAt].X, st[todayAt].O, st[todayAt].T = x0+float64(td)*sc, 1, s.today.Format("Jan 2")
		d.States = append(d.States, st)
		d.Steps = append(d.Steps, s.label)
	}
	d.In = dsCopy(st0)
	for i := range tasks {
		d.In[barAt+i].W, d.In[barAt+i].O = 0, map[bool]float64{true: 0, false: 1}[tasks[i].ms]
	}
	d.Facts = append(d.Facts, fmt.Sprintf("Ends %s, %d days from the start", start.AddDate(0, 0, end).Format("Jan 2, 2006"), end))
	if len(crit) > 0 {
		d.Facts = append(d.Facts, "Critical path: "+strings.Join(crit, " → "))
	}
	return d, probs
}
