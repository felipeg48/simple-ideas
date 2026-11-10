# 15. Styling: themes and your own CSS

- `theme:` picks a built-in theme (`studio`, `paper`, `mono`, `slate`,
  `twilight`, `forest`, `sunrise`, `neon`, `newsprint`, `crdb`,
  `acemq`, `acemq-pro`), each with a light and a dark palette. `si theme list`
  prints them with their display names and aliases. Names are matched loosely:
  case is ignored and spaces or underscores count as `-`, so `theme: AceMQ Pro`
  and `--theme "AceMQ Pro"` mean `acemq-pro`. An unknown `--theme` is an error. `crdb` is the CockroachDB
  University brand (cream / ink / purple; aliases `cockroachdb`,
  `cockroachlabs`). `acemq` is the AceMQ brand (orange / teal / ink, Urbanist
  and Mona Sans embedded) with extra components, below. `acemq-pro` (AceMQ
  Pro) is `acemq` plus the rest of the AceMQ deck style guide's layouts.
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

### acemq theme extras

`theme: acemq` adds more than tokens. Only decks using it carry the extras
(fonts and logo inlined, about 145 KB):

- **Background rhythm**, automatic: odd slides are plain with a faint grid and
  an orange glow at the top, even slides are flat `--si-surface-2`.
- **Eyebrow**: `<!-- eyebrow: Context -->` renders as an orange uppercase label
  with a dash. **Highlighter**: `<span class="hl">` in a heading draws an
  orange band that sweeps in. Use one per slide.
- **Footer**: the `footer:` text gets an orange dot, and the AceMQ logo is
  drawn at the bottom left of every slide that has a footer. Set
  `:root { --si-brand-logo: none; }` (in `--css`) to hide the logo, or
  `url(...)` to swap it.
- **Cover and closing**: the title slide is left-aligned. Optional blocks:
  `.lockup` (`<span class="lk-logo">` draws the brand logo, `<img>`,
  `.lk-x` divider, `.lk-name` wordmark; a slide with a `.lockup` drops the
  footer logo so the logo shows once, and keeps the footer text), `.pill`
  with `.pill-mark`,
  `.cover-sub`, `.lead`, `.contact` (a row of `.tag`s).
- **Slide classes** (`<!-- class: ... -->`): `on-ink` (a dark slide in light
  mode; use once or twice), `dense` (smaller table), `us-2` … `us-6` (tint
  table column N orange: "our" column), `us-row` (tint the first body row),
  `vs` (red marker on the second column's `###`).
- **Components** (raw HTML): `.cards` (`c2`, `c4`) of `.card` (`hi` the one
  recommended card, `orange`, `teal`/`ok`, `crit`) with `h4`, `p`, `.big`
  (`<em>` unit in orange) and `.tag` (`orange`, `teal`, `crit`); `.tl` of
  `.ev` incidents (`.when`, `.what`, `.impact`); `.steps` of `.st` (`.num`,
  `h4`, `p`); `.callout` (`warn`) under a table or grid; verdict marks
  `.y`, `.n`, `.p`, `.u` in table cells.
- Code blocks are a dark terminal in both modes; tables are cards; motion
  (rise, pop, highlighter) is off under reduced motion and in print.

```markdown
<!-- eyebrow: Two paths -->
<!-- class: us-3 -->
<!-- footer: Source: AceMQ docs -->
## Same workload, <span class="hl">half the nodes.</span>

| | Today | AceMQ |
|---|---|---|
| Nodes | 6 | <span class="y">3</span> |

<div class="cards c2">
<div class="card hi"><span class="tag">Recommended</span><h4>AceMQ</h4><p>Quorum by default.</p></div>
<div class="card crit"><span class="tag">Today</span><h4>Mirrored</h4><p>Split brain on a lost AZ.</p></div>
</div>
```

Lab documents (`type: lab`) under `acemq` get the fonts and palette.

### acemq-pro theme extras (AceMQ Pro)

`theme: acemq-pro` is everything in `acemq` (above) plus the layout catalogue
of the AceMQ deck style guide, as raw-HTML components. All of them reflow:
grids use auto-fit and stack on narrow or portrait slides, and each pops in
with a stagger on the active slide. `si --export themes` writes a full
AceMQ Pro sample deck that uses every one.

| Component | Markup | Use for |
|---|---|---|
| Fact grid | `.heard` of `.hc` (`.k` label, `.big` with `<em>` unit, `p`; `hi`) | "What we heard" |
| Incidents + verdict | `.inc` of `.ic` (`.when`, `h4`, `p`), then `<p class="verdict">` (`<b>` in orange) | outages with one root cause |
| Argument + visual | `.split2` with `.blist` (`li.ok`, `li.crit`) and `.topo` (a card for an SVG, `.cap`) or `.zones` | a claim next to a diagram |
| Zones | `.zones` of `.zg` (`.zl` label, `.zc` cells: `ok`, `us`, `hot`, `down`) | before / after topology |
| Options | `.opts` of `.opt` (`.ol` letter, `.tag`, `h4`, `p`, `dl` of `<div><dt><dd>`; `hi`) | options A–D, one recommended |
| HTML table | `table.tbl` with `.us` cells | a matrix with "our" column |
| Fact rows | `.facts` of `.fr` (`.fn` number with `<em>` unit, then a `div` with `h4`, `p`) | big number, then why |
| Feature grid | `.feat` of `.fc` (`.icon-chip`, `h4`, `p`) | product capabilities |
| Benchmark | `.bench` of `.brow` (`style="--w:84%"`; `us` row orange; `.bl`, `.bar`, `.bv`), then `.bks` of `.bk` (`.big`, `p`; `hi`) | measured results |
| Flow | `.flowline` of `.fs` (`h4`, `p`; `hot` pulses), then `.mnotes` (`p`, `p.warn`) | migration, process |
| Plans | `.poc` of `.pc2` (`.tag`, `h4`, `p`, `dl`; `hi`) with a `.term2` terminal (`.c` comment, `.k` prompt, `.ok` output) | trial vs PoC |
| Cover floaters | `.floaters` of three `.fl` (text, `<img>` or `<svg>`) on the title slide | tilted tiles that drift; hidden on narrow slides |
| Parts | `.icon-chip` (`--sm`, `--teal`, `--ink`; holds an inline `<svg>` or a glyph), `.btn`, `.btn--ghost` | icons, a call to action |
| Motion | `.r` rise, `.f` fade, `.pop` pop, `.pulse` ring ×3, `.grow-x` draw in; `style="--i:N"` staggers | hand-placed raw HTML |

`.pop` is the guide's `.p`: in si `.p` stays the "partial" verdict mark.
There is no icon font (a deck stays self-contained), so icon chips take an
inline SVG.

```markdown
<!-- eyebrow: Options -->
## Four ways forward, ==one we recommend.==

<div class="opts">
<div class="opt"><span class="ol">A</span><span class="tag">Today</span><h4>Keep as is</h4><p>Hand-run clusters.</p></div>
<div class="opt hi"><span class="ol">B</span><span class="tag">Recommended</span><h4>AceMQ Broker</h4><p>Quorum by default.</p><dl><div><dt>Lost zone</dt><dd>Alert</dd></div></dl></div>
</div>
<p class="verdict">Both keep your clients. <b>Only B removes the outage.</b></p>
```

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

A `.css` theme file replaces the built-in theme, so it must set the colour,
font and layout tokens itself (`si --export css-rules` writes a complete,
commented starter with every token and its studio default). The same file
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
