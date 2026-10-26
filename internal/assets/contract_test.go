package assets

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestContractCoversBuiltinCSS fails when the built-in CSS (or the presenter
// window) uses an --si-* property or a .si-* class that the exported contract
// (Tokens, Hooks) does not list, so `si --export css-rules` cannot drift.
func TestContractCoversBuiltinCSS(t *testing.T) {
	css := LayoutCSS + ThemeLooksCSS + PresenterHTML
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
	for _, m := range unique(classRE.FindAllString(LayoutCSS+ThemeLooksCSS, -1)) {
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
		if strings.Contains(LayoutCSS, bad) {
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
