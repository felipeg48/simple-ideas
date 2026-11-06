// Package codeshow reads and edits code windows and code shows in Markdown
// text: fenced code blocks with window= or id= options, the
// `<!-- callout: LINE text at=dx,dy -->` notes right after them, and runs of
// fences with the same id= (the steps of a show). It works on lines, so an
// edit changes only the lines it is about; the renderer draws the windows,
// the Studio's Window view edits them through Apply.
package codeshow

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Callout is one note on a step: Line is the 1-based code line it points
// at, At the note's offset from that line's end (nil: the default place).
type Callout struct {
	Line int     `json:"line"`
	Text string  `json:"text"`
	At   *[2]int `json:"at,omitempty"`
	src  int     // 0-based line of the directive in the text
}

var (
	calloutRE = regexp.MustCompile(`^\s*<!--\s*callout:\s*(.*?)\s*-->\s*$`)
	headRE    = regexp.MustCompile(`^(\d+)\s+(.+)$`)
	atRE      = regexp.MustCompile(`\s+at=(-?\d+),(-?\d+)$`)
	fenceRE   = regexp.MustCompile("^(\\s*)(`{3,}|~{3,})(.*)$")
	attrRE    = regexp.MustCompile(`([\w-]+)(?:="([^"]*)"|=(\S+))?`)
)

// ParseCallout reads a callout directive line. ok is false for any other
// line; err is set for a callout that does not start with a line number.
func ParseCallout(s string) (c Callout, ok bool, err error) {
	m := calloutRE.FindStringSubmatch(s)
	if m == nil {
		return c, false, nil
	}
	v := m[1]
	if a := atRE.FindStringSubmatch(v); a != nil {
		x, _ := strconv.Atoi(a[1])
		y, _ := strconv.Atoi(a[2])
		c.At = &[2]int{x, y}
		v = strings.TrimSpace(v[:len(v)-len(a[0])])
	}
	h := headRE.FindStringSubmatch(v)
	if h == nil {
		return c, true, errors.New("a callout starts with the code line it points at: <!-- callout: 3 text -->")
	}
	c.Line, _ = strconv.Atoi(h[1])
	c.Text = strings.TrimSpace(h[2])
	return c, true, nil
}

// String writes the directive back.
func (c Callout) String() string {
	s := fmt.Sprintf("<!-- callout: %d %s", c.Line, c.Text)
	if c.At != nil {
		s += fmt.Sprintf(" at=%d,%d", c.At[0], c.At[1])
	}
	return s + " -->"
}

// Attrs reads a fence info string: the language and the options, in order.
// The forms a reader writes all work: `go title="a.go"`, ` title="a.go"`
// (no language: a first word with = is an option), pandoc's
// `{.go title="a.go"}`, and `go {title="a.go"}`. Without a language, the
// title's extension names it (config.csproj → xml).
func Attrs(info string) (lang string, keys []string, vals map[string]string) {
	s := strings.TrimSpace(info)
	first, rest, _ := strings.Cut(s, " ")
	switch {
	case strings.HasPrefix(s, "{"): // {.xml title="x"}
		first, rest = "", s
	case strings.Contains(first, "="): // title="x" with no language
		first, rest = "", s
	}
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "{") {
		rest = strings.TrimSuffix(strings.TrimPrefix(rest, "{"), "}")
	}
	lang = strings.Trim(first, "{}")
	vals = map[string]string{}
	for _, m := range classRE.FindAllStringSubmatch(rest, -1) { // pandoc: .xml is the language
		if lang == "" {
			lang = m[1]
		}
	}
	rest = classRE.ReplaceAllString(rest, "")
	for _, m := range attrRE.FindAllStringSubmatch(rest, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		if _, seen := vals[m[1]]; !seen {
			keys = append(keys, m[1])
		}
		vals[m[1]] = v
	}
	lang = strings.TrimPrefix(lang, ".")
	if lang == "" {
		lang = LangOf(vals["title"])
	}
	return lang, keys, vals
}

var classRE = regexp.MustCompile(`(?:^|\s)\.([\w#+-]+)`)

