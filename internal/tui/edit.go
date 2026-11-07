package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"github.com/felipeg48/simple-ideas/internal/mindmap"
)

// Editing in the terminal Studio, through the same writers as the web Studio:
// a mind map page's topics (preview focused: ↑↓ pick a topic, enter renames,
// c adds a child, o a sibling, x deletes, [ ] move it) and a code window's
// look (the palette: window style, title, line numbers).

// mmRow is one line of a mind map page's editable tree: a map title (path nil)
// or a topic.
type mmRow struct {
	mapI  int
	path  []int
	depth int
	text  string
}

func mmRows(d *mindmap.Doc) []mmRow {
	var rows []mmRow
	for i, m := range d.Maps() {
		rows = append(rows, mmRow{mapI: i, text: m.Title})
		var walk func(ts []*mindmap.Topic, path []int)
		walk = func(ts []*mindmap.Topic, path []int) {
			for j, t := range ts {
				p := append(append([]int{}, path...), j)
				text := t.Text
				if t.Code != nil {
					text = strings.TrimSpace(t.Code.Lang + " code")
				}
				rows = append(rows, mmRow{mapI: i, path: p, depth: len(p), text: text})
				walk(t.Kids, p)
			}
		}
		walk(m.Topics, nil)
	}
	return rows
}

// mmDoc reads the previewed mind map page; nil when the preview is not one.
func (m *studioModel) mmDoc() (*mindmap.Doc, string) {
	if m.d == nil || !m.d.IsMindmap() {
		return nil, ""
	}
	raw, err := os.ReadFile(m.abs(m.file))
	if err != nil {
		return nil, ""
	}
	return mindmap.Parse(string(raw)), string(raw)
}

// mmKey handles a key on a focused mind map preview; false when not one.
func (m *studioModel) mmKey(k string) (bool, tea.Cmd) {
	d, _ := m.mmDoc()
	if d == nil {
		return false, nil
	}
	rows := mmRows(d)
	m.mmSel = max(0, min(m.mmSel, len(rows)-1))
	if len(rows) == 0 {
		return false, nil
	}
	r := rows[m.mmSel]
	topic := r.path != nil
	switch k {
	case "up", "k":
		m.mmSel = max(0, m.mmSel-1)
	case "down", "j":
		m.mmSel = min(len(rows)-1, m.mmSel+1)
	case "enter":
		if topic {
			m.field, m.overlay = "topic", "prompt"
			m.input = lineInput{prompt: "topic: ", value: r.text, cursor: len([]rune(r.text))}
		}
	case "c":
		return true, m.mmApply(mindmap.Op{Kind: "add-child", Map: r.mapI, Path: r.path, Text: "New topic"})
	case "o":
		if topic {
			return true, m.mmApply(mindmap.Op{Kind: "add-sibling", Map: r.mapI, Path: r.path, Text: "New topic"})
		}
	case "x", "delete", "backspace":
		if topic {
			return true, m.mmApply(mindmap.Op{Kind: "delete", Map: r.mapI, Path: r.path})
		}
	case "[", "]":
		if topic {
			return true, m.mmApply(mindmap.Op{Kind: map[string]string{"[": "up", "]": "down"}[k], Map: r.mapI, Path: r.path})
		}
	default:
		return false, nil
	}
	return true, nil
}

// mmRename writes the prompt's text as the selected topic's.
func (m *studioModel) mmRename(text string) tea.Cmd {
	d, _ := m.mmDoc()
	if d == nil {
		return nil
	}
	rows := mmRows(d)
	if m.mmSel >= len(rows) || rows[m.mmSel].path == nil {
		return nil
	}
	r := rows[m.mmSel]
	return m.mmApply(mindmap.Op{Kind: "rename", Map: r.mapI, Path: r.path, Text: text})
}

