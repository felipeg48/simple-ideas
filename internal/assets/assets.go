// Package assets holds the embedded web files (layout CSS, theme CSS, deck
// runtime, presenter window) and the styling contract: the --si-* tokens,
// the .si-* hooks and the built-in theme values.
package assets

import _ "embed"

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
