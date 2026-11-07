package tui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/studio"
)

// The Studio as a terminal app (si --terminal [dir]): the workspace's files,
// a live preview of the deck or document, the problems of si check, and the
// modes. Editing happens in $EDITOR, in this terminal or, inside tmux, in a
// pane beside it.

// StudioOptions configure si --terminal.
type StudioOptions struct {
	Theme string
	Run   func(args []string, out io.Writer) int // the si CLI, for Build
	Self  string                                 // the si binary, for Present
}

// Studio runs the terminal Studio on dir.
func Studio(dir string, opt StudioOptions) error {
	ws, err := studio.OpenWorkspace(dir)
	if err != nil {
		return err
	}
	m := &studioModel{ws: ws, opt: opt, pal: PaletteFor(opt.Theme, ""), expanded: map[string]bool{"": true}, mode: 1, k: 1 << 30}
	m.refreshTree()
	m.selectFirstDeck()
	m.loadPreview()
	_, err = tea.NewProgram(m).Run()
	return err
}

type entry struct {
	path  string // workspace-relative
	name  string
	depth int
	dir   bool
	kind  string // deck, article, guide, nav, markdown, image, …
	guide string // a folder holding a guide, book or site
	title string
}

type studioModel struct {
	ws       *studio.Workspace
	opt      StudioOptions
	pal      Palette
	w, h     int
	tree     studio.Node
	expanded map[string]bool
	rows     []entry
	sel, top int
	decks    map[string]studio.DeckInfo
	problems []studio.Problem
	checked  int
	focus    int // 0 files, 1 preview, 2 problems
	psel     int // selected problem

	// preview
	file   string // workspace-relative
	d      *deck.Deck
	perr   string
	slide  int
	k      int // builds shown (Rehearse steps through them)
	scroll int
	mtimes map[string]time.Time

	mode    int    // 1 Write, 2 Rehearse, 4 Lab (3 Present hands the terminal over)
	overlay string // help, new, newname, palette, storyboard, inspector, prompt, output
	input   lineInput
	list    []item
	lsel    int
	output  []string // build output, editor errors
	status  string
	newKind string
	field   string // the inspector field being edited
	mmSel   int    // the selected row of a mind map page's editable tree (edit.go)
	cwLine  int    // the code window whose title the prompt edits
	lab     *labState
}

type item struct {
	label, hint string
	run         func(m *studioModel) tea.Cmd
}

type checkMsg struct {
	problems []studio.Problem
	checked  int
}
type studioTick time.Time
type editorDone struct{ err error }
type buildDone struct{ out string }

func (m *studioModel) Init() tea.Cmd {
	return tea.Batch(m.check(), tea.Tick(time.Second, func(t time.Time) tea.Msg { return studioTick(t) }))
}

func (m *studioModel) check() tea.Cmd {
	ws := m.ws
	return func() tea.Msg {
		p, n := ws.Check()
		return checkMsg{p, n}
	}
}

// ---------- the tree ----------

func (m *studioModel) refreshTree() {
	if t, err := m.ws.Tree(); err == nil {
		m.tree = t
	}
	m.decks = map[string]studio.DeckInfo{}
	for _, d := range m.ws.Decks() {
		m.decks[d.Path] = d
	}
	m.rows = nil
	var walk func(n studio.Node, depth int)
	walk = func(n studio.Node, depth int) {
		for _, c := range n.Children {
			e := entry{path: c.Path, name: c.Name, depth: depth, dir: c.Kind == "dir", kind: c.Kind, guide: c.Guide}
			if d, ok := m.decks[c.Path]; ok {
				e.title = d.Title
			}
			m.rows = append(m.rows, e)
			if e.dir && m.expanded[c.Path] {
				walk(c, depth+1)
			}
		}
	}
	walk(m.tree, 0)
	m.sel = min(m.sel, max(0, len(m.rows)-1))
}

func (m *studioModel) selectFirstDeck() {
	for i, r := range m.rows {
		if r.kind == "deck" {
			m.sel = i
			return
		}
	}
}

func (m *studioModel) selectPath(p string) {
	// open the folders on the way
	parts := strings.Split(p, "/")
	for i := 1; i < len(parts); i++ {
		m.expanded[strings.Join(parts[:i], "/")] = true
	}
	m.refreshTree()
	for i, r := range m.rows {
		if r.path == p {
			m.sel = i
		}
	}
	m.loadPreview()
}

func (m *studioModel) abs(rel string) string {
	return filepath.Join(m.ws.Root, filepath.FromSlash(rel))
}

// ---------- the preview ----------

func (m *studioModel) loadPreview() {
	if m.sel >= len(m.rows) {
		return
	}
	r := m.rows[m.sel]
	if r.path != m.file {
		m.slide, m.scroll = 0, 0
	}
	m.file, m.d, m.perr = r.path, nil, ""
	if r.dir || !isMarkdown(r.path) {
		return
	}
	src := m.abs(r.path)
	// a file a deck includes previews in that deck, at its first slide
	for _, info := range m.decks {
		for _, f := range info.Files[1:] {
			if f == r.path {
				src = m.abs(info.Path)
			}
		}
	}
	d, err := parsePage(src, m.ws.Root)
	if err != nil {
		m.perr = err.Error()
		return
	}
	m.d = d
	m.slide = min(m.slide, max(0, len(d.Slides)-1))
	m.mtimes = fileTimes(d)
}

