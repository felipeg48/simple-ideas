package themefile

import (
	"os"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

const acme = `/* si theme: Acme · extends studio
   made in si Studio */
:root {
  --si-accent: #e4572e;
  --si-font-heading: "Fraunces", Georgia, serif;
}
:root[data-si-mode="dark"] {
  --si-accent: #ff7a54;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-si-mode]) {
    --si-accent: #ff7a54;
  }
}

/* a heavier rule */
.si-content h2.si-heading::after { height: .16em; content: "a;b{"; }
`

func TestParseAndWrite(t *testing.T) {
	f := Parse(acme)
	if f.Name != "Acme" || f.Extends != "studio" {
		t.Fatalf("header: %q %q", f.Name, f.Extends)
	}
	if f.Light["--si-accent"] != "#e4572e" || f.Dark["--si-accent"] != "#ff7a54" || f.Light["--si-font-heading"] != `"Fraunces", Georgia, serif` {
		t.Fatalf("tokens: %v %v", f.Light, f.Dark)
	}
	if !strings.Contains(f.Extra, `content: "a;b{"`) || !strings.HasPrefix(f.Extra, "/* a heavier rule */") {
		t.Fatalf("extra: %q", f.Extra)
	}
	if got := f.String(); got != acme {
		t.Errorf("round trip changed the file:\n%s", got)
	}
	// an edit changes only its line
	if err := f.Set("--si-accent", "light", "#0b3d91"); err != nil {
		t.Fatal(err)
	}
	if got, want := f.String(), strings.Replace(acme, "#e4572e", "#0b3d91", 1); got != want {
		t.Errorf("edit:\n%s", got)
	}
	if err := f.Set("--si-accent", "dark", ""); err != nil || len(f.Dark) != 0 {
		t.Fatalf("reset dark: %v %v", err, f.Dark)
	}
	if strings.Contains(f.String(), "prefers-color-scheme") {
		t.Error("an empty dark block is still written")
	}
}

// harbor.css (hand-written, its own comment, extra rules, a layer-less file)
// keeps its tokens and rules.
func TestParseHarbor(t *testing.T) {
	src, err := os.ReadFile("../../examples/css/harbor.css")
	if err != nil {
		t.Fatal(err)
	}
	f := Parse(string(src))
	if f.Name != "Harbor" || f.Extends != "studio" {
		t.Errorf("header: %q %q", f.Name, f.Extends)
	}
	if f.Light["--si-bg"] != "#f3efe6" || f.Dark["--si-bg"] != "#0b1a28" || !strings.Contains(f.Extra, ".si-kicker") {
		t.Errorf("harbor: %v %v %q", f.Light, f.Dark, f.Extra)
	}
	g := Parse(f.String())
	if g.String() != f.String() {
		t.Error("second round trip differs")
	}
	for _, is := range f.Check(string(src)) {
		if is.Mode == "" { // contrast findings are fine here; a broken value is not
			t.Errorf("harbor: %+v", is)
		}
	}
}

func TestLayerWrapperAndNoHeader(t *testing.T) {
	f := Parse("@layer si.theme {\n  :root { --si-radius: .2em; }\n}\n")
	if f.Light["--si-radius"] != ".2em" || f.Name != "" {
		t.Fatalf("%+v", f)
	}
	if strings.HasPrefix(f.String(), "/*") {
		t.Error("a file without a header gained one")
	}
	if f := Parse(".x { color: red }"); f.Extra != ".x { color: red }" || len(f.Light) != 0 {
		t.Errorf("plain css: %+v", f)
	}
}

func TestCheckValue(t *testing.T) {
	for _, c := range []struct {
		name, v string
		ok      bool
	}{
		{"--si-accent", "#e4572e", true},
		{"--si-accent", "oklch(60% .2 30)", true},
		{"--si-accent", "#e457", true},
		{"--si-accent", "#e4572", false},
		{"--si-accent", "red; } body { display:none", false},
		{"--si-accent", "url(x.png)", false},
		{"--si-h1-size", "3.4em", true},
		{"--si-h1-size", "48px", false},
		{"--si-h1-size", "3", false},
		{"--si-pad-x", "max(1rem, 7cqi)", true},
		{"--si-heading-case", "uppercase", true},
		{"--si-heading-case", "shout", false},
		{"--si-step-duration", ".4s", true},
		{"--si-step-duration", "fast", false},
		{"--si-heading-weight", "700", true},
		{"--si-nope", "1", false},
		{"--si-font-body", `"Inter", system-ui, sans-serif`, true},
		{"--si-font-body", `"x" </style><script>`, false},
	} {
		if err := CheckValue(c.name, c.v); (err == nil) != c.ok {
			t.Errorf("%s: %s: %v", c.name, c.v, err)
		}
	}
}

func TestCheckContrastAndFix(t *testing.T) {
	src := "/* si theme: Low · extends studio */\n:root {\n  --si-muted: #bbbbbb;\n}\n"
	f := Parse(src)
	iss := f.Check(src)
	var hit *Issue
	for i := range iss {
		if iss[i].Token == "--si-muted" && iss[i].Mode == "light" && iss[i].On == "--si-bg" {
			hit = &iss[i]
		}
	}
	if hit == nil || hit.Line != 3 || hit.Fix == "" {
		t.Fatalf("issues: %+v", iss)
	}
	light, _ := f.Effective()
	if r, _ := Contrast(hit.Fix, light["--si-bg"]); r < 4.5 {
		t.Errorf("fix %s is %.2f:1", hit.Fix, r)
	}
	if r, ok := Contrast("#000", "#fff"); !ok || r < 20.9 {
		t.Errorf("black on white %.2f", r)
	}
}

// The public built-ins pass the checks in both modes.
func TestBuiltinsPass(t *testing.T) {
	for _, n := range assets.ThemeNames() {
		if assets.UnlistedThemes[n] {
			continue
		}
		f, _ := Builtin(n)
		for _, is := range f.Check("") {
			t.Errorf("%s: %s", n, is.Msg)
		}
		if g := Parse(f.String()); len(g.Light) != len(f.Light) || len(g.Dark) != len(f.Dark) {
			t.Errorf("%s: export does not read back", n)
		}
	}
}

func TestDTCG(t *testing.T) {
	f := Parse(acme)
	f.Set("--si-h1-size", "", "3.4em")
	f.Set("--si-pad-x", "", "7cqi")
	f.Set("--si-step-duration", "", ".4s")
	js, unmapped := f.DTCG()
	if len(unmapped) != 1 || unmapped[0] != "--si-pad-x" {
		t.Errorf("unmapped: %v", unmapped)
	}
	g, un, err := FromDTCG(js)
	if err != nil || len(un) != 0 {
		t.Fatalf("%v %v", err, un)
	}
	if g.Name != "Acme" || g.Extends != "studio" || g.Light["--si-accent"] != "#e4572e" || g.Dark["--si-accent"] != "#ff7a54" ||
		g.Light["--si-h1-size"] != "3.4em" || g.Light["--si-step-duration"] != ".4s" && g.Light["--si-step-duration"] != "0.4s" ||
		g.Light["--si-font-heading"] != `Fraunces, Georgia, serif` {
		t.Errorf("back: %v %v", g.Light, g.Dark)
	}
	// a plain tokens file: light/dark sets, a colour object, an unknown token
	h, un, _ := FromDTCG([]byte(`{"light":{"accent":{"$type":"color","$value":{"colorSpace":"srgb","hex":"#0b3d91"}},"brand":{"$value":"#fff"}},
		"dark":{"accent":{"$type":"color","$value":"#6f9cff"}}}`))
	if h.Light["--si-accent"] != "#0b3d91" || h.Dark["--si-accent"] != "#6f9cff" || len(un) != 1 || un[0] != "light.brand" {
		t.Errorf("plain: %v %v %v", h.Light, h.Dark, un)
	}
}
