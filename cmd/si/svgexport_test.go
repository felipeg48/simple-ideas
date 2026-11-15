package main

import (
	"encoding/base64"
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

// TestFrameSVGLooks: gradients, box shadows and web fonts reach the SVG: a
// linear and a radial background as SVG gradients, a shadow as a blurred rect
// under its box, and the @font-face of the family the text uses (only that
// one) with its file as a data URI.
func TestFrameSVGLooks(t *testing.T) {
	chrome := findChrome()
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	font, err := os.ReadFile("../../examples/site/fonts/fraunces-600.woff2")
	if err != nil {
		t.Fatal(err)
	}
	uri := "data:font/woff2;base64," + base64.StdEncoding.EncodeToString(font)
	page := filepath.Join(t.TempDir(), "p.html")
	os.WriteFile(page, []byte(`<!doctype html><style>
@font-face{font-family:"Fraunces";src:url("`+uri+`") format("woff2");font-weight:600}
@font-face{font-family:"Unused";src:url("data:font/woff2;base64,AAAA") format("woff2")}
body{margin:0}
.f{width:600px;height:400px;padding:20px;box-sizing:border-box;background:linear-gradient(90deg,#ff0000,#0000ff 80%)}
.c{width:200px;height:100px;border-radius:12px;background:radial-gradient(circle at 25% 50%,rgba(255,255,0,.5),#008000);box-shadow:0 8px 24px rgba(0,0,0,.25);font:600 30px Fraunces,serif}
</style><div class="f"><div class="c">Hello</div></div>`), 0o644)
	svg, err := frameSVG(chrome, "file://"+filepath.ToSlash(page), ".f", 800, 600, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<linearGradient id="si-g`, `stop-color="rgb(255,0,0)"`, `<stop offset="0.8" stop-color="rgb(0,0,255)"`,
		`<radialGradient id="si-g`, `stop-color="rgb(255,255,0)" stop-opacity="0.5"`,
		`<feGaussianBlur stdDeviation="12"/>`, `fill-opacity="0.25" filter="url(#si-s`,
		`font-family:`, uri[:80], `>Hello</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg lacks %q", want)
		}
	}
	if strings.Contains(svg, "Unused") {
		t.Error("a font the text does not use went in")
	}
	if strings.Index(svg, `filter="url(#si-s`) > strings.LastIndex(svg, `fill="url(#si-g`) {
		t.Error("the shadow is drawn over its box")
	}
	if err := xml.NewDecoder(strings.NewReader(svg)).Decode(new(struct{})); err != nil {
		t.Errorf("not well-formed: %v", err)
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
