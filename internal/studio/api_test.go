package studio

import (
	"path/filepath"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func deckFx(t *testing.T) *fx {
	f := newFx(t)
	f.put("main.md", "# Main\n\n---\n\n<!-- include: sections/a.md -->\n")
	f.put("sections/a.md", "## Part\n\n---\n\n# Slide\n\n<!-- layout: nope -->\n\nbody\n")
	return f
}

func TestDecksListsIncludedFiles(t *testing.T) {
	f := deckFx(t)
	r := f.doAny("GET", "decks")
	if len(r) != 1 {
		t.Fatalf("want 1 deck (modules are not decks): %v", r)
	}
	d := r[0].(map[string]any)
	files := d["files"].([]any)
	if d["path"] != "main.md" || len(files) != 2 || files[0] != "main.md" || files[1] != "sections/a.md" {
		t.Fatalf("got %v", d)
	}
}

func TestDecksReportRunbook(t *testing.T) {
	f := newFx(t)
	f.put("a.md", "---\nrunbook: labs/rb.json\n---\n\n# A\n")
	r := f.doAny("GET", "decks")
	if len(r) != 1 || r[0].(map[string]any)["runbook"] != "labs/rb.json" {
		t.Fatalf("got %v", r)
	}
}

func TestCheckReportsProblemsAndSlides(t *testing.T) {
	f := deckFx(t)
	code, m := f.do("GET", "check?deck=main.md", nil)
	if code != 200 {
		t.Fatalf("%d %v", code, m)
	}
	ps := m["problems"].([]any)
	if len(ps) != 1 {
		t.Fatalf("problems: %v", ps)
	}
	p := ps[0].(map[string]any)
	if p["file"] != "sections/a.md" || p["line"].(float64) < 1 {
		t.Fatalf("problem: %v", p)
	}
	sl := m["slides"].([]any)
	if len(sl) < 2 {
		t.Fatalf("slides: %v", sl)
	}
	s := sl[1].(map[string]any)
	if s["file"] != "sections/a.md" || s["index"].(float64) != 2 || s["topic"].(float64) != 1 {
		t.Fatalf("slide[1]: %v", s)
	}
	if code, _ := f.do("GET", "check?deck=../x.md", nil); code != 403 {
		t.Fatalf("outside: %d", code)
	}
	if code, _ := f.do("GET", "check?deck=nope.md", nil); code != 404 {
		t.Fatalf("missing: %d", code)
	}
}

func TestRulesMatchDeckPackage(t *testing.T) {
	f := newFx(t)
	_, m := f.do("GET", "rules", nil)
	if l := m["layouts"].([]any); len(l) != len(deck.Layouts) || l[0] != deck.Layouts[0] {
		t.Fatalf("layouts %v", l)
	}
	dirs := m["directives"].(map[string]any)
	for k, v := range deck.Directives {
		if dirs[k] != v {
			t.Fatalf("directive %s: %v", k, dirs[k])
		}
	}
	if len(m["transitions"].([]any)) != len(deck.Transitions) || len(m["metaKeys"].([]any)) != len(deck.MetaKeys) {
		t.Fatal("transitions/metaKeys")
	}
}

func TestCheckFilesAreWorkspaceRelativeFromParentCwd(t *testing.T) {
	f := deckFx(t)
	t.Chdir(filepath.Dir(f.root))
	_, m := f.do("GET", "check?deck=main.md", nil)
	p := m["problems"].([]any)[0].(map[string]any)
	if p["file"] != "sections/a.md" {
		t.Fatalf("problem file: %v", p["file"])
	}
	if s := m["slides"].([]any)[1].(map[string]any); s["file"] != "sections/a.md" {
		t.Fatalf("slide file: %v", s["file"])
	}
}
