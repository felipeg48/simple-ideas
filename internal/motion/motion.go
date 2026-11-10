// Package motion is the frame model of an animated SVG canvas, for the
// Studio's Motion view: frame 1, the frames after it, and how each part looks
// in each frame (moved, turned, scaled, recoloured, hidden), written as a
// normal si canvas: parts carry data-part="name", and one marked block in the
// SVG's <style> holds a rule per frame and part. Read gives the model back
// from that block; Apply changes the SVG by one op and rewrites only the
// block and the elements the op names, so hand-written SVG and CSS stay as
// they are. Coordinates are the SVG's own viewBox units.
package motion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Look is how one part looks in one frame, relative to how it is drawn.
type Look struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Scale  float64 `json:"scale"`  // 1 = as drawn
	Rotate float64 `json:"rotate"` // degrees
	Fill   string  `json:"fill"`   // "" = as in frame 1
	Hidden bool    `json:"hidden"`
}

func (l Look) moved() bool {
	return l.X != 0 || l.Y != 0 || (l.Scale != 1 && l.Scale != 0) || l.Rotate != 0
}

func (l Look) zero() bool { return !l.moved() && l.Fill == "" && !l.Hidden }

// Frame is one picture of the animation: its name (the state word), how long
// the move into it takes, its easing, and the looks of the parts it changes.
type Frame struct {
	Name  string          `json:"name"`
	Dur   float64         `json:"dur"`  // seconds; 0 = the deck's canvas length
	Ease  string          `json:"ease"` // "" = the default
	Parts map[string]Look `json:"parts"`
}

// Part is a shape in the SVG: its name (data-part, "" when it has none), its
// tag and its place among the shapes (the index ops name an unnamed one by).
type Part struct {
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	Index int    `json:"index"`
	Label string `json:"label"` // its text, for a <text>
}

// Model is a canvas as frames.
type Model struct {
	ViewBox string  `json:"viewBox"`
	Parts   []Part  `json:"parts"`
	Frames  []Frame `json:"frames"`
}

// Eases are the easings the view offers, by name.
var Eases = map[string]string{
	"ease":        "ease",
	"linear":      "linear",
	"ease-in-out": "ease-in-out",
	"spring":      "cubic-bezier(.3,1.5,.5,1)",
	"snap":        "cubic-bezier(.2,.9,.1,1)",
}

const (
	begin = "/* si motion:"
	end   = "/* si motion end */"
)

var (
	shapeRE   = regexp.MustCompile(`<(rect|circle|ellipse|line|polyline|polygon|path|text|g|image|use)\b[^>]*>`)
	partRE    = regexp.MustCompile(`\sdata-part="([^"]*)"`)
	viewBoxRE = regexp.MustCompile(`<svg\b[^>]*\bviewBox="([^"]*)"`)
	rootRE    = regexp.MustCompile(`(?s)<svg\b[^>]*>`)
	styleRE   = regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
	nameRE    = regexp.MustCompile(`^[A-Za-z][\w-]*$`)
	timingRE  = regexp.MustCompile(`^\[data-state~="([\w-]+)"\] \*\{transition-duration:([\d.]+)s(?:;transition-timing-function:([^}]+))?\}$`)
	lookRE    = regexp.MustCompile(`^\[data-state~="([\w-]+)"\] \[data-part="([\w-]+)"\]\{(.*)\}$`)
	transRE   = regexp.MustCompile(`translate\(([-\d.e]+)px,\s*([-\d.e]+)px\)|rotate\(([-\d.e]+)deg\)|scale\(([-\d.e]+)\)`)
)

