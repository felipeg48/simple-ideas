package render

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenStageChain(t *testing.T) {
	svg, err := genStage("dir: LR\nStart --> Validate --> Done\ndot Start..Done 1s", 1)
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
	svg, err := genStage("Cache -->|hit| Fast[Fast path]", 1)
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
	svg, err := genStage("Req --> Hit\nReq --> Miss\nflow Req..Miss 1.2s", 1)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(svg, `class="node"`); n != 3 {
		t.Errorf("nodes = %d, want 3", n)
	}
	if !strings.Contains(svg, `class="flow a0"`) || !strings.Contains(svg, `animation:si-stage-flow 1.2s`) {
		t.Error("flow animation not emitted")
	}
}

func TestGenStageTB(t *testing.T) {
	lr, _ := genStage("A --> B --> C", 1)
	tb, _ := genStage("dir: TB\nA --> B --> C", 1)
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
		"dot A..Z\nA --> B",  // Z unknown
		"flow A..C\nA --> B", // no route A->C (C unknown)
		"dot nope",           // no A..B
		"A -->|oops B",       // unterminated edge label
		"",                   // no nodes
	} {
		if _, err := genStage(bad, 1); err == nil {
			t.Errorf("expected error for %q", bad)
		}
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
