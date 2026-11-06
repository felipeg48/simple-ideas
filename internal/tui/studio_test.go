package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/felipeg48/simple-ideas/internal/studio"
)

func studioFx(t *testing.T) *studioModel {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("cp", "-R", "../../examples/.", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	os.Remove(filepath.Join(dir, "embed.go"))
	ws, err := studio.OpenWorkspace(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := &studioModel{ws: ws, pal: PaletteFor("studio", ""), expanded: map[string]bool{"": true}, mode: 1, k: 1 << 30, w: 140, h: 40}
	m.refreshTree()
	m.selectFirstDeck()
	m.loadPreview()
	m.problems, m.checked = ws.Check()
	return m
}

func press(m *studioModel, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		default:
			msg = tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}
		}
		m.Update(msg)
	}
}

func TestStudioWriteView(t *testing.T) {
	m := studioFx(t)
	out := m.render()
	if p := os.Getenv("SI_SHOT"); p != "" {
		os.WriteFile(p, []byte(out), 0o644)
	}
	pl := plain(out)
	for _, want := range []string{"1 Write", "Files", "showcase.md", "Slide 1/", "Problems", "e edit"} {
		if !strings.Contains(pl, want) {
			t.Errorf("lacks %q:\n%s", want, pl)
		}
	}
	if n := len(strings.Split(out, "\n")); n != m.h {
		t.Errorf("%d lines for %d rows", n, m.h)
	}
}

func TestStudioPaletteStoryboardInspector(t *testing.T) {
	m := studioFx(t)
	press(m, ":")
	for _, r := range "retry" {
		press(m, string(r))
	}
	if m.overlay != "palette" || len(m.list) == 0 || !strings.Contains(m.list[0].label+m.list[0].hint, "retry") {
		t.Fatalf("palette finds retry: %v", m.list)
	}
	press(m, "enter")
	if !strings.Contains(m.file, "retry") {
		t.Errorf("enter opens it: %s", m.file)
	}
	m.selectPath("showcase.md")
	press(m, "s")
	if m.overlay != "storyboard" || len(m.list) < 10 {
		t.Fatalf("storyboard: %s %d", m.overlay, len(m.list))
	}
	press(m, "down", "down", "enter")
	if m.slide != 2 || m.overlay != "" {
		t.Errorf("storyboard goes to slide 3: %d", m.slide)
	}
	press(m, "i", "enter") // layout
	if m.overlay != "prompt" {
		t.Fatalf("inspector prompt: %s", m.overlay)
	}
	for _, r := range "center" {
		press(m, string(r))
	}
	press(m, "enter")
	if m.d.Slides[2].Layout != "center" {
		raw, _ := os.ReadFile(m.abs("showcase.md"))
		t.Errorf("layout written: %q\n%s", m.d.Slides[2].Layout, raw)
	}
}

func TestStudioNewStarter(t *testing.T) {
	m := studioFx(t)
	m.selectPath("site/index.md")
	press(m, "n", "down", "enter") // Article
	for _, r := range "Queue depth" {
		press(m, string(r))
	}
	press(m, "enter")
	if _, err := os.Stat(m.abs("site/queue-depth.md")); err != nil {
		t.Fatalf("starter written: %v (%s)", err, m.status)
	}
	if !strings.Contains(m.status, "_site.md") || m.file != "site/queue-depth.md" {
		t.Errorf("joins the site and opens it: %s / %s", m.status, m.file)
	}
}
