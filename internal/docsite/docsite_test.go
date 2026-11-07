package docsite

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/docs"
)

// Every numbered section of RULES.md gets a page and a nav entry.
func TestEveryRulesSectionHasAPage(t *testing.T) {
	want := regexp.MustCompile(`(?m)^## (\d+)\. `).FindAllStringSubmatch(docs.Rules, -1)
	_, secs := Split(docs.Rules)
	if len(secs) != len(want) {
		t.Fatalf("split found %d sections, RULES.md has %d '## N.' headings", len(secs), len(want))
	}
	dst := filepath.Join(t.TempDir(), "site")
	n, err := Generate(dst)
	if err != nil || n != len(secs)+1 {
		t.Fatalf("generate: %d %v", n, err)
	}
	nav, _ := os.ReadFile(filepath.Join(dst, "rules", "_nav.md"))
	for _, s := range secs {
		page, err := os.ReadFile(filepath.Join(dst, "rules", s.Slug+".md"))
		if err != nil {
			t.Errorf("section %s has no page: %v", s.N, err)
			continue
		}
		if !strings.Contains(string(page), "# "+s.N+". "+s.Title) || !strings.Contains(string(nav), "("+s.Slug+".md)") {
			t.Errorf("section %s: page or nav entry missing", s.N)
		}
	}
	for _, f := range []string{"_site.md", "index.md", "examples.md", "manual.md", "rules/index.md", "_examples/showcase.md", "_examples/assets/dusk.jpg"} {
		if _, err := os.Stat(filepath.Join(dst, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if _, err := Generate(dst); err == nil {
		t.Error("generating into a non-empty folder must refuse")
	}
}

func TestSplitAndRefs(t *testing.T) {
	intro, secs := Split("# R\n\nintro\n\n## 1. One\n\nsee §2\n\n```md\n## 9. Not a section\n```\n\n## 2. Two\n\nback to §1 and `§1`\n")
	if intro != "# R\n\nintro" || len(secs) != 2 || !strings.Contains(secs[0].Body, "## 9. Not a section") {
		t.Fatalf("%q %+v", intro, secs)
	}
	got := linkRefs(secs[1].Body, map[string]string{"1": "01-one"})
	if got != "back to [§1](01-one.md) and `§1`" {
		t.Errorf("refs: %q", got)
	}
}
