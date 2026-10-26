// Package assets holds the embedded web files (layout CSS, theme CSS, deck
// runtime, presenter window) and the styling contract: the --si-* tokens,
// the .si-* hooks and the built-in theme values.
package assets

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// LayoutCSS is the unlayered layout mechanics (fluid sizing, slides,
// transitions, steps, overview, print, reduced motion).
//
//go:embed web/layout.css
var LayoutCSS string

// ThemeLooksCSS is the built-in look in the si.theme layer.
//
//go:embed web/theme.css
var ThemeLooksCSS string

// DeckJS is the deck runtime.
//
//go:embed web/deck.js
var DeckJS string

// PresenterHTML is the presenter window document; __CSS__, __TITLE__ and
// __DATA__ are filled in by DeckJS when the window opens.
//
//go:embed web/presenter.html
var PresenterHTML string

// LayerOrder declares the cascade layers; user CSS goes in si.user.
const LayerOrder = "@layer si.theme, si.user;\n"

// animFS holds the built-in timer animations (web/anim/<name>.js|.svg).
//
//go:embed web/anim
var animFS embed.FS

// Animation returns the code and kind ("js" or "svg") of a built-in timer
// animation by name (file stem), or ok=false if there is no such built-in.
func Animation(name string) (code, kind string, ok bool) {
	for _, k := range []string{"js", "svg"} {
		if b, err := animFS.ReadFile("web/anim/" + name + "." + k); err == nil {
			return string(b), k, true
		}
	}
	return "", "", false
}

// AnimationNames lists the built-in timer animation names, sorted.
func AnimationNames() []string {
	var names []string
	entries, _ := fs.ReadDir(animFS, "web/anim")
	for _, e := range entries {
		n := e.Name()
		if i := strings.LastIndexByte(n, '.'); i > 0 {
			names = append(names, n[:i])
		}
	}
	sort.Strings(names)
	return names
}
