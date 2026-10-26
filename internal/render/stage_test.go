package render

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenStageChain(t *testing.T) {
	svg, _, err := genStage("dir: LR\nStart --> Validate --> Done\ndot Start..Done 1s", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`viewBox="0 0`, `class="si-stage s1"`,
		`>Start</text>`, `>Validate</text>`, `>Done</text>`,
		`class="dot a0"`, `animation:si-stage-move 1s linear infinite`,
		`var(--si-surface`, `prefers-reduced-motion`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("missing %q", want)
		}
	}
	if n := strings.Count(svg, `class="edge"`); n != 2 {
		t.Errorf("edges = %d, want 2", n)
	}
	if strings.Contains(stylePart(svg), "<") {
		t.Error("style block must not contain a less-than sign")
	}
}

func TestGenStageLabels(t *testing.T) {
	// node label A[Label], edge label -->|hit|
	svg, _, err := genStage("Cache -->|hit| Fast[Fast path]", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, `>Fast path</text>`) {
		t.Error("node label A[Label] not rendered")
	}
	if !strings.Contains(svg, `class="elbl"`) || !strings.Contains(svg, `>hit</text>`) {
		t.Error("edge label -->|hit| not rendered")
	}
}

func TestGenStageFanOutAndFlow(t *testing.T) {
	// one parent, two children (fan-out) with a flow animation down one branch
	svg, _, err := genStage("Req --> Hit\nReq --> Miss\nflow Req..Miss 1.2s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(svg, `class="node n`); n != 3 {
		t.Errorf("nodes = %d, want 3", n)
	}
	if !strings.Contains(svg, `class="flow a0"`) || !strings.Contains(svg, `animation:si-stage-flow 1.2s`) {
		t.Error("flow animation not emitted")
	}
}

func TestGenStageTB(t *testing.T) {
	lr, _, _ := genStage("A --> B --> C", 1)
	tb, _, _ := genStage("dir: TB\nA --> B --> C", 1)
	// a vertical chain is taller than wide; a horizontal one is wider than tall.
	if vbTall(tb) == vbTall(lr) {
		t.Error("TB and LR should produce different aspect ratios")
	}
	if !vbTall(tb) {
		t.Error("TB chain should be taller than wide")
	}
}

func TestGenStageErrors(t *testing.T) {
	for _, bad := range []string{
		"dir: diagonal\nA --> B",
		"dot A..Z\nA --> B",   // Z unknown
		"flow A..C\nA --> B",  // no route A->C (C unknown)
		"dot nope",            // no A..B
		"A -->|oops B",        // unterminated edge label
		"step Ghost\nA --> B", // step references unknown node
		"",                    // no nodes
	} {
		if _, _, err := genStage(bad, 1); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestGenStageSteps(t *testing.T) {
	svg, ops, err := genStage("A[Alpha] --> B[Beta] --> C[Gamma]\nstep A\nstep ok B\nstep show C", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 3 {
		t.Fatalf("stepOps = %d, want 3: %v", len(ops), ops)
	}
	for _, want := range []string{
		`class="si-stage si-canvas s1"`, // joins the canvas step machinery
		`data-base=""`,
		`+k1s1`, `+k1s3`, // step ops are seq-scoped
		`.s1[data-state~="k1s1"] .n0{stroke:var(--si-accent`, // step 1 highlights node 0
		`.s1[data-state~="k1s2"] .n1{stroke:var(--si-ok`,     // step 2 oks node 1
		`.s1[data-state~="k1s3"] .n2,`,                       // step 3 shows node 2
		`.s1 .n2,.s1 .l2{opacity:0}`,                         // show-target starts hidden
	} {
		if want[0] == '+' {
			found := false
			for _, o := range ops {
				if o == want {
					found = true
				}
			}
			if !found {
				t.Errorf("stepOps missing %q: %v", want, ops)
			}
			continue
		}
		if !strings.Contains(svg, want) {
			t.Errorf("missing %q", want)
		}
	}
	// alert pulses; a stage with no step lines stays a plain figure
	al, _, _ := genStage("A --> B\nstep alert B", 1)
	if !strings.Contains(al, "si-stage-pulse") {
		t.Error("alert op should pulse")
	}
	plain, ops2, _ := genStage("A --> B", 1)
	if len(ops2) != 0 || strings.Contains(plain, "si-canvas") {
		t.Error("unstepped stage should not be a canvas or emit step ops")
	}
}

// --- helpers ---

func stylePart(svg string) string {
	a := strings.Index(svg, "<style>")
	b := strings.Index(svg, "</style>")
	if a < 0 || b < 0 {
		return ""
	}
	return svg[a+len("<style>") : b]
}

// vbTall reports whether the viewBox height exceeds its width.
func vbTall(svg string) bool {
	i := strings.Index(svg, `viewBox="0 0 `)
	if i < 0 {
		return false
	}
	rest := svg[i+len(`viewBox="0 0 `):]
	rest = rest[:strings.IndexByte(rest, '"')]
	f := strings.Fields(rest)
	w, _ := strconv.Atoi(f[0])
	h, _ := strconv.Atoi(f[1])
	return h > w
}
