package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

func TestScaffoldCanvas(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	var out, errb bytes.Buffer
	if code := scaffoldCanvas("pipeline", &out, &errb); code != exitOK {
		t.Fatalf("scaffoldCanvas = %d, stderr=%s", code, errb.String())
	}

	svg, err := os.ReadFile(filepath.Join(dir, "pipeline.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svg), "viewBox") || !strings.Contains(string(svg), `class="pipeline"`) {
		t.Error("starter svg missing viewBox or class")
	}
	if strings.Contains(string(svg), "CANVAS") {
		t.Error("placeholder CANVAS not replaced in svg")
	}

	// the demo deck must parse and render as a valid deck with exactly one canvas
	d, err := deck.Parse(filepath.Join(dir, "pipeline.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := render.Deck(d, render.Options{})
	if len(res.Problems) != 0 {
		t.Errorf("demo deck has problems: %v", res.Problems)
	}
	if res.Canvases != 1 {
		t.Errorf("canvases = %d, want 1", res.Canvases)
	}

	// invalid name is rejected, existing files are not overwritten
	if code := scaffoldCanvas("../evil", &out, &errb); code == exitOK {
		t.Error("invalid name should be rejected")
	}
	if code := scaffoldCanvas("pipeline", &out, &errb); code == exitOK {
		t.Error("existing files should not be overwritten")
	}
}
