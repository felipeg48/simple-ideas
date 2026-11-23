package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

var (
	docMainRE = regexp.MustCompile(`(?s)<main class="si-doc-body">.*?</main>`)
	svgRE     = regexp.MustCompile(`(?s)<svg.*?</svg>`)
)

// TestArchitectureAndStatusGolden renders the two examples and compares
// their page bodies with testdata/{architecture,status}.golden.html (SVGs
// left out: the flow diagram has its own tests).
func TestArchitectureAndStatusGolden(t *testing.T) {
	for _, kind := range []string{"architecture", "status"} {
		d, err := deck.Parse(filepath.Join("..", "..", "examples", kind, "labs.md"))
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Problems) > 0 {
			t.Fatalf("%s: %v", kind, d.Problems)
		}
		res := Document(d, Options{})
		if len(res.Problems) > 0 {
			t.Fatalf("%s problems: %v", kind, res.Problems)
		}
		page := string(res.HTML)
		if !strings.Contains(page, `data-si-doc="`+kind+`"`) || strings.Contains(page, `class="si-doc-toc"`) {
			t.Errorf("%s: want data-si-doc and no contents column", kind)
		}
		got := svgRE.ReplaceAllString(docMainRE.FindString(page), "<svg/>") + "\n"
		golden := filepath.Join("testdata", kind+".golden.html")
		if *update {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%v (run with -update to create it)", err)
		}
		if got != string(want) {
			t.Errorf("%s golden mismatch (run go test ./internal/render -update if intended)\n%s", kind, got)
		}
	}
}

func renderTypedDoc(t *testing.T, src string) (string, Result) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Document(d, Options{})
	return string(res.HTML), res
}

func TestArchitectureBlocks(t *testing.T) {
	page, res := renderTypedDoc(t, "---\ntype: architecture\ntitle: T\nnumber: off\n---\n\n## One\n\nThe lead.\n\nNot the lead.\n\n"+
		"- **A** — a\n- plain item\n\n**Order** — what was bought\n**Cart** — what may be bought\n\n```flow\nA --> B\n```\n\n## Phase plan\n\n1. First (now)\n2. Second\n")
	for _, want := range []string{`<p class="si-arch-lead">The lead.</p>`, "<p>Not the lead.</p>", `<dl class="si-arch-defs"><div><dt>Order</dt><dd>what was bought</dd></div>`,
		`<ol class="si-arch-phases">`, `<li class="si-arch-now">First`, `<figure class="si-figure"><svg`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	if main := docMainRE.FindString(page); strings.Contains(main, `"si-arch-n"`) || strings.Contains(main, "si-arch-cards") {
		t.Errorf("number: off numbers no section; a half-card list stays a list:\n%s", main)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0].Msg, "1 of 2 items") {
		t.Errorf("warnings = %v, want the half-card list", res.Warnings)
	}
}

func TestStatusBoardMarks(t *testing.T) {
	page, _ := renderTypedDoc(t, "---\ntype: status-board\ntitle: B\ntiles:\n  - num: \"{pct}\"; lbl: Done; sub: {open} open\n---\n\n## A\n\n### Done\n- x `v1.2 · abc1234`\n- y `improvement`\n\n### Missing\n- z `P0` `gated`\n\n## Next ideas\n\n- not counted\n")
	for _, want := range []string{`data-si-doc="status"`, `<div class="si-status-num">67%</div>`, `<div class="si-status-sub">1 open</div>`,
		`<span class="si-status-code">v1.2 · abc1234</span>`, `<span class="si-status-badge" data-kind="imp">improvement</span>`,
		`<span class="si-status-tag">P0</span> <span class="si-status-badge" data-kind="gate">gated</span>`, `<p class="si-status-empty">`,
		`style="width:66.67%"`, `aria-label="A: 67% done; 2 done, 1 missing, 0 improve"`, `<section class="si-status-note"><h2 id="next-ideas">Next ideas</h2>`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}
