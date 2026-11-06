package mindmap

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Every example mind map survives a round trip: the model reads back the
// same, and the canonical examples come back byte for byte.
func TestRoundTripExamples(t *testing.T) {
	files, _ := filepath.Glob("../../examples/mindmap/*.md")
	if len(files) == 0 {
		t.Fatal("no example mind maps")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d1 := Parse(string(raw))
		out := d1.String()
		d2 := Parse(out)
		if !reflect.DeepEqual(d1, d2) {
			t.Errorf("%s: model changed after a round trip:\n%s", f, out)
		}
		if out != string(raw) {
			t.Errorf("%s: text changed after a round trip:\n%s", f, out)
		}
	}
}

func TestLabel(t *testing.T) {
	d := Parse("- [!sticky] Dusk ![The vans](a.jpg) {badge=two_words collapsed pos=4,-2 shade=dark} → slides: talk.md#3\n")
	tp := d.Maps()[0].Topics[0]
	if !tp.Sticky || tp.Text != "Dusk" || tp.Img != "a.jpg" || tp.ImgAlt != "The vans" || tp.Badge != "two words" || !tp.Collapsed ||
		*tp.Pos != [2]float64{4, -2} || tp.Attrs != "shade=dark" || tp.Target != "slides: talk.md#3" {
		t.Fatalf("%+v", tp)
	}
	if got := tp.Label(); got != "[!sticky] Dusk ![The vans](a.jpg) {badge=two_words collapsed pos=4,-2 shade=dark} → slides: talk.md#3" {
		t.Fatal(got)
	}
}

func TestEdits(t *testing.T) {
	src := "---\ntype: mindmap\ntitle: T\n---\n# T\n\n- A\n  - A1\n- B\n"
	d := Parse(src)
	step := func(op Op) Sel {
		t.Helper()
		s, err := d.Apply(op)
		if err != nil {
			t.Fatal(op.Kind, err)
		}
		return s
	}
	if s := step(Op{Kind: "add-child", Path: []int{0}, Text: "A2"}); !reflect.DeepEqual(s.Path, []int{0, 1}) {
		t.Fatal(s)
	}
	step(Op{Kind: "add-sibling", Path: []int{0}, Text: "Mid"})
	step(Op{Kind: "rename", Path: []int{2}, Text: "Bee"})
	step(Op{Kind: "set", Path: []int{2}, Field: "badge", Value: "new one"})
	step(Op{Kind: "move", Path: []int{0, 0}, To: []int{2}, Index: -1}) // A1 under Bee
	step(Op{Kind: "pos", Path: []int{1}, X: 12.4, Y: -3})
	step(Op{Kind: "meta", Field: "layout", Value: "right"})
	step(Op{Kind: "up", Path: []int{2}})
	got := d.String()
	want := "---\ntype: mindmap\ntitle: T\nlayout: right\n---\n# T\n\n- A\n  - A2\n- Bee {badge=new_one}\n  - A1\n- Mid {pos=12,-3}\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := d.Apply(Op{Kind: "move", Path: []int{1}, To: []int{1, 0}}); err == nil {
		t.Fatal("a topic moved into its own branch")
	}
	step(Op{Kind: "delete", Path: []int{1}})
	step(Op{Kind: "tidy"})
	if strings.Contains(d.String(), "pos=") || strings.Contains(d.String(), "Bee") {
		t.Fatal(d.String())
	}
}

func TestFence(t *testing.T) {
	src := "# Deck\n\n```go\nx := 1\n```\n\n```mindmap layout=right\n# Plan\n- One\n```\ntail\n"
	f, ok := FenceAt(src, 9)
	if !ok || f.Body != "# Plan\n- One" {
		t.Fatalf("%+v %v", f, ok)
	}
	if _, ok := FenceAt(src, 4); ok {
		t.Fatal("found a mind map in a go fence")
	}
	d := Parse(f.Body)
	d.Apply(Op{Kind: "add-sibling", Path: []int{0}, Text: "Two"})
	out := f.Replace(src, SetInfo(f.Info, "lines", "tapered"), d.String())
	if !strings.Contains(out, "```mindmap layout=right lines=tapered\n# Plan\n\n- One\n- Two\n```\ntail") {
		t.Fatal(out)
	}
	if SetInfo("```mindmap layout=right lines=x", "layout", "") != "```mindmap lines=x" {
		t.Fatal(SetInfo("```mindmap layout=right lines=x", "layout", ""))
	}
}

func TestImport(t *testing.T) {
	for _, f := range []string{"plan.opml", "plan.mm", "plan.mmd"} {
		raw, err := os.ReadFile(filepath.Join("testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		d, err := Import(f, raw)
		if err != nil {
			t.Fatal(f, err)
		}
		got := d.String()
		if !strings.Contains(got, "title: Launch & land\n---\n# Launch & land\n\n- Plan\n  - Budget") || !strings.Contains(got, "\n- Ship\n") {
			t.Errorf("%s:\n%s", f, got)
		}
	}
	if _, err := Import("x.txt", nil); err == nil {
		t.Fatal("imported an unknown format")
	}
}

func TestExportRoundTrip(t *testing.T) {
	raw, _ := os.ReadFile("../../examples/mindmap/payments.md")
	d := Parse(string(raw))
	for _, f := range []struct{ format, ext string }{{"opml", ".opml"}, {"freemind", ".mm"}, {"xmind", ".xmind"}, {"mermaid", ".mmd"}} {
		out, err := Export(d, f.format, "")
		if err != nil {
			t.Fatal(f.format, err)
		}
		back, err := Import("x"+f.ext, out)
		if err != nil {
			t.Fatal(f.format, err)
		}
		s := back.String()
		for _, want := range []string{"# Payment", "- Fraud score", "  - Model", "- Refunds"} {
			if !strings.Contains(s, want) {
				t.Errorf("%s lost %q:\n%s", f.format, want, s)
			}
		}
	}
	if out, _ := Export(d, "xmind", ""); !strings.Contains(string(mustSheets(t, out)), "Refunds") {
		t.Error("the xmind export lost the second map")
	}
}

func mustSheets(t *testing.T, x []byte) []byte {
	back, err := Import("x.xmind", x)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Maps()) != 2 {
		t.Fatalf("%d sheets", len(back.Maps()))
	}
	return []byte(back.String())
}
