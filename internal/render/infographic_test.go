package render

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestIGScale(t *testing.T) {
	for _, c := range []struct{ v, top, want float64 }{{920, 920, 100}, {460, 920, 50}, {0, 10, 0}, {-3, 10, 0}, {5, 0, 0}, {20, 10, 100}} {
		if got := IGScale(c.v, c.top); got != c.want {
			t.Errorf("IGScale(%v, %v) = %v, want %v", c.v, c.top, got, c.want)
		}
	}
	if got := igPercent((0.6 - 2.4) / 2.4 * 100); got != "−75%" {
		t.Errorf("change %q", got)
	}
}

func blocks(spec ...[2]int) []*deck.IGBlock {
	var out []*deck.IGBlock
	for _, s := range spec {
		out = append(out, &deck.IGBlock{Span: s[0], Rows: s[1]})
	}
	return out
}

func TestIGPlace(t *testing.T) {
	// a wide block, then three singles on 3 columns: no holes
	pos, rows, holes := igPlace(blocks([2]int{2, 1}, [2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}), 3, "grid")
	if rows != 2 || holes != 0 || pos[0] != [4]int{1, 1, 1, 2} || pos[1] != [4]int{1, 3, 1, 1} {
		t.Fatalf("pos %v rows %d holes %d", pos, rows, holes)
	}
	if pos[4] != [4]int{2, 3, 1, 1} {
		t.Errorf("last block %v", pos[4])
	}
	// a hole at the end of a row: the block on its left widens
	pos, _, _ = igPlace(blocks([2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}), 3, "grid")
	if pos[3] != [4]int{2, 1, 1, 3} {
		t.Errorf("widened %v", pos[3])
	}
	// columns: filled down, then across
	pos, rows, _ = igPlace(blocks([2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}), 2, "columns")
	if rows != 2 || pos[2] != [4]int{1, 2, 2, 1} {
		t.Errorf("columns %v rows %d", pos, rows)
	}
	if c := igCols(blocks([2]int{2, 1}, [2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1}, [2]int{2, 1}, [2]int{1, 1}, [2]int{1, 1}), "grid"); c != 3 && c != 4 {
		t.Errorf("columns for 9 cells: %d", c)
	}
}

func TestInfographicRender(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ig.md")
	src := "---\ntype: infographic\nsteps: on\ntitle: Load\nsources: Our own data\n---\n\n## 2.4 s · stat\nbefore\n\n## Share · chart=bar\n| a | n |\n|---|---|\n| x | 50 |\n| y | 100 |\n\n## Bad · chart=line\n| a | n |\n|---|---|\n| x | lots |\n\nNotes: Two point four seconds. [CLICK] [CLICK] [CLICK]\n"
	os.WriteFile(p, []byte(src), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	page := string(res.HTML)
	for _, want := range []string{`class="si-slide si-content si-ig-slide"`, `data-orient="landscape"`, `<h2 class="si-ig-title">Load</h2>`,
		`class="si-ig-block si-step" data-kind="stat" data-step="1"`, `data-step="3"`, `width="50"`, `width="100"`, `Our own data`, `.si-ig-grid{`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0].Msg, `"lots" is not a number`) || res.Problems[0].Line != 17 {
		t.Errorf("problems %v", res.Problems)
	}
	// portrait: a page of its own, no build steps
	os.WriteFile(p, []byte(strings.Replace(src, "title: Load", "title: Load\norientation: portrait", 1)), 0o644)
	d, _ = deck.Parse(p)
	page = string(Deck(d, Options{}).HTML)
	if !strings.Contains(page, `class="si-doc si-ig-page"`) || !strings.Contains(page, `data-orient="portrait"`) || strings.Contains(page, `si-step" data-step`) {
		t.Errorf("portrait page wrong")
	}
}

// TestInfographicFit: the page carries the fit pass, and si check warns
// about a block with more text than its cell holds at the floor.
func TestInfographicFit(t *testing.T) {
	long := strings.Repeat("Every request first passes the edge cache, then the gateway checks the token. ", 30)
	for _, c := range []struct {
		body string
		warn bool
	}{
		{"## Short · text\nA line.\n\n## 3× · stat\nfaster\n\n## Two · text\nAnother.\n", false},
		{"## Long · text\n" + long + "\n\n## 3× · stat\nfaster\n\n## A · text\nx\n\n## B · text\ny\n\n## C · text\nz\n\n## D · text\nw\n\n## E · text\nv\n\n## F · text\nu\n", true},
	} {
		dir := t.TempDir()
		p := filepath.Join(dir, "ig.md")
		os.WriteFile(p, []byte("---\ntype: infographic\ntitle: Fit\n---\n\n"+c.body), 0o644)
		d, err := deck.Parse(p)
		if err != nil {
			t.Fatal(err)
		}
		res := Deck(d, Options{})
		if !strings.Contains(string(res.HTML), "window.siInfographicFit") {
			t.Error("page lacks the fit pass")
		}
		warned := false
		for _, w := range res.Warnings {
			warned = warned || strings.Contains(w.Msg, "holds more text than its cell")
		}
		if warned != c.warn {
			t.Errorf("warned %v, want %v: %v", warned, c.warn, res.Warnings)
		}
	}
}

