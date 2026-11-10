# 23. Sites

A **site** is a folder with a `_site.md`: the guide format (front matter plus
a nested markdown list), but every page gets the site's **navigator** instead
of the guide bar and sidebar. Pages are plain pages by default (`type: page`:
no `---` slide breaks, no slide directives), and any page may say
`type: article` (byline, contents) or `type: deck` (a talk, built as slides).
A link in the list to a folder that holds its own `_nav.md` or `_site.md`
(a guide, a book, another site) opens that one; it is built and checked on its
own, and links into it are not checked by the parent.

````markdown
---
title: Northwind Engineering
description: How Northwind Logistics moves forty thousand orders a day.
navigator: topbar
logo: Northwind Engineering
cta: Run the lab → docs/index
theme: studio
---

- [Home](index.md)
- Writing
  - [Retries without the stampede](blog/retries.md)
  - [The order journey, the talk](talks/journey.md)
- [Docs](docs/)
- [About](about.md)
````

| Key | What |
|---|---|
| `navigator:` | `topbar` (default: logo left, menu, call to action right), `topbar-centred` (menu left, logo centred), `mega` (sections open as wide panels), `sidebar` (a sticky column on the left). On narrow screens every navigator folds its menu behind a **Menu** button. |
| `logo:` | text, or an image path (inlined like any image); an image followed by text (`logo: brand/mark.svg simple-ideas`) shows the picture with the name beside it |
| `favicon:` | the browser tab icon, an image path (inlined); without it pages carry no icon |
| `touch-icon:` | the home-screen icon (`apple-touch-icon`), an image path |
| `og-image:` | the picture link previews show (`og:image`, `twitter:card`), an absolute `https://` URL |
| `cta:` | `Label → page` (or `→ https://…`, `→ #modal`): the highlighted button |
| `version:` | a badge beside the logo, e.g. `v2.5.0 → https://…/releases/tag/v2.5.0` (the link is optional) |
| `title: off` | on a page: no `<h1>` (a home page that opens with its own content) |
| `cta:` on a page | a large button under that page's title and description, same `Label → page` form (a landing page's hero); hidden on paper |

`favicon:`, `touch-icon:` and `og-image:` also work in any deck's or document's
front matter. si never adds an icon or branding of its own to pages you build.

```markdown
---
title: Northwind docs
logo: brand/mark.svg Northwind
favicon: brand/favicon.svg
touch-icon: brand/touch.png
og-image: https://docs.northwind.example/brand/card.png
---
```
| `header: off` / `footer-page: off` | on a page: leave out `_header.md` / `_footer.md` |

An item without a link (`- Writing` above) is a menu section: its children
open as a dropdown. `_header.md` and `_footer.md` in the site folder are
rendered above the navigator and below every page, with their links rewritten
for each page. `/` searches every page of the site; `watermark:` works as in
every document. `si build site/`, `si check site/` and `si serve site/` handle
the whole folder. The index page title is `--site-title <text>` (the old
`--site <text>` still works, with a warning).

### Plugins

Plugins are directives that work in every document (pages, articles, guides,
books, labs, demos), so in every site. They are plain HTML plus a few lines of
runtime; nothing is fetched. `badge`, `logo`, `qr` and `cards` also work on slides;
the others, on a slide, are reported as a problem.
Options follow the text after a `·`: `key=value`, or a kind word (`info`,
`ok`, `warn`, `crit`). Any other `· part` stays in the text, so
`<!-- badge: Prepared for Northwind · Session 1 of 5 -->` keeps its dot.

```markdown
<!-- alert: The lab moves to the new cluster on 24 November. · until=2026-11-25 · warn -->
<!-- badge: Pipeline healthy, p95 checkout 410 ms · ok -->
<!-- event: 2026-11-20T17:00 Live: replaying Black Friday 2025 · Save a seat → #seat -->
```

