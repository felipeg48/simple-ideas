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

<!-- si-video: themes -->

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
	b.WriteString(tick.Replace(`## Make your own

### In the Studio

§si --web .§, then ⌘K › **New theme**: start from any built-in, pick colours
for light and dark, fonts and sizes, and watch a real deck change beside you
at desktop, phone, tablet and 4K. The checks say when a pair of colours is
hard to read, and **Fix** finds the nearest one that passes.

<div class="tour-grid tour-wide-even">
<figure><img src="features/img/themes/new.jpg" alt="The New theme dialog: every built-in, and From a logo" loading="lazy"><figcaption>Start from a built-in, or from a logo</figcaption></figure>
<figure><img src="features/img/themes/editor.jpg" alt="The Theme view: a deck in light and dark beside the colour pickers" loading="lazy"><figcaption>Light and dark side by side; every token with a picker; WCAG AA checks</figcaption></figure>
</div>

### From your logo

§§§bash
si theme from logo.svg --name Stack     # PNG, JPEG, GIF or SVG
§§§

si finds the logo's colours, makes them the accents, tints the backgrounds and
text to match, and moves any colour a little until every pair passes WCAG AA,
light and dark. In the Studio it is **New theme › From a logo…**

<div class="tour-grid tour-3">
<figure><img src="features/img/themes/stack-logo.svg" alt="The si logo: two greens and a violet" loading="lazy" style="background:var(--si-surface);padding:2em;box-sizing:border-box;aspect-ratio:16/10;object-fit:contain"><figcaption>The logo</figcaption></figure>
<figure><img src="features/img/themes/logo.jpg" alt="From a logo: the colours found, light and dark swatches" loading="lazy"><figcaption>The colours found, light and dark</figcaption></figure>
<figure><img src="features/img/themes/logo-editor.jpg" alt="The theme made from the logo, open in the Theme view" loading="lazy"><figcaption>The theme, ready to adjust: all checks pass</figcaption></figure>
</div>

<div class="tour-grid tour-wide-even">
<figure><img src="features/img/themes/stack-light.jpg" alt="A slide in the Stack theme, light" loading="lazy"><figcaption>A slide in the new theme, light</figcaption></figure>
<figure><img src="features/img/themes/stack-dark.jpg" alt="The same slide, dark" loading="lazy"><figcaption>and dark</figcaption></figure>
</div>

### By hand

A theme is a CSS file of §--si-*§ tokens: §theme: brand.css§ (or an §https://§
address). Start it with §/* si theme: Brand · extends paper */§ and it sets
only what it changes; the built-in fills in the rest.

§§§bash
si theme check brand.css               # values and contrast, light and dark
si theme export twilight --format dtcg # share it as W3C Design Tokens (or as CSS)
si theme import brand.tokens.json      # bring one in
si --export css-rules                  # a starter with every token explained
§§§

See [Styling](rules/15-styling-themes-and-your-own-css.md).
`) + tourCSS)
	return b.String()
}
