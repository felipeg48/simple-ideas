package render

import (
	"os"
	"path/filepath"
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
		`<span class="si-co" data-co="1" aria-label="callout 1"></span>`,
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
