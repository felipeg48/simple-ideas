package themefile

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// Issue is one finding of Check.
type Issue struct {
	Line  int     `json:"line,omitempty"` // 1-based line in the file, 0 if unknown
	Token string  `json:"token,omitempty"`
	Mode  string  `json:"mode,omitempty"` // light or dark, for contrast
	Msg   string  `json:"msg"`
	Ratio float64 `json:"ratio,omitempty"` // contrast: the pair's ratio
	Want  float64 `json:"want,omitempty"`  // contrast: the minimum
	On    string  `json:"on,omitempty"`    // contrast: the background token
	Fix   string  `json:"fix,omitempty"`   // contrast: the nearest passing text colour
}

// Pair is a text/background pair Check measures, with its WCAG minimum
// (4.5:1 for body text, 3:1 for large text and UI marks).
type Pair struct {
	Text, Ground string
	Min          float64
}

// Pairs are the pairs Check measures, in light and in dark.
var Pairs = []Pair{
	{"--si-fg", "--si-bg", 4.5}, {"--si-fg", "--si-surface", 4.5},
	{"--si-muted", "--si-bg", 4.5}, {"--si-muted", "--si-surface", 4.5},
	{"--si-accent", "--si-bg", 3}, {"--si-accent-2", "--si-bg", 3},
	{"--si-on-accent", "--si-accent", 4.5},
	{"--si-ok", "--si-bg", 3}, {"--si-warn", "--si-bg", 3}, {"--si-crit", "--si-bg", 3},
}

var (
	pxRE      = regexp.MustCompile(`\d(\.\d+)?px\b`)
	bareNumRE = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)$`)
	hexRE     = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	durRE     = regexp.MustCompile(`^(\d+\.?\d*|\.\d+)m?s$`)
	colorRE   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*(\(.*\))?$`)
)

// CheckValue says why value is not a valid value for the token name, or nil.
// It refuses unknown tokens, anything that could leave the declaration
// (braces, ";", "<", @import, expression(), url()) and values of the wrong
// kind; a length in px breaks the fluid layout, so it is refused too.
func CheckValue(name, value string) error {
	m, ok := assets.Meta(name)
	if !ok {
		return fmt.Errorf("%s is not an si token (see si --export css-rules)", name)
	}
	v := strings.TrimSpace(value)
	low := strings.ToLower(v)
	if v == "" {
		return fmt.Errorf("%s: empty value", name)
	}
	for _, bad := range []string{"{", "}", ";", "<", "@import", "expression(", "url(", "javascript:"} {
		if strings.Contains(low, bad) {
			return fmt.Errorf("%s: %q is not allowed in a token value", name, bad)
		}
	}
	if strings.HasPrefix(low, "var(") || strings.HasPrefix(low, "calc(") || strings.HasPrefix(low, "color-mix(") {
		return nil
	}
	switch m.Kind {
	case assets.KindLength:
		if pxRE.MatchString(low) {
			return fmt.Errorf("%s: px is a fixed size; use em, rem, cqi, cqb, %% or ch", name)
		}
		if bareNumRE.MatchString(v) && v != "0" {
			return fmt.Errorf("%s: %s needs a unit (em, rem, cqi, cqb, %% or ch)", name, v)
		}
	case assets.KindNumber:
		if _, err := strconv.ParseFloat(v, 64); err != nil && low != "normal" && low != "bold" {
			return fmt.Errorf("%s: %s is not a number", name, v)
		}
	case assets.KindDuration:
		if !durRE.MatchString(low) {
			return fmt.Errorf("%s: %s is not a duration (like .4s or 300ms)", name, v)
		}
	case assets.KindEnum:
		for _, e := range m.Enum {
			if low == e {
				return nil
			}
		}
		return fmt.Errorf("%s: %s is not one of %s", name, v, strings.Join(m.Enum, ", "))
	case assets.KindColor:
		if strings.HasPrefix(v, "#") && !hexRE.MatchString(v) {
			return fmt.Errorf("%s: %s is not a hex colour", name, v)
		}
		if !strings.HasPrefix(v, "#") && !colorRE.MatchString(v) {
			return fmt.Errorf("%s: %s is not a colour", name, v)
		}
	case assets.KindShadow:
		if pxRE.MatchString(low) {
			return fmt.Errorf("%s: px is a fixed size; use em", name)
		}
	}
	return nil
}

