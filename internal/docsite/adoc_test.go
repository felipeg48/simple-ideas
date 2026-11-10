package docsite

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// Every Markdown example of the rules and the manual, written as AsciiDoc by
// the site, renders the same slides as the Markdown.
func TestAdocExamplesRenderTheSame(t *testing.T) {
	dir := t.TempDir()
	body := regexp.MustCompile(`(?s)<main.*</main>|<section class="si-slide.*`)
	build := func(name, src string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(src), 0o644)
		d, err := deck.Parse(p)
		if err != nil {
			return "error: " + err.Error()
		}
		return regexp.MustCompile(` data-src="[^"]*"`).ReplaceAllString(body.FindString(string(render.Deck(d, render.Options{}).HTML)), "")
	}
	n := 0
	srcs := []string{docs.Rules, docs.Manual}
	fs.WalkDir(docs.Site, "site", func(p string, e fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(p, ".md") {
			raw, _ := fs.ReadFile(docs.Site, p)
			srcs = append(srcs, string(raw))
		}
		return nil
	})
	for _, src := range srcs {
		for _, ex := range mdExamples(src) {
			n++
			md, ad := build("e.md", ex), strings.ReplaceAll(build("e.adoc", deck.MarkdownToAdoc(ex)), `"e.adoc"`, `"e.md"`)
			if md != ad {
				t.Errorf("example %d renders differently as AsciiDoc:\n%s\n--- adoc ---\n%s", n, ex, deck.MarkdownToAdoc(ex))
			}
		}
	}
	if n < 40 {
		t.Errorf("only %d examples found", n)
	}
}

func TestWithAdocTabs(t *testing.T) {
	got := withAdoc("Intro\n\n  ```markdown\n  ## Hi **there**\n  ```\n")
	for _, w := range []string{"  <!-- tabs -->", "  ```markdown title=Markdown", "  ## Hi **there**", "  ```asciidoc title=AsciiDoc", "  == Hi *there*"} {
		if !strings.Contains(got, w+"\n") {
			t.Errorf("missing %q in\n%s", w, got)
		}
	}
}
