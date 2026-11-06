package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

var mmDataRE = regexp.MustCompile(`(?s)<script type="application/json" class="si-mm-data">(.*?)</script>`)

type mmData struct {
	Start string   `json:"start"`
	Maps  []*mmMap `json:"maps"`
}

// renderFiles writes files into a temp dir and renders main.
func renderFiles(t *testing.T, files map[string]string, main string) (Result, mmData) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	d, err := deck.Parse(filepath.Join(dir, main))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	var data mmData
	if m := mmDataRE.FindSubmatch(res.HTML); m != nil {
		if err := json.Unmarshal(m[1], &data); err != nil {
			t.Fatal(err)
		}
	}
	return res, data
}

func problemsText(res Result) string {
	var s []string
	for _, p := range res.Problems {
		s = append(s, p.Msg)
	}
	return strings.Join(s, "\n")
}

func TestMindmapPage(t *testing.T) {
	png := "\x89PNG\r\n\x1a\n"
	res, data := renderFiles(t, map[string]string{
		"pic.png": png,
		"talk.md": "# Talk\n",
		"m.md": "---\ntype: mindmap\nlayout: radial\nlines: tapered\nwatermark: Northwind\n---\n# Order\n\n" +
			"- Checkout {badge=Thu_night collapsed}\n  - Payment → pay.md\n  - Moved {pos=40,-12 branch-pos=1.5,2}\n" +
			"- Refunds → #refunds\n- Slides → slides: talk.md#5\n- Docs → https://example.com/x\n- Talk → talk.md\n" +
			"- Dusk ![a van](pic.png)\n- [!sticky] Ask ops.\n- Retry\n  ```go\n  for i := 0; i < 3; i++ {}\n  ```\n\n---\n\n# Refunds {#refunds}\n\n- Partial\n",
		"pay.md": "---\ntype: mindmap\n---\n# Payment\n\n- Back home → m.md\n",
	}, "m.md")
	if len(res.Problems) > 0 {
		t.Fatal(problemsText(res))
	}
	h := string(res.HTML)
	for _, want := range []string{`data-layout="radial"`, `data-lines="tapered"`, `data-full`, `data-wm="Northwind"`, `data-wm-pos="corner"`, `class="si-mm-list"`, "si mind maps"} {
		if !strings.Contains(h, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	ids := []string{}
	for _, m := range data.Maps {
		ids = append(ids, m.ID)
	}
	if data.Start != "order" || strings.Join(ids, ",") != "order,refunds,payment" {
		t.Fatalf("start %q maps %v", data.Start, ids)
	}
	k := data.Maps[0].Kids
	if k[0].Badge != "Thu night" || !k[0].Collapsed || k[0].Kids[0].Link != "payment" {
		t.Errorf("checkout: %+v %+v", k[0], k[0].Kids[0])
	}
	if p, b := k[0].Kids[1].Pos, k[0].Kids[1].BPos; p == nil || *p != [2]float64{40, -12} || b == nil || *b != [2]float64{1.5, 2} {
		t.Errorf("pos %v bpos %v", p, b)
	}
	if k[1].Link != "refunds" || k[2].Href != "talk.html#5" || k[2].Chip != "slide 5" || k[3].Chip != "link" || k[4].Href != "talk.html" || k[4].Chip != "page" {
		t.Errorf("links: %+v %+v %+v %+v", k[1], k[2], k[3], k[4])
	}
	if !strings.HasPrefix(k[5].Img, "data:image/png") || k[5].T != "Dusk" || !k[6].Sticky || k[6].T != "Ask ops." {
		t.Errorf("picture or sticky: %+v %+v", k[5], k[6])
	}
	if c := k[7].Kids[0].Code; c == nil || c.Lang != "go" || c.Lines != 1 || !strings.Contains(c.HTML, "si-tk-k") {
		t.Errorf("code: %+v", k[7].Kids)
	}
	if back := data.Maps[2].Kids[0].Link; back != "order" { // a cycle back to the page resolves to its first map
		t.Errorf("cycle link %q", back)
	}
}

func TestMindmapProblems(t *testing.T) {
	res, _ := renderFiles(t, map[string]string{
		"m.md": "---\ntype: mindmap\n---\n# M\n\n- a → nope.md\n- b → #ghost\n- c {size=3}\n- d {pos=x}\n- e ![](gone.png)\n",
	}, "m.md")
	got := problemsText(res)
	for _, want := range []string{"missing file: nope.md", "no map #ghost", `unknown topic option "size"`, "pos=x needs two numbers", "missing file: gone.png"} {
		if !strings.Contains(got, want) {
			t.Errorf("problems lack %q:\n%s", want, got)
		}
	}
}

func TestMindmapFence(t *testing.T) {
	res, data := renderFiles(t, map[string]string{
		"d.md": "# Deck\n\n---\n\n## Map\n\n```mindmap layout=org theme=ink\n# Order\n- a\n  - b\n```\n",
	}, "d.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 || data.Start != "order" || !strings.Contains(h, `data-layout="org"`) || !strings.Contains(h, `data-mm-theme="ink"`) || regexp.MustCompile(`<div class="si-mindmap"[^>]*data-full`).MatchString(h) {
		t.Fatalf("fence: %s start=%q", problemsText(res), data.Start)
	}
	if !strings.Contains(h, ".si-mindmap{") || !strings.Contains(h, "si mind maps") {
		t.Error("a deck with a mindmap fence must load mindmap.css and mindmap.js")
	}
	res, _ = renderFiles(t, map[string]string{"d.md": "# Deck\n\n---\n\n## Map\n\n```mindmap layout=spiral colour=red\n- a\n```\n"}, "d.md")
	got := problemsText(res)
	if !strings.Contains(got, `unknown mindmap layout "spiral"`) || !strings.Contains(got, `unknown fence option "colour"`) {
		t.Errorf("fence options: %s", got)
	}
	res, _ = renderFiles(t, map[string]string{"d.md": "# Deck\n\n---\n\n## Plain\n\ntext\n"}, "d.md")
	if strings.Contains(string(res.HTML), "si mind maps") {
		t.Error("a deck without a map must not load mindmap.js")
	}
}

// TestMindmap3D: three.js and the 3D renderer are inlined only into pages with
// a map that starts in 3D; 2D maps keep their markup and their size.
func TestMindmap3D(t *testing.T) {
	const three = "Copyright 2010-2021 Three.js Authors"
	res, _ := renderFiles(t, map[string]string{
		"d.md": "# Deck\n\n---\n\n## Map\n\n```mindmap view=3d camera=perspective floors=off\n# Order\n- a\n```\n",
	}, "d.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 || !strings.Contains(h, `data-view="3d"`) || !strings.Contains(h, `data-camera="perspective"`) || !strings.Contains(h, `data-floors="off"`) {
		t.Fatalf("3D fence: %s", problemsText(res))
	}
	if !strings.Contains(h, three) || !strings.Contains(h, "SIMindmap3D") || strings.Index(h, "SIMindmap3D") > strings.Index(h, "si mind maps:") {
		t.Error("a 3D map must inline three.js and mindmap3d.js before mindmap.js")
	}
	res, _ = renderFiles(t, map[string]string{"d.md": "# Deck\n\n---\n\n## Map\n\n```mindmap\n# Order\n- a\n```\n"}, "d.md")
	if h := string(res.HTML); strings.Contains(h, three) || strings.Contains(h, `data-view="3d"`) {
		t.Error("a 2D map must not carry three.js or a data-view attribute")
	}
	res, _ = renderFiles(t, map[string]string{"m.md": "---\ntype: mindmap\nview: 3d\n---\n# M\n\n- a\n"}, "m.md")
	if h := string(res.HTML); !strings.Contains(h, three) || !strings.Contains(h, `data-view="3d"`) {
		t.Error("a view: 3d page must inline three.js")
	}
	res, _ = renderFiles(t, map[string]string{"d.md": "# Deck\n\n---\n\n## Map\n\n```mindmap view=4d camera=fisheye\n- a\n```\n"}, "d.md")
	if got := problemsText(res); !strings.Contains(got, `unknown mindmap view "4d"`) || !strings.Contains(got, `unknown mindmap camera "fisheye"`) {
		t.Errorf("3D fence options: %s", got)
	}
}
