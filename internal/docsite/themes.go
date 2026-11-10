package docsite

import (
	"fmt"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// unlisted are built-in themes made for client work: they stay in si but are
// not shown on the site (their names are other companies' trademarks; see
// TRADEMARKS.md).
var unlisted = assets.UnlistedThemes

// themesPage is the site's Themes page, made from the built-in theme list so
// it cannot drift: each theme's name, what it is, and its picture
// (docs/site/themes/<name>.jpg, `make theme-shots`), light beside dark.
func themesPage() string {
	tick := strings.NewReplacer("§", "`") // Markdown code spans inside Go raw strings
	var b strings.Builder
	b.WriteString(tick.Replace(`---
title: Themes
description: Every built-in si theme, light and dark, with the name to use. Or make your own from a CSS file.
---

# Themes

Every theme comes in light and dark (the reader's system setting, or §d§
while presenting). Pick one by name:

§§§markdown
---
title: The Thursday Promise
theme: twilight
---
§§§

or for a whole build, §si build talk.md --theme twilight§, or for a folder,
§theme: twilight§ in its §si.conf§. §si theme list§ prints them all;
§si --export themes§ writes one sample deck per theme. Each picture is the
same slide of the [showcase](live/showcase.md) in that theme.

`))
	for _, n := range assets.ThemeNames() {
		if unlisted[n] {
			continue
		}
		th := assets.Themes[n]
		fmt.Fprintf(&b, "## %s\n\n![%s, light and dark](themes/%s.jpg)\n\n`theme: %s`: %s.", th.Label, th.Label, n, n, th.About)
		if al := assets.ThemeAliases(n); len(al) > 0 {
			fmt.Fprintf(&b, " Also: `%s`.", strings.Join(al, "`, `"))
		}
		b.WriteString("\n\n")
	}
	b.WriteString(tick.Replace(`## Your own

A theme is a CSS file of §--si-*§ tokens: §theme: brand.css§ (or an §https://§
address). Start it with §/* si theme: Brand · extends paper */§ and it sets
only what it changes; the built-in fills in the rest. The easiest way is the
Studio's Theme view (§si --web§, then ⌘K › New theme): colours light and dark,
fonts and sizes beside a live deck, with contrast checks that fix a failing
pair, or from a logo (§si theme from logo.svg§, or From a logo… in the Studio):
its colours become the accents and tinted neutrals, light and dark, every pair
passing WCAG AA. §si theme export§ and §si theme import§ share a theme as one CSS file or
as W3C Design Tokens (§--format dtcg§). §si --export css-rules§ writes a starter
with every token and what it does; see
[Styling](rules/15-styling-themes-and-your-own-css.md).
`))
	return b.String()
}
