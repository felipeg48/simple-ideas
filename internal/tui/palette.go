// Package tui is si in the terminal: slides drawn with text and colour
// (si play --terminal), the teleprompter in a second terminal (--notes), and
// the Studio as a keyboard app (si --terminal). It reads the same parsed deck
// as the HTML renderer and uses the Charm libraries (Bubble Tea, Lip Gloss) for
// the terminal itself; a built deck never touches this package.
package tui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// Palette is a theme's colours for the terminal.
type Palette struct {
	BG, FG, Muted, Accent, Accent2, OK, Warn, Crit, Surface, Line, HL color.Color
}

// PaletteFor builds the palette of a built-in theme from its tokens (the
// dark values unless mode is "light"); any other theme gets studio's.
func PaletteFor(theme, mode string) Palette {
	th, ok := assets.Themes[assets.CanonicalTheme(theme)]
	if !ok {
		th = assets.Themes["studio"]
	}
	tok := func(name, fallback string) color.Color {
		for _, m := range [][]map[string]string{{th.Dark, th.Base}, {th.Light, th.Base}}[boolIndex(mode == "light")] {
			if c, ok := parseHex(m[name]); ok {
				return c
			}
		}
		c, _ := parseHex(fallback)
		return c
	}
	light := mode == "light"
	def := func(dark, l string) string {
		if light {
			return l
		}
		return dark
	}
	p := Palette{
		BG:      tok("--si-bg", def("#0f1116", "#f6f5f1")),
		FG:      tok("--si-fg", def("#eceef3", "#16171b")),
		Muted:   tok("--si-muted", def("#99a0ad", "#5d626c")),
		Accent:  tok("--si-accent", def("#7d9bff", "#3657d6")),
		Accent2: tok("--si-accent-2", def("#5fd1c4", "#0b8a7a")),
		OK:      tok("--si-ok", def("#4fd18b", "#1f8a4c")),
		Warn:    tok("--si-warn", def("#f2b54a", "#b7791f")),
		Crit:    tok("--si-crit", def("#ff6b6b", "#c53030")),
		Surface: tok("--si-surface", def("#171a22", "#ffffff")),
		Line:    tok("--si-line", def("#2c313c", "#d9d6ce")),
	}
	p.HL = mix(p.Accent, p.BG, 0.35) // the ==highlight== band
	return p
}

func boolIndex(b bool) int {
	if b {
		return 1
	}
	return 0
}

// parseHex reads #rgb or #rrggbb; anything else (a var(), color-mix()) is no colour.
func parseHex(s string) (color.Color, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return nil, false
	}
	h := s[1:]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return nil, false
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return nil, false
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, true
}

// mix is a + (b - a) * (1 - t): t of a, the rest b.
func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	f := func(x, y uint32) uint8 { return uint8((float64(x>>8)*t + float64(y>>8)*(1-t)) + .5) }
	return color.RGBA{f(ar, br), f(ag, bg), f(ab, bb), 255}
}

// hex is a colour as #rrggbb (for lipgloss).
func hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
