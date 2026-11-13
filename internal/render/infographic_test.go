package render

import (
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
