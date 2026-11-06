package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Lab mode (4 in si --terminal): the slide, the deck's runbook and demo
// commands, and a shell. Inside tmux the shell is a real pane below: Enter
// types the command there, x runs it. Outside tmux Enter runs the command
// and shows its output here, and ! opens a full shell until you exit it.

type labCmd struct {
	group, label, desc, command string
}

type labState struct {
	cmds   []labCmd
	sel    int
	focus  int // 0 commands, 1 slide
	out    []string
	pane   string // the tmux shell pane, "" outside tmux
	ch     chan labOutput
	vars   map[string]string // placeholder values, kept for the session
	asking []string          // placeholders still to ask for the pending command
	pend   *labCmd
	runIt  bool
}

type labOutput struct {
	line string
	done bool
}

func (l *labState) poll() {}

func (l *labState) add(o labOutput) {
	if o.line != "" || !o.done {
		l.out = append(l.out, o.line)
	}
	if o.done {
		l.out = append(l.out, "")
	}
	if len(l.out) > 2000 {
		l.out = l.out[len(l.out)-2000:]
	}
}

// labCommands are the deck's runbook entries and the shell blocks of its demo scripts.
func labCommands(d *deck.Deck) []labCmd {
	var out []labCmd
	base := filepath.Dir(d.Path)
	if rb := strings.TrimSpace(d.Meta.Runbook); rb != "" {
		p := rb
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, rb)
		}
		var book struct {
			Categories []struct {
				Name    string `json:"name"`
				Entries []struct {
					Label, Description, Command string
				} `json:"entries"`
			} `json:"categories"`
		}
		if raw, err := os.ReadFile(p); err == nil && json.Unmarshal(raw, &book) == nil {
			for _, c := range book.Categories {
				for _, e := range c.Entries {
					out = append(out, labCmd{group: c.Name, label: e.Label, desc: e.Description, command: e.Command})
				}
			}
		}
	}
	if spec := strings.TrimSpace(d.Meta.Demos); spec != "" {
		files, _ := render.DemoFiles(base, spec)
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			title := strings.TrimSuffix(filepath.Base(f), ".md")
			if dd, err := deck.Parse(f); err == nil && dd.Title() != "" {
				title = dd.Title()
			}
			doc := md.Parser().Parse(text.NewReader(raw))
			heading := ""
			for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
				if h, ok := n.(*ast.Heading); ok {
					heading = render.TextOf(h, raw)
				}
				fc, ok := n.(*ast.FencedCodeBlock)
				if !ok || !render.RunLang(string(fc.Language(raw))) {
					continue
				}
				code := strings.TrimSpace(blockText(fc, raw))
				label := strings.SplitN(code, "\n", 2)[0]
				out = append(out, labCmd{group: "Demo · " + title, label: ansi.Truncate(label, 60, "…"), desc: heading, command: code})
			}
		}
	}
	return out
}

func (m *studioModel) openLab() tea.Cmd {
	if !m.isSlides() {
		m.status = "Lab works on a deck: pick one"
		return nil
	}
	l := &labState{cmds: labCommands(m.d), vars: map[string]string{}}
	if len(l.cmds) == 0 {
		m.status = "this deck has no runbook: or demos: (front matter) for Lab"
	}
	if os.Getenv("TMUX") != "" {
		out, err := exec.Command("tmux", "split-window", "-v", "-l", "35%", "-d", "-c", m.ws.Root, "-P", "-F", "#{pane_id}").Output()
		if err == nil {
			l.pane = strings.TrimSpace(string(out))
		}
	}
	m.lab, m.mode, m.k = l, 4, 1<<30
	return nil
}

