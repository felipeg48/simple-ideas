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
	_, probs := ParseInfographic("## A · chart=scatter\n| a | b |\n|---|---|\n| x | 1 |\n\n## B · chart=bar\n| a | b |\n|---|---|\n| x | lots |\n\n## C · text\n\n## D · shiny\nx\n\n## E · compare\nno table\n")
	var msgs []string
	for _, p := range probs {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{`unknown chart "scatter"`, `"lots" is not a number`, `block "C": empty block`, `compare needs a Markdown table`} {
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

func TestInfographicMotionOptions(t *testing.T) {
	src := "## 54% · stat · enter=rise · at=2.5s\nof visits\n\n## Path · flow\n\nClient -[HTTP]-> API -> Cache\nAPI -> Database\n\n## Bad · stat · enter=spin · at=soon\n"
	ig, probs := ParseInfographic(src)
	if b := ig.Blocks[0]; b.Enter != "rise" || b.At != 2.5 {
		t.Errorf("stat motion: %+v", b)
	}
	f := ig.Blocks[1]
	if f.Kind != "flow" || f.Span != 2 || f.At != -1 || f.Flow == nil {
		t.Fatalf("flow block: %+v", f)
	}
	if strings.Join(f.Flow.Nodes, ",") != "Client,API,Cache,Database" || len(f.Flow.Edges) != 3 || f.Flow.Edges[0].Label != "HTTP" || f.Flow.Edges[2] != (IGEdge{1, 3, ""}) {
		t.Errorf("flow: %+v", f.Flow)
	}
	if len(probs) != 2 || probs[0].Line != 9 || !strings.Contains(probs[0].Msg, "enter") || !strings.Contains(probs[1].Msg, "at=") {
		t.Errorf("problems: %+v", probs)
	}
	if k, _ := GuessIGKind("Path", "A -> B"); k != "flow" {
		t.Errorf("guess flow: %s", k)
	}
}

func TestParseIGFlowErrors(t *testing.T) {
	for _, c := range []struct {
		body string
		line int
		msg  string
	}{
		{"A -> B\nA -> ", 11, "box on each side"},
		{"\nA -[HTTP-> B", 11, "not closed"},
		{"Just a box", 10, "no arrows"},
		{"-> B", 10, "box on each side"},
		{"[db] -> B", 10, "box on each side"},
		{"[disk] Orders -> B", 10, "unknown icon [disk]"},
	} {
		_, probs := ParseIGFlow(c.body, 10)
		if len(probs) == 0 || probs[0].Line != c.line || !strings.Contains(probs[0].Msg, c.msg) {
			t.Errorf("%q: %+v", c.body, probs)
		}
	}
	if f, probs := ParseIGFlow("A → B -[ack]-> A", 1); len(probs) != 0 || len(f.Nodes) != 2 || f.Edges[1] != (IGEdge{1, 0, "ack"}) {
		t.Errorf("arrows: %+v %v", f, probs)
	}
	// icons before a name, once is enough, aliases too; boxes meet by name
	f, probs := ParseIGFlow("[user] Client -[HTTP]-> [server] API -> [db] Orders\nAPI -> [Queue] Events\nOrders -> Events", 1)
	if len(probs) != 0 || strings.Join(f.Nodes, ",") != "Client,API,Orders,Events" || strings.Join(f.Icons, ",") != "user,server,database,queue" || len(f.Edges) != 4 {
		t.Errorf("icons: %+v %v", f, probs)
	}
	ig, probs := ParseInfographic("## Path · flow · dir=tb\nA -> B\n\n## Other · flow · dir=up\nA -> B\n")
	if ig.Blocks[0].Dir != "tb" || len(probs) != 1 || !strings.Contains(probs[0].Msg, "dir=up") {
		t.Errorf("dir=: %q %+v", ig.Blocks[0].Dir, probs)
	}
}

func TestIGSizesAndMeta(t *testing.T) {
	for a, want := range map[string][2]float64{"phone": {9, 19.5}, "story": {9, 16}, "portrait": {4, 5}, "square": {1, 1}, "landscape": {16, 9}, "1080x1350": {1080, 1350}} {
		if w, h, ok := IGSize(a); !ok || w != want[0] || h != want[1] {
			t.Errorf("%s: %v %v %v", a, w, h, ok)
		}
	}
	if _, _, ok := IGSize("16:9"); ok {
		t.Error("16:9 is a ratio, not an exact frame")
	}
	if !(Meta{Aspect: "phone"}).IGPortrait() || (Meta{Aspect: "square"}).IGPortrait() || !(Meta{Aspect: "1080x1350"}).IGPortrait() {
		t.Error("orientation from an exact size")
	}
	for s, want := range map[string]float64{"12s": 12, "1.5 s": 1.5, "800ms": 0.8, "3": 3} {
		if v, ok := IGSeconds(s); !ok || v != want {
			t.Errorf("IGSeconds(%q) = %v %v", s, v, ok)
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("---\ntype: infographic\naspect: 1080x1350\nanimate: twirl\nduration: soon\n---\n\n## 1 · stat\n"), 0o644)
	d, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, pr := range d.Problems {
		msgs = append(msgs, pr.Msg)
	}
	all := strings.Join(msgs, "|")
	if !strings.Contains(all, `unknown animate "twirl"`) || !strings.Contains(all, "duration") || strings.Contains(all, "aspect") {
		t.Errorf("problems: %s", all)
	}
}
