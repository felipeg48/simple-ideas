# 15. Styling: themes and your own CSS

- `theme:` picks a built-in theme (`studio`, `paper`, `mono`, `slate`,
  `twilight`, `forest`, `sunrise`, `neon`, `newsprint`), each with a light and a dark palette. `si theme list`
  prints them with their display names and aliases. Names are matched loosely:
  case is ignored and spaces or underscores count as `-`, so `theme: Twilight`
  and `--theme TWILIGHT` mean `twilight`. An unknown `--theme` is an error.
- `theme:` (or `--theme`) can instead be a path ending in `.css`: the file's
  contents become the `si.theme` layer, so you ship a whole palette from one
  file. Define the `--si-*` tokens (see `si --export css-rules`); resolved
  relative to the deck, then the current directory. A missing file is a `si
  check` problem. `examples/css/harbor.css` is a complete example theme.
- `theme:` can also be an `https://…` (or `http://…`) URL — for example a raw
  file from a GitHub repo. It is fetched **at build time** and cached for an
  hour under the temp dir, and its contents are inlined into the deck, so the
  built HTML stays self-contained (no runtime network dependency). An
  unreachable URL is a `si check` problem (a fresh-enough cache is reused if the
  network is down); builds that use a URL theme need network access the first
  time.
- `--theme <name|file.css>` on `si play` / `si build` overrides the deck's
  `theme:` front matter, so one set of decks can be built in several themes.
- `si --export themes [--output dir]` writes one sample deck per built-in theme;
  `si play dir` (or `si build dir`) then shows them side by side as a gallery.
- `--css <file.css|dir>` (repeatable) loads your stylesheets after the theme;
  `si play` watches them and reloads, static output inlines them (relative
  `url()` files become data URIs).
- Cascade layers: `@layer si.theme, si.user`. Your CSS lands in `si.user`, so
  it overrides the theme without specificity games. The layout mechanics
  (slide sizing, fluid type, container queries, transitions, steps) are
  unlayered and stay in charge: **themes change tokens and looks, never the
  fluid layout.** Use relative units.
- The contract is the `--si-*` custom properties and the `.si-*` hooks.
  `si --export css-rules --output my-theme.css` writes a commented starter
  file listing every token with its default and every hook with a one-line
  description. `examples/css/harbor.css` is a complete example theme.

```css
:root { --si-accent: #e4572e; --si-font-heading: Georgia, serif; }
:root[data-si-mode="dark"] { --si-bg: #0b1a28; --si-fg: #f1ece2; }
.si-step.is-latest { color: var(--si-accent-2); }
```

### Make a theme: `extends`, the Theme view, check, export, import

- **`extends`**: a theme file whose first comment starts with
  `si theme: <Name> · extends <built-in>` gets that built-in's tokens first and
  its own after (the file wins), so it only needs the tokens it changes:
  headings, padding, shadows and dark mode come from the built-in. Without the
  header a `.css` theme has nothing underneath, as before. An unknown built-in
  is a `si check` problem.
- **The Studio's Theme view** (`si --web`): ⌘K › *New theme* (or File › New
  Theme… in the Mac app) asks for a name and a built-in to start from and
  creates `themes/<name>.css`; opening any `.css` theme shows it as controls:
  colours light beside dark with a picker, fonts, sizes on sliders (never
  `px`), motion, beside a specimen deck drawn with it, light and dark, at four
  screen sizes, and the contrast checks with **Fix**. Every change is one
  undoable edit of the file, which stays plain CSS (the Text view edits it
  too). *Use in this folder* writes `theme:` into `si.conf`; a `.css` path in
  `si.conf` is relative to `si.conf`.
- **Checks**: `si theme check <file.css>` (or `si check <file.css>`) reports
  unknown tokens and values of the wrong kind (errors), and every
  text/background pair under its WCAG AA minimum in light or dark (warnings,
  each with a colour that passes): body and muted text 4.5:1 on the
  background and on surfaces, text on the accent 4.5:1, accents and status
  colours 3:1.
- **Share**: `si theme export <name|file.css> [-o file]` writes one
  self-contained file (a built-in in full; a file's `url()`s inlined);
  `--format dtcg` writes W3C Design Tokens JSON (`.tokens.json`: colours with
  their dark value in `$extensions`, font families, durations, numbers, `rem`
  and `em` sizes; other tokens are listed, never guessed).
  `si theme import <file.css|file.tokens.json|https://…> [-o dir] [--use]`
  checks a theme (a broken value refuses it), writes it to `themes/<name>.css`
  and, with `--use`, sets it in `si.conf`.
- The index page of `si build` and `si play` follows the decks' theme, a
  `.css` theme included.

```css
/* si theme: Acme · extends paper
   tokens and looks only; edit here or in the Theme view of si Studio */
:root {
  --si-accent: #0b3d91;
  --si-font-heading: "Fraunces", Georgia, serif;
  --si-h1-size: 3.4em;
}
:root[data-si-mode="dark"] {
  --si-accent: #8fb3ff;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-si-mode]) {
    --si-accent: #8fb3ff;
  }
}
```

Use it with `theme: themes/acme.css` in a deck's front matter (a path from
the deck) or in `si.conf`. Load a web font like Fraunces with `fonts:` (below).

### Themes for documents (labs, articles, guides)

Every theme styles documents too, from the same tokens as slides. Documents
read for minutes, so they add a few tokens of their own. Each is **unset by
default** and the built-in CSS falls back to the value below, so a theme (or a
`.css` theme file) that never sets them still works; set only what you want
to change.

