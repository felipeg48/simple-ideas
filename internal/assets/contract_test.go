package assets

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestContractCoversBuiltinCSS fails when the built-in CSS (or the presenter
// window) uses an --si-* property or a .si-* class that the exported contract
// (Tokens, Hooks) does not list, so `si --export css-rules` cannot drift.
func TestContractCoversBuiltinCSS(t *testing.T) {
	css := LayoutCSS + ThemeLooksCSS + DocCSS + MindmapCSS + CodewinCSS + PresenterHTML
	for name := range Themes {
		css += ThemeCSS(name)
	}
	tokens := map[string]bool{}
	for _, tk := range Tokens {
		tokens[tk.Name] = true
	}
	for _, m := range unique(regexp.MustCompile(`--si-[a-z0-9-]+`).FindAllString(css, -1)) {
		if !tokens[m] {
			t.Errorf("built-in CSS uses %s, which is not in assets.Tokens", m)
		}
	}
	hooks := ""
	for _, h := range Hooks {
		hooks += h.Selector + "\n"
	}
	classRE := regexp.MustCompile(`\.si-[a-z0-9-]+`)
	for _, m := range unique(classRE.FindAllString(css, -1)) {
		if !regexp.MustCompile(regexp.QuoteMeta(m) + `\b`).MatchString(hooks) {
			t.Errorf("built-in CSS uses %s, which is not in assets.Hooks", m)
		}
	}
}

func TestStarterCSSListsEverything(t *testing.T) {
	s := StarterCSS()
	for _, tk := range Tokens {
		if !strings.Contains(s, tk.Name+":") {
			t.Errorf("starter CSS lacks %s", tk.Name)
		}
	}
	for _, h := range Hooks {
		if !strings.Contains(s, h.Selector) {
			t.Errorf("starter CSS lacks hook %s", h.Selector)
		}
	}
	if strings.Count(s, "/*") != strings.Count(s, "*/") {
		t.Error("unbalanced comments in the starter CSS")
	}
}

func TestThemesDefineEveryColour(t *testing.T) {
	for name, th := range Themes {
		for _, mode := range []map[string]string{th.Light, th.Dark} {
			for _, k := range []string{"--si-bg", "--si-fg", "--si-muted", "--si-surface", "--si-surface-2", "--si-line", "--si-accent", "--si-accent-2", "--si-on-accent", "--si-ok", "--si-warn", "--si-crit"} {
				if mode[k] == "" {
					t.Errorf("theme %s lacks %s in a mode", name, k)
				}
			}
		}
	}
}

func TestLayoutHasNoFixedCanvas(t *testing.T) {
	// the fluid rule: no fixed slide canvas scaled by a transform
	for _, bad := range []string{"1280px", "1920px", "720px", "1080px", "scale(var("} {
		if strings.Contains(LayoutCSS+DocCSS, bad) {
			t.Errorf("layout CSS contains %q: slides must reflow, never scale a fixed canvas", bad)
		}
	}
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func TestThemeNames(t *testing.T) {
	names := ThemeNames()
	if len(names) != len(Themes) {
		t.Fatalf("ThemeNames=%d, Themes=%d", len(names), len(Themes))
	}
	if names[0] != "studio" {
		t.Errorf("studio should be first, got %q", names[0])
	}
	for _, n := range names {
		if _, ok := Themes[n]; !ok {
			t.Errorf("%q not a built-in theme", n)
		}
	}
}

func TestAcemqThemeEmbedsAssets(t *testing.T) {
	css := ThemeCSS("acemq")
	if strings.Count(css, `url("data:font/woff2;base64,`) != 2 || !strings.Contains(css, `url("data:image/png;base64,`) {
		t.Error("acemq embeds both fonts and the logo as data URIs")
	}
	if regexp.MustCompile(`https?://`).MatchString(css) || strings.Contains(css, `url("acemq/`) {
		t.Error("acemq theme CSS must not reference anything outside the deck")
	}
	if strings.Contains(ThemeCSS("studio"), "@font-face") {
		t.Error("only acemq decks carry the acemq extras")
	}
}

func TestCanonicalTheme(t *testing.T) {
	for in, want := range map[string]string{
		"AceMQ Pro": "acemq-pro", "acemq_pro": "acemq-pro", "acemqpro": "acemq-pro", " ACEMQ ": "acemq",
		"CockroachLabs": "crdb", "studio": "studio", "neon": "neon",
	} {
		if got := CanonicalTheme(in); got != want {
			t.Errorf("CanonicalTheme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAcemqProCarriesBothExtras(t *testing.T) {
	css := ThemeCSS("AceMQ Pro")
	for _, want := range []string{".lockup", ".opts", ".bench", ".verdict", `url("data:font/woff2;base64,`} {
		if !strings.Contains(css, want) {
			t.Errorf("acemq-pro CSS lacks %q", want)
		}
	}
	if strings.Contains(ThemeCSS("acemq"), ".opts") {
		t.Error("the pro catalogue must not leak into acemq")
	}
	if ThemeSample("acemq-pro") == "" || ThemeSample("studio") != "" {
		t.Error("only acemq-pro ships its own gallery sample")
	}
}

// The v2.15.0 themes pass WCAG AA in both modes: body, muted, accent and
// accent-2 text 4.5:1 on the background and on surfaces, and text on an
// accent fill.
func TestNewThemesContrast(t *testing.T) {
	lum := func(hex string) float64 {
		var r, g, b int
		fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
		ch := func(v int) float64 {
			x := float64(v) / 255
			if x <= 0.03928 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return 0.2126*ch(r) + 0.7152*ch(g) + 0.0722*ch(b)
	}
	ratio := func(a, b string) float64 {
		la, lb := lum(a), lum(b)
		return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
	}
	for _, name := range []string{"twilight", "forest", "sunrise", "slate", "neon", "newsprint"} {
		th, ok := Themes[name]
		if !ok {
			t.Fatalf("theme %s missing", name)
		}
		for mode, v := range map[string]map[string]string{"light": th.Light, "dark": th.Dark} {
			for _, ground := range []string{"--si-bg", "--si-surface"} {
				for _, text := range []string{"--si-fg", "--si-muted", "--si-accent", "--si-accent-2"} {
					if r := ratio(v[text], v[ground]); r < 4.5 {
						t.Errorf("%s %s: %s on %s is %.2f:1 (want 4.5)", name, mode, text, ground, r)
					}
				}
			}
			if r := ratio(v["--si-on-accent"], v["--si-accent"]); r < 4.5 {
				t.Errorf("%s %s: on-accent on accent is %.2f:1 (want 4.5)", name, mode, r)
			}
		}
	}
}