func TestIGFlowLayout(t *testing.T) {
	f, _ := deck.ParseIGFlow("Client -> LB -> API -> Cache\nAPI -> DB\nLB -> API2 -> DB\nDB -[ack]-> API", 1)
	for _, dir := range []string{"LR", "TB"} {
		g := igFlowGraph(f, dir)
		for i := range g.nodes { // no two boxes overlap
			for j := i + 1; j < len(g.nodes); j++ {
				if g.tlx[i] < g.tlx[j]+sBoxW && g.tlx[j] < g.tlx[i]+sBoxW && g.tly[i] < g.tly[j]+sBoxH && g.tly[j] < g.tly[i]+sBoxH {
					t.Errorf("%s: boxes %s and %s overlap", dir, g.nodes[i].id, g.nodes[j].id)
				}
			}
			if g.tlx[i] < 0 || g.tly[i] < 0 || g.tlx[i]+sBoxW > g.vbW || g.tly[i]+sBoxH > g.vbH {
				t.Errorf("%s: box %s outside the viewBox", dir, g.nodes[i].id)
			}
		}
		on := func(n int, x, y float64) bool { // a point on box n's border
			l, tp := float64(g.tlx[n]), float64(g.tly[n])
			in := x >= l-1 && x <= l+sBoxW+1 && y >= tp-1 && y <= tp+sBoxH+1
			edge := math.Abs(x-l) < 1.5 || math.Abs(x-l-sBoxW) < 1.5 || math.Abs(y-tp) < 1.5 || math.Abs(y-tp-sBoxH) < 1.5
			return in && edge
		}
		for k, e := range f.Edges { // every arrow ends on its target's border
			s := igEdgeGeom(g, e.From, e.To, 0)
			if !on(e.To, s.hx, s.hy) {
				t.Errorf("%s: arrow %d ends at %.0f,%.0f, not on %s", dir, k, s.hx, s.hy, f.Nodes[e.To])
			}
		}
		if g.depth[g.idx["DB"]] <= g.depth[g.idx["API"]] {
			t.Errorf("%s: the ack (a cycle) moved the database before the API", dir)
		}
	}
	if back := igBackEdges(f); !back[len(f.Edges)-1] || len(back) != 1 {
		t.Errorf("back edges %v", back)
	}
	ev := igFlowEvents(f)
	if ev[0] != (igFlowEvent{0, -1}) || ev[1] != (igFlowEvent{-1, 0}) || ev[2] != (igFlowEvent{1, -1}) || len(ev) != len(f.Nodes)+len(f.Edges) {
		t.Errorf("events %v", ev)
	}
}

func TestIGSchedule(t *testing.T) {
	ig, _ := deck.ParseInfographic("## 54% · stat\nx\n\n## Path · flow\nA -> B -> C\n\n## How · steps\n1. a\n2. b\n\n## Late · text · at=9s\nwords\n\n## After · quote\n> q\n")
	cues, end, total := igSchedule(ig.Blocks, false, 0)
	want := []string{"count", "draw", "light", "rise", "fade"}
	for i, c := range cues {
		if c.Enter != want[i] {
			t.Errorf("block %d enters %s, want %s", i, c.Enter, want[i])
		}
		if i > 0 && i != 3 && c.At <= cues[i-1].At {
			t.Errorf("block %d starts at %v, not after %v", i, c.At, cues[i-1].At)
		}
	}
	if cues[0].At != igStart || cues[3].At != 9 || cues[4].At <= 9 {
		t.Errorf("times %+v", cues)
	}
	if total <= end || end < cues[4].At+cues[4].Dur {
		t.Errorf("end %v total %v", end, total)
	}
	if _, _, tot := igSchedule(ig.Blocks, true, 12); tot != 12 {
		t.Errorf("duration: given %v", tot)
	}
}

func TestAnimatedInfographicRender(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ig.md")
	os.WriteFile(p, []byte("---\ntype: infographic\naspect: phone\nanimate: loop\nduration: 4s\n---\n\n## 1,200 · stat\nmessages\n\n## Path · flow\nA -[x]-> B -> C\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Document(d, Options{Media: MediaInline})
	h := string(res.HTML)
	for _, w := range []string{`data-fixed="tall"`, `data-animate="loop" data-duration="4"`, `data-enter="count"`, `<span class="si-ig-cnt">1,200</span>`, `class="si-igf-arrow"`, `class="si-igf-pkt"`, "--_ratio:9/19.5", "grid-template-rows:", "siIG"} {
		if !strings.Contains(h, w) {
			t.Errorf("page lacks %s", w)
		}
	}
	if len(res.Warnings) == 0 || !strings.Contains(strings.Join(func() []string {
		var s []string
		for _, w := range res.Warnings {
			s = append(s, w.Msg)
		}
		return s
	}(), "|"), "longer than duration") {
		t.Errorf("want a warning for blocks past duration: %v", res.Warnings)
	}
	if IGGIFEstimate(720, 1560, 180, 15) < igGIFBudget || IGGIFEstimate(720, 1560, 8, 15) > igGIFBudget {
		t.Error("GIF estimate: 180 s should be over 8 MB, 8 s under")
	}
}
