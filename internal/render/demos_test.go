package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestDemos(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("demos/c1-demo.md", "# Session 1 demo\n\n```bash\ndocker compose up -d\n```\n\nYou should see:\n\n```text\nHealthy\n```\n\n> Say: \"one broker\"\n")
	write("demos/c2-demo.md", "no heading\n\n```sql\nSELECT 1;\n```\n")
	write("demos/README.md", "# Index\n")
	write("demos/_draft.md", "# Draft\n")
	write("d.md", "---\ndemos: demos/\n---\n# D\n")
	d, err := deck.Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{Terminal: true})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	page := string(res.HTML)
	for _, want := range []string{"window.SI_DEMOS=", "Session 1 demo", `"title":"c2-demo"`, `si-dm-code si-dm-run`, "compose up"} {
		if !strings.Contains(page, want) {
			t.Errorf("play page lacks %q", want)
		}
	}
	if strings.Contains(page, `"title":"Index"`) || strings.Contains(page, "Draft") {
		t.Error("README.md and _-prefixed files are not demos")
	}
	if strings.Count(page, `si-dm-code si-dm-run`) != 2 || strings.Count(page, `si-dm-code\"`) != 1 {
		t.Errorf("want 2 runnable blocks (bash, sql) and 1 plain output block")
	}
	if strings.Contains(string(Deck(d, Options{}).HTML), "SI_DEMOS") {
		t.Error("a built deck must not carry the demos")
	}
	write("e.md", "---\ndemos: nope/, demos/c1-demo.md\n---\n# E\n")
	e, _ := deck.Parse(filepath.Join(dir, "e.md"))
	if r := Deck(e, Options{}); len(r.Problems) != 1 || !strings.Contains(r.Problems[0].Msg, "demos: cannot read nope/") {
		t.Errorf("a missing demos path is a check problem: %v", r.Problems)
	}
}
