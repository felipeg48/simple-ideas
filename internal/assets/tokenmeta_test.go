package assets

import "testing"

// Every token has a group of TokenGroups and a kind; the tokens a built-in
// theme sets per mode are light/dark pairs, the mode-independent ones are not.
func TestTokenMeta(t *testing.T) {
	groups := map[string]int{}
	for _, m := range AllMeta() {
		groups[m.Group]++
		if m.Kind == "" {
			t.Errorf("%s has no kind", m.Name)
		}
	}
	for _, g := range TokenGroups {
		if groups[g] == 0 {
			t.Errorf("group %s has no token", g)
		}
		delete(groups, g)
	}
	for g := range groups {
		t.Errorf("group %s is not in TokenGroups", g)
	}
	for name, th := range Themes {
		for _, mode := range []map[string]string{th.Light, th.Dark} {
			for k := range mode {
				if m, _ := Meta(k); !m.Pair {
					t.Errorf("%s sets %s per mode, but it is a %s", name, k, m.Kind)
				}
			}
		}
		for k := range th.Base {
			if m, ok := Meta(k); !ok || m.Pair || m.Group == "Element" {
				t.Errorf("%s base sets %s (%+v)", name, k, m)
			}
		}
	}
	want := map[string]string{"--si-accent": KindColor, "--si-h1-size": KindLength, "--si-font-body": KindFont,
		"--si-heading-case": KindEnum, "--si-step-duration": KindDuration, "--si-heading-weight": KindNumber, "--si-shadow": KindShadow}
	for n, k := range want {
		if m, _ := Meta(n); m.Kind != k {
			t.Errorf("%s: kind %s, want %s", n, m.Kind, k)
		}
	}
}
