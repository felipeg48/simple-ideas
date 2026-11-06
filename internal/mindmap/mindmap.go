// Package mindmap reads and writes the Markdown of mind maps for editors (the
// Studio's map editor, the terminal Studio) and converters (OPML, FreeMind,
// XMind, Mermaid). It keeps the text it does not model (front matter keys,
// the lines between maps, comments) as it was; the renderer has its own
// reader (internal/render/mindmap.go) for drawing.
//
// A topic line is `- [!sticky] text ![alt](img) {attrs} → target`; a fenced
// code block under a topic is a code topic; `# Title {#id}` starts a map.
package mindmap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Topic is one item of a map.
type Topic struct {
	Text      string      `json:"text"`
	Sticky    bool        `json:"sticky,omitempty"`
	Img       string      `json:"img,omitempty"`
	ImgAlt    string      `json:"imgAlt,omitempty"`
	Badge     string      `json:"badge,omitempty"`
	Collapsed bool        `json:"collapsed,omitempty"`
	Pos       *[2]float64 `json:"pos,omitempty"`
	BPos      *[2]float64 `json:"bpos,omitempty"`
	Attrs     string      `json:"attrs,omitempty"`  // {…} words the model does not know, as typed
	Target    string      `json:"target,omitempty"` // after →: #id, file.md, slides: deck.md#5, a URL
	Code      *Code       `json:"code,omitempty"`   // a code topic: a fenced block under its parent
	Kids      []*Topic    `json:"kids,omitempty"`
	Extra     []string    `json:"-"` // lines under the item the model does not read, kept verbatim
}

// Code is a fenced code block under a topic.
type Code struct {
	Lang string `json:"lang"`
	Text string `json:"text"`
}

// Map is one `# Title {#id}` and its list.
type Map struct {
	Title   string   `json:"title"`
	ID      string   `json:"id,omitempty"` // the explicit {#id}, if any
	Heading bool     `json:"heading"`      // false: the first map of a page, titled by its front matter
	Intro   []string `json:"-"`            // lines between the heading and the list
	Topics  []*Topic `json:"topics"`
}

// Part is a map or the text between maps (`---` separators, comments).
type Part struct {
	Map  *Map
	Text []string
}

// Doc is a whole mind map source.
type Doc struct {
	Front []string // front matter with both --- lines; nil without
	Parts []Part
}

// Maps lists the maps in order.
func (d *Doc) Maps() []*Map {
	var out []*Map
	for _, p := range d.Parts {
		if p.Map != nil {
			out = append(out, p.Map)
		}
	}
	return out
}

var (
	itemRE    = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	headRE    = regexp.MustCompile(`^#\s+(.+?)\s*$`)
	idRE      = regexp.MustCompile(`\s*\{#([A-Za-z][\w-]*)\}\s*$`)
	attrRE    = regexp.MustCompile(`\s*\{([^{}]*)\}\s*$`)
	imgRE     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	posRE     = regexp.MustCompile(`^(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)$`)
	fenceRE   = regexp.MustCompile("^(\\s*)(```+|~~~+)\\s*([\\w+-]*)")
	frontKey  = regexp.MustCompile(`^([\w-]+)\s*:`)
	blankLine = func(s string) bool { return strings.TrimSpace(s) == "" }
)

// Parse reads a mind map source (a page with front matter, or a fence body).
func Parse(src string) *Doc {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	d := &Doc{}
	i := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				d.Front, i = append([]string(nil), lines[:j+1]...), j+1
				break
			}
		}
	}
	var cur *Map
	var text []string
	flush := func() {
		if len(text) > 0 {
			d.Parts = append(d.Parts, Part{Text: text})
			text = nil
		}
	}
	for i < len(lines) {
		l := lines[i]
		if m := headRE.FindStringSubmatch(l); m != nil && !strings.HasPrefix(l, "##") {
			flush()
			t, id := m[1], ""
			if im := idRE.FindStringSubmatch(t); im != nil {
				id, t = im[1], strings.TrimSpace(idRE.ReplaceAllString(t, ""))
			}
			cur = &Map{Title: t, ID: id, Heading: true}
			d.Parts = append(d.Parts, Part{Map: cur})
			i++
			for i < len(lines) && !itemRE.MatchString(lines[i]) && !headRE.MatchString(lines[i]) && strings.TrimSpace(lines[i]) != "---" {
				if !blankLine(lines[i]) {
					cur.Intro = append(cur.Intro, lines[i])
				}
				i++
			}
			continue
		}
		if m := itemRE.FindStringSubmatch(l); m != nil && m[1] == "" {
			if cur == nil {
				flush()
				cur = &Map{}
				d.Parts = append(d.Parts, Part{Map: cur})
			}
			var ts []*Topic
			ts, i = parseItems(lines, i, 0)
			cur.Topics = append(cur.Topics, ts...)
			continue
		}
		text = append(text, l)
		i++
	}
	flush()
	return d
}

