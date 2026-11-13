package main

import (
	"strings"
	"testing"
)

func TestTimelinePages(t *testing.T) {
	era := func(name string, n int) string {
		s := "## " + name + "\n\n"
		for i := 0; i < n; i++ {
			s += "### 19" + string(rune('0'+i)) + "0 · M\nText.\n\n"
		}
		return s
	}
	src := "---\ntype: timeline\ntitle: T\nclosing: The end.\nrequire: 20 dates\n---\n\nIntro.\n\n" +
		era("A", 5) + era("B", 5) + era("C", 6) + "## Fuentes\n\n- Book.\n"
	pages := timelinePages(src, 12)
	if len(pages) != 2 {
		t.Fatalf("want 2 pages, got %d", len(pages))
	}
	if !strings.Contains(pages[0], "Intro.") || !strings.Contains(pages[0], "## A") || !strings.Contains(pages[0], "## B") || strings.Contains(pages[0], "## C") {
		t.Fatalf("page 1: %s", pages[0])
	}
	for i, p := range pages {
		if strings.Contains(p, "require:") || !strings.Contains(p, "title: T") {
			t.Fatalf("page %d front matter: %s", i+1, p)
		}
	}
	if strings.Contains(pages[0], "closing:") || strings.Contains(pages[0], "Fuentes") {
		t.Fatal("closing and sources belong on the last page")
	}
	if !strings.Contains(pages[1], "closing: The end.") || !strings.Contains(pages[1], "## Fuentes") || !strings.Contains(pages[1], "## C") {
		t.Fatalf("last page: %s", pages[1])
	}
	if got := timelinePages(era("A", 3), 12); len(got) != 1 {
		t.Fatalf("a short timeline stays one page, got %d", len(got))
	}
}