// parsePage parses a file with its guide's settings, as si check does.
func parsePage(file, root string) (*deck.Deck, error) {
	if g := deck.FindGuide(file, root); g != nil && g.Item(file) != nil {
		return deck.ParseWith(file, g.Defaults())
	}
	return deck.Parse(file)
}

func isMarkdown(p string) bool {
	e := strings.ToLower(filepath.Ext(p))
	return e == ".md" || e == ".adoc" || e == ".asciidoc"
}

func (m *studioModel) isSlides() bool { return m.d != nil && !m.d.IsDoc() }

// ---------- update ----------

func (m *studioModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case checkMsg:
		m.problems, m.checked = msg.problems, msg.checked
		m.psel = min(m.psel, max(0, len(m.problems)-1))
	case studioTick:
		var cmd tea.Cmd
		if m.d != nil && changed(m.mtimes, fileTimes(m.d)) {
			m.loadPreview()
			cmd = m.check()
		}
		if m.lab != nil {
			m.lab.poll()
		}
		return m, tea.Batch(cmd, tea.Tick(time.Second, func(t time.Time) tea.Msg { return studioTick(t) }))
	case editorDone:
		if msg.err != nil {
			m.status = "editor: " + msg.err.Error()
		}
		m.refreshTree()
		m.loadPreview()
		return m, m.check()
	case buildDone:
		m.overlay, m.output, m.lsel = "output", strings.Split(strings.TrimRight(msg.out, "\n"), "\n"), 0
		m.refreshTree()
	case labLine:
		if m.lab != nil {
			m.lab.add(msg.o)
		}
		return m, waitLine(msg.ch)
	case tea.PasteMsg:
		if m.overlay != "" {
			m.input.insert(msg.Content)
		}
	case tea.KeyPressMsg:
		if m.overlay != "" {
			return m, m.overlayKey(msg)
		}
		if m.mode == 4 && m.lab != nil {
			if cmd, done := m.labKey(msg); done {
				return m, cmd
			}
		}
		return m, m.key(msg)
	}
	return m, nil
}

func (m *studioModel) key(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	switch k {
	case "q", "ctrl+c":
		if m.lab != nil && m.lab.pane != "" {
			exec.Command("tmux", "kill-pane", "-t", m.lab.pane).Run()
		}
		return tea.Quit
	case "?":
		m.overlay = "help"
		return nil
	case "1":
		m.mode, m.k = 1, 1<<30
		return nil
	case "2":
		if m.isSlides() {
			m.mode, m.k = 2, 0
		} else {
			m.status = "Rehearse works on slides: pick a deck"
		}
		return nil
	case "3":
		return m.present()
	case "4":
		return m.openLab()
	case "tab":
		m.focus = (m.focus + 1) % 3
		return nil
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
		return nil
	case "e":
		return m.edit(m.editTarget())
	case "n":
		m.openNew()
		return nil
	case "b":
		return m.build()
	case "s":
		m.openStoryboard()
		return nil
	case "i":
		m.openInspector()
		return nil
	case ":", "ctrl+k":
		m.openPalette()
		return nil
	case "r":
		m.refreshTree()
		m.loadPreview()
		return m.check()
	}
	// navigation depends on focus and mode
	if m.mode == 2 {
		switch k {
		case "right", "space", "l", "n", "pgdown":
			m.rehearseStep(1)
		case "left", "h", "p", "pgup":
			m.rehearseStep(-1)
		}
		return nil
	}
	switch m.focus {
	case 0:
		switch k {
		case "up", "k":
			m.sel = max(0, m.sel-1)
			m.loadPreview()
		case "down", "j":
			m.sel = min(len(m.rows)-1, m.sel+1)
			m.loadPreview()
		case "enter", "right", "l":
			if m.sel < len(m.rows) && m.rows[m.sel].dir {
				p := m.rows[m.sel].path
				m.expanded[p] = !m.expanded[p] || k == "right" || k == "l"
				m.refreshTree()
			} else if k == "enter" {
				return m.edit(m.editTarget())
			} else {
				m.focus = 1
			}
		case "left", "h":
			if m.sel < len(m.rows) && m.rows[m.sel].dir && m.expanded[m.rows[m.sel].path] {
				m.expanded[m.rows[m.sel].path] = false
				m.refreshTree()
			}
		}
	case 1:
		if ok, cmd := m.mmKey(k); ok {
			return cmd
		}
		switch k {
		case "right", "l", "space", "pgdown":
			if m.isSlides() {
				m.slide = min(len(m.d.Slides)-1, m.slide+1)
			} else {
				m.scroll += 10
			}
		case "left", "h", "pgup":
			if m.isSlides() {
				m.slide = max(0, m.slide-1)
			} else {
				m.scroll = max(0, m.scroll-10)
			}
		case "down", "j":
			m.scroll++
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "g", "home":
			m.slide, m.scroll = 0, 0
		}
	case 2:
		switch k {
		case "up", "k":
			m.psel = max(0, m.psel-1)
		case "down", "j":
			m.psel = min(len(m.problems)-1, m.psel+1)
		case "enter":
			if m.psel < len(m.problems) {
				p := m.problems[m.psel]
				return m.editAt(p.File, p.Line)
			}
		}
	}
	return nil
}

