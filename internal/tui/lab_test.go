package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/felipeg48/simple-ideas/internal/studio"
)

func TestLabRunsCommandsWithPlaceholders(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("SHELL", "/bin/sh")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("---\nrunbook: rb.json\ndemos: demos/\n---\n# Talk\n\n---\n\n## Two\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "rb.json"), []byte(`{"categories":[{"name":"say","entries":[{"label":"Greet","command":"echo hello {{name:world}}"}]}]}`), 0o644)
	os.MkdirAll(filepath.Join(dir, "demos"), 0o755)
	os.WriteFile(filepath.Join(dir, "demos", "d1.md"), []byte("# Demo one\n\n## Start\n\n```bash\necho from-demo\n```\n\n```text\nfrom-demo\n```\n"), 0o644)
	ws, err := studio.OpenWorkspace(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := &studioModel{ws: ws, pal: PaletteFor("studio", ""), expanded: map[string]bool{"": true}, mode: 1, k: 1 << 30, w: 120, h: 36}
	m.refreshTree()
	m.selectPath("talk.md")
	press(m, "4")
	if m.mode != 4 || m.lab == nil || len(m.lab.cmds) != 2 {
		t.Fatalf("lab with the runbook entry and the demo's bash block (not its text block): %v %+v", m.mode, m.lab)
	}
	press(m, "enter") // Greet: asks for {{name}}, prefilled with its default
	if m.lab.asking == nil || m.input.value != "world" {
		t.Fatalf("placeholder prompt: %v %q", m.lab.asking, m.input.value)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(m, cmd)
	press(m, "down")
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(m, cmd)
	out := strings.Join(m.lab.out, "\n")
	if !strings.Contains(out, "hello world") || !strings.Contains(out, "from-demo") || !strings.Contains(out, "[exit 0]") {
		t.Errorf("output:\n%s", out)
	}
	if p := plain(m.render()); !strings.Contains(p, "Commands") || !strings.Contains(p, "DEMO · DEMO ONE") {
		t.Errorf("lab view:\n%s", p)
	}
}

// drain feeds a command's messages back into the model until it is done.
func drain(m *studioModel, cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return
		}
		_, cmd = m.Update(msg)
	}
}