// LangOf is the language a file name suggests ("" when it suggests none).
func LangOf(name string) string {
	base := strings.ToLower(name)
	if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
		base = base[i+1:]
	}
	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") || base == "containerfile" {
		return "dockerfile"
	}
	if base == "makefile" {
		return "make"
	}
	i := strings.LastIndex(base, ".")
	if i < 0 {
		return ""
	}
	return map[string]string{
		"go": "go", "java": "java", "kt": "kotlin", "cs": "csharp", "py": "python", "js": "javascript", "mjs": "javascript",
		"ts": "javascript", "tsx": "javascript", "jsx": "javascript", "rs": "rust", "rb": "ruby", "php": "php", "swift": "swift",
		"scala": "scala", "c": "c", "h": "c", "cpp": "cpp", "cc": "cpp", "hpp": "cpp", "sh": "bash", "bash": "bash", "zsh": "bash",
		"ps1": "powershell", "sql": "sql", "yml": "yaml", "yaml": "yaml", "toml": "toml", "json": "json", "css": "css",
		"html": "html", "htm": "html", "xml": "xml", "csproj": "xml", "fsproj": "xml", "vbproj": "xml", "props": "xml",
		"targets": "xml", "nuspec": "xml", "resx": "xml", "xaml": "xml", "svg": "xml", "plist": "xml", "config": "xml",
	}[base[i+1:]]
}

// Info is Attrs without the order: the language and the options of a fence.
func Info(info string) (string, map[string]string) {
	lang, _, vals := Attrs(info)
	return lang, vals
}

// SetAttr sets key in a fence info string: val "" removes it, "true" writes
// a bare flag (numbers), a value with spaces is quoted.
func SetAttr(info, key, val string) string {
	lang, keys, vals := Attrs(info)
	if _, ok := vals[key]; !ok && val != "" {
		keys = append(keys, key)
	}
	vals[key] = val
	parts := []string{lang}
	for _, k := range keys {
		v := vals[k]
		switch {
		case k == key && val == "":
		case v == "true" || (v == "" && k != key):
			parts = append(parts, k)
		case strings.ContainsAny(v, " \t\"") || v == "":
			parts = append(parts, k+`="`+strings.ReplaceAll(v, `"`, "'")+`"`)
		default:
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, " ")
}

// Step is one fence of a show, with the callouts written right after it.
type Step struct {
	Start    int               `json:"start"` // 0-based line of the opening fence
	End      int               `json:"end"`   // 0-based line of the closing fence
	Last     int               `json:"last"`  // last line of its callouts (End when none)
	Marker   string            `json:"-"`     // indent + ``` of the fence
	Info     string            `json:"info"`
	Lang     string            `json:"lang"`
	Attrs    map[string]string `json:"attrs"`
	Code     string            `json:"code"`
	Callouts []Callout         `json:"callouts"`
}

// Show is a window: one step, or a run of fences with the same id=.
type Show struct {
	ID    string `json:"id"`
	Steps []Step `json:"steps"`
}

// Shows lists every fenced code block of text as a show (a plain fence is a
// one-step show without options), grouping same-id runs.
func Shows(text string) []Show {
	lines := strings.Split(text, "\n")
	var out []Show
	for i := 0; i < len(lines); i++ {
		st, ok := fenceAt(lines, i)
		if !ok {
			continue
		}
		id := st.Attrs["id"]
		if n := len(out); n > 0 && id != "" && out[n-1].ID == id && onlyBlank(lines, out[n-1].Steps[len(out[n-1].Steps)-1].Last+1, st.Start) {
			out[n-1].Steps = append(out[n-1].Steps, st)
		} else {
			out = append(out, Show{ID: id, Steps: []Step{st}})
		}
		i = st.Last
	}
	return out
}

// ShowAt is the show holding 1-based line (a fence or one of its callouts).
func ShowAt(text string, line int) (Show, bool) {
	for _, s := range Shows(text) {
		if line-1 >= s.Steps[0].Start && line-1 <= s.Steps[len(s.Steps)-1].Last {
			return s, true
		}
	}
	return Show{}, false
}

func onlyBlank(lines []string, from, to int) bool {
	for i := from; i < to && i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return false
		}
	}
	return true
}

