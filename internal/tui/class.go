package tui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The live class view in the terminal (si play course/ --terminal --class):
// the class view's data (/_si/class/data, internal/serve/class.go) as text,
// fetched again every two seconds: learners by lessons (✓ done, a quiz's best
// score, ✗ a failing check, · started), who is stuck, and the links to share.

// ClassData is the part of the class view's JSON the terminal shows.
type ClassData struct {
	Course struct {
		Title   string `json:"title"`
		Lessons []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Kind   string `json:"kind"`
			Module string `json:"module"`
		} `json:"lessons"`
	} `json:"course"`
	Learners []struct {
		Label     string    `json:"label"`
		Done      int       `json:"done"`
		Completed bool      `json:"completed"`
		Last      time.Time `json:"last"`
		Cells     map[string]*struct {
			Done bool `json:"done"`
			Quiz *struct {
				Best int  `json:"best"`
				Pass bool `json:"pass"`
			} `json:"quiz"`
			Checks []struct {
				Pass bool `json:"pass"`
			} `json:"checks"`
			Demo int `json:"demo"`
		} `json:"cells"`
	} `json:"learners"`
	Stuck []struct {
		Label  string    `json:"label"`
		Title  string    `json:"title"`
		Reason string    `json:"reason"`
		Since  time.Time `json:"since"`
	} `json:"stuck"`
	StuckAfter int       `json:"stuckAfter"`
	Events     int       `json:"events"`
	Now        time.Time `json:"now"`
}

// ClassText draws the class view in w columns: the header lines (links), the
// counts, the grid of learners by lessons and its key, and who is stuck.
func ClassText(v ClassData, header []string, w int, pal Palette) string {
	fg := lipgloss.NewStyle().Foreground(pal.FG)
	mu := lipgloss.NewStyle().Foreground(pal.Muted)
	ok := lipgloss.NewStyle().Foreground(pal.OK)
	bad := lipgloss.NewStyle().Foreground(pal.Crit)
	acc := lipgloss.NewStyle().Foreground(pal.Accent).Bold(true)
	var out []string
	out = append(out, acc.Render("Class · "+v.Course.Title))
	for _, h := range header {
		out = append(out, mu.Render(ansi.Truncate(h, w, "…")))
	}
	done := 0
	for _, l := range v.Learners {
		if l.Completed {
			done++
		}
	}
	counts := fmt.Sprintf("%d learners · %d completed · %d stuck · %d events", len(v.Learners), done, len(v.Stuck), v.Events)
	out = append(out, "", fg.Bold(true).Render(counts), "")

	// the grid: a name column, then one 4-wide column per lesson that fits
	nameW := 12
	for _, l := range v.Learners {
		nameW = max(nameW, min(22, ansi.StringWidth(l.Label)+1))
	}
	const cw = 4
	fit := max(1, (w-nameW-8)/cw)
	lessons := v.Course.Lessons
	more := 0
	if len(lessons) > fit {
		more, lessons = len(lessons)-fit, lessons[:fit]
	}
	head := cpad("Learner", nameW)
	for i := range lessons {
		head += cpadLeft(strconv.Itoa(i+1), cw-1) + " "
	}
	head += " done"
	out = append(out, mu.Render(head))
	if len(v.Learners) == 0 {
		out = append(out, mu.Render("Waiting for the first learner."))
	}
	for _, l := range v.Learners {
		row := fg.Render(cpad(ansi.Truncate(l.Label, nameW-1, "…"), nameW))
		for _, ls := range lessons {
			c := l.Cells[ls.ID]
			cell, sty := "", mu
			switch {
			case c == nil:
				cell = ""
			case c.Quiz != nil:
				cell, sty = strconv.Itoa(c.Quiz.Best), bad
				if c.Quiz.Pass {
					sty = ok
				}
			case failing(c.Checks):
				cell, sty = "✗", bad
			case c.Done:
				cell, sty = "✓", ok
			default:
				cell = "·"
			}
			row += sty.Render(cpadLeft(cell, cw-1)) + " "
		}
		row += mu.Render(fmt.Sprintf(" %d/%d", l.Done, len(v.Course.Lessons)))
		if l.Completed {
			row += ok.Render(" ✓ course")
		}
		out = append(out, row)
	}
	out = append(out, "")
	for i, ls := range lessons {
		out = append(out, mu.Render(ansi.Truncate(fmt.Sprintf("%3d  %s (%s) · %s", i+1, ls.Title, ls.Kind, ls.Module), w, "…")))
	}
	if more > 0 {
		out = append(out, mu.Render(fmt.Sprintf("     +%d more lessons: a wider terminal shows them", more)))
	}
	out = append(out, "", fg.Bold(true).Render(fmt.Sprintf("Who's stuck (no progress for %d min, or failing)", v.StuckAfter)))
	if len(v.Stuck) == 0 {
		out = append(out, mu.Render("Nobody is stuck right now."))
	}
	for _, s := range v.Stuck {
		ago := v.Now.Sub(s.Since).Round(time.Minute)
		out = append(out, bad.Render("● ")+fg.Render(ansi.Truncate(fmt.Sprintf("%s · %s · %s · %s", s.Label, s.Title, s.Reason, ago), w-2, "…")))
	}
	out = append(out, "", mu.Render("q quits (the class stops) · ✓ done · 85 a quiz's best score · ✗ a failing check · · started"))
	return strings.Join(out, "\n")
}

// cpad and cpadLeft pad to n terminal columns (✓ is one column, three bytes).
func cpad(s string, n int) string { return s + strings.Repeat(" ", max(0, n-ansi.StringWidth(s))) }

func cpadLeft(s string, n int) string { return strings.Repeat(" ", max(0, n-ansi.StringWidth(s))) + s }

func failing(cs []struct {
	Pass bool `json:"pass"`
}) bool {
	for _, c := range cs {
		if !c.Pass {
			return true
		}
	}
	return false
}

type classTick time.Time

type classModel struct {
	fetch  func() ([]byte, error)
	header []string
	pal    Palette
	w, h   int
	v      ClassData
	err    error
}

func (m *classModel) load() {
	b, err := m.fetch()
	if err == nil {
		var v ClassData
		if err = json.Unmarshal(b, &v); err == nil {
			m.v = v
		}
	}
	m.err = err
}

func classTicker() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return classTick(t) })
}

func (m *classModel) Init() tea.Cmd { return classTicker() }

func (m *classModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case classTick:
		m.load()
		return m, classTicker()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.load()
		}
	}
	return m, nil
}

func (m *classModel) View() tea.View {
	s := ClassText(m.v, m.header, max(40, m.w), m.pal)
	if m.err != nil {
		s += "\n" + lipgloss.NewStyle().Foreground(m.pal.Crit).Render("cannot read the class: "+m.err.Error())
	}
	lines := strings.Split(s, "\n")
	if m.h > 0 && len(lines) > m.h {
		lines = lines[:m.h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.BackgroundColor = m.pal.BG
	v.ForegroundColor = m.pal.FG
	v.WindowTitle = "Class · " + m.v.Course.Title
	return v
}

// Class runs the live class view until q: fetch returns the class view's
// JSON, header the lines above it (the links to share).
func Class(fetch func() ([]byte, error), header []string, theme string) error {
	m := &classModel{fetch: fetch, header: header, pal: PaletteFor(theme, "")}
	m.load()
	_, err := tea.NewProgram(m).Run()
	return err
}