func (m *studioModel) rehearseStep(dir int) {
	_, steps := Slide(m.d, m.slide, 0, 80, 24, m.pal)
	switch {
	case dir > 0 && m.k < steps:
		m.k++
	case dir > 0 && m.slide < len(m.d.Slides)-1:
		m.slide, m.k = m.slide+1, 0
	case dir < 0 && m.k > 0:
		m.k--
	case dir < 0 && m.slide > 0:
		m.slide--
		_, m.k = Slide(m.d, m.slide, 0, 80, 24, m.pal)
	}
}

// ---------- editing ----------

// editTarget is the file to edit: the slide's own file when the preview
// shows a slide of an included module.
func (m *studioModel) editTarget() (string, int) {
	if m.sel >= len(m.rows) || m.rows[m.sel].dir {
		return "", 0
	}
	if m.d != nil && m.slide < len(m.d.Slides) && m.focus == 1 {
		s := m.d.Slides[m.slide]
		if s.File != "" {
			rel, _ := filepath.Rel(m.ws.Root, s.File)
			return filepath.ToSlash(rel), s.Line
		}
	}
	return m.rows[m.sel].path, 0
}

func (m *studioModel) edit(rel string, line int) tea.Cmd { return m.editAt(rel, line) }

func (m *studioModel) editAt(rel string, line int) tea.Cmd {
	if rel == "" {
		return nil
	}
	editor := strings.Fields(os.Getenv("VISUAL"))
	if len(editor) == 0 {
		editor = strings.Fields(os.Getenv("EDITOR"))
	}
	if len(editor) == 0 {
		editor = []string{"vi"}
	}
	path := m.abs(rel)
	args := append([]string{}, editor[1:]...)
	base := filepath.Base(editor[0])
	switch {
	case line > 0 && (base == "hx" || base == "helix"):
		args = append(args, fmt.Sprintf("%s:%d", path, line))
	case line > 0 && (base == "code" || base == "cursor" || base == "zed"):
		args = append(args, "-g", fmt.Sprintf("%s:%d", path, line))
	case line > 0:
		args = append(args, "+"+strconv.Itoa(line), path)
	default:
		args = append(args, path)
	}
	if os.Getenv("TMUX") != "" { // a pane beside the Studio: the preview stays live while you type
		cmd := append([]string{"split-window", "-h", "-c", m.ws.Root, "--", editor[0]}, args...)
		if err := exec.Command("tmux", cmd...).Run(); err == nil {
			m.status = "editing " + rel + " in the pane beside; the preview follows every save"
			return nil
		}
	}
	c := exec.Command(editor[0], args...)
	c.Dir = m.ws.Root
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDone{err} })
}

// present hands the terminal to si play --terminal (and, inside tmux, opens
// the teleprompter in a pane beside it).
func (m *studioModel) present() tea.Cmd {
	if !m.isSlides() {
		m.status = "Present works on slides: pick a deck"
		return nil
	}
	if os.Getenv("TMUX") != "" {
		exec.Command("tmux", "split-window", "-h", "-l", "40%", "-c", m.ws.Root, "--", m.opt.Self, "play", m.d.Path, "--terminal", "--notes").Run()
	}
	c := exec.Command(m.opt.Self, "play", m.d.Path, "--terminal")
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDone{err} })
}

func (m *studioModel) build() tea.Cmd {
	if m.opt.Run == nil {
		m.status = "build is not available"
		return nil
	}
	m.status = "building…"
	root, run := m.ws.Root, m.opt.Run
	return func() tea.Msg {
		var out bytes.Buffer
		run([]string{"build", root, "-o", filepath.Join(root, "html")}, &out)
		return buildDone{out.String()}
	}
}

// ---------- overlays ----------

var starterLabels = []struct{ kind, label, what string }{
	{"deck", "Slides", "a talk: slides with a teleprompter script"},
	{"article", "Article", "one long read with a lede and contents"},
	{"guide", "Guide", "a folder of pages in order, with a sidebar"},
	{"book", "Book", "chapters, numbered sections, an index"},
	{"site", "Site", "a folder of pages with a menu"},
	{"page", "Site page", "one plain page for a site"},
	{"lab", "Lab", "hands-on steps with commands and output"},
	{"demo", "Demo script", "what the presenter says and runs"},
	{"mindmap", "Mind map", "topics that branch out; maps that open into maps"},
}

func (m *studioModel) openNew() {
	m.overlay, m.lsel, m.list = "new", 0, nil
	for _, s := range starterLabels {
		s := s
		m.list = append(m.list, item{label: s.label, hint: s.what, run: func(m *studioModel) tea.Cmd {
			m.newKind, m.overlay = s.kind, "newname"
			m.input = lineInput{prompt: s.label + " name: "}
			return nil
		}})
	}
}

