package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseInfographic(t *testing.T) {
	src := "# Title\nA subtitle\nin two lines\n\n## 2.4 s · stat · accent\nMedian load.\n\n## Paris · London\n| A | B |\n|---|---|\n| 1 | 2 |\n\n## Share · chart=donut · span=2 · tall\n| x | % |\n|---|---|\n| a | 60 |\n| b | 40 |\n\n## Sources · sources\n- One\n"
	ig, probs := ParseInfographic(src)
	if len(probs) != 0 {
		t.Fatalf("problems: %v", probs)
	}
	if ig.Title != "Title" || ig.Subtitle != "A subtitle in two lines" {
		t.Errorf("title %q subtitle %q", ig.Title, ig.Subtitle)
	}
	if len(ig.Blocks) != 3 || ig.Sources == nil {
		t.Fatalf("blocks %d, sources %v", len(ig.Blocks), ig.Sources)
	}
	b := ig.Blocks[0]
	if b.Title != "2.4 s" || b.Kind != "stat" || !b.Accent || b.Guessed || b.Line != 5 {
		t.Errorf("stat block: %+v", b)
	}
	if b := ig.Blocks[1]; b.Title != "Paris · London" || b.Kind != "compare" || !b.Guessed {
		t.Errorf("a title with a middle dot: %+v", b)
	}
	if b := ig.Blocks[2]; b.Kind != "chart" || b.Chart != "donut" || b.Span != 2 || b.Rows != 2 {
		t.Errorf("chart block: %+v", b)
	}
}

func TestGuessIGKind(t *testing.T) {
	for _, c := range []struct{ title, body, kind, chart string }{
		{"54%", "of visits", "stat", ""},
		{"$1.2B", "", "stat", ""},
		{"Share", "| a | n |\n|---|---|\n| x | 1 |\n| y | 2 |", "chart", "bar"},
		{"Before/after", "| Before | After |\n|---|---|\n| 2.4 s | 0.6 s |", "compare", ""},
		{"Words", "| a | b |\n|---|---|\n| x | yes |\n| y | no |", "compare", ""},
		{"How", "1. One\n2. Two", "steps", ""},
		{"Why", "- ![](a.svg) Fast\n- ![](b.svg) Small", "icons", ""},
		{"Why", "- Fast\n- Small", "list", ""},
		{"Said", "> Words.", "quote", ""},
		{"Look", "![A cat](cat.jpg)", "image", ""},
		{"Fuentes", "- A book", "sources", ""},
		{"Note", "Just text.", "text", ""},
	} {
		if k, ch := GuessIGKind(c.title, c.body); k != c.kind || ch != c.chart {
			t.Errorf("%q: got %s %s, want %s %s", c.title, k, ch, c.kind, c.chart)
		}
	}
}

func TestIGNumber(t *testing.T) {
	for s, want := range map[string]float64{"54": 54, "12.5%": 12.5, "$1,200": 1200, "2.4 s": 2.4, "−3": -3, "**7**": 7} {
		if v, ok := IGNumber(s); !ok || v != want {
			t.Errorf("%q: %v %v, want %v", s, v, ok, want)
		}
	}
	for _, s := range []string{"n/a", "about 4 or 5", "", "4 to 5"} {
		if _, ok := IGNumber(s); ok {
			t.Errorf("%q read as a number", s)
		}
	}
}

func TestInfographicProblems(t *testing.T) {
	_, probs := ParseInfographic("## A · chart=pie\n| a | b |\n|---|---|\n| x | 1 |\n\n## B · chart=bar\n| a | b |\n|---|---|\n| x | lots |\n\n## C · text\n\n## D · shiny\nx\n\n## E · compare\nno table\n")
	var msgs []string
	for _, p := range probs {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{`unknown chart "pie"`, `"lots" is not a number`, `block "C": empty block`, `compare needs a Markdown table`} {
		if !strings.Contains(all, want) {
			t.Errorf("no problem %q in:\n%s", want, all)
		}
	}
	// "D · shiny": shiny is not an option, so it stays in the title
	ig, _ := ParseInfographic("## D · shiny\nx\n")
	if ig.Blocks[0].Title != "D · shiny" {
		t.Errorf("title %q", ig.Blocks[0].Title)
	}
	// a part that starts with an option is options: the bad one is reported
	_, probs = ParseInfographic("## D · wide shiny\nx\n")
	if len(probs) != 1 || !strings.Contains(probs[0].Msg, `unknown block option "shiny"`) {
		t.Errorf("problems %v", probs)
	}
}

func TestInfographicFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.md")
	os.WriteFile(p, []byte("---\ntype: infographic\nlayout: grid\naspect: 16:9\ntitle: X\n---\n\n## 54% · stat\nof visits\n\nNotes: Half of our visits. [CLICK]\n"), 0o644)
	d, err := Parse(p)
	if err != nil || len(d.Problems) != 0 || len(d.Warnings) != 0 {
		t.Fatalf("%v %v %v", err, d.Problems, d.Warnings)
	}
	if d.IsDoc() || len(d.Slides) != 1 || !strings.HasPrefix(d.Slides[0].Body, "~~~~~~~~ infographic page") || d.Slides[0].Notes == "" {
		t.Fatalf("slide %+v", d.Slides[0])
	}
	os.WriteFile(p, []byte("---\ntype: infographic\norientation: portrait\naspect: a4\nlayout: bento\n---\n\n## 54% · stat\nx\n"), 0o644)
	d, _ = Parse(p)
	if !d.IsDoc() || len(d.Problems) != 1 || !strings.Contains(d.Problems[0].Msg, `infographic layout "bento"`) {
		t.Fatalf("portrait: doc %v, problems %v", d.IsDoc(), d.Problems)
	}
	os.WriteFile(p, []byte("---\ntitle: A deck\naspect: a4\n---\n\n# A deck\n"), 0o644)
	if d, _ = Parse(p); len(d.Problems) != 1 {
		t.Fatalf("a deck takes no a4: %v", d.Problems)
	}
}
