# 2. Front matter

Optional deck settings: `key: value` lines between two `---` lines at the very
top of the file. Flat keys only; `#` starts a comment line.

```markdown
---
title: The Thursday Promise
author: Ada Lindqvist
date: Spring 2026
theme: studio
mode: auto
accent: "#3657d6"
transition: fade
aspect: 16:9
background: linear-gradient(160deg, #f6f5f1, #e9eefb)
mermaid: off
lang: en
---
```

| Key | Values | Default | Effect |
|---|---|---|---|
| `title` | text | the `# H1` | page title and index card |
| `author`, `date` | text | none | byline on the title slide |
| `year`, `copyright` | text | none | a copyright line on the title slide: `© {year} {copyright}` (either alone works) |
| `theme` | `studio`, `paper`, `mono`, `slate`, `twilight`, `forest`, `sunrise`, `neon`, `newsprint`, … (`si theme list`) | `studio` | built-in look; pictures of each on [the Themes page](https://simple-ideas.org/themes.html) |
| `mode` | `auto`, `light`, `dark` | `auto` | `auto` follows `prefers-color-scheme`; `t` toggles while presenting |
| `accent` | CSS colour | theme accent | sets `--si-accent` |
| `transition` | `fade`, `slide`, `zoom`, `none` | `fade` | default slide transition |
| `aspect` | `16:9`, `16:10`, `4:3`, `1:1`, `9:16` | `16:9` | shape of overview tiles, teleprompter previews and printed pages; slides on screen always fill the window |
| `background` | as the [background directive](#7-backgrounds) | none | background of every slide without its own |
| `mermaid` | `off` or a script URL | pinned CDN URL | how Mermaid fences render |
| `lang` | language tag | `en` | `<html lang>` |
| `status` | short text (`beta`, `preview`, `new`, `deprecated`, …) | none | a badge on the deck's index tile / list row; `beta`/`new`/`deprecated` are colour-coded |
| `version` | text (e.g. `v1.2.0`) | none | an outlined version badge on the index tile / list row |
| `footer` | text | none | small text in a corner of every slide; a slide's own `<!-- footer: … -->` overrides it, `<!-- footer: off -->` hides it (e.g. the title slide) |
| `footer-position` | `bottom-right` (default), `bottom-left`, `bottom-center`, `top-right`, `top-left`, `top-center` (short: `br`, `bl`, `bc`, `tr`, `tl`, `tc`) | `bottom-right` | which corner the footer sits in |
| `watermark` | text, or an image path | none | a faint layer behind every slide — literal text (e.g. `Draft`) is drawn large and angled; an image path is inlined once and centred. In every document type too (labs, articles, guides, books, demos): a fixed layer over the page, on every printed page; a guide or book sets it once in `_nav.md`. |
| `watermark-opacity` | number `0`–`1` | `0.06` | how visible the watermark is |
| `labs-label` | text | `Now, let's do a Lab!` | the heading on every lab slide (see the `labs` directive) |
| `type` | `deck`, `lab` | `deck` | `lab` renders the file as a readable guide page instead of slides; see [Lab documents](#18-lab-documents). Anything else is a `si check` problem |
| `runbook` | path to a `.json` file | none | a command cheat-sheet shown in the **`si play` terminal drawer only** (never in a built deck); see §14 |
| `demos` | markdown files or folders, comma-separated | none | demo scripts shown in the **`si play` terminal drawer only**, with Copy / Run on every shell block; see §14 |
| `type` values | `deck`, `lab`, `article`, `guide`, `book`, `mindmap` | `deck` | `lab` §18, `article` §19, `guide` §20 (guide pages get it from `_nav.md`), `book` §21 (in `_nav.md` only), `mindmap` §24 (with `layout`, `lines`, `nodes`, `map-theme`, `watermark-position`) |
| `subtitle`, `edition` | text | none | books: the cover's subtitle and edition (§21) |
| `session`, `slides`, `duration` | text | none | demo scripts: the facts under the title; `duration: 14 min` is checked against the script's steps (§22) |
| `description` | text | none | documents: the lede under the title, the index tile subtitle and `<meta name="description">` (§19, §20) |
| `edit-url` | URL | none | guides: base of the "Edit this page" link (§20) |
| `header`, `footer-page` | `off` | on | a guide page without the guide's `_header.md` / `_footer.md` (§20) |
| `repo` | `OWNER/REPO`, a github.com URL, or a local clone folder | none | the default repo of `code:` and `diff:` (§13) |
| `linkedin`, `tags`, `canonical`, `cover` | text; comma-separated tags; a URL; a picture | none | `si publish` (§26): the LinkedIn post text, dev.to / Hashnode tags, the article's original URL, the picture of a LinkedIn link post |

An included module's own front matter is ignored; the including deck decides.

The generated `index.html` offers a **Tiles / List** toggle (top-right, remembered
per browser); list view is a table with Course, Status, Slides, Time and a PDF
column. `status`/`version` badges and the PDF column come from each deck's front
matter and `si build --pdf`.
