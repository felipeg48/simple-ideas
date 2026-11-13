package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Infographics (`type: infographic`, the ```infographic fence): a one-page
// visual made of blocks. Every `## heading` is a block; `· kind` and the
// bento options follow the heading:
//
//	## 2.4 s · stat · accent
//	## Share of traffic · chart=donut · wide
//
// A block without a kind gets one guessed from its text. The renderer
// (internal/render/infographic.go) draws each kind; this file only reads them.

// Infographic front matter and block values.
var (
	IGOrientations = []string{"landscape", "portrait"}
	IGAspects      = []string{"16:9", "4:3", "9:16", "a4"}
	IGLayouts      = []string{"grid", "columns", "flow"}
	IGKinds        = []string{"stat", "compare", "steps", "chart", "quote", "icons", "list", "text", "image", "sources"}
	IGCharts       = []string{"bar", "donut", "line", "progress"}
)

// IsInfographic reports whether the file is an infographic (`type: infographic`).
func (d *Deck) IsInfographic() bool { return d.Meta.Type == "infographic" }

// IGPortrait reports whether an infographic is a tall poster (a page) rather
// than a slide: orientation: portrait, or a tall aspect without an orientation.
func (m Meta) IGPortrait() bool {
	if m.Orientation != "" {
		return m.Orientation == "portrait"
	}
	return m.Aspect == "9:16" || m.Aspect == "a4"
}

// Infographic is the Markdown of an infographic: an optional `# Title` and
// subtitle paragraph (in a fence; a page has them in its front matter), then
// the blocks.
type Infographic struct {
	Title, Subtitle string
	Blocks          []*IGBlock
	Sources         *IGBlock // the `· sources` block: drawn as the footer
}

// IGBlock is one `##` block.
type IGBlock struct {
	Title   string // the heading without its options (a stat's number)
	Kind    string // one of IGKinds
	Chart   string // Kind chart: one of IGCharts
	Span    int    // grid columns it covers (wide: 2)
	Rows    int    // grid rows it covers (tall: 2)
	Accent  bool   // drawn on the accent colour
	Guessed bool   // Kind was guessed from the text
	Body    string // the Markdown under the heading
	Line    int    // 1-based line of the heading in the source
}

// IGProblem is a mistake in an infographic, at a 1-based line of its source.
type IGProblem struct {
	Line int
	Msg  string
	Warn bool // a warning, not an error
}

var igHeadRE = regexp.MustCompile(`^ {0,3}(#{1,2})\s+(.+?)\s*#*\s*$`)

// ParseInfographic reads an infographic's Markdown (a page body or a fence).
func ParseInfographic(src string) (*Infographic, []IGProblem) {
	ig := &Infographic{}
	var probs []IGProblem
	var b *IGBlock
	var body, sub []string
	var ft fenceTracker
	flush := func() {
		if b == nil {
			return
		}
		b.Body = strings.TrimSpace(strings.Join(body, "\n"))
		probs = append(probs, ig.add(b)...)
		body = nil
	}
	for i, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		if !ft.inFence(line) {
			if m := igHeadRE.FindStringSubmatch(line); m != nil {
				if m[1] == "#" && b == nil && ig.Title == "" {
					ig.Title = m[2]
					continue
				}
				if m[1] == "##" {
					flush()
					b = &IGBlock{Line: i + 1, Span: 1, Rows: 1}
					probs = append(probs, b.options(m[2])...)
					continue
				}
			}
		}
		if b == nil {
			if t := strings.TrimSpace(line); t != "" {
				sub = append(sub, t)
			}
			continue
		}
		body = append(body, line)
	}
	flush()
	ig.Subtitle = strings.Join(sub, " ")
	if len(ig.Blocks) == 0 {
		probs = append(probs, IGProblem{Line: 1, Msg: "infographic: no blocks (start each block with a `## heading`)"})
	}
	return ig, probs
}

// options splits `Title · kind · span=2` into the title and the options. A
// part is options when its first word is one; else it is part of the title.
func (b *IGBlock) options(head string) []IGProblem {
	parts := strings.Split(head, "·")
	title := []string{strings.TrimSpace(parts[0])}
	var probs []IGProblem
	for _, p := range parts[1:] {
		words := strings.Fields(p)
		if len(words) == 0 || !igOption(words[0]) {
			title = append(title, strings.TrimSpace(p))
			continue
		}
		for _, w := range words {
			if msg := b.option(w); msg != "" {
				probs = append(probs, IGProblem{Line: b.Line, Msg: msg})
			}
		}
	}
	b.Title = strings.Join(title, " · ")
	return probs
}