func (m *studioModel) createStarter(name string) tea.Cmd {
	dir := ""
	if m.sel < len(m.rows) {
		r := m.rows[m.sel]
		dir = r.path
		if !r.dir {
			dir = filepath.ToSlash(filepath.Dir(r.path))
		}
		if dir == "." {
			dir = ""
		}
	}
	slug := slugify(name)
	if slug == "" {
		m.status = "give it a name"
		return nil
	}
	rel := filepath.ToSlash(filepath.Join(dir, slug))
	folder := m.newKind == "guide" || m.newKind == "book" || m.newKind == "site"
	if !folder {
		rel += ".md"
	}
	_, open, linked, err := studio.WriteStarter(m.newKind, m.abs(rel), name, m.ws.Root)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	o, _ := filepath.Rel(m.ws.Root, open)
	m.status = "created " + filepath.ToSlash(o)
	if linked != "" {
		l, _ := filepath.Rel(m.ws.Root, linked)
		m.status += ", added to " + filepath.ToSlash(l)
	}
	m.selectPath(filepath.ToSlash(o))
	return m.check()
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(strings.TrimSuffix(s, ".md"))) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func (m *studioModel) openStoryboard() {
	if !m.isSlides() {
		m.status = "the storyboard needs a deck"
		return
	}
	m.overlay, m.lsel, m.list = "storyboard", m.slide, nil
	for i, s := range m.d.Slides {
		i := i
		_, steps := Slide(m.d, i, 0, 80, 24, m.pal)
		hint := ""
		if s.Minutes > 0 {
			hint += fmt.Sprintf("%.1f min", s.Minutes)
		}
		if s.Layout != "" && s.Layout != "default" {
			hint += "  " + s.Layout
		}
		if steps > 0 {
			hint += fmt.Sprintf("  %d build(s)", steps)
		}
		if strings.TrimSpace(s.Notes) == "" {
			hint += "  no script"
		}
		m.list = append(m.list, item{label: fmt.Sprintf("%3d  %s", i+1, orName(stripMarks(s.Title), "(no heading)")), hint: strings.TrimSpace(hint),
			run: func(m *studioModel) tea.Cmd { m.slide, m.overlay, m.focus = i, "", 1; return nil }})
	}
}

var inspectorFields = []string{"layout", "transition", "background", "class", "eyebrow"}

func (m *studioModel) openInspector() {
	if !m.isSlides() {
		m.status = "the inspector needs a deck"
		return
	}
	s := m.d.Slides[m.slide]
	val := map[string]string{"layout": s.Layout, "transition": s.Transition, "background": s.Background, "class": strings.Join(s.Classes, " "), "eyebrow": s.Eyebrow}
	m.overlay, m.lsel, m.list = "inspector", 0, nil
	for _, f := range inspectorFields {
		f := f
		m.list = append(m.list, item{label: f, hint: orName(val[f], "—"), run: func(m *studioModel) tea.Cmd {
			m.field, m.overlay = f, "prompt"
			m.input = lineInput{prompt: f + ": ", value: val[f], cursor: len([]rune(val[f]))}
			return nil
		}})
	}
}

// setField writes the inspector's field into the slide's own lines.
func (m *studioModel) setField(value string) tea.Cmd {
	s := m.d.Slides[m.slide]
	file := s.File
	if file == "" {
		file = m.d.Path
	}
	end := 1 << 30
	for _, o := range m.d.Slides {
		if o.File == s.File && o.Line > s.Line && o.Line-1 < end {
			end = o.Line - 1
		}
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	out := deck.SetDirective(string(raw), s.Line, end, m.field, value)
	if out != string(raw) {
		if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
			m.status = err.Error()
			return nil
		}
		m.status = fmt.Sprintf("slide %d: %s set", m.slide+1, m.field)
	}
	m.loadPreview()
	return m.check()
}

func (m *studioModel) openPalette() {
	m.overlay, m.lsel = "palette", 0
	m.input = lineInput{prompt: ": "}
	m.filterPalette()
}

func (m *studioModel) commands() []item {
	cmds := []item{
		{"Mode: Write", "1", func(m *studioModel) tea.Cmd { m.mode = 1; return nil }},
		{"Mode: Rehearse", "2", func(m *studioModel) tea.Cmd { m.mode, m.k = 2, 0; return nil }},
		{"Present", "3", func(m *studioModel) tea.Cmd { return m.present() }},
		{"Mode: Lab", "4", func(m *studioModel) tea.Cmd { return m.openLab() }},
		{"New slides, article, guide…", "n", func(m *studioModel) tea.Cmd { m.openNew(); return nil }},
		{"Build the folder to HTML", "b", func(m *studioModel) tea.Cmd { return m.build() }},
		{"Storyboard", "s", func(m *studioModel) tea.Cmd { m.openStoryboard(); return nil }},
		{"Inspector", "i", func(m *studioModel) tea.Cmd { m.openInspector(); return nil }},
		{"Check the workspace again", "r", func(m *studioModel) tea.Cmd { return m.check() }},
	}
	cmds = append(cmds, m.windowCommands()...)
	var files []item
	var walk func(n studio.Node)
	walk = func(n studio.Node) {
		for _, c := range n.Children {
			if c.Kind == "dir" {
				walk(c)
				continue
			}
			p := c.Path
			files = append(files, item{label: c.Name, hint: p, run: func(m *studioModel) tea.Cmd { m.selectPath(p); return nil }})
		}
	}
	walk(m.tree)
	return append(cmds, files...)
}

