package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestDiagramFences(t *testing.T) {
	dir := t.TempDir()
	board := "```kanban\ncolumns: To do, Done\nA-1 One · To do\nday 2\nA-1 → Done\nday 3\nA-1 blocked\n```\n"
	write := func(name, src string) *deck.Deck {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(src), 0o644)
		d, err := deck.Parse(p)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	slides := string(Deck(write("talk.md", "---\ntitle: T\n---\n\n## Board\n\n"+board), Options{}).HTML)
	if strings.Count(slides, `class="si-step si-dg-mark"`) != 2 || !strings.Contains(slides, `data-on="0"`) || !strings.Contains(slides, "siDiagram") {
		t.Error("a slide: one build step per day, drawn at its start, the script")
	}
	doc := string(Deck(write("note.md", "---\ntype: article\ntitle: N\n---\n\n"+board), Options{}).HTML)
	if !strings.Contains(doc, `data-dg-nav`) || !strings.Contains(doc, `data-on="2"`) || strings.Contains(doc, `class="si-step si-dg-mark"`) {
		t.Error("a document: drawn at its end, with ← → Play")
	}
	page := string(Deck(write("board.md", "---\ntype: diagram\ntitle: Week 49\norientation: portrait\n---\n\n"+board), Options{}).HTML)
	if !strings.Contains(page, `class="si-ig-frame si-dg-frame" data-orient="portrait"`) || !strings.Contains(page, "<h1>Week 49</h1>") {
		t.Error("a diagram page: a portrait frame with its title")
	}
	off := string(Deck(write("off.md", "---\ntitle: T\n---\n\n## Board\n\n"+strings.Replace(board, "```kanban", "```kanban steps=off", 1)), Options{}).HTML)
	if strings.Contains(off, `class="si-step si-dg-mark"`) {
		t.Error("steps=off: the last state, no build steps")
	}
}