// Check reports the problems of a theme file: unknown tokens, values of the
// wrong kind, and text/background pairs under their WCAG minimum in light
// or dark (with the nearest passing text colour). src is the file's text,
// for line numbers; "" leaves them out.
func (f *File) Check(src string) []Issue {
	var out []Issue
	line := func(tok string) int {
		if src == "" {
			return 0
		}
		if i := strings.Index(src, tok+":"); i >= 0 {
			return strings.Count(src[:i], "\n") + 1
		}
		return 0
	}
	if f.Extends != "" {
		if _, _, ok := assets.ThemeTokens(f.Extends); !ok {
			out = append(out, Issue{Line: 1, Msg: fmt.Sprintf("extends %s: no such built-in theme (see si theme list)", f.Extends)})
		}
	}
	for _, m := range []map[string]string{f.Light, f.Dark} {
		for _, k := range Order(m) {
			if err := CheckValue(k, m[k]); err != nil {
				out = append(out, Issue{Line: line(k), Token: k, Msg: err.Error()})
			}
		}
	}
	light, dark := f.Effective()
	for _, mode := range []struct {
		name string
		v    map[string]string
	}{{"light", light}, {"dark", dark}} {
		for _, p := range Pairs {
			r, ok := Contrast(mode.v[p.Text], mode.v[p.Ground])
			if !ok || r >= p.Min {
				continue
			}
			out = append(out, Issue{Line: line(p.Text), Token: p.Text, Mode: mode.name, On: p.Ground, Ratio: math.Round(r*100) / 100, Want: p.Min,
				Msg: fmt.Sprintf("%s on %s is %.2f:1 in %s mode (want %.1f:1)", p.Text, p.Ground, r, mode.name, p.Min),
				Fix: FixContrast(mode.v[p.Text], mode.v[p.Ground], p.Min)})
		}
	}
	return out
}

// rgb parses #rgb / #rrggbb (alpha ignored) into 0..1 channels.
func rgb(hex string) (r, g, b float64, ok bool) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if !strings.HasPrefix(strings.TrimSpace(hex), "#") {
		return 0, 0, 0, false
	}
	switch len(h) {
	case 3, 4:
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	case 6, 8:
		h = h[:6]
	default:
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return float64(n>>16&255) / 255, float64(n>>8&255) / 255, float64(n&255) / 255, true
}

func toLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func toSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func luminance(r, g, b float64) float64 {
	return 0.2126*toLinear(r) + 0.7152*toLinear(g) + 0.0722*toLinear(b)
}

// Contrast is the WCAG contrast ratio of two hex colours; ok is false when
// either is not a hex colour (a var(), a mix: not measured).
func Contrast(a, b string) (float64, bool) {
	ar, ag, ab, ok1 := rgb(a)
	br, bg, bb, ok2 := rgb(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	la, lb := luminance(ar, ag, ab), luminance(br, bg, bb)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05), true
}

// FixContrast moves the text colour fg in OKLab lightness, keeping its hue
// and chroma, until it reaches min against bg; "" when fg or bg is not a
// hex colour or no lightness passes.
func FixContrast(fg, bg string, min float64) string {
	r, g, b, ok := rgb(fg)
	if _, _, _, ok2 := rgb(bg); !ok || !ok2 {
		return ""
	}
	L, A, B := oklab(r, g, b)
	br, bgg, bb, _ := rgb(bg)
	step := 0.005
	if luminance(br, bgg, bb) > 0.18 { // light ground: darken the text
		step = -step
	}
	for l := L; l >= 0 && l <= 1; l += step {
		h := hexOf(fromOklab(l, A, B))
		if c, _ := Contrast(h, bg); c >= min {
			return h
		}
	}
	return ""
}

func oklab(r, g, b float64) (L, A, B float64) {
	r, g, b = toLinear(r), toLinear(g), toLinear(b)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}

func fromOklab(L, A, B float64) (r, g, b float64) {
	l := math.Pow(L+0.3963377774*A+0.2158037573*B, 3)
	m := math.Pow(L-0.1055613458*A-0.0638541728*B, 3)
	s := math.Pow(L-0.0894841775*A-1.2914855480*B, 3)
	return toSRGB(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		toSRGB(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		toSRGB(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s)
}

func hexOf(r, g, b float64) string {
	c := func(v float64) int { return int(math.Round(math.Max(0, math.Min(1, v)) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", c(r), c(g), c(b))
}