func igOption(w string) bool {
	k, _, _ := strings.Cut(w, "=")
	switch k {
	case "chart", "span", "wide", "tall", "accent", "kind":
		return true
	}
	return contains(IGKinds, k)
}

func (b *IGBlock) option(w string) string {
	k, v, _ := strings.Cut(w, "=")
	switch {
	case k == "chart":
		if v == "" {
			v = "bar"
		}
		b.Kind = "chart"
		b.Chart = v
		if !contains(IGCharts, v) {
			return fmt.Sprintf("infographic: unknown chart %q (use %s)", v, strings.Join(IGCharts, ", "))
		}
	case k == "kind":
		b.Kind = v
		if !contains(IGKinds, v) {
			return fmt.Sprintf("infographic: unknown block kind %q (use %s, chart=bar|donut|line|progress)", v, strings.Join(IGKinds, ", "))
		}
	case k == "span":
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 4 {
			return fmt.Sprintf("infographic: span=%s: want 1 to 4 columns", v)
		}
		b.Span = n
	case k == "wide":
		b.Span = 2
	case k == "tall":
		b.Rows = 2
	case k == "accent":
		b.Accent = true
	case contains(IGKinds, k) && v == "":
		b.Kind = k
		if k == "chart" {
			b.Chart = "bar"
		}
	default:
		return fmt.Sprintf("infographic: unknown block option %q (use a kind: %s; chart=, span=, wide, tall, accent)", w, strings.Join(IGKinds, ", "))
	}
	return ""
}

// add guesses the block's kind when it has none, checks it and keeps it.
func (ig *Infographic) add(b *IGBlock) []IGProblem {
	var probs []IGProblem
	if b.Kind == "" {
		b.Kind, b.Chart = GuessIGKind(b.Title, b.Body)
		b.Guessed = true
	}
	at := func(warn bool, f string, a ...any) {
		probs = append(probs, IGProblem{Line: b.Line, Warn: warn, Msg: fmt.Sprintf("infographic: block %q: ", b.Title) + fmt.Sprintf(f, a...)})
	}
	switch b.Kind {
	case "chart":
		head, rows, ok := IGTable(b.Body)
		if !ok {
			at(false, "a chart needs a Markdown table (labels in the first column, numbers after)")
			break
		}
		if len(head) < 2 {
			at(false, "a chart table needs a column of numbers after the labels")
		}
		for _, r := range rows {
			for j := 1; j < len(r) && j < len(head); j++ {
				if _, ok := IGNumber(r[j]); !ok {
					at(false, "%q is not a number (chart=%s draws the numbers after the first column)", r[j], b.Chart)
				} else if v, _ := IGNumber(r[j]); v < 0 && (b.Chart == "donut" || b.Chart == "progress") {
					at(false, "%q: a %s takes no negative numbers", r[j], b.Chart)
				}
			}
		}
	case "compare":
		if _, _, ok := IGTable(b.Body); !ok {
			at(false, "compare needs a Markdown table: one column per side (| Before | After |)")
		}
	case "stat":
		if b.Title == "" {
			at(false, "a stat is its heading: write the number there (## 2.4 s · stat)")
		}
	case "sources":
		if ig.Sources != nil {
			at(true, "a second sources block (the first is the footer; this one is drawn as text)")
			b.Kind = "text"
		} else {
			ig.Sources = b
			return probs
		}
	default:
		if b.Body == "" && b.Kind != "stat" {
			at(false, "empty block (write its text under the heading)")
		}
	}
	ig.Blocks = append(ig.Blocks, b)
	return probs
}

var (
	igStatRE   = regexp.MustCompile(`^[~≈<>+±$€£¥−-]?\s?\d`)
	igOrderRE  = regexp.MustCompile(`^\d+[.)]\s`)
	igBulletRE = regexp.MustCompile(`^[-*+]\s`)
	igImageRE  = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	igNumRE    = regexp.MustCompile(`[-+−]?(?:\d[\d,]*)?\.?\d+`)
)