// fenceAt reads the fence opening at line i, its code and its callouts.
func fenceAt(lines []string, i int) (Step, bool) {
	m := fenceRE.FindStringSubmatch(lines[i])
	if m == nil {
		return Step{}, false
	}
	st := Step{Start: i, Marker: m[1] + m[2], Info: strings.TrimSpace(m[3])}
	st.Lang, _, st.Attrs = Attrs(st.Info)
	end := -1
	for j := i + 1; j < len(lines); j++ {
		if strings.HasPrefix(strings.TrimSpace(lines[j]), m[2]) && strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(lines[j]), m[2][:1])) == "" {
			end = j
			break
		}
	}
	if end < 0 {
		return Step{}, false
	}
	st.End, st.Last = end, end
	st.Code = strings.Join(lines[i+1:end], "\n")
	for j := end + 1; j < len(lines); j++ {
		t := strings.TrimSpace(lines[j])
		if t == "" {
			continue
		}
		c, ok, err := ParseCallout(lines[j])
		if !ok || err != nil {
			break
		}
		c.src = j
		st.Callouts = append(st.Callouts, c)
		st.Last = j
	}
	return st, true
}

// Op is one edit from the Window view. Step indexes the show's steps; Index
// a callout of that step.
type Op struct {
	Kind  string `json:"kind"` // set, set-step, add-step, del-step, move-step, add-callout, edit-callout, move-callout, del-callout
	Step  int    `json:"step"`
	Index int    `json:"index"`
	Field string `json:"field,omitempty"`
	Value string `json:"value,omitempty"`
	Line  int    `json:"line,omitempty"`
	Text  string `json:"text,omitempty"`
	At    []int  `json:"at,omitempty"`
	To    int    `json:"to,omitempty"`
}

// Apply edits the show holding 1-based line in text and returns the new text
// and the 1-based line of the show's first fence afterwards.
func Apply(text string, line int, op Op) (string, int, error) {
	sh, ok := ShowAt(text, line)
	if !ok {
		return text, line, errors.New("no code block at that line")
	}
	lines := strings.Split(text, "\n")
	if op.Step < 0 || op.Step >= len(sh.Steps) {
		return text, line, errors.New("no such step")
	}
	st := &sh.Steps[op.Step]
	setInfo := func(s *Step, info string) {
		lines[s.Start] = s.Marker + info
		s.Info = info
	}
	switch op.Kind {
	case "set": // a show option: every step carries it, like the renderer reads it
		for i := range sh.Steps {
			setInfo(&sh.Steps[i], SetAttr(sh.Steps[i].Info, op.Field, op.Value))
		}
	case "set-step": // one step's own option: title, hl, step (its name)
		setInfo(st, SetAttr(st.Info, op.Field, op.Value))
	case "add-callout", "edit-callout", "move-callout", "del-callout":
		if op.Kind != "add-callout" && (op.Index < 0 || op.Index >= len(st.Callouts)) {
			return text, line, errors.New("no such callout")
		}
		switch op.Kind {
		case "add-callout":
			c := Callout{Line: op.Line, Text: strings.TrimSpace(op.Text)}
			if c.Line < 1 || c.Text == "" {
				return text, line, errors.New("a callout needs a line and a text")
			}
			lines = insert(lines, st.Last+1, c.String())
		case "edit-callout":
			c := st.Callouts[op.Index]
			if op.Line > 0 {
				c.Line = op.Line
			}
			if t := strings.TrimSpace(op.Text); t != "" {
				c.Text = t
			}
			lines[c.src] = c.String()
		case "move-callout":
			c := st.Callouts[op.Index]
			if len(op.At) != 2 {
				c.At = nil
			} else {
				c.At = &[2]int{op.At[0], op.At[1]}
			}
			lines[c.src] = c.String()
		case "del-callout":
			lines = append(lines[:st.Callouts[op.Index].src], lines[st.Callouts[op.Index].src+1:]...)
		}
	case "add-step", "del-step", "move-step":
		if sh.ID == "" && op.Kind == "add-step" { // a show needs an id
			id := "show"
			if t := st.Attrs["title"]; t != "" {
				id = slug(t)
			}
			for i := range sh.Steps {
				setInfo(&sh.Steps[i], SetAttr(sh.Steps[i].Info, "id", id))
			}
			sh.ID = id
		}
		blocks := make([][]string, len(sh.Steps))
		for i, s := range sh.Steps {
			blocks[i] = append([]string{}, lines[s.Start:s.Last+1]...)
		}
		switch op.Kind {
		case "add-step": // a copy of the step's code after it, without its callouts or name
			s := sh.Steps[op.Step]
			info := SetAttr(SetAttr(s.Info, "step", ""), "hl", "")
			dup := append([]string{s.Marker + info}, lines[s.Start+1:s.End+1]...)
			blocks = append(blocks[:op.Step+1], append([][]string{dup}, blocks[op.Step+1:]...)...)
		case "del-step":
			if len(blocks) == 1 {
				return text, line, errors.New("a show keeps at least one step")
			}
			blocks = append(blocks[:op.Step], blocks[op.Step+1:]...)
		case "move-step":
			if op.To < 0 || op.To >= len(blocks) {
				return text, line, errors.New("no such place")
			}
			b := blocks[op.Step]
			blocks = append(blocks[:op.Step], blocks[op.Step+1:]...)
			blocks = append(blocks[:op.To], append([][]string{b}, blocks[op.To:]...)...)
		}
		var region []string
		for i, b := range blocks {
			if i > 0 {
				region = append(region, "")
			}
			region = append(region, b...)
		}
		first, last := sh.Steps[0].Start, sh.Steps[len(sh.Steps)-1].Last
		lines = append(lines[:first], append(region, lines[last+1:]...)...)
	default:
		return text, line, fmt.Errorf("unknown edit %q", op.Kind)
	}
	return strings.Join(lines, "\n"), sh.Steps[0].Start + 1, nil
}

