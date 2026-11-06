package studio

import (
	"fmt"
	"strings"
	"testing"
)

func siteFx(t *testing.T) *fx {
	f := newFx(t)
	f.put("_site.md", "---\ntitle: Harbor\n---\n\n- [Home](index.md)\n- [Post](post.md)\n- [Docs](docs/)\n- [Talk](talk.md)\n")
	f.put("index.md", "# Home\n\n<!-- alert: Open late on Fridays · ok -->\n\nOne.\n\n---\n\nStill the home page. See [the post](post.md#gone).\n")
	f.put("post.md", "---\ntype: article\n---\n# Post\n\n## Here\n\nText.\n")
	f.put("talk.md", "---\ntype: deck\n---\n# Talk\n\n---\n\n## Two\n")
	f.put("docs/_nav.md", "---\ntitle: Docs\n---\n\n- [Start](start.md)\n")
	f.put("docs/start.md", "# Start\n\nGo.\n")
	return f
}

func TestCheckSitePageLikeSiCheck(t *testing.T) {
	f := siteFx(t)
	code, m := f.do("GET", "check?deck=index.md", nil)
	if code != 200 {
		t.Fatalf("%d %v", code, m)
	}
	ps := fmt.Sprint(m["problems"])
	if strings.Contains(ps, "not on a slide") {
		t.Errorf("a site page is not a slide: %s", ps)
	}
	if !strings.Contains(ps, "post.md#gone") {
		t.Errorf("links between pages are checked: %s", ps)
	}
	if m["kind"] != "page" {
		t.Errorf("kind %v", m["kind"])
	}
	if _, m := f.do("GET", "check?deck=_site.md", nil); m["kind"] != "site" {
		t.Errorf("_site.md checks the site: %v", m)
	}
	if _, m := f.do("GET", "check?deck=docs/start.md", nil); m["kind"] != "guide" || fmt.Sprint(m["problems"]) != "[]" {
		t.Errorf("sub-guide page: %v", m)
	}
}

func TestDecksAndTreeKnowSites(t *testing.T) {
	f := siteFx(t)
	kinds := map[string]string{}
	for _, d := range f.doAny("GET", "decks") {
		m := d.(map[string]any)
		kinds[m["path"].(string)] = m["kind"].(string) + " " + m["url"].(string)
	}
	for path, want := range map[string]string{
		"_site.md": "site /index.html", "index.md": "page /index.html", "post.md": "article /post.html",
		"talk.md": "deck /talk.html", "docs/_nav.md": "guide /docs/start.html", "docs/start.md": "guide /docs/start.html",
	} {
		if kinds[path] != want {
			t.Errorf("%s: %q, want %q", path, kinds[path], want)
		}
	}
	_, tree := f.do("GET", "tree", nil)
	if tree["guide"] != "site" {
		t.Errorf("root is a site: %v", tree["guide"])
	}
	s := fmt.Sprint(tree["children"])
	for _, want := range []string{"guide:guide", "kind:nav", "kind:article", "kind:page"} {
		if !strings.Contains(s, want) {
			t.Errorf("tree lacks %s: %s", want, s)
		}
	}
}

func TestTitlesDropHighlightMarks(t *testing.T) {
	f := newFx(t)
	f.put("talk.md", "# The ==Thursday== promise\n\n---\n\n## Five things ==to keep==\n")
	_, m := f.do("GET", "check?deck=talk.md", nil)
	if s := fmt.Sprint(m["slides"]); strings.Contains(s, "==") || !strings.Contains(s, "Five things to keep") {
		t.Errorf("slide titles: %s", s)
	}
	if d := fmt.Sprint(f.doAny("GET", "decks")); strings.Contains(d, "==") {
		t.Errorf("deck titles: %s", d)
	}
}