var placeholderRE = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}`)

// labKey handles Lab's own keys; done is false for keys the Studio handles.
func (m *studioModel) labKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	l := m.lab
	k := msg.String()
	if l.asking != nil { // filling a placeholder
		switch k {
		case "esc":
			l.asking, l.pend = nil, nil
		case "enter":
			l.vars[l.asking[0]] = m.input.value
			l.asking = l.asking[1:]
			if len(l.asking) > 0 {
				m.askVar()
				return nil, true
			}
			c := *l.pend
			l.asking, l.pend = nil, nil
			c.command = placeholderRE.ReplaceAllStringFunc(c.command, func(s string) string {
				return l.vars[placeholderRE.FindStringSubmatch(s)[1]]
			})
			return m.labRun(c, l.runIt), true
		default:
			m.input.key(msg)
		}
		return nil, true
	}
	switch k {
	case "up", "k":
		if l.focus == 0 {
			l.sel = max(0, l.sel-1)
			return nil, true
		}
	case "down", "j":
		if l.focus == 0 {
			l.sel = min(len(l.cmds)-1, l.sel+1)
			return nil, true
		}
	case "tab":
		l.focus = 1 - l.focus
		return nil, true
	case "right", "space", "pgdown":
		m.rehearseStep(1)
		return nil, true
	case "left", "pgup":
		m.rehearseStep(-1)
		return nil, true
	case "enter", "x", "shift+enter":
		if l.sel < len(l.cmds) {
			return m.labStart(l.cmds[l.sel], k != "enter" || l.pane == ""), true
		}
		return nil, true
	case "!":
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		c := exec.Command(shell)
		c.Dir = m.ws.Root
		return tea.ExecProcess(c, func(err error) tea.Msg { return editorDone{err} }), true
	case "ctrl+t":
		if os.Getenv("TMUX") != "" {
			exec.Command("tmux", "split-window", "-h", "-d", "-c", m.ws.Root).Run()
			m.status = "a new shell pane beside"
		}
		return nil, true
	case "c":
		l.out = nil
		return nil, true
	case "1", "2", "3":
		if l.pane != "" {
			exec.Command("tmux", "kill-pane", "-t", l.pane).Run()
		}
		m.lab = nil
		return nil, false
	}
	return nil, false
}

// labStart asks for the command's placeholders, then types or runs it.
func (m *studioModel) labStart(c labCmd, run bool) tea.Cmd {
	l := m.lab
	var names []string
	seen := map[string]bool{}
	for _, p := range placeholderRE.FindAllStringSubmatch(c.command, -1) {
		if !seen[p[1]] {
			seen[p[1]] = true
			names = append(names, p[1])
			if _, ok := l.vars[p[1]]; !ok {
				l.vars[p[1]] = p[2]
			}
		}
	}
	if len(names) == 0 {
		return m.labRun(c, run)
	}
	l.asking, l.pend, l.runIt = names, &c, run
	m.askVar()
	return nil
}

func (m *studioModel) askVar() {
	name := m.lab.asking[0]
	v := m.lab.vars[name]
	m.input = lineInput{prompt: name + ": ", value: v, cursor: len([]rune(v))}
}

// labRun types the command into the tmux shell pane (and presses Enter when
// run), or, outside tmux, runs it and streams its output into the pane.
func (m *studioModel) labRun(c labCmd, run bool) tea.Cmd {
	l := m.lab
	if l.pane != "" {
		args := []string{"send-keys", "-t", l.pane, "-l", c.command}
		if err := exec.Command("tmux", args...).Run(); err == nil {
			if run {
				exec.Command("tmux", "send-keys", "-t", l.pane, "Enter").Run()
				m.status = "ran: " + c.label
			} else {
				m.status = "typed into the shell below: press x to run it, or Enter there"
			}
			return nil
		}
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", c.command)
	cmd.Dir = m.ws.Root
	pr, pw, err := os.Pipe()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	cmd.Stdout, cmd.Stderr = pw, pw
	l.add(labOutput{line: "$ " + c.command})
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		l.add(labOutput{line: err.Error(), done: true})
		return nil
	}
	pw.Close()
	ch := make(chan labOutput, 64)
	l.ch = ch
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			ch <- labOutput{line: sc.Text()}
		}
		err := cmd.Wait()
		pr.Close()
		end := "exit 0"
		if err != nil {
			end = err.Error()
		}
		ch <- labOutput{line: "[" + end + "]", done: true}
		close(ch)
	}()
	m.status = "running: " + c.label
	return waitLine(ch)
}

func waitLine(ch chan labOutput) tea.Cmd {
	return func() tea.Msg {
		o, ok := <-ch
		if !ok {
			return nil
		}
		return labLine{o, ch}
	}
}

type labLine struct {
	o  labOutput
	ch chan labOutput
}

func (m *studioModel) labView(w, h int) []string {
	l := m.lab
	if l == nil || m.d == nil {
		return m.writeView(w, h)
	}
	st, mu := m.st(), m.st().Foreground(m.pal.Muted)
	cw := min(48, w*2/5)
	sw := w - cw
	outH := 8
	if l.pane != "" {
		outH = 3
	}
	topH := h - outH
	slide, steps := Slide(m.d, m.slide, m.k, sw-4, topH-3, m.pal)
	left := m.pane(fmt.Sprintf("Slide %d/%d · build %d/%d", m.slide+1, len(m.d.Slides), min(m.k, steps), steps), strings.Split(slide, "\n"), sw, topH, l.focus == 1)
	var cmds []string
	group := ""
	selRow := 0
	for i, c := range l.cmds {
		if c.group != group {
			group = c.group
			cmds = append(cmds, mu.Render(strings.ToUpper(group)))
		}
		line := "  " + c.label
		if i == l.sel {
			selRow = len(cmds)
			line = lipgloss.NewStyle().Background(m.pal.HL).Bold(true).Render(ansi.Truncate("▸ "+c.label, cw-4, "…") + strings.Repeat(" ", max(0, cw-4-ansi.StringWidth("▸ "+c.label))))
			cmds = append(cmds, line)
			for _, cl := range strings.Split(c.command, "\n") {
				cmds = append(cmds, st.Foreground(m.pal.Accent2).Render("    "+cl))
			}
			if c.desc != "" {
				cmds = append(cmds, mu.Render("    "+c.desc))
			}
			continue
		}
		cmds = append(cmds, line)
	}
	if len(cmds) == 0 {
		cmds = []string{mu.Render("No runbook: or demos: in this deck's front matter.")}
	}
	view := topH - 2
	if selRow >= view {
		cmds = cmds[selRow-view/2:]
	}
	right := m.pane("Commands · Enter "+map[bool]string{true: "types", false: "runs"}[l.pane != ""]+" · x runs", cmds, cw, topH, l.focus == 0)
	var out []string
	for i := range left {
		out = append(out, left[i]+right[i])
	}
	var shell []string
	title := "Output · ! opens a shell · c clears"
	if l.pane != "" {
		title = "Shell · the tmux pane below · Ctrl+T another"
		shell = []string{mu.Render("Commands are typed into the shell pane under this window.")}
	} else {
		shell = l.out
		if len(shell) > outH-2 {
			shell = shell[len(shell)-(outH-2):]
		}
	}
	if l.asking != nil {
		shell = []string{m.input.view(m.pal, w-6), mu.Render("Enter keeps it · Esc cancels")}
		title = "A value for the command"
	}
	return append(out, m.pane(title, shell, w, outH, l.asking != nil)...)
}
