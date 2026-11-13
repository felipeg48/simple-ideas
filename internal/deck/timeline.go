package deck

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Timelines (`type: timeline` pages and the ```timeline fence): eras
// (`## Name · color=#hex`), milestones (`### date · title`, a description and
// an optional `![caption](src "credit")`) and a sources block
// (`## Fuentes · sources`). Dates are kept as written and read only for order.

// Timeline layouts, orientations and scales.
var (
	TimelineLayouts      = []string{"snake", "line", "vertical"}
	TimelineOrientations = []string{"landscape", "portrait"}
	TimelineScales       = []string{"even", "time"}
)

// TimelineMeta is the front matter of a `type: timeline` page that no other
// type uses (title, subtitle, lang, layout, aspect and theme are shared keys).
type TimelineMeta struct {
	Alternate   string // on (default) | off: milestones on both sides of the line
	Scale       string // even (default) | time: gaps by real distance
	Orientation string // landscape (default: a 16:9 slide) | portrait (a scrolling poster)
	Closing     string // the line at the end of the timeline
	Require     string // si check: "20 dates, 15 images"
	Steps       string // eras (default on a page): one build step per era; off
}

var timelineKeys = []string{"alternate", "scale", "orientation", "closing", "require", "steps"}

func init() { MetaKeys = append(MetaKeys, timelineKeys...) }

func (t *TimelineMeta) field(key string) *string {
	return map[string]*string{"alternate": &t.Alternate, "scale": &t.Scale, "orientation": &t.Orientation,
		"closing": &t.Closing, "require": &t.Require, "steps": &t.Steps}[key]
}

// IsTimeline reports whether the file is a timeline page (`type: timeline`).
func (d *Deck) IsTimeline() bool { return d.Meta.Type == "timeline" }

// isPoster: a portrait timeline is a document (a scrolling page), a landscape
// one a deck of one slide.
func (d *Deck) isPoster() bool {
	return d.IsTimeline() && d.Meta.Timeline.Orientation == "portrait"
}

// timelineMeta checks the timeline keys once the type is known.
func (d *Deck) timelineMeta(file string, keyLine map[string]int) {
	if !d.IsTimeline() {
		for _, k := range timelineKeys {
			if ln, ok := keyLine[k]; ok {
				d.Warnings = append(d.Warnings, Problem{File: rel(file), Line: ln, Msg: fmt.Sprintf("front matter %q is for timelines (type: timeline)", k)})
			}
		}
		if ln, ok := keyLine["layout"]; ok && d.IsMindmap() && contains(TimelineLayouts, d.Meta.MapLayout) {
			d.problem(file, ln, "%s", oneOf("mind map layout", d.Meta.MapLayout, MapLayouts))
		}
		return
	}
	t := d.Meta.Timeline
	check := func(key, val, what string, allowed []string) {
		if ln, ok := keyLine[key]; ok {
			if msg := oneOf(what, val, allowed); msg != "" {
				d.problem(file, ln, "%s", msg)
			}
		}
	}
	check("layout", d.Meta.MapLayout, "timeline layout", TimelineLayouts)
	check("alternate", t.Alternate, "alternate", OnOff)
	check("scale", t.Scale, "timeline scale", TimelineScales)
	check("orientation", t.Orientation, "orientation", TimelineOrientations)
	check("steps", t.Steps, "timeline steps", []string{"eras", "off"})
	if ln, ok := keyLine["require"]; ok {
		if _, err := ParseRequire(t.Require); err != nil {
			d.problem(file, ln, "%v", err)
		}
	}
}

// Timeline is a parsed timeline.
type Timeline struct {
	Title        string // a `# Title` line (a fence; a page has title:)
	Eras         []*TimelineEra
	SourcesTitle string   // the heading of the sources block ("Fuentes")
	Sources      []string // its items, inline Markdown
}

// TimelineEra is one `## Era`: a colour band and its milestones.
type TimelineEra struct {
	Name  string // "" for milestones written before any era
	Color string // color= (a CSS colour), "" for the theme's palette
	Line  int
	Items []*Milestone
}

// Milestone is one `### date · title`.
type Milestone struct {
	Date    string // as written: "c. 300 a.C.", "2022–2024"
	Title   string
	Text    string // the description (inline Markdown)
	Img     string // picture: a path relative to the file or an https:// URL
	Caption string // its alt text
	Credit  string // its title: "Wikimedia Commons, dominio público"
	Year    float64
	Dated   bool // Year was read from Date
	Line    int
}

var (
	tlMilestoneRE = regexp.MustCompile(`^ {0,3}###\s+(.+?)\s*#*\s*$`)
	tlImageRE     = regexp.MustCompile(`^\s*!\[([^\]]*)\]\(\s*<?([^\s)>]+)>?(?:\s+"([^"]*)")?\s*\)\s*$`)
	tlItemRE      = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.*)$`)
)

// splitDot splits "a · b · c" on the middle dot.
func splitDot(s string) []string {
	parts := strings.Split(s, "·")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// ParseTimeline reads timeline Markdown (the body of a page or a fence);
// first is the line number of its first line, file names it in problems.
// Problems are errors (a milestone without a title); warnings are advice.
func ParseTimeline(text, file string, first int) (*Timeline, []Problem, []Problem) {
	lines := splitLines(text)
	nos := make([]int, len(lines))
	for i := range nos {
		nos[i] = first + i
	}
	return parseTimeline(lines, nos, file)
}

func parseTimeline(lines []string, nos []int, file string) (*Timeline, []Problem, []Problem) {
	t := &Timeline{}
	var probs, warns []Problem
	at := func(n int) Problem { return Problem{File: rel(file), Line: nos[n]} }
	var era *TimelineEra
	var ms *Milestone
	inSources := false
	var desc []string
	flush := func() {
		if ms != nil {
			ms.Text = strings.TrimSpace(strings.Join(desc, " "))
		}
		desc = nil
	}
	var ft fenceTracker
	for i, line := range lines {
		if ft.inFence(line) {
			continue
		}
		trim := strings.TrimSpace(line)
		switch {
		case h1RE.MatchString(line) && !strings.HasPrefix(trim, "##"):
			flush()
			ms = nil
			t.Title = h1RE.FindStringSubmatch(line)[1]
		case h2RE.MatchString(line) && !strings.HasPrefix(trim, "###"):
			flush()
			ms = nil
			parts := splitDot(h2RE.FindStringSubmatch(line)[1])
			name, opts := parts[0], parts[1:]
			src, color := false, ""
			for _, o := range opts {
				k, v, _ := strings.Cut(o, "=")
				switch strings.TrimSpace(k) {
				case "sources":
					src = true
				case "color", "colour":
					color = strings.TrimSpace(v)
					if !colourRE.MatchString(color) {
						p := at(i)
						p.Msg = fmt.Sprintf("era %q: color=%q is not a CSS colour", name, color)
						probs = append(probs, p)
						color = ""
					}
				default:
					p := at(i)
					p.Msg = fmt.Sprintf("era %q: unknown option %q (use color=#hex, or sources for the sources block)", name, o)
					probs = append(probs, p)
				}
			}
			inSources = src
			if src {
				t.SourcesTitle = name
				continue
			}
			era = &TimelineEra{Name: name, Color: color, Line: nos[i]}
			t.Eras = append(t.Eras, era)
		case tlMilestoneRE.MatchString(line) && !inSources:
			flush()
			parts := splitDot(tlMilestoneRE.FindStringSubmatch(line)[1])
			ms = &Milestone{Date: parts[0], Line: nos[i]}
			if len(parts) > 1 {
				ms.Title = strings.Join(parts[1:], " · ")
			} else {
				p := at(i)
				p.Msg = fmt.Sprintf("milestone %q has no title: write ### date · title", parts[0])
				probs = append(probs, p)
			}
			ms.Year, ms.Dated = ParseYear(ms.Date)
			if !ms.Dated {
				p := at(i)
				p.Msg = fmt.Sprintf("milestone %q: cannot read a year in the date (it stays as written, but si cannot check the order)", ms.Date)
				warns = append(warns, p)
			}
			if era == nil {
				era = &TimelineEra{Line: nos[i]}
				t.Eras = append(t.Eras, era)
			}
			era.Items = append(era.Items, ms)
		case inSources:
			if m := tlItemRE.FindStringSubmatch(line); m != nil {
				t.Sources = append(t.Sources, strings.TrimSpace(m[1]))
			} else if trim != "" && len(t.Sources) > 0 { // a wrapped item
				t.Sources[len(t.Sources)-1] += " " + trim
			} else if trim != "" {
				t.Sources = append(t.Sources, trim)
			}
		case ms != nil:
			if m := tlImageRE.FindStringSubmatch(line); m != nil {
				if ms.Img != "" {
					p := at(i)
					p.Msg = fmt.Sprintf("milestone %q has a second picture; one per milestone is shown", ms.Title)
					warns = append(warns, p)
					continue
				}
				ms.Caption, ms.Img, ms.Credit = strings.TrimSpace(m[1]), m[2], strings.TrimSpace(m[3])
				continue
			}
			if trim != "" {
				desc = append(desc, trim)
			}
		case trim != "" && !strings.HasPrefix(trim, "<!--"):
			p := at(i)
			p.Msg = "text outside a milestone is not shown: put it under a ### date · title"
			warns = append(warns, p)
		}
	}
	flush()
	return t, probs, warns
}

// Milestones is every milestone, in the order written.
func (t *Timeline) Milestones() []*Milestone {
	var out []*Milestone
	for _, e := range t.Eras {
		out = append(out, e.Items...)
	}
	return out
}

// Pictures counts the milestones with a picture.
func (t *Timeline) Pictures() int {
	n := 0
	for _, m := range t.Milestones() {
		if m.Img != "" {
			n++
		}
	}
	return n
}

// Summary is the counts si check prints.
func (t *Timeline) Summary() string {
	eras := 0
	for _, e := range t.Eras {
		if e.Name != "" {
			eras++
		}
	}
	return fmt.Sprintf("%d milestones, %d pictures, %d eras, %d sources", len(t.Milestones()), t.Pictures(), eras, len(t.Sources))
}

// TimelineDescMax is the description length (characters) above which si
// check warns: 1–2 lines read best on a one-page timeline.
const TimelineDescMax = 220

// Check returns the advice about a parsed timeline: order, description
// length, missing pictures (dir resolves relative picture paths).
func (t *Timeline) Check(file, dir string) (probs, warns []Problem) {
	prev := math.Inf(-1)
	prevDate := ""
	for _, m := range t.Milestones() {
		p := Problem{File: rel(file), Line: m.Line}
		if m.Dated && m.Year < prev {
			p.Msg = fmt.Sprintf("milestone %q (%s) comes after %s: out of chronological order", m.Title, m.Date, prevDate)
			warns = append(warns, p)
		}
		if m.Dated {
			prev, prevDate = m.Year, m.Date
		}
		if n := len([]rune(m.Text)); n > TimelineDescMax {
			p.Msg = fmt.Sprintf("milestone %q: the description is %d characters; 1–2 lines (≤ %d) read best", m.Title, n, TimelineDescMax)
			warns = append(warns, p)
		}
		if m.Text == "" {
			p.Msg = fmt.Sprintf("milestone %q has no description", m.Title)
			warns = append(warns, p)
		}
		if m.Img != "" && !strings.Contains(m.Img, "://") && !strings.HasPrefix(m.Img, "data:") {
			if !exists(filepath.Join(dir, filepath.FromSlash(m.Img))) {
				p.Msg = fmt.Sprintf("milestone %q: picture %s not found", m.Title, m.Img)
				probs = append(probs, p)
			}
		}
	}
	return probs, warns
}

// Require is a rubric for si check: at least this many of each.
type Require struct{ Dates, Images, Eras, Sources int }

var requireRE = regexp.MustCompile(`^(\d+)\s+(\S+)$`)

// ParseRequire reads `require: 20 dates, 15 images, 7 eras, 3 sources`.
func ParseRequire(s string) (Require, error) {
	var r Require
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m := requireRE.FindStringSubmatch(part)
		if m == nil {
			return r, fmt.Errorf("require: want a count and a thing, like 20 dates, 15 images, not %q", part)
		}
		n, _ := strconv.Atoi(m[1])
		switch strings.ToLower(m[2]) {
		case "dates", "date", "milestones", "milestone", "fechas", "hitos":
			r.Dates = n
		case "images", "image", "pictures", "picture", "imágenes", "imagenes":
			r.Images = n
		case "eras", "era", "épocas", "epocas":
			r.Eras = n
		case "sources", "source", "fuentes":
			r.Sources = n
		default:
			return r, fmt.Errorf("require: %q is not dates, images, eras or sources", m[2])
		}
	}
	return r, nil
}

// Unmet lists what the timeline lacks against the rubric.
func (t *Timeline) Unmet(r Require) []string {
	var out []string
	eras := 0
	for _, e := range t.Eras {
		if e.Name != "" {
			eras++
		}
	}
	for _, c := range []struct {
		have, want int
		what       string
	}{{len(t.Milestones()), r.Dates, "dates"}, {t.Pictures(), r.Images, "images"}, {eras, r.Eras, "eras"}, {len(t.Sources), r.Sources, "sources"}} {
		if c.have < c.want {
			out = append(out, fmt.Sprintf("require: %d %s, the timeline has %d", c.want, c.what, c.have))
		}
	}
	return out
}

var (
	yearRE  = regexp.MustCompile(`(\d{1,4})(?:\s*[-/]\s*(\d{1,2}))?`)
	bcRE    = regexp.MustCompile(`(?i)(a\.\s*c\.|a\.\s*e\.\s*c\.|\bbce?\b|\bb\.c\.(e\.)?|\bav\.\s*j\.?-?\s*c\.?)`)
	rangeRE = regexp.MustCompile(`^(\d{1,4})\s*[–—-]\s*\d{3,4}$`)
)

// ParseYear reads a written date for ordering: "c. 300 a.C." is -300,
// "1936-05" is 1936.33, "2022–2024" is 2022 (its start), "240 BC" -240.
func ParseYear(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	bc := bcRE.MatchString(t)
	t = bcRE.ReplaceAllString(t, "")
	for _, p := range []string{"circa", "ca.", "c.", "~", "hacia", "approx."} {
		t = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(t), p))
	}
	if m := rangeRE.FindStringSubmatch(t); m != nil {
		y, _ := strconv.Atoi(m[1])
		return sign(float64(y), bc), true
	}
	m := yearRE.FindStringSubmatch(t)
	if m == nil {
		return 0, false
	}
	y, _ := strconv.Atoi(m[1])
	v := float64(y)
	if m[2] != "" {
		if mo, _ := strconv.Atoi(m[2]); mo >= 1 && mo <= 12 {
			v += float64(mo-1) / 12
		}
	}
	return sign(v, bc), true
}

func sign(v float64, bc bool) float64 {
	if bc {
		return -v
	}
	return v
}

// buildTimeline makes a timeline page one slide (landscape: a deck of one
// slide, presentable) or one document part (portrait), its body a timeline
// fence the renderer draws; Notes: and Points: work as on any slide.
func (d *Deck) buildTimeline(lines []srcLine) {
	lines = trimBlank(lines)
	if len(lines) == 0 {
		d.problem(d.Path, 0, "empty timeline: write ## Era and ### date · title lines")
		return
	}
	s := d.buildSlide(lines, false)
	if s.Title != "" { // the first ## era is not a slide title
		s.Body = "## " + s.Title + "\n" + s.Body
		s.Title = ""
	}
	var texts []string
	var nos []int
	var ft fenceTracker
	for _, l := range lines { // the timeline itself: up to Notes: / Points:
		if !ft.inFence(l.text) && (notesRE.MatchString(l.text) || pointsRE.MatchString(l.text)) {
			break
		}
		texts, nos = append(texts, l.text), append(nos, l.no)
	}
	t, probs, warns := parseTimeline(texts, nos, d.Path)
	cp, cw := t.Check(d.Path, filepath.Dir(d.Path))
	d.Problems = append(append(d.Problems, probs...), cp...)
	d.Warnings = append(append(d.Warnings, warns...), cw...)
	if r, err := ParseRequire(d.Meta.Timeline.Require); err == nil {
		for _, msg := range t.Unmet(r) {
			d.problem(d.Path, 0, "%s", msg)
		}
	}
	d.Timeline = t
	s.Body = "`````timeline si-page\n" + s.Body + "\n`````"
	if d.isPoster() {
		s.Kind = KindDocument
		s.Title = d.Title()
	} else {
		s.Classes = append(s.Classes, "si-tl-slide")
	}
	d.Slides = append(d.Slides, s)
}
