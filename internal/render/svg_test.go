package render

import "testing"

func TestLintSVG(t *testing.T) {
	cases := []struct {
		name string
		svg  string
		want int
	}{
		{"clean", `<svg viewBox="0 0 10 10"><style>.a{fill:red}</style><rect/></svg>`, 0},
		{"lt-in-style", `<svg viewBox="0 0 10 10"><style>.a::before{content:"<"}</style></svg>`, 1},
		{"no-viewbox-no-size", `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`, 1},
		{"no-viewbox-but-sized", `<svg width="10" height="10"><rect/></svg>`, 0},
		{"both", `<svg xmlns="x"><style>a<b{}</style></svg>`, 2},
		{"style-with-attr-ok", `<svg viewBox="0 0 10 10"><style type="text/css">.a{fill:red}</style></svg>`, 0},
	}
	for _, c := range cases {
		if got := len(lintSVG(c.svg)); got != c.want {
			t.Errorf("%s: got %d problems, want %d: %v", c.name, got, c.want, lintSVG(c.svg))
		}
	}
}