| Plugin | What |
|---|---|
| `alert: text · kind · until=YYYY-MM-DD` | a dismissible band; the reader's dismissal is remembered in that browser; hidden after `until`. Kinds: `info` (default), `ok`, `warn`, `crit`. |
| `badge: text · kind` | a status pill, same kinds; on a slide, the pill paragraph a cover or closing slide carries |
| `cards: 2` | the `###` headings that follow become a grid of cards (2, 3 or 4 columns, or as many as fit): `### Title · tag · colour` (the title may be a link, `### [Dock board](work/dock.md) · Client`), colour one of `teal`, `orange`, `ok`, `crit`, `hi` (the highlighted card); the grid ends at the next `##`, `---` or directive; a `<!-- step -->` before it reveals the whole grid |
| `logo: [image] × Partner · alt=Brand` | a logo lockup: the image (or, with none, the theme's own logo: `--si-brand-logo`), a bar, and the partner's name; on slides and pages |
| `cloud: id · hidden` | a word cloud of the room's short answers under `si play --audience` (`hidden`: on the next click) |
| `poll: id · option · option … · multiple · hidden` | a live poll the room answers on their phones under `si play --audience` (`hidden`: results on the next click); elsewhere the list of options (above) |
| `qr: text or URL · caption=… · style=dots · logo=path · size=large · ec=H` | a QR code (below); `qr: audience` is the live follow-along code of `si play --audience` |
| `event: YYYY-MM-DDTHH:MM text · Label → link` | a corner toast with a live countdown, shown from two days before the time until it starts |
| `accordion` | the `###` questions that follow become a one-open-at-a-time list; it ends at the next `##`, `---` or directive |
| `modal: id` … `end` | the blocks up to `<!-- end -->` become a dialog; any `[link](#id)` (or the `cta:`) opens it, and so does a URL ending in `#id`. A document part ends at `##`, so use `###` headings inside a modal. |
| `carousel: auto=7s` | the image paragraphs that follow slide one at a time: arrows, dots, swipe and ←/→; `auto=` advances it (never with reduced motion); printing shows every image |
| `embed: …` | another document, inside this page (below) |

```markdown
<!-- carousel: auto=7s -->

![Pickers at aisle 14](images/aisle.svg "Evening wave, aisle 14")

![The packing line](images/packing.svg "Packing, stations 3 to 5")

<!-- accordion -->

### Can I run the lab on a laptop?

Yes. It needs Docker and about 4 GB of memory.

### Is the lab the real code?

It is a trimmed copy with stubbed providers.
```

### Embed

`<!-- embed: … -->` puts another si document inside the page:

| Form | Result |
|---|---|
| `embed: blog/retries.md` | a card: type, title, description, reading time, linking to the page |
| `embed: blog/retries.md#what-we-changed` | that section's text, with a *From …* link; give a heading a fixed id with `## What we changed {#changes}` so the link survives a reworded title |
| `embed: blog/retries.md full` | the whole body |
| `embed: talks/journey.md 1-3` | a deck: its slides, live, in a frame (as `<!-- slides: -->`) |
| `embed: docs/` | a guide, book or site folder: a card with its first pages or chapters |
| `embed: talks/journey.md 1-3 bare` | the same without the caption (`full bare`: the body without its title and *From* line) |

Cards, on a slide or a page:

```markdown
<!-- cards: 2 -->

### High availability · In seconds · teal

One cluster, one site. **Recovers by itself.**

### Disaster recovery · By decision · orange

Two clusters, two sites. **Recovers by decision.**
```

On a cover slide:

```markdown
# Message flows, end to end

<!-- logo: × Northwind · alt=Harbor -->

<!-- badge: Prepared for Northwind · Session 1 of 5 -->
```

### QR codes

`<!-- qr: … -->` draws a QR code for a link or any text, on a slide or a page.
It is an SVG made at build time (nothing is fetched, no script), sized by its
container like any figure; on a slide it fits a square in either orientation.
A web address is also a link, so a reader on a laptop can click it.

```markdown
## Everything from today

<!-- qr: https://simple-ideas.org/rules/ · caption=simple-ideas.org/rules · size=large -->
```

| Option | What |
|---|---|
| `caption=text` | a line under the code |
| `size=small\|medium\|large` | the width (default medium); `--si-qr-size` overrides it |
| `style=dots` (or the word `dots`) | round modules and rounded finder squares; `square` is the default |
| `logo=path/mark.svg` | a picture in the middle; the code switches to error correction H, and a picture too big to read past is a problem (`logo-size=20` is its share of the width, in percent) |
| `ec=L\|M\|Q\|H` | error correction (default M, raised when it costs no size) |

The code is dark on light in every theme (`--si-qr-fg`, default `#111`, on
`--si-qr-bg`, default `#fff`): a light code on a dark ground is one many
phones do not read. To make one as a file, see `si qr` in the manual
(`si qr https://… --png --svg --pdf`).

Start one with `si new mysite --type site` (or `--type website`): `_site.md`,
a home page and an about page. Example: `si --export example --type site`
writes this site; `si serve si-example` serves it.

### Start from a kind

`--kind` (or the kind as the `--type`) starts a website made for one job,
with real content to replace:

```bash
si new marta --type website --kind resume     # or: si new marta --type resume
si new tomas --type portfolio
si new dockside --type landing
```

| Kind | What you get |
|---|---|
| `resume` | `index.md`: name, title, contact lines, experience, skills, education, set to print on A4 or Letter (the navigator and buttons stay off the paper; print the page or save it as PDF from the browser); `projects.md` (cards) and `about.md` |
| `portfolio` | a home with project cards, each with a picture (`images/*.svg`) and a linked title; one `type: article` case study per project under `work/`; `about.md` |
| `landing` | a product page: a hero with a page `cta:` button, features as cards, results as a table, questions as an accordion, a QR code, a `#demo` modal opened by the navigator's `cta:`; `pricing.md` and `_footer.md` |

A landing page's hero is its title, its `description:` and its own `cta:`:

```markdown
---
title: Every van, every door, one screen
description: Dockside plans your loading docks minute by minute.
cta: Start a free 30-day trial → pricing
---
```