func indentOf(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// parseItems reads the items at indent from line i; it returns them and the
// first line after the list.
func parseItems(lines []string, i, indent int) ([]*Topic, int) {
	var out []*Topic
	for i < len(lines) {
		l := lines[i]
		if blankLine(l) {
			// a blank line inside a list: keep going if the list goes on
			j := i
			for j < len(lines) && blankLine(lines[j]) {
				j++
			}
			if j < len(lines) && indentOf(lines[j]) >= indent && (itemRE.MatchString(lines[j]) || indentOf(lines[j]) > indent) && !(indent == 0 && !itemRE.MatchString(lines[j])) {
				i = j
				continue
			}
			return out, i
		}
		m := itemRE.FindStringSubmatch(l)
		in := indentOf(l)
		if m == nil || in < indent {
			return out, i
		}
		if in > indent { // deeper than a sibling but not under an item: treat as one
			indent = in
		}
		t := &Topic{}
		label := m[3]
		content := in + len(m[2]) + 1
		i++
		// lazy continuation lines of the label
		for i < len(lines) && !blankLine(lines[i]) && !itemRE.MatchString(lines[i]) && !fenceRE.MatchString(lines[i]) && indentOf(lines[i]) >= content {
			label += " " + strings.TrimSpace(lines[i])
			i++
		}
		parseLabel(t, label)
		for i < len(lines) {
			l := lines[i]
			if blankLine(l) {
				j := i
				for j < len(lines) && blankLine(lines[j]) {
					j++
				}
				if j < len(lines) && indentOf(lines[j]) >= content {
					i = j
					continue
				}
				break
			}
			if indentOf(l) < content {
				break
			}
			if f := fenceRE.FindStringSubmatch(l); f != nil {
				code, next := readFence(lines, i, f)
				t.Kids = append(t.Kids, code)
				i = next
				continue
			}
			if itemRE.MatchString(l) {
				var kids []*Topic
				kids, i = parseItems(lines, i, indentOf(l))
				t.Kids = append(t.Kids, kids...)
				continue
			}
			t.Extra = append(t.Extra, strings.TrimPrefix(l, strings.Repeat(" ", content)))
			i++
		}
		out = append(out, t)
	}
	return out, i
}

func readFence(lines []string, i int, f []string) (*Topic, int) {
	ind, mark, lang := len(f[1]), f[2], f[3]
	var body []string
	i++
	for i < len(lines) {
		l := lines[i]
		if strings.HasPrefix(strings.TrimSpace(l), mark) && strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), mark[:1])) == "" {
			i++
			break
		}
		if indentOf(l) >= ind {
			l = l[min(len(l), byteIndent(l, ind)):]
		} else {
			l = strings.TrimLeft(l, " \t")
		}
		body = append(body, l)
		i++
	}
	name := lang
	if name == "" {
		name = "code"
	}
	return &Topic{Text: name + " code", Code: &Code{Lang: lang, Text: strings.Join(body, "\n")}}, i
}

// byteIndent is the byte offset of the n-th column of indentation.
func byteIndent(s string, n int) int {
	col := 0
	for b, r := range s {
		if col >= n {
			return b
		}
		if r == '\t' {
			col += 4
		} else if r == ' ' {
			col++
		} else {
			return b
		}
	}
	return len(s)
}