| Token | Fallback | What it sets |
|---|---|---|
| `--si-font-prose` | `var(--si-font-body)` | font of the text (articles, guides, labs) |
| `--si-prose-size` | `1rem` | size of the text (paper `1.06rem`, mono `.94rem`) |
| `--si-prose-leading` | `1.7` | line height of the text |
| `--si-doc-measure` | `78ch` | longest line; articles use 88% of it; code and tables may use the whole column |
| `--si-lede-size` | `1.25em` | the `description:` lede under an article or guide page title |
| `--si-nav-bg` | `surface-2` mixed into `bg` | background of a guide's navigation |
| `--si-nav-active` | `accent` at 12% | the current page in the navigation |
| `--si-step-badge` | `var(--si-accent)` | the number badge of a `## Step N:` heading (labs, guides) |

A custom theme for documents is an ordinary theme file: set the colour tokens
(light and dark) and, if you like, the document tokens. Hook classes for
finer control are listed by `si --export css-rules` (`.si-doc[data-si-doc]`,
`.si-lede`, `.si-doc-nav`, `.si-doc-crumb`, `.si-step-num`, `.si-doc-site`,
...).

```css
/* harbor-docs.css: use with  theme: harbor-docs.css  in _nav.md or a page */
:root {
  --si-bg: #f7f4ee; --si-fg: #1d2430; --si-muted: #5b6574; --si-surface: #ffffff;
  --si-surface-2: #ece7dd; --si-line: #dcd4c6; --si-accent: #0b6e8a; --si-accent-2: #a2462b;
  --si-on-accent: #ffffff; --si-ok: #2f7d4a; --si-warn: #b7791f; --si-crit: #b42318;
  --si-font-body: "Iowan Old Style", Georgia, serif;
  --si-font-heading: ui-sans-serif, system-ui, sans-serif;
  --si-font-mono: ui-monospace, Menlo, monospace;
  --si-heading-weight: 700; --si-heading-tracking: -0.01em; --si-heading-case: none;
  --si-radius: .4em; --si-shadow: none;
  /* documents */
  --si-prose-size: 1.08rem; --si-doc-measure: 72ch; --si-step-badge: var(--si-accent-2);
}
:root[data-si-mode="dark"] {
  --si-bg: #11161d; --si-fg: #e8e4dc; --si-muted: #9aa3b0; --si-surface: #19202a;
  --si-surface-2: #202834; --si-line: #2c3644; --si-accent: #5cc3e0; --si-accent-2: #f0a27c;
  --si-on-accent: #11161d;
}
/* repeat the dark block inside @media (prefers-color-scheme: dark) { :root:not([data-si-mode]) { … } } */
```

A `.css` theme file without an `extends` header replaces the built-in theme,
so it must set the colour, font and layout tokens itself (`si --export
css-rules` writes a complete, commented starter with every token and its
studio default); with `si theme: Name · extends studio` it sets only what it
changes. The same file
styles decks, articles, guides and labs. To change only a few things on top of
a built-in theme, keep `theme: studio` (or any) and pass the overrides with
`--css`, which lands in the `si.user` layer after the theme.

### Fonts

`fonts:` gives the headings, the body text and the code free web fonts.
They are fetched when the deck or page is built and inlined as `@font-face`
data URIs, so the file still works offline and from `file://`. Put the key in
a deck's or page's front matter. A guide's `_nav.md` or a site's `_site.md`
sets it for every page, and `si.conf` sets it for every deck.

```yaml
fonts: heading=Fraunces:600, body=Inter:400;700, mono=JetBrains Mono
```

| Entry | Font |
|---|---|
| `heading=Fraunces:600` | Google Fonts (the default source), weight 600; one heading weight also sets `--si-heading-weight` |
| `body=bunny:Inter:400;700` | Bunny Fonts (the Google catalogue from a European, no-tracking CDN) |
| `heading=fontshare:Satoshi:700` | Fontshare (Indian Type Foundry's free faces: Satoshi, General Sans, Clash Display …) |
| `mono=fontsource:Fira Code` | Fontsource (Google's and other open-source fonts, via jsDelivr) |
| `heading=fonts/Brand.woff2:600` | a local `.woff2`, `.woff`, `.ttf` or `.otf` file you have the right to embed |

- Roles: `heading` (`--si-font-heading`), `body` (`--si-font-body` and
  `--si-font-prose`) and `mono` (`--si-font-mono`). They override the theme's
  fonts and nothing else; your `--css` still wins over them.
- Weights are 100 to 900 in steps of 100, separated by `;`. The defaults are
  700 for heading, 400;700 for body and 400 for mono. Every weight you list
  adds to the file, typically 20 to 60 KB per weight.
- A split family keeps only the latin and latin-ext character sets unless
  `fonts-subset:` names others: `fonts-subset: latin, greek` (or `cyrillic`,
  `vietnamese`, `cyrillic-ext` …; also in `si.conf`). Each set adds to the file.
- A stylesheet and its files are cached for good: after one online build,
  `--offline` builds the same output. `si check` reports a family or weight
  the source does not have.
- Google, Bunny and Fontsource fonts are under the SIL Open Font License or
  Apache 2.0, which allow embedding. Fontshare fonts are under the ITF Free
  Font License, free for personal and commercial use. A local file is your
  responsibility.
- `fonts: off` turns off fonts that `si.conf` or a `_nav.md` would set.

Example: the site example's `_site.md` sets
`fonts: heading=fonts/fraunces-600.woff2:600` (Fraunces, OFL, licence beside
the file).
