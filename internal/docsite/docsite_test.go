package docsite

import (
	"io/fs"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
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
	// rules/ is its own guide: it carries the site's icons and footer itself
	if !strings.Contains(string(nav), "favicon: ../brand/favicon.svg") || !strings.Contains(string(nav), "touch-icon: ../brand/apple-touch-icon.png") {
		t.Error("rules/_nav.md lacks the site's favicon or touch icon")
	}
	for _, f := range []string{"_site.md", "index.md", "examples.md", "manual.md", "rules/index.md", "rules/_footer.md", "brand/favicon.svg", "brand/apple-touch-icon.png", "download.md", "live/showcase.md", "live/assets/dusk.jpg", "live/guide/_nav.md", "live/book/_nav.md"} {
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

// Every main feature has a page in the menu, and the page, with its live
// example and the Markdown read from examples/, renders without problems.
func TestEveryFeatureHasAWorkingPage(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(dst); err != nil {
		t.Fatal(err)
	}
	menu, _ := os.ReadFile(filepath.Join(dst, "_site.md"))
	for _, f := range Features {
		p := filepath.Join(dst, "features", f.Slug+".md")
		if !strings.Contains(string(menu), "(features/"+f.Slug+".md)") {
			t.Errorf("%s is not in the site menu", f.Slug)
		}
		d, err := deck.ParseWith(p, deck.Meta{Type: "page"}) // a site page
		if err != nil {
			t.Fatalf("%s: %v", f.Slug, err)
		}
		res := render.Deck(d, render.Options{})
		for _, pr := range append(d.Problems, res.Problems...) {
			t.Errorf("%s: %s", f.Slug, pr.Msg)
		}
		if !strings.Contains(string(res.HTML), "## The Markdown behind it") && !strings.Contains(string(res.HTML), "The Markdown behind it") {
			t.Errorf("%s: no source section", f.Slug)
		}
	}
}

// The Themes page shows every built-in theme but the unlisted ones, and each
// has its picture.
func TestThemesPageHasEveryTheme(t *testing.T) {
	page := themesPage()
	for _, n := range assets.ThemeNames() {
		if unlisted[n] {
			if strings.Contains(page, "`theme: "+n+"`") || strings.Contains(page, "themes/"+n+".jpg") {
				t.Errorf("unlisted theme %s is on the Themes page", n)
			}
			continue
		}
		if !strings.Contains(page, "themes/"+n+".jpg") || !strings.Contains(page, "`theme: "+n+"`") {
			t.Errorf("theme %s is not on the Themes page", n)
		}
		if _, err := fs.Stat(docs.Site, "site/themes/"+n+".jpg"); err != nil {
			t.Errorf("theme %s has no picture: run make theme-shots", n)
		}
	}
}