// mmApply runs one edit on the page, writes it and selects the topic the edit
// points at.
func (m *studioModel) mmApply(op mindmap.Op) tea.Cmd {
	d, _ := m.mmDoc()
	if d == nil {
		return nil
	}
	sel, err := d.Apply(op)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if err := os.WriteFile(m.abs(m.file), []byte(d.String()), 0o644); err != nil {
		m.status = err.Error()
		return nil
	}
	for i, r := range mmRows(d) {
		if r.mapI == sel.Map && fmt.Sprint(r.path) == fmt.Sprint(sel.Path) && (r.path != nil) == (sel.Path != nil) {
			m.mmSel = i
		}
	}
	m.status = op.Kind + " written to " + m.file
	m.loadPreview()
	return m.check()
}

// mmLines draws the editable tree of the focused mind map preview.
func (m *studioModel) mmLines(w int) []string {
	d, _ := m.mmDoc()
	if d == nil {
		return nil
	}
	st := m.st()
	out := []string{st.Foreground(m.pal.Muted).Render("↑↓ topic · enter rename · c child · o sibling · x delete · [ ] move")}
	for i, r := range mmRows(d) {
		line := strings.Repeat("  ", r.depth) + "• " + r.text
		s := st
		if r.path == nil {
			line, s = r.text, st.Bold(true).Foreground(m.pal.Accent)
			out = append(out, "")
		}
		if i == m.mmSel {
			s = s.Reverse(true)
		}
		out = append(out, s.Render(ansi.Truncate(line, w, "…")))
	}
	return out
}

// windowCommands are the palette's code window edits for every code block of
// the previewed file: window style, title, line numbers.
func (m *studioModel) windowCommands() []item {
	if m.file == "" || !isMarkdown(m.file) {
		return nil
	}
	raw, err := os.ReadFile(m.abs(m.file))
	if err != nil {
		return nil
	}
	var out []item
	for _, sh := range codeshow.Shows(string(raw)) {
		first := sh.Steps[0]
		line := first.Start + 1
		name := first.Attrs["title"]
		if name == "" {
			name = fmt.Sprintf("%s block at line %d", orName(first.Lang, "code"), line)
		}
		set := func(field, val string) func(m *studioModel) tea.Cmd {
			return func(m *studioModel) tea.Cmd { return m.windowSet(line, field, val) }
		}
		for _, w := range []string{"mac", "windows", "terminal", "none"} {
			out = append(out, item{label: "Window " + name + ": " + w, hint: "window=" + w, run: set("window", w)})
		}
		on := first.Attrs["numbers"] != "" || hasFlag(first.Info, "numbers")
		out = append(out, item{label: "Window " + name + ": line numbers " + map[bool]string{true: "off", false: "on"}[on], hint: "numbers", run: set("numbers", map[bool]string{true: "", false: "true"}[on])})
		l := line
		out = append(out, item{label: "Window " + name + ": title…", hint: "title=", run: func(m *studioModel) tea.Cmd {
			m.field, m.overlay, m.cwLine = "window title", "prompt", l
			m.input = lineInput{prompt: "window title: ", value: first.Attrs["title"], cursor: len([]rune(first.Attrs["title"]))}
			return nil
		}})
	}
	return out
}

func hasFlag(info, flag string) bool {
	for _, f := range strings.Fields(info) {
		if f == flag {
			return true
		}
	}
	return false
}

// windowSet writes one option of the code window at 1-based line.
func (m *studioModel) windowSet(line int, field, val string) tea.Cmd {
	file := m.abs(m.file)
	raw, err := os.ReadFile(file)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	kind := "set"
	if field == "title" {
		kind = "set-step"
	}
	out, _, err := codeshow.Apply(string(raw), line, codeshow.Op{Kind: kind, Field: field, Value: val})
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
		m.status = err.Error()
		return nil
	}
	m.status = fmt.Sprintf("line %d: %s set", line, field)
	m.loadPreview()
	return m.check()
}
