package mindmap

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
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

func TestExportXMind8(t *testing.T) {
	raw, _ := os.ReadFile("../../examples/mindmap/payments.md")
	d := Parse(string(raw))
	out, err := Export(d, "xmind8", "")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range z.File {
		r, _ := f.Open()
		files[f.Name], _ = io.ReadAll(r)
		r.Close()
	}
	// every part parses, in its XMind 8 namespace
	for name, want := range map[string]string{"content.xml": "urn:xmind:xmap:xmlns:content:2.0 xmap-content",
		"meta.xml": "urn:xmind:xmap:xmlns:meta:2.0 meta", "META-INF/manifest.xml": "urn:xmind:xmap:xmlns:manifest:1.0 manifest"} {
		var root struct{ XMLName xml.Name }
		if err := xml.Unmarshal(files[name], &root); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := root.XMLName.Space + " " + root.XMLName.Local; got != want {
			t.Errorf("%s: root %q, want %q", name, got, want)
		}
	}
	var c struct {
		Sheets []struct {
			ID    string `xml:"id,attr"`
			Title string `xml:"title"`
			Topic struct {
				ID       string `xml:"id,attr"`
				Title    string `xml:"title"`
				Children struct {
					Topics []struct {
						Type  string `xml:"type,attr"`
						Topic []struct {
							Title  string   `xml:"title"`
							Href   string   `xml:"http://www.w3.org/1999/xlink href,attr"`
							Labels []string `xml:"labels>label"`
						} `xml:"topic"`
					} `xml:"topics"`
				} `xml:"children"`
			} `xml:"topic"`
		} `xml:"sheet"`
	}
	if err := xml.Unmarshal(files["content.xml"], &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Sheets) != 2 || c.Sheets[1].Title != "Refunds" || c.Sheets[1].Topic.ID == "" {
		t.Fatalf("sheets: %+v", c.Sheets)
	}
	kids := c.Sheets[0].Topic.Children.Topics
	if len(kids) != 1 || kids[0].Type != "attached" {
		t.Fatalf("children: %+v", kids)
	}
	links := 0
	for _, k := range kids[0].Topic {
		if k.Title == "Refunds" {
			links++
			if k.Href != "xmind:#"+c.Sheets[1].Topic.ID {
				t.Errorf("→ #refunds is %q", k.Href)
			}
		}
	}
	if links != 1 {
		t.Error("no Refunds topic on the first sheet")
	}
	if !bytes.Contains(files["content.xml"], []byte("<label>0.82</label>")) {
		t.Error("the badge is not a label")
	}
	back, err := Import("x.xmind", out)
	if err != nil {
		t.Fatal(err)
	}
	s := back.String()
	for _, want := range []string{"# Payment", "- Fraud score", "  - Model {badge=0.82}", "- Refunds → #refunds", "# Refunds"} {
		if !strings.Contains(s, want) {
			t.Errorf("round trip lost %q:\n%s", want, s)
		}
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

func TestCodeWindowRoundTrip(t *testing.T) {
	src := "# Plan\n\n- Retry\n  ```go window=mac title=\"retry.go\"\n  for try := 1; try <= 5; try++ {}\n  ```\n"
	if got := Parse(src).String(); got != src {
		t.Fatalf("round trip:\n%q\n%q", src, got)
	}
}

func TestCopyCutPaste(t *testing.T) {
	src := "# Plan\n\n- A\n  - A1 {badge=x}\n  - A2\n- B\n- C → #other\n"
	d := Parse(src)
	got, err := d.Branches(0, [][]int{{2}, {0}, {0, 1}}) // A2 lies inside A: once
	if err != nil || got != "- A\n  - A1 {badge=x}\n  - A2\n- C → #other\n" {
		t.Fatalf("copy: %q %v", got, err)
	}
	if _, err := d.Apply(Op{Kind: "delete-many", Paths: [][]int{{0, 1}, {0}, {2}}}); err != nil {
		t.Fatal(err)
	}
	if s := d.String(); s != "# Plan\n\n- B\n" {
		t.Fatalf("cut:\n%s", s)
	}
	sel, err := d.Apply(Op{Kind: "paste", Path: []int{0}, Text: got})
	if err != nil || fmt.Sprint(sel.Path) != "[0 0]" {
		t.Fatalf("paste: %v %v", sel, err)
	}
	if s := d.String(); s != "# Plan\n\n- B\n  - A\n    - A1 {badge=x}\n    - A2\n  - C → #other\n" {
		t.Fatalf("paste under B:\n%s", s)
	}
	if _, err := d.Apply(Op{Kind: "paste", Text: "just words"}); err == nil {
		t.Error("pasting text that is not a list is refused")
	}
}