func (m *studioModel) filterPalette() {
	q := m.input.value
	type scored struct {
		it item
		s  int
	}
	var all []scored
	for _, it := range m.commands() {
		s := fuzzyScore(it.label, q)
		if hs := fuzzyScore(it.hint, q) - 1; hs > s {
			s = hs
		}
		if s >= 0 {
			all = append(all, scored{it, s})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].s > all[j].s })
	m.list = nil
	for _, a := range all {
		if len(m.list) < 40 {
			m.list = append(m.list, a.it)
		}
	}
	m.lsel = 0
}

// fuzzyScore: a plain substring wins, then letters in order; -1 is no match.
func fuzzyScore(label, q string) int {
	l, s := strings.ToLower(label), strings.ToLower(strings.ReplaceAll(q, " ", ""))
	if s == "" {
		return 0
	}
	if i := strings.Index(l, s); i >= 0 {
		return 1000 - i
	}
	i, score, last := 0, 0, -2
	rs := []rune(s)
	for j, r := range []rune(l) {
		if i < len(rs) && r == rs[i] {
			if last == j-1 {
				score += 3
			} else {
				score++
			}
			last = j
			i++
		}
	}
	if i < len(rs) {
		return -1
	}
	return score
}

func (m *studioModel) overlayKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	if k == "ctrl+c" {
		return tea.Quit
	}
	if k == "esc" || (m.overlay == "help" || m.overlay == "output") && k != "up" && k != "down" && k != "j" && k != "k" {
		if k == "esc" || m.overlay == "help" || k == "q" || k == "enter" {
			m.overlay = ""
			return nil
		}
	}
	switch m.overlay {
	case "newname", "prompt":
		switch k {
		case "enter":
			v := m.input.value
			m.overlay = ""
			if m.field != "" && m.input.prompt == m.field+": " {
				defer func() { m.field = "" }()
				switch m.field {
				case "topic":
					return m.mmRename(v)
				case "window title":
					return m.windowSet(m.cwLine, "title", v)
				}
				return m.setField(v)
			}
			return m.createStarter(v)
		default:
			m.input.key(msg)
		}
		return nil
	case "palette":
		switch k {
		case "up", "ctrl+p":
			m.lsel = max(0, m.lsel-1)
		case "down", "ctrl+n":
			m.lsel = min(len(m.list)-1, m.lsel+1)
		case "enter":
			m.overlay = ""
			if m.lsel < len(m.list) {
				return m.list[m.lsel].run(m)
			}
		default:
			m.input.key(msg)
			m.filterPalette()
		}
		return nil
	case "output":
		switch k {
		case "up", "k":
			m.lsel = max(0, m.lsel-1)
		case "down", "j":
			m.lsel = min(max(0, len(m.output)-1), m.lsel+1)
		}
		return nil
	}
	// lists: new, storyboard, inspector
	switch k {
	case "up", "k":
		m.lsel = max(0, m.lsel-1)
	case "down", "j":
		m.lsel = min(len(m.list)-1, m.lsel+1)
	case "enter", "right", "l":
		if m.lsel < len(m.list) {
			return m.list[m.lsel].run(m)
		}
	case "q":
		m.overlay = ""
	}
	return nil
}

// ---------- view ----------

func (m *studioModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.BackgroundColor = m.pal.BG
	v.ForegroundColor = m.pal.FG
	v.WindowTitle = "si · " + filepath.Base(m.ws.Root)
	return v
}

func (m *studioModel) st() lipgloss.Style { return lipgloss.NewStyle().Foreground(m.pal.FG) }

func (m *studioModel) render() string {
	if m.w < 50 || m.h < 14 {
		return "make the terminal at least 50×14"
	}
	st, mu := m.st(), m.st().Foreground(m.pal.Muted)
	// top: the modes and where we are
	var tabs []string
	for i, name := range []string{"Write", "Rehearse", "Present", "Lab"} {
		t := fmt.Sprintf(" %d %s ", i+1, name)
		if m.mode == i+1 {
			tabs = append(tabs, lipgloss.NewStyle().Background(m.pal.Accent).Foreground(m.pal.BG).Bold(true).Render(t))
		} else {
			tabs = append(tabs, mu.Render(t))
		}
	}
	top := " " + strings.Join(tabs, "") + mu.Render("   "+filepath.Base(m.ws.Root)+" › "+m.file)
	top = ansi.Truncate(top, m.w, "…")

	bodyH := m.h - 2
	var body []string
	switch m.mode {
	case 2:
		body = m.rehearseView(m.w, bodyH)
	case 4:
		body = m.labView(m.w, bodyH)
	default:
		body = m.writeView(m.w, bodyH)
	}
	keys := " e edit · n new · b build · s storyboard · i inspector · : commands · tab focus · ? keys · q quit"
	if m.status != "" {
		keys = " " + m.status
	}
	lines := append([]string{top}, body...)
	lines = append(lines, mu.Render(ansi.Truncate(keys, m.w, "…")))
	out := strings.Join(lines, "\n")
	if m.overlay != "" {
		out = m.overlayOn(out)
	}
	_ = st
	return out
}

