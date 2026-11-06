package mindmap

import (
	"errors"
	"math"
	"regexp"
	"strings"
)

// Op is one edit from a map editor. Map is the index of the map in the
// source, Path the topic (child indexes from the central topic; empty: the
// central topic itself).
type Op struct {
	Kind   string  `json:"kind"` // add-child, add-sibling, delete, rename, set, move, up, down, pos, tidy, meta, add-map
	Map    int     `json:"map"`
	Path   []int   `json:"path"`
	Text   string  `json:"text,omitempty"`
	Field  string  `json:"field,omitempty"` // set: text, badge, collapsed, sticky, target, img, imgAlt, lang, code; meta: the key
	Value  string  `json:"value,omitempty"`
	To     []int   `json:"to,omitempty"`    // move: the new parent
	Index  int     `json:"index,omitempty"` // move: position under the new parent (-1: last)
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Branch bool    `json:"branch,omitempty"` // pos: move the branch (branch-pos=)
}

// Sel is the topic to select after an edit.
type Sel struct {
	Map  int   `json:"map"`
	Path []int `json:"path"`
}

var errPath = errors.New("no such topic")

func (d *Doc) mapAt(i int) (*Map, error) {
	ms := d.Maps()
	if i < 0 || i >= len(ms) {
		return nil, errors.New("no such map")
	}
	return ms[i], nil
}

// list returns the list holding the topic at path and its index there.
func list(m *Map, path []int) (*[]*Topic, int, error) {
	if len(path) == 0 {
		return nil, -1, errPath
	}
	l := &m.Topics
	for _, i := range path[:len(path)-1] {
		if i < 0 || i >= len(*l) {
			return nil, 0, errPath
		}
		l = &(*l)[i].Kids
	}
	i := path[len(path)-1]
	if i < 0 || i >= len(*l) {
		return nil, 0, errPath
	}
	return l, i, nil
}

func kids(m *Map, path []int) (*[]*Topic, error) {
	if len(path) == 0 {
		return &m.Topics, nil
	}
	l, i, err := list(m, path)
	if err != nil {
		return nil, err
	}
	return &(*l)[i].Kids, nil
}

func clone(p []int, more ...int) []int { return append(append([]int{}, p...), more...) }

func insert(l *[]*Topic, at int, t *Topic) int {
	if at < 0 || at > len(*l) {
		at = len(*l)
	}
	*l = append(*l, nil)
	copy((*l)[at+1:], (*l)[at:])
	(*l)[at] = t
	return at
}

