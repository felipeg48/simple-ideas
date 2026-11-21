package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func renderDoc(t *testing.T, body string) Result {
	t.Helper()
	p := filepath.Join(t.TempDir(), "d.md")
	if err := os.WriteFile(p, []byte("---\ntype: lab\n---\n# D\n\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	return Deck(d, Options{})
}

func TestCodeLinesAndCallouts(t *testing.T) {
	res := renderDoc(t, "```java title=\"App.java\" hl=\"2\" linenos start=10\nvar a = 1; // <1>\nvar b = 2;\nvar c = 3; # <2>\n```\n<1> First.\n<2> Third.\n")
	h := string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	for _, want := range []string{
		`<span class="si-code-title">App.java</span>`,
		`<span class="si-line"><span class="si-ln" data-n="10"></span>`,
		`<span class="si-line si-hl"><span class="si-ln" data-n="11"></span>`,
		`<span class="si-co" data-co="1" role="img" aria-label="callout 1"></span>`,
		`<ol class="si-callouts"><li data-co="1">`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	if strings.Contains(h, "// &lt;1&gt;") || strings.Contains(h, "&lt;2&gt;") {
		t.Error("markers and their comment are removed from the code")
	}
	bad := renderDoc(t, "```go\nx := 1 // <1>\n```\n<2> Nothing marks this.\n")
	var msgs []string
	for _, p := range bad.Problems {
		msgs = append(msgs, p.Msg)
	}
	if all := strings.Join(msgs, "\n"); !strings.Contains(all, "callout <2> has no <2> marker") || !strings.Contains(all, "code marker <1> has no") {
		t.Errorf("callout problems: %s", all)
	}
}

// AsciiDoc books (issue #22): <.> numbers itself, /* <1> */ is a marker, and
// a callout that wraps onto a second line is one item.
func TestCalloutsAsciiDocForms(t *testing.T) {
	res := renderDoc(t, "```java\nint a; // <.>\nint b; /* <.> */\n```\n<.> The first,\nwrapped.\n<.> The second.\n")
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{
		`<span class="si-co" data-co="1" role="img" aria-label="callout 1"></span>`,
		`<span class="si-co" data-co="2" role="img" aria-label="callout 2"></span>`,
		`<span>The first, wrapped.</span>`,
		`<li data-co="2"><span class="si-co" data-co="2" aria-hidden="true"></span><span>The second.</span>`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	pre := h[strings.Index(h, "<pre"):]
	pre = pre[:strings.Index(pre, "</pre>")]
	if strings.Contains(pre, "&lt;.&gt;") || strings.Contains(pre, "/*") {
		t.Error("the markers and their comment leave the code")
	}
}

// Issues #24, #25: a code block and its callout list are one card, a
// figure holding both, in documents, in tabs and on slides.
func TestCalloutCard(t *testing.T) {
	card := regexp.MustCompile(`(?s)<figure class="si-codecallout">\s*(<div class="si-codeblock">.*?</div>\s*|<pre class="si-code"[^>]*>.*?</pre>\s*)<ol class="si-callouts">.*?</ol>\s*</figure>`)
	h := string(renderDoc(t, "```go\nx := 1 // <1>\n```\n<1> One.\n\nAfter.\n").HTML)
	if !card.MatchString(h) {
		t.Errorf("document: no card:\n%s", h)
	}
	h = string(renderDoc(t, "<!-- tabs -->\n```go\nx := 1 // <1>\n```\n<1> One.\n\n```python\ny = 2\n```\n").HTML)
	if !regexp.MustCompile(`<div class="si-tab" data-label="Go">\s*<figure class="si-codecallout">`).MatchString(h) || !strings.Contains(h, `data-label="Python"`) {
		t.Errorf("tabs: the card is the first tab:\n%s", h)
	}
	p := filepath.Join(t.TempDir(), "s.md")
	os.WriteFile(p, []byte("# S\n\n```go\nx := 1 // <1>\n```\n<1> One.\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if h := string(Deck(d, Options{}).HTML); !card.MatchString(h) {
		t.Errorf("slide: no card")
	}
}

func TestCodeTabs(t *testing.T) {
	res := renderDoc(t, "<!-- tabs -->\n```java\nA\n```\n```python title=\"Py 3\"\nB\n```\n\nAfter.\n")
	h := string(res.HTML)
	for _, want := range []string{`<div class="si-tabs">`, `role="tablist"`, `aria-selected="true" data-label="Java"`, `data-label="Py 3">Py 3</button>`,
		`<div class="si-tab" data-label="Java">`, `<div class="si-tab" data-label="Py 3" hidden`} {
		if !strings.Contains(h, want) {
			t.Errorf("tabs lack %q", want)
		}
	}
	if len(res.Warnings) > 0 {
		t.Errorf("tabs are a document directive, no warning: %v", res.Warnings)
	}
	if p := renderDoc(t, "<!-- tabs -->\n\nJust text.\n").Problems; len(p) != 1 {
		t.Errorf("tabs without code blocks: %v", p)
	}
}

func TestSplitHTMLLines(t *testing.T) {
	got := splitHTMLLines(`a<span class="x">b` + "\n" + `c</span>d`)
	if len(got) != 2 || got[0] != `a<span class="x">b</span>` || got[1] != `<span class="x">c</span>d` {
		t.Errorf("%q", got)
	}
}

func TestEmbedSlidesAndDiff(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("---\naspect: 4:3\n---\n# Talk\n\n---\n\n## Second slide\n\nB\n\n---\n\n## Third slide\n\nC\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "d.md"), []byte("---\ntype: lab\n---\n# D\n\n<!-- slides: talk.md 2-3 -->\n\n<!-- slides: talk.md#third-slide -->\n\n<!-- slides: talk.md 9 -->\n\n"+
		"```diff\n@@ -1,2 +1,2 @@\n ctx\n-old\n+new\n```\n"), 0o644)
	d, _ := deck.Parse(filepath.Join(dir, "d.md"))
	res := Deck(d, Options{})
	h := string(res.HTML)
	if strings.Count(h, `class="si-embed-slides"`) != 2 || !strings.Contains(h, `--_aspect:4/3`) || !strings.Contains(h, "slides 2–3") || !strings.Contains(h, "slide 3.") {
		t.Error("two embeds expected: slides 2–3 and slide 3, at 4/3")
	}
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0].Msg, "has 3 slides") {
		t.Errorf("an out-of-range embed is a problem: %v", res.Problems)
	}
	for _, want := range []string{`<span class="si-line si-diff-del"><span class="si-ln" data-n="2"></span><span class="si-ln" data-n=""></span>-old</span>`,
		`<span class="si-line si-diff-add"><span class="si-ln" data-n=""></span><span class="si-ln" data-n="2"></span>+new</span>`} {
		if !strings.Contains(h, want) {
			t.Errorf("diff lacks %q", want)
		}
	}
}

// A failed include inside a lab solution is its only problem: the solution
// still finds its end (issue #10), and a solution without an end is reported
// at its own line, not at the part's first line.
func TestFailedIncludeInSolution(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.py"), []byte("def f():\n    # tag::a[]\n    return 1\n    # end::a[]\n"), 0o644)
	p := filepath.Join(dir, "u.md")
	os.WriteFile(p, []byte("---\ntype: lab\ntitle: T\n---\n\n# T\n\n## Step 1: A\n\n<!-- solution -->\n\n<!-- include: x.py tag=a title=x (part one) -->\n\n<!-- end -->\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) != 1 || res.Problems[0].Line != 12 ||
		!strings.Contains(res.Problems[0].Msg, `unknown option "one"`) ||
		!strings.HasSuffix(res.Problems[0].Msg, `title with spaces must be quoted: title="x (part one)"`) {
		t.Fatalf("want only the include problem at line 12 with the quoting hint, got %v", res.Problems)
	}
	os.WriteFile(p, []byte("---\ntype: lab\n---\n\n# T\n\nText.\n\n<!-- solution -->\n\nAnswer.\n"), 0o644)
	d, _ = deck.Parse(p)
	res = Deck(d, Options{})
	if len(res.Problems) != 1 || res.Problems[0].Line != 9 || !strings.Contains(res.Problems[0].Msg, "solution: no <!-- end -->") {
		t.Fatalf("want the solution problem at line 9, got %v", res.Problems)
	}
}
