package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestBook(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"_nav.md":     "---\ntype: book\ntitle: The Reference\nsubtitle: All of it.\nedition: 2nd edition\n---\n- [Preface](preface.md)\n- Part I\n  - [Flows](flows.md)\n  - [Queues](queues.md)\n- Appendices\n  - [Glossary](glossary.md)\n",
		"preface.md":  "# Preface\n\nRead [](flows.md) first.\n",
		"flows.md":    "# Flows\n\n## Exchanges\n\nAn ((exchange)) routes. See [](#tbl-routes) and [](queues.md#quorum).\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nTable: Routes\n\n## Bindings\n\n(((binding, topic)))Text. A [](#nowhere) link.\n",
		"queues.md":   "# Queues\n\n## Quorum\n\nA ((quorum queue)) and an ((exchange)).\n",
		"glossary.md": "# Glossary\n\nWords.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, probs, _ := RenderGuide(g, Options{})
	if len(probs) != 1 || !strings.Contains(probs[0].Msg, "cross-reference #nowhere") {
		t.Errorf("want only the broken cross-reference: %v", probs)
	}
	byRel := map[string]string{}
	for _, p := range pages {
		byRel[p.Rel] = string(p.Result.HTML)
	}
	for _, rel := range []string{"index.html", "contents.html", "book-index.html", "book.html"} {
		if byRel[rel] == "" {
			t.Fatalf("generated page %s missing", rel)
		}
	}
	flows := byRel["flows.html"]
	for _, want := range []string{`<span class="si-book-chap">Chapter 1</span>`, `<span class="si-sec-num">1.1</span>Exchanges`, `<span class="si-sec-num">1.2</span>Bindings`,
		`<b class="si-fig-num">Table 1.1</b> · Routes`, `class="si-xref">Table 1.1</a>`, `class="si-xref">Section 2.1</a>`,
		`<span class="si-term" id="ix-1">exchange</span>`, `<span class="si-ix" id="ix-2"></span>`, `data-si-book="chapter"`} {
		if !strings.Contains(flows, want) {
			t.Errorf("flows.html lacks %q", want)
		}
	}
	if !strings.Contains(byRel["preface.html"], `class="si-xref">Chapter 1</a>`) || strings.Contains(byRel["preface.html"], `<span class="si-book-chap">`) {
		t.Error("the preface is front matter (no number) and links to Chapter 1")
	}
	if !strings.Contains(byRel["glossary.html"], "Appendix A") {
		t.Error("pages under Appendices are lettered")
	}
	ix := byRel["book-index.html"]
	for _, want := range []string{"<span>exchange</span>", `href="flows.html#ix-1">1.1</a>`, `href="queues.html#ix-`, ">2.1</a>", "<span>topic</span>", "<span>quorum queue</span>"} {
		if !strings.Contains(ix, want) {
			t.Errorf("index lacks %q", want)
		}
	}
	whole := byRel["book.html"]
	for _, want := range []string{`id="c-flows-tbl-routes"`, `href="#c-queues-quorum"`, `id="si-book-contents"`, `id="si-book-index"`, "The Reference"} {
		if !strings.Contains(whole, want) {
			t.Errorf("book.html lacks %q", want)
		}
	}
	cover := byRel["index.html"]
	if !strings.Contains(cover, "Book · 2nd edition") || !strings.Contains(cover, "All of it.") || !strings.Contains(cover, "Start reading: Flows") {
		t.Error("cover: kind, subtitle and the first chapter link")
	}
}