// Apply makes one edit and says what to select next.
func (d *Doc) Apply(op Op) (Sel, error) {
	sel := Sel{Map: op.Map, Path: op.Path}
	if op.Kind == "meta" {
		d.SetFront(op.Field, op.Value)
		return sel, nil
	}
	if op.Kind == "add-map" {
		title := strings.TrimSpace(op.Text)
		if title == "" {
			title = "New map"
		}
		d.Parts = append(d.Parts, Part{Text: []string{"", "---", ""}}, Part{Map: &Map{Title: title, ID: slug(title), Heading: true, Topics: []*Topic{{Text: "First idea"}}}})
		return Sel{Map: len(d.Maps()) - 1}, nil
	}
	m, err := d.mapAt(op.Map)
	if err != nil {
		return sel, err
	}
	switch op.Kind {
	case "add-child":
		l, err := kids(m, op.Path)
		if err != nil {
			return sel, err
		}
		if len(op.Path) > 0 {
			if l2, i, _ := list(m, op.Path); l2 != nil {
				(*l2)[i].Collapsed = false
			}
		}
		at := insert(l, -1, &Topic{Text: orNew(op.Text)})
		sel.Path = clone(op.Path, at)
	case "add-sibling":
		if len(op.Path) == 0 {
			return d.Apply(Op{Kind: "add-child", Map: op.Map, Text: op.Text})
		}
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		at := insert(l, i+1, &Topic{Text: orNew(op.Text)})
		sel.Path = clone(op.Path[:len(op.Path)-1], at)
	case "delete":
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		*l = append((*l)[:i], (*l)[i+1:]...)
		switch {
		case i < len(*l):
			sel.Path = clone(op.Path)
		case i > 0:
			sel.Path = clone(op.Path[:len(op.Path)-1], i-1)
		default:
			sel.Path = clone(op.Path[:len(op.Path)-1])
		}
	case "rename":
		if len(op.Path) == 0 {
			if !m.Heading && d.Front != nil && op.Map == 0 {
				d.SetFront("title", strings.TrimSpace(op.Text))
			} else {
				m.Title, m.Heading = strings.TrimSpace(op.Text), true
			}
			return sel, nil
		}
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		(*l)[i].Text = strings.Join(strings.Fields(op.Text), " ")
	case "set":
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		setField((*l)[i], op.Field, op.Value)
	case "up", "down":
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		j := i - 1
		if op.Kind == "down" {
			j = i + 1
		}
		if j < 0 || j >= len(*l) {
			return sel, nil
		}
		(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
		sel.Path = clone(op.Path[:len(op.Path)-1], j)
	case "move":
		if len(op.To) >= len(op.Path) && len(op.Path) > 0 && equal(op.To[:len(op.Path)], op.Path) {
			return sel, errors.New("a topic cannot move into its own branch")
		}
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		t := (*l)[i]
		dest, err := kids(m, op.To)
		if err != nil {
			return sel, err
		}
		// removing t first shifts later siblings (and later branches of the same list) up by one
		to := clone(op.To)
		parent := op.Path[:len(op.Path)-1]
		if len(to) > len(parent) && equal(to[:len(parent)], parent) && to[len(parent)] > i {
			to[len(parent)]--
		}
		index := op.Index
		if equal(to, parent) && index > i {
			index--
		}
		*l = append((*l)[:i], (*l)[i+1:]...)
		dest, err = kids(m, to)
		if err != nil {
			return sel, err
		}
		at := insert(dest, index, t)
		sel.Path = clone(to, at)
	case "pos":
		l, i, err := list(m, op.Path)
		if err != nil {
			return sel, err
		}
		p := &[2]float64{math.Round(op.X), math.Round(op.Y)}
		if p[0] == 0 && p[1] == 0 {
			p = nil
		}
		if op.Branch {
			(*l)[i].BPos = p
		} else {
			(*l)[i].Pos = p
		}
	case "tidy":
		var clear func([]*Topic)
		clear = func(ts []*Topic) {
			for _, t := range ts {
				t.Pos, t.BPos = nil, nil
				clear(t.Kids)
			}
		}
		clear(m.Topics)
	default:
		return sel, errors.New("unknown edit " + op.Kind)
	}
	return sel, nil
}

func orNew(s string) string {
	if s = strings.TrimSpace(s); s == "" {
		return "New topic"
	}
	return s
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func setField(t *Topic, f, v string) {
	switch f {
	case "text":
		t.Text = strings.Join(strings.Fields(v), " ")
	case "badge":
		t.Badge = strings.TrimSpace(v)
	case "collapsed":
		t.Collapsed = v == "true" || v == "on"
	case "sticky":
		t.Sticky = v == "true" || v == "on"
	case "target":
		t.Target = strings.TrimSpace(v)
	case "img":
		t.Img = strings.TrimSpace(v)
	case "imgAlt":
		t.ImgAlt = strings.TrimSpace(v)
	case "lang", "code":
		if t.Code == nil {
			t.Code = &Code{}
		}
		if f == "lang" {
			t.Code.Lang = strings.TrimSpace(v)
			t.Text = orNew(t.Code.Lang) + " code"
		} else {
			t.Code.Text = strings.TrimRight(v, "\n")
		}
	}
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		s = "map-" + s
	}
	return strings.TrimRight(s, "-")
}

// Fence is a ```mindmap fence inside a larger source.
type Fence struct {
	Start, End int    // lines (0-based) of the opening and closing fence
	Info       string // the opening line
	Body       string
}

var mmFenceRE = regexp.MustCompile("^(\\s*)(```+|~~~+)\\s*mindmap\\b(.*)$")

// FenceAt finds the ```mindmap fence that holds line (1-based); ok is false
// when the line is outside every mind map fence.
func FenceAt(src string, line int) (Fence, bool) {
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		m := mmFenceRE.FindStringSubmatch(lines[i])
		if m == nil {
			if f := fenceRE.FindStringSubmatch(lines[i]); f != nil { // skip other fences whole
				for j := i + 1; j < len(lines); j++ {
					if strings.HasPrefix(strings.TrimSpace(lines[j]), f[2]) {
						i = j
						break
					}
				}
			}
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == m[2] {
				end = j
				break
			}
		}
		if line-1 >= i && line-1 <= end {
			return Fence{Start: i, End: end, Info: lines[i], Body: strings.Join(lines[i+1:min(end, len(lines))], "\n")}, true
		}
		i = end
	}
	return Fence{}, false
}

// Replace puts a new body (and opening line) into src in place of the fence.
func (f Fence) Replace(src, info, body string) string {
	lines := strings.Split(src, "\n")
	end := min(f.End, len(lines))
	out := append([]string{}, lines[:f.Start]...)
	out = append(out, info)
	out = append(out, strings.Split(strings.TrimRight(body, "\n"), "\n")...)
	if f.End < len(lines) {
		out = append(out, lines[end:]...)
	} else {
		out = append(out, "```")
	}
	return strings.Join(out, "\n")
}

var infoAttrRE = regexp.MustCompile(`\s([\w-]+)(?:="[^"]*"|=\S+)?`)

// SetInfo sets (or with "" removes) an option on a fence's opening line.
func SetInfo(info, key, val string) string {
	found := false
	out := infoAttrRE.ReplaceAllStringFunc(info, func(s string) string {
		m := infoAttrRE.FindStringSubmatch(s)
		if m[1] != key {
			return s
		}
		found = true
		if val == "" {
			return ""
		}
		return " " + key + "=" + val
	})
	if !found && val != "" {
		out = strings.TrimRight(out, " ") + " " + key + "=" + val
	}
	return out
}

// InfoMap lists a fence line's options.
func InfoMap(info string) map[string]string {
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`\s([\w-]+)(?:="([^"]*)"|=(\S+))?`).FindAllStringSubmatch(info, -1) {
		if m[1] == "mindmap" {
			continue
		}
		out[m[1]] = m[2] + m[3]
	}
	return out
}