func insert(lines []string, at int, s string) []string {
	lines = append(lines, "")
	copy(lines[at+1:], lines[at:])
	lines[at] = s
	return lines
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	if s := strings.Trim(b.String(), "-"); s != "" {
		return s
	}
	return "show"
}

// Source is an article holding only the code windows of text (a fence with
// window= or id=; shows with the same id, on one page or across slides,
// joined in order), with look's front matter (theme, fonts, accent, mode).
// only keeps one show: its id, or window-N for the Nth block without one.
// It returns the source and the number of shows in it.
func Source(text string, look map[string]string, only string) (string, int) {
	lines := strings.Split(text, "\n")
	var order []string
	byID := map[string][]string{}
	for i, sh := range Shows(text) {
		if _, w := sh.Steps[0].Attrs["window"]; !w && sh.ID == "" {
			continue
		}
		id := sh.ID
		if id == "" {
			id = fmt.Sprintf("window-%d", i+1)
		}
		if only != "" && id != only {
			continue
		}
		if _, seen := byID[id]; !seen {
			order = append(order, id)
		}
		for _, st := range sh.Steps {
			byID[id] = append(byID[id], strings.Join(lines[st.Start:st.Last+1], "\n"))
		}
	}
	return article(look, order, byID), len(order)
}

// ShowSource is the article of one show (the Studio's Window view preview).
func ShowSource(text string, sh Show, look map[string]string) string {
	lines := strings.Split(text, "\n")
	var steps []string
	for _, st := range sh.Steps {
		steps = append(steps, strings.Join(lines[st.Start:st.Last+1], "\n"))
	}
	return article(look, []string{"show"}, map[string][]string{"show": steps})
}

func article(look map[string]string, order []string, byID map[string][]string) string {
	var b strings.Builder
	b.WriteString("---\ntype: article\ntitle: Code\n")
	for _, k := range []string{"theme", "fonts", "accent", "mode"} {
		if v := look[k]; v != "" {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	b.WriteString("---\n\n")
	for i, id := range order {
		if i > 0 {
			b.WriteString("\n\nThe next window.\n\n") // text between shows keeps two shows apart
		}
		b.WriteString(strings.Join(byID[id], "\n\n"))
	}
	return b.String()
}
