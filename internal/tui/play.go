package tui

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Options configure si play --terminal.
type Options struct {
	Theme string // --theme; else the deck's theme:
	Notes bool   // --notes: the teleprompter instead of the slides
	Addr  string // --notes: the si play server a browser deck is on (default 127.0.0.1:7070)
}

// Dump prints slide i (1-based) of the deck at w × h with every build step
// shown: si play --terminal --dump N, for scripts, docs and screenshots.
func Dump(path string, n, w, h int, opt Options) (string, error) {
	d, err := deck.Parse(path)
	if err != nil {
		return "", err
	}
	if n < 1 || n > len(d.Slides) {
		return "", fmt.Errorf("slide %d: the deck has %d", n, len(d.Slides))
	}
	pal := deckPalette(d, opt)
	body, steps := Slide(d, n-1, 1<<30, w, h-1, pal)
	return body + "\n" + Bar(d, n-1, steps, steps, w, "00:00", pal), nil
}

func deckPalette(d *deck.Deck, opt Options) Palette {
	theme := opt.Theme
	if theme == "" {
		theme = d.Meta.Theme
	}
	return PaletteFor(theme, d.Meta.Mode)
}

// SocketPath is where a terminal presenter listens for its teleprompter.
func SocketPath(deckPath string) string {
	abs, _ := filepath.Abs(deckPath)
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(os.TempDir(), fmt.Sprintf("si-%x.sock", sum[:6]))
}

// Play runs the terminal presenter (or, with opt.Notes, the teleprompter).
func Play(path string, opt Options) error {
	if opt.Notes {
		return playNotes(path, opt)
	}
	d, err := deck.Parse(path)
	if err != nil {
		return err
	}
	m := &presenter{path: path, opt: opt, d: d, pal: deckPalette(d, opt), start: time.Now(), slideStart: time.Now(), proto: PictureProtocol()}
	m.mtimes = fileTimes(d)
	p := tea.NewProgram(m)
	hub, err := listen(SocketPath(path), func(c command) { p.Send(c) })
	if err == nil {
		m.hub = hub
		defer hub.close()
	}
	_, err = p.Run()
	return err
}

type tickMsg time.Time

func tick() tea.Cmd { return tea.Tick(time.Second/2, func(t time.Time) tea.Msg { return tickMsg(t) }) }

type presenter struct {
	path       string
	opt        Options
	d          *deck.Deck
	pal        Palette
	i, k       int
	w, h       int
	start      time.Time
	slideStart time.Time
	help       bool
	overview   bool
	ov         int
	msg        string // a reload problem, shown in the bar
	mtimes     map[string]time.Time
	hub        *hub
	states     map[string][]string // the canvases' state at (i, k)
	prevStates map[string][]string // the state they come from, for the move
	changedAt  time.Time           // when (i, k) last changed: the move's start
	moving     bool                // the slide has canvases or a stage dot: redraw ~10 times a second
	proto      string              // real pictures: kitty or iterm ("" draws half blocks)
	pics       *Pictures           // the pictures of the frame on screen
	places     []placement         // and where they go
}

type drawPicsMsg struct{}

// redraw schedules the pictures after the new frame is on screen; a cleared
// screen first, so no picture of the last slide is left behind.
func (m *presenter) redraw() tea.Cmd {
	if m.proto == "" {
		return nil
	}
	return tea.Sequence(tea.ClearScreen, tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return drawPicsMsg{} }))
}

func (m *presenter) Init() tea.Cmd { return tea.Batch(tick(), frame()) }

func (m *presenter) steps(i int) int {
	_, n := Slide(m.d, i, 0, max(40, m.w), max(10, m.h), m.pal)
	return n
}

func (m *presenter) next() {
	if m.k < m.steps(m.i) {
		m.k++
	} else if m.i < len(m.d.Slides)-1 {
		m.i, m.k = m.i+1, 0
		m.slideStart = time.Now()
	}
	m.broadcast()
}

func (m *presenter) prev() {
	if m.k > 0 {
		m.k--
	} else if m.i > 0 {
		m.i--
		m.k = m.steps(m.i) // back to a slide shows it whole, as in the browser
		m.slideStart = time.Now()
	}
	m.broadcast()
}

func (m *presenter) goTo(i int) {
	m.i, m.k = max(0, min(i, len(m.d.Slides)-1)), 0
	m.slideStart = time.Now()
	m.broadcast()
}

func (m *presenter) broadcast() {
	if m.hub != nil {
		m.hub.send(state{I: m.i, K: m.k, Steps: m.steps(m.i), N: len(m.d.Slides), Start: m.start.Unix()})
	}
}

func (m *presenter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := [3]int{m.i, m.k, m.w*10000 + m.h}
	model, cmd := m.update(msg)
	if before != [3]int{m.i, m.k, m.w*10000 + m.h} || m.states == nil {
		if before[0] != m.i || before[1] != m.k || m.states == nil {
			m.states, m.prevStates = canvasStates(m.d, m.i, m.k)
			m.changedAt = time.Now()
			m.moving = len(m.states) > 0 || strings.Contains(m.d.Slides[m.i].Body, "```stage")
		}
		cmd = tea.Batch(cmd, m.redraw())
	}
	if _, ok := msg.(frameMsg); ok {
		cmd = tea.Batch(cmd, frame())
	}
	return model, cmd
}

type frameMsg struct{}

// frame redraws animated canvases about ten times a second (nothing else moves).
func frame() tea.Cmd { return tea.Tick(time.Second/10, func(time.Time) tea.Msg { return frameMsg{} }) }

