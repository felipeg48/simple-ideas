# simple-ideas

[![CI](https://github.com/felipeg48/simple-ideas/actions/workflows/ci.yml/badge.svg)](https://github.com/felipeg48/simple-ideas/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/felipeg48/simple-ideas?sort=semver)](https://github.com/felipeg48/simple-ideas/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/felipeg48/simple-ideas)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/felipeg48/simple-ideas)](https://goreportcard.com/report/github.com/felipeg48/simple-ideas)
[![License](https://img.shields.io/github/license/felipeg48/simple-ideas)](LICENSE)

**Docs, features and live examples: [simple-ideas.org](https://simple-ideas.org/)**

`si` turns markdown into animated slide decks that fill any screen. Write the
talk in markdown, run `si play` to present it with live reload, or build
self-contained HTML files you can mail, host or open from a USB stick.

![An animated SVG canvas: the same picture changes state slide by slide](docs/screenshots/canvas-declined.png)

- **One persistent SVG canvas** across slides: parts appear, fail, recover
  and move as the story goes, driven by state words in the markdown and CSS
  transitions in the SVG. Helpers animate reveals, pulses, growing bars,
  travelling dots and ticking counters — or scaffold one with `si new-canvas`.
- **Stage diagrams:** a ` ```stage ` fence generates themed, animated flow
  diagrams (chains, fan-out, merges) with a Mermaid-like syntax — native,
  offline, and step-drivable. See [docs/stage-diagrams.md](docs/stage-diagrams.md).
- **Fluid, never fixed:** no 1280×720 canvas; slides reflow from a phone to a
  4K screen, in portrait too.
- **Build steps** that reveal items and drive the canvas and stage state.
- **A real teleprompter:** notes in large type with auto-scroll, mirror mode,
  per-slide and total timers with pace, per-slide rehearsal timings, previews,
  synced with the audience window.
- **Live terminal drawer** (`si play` only): press `` ` `` to slide up a tabbed,
  real shell and run commands mid-demo (`docker ps`, `cockroach start-single-node`,
  `cockroach sql`) without leaving the slides. Loopback-only and never in a built
  deck — see [docs/terminal.md](docs/terminal.md).
- **Present with ease:** jump to any slide with the `/` find palette (search
  titles and text, or type a number); a full-screen break timer (digital or
  analog) with an optional animation; and a pen to annotate the slide live.
- **Backgrounds and images:** colours, gradients, images and muted video
  backgrounds with overlays; sized, aligned, captioned and full-bleed images.
- **Self-contained output:** CSS, JS, SVGs, images and video in one HTML file;
  or `si build --pdf` for a PDF per deck. Open on a phone with a QR code.
- **Articles and guides, not only slides:** `type: article` for a long read
  (lede, reading time, contents on the right), and a folder with a `_nav.md`
  for a multi-page guide (sidebar, breadcrumb, Previous / Next, shared header
  and footer, checked links and anchors). Labs (`type: lab`) for hands-on
  work. `si serve` shows them all live; `si --export example --type guide`
  gives a working one to copy (RULES.md §18–20). **Books**: `type: book` in
  `_nav.md` adds a cover, numbered chapters, sections, figures and tables,
  cross-references, an index, and a one-file edition and PDF (§21). **Demo
  scripts** (`type: demo`) are the presenter's run sheets: facts, `> Say:`
  lines, commands with expected output, and Copy / Run in the `si play`
  terminal (§22).
- **Sites (v1.0):** a folder with a `_site.md` becomes a site: a navigator
  (`topbar`, `topbar-centred`, `mega`, `sidebar`) with dropdowns and a phone
  menu, plain pages beside articles, talks, guides and books, and plugins for
  alerts, badges, event countdowns, accordions, modals, image carousels and
  embedding any other document or deck's slides. Still one self-contained
  HTML file per page; `si --export example --type site` (§23).
- **Mind maps (v1.12):** `type: mindmap` turns a nested list into a map
  (layouts map, right, org, radial; curved, straight, orthogonal or tapered
  lines; box, pill or underline topics; five map themes). Topics carry badges,
  pictures, sticky notes, highlighted code and links to slides or pages;
  `→ other.md` drills into another map with a zoom (inlined, so it works
  offline). Drag topics, fold branches, `/` searches every map, Play walks the
  branches. A ```` ```mindmap ```` fence puts one on a slide or in an article
  (§24).
- **3D mind maps (v1.13):** `view: 3d` draws the same maps as pills on a floor
  (isometric or perspective camera), cables with flowing dots, and the maps a
  topic links to as glass floors above it; fly in with a double-click, search
  with `/`, Play branch by branch, switch 2D/3D on the page. three.js is inlined
  only into pages with a 3D map.
- **3D kinds (v1.14):** `kind: bricks` builds the map from studded bricks on a
  baseplate with printed labels; `kind: toy` makes it a cartoon island (outlined
  candy shapes, roads, trees, clouds). A button cycles Blocks, Bricks and Toy.
- **The mind map builder (v2.0):** the Studio (`si --web` and the desktop app)
  opens a mind map as a map you edit directly: click a topic, Tab adds a child,
  Enter a sibling, type to rename, drag a topic onto another to move it there,
  Shift-drag to place it, panels for badges, links, pictures, sticky notes, code
  and the page's look. si writes every change back into the Markdown (the rest
  of the file stays as it was), as one undoable edit. Also new:
  `si build map.md --svg --png --pdf` (each map as SVG and PNG, one PDF page per
  map with working links), `si mindmap import` / `export` for OPML, FreeMind,
  XMind and Mermaid, `steps` on a slide to reveal one branch per click, and
  `flow: on` for dots running along the lines (§24).
- **Code windows (v2.1):** `window=mac|windows|terminal` on a code fence draws
  it as a window on a backdrop, with numbered callouts you drag into place
  (`<!-- callout: 3 text at=dx,dy -->`); fences with the same `id=` are a show
  whose lines glide from step to step, in documents (← → Play), on slides (one
  click per step, and across slides) and in mind maps. `si code file.md --png
  --svg --gif` exports every step and the animated show; the Studio's Window view
  edits it all (§25). Also: a fence without a language (``` title="config.csproj" ```) or
  in pandoc form (```{.xml title="x"}```) reads right; the title's extension names
  the language. Code is coloured more like an editor: calls and type names too,
  and labels read C#, JavaScript, Shell instead of CSHARP.
- **v2.1.1:** the desktop app runs a newer si you installed and tells you when a
  new release is out; the Studio editor completes code block and mind map
  options, include options and callout lines; a step's code is edited inside
  the Window view; a callout dragged in `si play` keeps its place in the
  Markdown; 3D mind maps on a phone start closer and offer a List view.
- **v2.2.0:** in `si play --terminal` a code show advances one step per click,
  with its title and callouts; Terraform/HCL, Protobuf, GraphQL, F# and Elixir
  are highlighted; `si build --share-assets` writes three.js once for a folder
  of 3D maps instead of into every page; pointer tests click, double-click and
  drag like a person on 3D maps, 2D maps and code windows.
- **v2.3.0:** `si publish` posts a deck to LinkedIn as a document post (its
  PDF, a swipeable carousel in the feed) and an article as a LinkedIn link or to
  dev.to / Hashnode, always as a dry run first (`--yes` sends; see
  `docs/publish.md`); `--mp4` films a code show or a mind map's Play (needs
  ffmpeg); `make docs-site` builds the rules into a browsable site with si itself
  (one page per RULES.md section, live examples on the home page); the Studio
  opens an empty folder with a start gallery of live starter thumbnails and a
  skippable tour.
- **v2.3.1:** the desktop Help menu opens the start gallery and the tour;
  `--mp4` also films decks (every slide and step; `--dwell` for a teaser) and 3D
  mind map Play; a LinkedIn link post carries a picture (`cover:`); ⌘K reaches
  the Studio from the map editor and previews; `.github/workflows/pages.yml`
  publishes the docs site to GitHub Pages once Pages is on (Settings ▸ Pages ▸
  Source: GitHub Actions) and the repository variable `SI_PAGES` is `on`
  (`gh variable set SI_PAGES --body on`).
- **v2.4.0:** the Studio's **History** panel (changed files and their diff,
  commit, a file's earlier versions with restore; git); **review links**
  (`si serve . --review`): reviewers on your network comment on slides and
  sections, the Studio's Review panel follows them live; `si check --budget`
  (and `budget.*` in si.conf) fails pages over 1 MB (3D maps 1.5 MB), and CI
  checks the examples; the map editor selects several topics and copies, cuts
  and pastes branches as Markdown lists; the terminal Studio edits mind map
  trees and code window looks.
- **In the terminal (v1.7):** `si play talk.md --terminal` presents a deck in
  the terminal itself, in the theme's colours, with block-letter titles, builds,
  code, tables, cards and pictures in half-block pixels; `--notes` in a second
  terminal is the teleprompter, following the presenter. Over SSH and in tmux.
  `si --terminal [dir]` (v1.8) is the Studio as a keyboard app: files, live
  preview, problems, New, Build, storyboard, inspector, Rehearse and Present,
  editing in your `$EDITOR`; and Lab (v1.9): the slide, the deck's runbook and demo
  commands, and a shell (a tmux pane, or output streamed in place). See
  `docs/terminal-studio.md`. Since v1.10 pictures are real pictures in Kitty,
  Ghostty, WezTerm and iTerm2, and `--notes` also follows a deck open in a browser.
- **si Studio for macOS (v1.6):** the browser editor (`si --web`) as a native
  app: its own windows, Dock icon, folders dropped on it open, teleprompter and
  full screen as app windows. Download `si-Studio-<version>-macos.zip` from a
  release (unsigned: right-click ▸ Open the first time). See `docs/desktop.md`.
- **Free web fonts (v1.1):** `fonts: heading=Fraunces:600, body=Inter:400;700`
  from Google Fonts, Bunny Fonts, Fontshare, Fontsource or a local file,
  fetched once, cached, and inlined, so the file stays self-contained (§15).
- **Markdown or AsciiDoc:** author decks in `*.md` or `*.adoc` — both feed the
  same pipeline ([docs/asciidoc.md](docs/asciidoc.md)).
- **Your own look:** four built-in themes (incl. the CockroachDB `crdb` brand),
  a CSS theme file or URL, or your CSS on a documented `--si-*` token and
  `.si-*` hook contract.

## Install

**Download a release binary** (no Go needed) from the
[releases page](https://github.com/felipeg48/simple-ideas/releases) — pick your
OS/arch, then:

```bash
# macOS / Linux
chmod +x si_* && sudo mv si_* /usr/local/bin/si
# Windows: move si_*.exe somewhere on your PATH
```

**Or build from source** (Go 1.26+). A one-off setup script checks for Go and
builds `bin/si`:

```bash
./setup.sh            # macOS / Linux
.\setup.ps1           # Windows (PowerShell)   — or: setup.bat
```

Or the usual Go tooling:

```bash
go install github.com/felipeg48/simple-ideas/cmd/si@latest
# or, from a clone — builds bin/si, copies it to ~/.bin, and offers to add
# ~/.bin to your PATH in .zshrc/.bashrc (override dir: make install INSTALL_DIR=~/bin)
make install
```

**Update** an installed binary in place (checksum-verified, no reinstall):

```bash
si selfupdate
```

## Quick start

```bash
si new talk                 # talk/talk.md uses every rule
si play talk --open         # present with live reload at http://127.0.0.1:7070/
si talk --output html/      # one self-contained HTML file per deck + index.html
si courses/ --site "CockroachDB Courses" -o site/   # a tiled landing page over many decks
si check talk               # validate; exit 1 on a problem
```

A deck is plain markdown:

```markdown
# The Thursday Promise

How one order crosses five teams and still arrives on time.

---

## Payment: declined, then paid

<!-- svg: assets/journey.svg +declined -->
<!-- steps: . -declined,retry,paid -->
- 21:04: the card fails a fraud check
- 21:05: an automatic retry goes through

Notes: (1.5 min)
The token reaches payments and turns red.
```

The full format, with an example for every rule: [RULES.md](docs/RULES.md)
(also `si rules`).

Or, if you prefer AsciiDoc, name the file `*.adoc` — si converts it and every
feature still works ([docs/asciidoc.md](docs/asciidoc.md)):

```asciidoc
= The Thursday Promise
:author: Logistics

How one order crosses five teams and still arrives on time.

== Payment: declined, then paid

NOTE: the token reaches payments and turns red.

* 21:04: the card fails a fraud check
* 21:05: an automatic retry goes through
```

## Commands

| Command | What it does |
|---|---|
| `si play <dir\|file.md> [--addr 127.0.0.1:7070] [--open] [--css path]... [--site text]` | serve the decks, rendered from markdown on every request; live reload over server-sent events when a deck, an asset or a stylesheet changes; the URL hash keeps your slide. Use `--addr 0.0.0.0:7070` to open on a phone (prints a LAN URL + QR). On a loopback address, press `` ` `` for a live terminal drawer; `--site text` titles the index page, as in `si build`. `--remote[=tp,draw,clicker]` adds a keyed second port for a tablet or phone: the teleprompter, a draw pad and a clicker that drive the projected deck (see [docs/remote.md](docs/remote.md)) |
| `si --web [dir] [--addr 127.0.0.1:7070] [--no-open]` | the Studio: edit decks in the browser with a live preview and the `si check` report; loopback only, opens a tokenised URL (see [docs/studio.md](docs/studio.md)) |
| `si [build] <dir\|file.md> [--output html/] [--css path]... [--no-inline-media] [--pdf] [--site text]` | write one self-contained HTML file per deck plus `index.html` (a landing page with a tile per deck); `--pdf` also renders `pdfs/<name>.pdf` per deck and adds a PDF download badge to each tile (needs headless Chrome); `--site text` (alias `--title`) sets the landing page's `<title>` and `<h1>` |
| `si serve <dir\|file.md> [--addr 127.0.0.1:7070] [--open] [--css path]...` | serve everything for readers with live reload: decks, articles, guides with their navigation, labs (no terminal drawer) |
| `si check <dir\|file.md>` | validate decks: title, directives, options, missing files, include cycles, malformed `stage` blocks, broken canvas SVGs |
| `si new <name>` | scaffold `<name>/<name>.md` with assets, a module and a theme; `--type article` (or deck, guide, book, site, page, lab, demo, mindmap) writes a small starter instead, added to the menu of the guide or site it lands in |
| `si fix <dir> [--write]` | turn raw-HTML components into si syntax (`==highlight==`, badge, logo, cards); a dry run unless `--write` |
| `si new-canvas <name>` | scaffold `<name>.svg` (a working animated canvas) + a deck to preview it |
| `si migrate <dir\|file> [--ai "<cmd>"]` | print an editorial-migration prompt for an existing deck; `--ai "claude -p"` pipes it to an agent |
| `si manual [--slides] [--pdf] [--out dir]` | render the built-in user manual as self-contained HTML (and PDF with `--pdf`) |
| `si selfupdate` | download the latest release for your OS/arch (checksum-verified) and replace the running `si` |
| `si rules` | print RULES.md |
| `si --theme <name\|file.css\|url>` | override the theme: a built-in (`si theme list`: `studio`, `paper`, `mono`, `crdb`, `acemq`, `acemq-pro`; `"AceMQ Pro"` works too), a CSS file, or an `https://` URL |
| `si theme list` | list the built-in themes: name, display name, aliases, summary |
| `si --export css-rules\|rules\|conf\|example\|themes\|agent-rules [--output path]` | starter stylesheet · RULES.md · starter `si.conf` · the showcase decks · a sample deck per theme (a gallery) · `CLAUDE.md` + `AGENTS.md` for a deck project, so a coding agent writes decks the si way (teleprompter `Notes:` as spoken script, `Points:` as talking points; never overwrites, `-o -` prints) |

Flags are GNU style and go before or after the path:
`si --css css/ --output html/ md/` and `si md/ -o html/` both work.
Exit codes: 0 ok, 1 deck problems or runtime error, 2 usage.

## Screenshots

| | |
|---|---|
| ![Title slide](docs/screenshots/title.png) | ![Canvas, later state](docs/screenshots/canvas-delivered.png) |
| ![Two columns at 1440×900](docs/screenshots/two-col.png) | ![Teleprompter](docs/screenshots/teleprompter.png) |
| ![Video background](docs/screenshots/video-background.png) | ![Harbor theme from --css](docs/screenshots/harbor-theme.png) |

Phone, 400×800: ![Two columns stacked on a phone](docs/screenshots/two-col-phone.png)
![Title, dark](docs/screenshots/title-phone-dark.png)

The `si play` terminal drawer (loopback only), two tabs and a live shell:
![The play terminal drawer](docs/screenshots/terminal-drawer.png)

More in [docs/screenshots](docs/screenshots/).

## Styling

Pick a built-in theme in front matter (`theme: crdb`), point `theme:` at a CSS
file or an `https://` URL, or layer your own CSS:

```bash
si --export css-rules --output my-theme.css   # edit tokens and hooks
si play --css my-theme.css decks/
si --export themes && si play si-themes        # browse every built-in theme
```

Your `--css` loads in the `si.user` cascade layer after the built-in theme, so
it wins without specificity games; the fluid layout mechanics stay built in.
`examples/css/harbor.css` is a complete brand theme (usable as `theme: harbor.css`).

## Documentation

- **[User manual](docs/reference/user-manual.md)** — rules, syntax, examples
  (also `si manual --slides` / `si manual --pdf`)
- **[RULES.md](docs/RULES.md)** — the complete format reference (also `si rules`)
- [docs/stage-diagrams.md](docs/stage-diagrams.md) — the `stage` flow DSL
- [docs/authoring-canvases.md](docs/authoring-canvases.md) — hand-authored animated canvases
- [docs/timer-animations.md](docs/timer-animations.md) — the break-timer animation layer
- [docs/asciidoc.md](docs/asciidoc.md) — authoring decks in AsciiDoc (`.adoc`)
- **[Developer reference](docs/reference/developer-reference.md)** — architecture and how to extend si

## Development

```bash
make test           # go vet + go test ./...
make build          # bin/si
make play-example   # si play examples --open
```

See [CLAUDE.md](CLAUDE.md) and the
[developer reference](docs/reference/developer-reference.md) for the code layout
and how to add a directive, theme, canvas helper or timer animation.

## License

Licensed under the Apache License, Version 2.0 — see [LICENSE](LICENSE).
© 2026 Felipe Gutierrez.
