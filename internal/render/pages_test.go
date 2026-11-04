package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestPages(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"talk.md":    "# The talk\n",
		"lab-01.md":  "---\ntype: lab\n---\n# First lab\n\n## A\n",
		"notes.html": "<html><head><title>Venue &amp; Wi-Fi</title></head></html>",
		"talk.html":  "<title>stale build</title>",
		"index.html": "<title>old index</title>",
		"other.txt":  "x",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var decks []*deck.Deck
	for _, f := range []string{"talk.md", "lab-01.md"} {
		d, err := deck.Parse(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		decks = append(decks, d)
	}
	cfg := &config.Config{Links: []config.Link{{Label: "Team wiki", URL: "https://wiki.example.com"}}}
	got := Pages(decks, true, true, dir, cfg)
	want := []Page{
		{"index.html", "Index", "index"},
		{"talk.html", "The talk", "deck"},
		{"lab-01.html", "First lab", "lab"},
		{"pdfs/talk.pdf", "The talk (PDF)", "pdf"},
		{"notes.html", "Venue & Wi-Fi", "page"},
		{"https://wiki.example.com", "Team wiki", "link"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("page %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