func (m *presenter) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case drawPicsMsg:
		if m.pics != nil && m.proto != "" && !m.overview && !m.help {
			return m, tea.Raw(m.pics.draw(m.places))
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		if changed(m.mtimes, fileTimes(m.d)) {
			if d, err := deck.Parse(m.path); err == nil {
				m.d, m.pal, m.msg = d, deckPalette(d, m.opt), ""
				m.i = min(m.i, len(d.Slides)-1)
				m.k = min(m.k, m.steps(m.i))
				m.broadcast()
			} else {
				m.msg = err.Error()
			}
			m.mtimes = fileTimes(m.d)
		}
		return m, tick()
	case command:
		switch msg.Cmd {
		case "next":
			m.next()
		case "prev":
			m.prev()
		case "go":
			m.goTo(msg.I)
		case "hello":
			m.broadcast()
		}
	case tea.KeyPressMsg:
		key := msg.String()
		if m.overview {
			switch key {
			case "up", "k":
				m.ov = max(0, m.ov-1)
			case "down", "j":
				m.ov = min(len(m.d.Slides)-1, m.ov+1)
			case "enter", "space":
				m.overview = false
				m.goTo(m.ov)
			case "o", "esc", "q":
				m.overview = false
			}
			return m, nil
		}
		if m.help && key != "ctrl+c" {
			m.help = false
			return m, nil
		}
		switch key {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "right", "space", "l", "n", "pgdown", "enter", "j", "down":
			m.next()
		case "left", "h", "p", "pgup", "backspace", "k", "up":
			m.prev()
		case "home", "g":
			m.goTo(0)
		case "end", "G":
			m.goTo(len(m.d.Slides) - 1)
		case "o":
			m.overview, m.ov = true, m.i
		case "?":
			m.help = true
		case "t":
			m.start, m.slideStart = time.Now(), time.Now()
		}
	}
	return m, nil
}

func (m *presenter) View() tea.View {
	var content string
	switch {
	case m.w == 0:
		content = ""
	case m.overview:
		content = m.overviewView()
	case m.help:
		content = helpView(m.pal, m.w, m.h)
	default:
		var pics *Pictures
		if m.proto != "" {
			pics = &Pictures{Proto: m.proto}
		}
		opts := &SlideOpts{Pics: pics, T: time.Since(m.start).Seconds(), States: m.states, Prev: m.prevStates,
			P: min(1, time.Since(m.changedAt).Seconds()/0.9)}
		if !m.moving {
			opts.T = 0
		}
		body, steps := Slide(m.d, m.i, m.k, m.w, m.h-1, m.pal, opts)
		if pics != nil {
			body, m.places = pics.placements(body)
			m.pics = pics
		}
		timer := clock(time.Since(m.start))
		bar := Bar(m.d, m.i, m.k, steps, m.w, timer, m.pal)
		if m.msg != "" {
			bar = lipgloss.NewStyle().Foreground(m.pal.Crit).Render(" " + ansi.Truncate(m.msg, m.w-2, "…"))
		}
		content = body + "\n" + bar
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = m.pal.BG
	v.ForegroundColor = m.pal.FG
	v.WindowTitle = m.d.Title()
	return v
}

func (m *presenter) overviewView() string {
	st := lipgloss.NewStyle().Foreground(m.pal.FG)
	lines := []string{"", "  " + st.Bold(true).Render(m.d.Title()) + st.Foreground(m.pal.Muted).Render("  ·  overview: ↑ ↓ choose, Enter open, o close"), ""}
	rows := m.h - 4
	first := max(0, min(m.ov-rows/2, len(m.d.Slides)-rows))
	for i := first; i < len(m.d.Slides) && i < first+rows; i++ {
		s := m.d.Slides[i]
		t := stripMarks(s.Title)
		if t == "" {
			t = "(no heading)"
		}
		line := fmt.Sprintf("  %3d  %s", i+1, t)
		if s.Minutes > 0 {
			line += st.Foreground(m.pal.Muted).Render(fmt.Sprintf("   %.1f min", s.Minutes))
		}
		if i == m.ov {
			line = st.Background(m.pal.HL).Bold(true).Render(ansi.Truncate(line, m.w-1, "…") + strings.Repeat(" ", max(0, m.w-1-ansi.StringWidth(line))))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func helpView(pal Palette, w, h int) string {
	st := lipgloss.NewStyle().Foreground(pal.FG)
	k := st.Foreground(pal.Accent).Bold(true)
	rows := [][2]string{
		{"→ space l n", "next build or slide"}, {"← h p", "back"}, {"g / G", "first / last slide"},
		{"o", "overview"}, {"t", "restart the timer"}, {"q esc", "quit"},
		{"", ""}, {"si play deck.md --terminal --notes", "the teleprompter, in a second terminal"},
	}
	var b strings.Builder
	b.WriteString("\n  " + st.Bold(true).Render("Keys") + "\n\n")
	for _, r := range rows {
		b.WriteString("  " + k.Render(fmt.Sprintf("%-36s", r[0])) + st.Render(r[1]) + "\n")
	}
	b.WriteString("\n  " + st.Foreground(pal.Muted).Render("any key closes this"))
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, b.String())
}

func clock(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// fileTimes are the modification times of the deck and the files it includes.
func fileTimes(d *deck.Deck) map[string]time.Time {
	out := map[string]time.Time{}
	add := func(p string) {
		if fi, err := os.Stat(p); err == nil {
			out[p] = fi.ModTime()
		}
	}
	add(d.Path)
	for _, s := range d.Slides {
		if s.File != "" {
			add(s.File)
		}
	}
	return out
}

func changed(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return true
	}
	for k, v := range a {
		if !b[k].Equal(v) {
			return true
		}
	}
	return false
}
