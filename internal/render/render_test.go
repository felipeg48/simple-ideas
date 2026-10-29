package render

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.html")

var mainRE = regexp.MustCompile(`(?s)<main .*</main>`)

func renderGolden(t *testing.T, opt Options) (Result, string) {
	t.Helper()
	d, err := deck.Parse(filepath.Join("testdata", "deck.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, opt)
	return res, string(res.HTML)
}

// TestGolden compares the slide markup of testdata/deck.md (every rule) with
// testdata/golden.html. Run `go test ./internal/render -update` after an
// intended change and review the diff.
func TestGolden(t *testing.T) {
	res, page := renderGolden(t, Options{})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	got := mainRE.FindString(page) + "\n"
	golden := filepath.Join("testdata", "golden.html")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("golden mismatch at line %d:\n got: %.300s\nwant: %.300s\n(run go test ./internal/render -update if intended)", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("golden mismatch: %d lines, want %d", len(gl), len(wl))
	}
}

func TestPage(t *testing.T) {
	res, page := renderGolden(t, Options{UserCSS: []Stylesheet{{Path: "testdata/my.css", Text: ".si-slide{color:red} a{background:url(assets/pic.svg)}"}}})
	if res.Steps != 5 || res.Canvases != 2 {
		t.Errorf("steps = %d, canvases = %d; want 5 and 2", res.Steps, res.Canvases)
	}
	for _, want := range []string{
		`<html lang="en" data-si-theme="paper">`,
		"@layer si.theme, si.user;",
		`--si-accent:#e4572e`,
		`<style id="si-user">`, "@layer si.user{", `url("data:image/svg+xml;base64,`,
		`"transition":"slide"`, `"plan":[0.5,0,1,0,0,0,0,0,0,0,0]`,
		`.si-canvas[data-state~="on"] [data-show~="on"]{opacity:1}`,
		`.si-canvas[data-state~="busy"] [data-flow~="busy"]`,
		`.si-canvas[data-state~="on"] .si-packets[data-for~="on"]`,
		"@page{size:13.333in 7.5in;margin:0}",
		"window.SI_PRESENTER",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	// hide rules come after show rules so data-hide wins
	if strings.Index(page, `[data-hide~="on"]`) < strings.Index(page, `[data-show~="on"]`) {
		t.Error("hide rules must follow show rules")
	}
	if strings.Contains(page, "mermaid.min.js") {
		t.Error("no mermaid fence, no mermaid script")
	}
	if strings.Contains(page, "EventSource") {
		t.Error("static output must not include the live reload client")
	}
}

func TestMediaModes(t *testing.T) {
	res, page := renderGolden(t, Options{Media: MediaCopy})
	if !strings.Contains(page, `<img src="assets/pic.svg"`) || len(res.Media) != 1 || res.Media[0].Rel != "assets/pic.svg" {
		t.Errorf("copy mode: media = %+v", res.Media)
	}
	_, page = renderGolden(t, Options{Media: MediaLink, LiveReload: true})
	if !strings.Contains(page, `<img src="assets/pic.svg"`) || !strings.Contains(page, "/_si/events") {
		t.Error("link mode links files and play adds live reload")
	}
}

func TestStateOps(t *testing.T) {
	tests := []struct {
		prev  []string
		words string
		want  string
	}{
		{nil, "a b", "a b"},
		{[]string{"a", "b"}, "+c", "a b c"},
		{[]string{"a", "b"}, "-a +c", "b c"},
		{[]string{"a", "b"}, "c", "c"}, // absolute words replace the state
		{[]string{"a"}, "", ""},
	}
	for _, tc := range tests {
		if got := strings.Join(resolveState(tc.prev, strings.Fields(tc.words)), " "); got != tc.want {
			t.Errorf("resolveState(%v, %q) = %q, want %q", tc.prev, tc.words, got, tc.want)
		}
	}
	if got := strings.Join(applyOps([]string{"a", "b"}, "-a,c . +d"), " "); got != "b c d" {
		t.Errorf("applyOps = %q", got)
	}
}

func TestRelativeCanvasState(t *testing.T) {
	_, page := renderGolden(t, Options{})
	// slide 3 ends at on+busy-on+bad = busy bad; slide 4 adds +busy
	if !strings.Contains(page, `data-state="on" data-base="on"`) || !strings.Contains(page, `data-state="busy bad" data-base="busy bad"`) {
		t.Error("relative canvas state did not carry across slides")
	}
	if strings.Count(page, `id="c1-box"`) != 1 || strings.Count(page, `id="c2-box"`) != 1 || !strings.Contains(page, `fill="url(#c2-g)"`) {
		t.Error("svg ids must be prefixed per instance, references too")
	}
	if !strings.Contains(page, `viewBox="0 0 200 100"`) {
		t.Error("a viewBox is derived from width and height")
	}
}

func TestHighlight(t *testing.T) {
	tests := []struct{ code, lang, want string }{
		{`if x := "s"; x { return 1 } // c`, "go", `<span class="si-tk-k">if</span> x := <span class="si-tk-s">&#34;s&#34;</span>; x { <span class="si-tk-k">return</span> <span class="si-tk-n">1</span> } <span class="si-tk-c">// c</span>`},
		{"si play --open $DIR # go", "bash", `<span class="si-tk-m">si</span> play<span class="si-tk-f"> --open</span> <span class="si-tk-v">$DIR</span> <span class="si-tk-c"># go</span>`},
		{"a < b", "text", "a &lt; b"},
	}
	for _, tc := range tests {
		if got := highlight(tc.code, tc.lang); got != tc.want {
			t.Errorf("highlight(%q, %s) =\n %s\nwant\n %s", tc.code, tc.lang, got, tc.want)
		}
	}
}

func TestToneOf(t *testing.T) {
	for in, want := range map[string]string{"#101820": "dark", "#fff": "light", "linear-gradient(#f6f5f1, #000)": "light", "red": ""} {
		if got := toneOf(in); got != want {
			t.Errorf("toneOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMissingFilesAreProblems(t *testing.T) {
	dir := t.TempDir()
	md := "# D\n\n---\n\n## A\n\n![x](nope.png)\n\n---\n\n## B\n\n<!-- background: gone.jpg -->\n"
	if err := os.WriteFile(filepath.Join(dir, "d.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	d, _ := deck.Parse(filepath.Join(dir, "d.md"))
	res := Deck(d, Options{})
	all := ""
	for _, p := range res.Problems {
		all += p.Msg + "\n"
	}
	if !strings.Contains(all, "missing file: nope.png") || !strings.Contains(all, "missing file: gone.jpg") {
		t.Errorf("problems = %q", all)
	}
}

func TestRunbook(t *testing.T) {
	dir := t.TempDir()
	rb := `{"categories":[{"name":"SQL","entries":[` +
		`{"label":"Nodes","description":"list nodes","command":"SELECT node_id FROM crdb_internal.gossip_nodes;","target":"sql"},` +
		`{"label":"Status","sql":"cockroach node status --insecure","target":"bash"}]}]}`
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rb.json", rb)
	write("d.md", "---\nrunbook: rb.json\n---\n# D\n\n---\n\n## A\n\nbody\n")
	d, err := deck.Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	// si play (Terminal) injects SI_RUNBOOK: highlighted html + the raw command
	res := Deck(d, Options{Terminal: true})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	page := string(res.HTML)
	for _, want := range []string{"window.SI_RUNBOOK=", "si-tk-k", "crdb_internal.gossip_nodes", "cockroach node status"} {
		if !strings.Contains(page, want) {
			t.Errorf("play page lacks %q", want)
		}
	}
	// a built deck (no Terminal) must NOT carry the runbook — stays self-contained
	if b := string(Deck(d, Options{}).HTML); strings.Contains(b, "SI_RUNBOOK") {
		t.Error("built deck must not contain the runbook")
	}
	// a missing runbook file is a problem (caught by si check, which never sets Terminal)
	write("b.md", "---\nrunbook: nope.json\n---\n# D\n")
	bd, _ := deck.Parse(filepath.Join(dir, "b.md"))
	if r := Deck(bd, Options{}); len(r.Problems) == 0 {
		t.Error("missing runbook file should be a problem")
	}
}

func TestTopicsFollowSectionsAndModules(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"d.md": "# D\n\n---\n\n## Agenda\n\n<!-- include: a.md -->\n<!-- include: b.md -->\n\n---\n\n# Part three\n\n---\n\n## In part three\n\n---\n\n# Thanks\n",
		"a.md": "## A one\n\n---\n\n## A two\n",
		"b.md": "# B section\n\n---\n\n## B one\n",
	}
	for name, md := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := deck.Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, s := range d.Slides {
		titles = append(titles, s.Title)
	}
	// title, agenda | A one, A two (module without a section) | B section, B one |
	// Part three, In part three, Thanks (the closing stays in the current column)
	want := []int{0, 0, 1, 1, 2, 2, 3, 3, 3}
	if got := Topics(d); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("topics = %v, want %v for %q", got, want, titles)
	}
}

func TestLabDirectiveWarningReachesResult(t *testing.T) {
	p := filepath.Join(t.TempDir(), "l.md")
	if err := os.WriteFile(p, []byte("---\ntype: lab\n---\n# Lab\n\n<!-- layout: two-col -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) != 0 || len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0].Msg, "no effect in a lab document") {
		t.Errorf("problems=%v warnings=%v", res.Problems, res.Warnings)
	}
}

var labBodyRE = regexp.MustCompile(`(?s)<body.*</footer>`)

func renderLab(t *testing.T, path string, opt Options) (Result, string) {
	t.Helper()
	d, err := deck.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, opt) // dispatches to Document for type: lab
	return res, string(res.HTML)
}

// TestLabGolden compares the page markup of testdata/lab.md (headings, callouts,
// a wide table, a long command, an image, an include, a .md link) with
// testdata/lab.golden.html. Run `go test ./internal/render -update` after an
// intended change and review the diff.
func TestLabGolden(t *testing.T) {
	res, page := renderLab(t, filepath.Join("testdata", "lab.md"), Options{})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	if res.Steps != 0 {
		t.Errorf("steps = %d, want 0", res.Steps)
	}
	got := labBodyRE.FindString(page) + "\n"
	golden := filepath.Join("testdata", "lab.golden.html")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Fatalf("lab golden mismatch (run go test ./internal/render -update if intended)")
	}
	for _, want := range []string{
		`<body class="si-doc">`, `<nav class="si-doc-toc"`, `<main class="si-doc-body"`,
		`<div class="si-callout" data-kind="note">`, `data-kind="tip"`, `data-kind="important"`, `data-kind="warning"`, `data-kind="caution"`,
		`<blockquote>`, `href="other.html#start"`, `id="read-it-back"`, `<button type="button" class="si-copy"`,
		"<div class=\"si-table-wrap\">\n<table>", `src="data:image/svg+xml;base64,`, "<hr>",
		`data-si-theme="studio"`, "@layer si.theme, si.user;", "© 2026 Harbor Labs",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("lab page lacks %q", want)
		}
	}
	if strings.Contains(page, "[!NOTE]") || strings.Contains(page, "si-deck") || strings.Contains(page, "SI_PRESENTER") {
		t.Error("callout markers and deck chrome must not reach a lab page")
	}
}

func TestLabHeadingIDs(t *testing.T) {
	_, page := renderLab(t, filepath.Join("testdata", "lab.md"), Options{})
	for _, want := range []string{
		`<h2 id="before-you-start">`, `<h2 id="set-up">`, `<h3 id="clone">`, `<h3 id="clone-2">`,
		`<a href="#clone-2">Clone</a>`, `<a class="si-anchor" href="#set-up"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("lab page lacks %q", want)
		}
	}
	s := newSlugger()
	for _, tc := range [][2]string{{"Set up", "set-up"}, {"What's `new`?", "whats-new"}, {"set up", "set-up-2"}, {"Set up", "set-up-3"},
		{"Ünïcode 2 go", "ünïcode-2-go"}, {"a_b-c", "a_b-c"}, {"!!!", "section"},
		{"Clone 2", "clone-2"}, {"Clone", "clone"}, {"Clone", "clone-3"}} {
		if got := s.slug(tc[0]); got != tc[1] {
			t.Errorf("slug(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

func TestLabSelfContained(t *testing.T) {
	resRE := regexp.MustCompile(`(?i)(src|href)="(https?:)?//|url\((["']?)(https?:)?//|@import`)
	_, page := renderLab(t, filepath.Join("testdata", "lab.md"), Options{})
	page = regexp.MustCompile(`<a [^>]*href="https?://[^"]*"`).ReplaceAllString(page, "") // links are navigation, not resources
	if m := resRE.FindString(page); m != "" {
		t.Errorf("lab page loads a network resource: %q", m)
	}
	if strings.Contains(page, "<script src=") {
		t.Error("no mermaid fence, no mermaid script")
	}
	p := filepath.Join(t.TempDir(), "m.md")
	if err := os.WriteFile(p, []byte("---\ntype: lab\n---\n# M\n\n```mermaid\ngraph LR\n a-->b\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page = renderLab(t, p, Options{LiveReload: true})
	if !strings.Contains(page, `<script src="`+MermaidURL+`">`) || !strings.Contains(page, `<pre class="si-mermaid">`) || !strings.Contains(page, "/_si/events") {
		t.Error("a lab with a mermaid fence loads the pinned Mermaid script; play adds live reload")
	}
}

func TestLabEdgeCases(t *testing.T) {
	dir := t.TempDir()
	write := func(name, md string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// a ## Content heading owns its id: the page template reserves none
	_, page := renderLab(t, write("c.md", "---\ntype: lab\n---\n# C\n\n## Content\n\ntext\n"), Options{})
	if strings.Count(page, `id="content"`) != 1 {
		t.Errorf("id=\"content\" appears %d times, want 1", strings.Count(page, `id="content"`))
	}
	// a raw HTML table stays balanced; a markdown table is wrapped
	_, page = renderLab(t, write("t.md", "---\ntype: lab\n---\n# T\n\n<table class=\"x\"><tr><td>raw</td></tr></table>\n\n| a | b |\n|---|---|\n| 1 | 2 |\n"), Options{})
	body := regexp.MustCompile(`(?s)<main .*</main>`).FindString(page)
	if strings.Count(body, "<div") != strings.Count(body, "</div>") || strings.Count(body, `class="si-table-wrap"`) != 1 {
		t.Errorf("unbalanced or missing table wrap:\n%s", body)
	}
	// the first H1 moves to the header only when it is the title
	_, page = renderLab(t, write("h.md", "---\ntype: lab\ntitle: Given\n---\n# Heading\n\ntext\n"), Options{})
	if !strings.Contains(page, `<h1 class="si-doc-title">Given</h1>`) || !strings.Contains(page, "<h1>Heading</h1>") {
		t.Error("an H1 that differs from the front matter title stays in the body")
	}
	_, page = renderLab(t, write("s.md", "---\ntype: lab\ntitle: Same\n---\n# Same\n\ntext\n"), Options{})
	if strings.Contains(page, "<h1>Same</h1>") {
		t.Error("an H1 equal to the title moves to the header")
	}
}