// GuessIGKind names the kind of a block written without one: a heading that
// starts with a number is a stat, a table of numbers a bar chart (else a
// compare), a numbered list steps, a list with pictures icons, a quote a
// quote, a lone picture an image; anything else is text.
func GuessIGKind(title, body string) (kind, chart string) {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "sources", "source", "references", "fuentes", "referencias", "bibliography":
		return "sources", ""
	}
	if igStatRE.MatchString(strings.TrimSpace(title)) {
		return "stat", ""
	}
	if head, rows, ok := IGTable(body); ok {
		if len(rows) >= 2 && len(head) >= 2 && numericCols(rows, len(head)) {
			return "chart", "bar"
		}
		return "compare", ""
	}
	first := ""
	for _, l := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			first = t
			break
		}
	}
	switch {
	case igOrderRE.MatchString(first):
		return "steps", ""
	case igBulletRE.MatchString(first):
		if igImageRE.MatchString(body) {
			return "icons", ""
		}
		return "list", ""
	case strings.HasPrefix(first, ">"):
		return "quote", ""
	case igImageRE.MatchString(first) && strings.TrimSpace(igImageRE.ReplaceAllString(body, "")) == "":
		return "image", ""
	}
	return "text", ""
}

func numericCols(rows [][]string, n int) bool {
	for _, r := range rows {
		for j := 1; j < n && j < len(r); j++ {
			if _, ok := IGNumber(r[j]); !ok {
				return false
			}
		}
	}
	return true
}

// IGTable reads the first Markdown table of a block: its header cells and
// its rows (each as long as the header).
func IGTable(body string) (head []string, rows [][]string, ok bool) {
	lines := strings.Split(body, "\n")
	for i := 0; i+1 < len(lines); i++ {
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") || !tableSepRE.MatchString(strings.TrimSpace(lines[i+1])) {
			continue
		}
		head = tableCells(lines[i])
		for _, l := range lines[i+2:] {
			if !strings.HasPrefix(strings.TrimSpace(l), "|") {
				break
			}
			r := tableCells(l)
			for len(r) < len(head) {
				r = append(r, "")
			}
			rows = append(rows, r[:len(head)])
		}
		return head, rows, true
	}
	return nil, nil, false
}

var tableSepRE = regexp.MustCompile(`^\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?$`)

func tableCells(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	l = strings.ReplaceAll(l, `\|`, "\x00")
	var out []string
	for _, c := range strings.Split(l, "|") {
		out = append(out, strings.TrimSpace(strings.ReplaceAll(c, "\x00", "|")))
	}
	return out
}

// IGNumber reads the number of a cell or a stat as written: "54", "12.5%",
// "$1,200", "2.4 s", "−3". It refuses a cell with more than one number or
// with words around it ("n/a", "about 4 or 5").
func IGNumber(s string) (float64, bool) {
	s = strings.Trim(strings.TrimSpace(s), "*_`")
	ms := igNumRE.FindAllStringIndex(s, -1)
	if len(ms) != 1 {
		return 0, false
	}
	pre, suf := strings.TrimSpace(s[:ms[0][0]]), strings.TrimSpace(s[ms[0][1]:])
	if len([]rune(pre)) > 2 || len([]rune(suf)) > 6 || strings.ContainsAny(suf, " ") {
		return 0, false
	}
	n := strings.ReplaceAll(strings.ReplaceAll(s[ms[0][0]:ms[0][1]], ",", ""), "−", "-")
	v, err := strconv.ParseFloat(n, 64)
	return v, err == nil
}

// buildInfographic makes the file one slide: its blocks in an
// ```infographic page fence (so a landscape infographic is a slide of a deck,
// presented with si play), then its trailing Notes: and Points:.
func (d *Deck) buildInfographic(lines []srcLine) {
	lines = trimBlank(lines)
	if len(lines) == 0 {
		d.problem(d.Path, 0, "empty infographic: write its blocks, each under a `## heading`")
		return
	}
	cut := len(lines)
	var ft fenceTracker
	for i, l := range lines {
		if !ft.inFence(l.text) && (notesRE.MatchString(l.text) || pointsRE.MatchString(l.text)) {
			cut = i
			break
		}
	}
	first := lines[0]
	c := []srcLine{{first.file, first.no, "~~~~~~~~ infographic page"}}
	c = append(c, lines[:cut]...)
	c = append(c, srcLine{first.file, lines[max(cut-1, 0)].no, "~~~~~~~~"})
	c = append(c, lines[cut:]...)
	s := d.buildSlide(c, false)
	s.Classes = append(s.Classes, "si-ig-slide")
	d.Slides = []*Slide{s}
	d.checkSpeakers()
}

// igKey is the validateMeta key of layout: and aspect: for this file's type.
func (d *Deck) igKey(k string) string {
	if d.IsInfographic() {
		return "ig-" + k
	}
	return k
}
