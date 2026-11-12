package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestRenderGuide(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"_nav.md":      "---\ntitle: The order journey\nversion: v1\nedit-url: https://git.example/edit/main/\n---\n- [Introduction](index.md)\n- The happy path\n  - [Pay](happy/pay.md)\n",
		"_header.md":   "[Home](index.md)\n",
		"_footer.md":   "Questions go to #orders.\n",
		"index.md":     "---\ndescription: Start here.\n---\n# Introduction\n\n## Step 1: Install\n\nSee [paying](happy/pay.md#step-2-wait).\n",
		"happy/pay.md": "# Pay\n\n## Step 1: Pay\n\n## Step 2: Wait\n\nBack to [the start](../index.md) and [nowhere](../index.md#no-such-anchor) and [gone](../gone.md).\n",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, probs, _ := RenderGuide(g, Options{})
	if len(pages) != 2 {
		t.Fatalf("pages: %d", len(pages))
	}
	var msgs []string
	for _, p := range probs {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	if len(probs) != 2 || !strings.Contains(all, "#no-such-anchor: no such anchor") || !strings.Contains(all, "gone.md: no such page") {
		t.Errorf("want the broken anchor and the missing page, got:\n%s", all)
	}
	pay := string(pages[1].Result.HTML)
	for _, want := range []string{
		`data-si-doc="guide"`, `class="si-doc-nav"`, `href="../index.html" data-page="index.html"`, `aria-current="page"`,
		`<span>The happy path</span>`, `class="si-doc-crumb"`, `rel="prev" href="../index.html"`,
		`href="https://git.example/edit/main/happy/pay.md"`, `<span class="si-step-num">2</span>Wait`,
		`class="si-doc-site"`, `<a href="../index.html">Home</a>`, `Questions go to #orders.`, `<title>Pay · The order journey</title>`, `>v1<`,
	} {
		if !strings.Contains(pay, want) {
			t.Errorf("happy/pay.html lacks %q", want)
		}
	}
	idx := string(pages[0].Result.HTML)
	if !strings.Contains(idx, `<p class="si-lede">Start here.</p>`) || !strings.Contains(idx, `rel="next" href="happy/pay.html"`) {
		t.Error("index: lede and next link to the second nav page expected")
	}
}

func TestArticle(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	words := strings.Repeat("word ", 700)
	os.WriteFile(p, []byte("---\ntype: article\ntitle: A claim\ndescription: The lede.\nauthor: Jo\n---\n# A claim\n\n## One\n\n"+words+"\n\n## Two\n\n## Three\n\n## Step 4: not a step in an article\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	h := string(Deck(d, Options{}).HTML)
	for _, want := range []string{`data-si-doc="article"`, `<p class="si-lede">The lede.</p>`, `<span class="si-reading-time">4 min read</span>`, `class="si-doc-toc"`, `<meta name="description" content="The lede.">`} {
		if !strings.Contains(h, want) {
			t.Errorf("article lacks %q", want)
		}
	}
	if strings.Contains(h, `<span class="si-step-num">`) {
		t.Error("articles have no step badges")
	}
}

// A site page with `unlisted: true` is built and is a valid link target (the
// navigator cta, _footer.md and body links to it resolve in si check), but it
// never appears in the navigator menu. A page that is neither listed nor
// unlisted is a genuine orphan: a warning, not built.
func TestSiteUnlisted(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"_site.md":   "---\ntitle: Northwind\ncta: Talk to us → contact\n---\n- [Home](index.md)\n- [About](about.md)\n",
		"_footer.md": "[Contact](contact.md)\n",
		"index.md":   "# Home\n\nSee [contact us](contact.md).\n",
		"about.md":   "# About\n",
		"contact.md": "---\nunlisted: true\ntitle: Talk to Us\n---\n# Talk to Us\n\nEmail us.\n",
		"orphan.md":  "# Orphan\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, probs, warns := RenderGuide(g, Options{})

	// contact.md is built; orphan.md is not.
	built := map[string]bool{}
	for _, p := range pages {
		built[p.Rel] = true
	}
	if !built["contact.html"] {
		t.Error("unlisted page contact.html was not built")
	}
	if built["orphan.html"] {
		t.Error("orphan.html should not be built")
	}

	// The cta/footer/body links to contact resolve: no "no such page" problem.
	for _, p := range probs {
		if strings.Contains(p.Msg, "no such page") {
			t.Errorf("unexpected link problem: %s", p.Msg)
		}
	}

	// orphan.md warns; contact.md (unlisted) does not.
	var warnText string
	for _, w := range warns {
		warnText += w.File + " " + w.Msg + "\n"
	}
	if !strings.Contains(warnText, "orphan.md") || !strings.Contains(warnText, "not listed") {
		t.Errorf("want an orphan warning for orphan.md, got:\n%s", warnText)
	}
	if strings.Contains(warnText, "contact.md") {
		t.Errorf("unlisted page must not warn, got:\n%s", warnText)
	}

	// The navigator menu of the home page lists Home and About only, never the
	// unlisted page — though its cta button does link to it.
	var idx string
	for _, p := range pages {
		if p.Rel == "index.html" {
			idx = string(p.Result.HTML)
		}
	}
	menu := idx
	if i := strings.Index(menu, "si-site-links"); i >= 0 {
		menu = menu[i:]
		if j := strings.Index(menu, "</nav>"); j >= 0 {
			menu = menu[:j]
		}
	}
	if strings.Contains(menu, "contact.html") || strings.Contains(menu, "Talk to Us") {
		t.Errorf("unlisted page must not appear in the navigator menu:\n%s", menu)
	}
	if !strings.Contains(idx, `class="si-site-cta" href="contact.html"`) {
		t.Error("the navigator cta should link to the unlisted page")
	}
}
