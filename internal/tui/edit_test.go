package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalEditsAMindMap(t *testing.T) {
	m := studioFx(t)
	m.selectPath("mindmap/warehouse.md")
	m.loadPreview()
	m.focus = 1
	file := filepath.Join(m.ws.Root, "mindmap", "warehouse.md")
	press(m, "down") // the first topic: Inbound
	press(m, "enter")
	m.input.value = "Receiving dock"
	press(m, "enter")
	press(m, "c") // a child under it, selected
	raw, _ := os.ReadFile(file)
	if !strings.Contains(string(raw), "- Receiving dock\n  - Receive\n  - Put away\n  - New topic\n") {
		t.Fatalf("rename + add child:\n%s", raw)
	}
	press(m, "x") // delete New topic again
	m.mmSel = 1   // Receiving dock
	press(m, "]") // moves below Picking
	raw, _ = os.ReadFile(file)
	if strings.Contains(string(raw), "New topic") || strings.Index(string(raw), "- Picking") > strings.Index(string(raw), "- Receiving dock") {
		t.Fatalf("delete + move down:\n%s", raw)
	}
	if v := strings.Join(m.docLines(100, 40), "\n"); !strings.Contains(v, "enter rename") {
		t.Errorf("the focused tree shows its keys:\n%s", v)
	}
}

func TestTerminalEditsACodeWindow(t *testing.T) {
	m := studioFx(t)
	m.selectPath("code/retry.md")
	m.loadPreview()
	var style, numbers *item
	cmds := m.commands()
	for i := range cmds {
		if strings.HasPrefix(cmds[i].label, "Window retry.go: terminal") && style == nil {
			style = &cmds[i]
		}
		if strings.HasPrefix(cmds[i].label, "Window retry.go: line numbers") && numbers == nil {
			numbers = &cmds[i]
		}
	}
	if style == nil || numbers == nil {
		t.Fatal("the palette offers the code window edits")
	}
	style.run(m)
	numbers.run(m)
	raw, _ := os.ReadFile(filepath.Join(m.ws.Root, "code", "retry.md"))
	first := strings.SplitN(string(raw[strings.Index(string(raw), "```go"):]), "\n", 2)[0]
	if !strings.Contains(first, "window=terminal") || strings.Contains(first, " numbers") {
		t.Errorf("window style and numbers written on every step's fence: %s", first)
	}
}