// pane draws lines in a titled box of exactly w × h.
func (m *studioModel) pane(title string, lines []string, w, h int, focused bool) []string {
	border := m.pal.Line
	if focused {
		border = m.pal.Accent
	}
	r := &textRenderer{pal: m.pal}
	inner := h - 2
	if len(lines) > inner {
		lines = lines[:inner]
	}
	for len(lines) < inner {
		lines = append(lines, "")
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > w-4 {
			lines[i] = ansi.Truncate(l, w-4, "…")
		}
	}
	return r.box(title, lines, w-4, border)
}

func (m *studioModel) writeView(w, h int) []string {
	fw := min(36, max(24, w/4))
	probH := min(8, max(3, len(m.problems)+2))
	if len(m.problems) == 0 {
		probH = 3
	}
	mainH := h - probH
	// files
	var files []string
	view := mainH - 2
	if m.sel < m.top {
		m.top = m.sel
	}
	if m.sel >= m.top+view {
		m.top = m.sel - view + 1
	}
	for i := m.top; i < len(m.rows) && i < m.top+view; i++ {
		files = append(files, m.fileRow(m.rows[i], i == m.sel, fw-4))
	}
	left := m.pane("Files", files, fw, mainH, m.focus == 0)
	right := m.pane(m.previewTitle(), m.previewLines(w-fw-4, mainH-2), w-fw, mainH, m.focus == 1)
	var out []string
	for i := range left {
		out = append(out, left[i]+right[i])
	}
	// problems
	var probs []string
	if len(m.problems) == 0 {
		probs = []string{m.st().Foreground(m.pal.OK).Render("✓ ") + m.st().Render(fmt.Sprintf("%d files checked, no problems", m.checked))}
	}
	for i, p := range m.problems {
		loc := p.File
		if p.Line > 0 {
			loc += ":" + strconv.Itoa(p.Line)
		}
		l := m.st().Foreground(m.pal.Warn).Render("▲ ") + m.st().Foreground(m.pal.Muted).Render(loc+"  ") + m.st().Render(p.Msg)
		if i == m.psel && m.focus == 2 {
			l = lipgloss.NewStyle().Background(m.pal.HL).Render(ansi.Strip(l))
		}
		probs = append(probs, l)
	}
	if m.psel >= probH-2 {
		probs = probs[m.psel-(probH-3):]
	}
	title := "Problems"
	if n := len(m.problems); n > 0 {
		title = fmt.Sprintf("Problems (%d) · Enter opens the line", n)
	}
	return append(out, m.pane(title, probs, w, probH, m.focus == 2)...)
}

var kindIcon = map[string]string{
	"deck": "▣", "article": "▤", "page": "▢", "guide": "▦", "book": "▥", "site": "◍", "demo": "▶", "lab": "⚗", "mindmap": "✲",
	"nav": "≡", "markdown": "·", "image": "▨", "svg": "◇", "css": "#", "json": "{", "other": " ",
}

func (m *studioModel) fileRow(e entry, on bool, w int) string {
	st := m.st()
	ind := strings.Repeat("  ", e.depth)
	icon := kindIcon[e.kind]
	col := m.pal.FG
	switch {
	case e.dir:
		icon = "▸"
		if m.expanded[e.path] {
			icon = "▾"
		}
		col = m.pal.Accent2
	case e.kind == "deck":
		col = m.pal.Accent
	case e.kind == "nav", e.kind == "other":
		col = m.pal.Muted
	}
	label := st.Foreground(col).Render(icon) + " " + st.Render(e.name)
	if e.guide != "" {
		label += st.Foreground(m.pal.OK).Render(" " + e.guide)
	}
	if e.title != "" && e.title != strings.TrimSuffix(e.name, ".md") {
		label += st.Foreground(m.pal.Muted).Render("  " + stripMarks(e.title))
	}
	l := ind + label
	if ansi.StringWidth(l) > w {
		l = ansi.Truncate(l, w, "…")
	}
	if on {
		bg := m.pal.Surface
		if m.focus == 0 {
			bg = m.pal.HL
		}
		l = lipgloss.NewStyle().Background(bg).Render(ansi.Strip(l) + strings.Repeat(" ", max(0, w-ansi.StringWidth(l))))
	}
	return l
}

func (m *studioModel) previewTitle() string {
	switch {
	case m.d == nil:
		return "Preview"
	case m.isSlides():
		return fmt.Sprintf("Slide %d/%d · %s", m.slide+1, len(m.d.Slides), m.d.Title())
	default:
		return m.d.Meta.Type + " · " + m.d.Title()
	}
}

