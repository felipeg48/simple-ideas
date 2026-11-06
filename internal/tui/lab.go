package tui

import tea "charm.land/bubbletea/v2"

// Lab mode arrives in 1.9: the slide, the deck's runbook and demos, and a shell.

type labState struct{}
type labOutput struct{}

func (l *labState) poll()           {}
func (l *labState) add(o labOutput) {}

func (m *studioModel) openLab() tea.Cmd {
	m.status = "Lab arrives in si 1.9"
	return nil
}

func (m *studioModel) labKey(msg tea.KeyPressMsg) (tea.Cmd, bool) { return nil, false }

func (m *studioModel) labView(w, h int) []string { return m.writeView(w, h) }
