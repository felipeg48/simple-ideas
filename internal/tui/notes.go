package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// The terminal teleprompter (si play deck.md --terminal --notes): the
// speaker's script for the slide on screen, in a second terminal that follows
// the presenter over its socket. Without a presenter it moves on its own.

type stateMsg struct {
	s    state
	live bool
}

type notesTick time.Time

func notesTicker() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg { return notesTick(t) })
}

type teleprompter struct {
	path       string
	opt        Options
	d          *deck.Deck
	pal        Palette
	i, k       int
	w, h       int
	live       bool
	start      time.Time
	slideStart time.Time
	points     bool    // Tab: the speaker's Points: instead of the script
	auto       bool    // auto-scroll
	speed      float64 // lines a second
	offset     float64
	mtimes     map[string]time.Time
	f          *follower
}

func playNotes(path string, opt Options) error {
	d, err := deck.Parse(path)
	if err != nil {
		return err
	}
	m := &teleprompter{path: path, opt: opt, d: d, pal: deckPalette(d, opt), start: time.Now(), slideStart: time.Now(), auto: true, speed: 0.6}
	m.mtimes = fileTimes(d)
	p := tea.NewProgram(m)
	m.f = follow(SocketPath(path), func(s state, live bool) { p.Send(stateMsg{s, live}) })
	_, err = p.Run()
	return err
}

func (m *teleprompter) Init() tea.Cmd { return notesTicker() }

func (m *teleprompter) move(i, k int) {
	i = max(0, min(i, len(m.d.Slides)-1))
	if i != m.i {
		m.slideStart = time.Now()
		m.offset = 0
	}
	m.i, m.k = i, k
}

func (m *teleprompter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case stateMsg:
		m.live = msg.live
		if msg.live {
			if msg.s.Start > 0 {
				m.start = time.Unix(msg.s.Start, 0)
			}
			m.move(msg.s.I, msg.s.K)
		}
	case notesTick:
		if m.auto {
			m.offset += m.speed / 10
		}
		if changed(m.mtimes, fileTimes(m.d)) {
			if d, err := deck.Parse(m.path); err == nil {
				m.d, m.pal = d, deckPalette(d, m.opt)
				m.i = min(m.i, len(d.Slides)-1)
			}
			m.mtimes = fileTimes(m.d)
		}
		return m, notesTicker()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "right", "space", "l", "n", "pgdown", "enter":
			if !m.f.send(command{Cmd: "next"}) {
				m.localStep(1)
			}
		case "left", "h", "p", "pgup", "backspace":
			if !m.f.send(command{Cmd: "prev"}) {
				m.localStep(-1)
			}
		case "up", "k":
			m.offset = max(0, m.offset-1)
		case "down", "j":
			m.offset++
		case "+", "=":
			m.speed = min(6, m.speed+0.2)
		case "-", "_":
			m.speed = max(0.2, m.speed-0.2)
		case "a":
			m.auto = !m.auto
		case "tab":
			m.points, m.offset = !m.points, 0
		case "r":
			m.start, m.slideStart = time.Now(), time.Now()
		case "home", "g":
			m.f.send(command{Cmd: "go", I: 0})
		}
	}
	return m, nil
}

// localStep moves through builds and slides without a presenter.
func (m *teleprompter) localStep(dir int) {
	_, steps := Slide(m.d, m.i, 0, 80, 24, m.pal)
	switch {
	case dir > 0 && m.k < steps:
		m.k++
	case dir > 0:
		m.move(m.i+1, 0)
	case m.k > 0:
		m.k--
	default:
		m.move(m.i-1, 0)
	}
}

var clickRE = regexp.MustCompile(`(?i)\[click\]`)

func (m *teleprompter) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.BackgroundColor = m.pal.BG
	v.ForegroundColor = m.pal.FG
	v.WindowTitle = "Teleprompter · " + m.d.Title()
	return v
}