// previewLines is the preview pane's content at w × h.
func (m *studioModel) previewLines(w, h int) []string {
	mu := m.st().Foreground(m.pal.Muted)
	if m.perr != "" {
		return []string{m.st().Foreground(m.pal.Crit).Render(m.perr)}
	}
	if m.sel >= len(m.rows) {
		return nil
	}
	r := m.rows[m.sel]
	switch {
	case r.dir:
		return []string{mu.Render("a folder: Enter opens it, n creates something in it")}
	case m.d != nil && m.isSlides():
		s, _ := Slide(m.d, m.slide, m.k, w, h-1, m.pal)
		lines := strings.Split(s, "\n")
		return append(lines, mu.Render("← → slides (Tab to focus here) · s storyboard · i inspector · 3 present"))
	case m.d != nil:
		return m.docLines(w, h)
	case r.kind == "image":
		tr := &textRenderer{pal: m.pal, w: w, maxH: h - 2, dir: m.ws.Root}
		return tr.image(r.path, "")
	}
	raw, err := os.ReadFile(m.abs(r.path))
	if err != nil {
		return []string{mu.Render(err.Error())}
	}
	lines := strings.Split(string(raw), "\n")
	if m.scroll < len(lines) {
		lines = lines[m.scroll:]
	}
	for i := range lines {
		lines[i] = mu.Render(ansi.Truncate(strings.ReplaceAll(lines[i], "\t", "  "), w, "…"))
	}
	return lines
}

// docLines renders a document (article, guide page, lab…) as one scrolling page.
func (m *studioModel) docLines(w, h int) []string {
	st := m.st()
	var out []string
	if t := m.d.Title(); t != "" {
		out = append(out, st.Bold(true).Render(stripMarks(t)), st.Foreground(m.pal.Accent).Render("━━━━━━"))
	}
	if desc := m.d.Meta.Description; desc != "" {
		out = append(out, st.Foreground(m.pal.Muted).Render(desc))
	}
	if m.d.IsMindmap() && m.focus == 1 { // a focused mind map page: the tree to edit (edit.go)
		out = append(out, m.mmLines(w)...)
		m.scroll = max(0, min(max(min(m.scroll, m.mmSel), m.mmSel+3-h+1), len(out)-1))
		return out[min(m.scroll, len(out)):]
	}
	for i, s := range m.d.Slides {
		r := &textRenderer{pal: m.pal, dir: s.Dir, w: w, maxH: h / 2, k: 1 << 30}
		if m.d.IsMindmap() { // a mind map page: every map as a tree
			out = append(out, "")
			out = append(out, r.mindmap(s.Body)...)
			continue
		}
		if i > 0 && s.Title != "" {
			out = append(out, "", st.Bold(true).Foreground(m.pal.Accent2).Render(stripMarks(s.Title)))
		}
		for _, b := range r.body(s.Body) {
			out = append(out, "")
			out = append(out, b.lines...)
		}
	}
	m.scroll = min(m.scroll, max(0, len(out)-h+1))
	return out[m.scroll:]
}

// rehearseView: the slide, and its script beside it.
func (m *studioModel) rehearseView(w, h int) []string {
	if !m.isSlides() {
		return m.writeView(w, h)
	}
	sw := w * 3 / 5
	s := m.d.Slides[m.slide]
	slide, steps := Slide(m.d, m.slide, m.k, sw-4, h-3, m.pal)
	left := m.pane(fmt.Sprintf("Slide %d/%d · build %d/%d", m.slide+1, len(m.d.Slides), m.k, steps), strings.Split(slide, "\n"), sw, h, false)
	cue := 0
	src := clickRE.ReplaceAllStringFunc(s.Notes, func(string) string { cue++; return fmt.Sprintf("CUE%d", cue) })
	nr := &textRenderer{pal: m.pal, w: w - sw - 4, k: 1 << 30}
	var script []string
	for i, b := range nr.body(src) {
		if i > 0 {
			script = append(script, "")
		}
		script = append(script, b.lines...)
	}
	for i := range script {
		script[i] = cueMarks(script[i], m.k, m.pal)
	}
	if len(script) == 0 {
		script = []string{m.st().Foreground(m.pal.Muted).Render("No Notes: on this slide.")}
	}
	right := m.pane(fmt.Sprintf("Script · %.1f min", s.Minutes), script, w-sw, h, true)
	var out []string
	for i := range left {
		out = append(out, left[i]+right[i])
	}
	return out
}

// cueMarks draws the [CLICK] placeholders of a script line: done ✓, next ▶ CLICK.
func cueMarks(l string, k int, pal Palette) string {
	st := lipgloss.NewStyle().Foreground(pal.FG)
	return cueRE.ReplaceAllStringFunc(l, func(c string) string {
		n, _ := strconv.Atoi(cueRE.FindStringSubmatch(c)[1])
		switch {
		case n <= k:
			return st.Foreground(pal.Muted).Render("✓ click")
		case n == k+1:
			return lipgloss.NewStyle().Background(pal.Accent).Foreground(pal.BG).Bold(true).Render(" ▶ CLICK ")
		}
		return st.Foreground(pal.Accent).Render("▶ click")
	})
}

