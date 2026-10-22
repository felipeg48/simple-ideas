package render

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.html")

var mainRE = regexp.MustCompile(`(?s)<main .*</main>`)

func renderGolden(t *testing.T, opt Options) (Result, string) {
	t.Helper()
	d, err := deck.Parse(filepath.Join("testdata", "deck.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, opt)
	return res, string(res.HTML)
}

// TestGolden compares the slide markup of testdata/deck.md (every rule) with
// testdata/golden.html. Run `go test ./internal/render -update` after an
// intended change and review the diff.
func TestGolden(t *testing.T) {
	res, page := renderGolden(t, Options{})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	got := mainRE.FindString(page) + "\n"
	golden := filepath.Join("testdata", "golden.html")
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
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("golden mismatch at line %d:\n got: %.300s\nwant: %.300s\n(run go test ./internal/render -update if intended)", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("golden mismatch: %d lines, want %d", len(gl), len(wl))
	}
}

func TestPage(t *testing.T) {
	res, page := renderGolden(t, Options{UserCSS: []Stylesheet{{Path: "testdata/my.css", Text: ".si-slide{color:red} a{background:url(assets/pic.svg)}"}}})
	if res.Steps != 5 || res.Canvases != 2 {
		t.Errorf("steps = %d, canvases = %d; want 5 and 2", res.Steps, res.Canvases)
	}
	for _, want := range []string{
		`<html lang="en" data-si-theme="paper">`,
		"@layer si.theme, si.user;",
		`--si-accent:#e4572e`,
		`<style id="si-user">`, "@layer si.user{", `url("data:image/svg+xml;base64,`,
		`"transition":"slide"`, `"plan":[0.5,0,1,0,0,0,0,0,0,0]`,
		`.si-canvas[data-state~="on"] [data-show~="on"]{opacity:1}`,
		`.si-canvas[data-state~="busy"] [data-flow~="busy"]`,
		`.si-canvas[data-state~="on"] .si-packets[data-for~="on"]`,
		"@page{size:13.333in 7.5in;margin:0}",
		"window.SI_PRESENTER",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	// hide rules come after show rules so data-hide wins
	if strings.Index(page, `[data-hide~="on"]`) < strings.Index(page, `[data-show~="on"]`) {
		t.Error("hide rules must follow show rules")
	}
	if strings.Contains(page, "mermaid.min.js") {
		t.Error("no mermaid fence, no mermaid script")
	}
	if strings.Contains(page, "EventSource") {
		t.Error("static output must not include the live reload client")
	}
}

func TestMediaModes(t *testing.T) {
	res, page := renderGolden(t, Options{Media: MediaCopy})
	if !strings.Contains(page, `<img src="assets/pic.svg"`) || len(res.Media) != 1 || res.Media[0].Rel != "assets/pic.svg" {
		t.Errorf("copy mode: media = %+v", res.Media)
	}
	_, page = renderGolden(t, Options{Media: MediaLink, LiveReload: true})
	if !strings.Contains(page, `<img src="assets/pic.svg"`) || !strings.Contains(page, "/_si/events") {
		t.Error("link mode links files and play adds live reload")
	}
}

func TestStateOps(t *testing.T) {
	tests := []struct {
		prev  []string
		words string
		want  string
	}{
		{nil, "a b", "a b"},
		{[]string{"a", "b"}, "+c", "a b c"},
		{[]string{"a", "b"}, "-a +c", "b c"},
		{[]string{"a", "b"}, "c", "c"}, // absolute words replace the state
		{[]string{"a"}, "", ""},
	}
	for _, tc := range tests {
		if got := strings.Join(resolveState(tc.prev, strings.Fields(tc.words)), " "); got != tc.want {
			t.Errorf("resolveState(%v, %q) = %q, want %q", tc.prev, tc.words, got, tc.want)
		}
	}
	if got := strings.Join(applyOps([]string{"a", "b"}, "-a,c . +d"), " "); got != "b c d" {
		t.Errorf("applyOps = %q", got)
	}
}

func TestRelativeCanvasState(t *testing.T) {
	_, page := renderGolden(t, Options{})
	// slide 3 ends at on+busy-on+bad = busy bad; slide 4 adds +busy
	if !strings.Contains(page, `data-state="on" data-base="on"`) || !strings.Contains(page, `data-state="busy bad" data-base="busy bad"`) {
		t.Error("relative canvas state did not carry across slides")
	}
	if strings.Count(page, `id="c1-box"`) != 1 || strings.Count(page, `id="c2-box"`) != 1 || !strings.Contains(page, `fill="url(#c2-g)"`) {
		t.Error("svg ids must be prefixed per instance, references too")
	}
	if !strings.Contains(page, `viewBox="0 0 200 100"`) {
		t.Error("a viewBox is derived from width and height")
	}
}

func TestHighlight(t *testing.T) {
	tests := []struct{ code, lang, want string }{
		{`if x := "s"; x { return 1 } // c`, "go", `<span class="si-tk-k">if</span> x := <span class="si-tk-s">&#34;s&#34;</span>; x { <span class="si-tk-k">return</span> <span class="si-tk-n">1</span> } <span class="si-tk-c">// c</span>`},
		{"si play --open $DIR # go", "bash", `<span class="si-tk-m">si</span> play<span class="si-tk-f"> --open</span> <span class="si-tk-v">$DIR</span> <span class="si-tk-c"># go</span>`},
		{"a < b", "text", "a &lt; b"},
	}
	for _, tc := range tests {
		if got := highlight(tc.code, tc.lang); got != tc.want {
			t.Errorf("highlight(%q, %s) =\n %s\nwant\n %s", tc.code, tc.lang, got, tc.want)
		}
	}
}

func TestToneOf(t *testing.T) {
	for in, want := range map[string]string{"#101820": "dark", "#fff": "light", "linear-gradient(#f6f5f1, #000)": "light", "red": ""} {
		if got := toneOf(in); got != want {
			t.Errorf("toneOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMissingFilesAreProblems(t *testing.T) {
	dir := t.TempDir()
	md := "# D\n\n---\n\n## A\n\n![x](nope.png)\n\n---\n\n## B\n\n<!-- background: gone.jpg -->\n"
	if err := os.WriteFile(filepath.Join(dir, "d.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	d, _ := deck.Parse(filepath.Join(dir, "d.md"))
	res := Deck(d, Options{})
	all := ""
	for _, p := range res.Problems {
		all += p.Msg + "\n"
	}
	if !strings.Contains(all, "missing file: nope.png") || !strings.Contains(all, "missing file: gone.jpg") {
		t.Errorf("problems = %q", all)
	}
}
