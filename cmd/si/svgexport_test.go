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

// TestFrameSVGEffects: a conic gradient becomes wedges clipped to its box, an
// inset shadow a blurred frame cut out by the box (inside it), a text shadow
// a blurred copy of the text under it.
func TestFrameSVGEffects(t *testing.T) {
	chrome := findChrome()
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	page := filepath.Join(t.TempDir(), "p.html")
	os.WriteFile(page, []byte(`<!doctype html><style>body{margin:0}
.f{width:600px;height:400px;background:#fff}
.d{width:200px;height:200px;border-radius:50%;background:conic-gradient(from 90deg,#ff0000 0 25%,#0000ff 25% 75%,#00ff00 75%)}
.i{width:200px;height:80px;box-shadow:inset 0 4px 10px rgba(0,0,0,.5)}
.t{font:700 40px sans-serif;text-shadow:2px 3px 6px rgba(255,0,0,.8)}
</style><div class="f"><div class="d"></div><div class="i"></div><div class="t">Glow</div></div>`), 0o644)
	svg, err := frameSVG(chrome, "file://"+filepath.ToSlash(page), ".f", 800, 600, false)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(svg, `<path d="M100 100L`); n < 170 {
		t.Errorf("conic: %d wedges from the centre", n)
	}
	for _, want := range []string{`fill="rgb(255,0,0)"`, `fill="rgb(0,0,255)"`, `fill="rgb(0,255,0)"`, `fill-rule="evenodd"`, `fill-opacity="0.5" filter="url(#si-s`, `fill-opacity="0.8" filter="url(#si-t`, `<feGaussianBlur stdDeviation="3"/>`} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg lacks %q", want)
		}
	}
	if strings.Count(svg, `>Glow</text>`) != 2 || strings.Index(svg, `filter="url(#si-t`) > strings.LastIndex(svg, `>Glow</text>`) {
		t.Error("text shadow: a copy under the text")
	}
	if err := xml.NewDecoder(strings.NewReader(svg)).Decode(new(struct{})); err != nil {
		t.Errorf("not well-formed: %v", err)
	}
}

// TestSVGFonts: an embedded TrueType font is cut to the SVG's text; a WOFF2
// font si may not fetch again stays whole, with a note.
func TestSVGFonts(t *testing.T) {
	ttf, err := os.ReadFile("/System/Library/Fonts/Supplemental/Arial.ttf")
	if err != nil {
		t.Skip("no TrueType font here")
	}
	woff2 := "d09GMgABAAAAAAAA"
	svg := `<svg xmlns="http://www.w3.org/2000/svg"><defs><style>@font-face{font-family:"Body";src:url("data:font/ttf;base64,` + base64.StdEncoding.EncodeToString(ttf) + `") format("truetype");font-weight:400}
@font-face{font-family:"Brand";src:url("data:font/woff2;base64,` + woff2 + `") format("woff2");font-weight:700}</style></defs><text>Hitos 1936</text></svg>`
	out, notes := svgFonts(svg, map[string]bool{})
	if len(out) > len(svg)/3 || !strings.Contains(out, `format("truetype");font-weight:400`) || !strings.Contains(out, woff2) {
		t.Errorf("%d of %d bytes", len(out), len(svg))
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "Brand 700: WOFF2 only") {
		t.Errorf("notes: %q", notes)
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
