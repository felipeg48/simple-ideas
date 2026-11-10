package assets

import "strings"

// Token kinds: what a value of a token is, for the Theme editor's controls,
// the theme checker and the Design Tokens mapping.
const (
	KindColor    = "color"    // a colour (or a gradient for --si-slide-bg)
	KindLength   = "length"   // em, rem, cqi, cqb, %, ch (never px)
	KindFont     = "font"     // a font stack
	KindNumber   = "number"   // a plain number (weight, scale, line height)
	KindEnum     = "enum"     // one of Enum
	KindDuration = "duration" // s or ms
	KindShadow   = "shadow"   // a box-shadow (or none)
	KindOther    = "other"    // set per element by a directive or the runtime
)

// TokenGroups are the Theme editor's groups, in order. "Element" holds the
// tokens directives and the runtime set on one slide or image: not a theme's.
var TokenGroups = []string{"Colours", "Type", "Headings", "Space", "Shape", "Motion", "Documents", "Code", "Mind maps", "Code windows", "Audience", "QR", "Element"}

// TokenMeta describes one token for editors and checkers.
type TokenMeta struct {
	Name  string   `json:"name"`
	Desc  string   `json:"desc"`
	Group string   `json:"group"`
	Kind  string   `json:"kind"`
	Units []string `json:"units,omitempty"` // length: the units a value may use
	Enum  []string `json:"enum,omitempty"`
	// Pair: the token takes a light and a dark value (colours and shadows).
	Pair bool `json:"pair"`
}

// LengthUnits are the units a length token may use: all relative, so the
// layout stays fluid.
var LengthUnits = []string{"em", "rem", "cqi", "cqb", "%", "ch"}

var tokenEnums = map[string][]string{
	"--si-heading-case": {"none", "uppercase", "lowercase", "capitalize"},
	"--si-bg-fit":       {"cover", "contain"},
	"--si-img-fit":      {"cover", "contain", "fill", "none", "scale-down"},
}

var tokenNumbers = map[string]bool{"--si-scale": true, "--si-heading-weight": true, "--si-prose-leading": true, "--si-wm-opacity": true}

var tokenLengths = map[string]bool{"--si-radius": true, "--si-h1-size": true, "--si-h2-size": true, "--si-gap": true, "--si-pad-x": true,
	"--si-pad-y": true, "--si-measure": true, "--si-figure-max": true, "--si-heading-tracking": true, "--si-prose-size": true,
	"--si-doc-measure": true, "--si-lede-size": true, "--si-mm-height": true, "--si-qr-size": true, "--si-cw-pad": true,
	"--si-cw-radius": true, "--si-cw-size": true}

// elementTokens are set per slide, image or topic, never by a theme.
var elementTokens = map[string]bool{"--si-slide-bg": true, "--si-bg-fit": true, "--si-bg-position": true, "--si-bg-opacity": true,
	"--si-bg-blur": true, "--si-img-width": true, "--si-img-ratio": true, "--si-img-fit": true, "--si-img-position": true,
	"--si-ov-cols": true, "--si-wm-img": true, "--si-wm-opacity": true, "--si-mm-c": true, "--si-brand-logo": true}

// UnlistedThemes are built-ins kept out of the docs and the Studio's start
// list (client themes); they still work by name.
var UnlistedThemes = map[string]bool{"crdb": true, "acemq": true, "acemq-pro": true}

// ThemeTokens returns a built-in theme's token values: light holds the
// mode-independent defaults, the theme's base values and its light colours;
// dark holds what dark mode changes. ok is false for an unknown name.
func ThemeTokens(name string) (light, dark map[string]string, ok bool) {
	th, ok := Themes[CanonicalTheme(name)]
	if !ok {
		return nil, nil, false
	}
	light = map[string]string{}
	for _, m := range []map[string]string{layoutDefaults, th.Base, th.Light} {
		for k, v := range m {
			light[k] = v
		}
	}
	dark = map[string]string{}
	for k, v := range th.Dark {
		dark[k] = v
	}
	return light, dark, true
}

// Specimen is the deck the Theme editor previews a theme with: every part a
// theme touches (headings, lists, columns, a table, a callout, a code
// window, a big number, a quote, a mind map, a QR code).
func Specimen() string {
	md, _ := themesFS.ReadFile("web/themes/specimen.md")
	return string(md)
}

// Meta returns the metadata of a token of the contract; ok is false for a
// name that is not in Tokens.
func Meta(name string) (TokenMeta, bool) {
	for _, t := range Tokens {
		if t.Name == name {
			return tokenMeta(t), true
		}
	}
	return TokenMeta{}, false
}

// AllMeta returns the metadata of every token, in contract order.
func AllMeta() []TokenMeta {
	out := make([]TokenMeta, len(Tokens))
	for i, t := range Tokens {
		out[i] = tokenMeta(t)
	}
	return out
}

func tokenMeta(t Token) TokenMeta {
	n := t.Name
	m := TokenMeta{Name: n, Desc: t.Desc}
	switch {
	case elementTokens[n]:
		m.Kind = KindOther
	case tokenEnums[n] != nil:
		m.Kind, m.Enum = KindEnum, tokenEnums[n]
	case strings.Contains(n, "font") || n == "--si-mm-mono" || n == "--si-mm-hand":
		m.Kind = KindFont
	case strings.HasSuffix(n, "-duration"):
		m.Kind = KindDuration
	case strings.HasSuffix(n, "shadow"):
		m.Kind, m.Pair = KindShadow, true
	case tokenNumbers[n]:
		m.Kind = KindNumber
	case tokenLengths[n]:
		m.Kind, m.Units = KindLength, LengthUnits
	default:
		m.Kind, m.Pair = KindColor, true
	}
	switch {
	case elementTokens[n]:
		m.Group = "Element"
	case strings.HasPrefix(n, "--si-mm-"):
		m.Group = "Mind maps"
	case strings.HasPrefix(n, "--si-cw-"):
		m.Group = "Code windows"
	case strings.HasPrefix(n, "--si-qr-"):
		m.Group = "QR"
	case strings.HasPrefix(n, "--si-poll-") || strings.HasPrefix(n, "--si-qa-"):
		m.Group = "Audience"
	case strings.HasPrefix(n, "--si-code-") || strings.HasPrefix(n, "--si-diff-") || n == "--si-tab-active":
		m.Group = "Code"
	case strings.HasPrefix(n, "--si-heading-") || n == "--si-h1-size" || n == "--si-h2-size" || n == "--si-font-heading":
		m.Group = "Headings"
	case strings.Contains(t.Desc, "documents:") || strings.HasPrefix(t.Desc, "guides:") || strings.HasPrefix(t.Desc, "labs and guides:"):
		m.Group = "Documents"
	case m.Kind == KindFont || n == "--si-scale" || n == "--si-measure":
		m.Group = "Type"
	case m.Kind == KindDuration:
		m.Group = "Motion"
	case n == "--si-radius" || n == "--si-shadow":
		m.Group = "Shape"
	case m.Kind == KindLength:
		m.Group = "Space"
	default:
		m.Group = "Colours"
	}
	return m
}