func parseLabel(t *Topic, s string) {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "[!sticky]"); ok {
		t.Sticky, s = true, strings.TrimSpace(rest)
	}
	if i := strings.LastIndex(s, "→"); i >= 0 {
		t.Target, s = strings.TrimSpace(s[i+len("→"):]), s[:i]
	} else if i := strings.LastIndex(s, " -> "); i >= 0 {
		t.Target, s = strings.TrimSpace(s[i+4:]), s[:i]
	}
	if m := attrRE.FindStringSubmatch(s); m != nil {
		s = attrRE.ReplaceAllString(s, "")
		var unknown []string
		for _, f := range strings.Fields(m[1]) {
			k, v, _ := strings.Cut(f, "=")
			switch k {
			case "collapsed":
				t.Collapsed = true
			case "badge":
				t.Badge = strings.ReplaceAll(v, "_", " ")
			case "pos", "branch-pos":
				if pm := posRE.FindStringSubmatch(v); pm != nil {
					x, _ := strconv.ParseFloat(pm[1], 64)
					y, _ := strconv.ParseFloat(pm[2], 64)
					if k == "pos" {
						t.Pos = &[2]float64{x, y}
					} else {
						t.BPos = &[2]float64{x, y}
					}
					continue
				}
				unknown = append(unknown, f)
			default:
				unknown = append(unknown, f)
			}
		}
		t.Attrs = strings.Join(unknown, " ")
	}
	if m := imgRE.FindStringSubmatch(s); m != nil {
		t.Img, t.ImgAlt = m[2], m[1]
		s = strings.Replace(s, m[0], "", 1)
	}
	t.Text = strings.Join(strings.Fields(s), " ")
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// Label is the topic's line without the list marker.
func (t *Topic) Label() string {
	var b strings.Builder
	if t.Sticky {
		b.WriteString("[!sticky] ")
	}
	b.WriteString(t.Text)
	if t.Img != "" {
		if t.Text != "" {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "![%s](%s)", t.ImgAlt, t.Img)
	}
	var at []string
	if t.Badge != "" {
		at = append(at, "badge="+strings.ReplaceAll(t.Badge, " ", "_"))
	}
	if t.Collapsed {
		at = append(at, "collapsed")
	}
	if t.Pos != nil {
		at = append(at, "pos="+num(t.Pos[0])+","+num(t.Pos[1]))
	}
	if t.BPos != nil {
		at = append(at, "branch-pos="+num(t.BPos[0])+","+num(t.BPos[1]))
	}
	if t.Attrs != "" {
		at = append(at, t.Attrs)
	}
	if len(at) > 0 {
		b.WriteString(" {" + strings.Join(at, " ") + "}")
	}
	if t.Target != "" {
		b.WriteString(" → " + t.Target)
	}
	return b.String()
}

// String writes the source back.
func (d *Doc) String() string {
	var out []string
	out = append(out, d.Front...)
	for _, p := range d.Parts {
		if p.Map == nil {
			out = append(out, p.Text...)
			continue
		}
		m := p.Map
		if m.Heading {
			h := "# " + m.Title
			if m.ID != "" {
				h += " {#" + m.ID + "}"
			}
			out = append(out, h, "")
			if len(m.Intro) > 0 {
				out = append(out, m.Intro...)
				out = append(out, "")
			}
		}
		writeTopics(&out, m.Topics, 0)
	}
	return strings.Join(out, "\n") + "\n"
}

func writeTopics(out *[]string, ts []*Topic, indent int) {
	pad := strings.Repeat(" ", indent)
	for _, t := range ts {
		if t.Code != nil {
			*out = append(*out, pad+"```"+t.Code.Lang)
			for _, l := range strings.Split(t.Code.Text, "\n") {
				if l == "" {
					*out = append(*out, "")
				} else {
					*out = append(*out, pad+l)
				}
			}
			*out = append(*out, pad+"```")
			continue
		}
		*out = append(*out, pad+"- "+t.Label())
		for _, l := range t.Extra {
			*out = append(*out, pad+"  "+l)
		}
		writeTopics(out, t.Kids, indent+2)
	}
}

// FrontValue is a front matter key's value ("" if absent).
func (d *Doc) FrontValue(key string) string {
	for _, l := range d.Front {
		if m := frontKey.FindStringSubmatch(l); m != nil && m[1] == key {
			_, v, _ := strings.Cut(l, ":")
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

// SetFront sets (or with "" removes) a front matter key.
func (d *Doc) SetFront(key, val string) {
	if d.Front == nil {
		if val == "" {
			return
		}
		d.Front = []string{"---", "---"}
	}
	line := key + ": " + val
	for i, l := range d.Front {
		if m := frontKey.FindStringSubmatch(l); m != nil && m[1] == key && i > 0 && i < len(d.Front)-1 {
			if val == "" {
				d.Front = append(d.Front[:i:i], d.Front[i+1:]...)
			} else {
				d.Front[i] = line
			}
			return
		}
	}
	if val == "" {
		return
	}
	n := len(d.Front) - 1
	d.Front = append(d.Front[:n:n], line, d.Front[n])
}

// FrontMap lists the front matter as key → value.
func (d *Doc) FrontMap() map[string]string {
	out := map[string]string{}
	for i, l := range d.Front {
		if i == 0 || i == len(d.Front)-1 {
			continue
		}
		if m := frontKey.FindStringSubmatch(l); m != nil {
			_, v, _ := strings.Cut(l, ":")
			out[m[1]] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return out
}
