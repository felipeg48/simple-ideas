package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseText(t *testing.T, src string) *Deck {
	t.Helper()
	p := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestStatusCounts(t *testing.T) {
	for _, c := range []struct {
		c      StatusCounts
		pct    int
		shares [3]float64
	}{
		{StatusCounts{15, 17, 11}, 47, [3]float64{34.88, 39.53, 25.58}}, // the reference board
		{StatusCounts{1, 2, 0}, 33, [3]float64{33.33, 66.67, 0}},
		{StatusCounts{0, 0, 3}, 100, [3]float64{0, 0, 100}}, // nothing missing: done
		{StatusCounts{}, 100, [3]float64{}},
	} {
		if p := c.c.Percent(); p != c.pct {
			t.Errorf("%v: percent %d, want %d", c.c, p, c.pct)
		}
		if s := c.c.Shares(); s != c.shares {
			t.Errorf("%v: shares %v, want %v", c.c, s, c.shares)
		}
	}
}

func TestStatusParse(t *testing.T) {
	d := parseText(t, "---\ntype: status-board\ntitle: B\nupdated: 2026-11-22\nmeta:\n  - git: on main | ok\n  - plain words\ntiles:\n  - num: \"25\"; lbl: Sessions; sub: per VM\n  - gate: Passed; tone: ok; lbl: Gate\n---\n\n"+
		"Intro words.\n\n## Control Plane\n\ndockyard\n\n### Done\n- API\n- Reaper `482434a`\n  continued\n\n### Missing\n- Terminal `P2`\n\n### Improve\n- Pool `partial`\n\n## Shipped\n- Gate `482434a`\n\n## Notes\n\nDone means merged.\n")
	if len(d.Problems) > 0 {
		t.Fatal(d.Problems)
	}
	st := d.Status
	if d.Meta.Type != "status" || st == nil {
		t.Fatalf("type %q, status %v", d.Meta.Type, st)
	}
	if len(st.Pills) != 2 || st.Pills[0] != (StatusPill{"git", "on main", "ok"}) || st.Pills[1] != (StatusPill{"", "plain words", "plain"}) {
		t.Errorf("pills %+v", st.Pills)
	}
	if len(st.Tiles) != 2 || st.Tiles[0].Num != "25" || !st.Tiles[1].Gate || st.Tiles[1].Tone != "ok" {
		t.Errorf("tiles %+v", st.Tiles)
	}
	if len(st.Areas) != 1 {
		t.Fatalf("areas %+v", st.Areas)
	}
	a := st.Areas[0]
	if a.Title != "Control Plane" || a.Scope != "dockyard" || a.Counts() != (StatusCounts{2, 1, 1}) {
		t.Errorf("area %+v", a)
	}
	if it := a.Cols[0][1]; it.Text != "Reaper `482434a` continued" && it.Text != "Reaper" {
		t.Errorf("item %+v", it)
	}
	if m := a.Cols[1][0].Marks; len(m) != 1 || m[0] != (StatusMark{"tag", "P2"}) {
		t.Errorf("marks %+v", m)
	}
	if m := a.Cols[2][0].Marks; len(m) != 1 || m[0].Kind != "part" {
		t.Errorf("marks %+v", m)
	}
	if st.Shipped == nil || len(st.Shipped.Cols[0]) != 1 || st.Shipped.Cols[0][0].Marks[0].Text != "482434a" {
		t.Errorf("shipped %+v", st.Shipped)
	}
	if len(st.Notes) != 1 || st.Notes[0].Body != "Done means merged." || st.Intro.Body != "Intro words." {
		t.Errorf("notes %+v intro %+v", st.Notes, st.Intro)
	}
}

func TestStatusProblems(t *testing.T) {
	for name, c := range map[string]struct{ src, want string }{
		"unknown column": {"## A\n\n### Doing\n- x\n", "unknown column ### Doing"},
		"item before":    {"## A\n\n- x\n\n### Done\n- y\n", "a list item before ### Done"},
		"twice":          {"## A\n\n### Done\n- x\n\n### Done\n- y\n", "### Done is written twice"},
		"no areas":       {"Just words.\n", "needs at least one ## area"},
		"bad tone":       {"## A\n\n### Done\n- x\n", "unknown tone"},
		"bad tile":       {"## A\n\n### Done\n- x\n", "unknown key"},
	} {
		fm := "---\ntype: status\ntitle: B\n"
		if name == "bad tone" {
			fm += "meta:\n  - a: b | green\n"
		}
		if name == "bad tile" {
			fm += "tiles:\n  - num: 1; colour: red\n"
		}
		d := parseText(t, fm+"---\n\n"+c.src)
		var all []string
		for _, p := range d.Problems {
			all = append(all, p.Msg)
		}
		if !strings.Contains(strings.Join(all, "\n"), c.want) {
			t.Errorf("%s: problems %v, want %q", name, all, c.want)
		}
	}
}

func TestStatusListsOnlyUnderListKeys(t *testing.T) {
	// a deck that starts with --- and a list is not front matter
	d := parseText(t, "---\n- a\n- b\n---\n\n# Next\n")
	if d.Meta.Type != "deck" || len(d.Meta.Lists) != 0 {
		t.Errorf("type %q lists %v", d.Meta.Type, d.Meta.Lists)
	}
	d = parseText(t, "---\ntitle: A\nmeta:\n  - x\n---\n\n# A\n")
	if len(d.Warnings) == 0 || !strings.Contains(d.Warnings[0].Msg, "status boards") {
		t.Errorf("warnings %v", d.Warnings)
	}
}
