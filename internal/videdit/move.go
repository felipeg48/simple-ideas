package videdit

import (
	"fmt"
	"strings"
)

// Move is a callout dragged in the Studio's Video view, in source pixels:
// To a bubble's tail or an arrow's head, At a text's place, From an arrow's
// tail, Rect a rectangle (X, Y, W, H).
type Move struct {
	Line int     `json:"line"`
	To   *[2]int `json:"to,omitempty"`
	At   *[2]int `json:"at,omitempty"`
	From *[2]int `json:"from,omitempty"`
	Rect *[4]int `json:"rect,omitempty"`
	Key  *int    `json:"key,omitempty"` // with At: keyframe Key of a moving callout's path (from 0)
}

// Apply writes m into its line of src (a `type: video` file) and returns the
// new text: an arrow's ends and a rectangle in the action, `· to X,Y` /
// `· at X,Y` as an option (replaced, else added). The rest of the line, and
// every other line, is kept as written.
func (m Move) Apply(src string) (string, error) {
	lines := strings.Split(src, "\n")
	if m.Line < 1 || m.Line > len(lines) {
		return "", fmt.Errorf("no line %d", m.Line)
	}
	raw := strings.TrimRight(lines[m.Line-1], "\r")
	body := strings.TrimLeft(raw, " \t")
	indent := raw[:len(raw)-len(body)]
	if !strings.HasPrefix(body, "- ") && !strings.HasPrefix(body, "* ") {
		return "", fmt.Errorf("line %d is not a timeline line", m.Line)
	}
	text := strings.TrimSpace(body[2:])
	it, err := parseItem(text)
	if err != nil {
		return "", fmt.Errorf("line %d: %v", m.Line, err)
	}
	parts := splitDots(text)
	if m.Key != nil {
		if m.At == nil || !movePathPoint(parts, *m.Key, fmt.Sprintf("%d,%d", m.At[0], m.At[1])) {
			return "", fmt.Errorf("line %d: no place %d on its path", m.Line, *m.Key+1)
		}
		lines[m.Line-1] = indent + body[:2] + strings.Join(parts, " · ")
		return strings.Join(lines, "\n"), nil
	}
	if len(it.Path) > 0 {
		return "", fmt.Errorf("line %d moves along a path: drag its places", m.Line)
	}
	xy := func(p [2]int) string { return fmt.Sprintf("%d,%d", p[0], p[1]) }
	opt := func(name, val string) {
		for i := 2; i < len(parts); i++ {
			if w, _ := cutWord(parts[i]); w == name {
				parts[i] = name + " " + val
				return
			}
		}
		parts = append(parts, name+" "+val)
	}
	switch {
	case it.Action == "arrow" && (m.From != nil || m.To != nil):
		tail, head := posText(it.Tail), posText(it.Point)
		if m.From != nil {
			tail = xy(*m.From)
		}
		if m.To != nil {
			head = xy(*m.To)
		}
		parts[1] = "arrow from " + tail + " to " + head
	case m.Rect != nil && it.Rect != nil:
		r := *m.Rect
		sep := " " // X,Y WxH, or X,Y,WxH as the line had it
		if strings.Count(parts[1], ",") >= 2 {
			sep = ","
		}
		verb := strings.Fields(parts[1])[0]
		if it.Action == "zoom" || it.Action == "pan" { // the camera's rectangle
			verb += " to"
		}
		parts[1] = fmt.Sprintf("%s %d,%d%s%dx%d", verb, r[0], r[1], sep, max(1, r[2]), max(1, r[3]))
	case m.To != nil && it.Action == "bubble":
		if it.Point == nil && it.At != nil { // a bubble placed with at: keeps the word it was written with
			opt("at", xy(*m.To))
		} else {
			opt("to", xy(*m.To))
		}
	case m.At != nil && it.IsCallout() && it.Action != "title" && it.Rect == nil && it.Action != "arrow":
		opt("at", xy(*m.At))
	default:
		return "", fmt.Errorf("line %d (%s): nothing to move there", m.Line, word(it.Action))
	}
	lines[m.Line-1] = indent + body[:2] + strings.Join(parts, " · ")
	return strings.Join(lines, "\n"), nil
}

// movePathPoint writes xy as place k of the line's path (the · to A → B or
// · path A B part); sizes, arrows and · over stay as written.
func movePathPoint(parts []string, k int, xy string) bool {
	for i := 2; i < len(parts); i++ {
		w, rest := cutWord(parts[i])
		if w != "path" && !((w == "to" || w == "at") && (strings.Contains(rest, "→") || strings.Contains(rest, "->"))) {
			continue
		}
		f := strings.Fields(strings.NewReplacer("→", " → ", "->", " -> ").Replace(rest))
		n := 0
		for j, t := range f {
			if t == "over" {
				break
			}
			if _, err := pos(t); err != nil {
				continue
			}
			if n == k {
				f[j] = xy
				parts[i] = w + " " + strings.Join(f, " ")
				return true
			}
			n++
		}
		return false
	}
	return false
}

// posText is a place as written (X,Y, a percentage or a word).
func posText(p *Pos) string {
	if p == nil {
		return "center"
	}
	if p.Word != "" {
		return p.Word
	}
	n := func(v Num) string {
		if v.Pct {
			return fmt.Sprintf("%g%%", v.V)
		}
		return fmt.Sprintf("%g", v.V)
	}
	return n(p.X) + "," + n(p.Y)
}
