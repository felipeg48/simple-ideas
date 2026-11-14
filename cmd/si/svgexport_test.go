package main

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// TestFrameSVG builds a timeline (each layout) and an infographic and checks
// that --svg writes well-formed SVG holding their texts as <text>, with the
// pictures embedded; it also checks the infographic fit pass in the browser
// (a block too full for its cell is lowered to the floor and marked).
func TestFrameSVG(t *testing.T) {
	chrome := findChrome()
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	png := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	pic := "data:image/png;base64," + png
	tl := "---\ntype: timeline\ntitle: Sorting, briefly\nlayout: %s\n---\n\n## Cards\n\n### 1890 · Hollerith tabulator\nPunched cards sorted by machine.\n![A tabulator](" + pic + ")\n\n## Programs\n\n### 1945 · Merge sort\nVon Neumann merges sorted runs.\n"
	for _, layout := range []string{"snake", "eras", "cards"} {
		md := filepath.Join(dir, layout+".md")
		os.WriteFile(md, []byte(strings.Replace(tl, "%s", layout, 1)), 0o644)
		svg := buildSVG(t, chrome, md, dir)
		for _, want := range []string{"Sorting, briefly", "Hollerith tabulator", "Merge sort", "<image ", pic[:40]} {
			if !strings.Contains(svg, want) {
				t.Errorf("%s: svg lacks %q", layout, want)
			}
		}
	}
	long := strings.Repeat("Every request passes the edge cache, then the gateway checks the token. ", 40)
	md := filepath.Join(dir, "ig.md")
	os.WriteFile(md, []byte("---\ntype: infographic\ntitle: The 600 ms checkout\n---\n\n## 2.4 s · stat\nbefore\n\n## Long · text\n"+long+"\n\n## A · text\nx\n\n## B · text\ny\n\n## C · text\nz\n\n## D · text\nw\n\n## E · text\nv\n\n## F · text\nu\n"), 0o644)
	svg := buildSVG(t, chrome, md, dir)
	for _, want := range []string{"The 600 ms checkout", "2.4", "BEFORE", "<text "} {
		if !strings.Contains(strings.ToUpper(svg), strings.ToUpper(want)) {
			t.Errorf("infographic: svg lacks %q", want)
		}
	}
	got, err := frameEval(chrome, "file://"+filepath.Join(dir, "ig.html")+"#1.999",
		`[...document.querySelectorAll('.si-ig-block')].map(b=>(b.style.getPropertyValue('--_fit')||'1')+(b.hasAttribute('data-overflow')?'!':'')).join(' ')`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `0.6!`) || !strings.HasPrefix(got, `"1 `) {
		t.Errorf("fit: block sizes %s, want the long block at the 0.6 floor, marked, and the stat untouched", got)
	}
}

func buildSVG(t *testing.T, chrome, md, dir string) string {
	t.Helper()
	d, err := deck.Parse(md)
	if err != nil {
		t.Fatal(err)
	}
	dest := strings.TrimSuffix(md, ".md") + ".html"
	if err := os.WriteFile(dest, render.Deck(d, render.Options{Media: render.MediaInline}).HTML, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exportFrameSVG(chrome, dest, dir, d, io.Discard); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, d.Name+".svg"))
	if err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("%s: not well-formed: %v", d.Name, err)
		}
	}
	return string(raw)
}