// Blank is a new, empty board: 16:9, the theme's tokens with fallbacks so the
// file looks right on its own too.
const Blank = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 450" class="motion" aria-label="An animation">
<style>
.motion{font-family:var(--si-font-body,system-ui,sans-serif)}
` + begin + ` frame1 */
` + end + `
</style>
</svg>
`

// Read reads the frame model of an SVG. An SVG without a motion block has
// one frame, frame1, with every part as drawn.
func Read(svg string) Model {
	m := Model{}
	if v := viewBoxRE.FindStringSubmatch(svg); v != nil {
		m.ViewBox = v[1]
	}
	m.Parts = parts(svg)
	block, _, _ := findBlock(svg)
	names := []string{"frame1"}
	if block != "" {
		first, _, _ := strings.Cut(block, "\n")
		if f := strings.Fields(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(first, begin)), "*/")); len(f) > 0 {
			names = f
		}
	}
	idx := map[string]int{}
	for i, n := range names {
		m.Frames = append(m.Frames, Frame{Name: n, Parts: map[string]Look{}})
		idx[n] = i
	}
	for _, ln := range strings.Split(block, "\n") {
		ln = strings.TrimSpace(ln)
		if t := timingRE.FindStringSubmatch(ln); t != nil {
			if i, ok := idx[t[1]]; ok {
				m.Frames[i].Dur, _ = strconv.ParseFloat(t[2], 64)
				m.Frames[i].Ease = easeName(t[3])
			}
			continue
		}
		l := lookRE.FindStringSubmatch(ln)
		if l == nil {
			continue
		}
		i, ok := idx[l[1]]
		if !ok {
			continue
		}
		look := Look{Scale: 1}
		for _, decl := range strings.Split(l[3], ";") {
			k, v, _ := strings.Cut(decl, ":")
			switch strings.TrimSpace(k) {
			case "transform":
				for _, t := range transRE.FindAllStringSubmatch(v, -1) {
					switch {
					case t[1] != "":
						look.X, _ = strconv.ParseFloat(t[1], 64)
						look.Y, _ = strconv.ParseFloat(t[2], 64)
					case t[3] != "":
						look.Rotate, _ = strconv.ParseFloat(t[3], 64)
					case t[4] != "":
						look.Scale, _ = strconv.ParseFloat(t[4], 64)
					}
				}
			case "fill":
				look.Fill = strings.TrimSpace(v)
			case "opacity":
				look.Hidden = strings.TrimSpace(v) == "0"
			}
		}
		m.Frames[i].Parts[l[2]] = look
	}
	return m
}

func easeName(css string) string {
	css = strings.TrimSpace(css)
	for k, v := range Eases {
		if v == css {
			return k
		}
	}
	return ""
}

func parts(svg string) []Part {
	var out []Part
	for i, loc := range shapeRE.FindAllStringSubmatchIndex(svg, -1) {
		tag := svg[loc[0]:loc[1]]
		p := Part{Tag: svg[loc[2]:loc[3]], Index: i}
		if n := partRE.FindStringSubmatch(tag); n != nil {
			p.Name = n[1]
		}
		if p.Tag == "text" {
			rest := svg[loc[1]:]
			if e := strings.Index(rest, "</text>"); e >= 0 && !strings.Contains(rest[:e], "<") {
				p.Label = strings.TrimSpace(rest[:e])
			}
		}
		out = append(out, p)
	}
	return out
}

// findBlock returns the motion block (from its first marker to the end
// marker, both included) and where it is; "" when there is none.
func findBlock(svg string) (string, int, int) {
	a := strings.Index(svg, begin)
	if a < 0 {
		return "", -1, -1
	}
	b := strings.Index(svg[a:], end)
	if b < 0 {
		return "", -1, -1
	}
	return svg[a : a+b+len(end)], a, a + b + len(end)
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// css writes the motion block. Frame 1 is the base state and stays on (the
// steps add frame N and take frame N-1 off), so a later frame writes every
// part it or frame 1 changes; what it leaves unset comes from frame 1.
func (m Model) css() string {
	var b strings.Builder
	names := make([]string, len(m.Frames))
	for i, f := range m.Frames {
		names[i] = f.Name
	}
	fmt.Fprintf(&b, "%s %s */\n", begin, strings.Join(names, " "))
	used := false
	for _, f := range m.Frames {
		for _, l := range f.Parts {
			used = used || !l.zero()
		}
	}
	if used {
		b.WriteString("[data-part]{transform-box:fill-box;transform-origin:center}\n")
	}
	var base map[string]Look
	if len(m.Frames) > 0 {
		base = m.Frames[0].Parts
	}
	for i, f := range m.Frames {
		if f.Dur > 0 {
			fmt.Fprintf(&b, `[data-state~="%s"] *{transition-duration:%ss`, f.Name, num(f.Dur))
			if e := Eases[f.Ease]; e != "" {
				b.WriteString(";transition-timing-function:" + e)
			}
			b.WriteString("}\n")
		}
		for _, p := range m.Parts {
			if p.Name == "" {
				continue
			}
			l, l0 := f.Parts[p.Name], base[p.Name]
			if l.zero() && (i == 0 || l0.zero()) {
				continue
			}
			var d []string
			if l.moved() {
				t := []string{}
				if l.X != 0 || l.Y != 0 {
					t = append(t, fmt.Sprintf("translate(%spx,%spx)", num(l.X), num(l.Y)))
				}
				if l.Rotate != 0 {
					t = append(t, "rotate("+num(l.Rotate)+"deg)")
				}
				if l.Scale != 1 && l.Scale != 0 {
					t = append(t, "scale("+num(l.Scale)+")")
				}
				d = append(d, "transform:"+strings.Join(t, " "))
			} else if i > 0 && l0.moved() {
				d = append(d, "transform:none")
			}
			if l.Fill != "" {
				d = append(d, "fill:"+l.Fill)
			}
			if l.Hidden {
				d = append(d, "opacity:0")
			} else if i > 0 && l0.Hidden {
				d = append(d, "opacity:1")
			}
			if len(d) > 0 {
				fmt.Fprintf(&b, `[data-state~="%s"] [data-part="%s"]{%s}`+"\n", f.Name, p.Name, strings.Join(d, ";"))
			}
		}
	}
	b.WriteString(end)
	return b.String()
}

// write puts m's block into svg: in place of the old one, else at the end of
// the first <style>, else in a new <style> after the root tag.
func write(svg string, m Model) string {
	block := m.css()
	if _, a, z := findBlock(svg); a >= 0 {
		return svg[:a] + block + svg[z:]
	}
	if loc := styleRE.FindStringSubmatchIndex(svg); loc != nil {
		at := loc[3]
		sep := "\n"
		if strings.HasSuffix(svg[:at], "\n") {
			sep = ""
		}
		return svg[:at] + sep + block + "\n" + svg[at:]
	}
	if loc := rootRE.FindStringIndex(svg); loc != nil {
		return svg[:loc[1]] + "\n<style>\n" + block + "\n</style>" + svg[loc[1]:]
	}
	return svg
}

// Op is one change from the Motion view.
type Op struct {
	Kind  string  `json:"kind"`  // blank, add-frame, del-frame, look, timing, name, add-shape, del-shape
	Frame int     `json:"frame"` // 0-based
	Part  string  `json:"part"`
	Index int     `json:"index"` // a shape's place (name, del-shape)
	Look  *Look   `json:"look"`
	Dur   float64 `json:"dur"`
	Ease  string  `json:"ease"`
	Name  string  `json:"name"`
	Shape string  `json:"shape"` // rect, circle, text, line
}

// Apply changes svg by one op and returns the new SVG.
func Apply(svg string, op Op) (string, error) {
	if op.Kind == "blank" {
		if strings.TrimSpace(svg) != "" {
			return "", fmt.Errorf("the file is not empty")
		}
		return Blank, nil
	}
	if rootRE.FindStringIndex(svg) == nil {
		return "", fmt.Errorf("not an SVG: start a blank board, or open an .svg file")
	}
	m := Read(svg)
	frame := func() (*Frame, error) {
		if op.Frame < 0 || op.Frame >= len(m.Frames) {
			return nil, fmt.Errorf("no frame %d", op.Frame+1)
		}
		return &m.Frames[op.Frame], nil
	}
	switch op.Kind {
	case "add-frame": // a copy of the frame, after it
		f, err := frame()
		if err != nil {
			return "", err
		}
		c := Frame{Dur: f.Dur, Ease: f.Ease, Parts: map[string]Look{}}
		for k, v := range f.Parts {
			c.Parts[k] = v
		}
		if op.Frame == 0 { // frame 1 is the base: a copy of it starts from its looks
			c.Dur, c.Ease = 0, ""
		}
		m.Frames = append(m.Frames[:op.Frame+1], append([]Frame{c}, m.Frames[op.Frame+1:]...)...)
		renumber(&m)
	case "del-frame":
		if _, err := frame(); err != nil {
			return "", err
		}
		if len(m.Frames) == 1 {
			return "", fmt.Errorf("an animation keeps at least one frame")
		}
		m.Frames = append(m.Frames[:op.Frame], m.Frames[op.Frame+1:]...)
		renumber(&m)
	case "look":
		f, err := frame()
		if err != nil {
			return "", err
		}
		if op.Look == nil || !named(m, op.Part) {
			return "", fmt.Errorf("no part %q", op.Part)
		}
		l := *op.Look
		if l.Scale == 0 {
			l.Scale = 1
		}
		if l.Fill != "" && strings.ContainsAny(l.Fill, ";{}<>\"") {
			return "", fmt.Errorf("not a colour: %s", l.Fill)
		}
		f.Parts[op.Part] = l
	case "timing":
		f, err := frame()
		if err != nil {
			return "", err
		}
		if op.Dur < 0 || op.Dur > 30 {
			return "", fmt.Errorf("a frame's move takes 0 to 30 seconds")
		}
		if _, ok := Eases[op.Ease]; op.Ease != "" && !ok {
			return "", fmt.Errorf("unknown easing %q", op.Ease)
		}
		f.Dur, f.Ease = op.Dur, op.Ease
	case "name":
		if !nameRE.MatchString(op.Name) {
			return "", fmt.Errorf("a part's name is letters, digits, - or _, starting with a letter")
		}
		if named(m, op.Name) {
			return "", fmt.Errorf("%q is taken", op.Name)
		}
		var err error
		old := ""
		if op.Index >= 0 && op.Index < len(m.Parts) {
			old = m.Parts[op.Index].Name
		}
		if svg, err = setName(svg, op.Index, op.Name); err != nil {
			return "", err
		}
		for _, f := range m.Frames {
			if l, ok := f.Parts[old]; ok && old != "" {
				delete(f.Parts, old)
				f.Parts[op.Name] = l
			}
		}
		m.Parts = parts(svg)
	case "add-shape":
		var err error
		if svg, err = addShape(svg, m, op.Shape); err != nil {
			return "", err
		}
		m.Parts = parts(svg)
	case "del-shape":
		locs := shapeRE.FindAllStringIndex(svg, -1)
		if op.Index < 0 || op.Index >= len(locs) {
			return "", fmt.Errorf("no shape %d", op.Index)
		}
		a, z := locs[op.Index][0], locs[op.Index][1]
		if t := m.Parts[op.Index].Tag; !strings.HasSuffix(svg[a:z], "/>") {
			if e := strings.Index(svg[z:], "</"+t+">"); e >= 0 && (t != "g" || !strings.Contains(svg[z:z+e], "<g")) {
				z += e + len("</"+t+">")
			} else {
				return "", fmt.Errorf("only a shape without nested groups can be deleted here; use the text view")
			}
		}
		if n := m.Parts[op.Index].Name; n != "" {
			for _, f := range m.Frames {
				delete(f.Parts, n)
			}
		}
		svg = strings.TrimRight(svg[:a], " \t") + strings.TrimPrefix(svg[z:], "\n")
		m.Parts = parts(svg)
	default:
		return "", fmt.Errorf("unknown op %q", op.Kind)
	}
	return write(svg, m), nil
}

func renumber(m *Model) {
	for i := range m.Frames {
		m.Frames[i].Name = "frame" + strconv.Itoa(i+1)
	}
}

func named(m Model, name string) bool {
	for _, p := range m.Parts {
		if p.Name == name && name != "" {
			return true
		}
	}
	return false
}

func setName(svg string, index int, name string) (string, error) {
	locs := shapeRE.FindAllStringSubmatchIndex(svg, -1)
	if index < 0 || index >= len(locs) {
		return "", fmt.Errorf("no shape %d", index)
	}
	a, z := locs[index][0], locs[index][1]
	tag := svg[a:z]
	if partRE.MatchString(tag) {
		tag = partRE.ReplaceAllString(tag, ` data-part="`+name+`"`)
	} else {
		at := locs[index][3]
		tag = svg[a:at] + ` data-part="` + name + `"` + svg[at:z]
	}
	return svg[:a] + tag + svg[z:], nil
}

// addShape draws a new part in the middle of the board, coloured with a theme
// token (with a fallback for the file alone).
func addShape(svg string, m Model, shape string) (string, error) {
	x, y, w, h := 0.0, 0.0, 800.0, 450.0
	if f := strings.Fields(m.ViewBox); len(f) == 4 {
		x, _ = strconv.ParseFloat(f[0], 64)
		y, _ = strconv.ParseFloat(f[1], 64)
		w, _ = strconv.ParseFloat(f[2], 64)
		h, _ = strconv.ParseFloat(f[3], 64)
	}
	cx, cy, u := x+w/2, y+h/2, min(w, h)/8
	n := 1
	for named(m, shape+strconv.Itoa(n)) {
		n++
	}
	name := shape + strconv.Itoa(n)
	var el string
	switch shape {
	case "rect":
		el = fmt.Sprintf(`<rect data-part="%s" x="%s" y="%s" width="%s" height="%s" rx="%s" fill="var(--si-accent,#3657d6)"/>`, name, num(cx-u), num(cy-u*0.6), num(2*u), num(1.2*u), num(u/6))
	case "circle":
		el = fmt.Sprintf(`<circle data-part="%s" cx="%s" cy="%s" r="%s" fill="var(--si-accent-2,#0b8a7a)"/>`, name, num(cx), num(cy), num(u*0.7))
	case "text":
		el = fmt.Sprintf(`<text data-part="%s" x="%s" y="%s" text-anchor="middle" font-size="%s" font-weight="700" fill="var(--si-fg,#16171b)">Text</text>`, name, num(cx), num(cy), num(u*0.5))
	case "line":
		el = fmt.Sprintf(`<line data-part="%s" x1="%s" y1="%s" x2="%s" y2="%s" stroke="var(--si-muted,#5d626c)" stroke-width="%s" stroke-linecap="round"/>`, name, num(cx-2*u), num(cy), num(cx+2*u), num(cy), num(u/10))
	default:
		return "", fmt.Errorf("unknown shape %q (rect, circle, text, line)", shape)
	}
	i := strings.LastIndex(svg, "</svg>")
	if i < 0 {
		return "", fmt.Errorf("no </svg>")
	}
	return svg[:i] + el + "\n" + svg[i:], nil
}

// Snippet is what puts the animation on a slide: the canvas in frame 1, then
// one state step per later frame (frame N on, frame N-1 off).
func Snippet(path string, m Model) string {
	if len(m.Frames) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- svg: %s %s -->\n", path, m.Frames[0].Name)
	for i := 1; i < len(m.Frames); i++ {
		ops := "+" + m.Frames[i].Name
		if i > 1 {
			ops = "-" + m.Frames[i-1].Name + " " + ops
		}
		fmt.Fprintf(&b, "<!-- step: %s -->\n", ops)
	}
	return b.String()
}