func (m *teleprompter) render() string {
	if m.w < 40 || m.h < 10 {
		return "make the terminal larger"
	}
	st := lipgloss.NewStyle().Foreground(m.pal.FG)
	mu := st.Foreground(m.pal.Muted)
	s := m.d.Slides[m.i]

	// the top line: timers, pace, clock, connection
	total := time.Since(m.start)
	planned, before := 0.0, 0.0
	for j, sl := range m.d.Slides {
		planned += sl.Minutes
		if j < m.i {
			before += sl.Minutes
		}
	}
	pace := ""
	if planned > 0 {
		diff := total.Minutes() - before
		switch {
		case diff > s.Minutes+0.25:
			pace = lipgloss.NewStyle().Background(m.pal.Warn).Foreground(m.pal.BG).Bold(true).Render(" " + clock(time.Duration((diff-s.Minutes)*float64(time.Minute))) + " behind ")
		case diff < -0.5:
			pace = st.Foreground(m.pal.OK).Render(clock(time.Duration(-diff*float64(time.Minute))) + " ahead")
		default:
			pace = st.Foreground(m.pal.OK).Render("on time")
		}
	}
	link := st.Foreground(m.pal.OK).Render("● following the presenter")
	if !m.live {
		link = mu.Render("○ no presenter: start  si play " + shortPath(m.path) + " --terminal")
	}
	top := " " + st.Bold(true).Render("slide "+clock(time.Since(m.slideStart))) + mu.Render(fmt.Sprintf("/%.1f", s.Minutes)) +
		"   " + st.Bold(true).Render("total "+clock(total)) + mu.Render(fmt.Sprintf("/%.0f", planned)) +
		"   " + pace + "   " + mu.Render("clock "+time.Now().Format("15:04")) + "   " + link

	// left: now and next
	lw := min(40, m.w/3)
	r := &textRenderer{pal: m.pal, dir: s.Dir, w: lw - 4, maxH: 6, k: m.k}
	var now []string
	now = append(now, st.Bold(true).Render(ansi.Truncate(stripMarks(s.Title), lw-4, "…")))
	for _, b := range r.body(s.Body) {
		for _, l := range b.lines {
			if len(now) < 9 {
				now = append(now, ansi.Truncate(l, lw-4, "…"))
			}
		}
	}
	left := r.box(fmt.Sprintf("NOW %d/%d", m.i+1, len(m.d.Slides)), now, lw-4, m.pal.Line)
	if m.i+1 < len(m.d.Slides) {
		nx := m.d.Slides[m.i+1]
		left = append(left, r.box("NEXT", []string{ansi.Truncate(orName(stripMarks(nx.Title), "(no heading)"), lw-4, "…")}, lw-4, m.pal.Line)...)
	}

	// right: the script (or the points), [CLICK] cues as badges
	rw := m.w - lw - 4
	src, label := s.Notes, "Teleprompter"
	if m.points {
		src, label = s.Points, "Speaker notes"
	}
	cue := 0
	src = clickRE.ReplaceAllStringFunc(src, func(string) string {
		cue++
		return fmt.Sprintf("\uE000CUE%d\uE001", cue) // private-use marks survive markdown (NUL does not)
	})
	nr := &textRenderer{pal: m.pal, w: rw, k: 1 << 30}
	var script []string
	for bi, b := range nr.body(src) {
		if bi > 0 {
			script = append(script, "")
		}
		script = append(script, b.lines...)
	}
	cueRE := regexp.MustCompile("\uE000CUE(\\d+)\uE001")
	for i, l := range script {
		script[i] = cueRE.ReplaceAllStringFunc(l, func(c string) string {
			n := 0
			fmt.Sscanf(cueRE.FindStringSubmatch(c)[1], "%d", &n)
			switch {
			case n <= m.k:
				return mu.Render("✓ click")
			case n == m.k+1:
				return lipgloss.NewStyle().Background(m.pal.Accent).Foreground(m.pal.BG).Bold(true).Render(" ▶ CLICK ")
			}
			return st.Foreground(m.pal.Accent).Render("▶ click")
		})
	}
	if len(script) == 0 {
		script = []string{mu.Render("No " + strings.ToLower(map[bool]string{false: "Notes:", true: "Points:"}[m.points]) + " on this slide.")}
	}
	view := m.h - 4
	maxOff := max(0, len(script)-view)
	m.offset = min(m.offset, float64(maxOff))
	off := int(m.offset)
	right := []string{st.Foreground(m.pal.Accent).Bold(true).Render(label) + mu.Render("   Tab: "+map[bool]string{false: "speaker notes", true: "teleprompter"}[m.points])}
	right = append(right, script[off:min(len(script), off+view)]...)

	lines := []string{top, mu.Render(" " + strings.Repeat("─", m.w-2))}
	for y := 0; y < m.h-3; y++ {
		l, rr := "", ""
		if y < len(left) {
			l = left[y]
		}
		if y < len(right) {
			rr = right[y]
		}
		lines = append(lines, " "+l+strings.Repeat(" ", max(0, lw-ansi.StringWidth(l)))+"  "+rr)
	}
	scroll := "auto-scroll off"
	if m.auto {
		scroll = fmt.Sprintf("auto-scroll %.1f lines/s", m.speed)
	}
	lines = append(lines, mu.Render(" space next · ← back · ↑↓ scroll · a "+scroll+" · +/- speed · Tab notes · r timers · q quit"))
	return strings.Join(lines[:min(len(lines), m.h)], "\n")
}

func shortPath(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