// overlayOn draws the open overlay in a box over the screen's middle.
func (m *studioModel) overlayOn(screen string) string {
	st, mu := m.st(), m.st().Foreground(m.pal.Muted)
	ow := min(m.w-4, 76)
	var title string
	var lines []string
	listLines := func(view int) []string {
		var out []string
		first := max(0, min(m.lsel-view/2, len(m.list)-view))
		for i := first; i < len(m.list) && i < first+view; i++ {
			it := m.list[i]
			l := st.Render(it.label)
			if it.hint != "" {
				l += mu.Render("  " + it.hint)
			}
			l = ansi.Truncate(l, ow-4, "…")
			if i == m.lsel {
				l = lipgloss.NewStyle().Background(m.pal.HL).Render(ansi.Strip(l) + strings.Repeat(" ", max(0, ow-4-ansi.StringWidth(l))))
			}
			out = append(out, l)
		}
		return out
	}
	view := min(m.h-8, 18)
	switch m.overlay {
	case "help":
		title = "Keys"
		for _, r := range [][2]string{
			{"↑ ↓ / j k", "move in the focused pane"}, {"Tab", "focus: files, preview, problems"}, {"Enter", "open a folder, edit a file, go to a problem"},
			{"e", "edit in $EDITOR (a pane beside, in tmux)"}, {"← →", "slides, in the preview"}, {"n", "new slides, article, guide, site…"},
			{"b", "build the folder to html/"}, {"s", "storyboard"}, {"i", "inspector: layout, transition, background…"},
			{": or Ctrl+K", "files and commands"}, {"1 2 3 4", "Write, Rehearse, Present, Lab"}, {"r", "check again"}, {"q", "quit"},
		} {
			lines = append(lines, st.Foreground(m.pal.Accent).Bold(true).Render(fmt.Sprintf("%-14s", r[0]))+st.Render(r[1]))
		}
	case "new":
		title, lines = "New · Enter picks, Esc closes", listLines(view)
	case "storyboard":
		title, lines = "Storyboard · Enter goes to the slide", listLines(view)
	case "inspector":
		title, lines = fmt.Sprintf("Inspector · slide %d · Enter edits a field", m.slide+1), listLines(view)
	case "newname", "prompt":
		title = "Enter saves · Esc cancels"
		if m.overlay == "prompt" {
			title = "Empty removes the directive · Enter saves · Esc cancels"
		}
		lines = []string{m.input.view(m.pal, ow-4)}
	case "palette":
		title = "Files and commands"
		lines = append([]string{m.input.view(m.pal, ow-4), ""}, listLines(view-2)...)
	case "output":
		title = "Build · Esc closes"
		first := min(m.lsel, max(0, len(m.output)-view))
		for i := first; i < len(m.output) && i < first+view; i++ {
			lines = append(lines, ansi.Truncate(m.output[i], ow-4, "…"))
		}
	}
	r := &textRenderer{pal: m.pal}
	box := r.box(title, lines, ow-4, m.pal.Accent)
	rows := strings.Split(screen, "\n")
	y0 := max(1, (len(rows)-len(box))/2)
	x0 := max(0, (m.w-ow)/2)
	for i, b := range box {
		y := y0 + i
		if y >= len(rows) {
			break
		}
		row := rows[y]
		left := ansi.Truncate(row, x0, "")
		left += strings.Repeat(" ", max(0, x0-ansi.StringWidth(left)))
		right := ansi.Cut(row, x0+ow, ansi.StringWidth(row))
		rows[y] = left + b + right
	}
	return strings.Join(rows, "\n")
}

// ---------- a one-line text input ----------

type lineInput struct {
	prompt string
	value  string
	cursor int // in runes
}

func (in *lineInput) insert(s string) {
	r := []rune(in.value)
	in.value = string(r[:in.cursor]) + s + string(r[in.cursor:])
	in.cursor += len([]rune(s))
}

func (in *lineInput) key(msg tea.KeyPressMsg) {
	r := []rune(in.value)
	switch msg.String() {
	case "backspace":
		if in.cursor > 0 {
			in.value = string(r[:in.cursor-1]) + string(r[in.cursor:])
			in.cursor--
		}
	case "delete":
		if in.cursor < len(r) {
			in.value = string(r[:in.cursor]) + string(r[in.cursor+1:])
		}
	case "left":
		in.cursor = max(0, in.cursor-1)
	case "right":
		in.cursor = min(len(r), in.cursor+1)
	case "home", "ctrl+a":
		in.cursor = 0
	case "end", "ctrl+e":
		in.cursor = len(r)
	case "ctrl+u":
		in.value, in.cursor = "", 0
	default:
		if msg.Text != "" {
			in.insert(msg.Text)
		}
	}
}

func (in *lineInput) view(pal Palette, w int) string {
	st := lipgloss.NewStyle().Foreground(pal.FG)
	r := []rune(in.value)
	cur := " "
	if in.cursor < len(r) {
		cur = string(r[in.cursor])
	}
	after := ""
	if in.cursor < len(r) {
		after = string(r[in.cursor+1:])
	}
	s := st.Foreground(pal.Accent).Render(in.prompt) + st.Render(string(r[:in.cursor])) +
		lipgloss.NewStyle().Background(pal.FG).Foreground(pal.BG).Render(cur) + st.Render(after)
	return ansi.Truncate(s, w, "…")
}
