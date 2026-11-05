# si documents: articles, guides, labs and demos: design

Status: draft for review (2026-11-05). No code yet.

## Intent

si already writes slides (decks) and one kind of long-form page: the lab
document (`type: lab`, RULES.md §18). Teams also need **articles** (one long
read), **tutorials/guides** (many pages read in order, with navigation) and
**demo scripts** (instructor-only run sheets; today the CACI demos are built by
pandoc). This spec turns the lab engine into one **document family** with four
types, adds the multi-page guide, and adds the code features technical writing
needs: code from files and from git (with line ranges and tags), diffs, tabs,
callouts and line highlights. AsciiDoc input gets the same features.

The rules here become RULES.md sections and the agent guide (`si --export
agent-rules`) when each phase ships.

## The rules that do not change

- **Self-contained output** (CLAUDE.md rule 2): one HTML file per page, CSS,
  JS, images and every piece of fetched code inlined. Nothing is fetched when a
  reader opens a page. Git code and diffs are fetched **at build time** and
  cached, like `theme: https://…` already is.
- **Fluid** (rule 1): pages reflow from 400 px to 4K; tabs, callouts and diffs
  included. Print shows everything (all tabs expanded, no scroll boxes).
- **Documented and used** (rule 3): every rule below gets a RULES.md section
  with an example and is used in a real example under `examples/` (a short
  guide about the order-journey, an article, a lab, a demo).
- **Contract** (rule 5): new `--si-*` tokens and `.si-*` hooks go into
  `internal/assets/contract.go`.
- **Stdlib + goldmark** (rule 8): git access over `net/http`; local clones via
  the `git` CLI when the user has it (optional, `exec`), no git library.
- **`si check` catches broken references before readers do**: links, anchors,
  includes, ranges, tags, git refs.

## 1. The document family

| `type:` | Is | Navigation | Badge | Audience |
|---|---|---|---|---|
| `article` | one long read | "On this page" contents | none | readers |
| `guide` | one page of a multi-page tutorial | the guide sidebar, Prev / Next, breadcrumb | none | readers |
| `lab` | a hands-on guide (today's type, unchanged) | contents; Prev / Next between labs | LAB | participants |
| `demo` | an instructor run sheet | contents; the demo picker in `si play` | INSTRUCTOR ONLY | the presenter |

All four use the same engine (`render.Document`, `doc.css`, `doc.js`): title
and byline header, readable measure, GitHub callouts, highlighted code with copy,
key/value tables, figures, includes, theme / mode / accent / `--css`, print
stylesheet. Deck-only directives and keys stay warnings (as for labs today).

### Shared front matter

| Key | All types | Notes |
|---|---|---|
| `type` | required | `article`, `guide`, `lab`, `demo` |
| `title` | required | page title; the matching `# H1` is dropped from the body |
| `author`, `date`, `version` | optional | byline; ` · ` splits items |
| `description` | optional | one-sentence summary: shown under the title of an article, in the guide index and as `<meta name="description">` |
| `footer`, `year`, `copyright` | optional | page footer |
| `theme`, `mode`, `accent` | optional | as for decks |
| `header` / `footer-page` | optional | `off` drops the guide's shared header / footer on this page (§2) |

## 2. Guides: a folder with a `_nav.md`

A folder that contains `_nav.md` is **one guide**. `_nav.md` is a markdown list
of links, nested for sections; its front matter holds the guide-wide settings,
inherited by every page (a page's own front matter wins).

```markdown
---
title: RabbitMQ for operators
theme: acemq
version: 4.3
edit-url: https://github.com/acme/rmq-guide/edit/main/
repo: https://github.com/acme/rmq-app      # default repo for `code:` (§4)
---

- [Introduction](index.md)
- Getting started
  - [Install](install.md)
  - [A first broker](first-broker.md)
- [Operations](ops/index.md)
  - [Rolling upgrades](ops/upgrades.md)
```

Rules:

- **Order** is the list order. Prev / Next follow it (not file names); the
  sidebar shows it, with the current page marked and its section open. A list
  item without a link is a section heading.
- **Pages** default to `type: guide`; a page may say `type: lab` (a lab inside
  a guide keeps its badge). Subfolders are allowed in a guide (unlike decks,
  where `si build` ignores them).
- **Shared parts**: `_header.md` and `_footer.md` in the guide folder are
  rendered on every page (above the title and above the page footer). A page
  turns one off with `header: off` / `footer-page: off`.
- **Breadcrumb**: guide title › section › page.
- **Links** between pages are written to the `.md` file
  (`[install](install.md#ports)`) and become `.html`; `si check` reports a link
  to a missing page or a missing `#anchor`, and a page in the folder that
  `_nav.md` does not list.
- **Edit link**: with `edit-url:`, each page ends with "Edit this page"
  (`edit-url` + the page path).
- **Build**: `si build guide/ -o out/` writes one self-contained HTML file per
  page, mirroring the folder (`ops/upgrades.md` → `out/ops/upgrades.html`);
  `index.md` → `index.html`. A guide folder inside a `si build` source folder
  becomes one tile in the index.
- **Search**: the `/` find palette searches the current page and lists the
  other pages of the guide (the page list of v0.9.3), in nav order. Full-text
  search across pages is phase 4.

## 3. Articles

- One page, `type: article`. Required: `title`; recommended: `author`, `date`,
  `description`.
- Header: title, `description` as the lede, byline, **reading time** (words /
  230, rounded up, "6 min read").
- "On this page" contents from `##` / `###` when the article has three or more
  `##` headings, else none.
- Optional `cover:` image (a figure under the header, like any image: `%` /
  `em` sizes, inlined).
- An article may live alone or inside a guide (then it gets the guide's nav).

## 4. Code features (all document types; decks where noted)

### 4.1 Code from a local file

```markdown
<!-- include: src/App.java -->                     the whole file as a code block
<!-- include: src/App.java lines=5-10 -->          lines 5 to 10 (also 5-, -10, 5-10,20-24)
<!-- include: src/App.java tag=connect -->         between `tag::connect[]` and `end::connect[]`
<!-- include: src/App.java tag=connect lang=java title="App.java" hl=3 -->
```

- An include of a `.md` / `.adoc` file pastes the text (today's behaviour); any
  other file becomes a **code block**, highlighted by extension (`lang=`
  overrides), titled with the file name (`title=` overrides).
- **Prefer `tag=` over `lines=`**: tags move with the code, line numbers do
  not. Tag markers are AsciiDoc's, inside any comment syntax
  (`// tag::connect[]`, `# tag::connect[]`, `<!-- tag::connect[] -->`); marker
  lines are never shown. Several `tag=` values join in file order.
- Indentation common to all shown lines is removed (`indent=keep` keeps it).
- `si check`: missing file, range past the end, tag not found or not closed.
- Decks: works too (a code block on a slide).

### 4.2 Code from git

Paste a **permalink** (GitHub "Copy permalink"; GitLab and Bitbucket permalinks
too):

```markdown
<!-- code: https://github.com/acme/rmq-app/blob/3f2a9c1/src/App.java#L5-L10 -->
<!-- code: https://github.com/acme/rmq-app/blob/3f2a9c1/src/App.java tag=connect -->
<!-- code: path=src/App.java ref=3f2a9c1 lines=5-10 -->   uses `repo:` from front matter or _nav.md
<!-- code: repo=../rmq-app path=src/App.java ref=3f2a9c1 tag=connect -->   a local clone (git show)
```

- Fetched at build time (raw file URL), cached under the temp dir keyed by URL
  + ref; a commit SHA ref is immutable, so its cache never expires. The page
  inlines the code; a reader never fetches it.
- **Pin to a commit.** A ref that is a branch or tag (`main`, `v1.2`) builds,
  but `si check` warns "moving ref": with `lines=` the shown code can drift.
- The code block gets a source link (the permalink) in its title bar.
- Private repos: `GITHUB_TOKEN` / `GITLAB_TOKEN` env vars, or a local clone via
  `repo=<path>` (`git -C <path> show <ref>:<path>`).
- `si build --offline` / `si check --offline` use the cache only and fail on a
  miss. Network errors fall back to a cached copy (as URL themes do).
- Options from 4.1 (`lines`, `tag`, `lang`, `title`, `hl`, `indent`) apply.

### 4.3 Diffs

```markdown
<!-- diff: https://github.com/acme/rmq-app/commit/3f2a9c1 path=src/App.java -->
<!-- diff: https://github.com/acme/rmq-app/compare/v1.1...v1.2 path=src/ -->
<!-- diff: repo=../rmq-app ref=v1.1..v1.2 path=src/App.java -->
```

- Fetched at build time (`.diff` of the commit / compare), filtered to `path=`
  (a file or a folder prefix; omitted = the whole diff).
- Shown as a diff block: file header, hunk headers, added / removed / context
  lines coloured with `--si-ok` / `--si-crit` tints, old and new line numbers.
  `context=3` trims context lines. Side-by-side is phase 4.
- Copy copies the patch.

### 4.4 Tabs

````markdown
<!-- tabs -->
```java title="Java"
channel.basicQos(10);
```
```python title="Python"
channel.basic_qos(prefetch_count=10)
```
````

- `<!-- tabs -->` groups the code blocks that follow it, up to the first
  non-code block (an `include:` / `code:` counts as a code block). The tab label
  is `title=`, else the language name.
- Each tab keeps its own highlighter, copy button and callouts.
- **Synced**: choosing "Java" switches every group on the page that has a "Java"
  tab, and is remembered per site (localStorage) for the next page.
- Keyboard: tabs are a `tablist`; ← / → move, as ARIA tabs do.
- Print and PDF: every tab is shown, one after another, each with its label.
- Decks: tabs work on a slide too (only the chosen tab shows; no sync).

### 4.5 Callouts and highlighted lines

````markdown
```java hl="3,5-6" title="Consumer.java"
Connection c = factory.newConnection();   // <1>
Channel ch = c.createChannel();
ch.basicQos(10);                          // <2>
```
<1> One long-lived connection per process.
<2> Prefetch caps the unacked messages per consumer.
````

- A `<N>` at the end of a code line (bare, or as the whole content of a trailing
  comment: `// <1>`, `# <1>`, `-- <1>`, `<!-- 1 -->`) becomes a numbered marker;
  the marker and its comment are removed from the code and from Copy.
- The `<N> text` lines right after the block become the callout list; marker
  and list item link to each other (hover / focus highlights both).
- `si check`: a marker without a list item, or a list item without a marker.
- `hl=` highlights lines (1-based, of the lines shown, so it works on included,
  ranged and fetched code); `lines` shows line numbers, `lines=5` numbers from 5
  (default for `lines=` ranges: the file's own numbers).
- Fence attributes are the info string after the language:
  `` ```java title="…" hl="…" lines `` (unknown attributes are a check warning).

## 5. Demos: `type: demo`

A demo is the presenter's run sheet for a live part of a session. It is a
document (built like a lab) **and** the content of the ▶ Demos panel in the
`si play` terminal (RULES.md §14, `demos:` in a deck). The structure, taken from
the CACI demos:

````markdown
---
type: demo
title: "Session 1 demo: a first broker"
session: 1 · Understanding CACI message flows
slides: 23, 24
duration: 14 min
---

# Session 1 demo: a first broker

## What this demo must accomplish
1. **Routing is decided by bindings, not by the publisher.** …

## Before the session (T-30 min)
```bash
docker compose up -d --wait
```

## The script
### Slide 23, "Live: a first broker" (8 min)
> Say: "One RabbitMQ 4.3.7 node, the same software CACI runs."
```bash
rmq list queues
```
```text
orders.billing  quorum
```

## If it goes wrong
## Reset between rehearsals
````

Rules:

- Badge **INSTRUCTOR ONLY** in the header; the front matter fields `session`,
  `slides`, `duration` fill a key/value block under the title.
- **`> Say:` lines are what the presenter says to the room, verbatim** (the
  teleprompter rule, CLAUDE.md rule 5): spoken sentences, no stage directions.
  Instructions to the presenter ("point at the unacked column") are plain prose.
  `> Say:` renders as a distinct speech callout.
- Shell-type fences (`bash sh shell zsh console sql powershell`) are commands
  (Copy / Run in the panel); `text` fences are expected output. A command block
  should be followed by its expected output.
- Script steps are `### Slide N, "title" (M min)`; `si check` warns when the
  minutes of the steps do not add up to `duration`, or a step names a slide
  number the linked deck does not have (when the deck lists the demo).
- `si build` builds demos like labs (into an "Instructor" group of the index);
  `si build --no-instructor` leaves them out of a participant build. A
  pandoc build (`demos/build-html.sh` in CACI) keeps working: the ▶ Demos panel
  reads the markdown either way.

## 6. AsciiDoc mapping

AsciiDoc input (`.adoc`) keeps converting to the markdown pipeline
(`internal/deck/adoc.go`); each new feature maps onto the markdown form:

| AsciiDoc | si markdown |
|---|---|
| `:doctype: article` / `:si-type: guide` | `type:` |
| `include::src/App.java[lines=5..10]` | `<!-- include: src/App.java lines=5-10 -->` |
| `include::src/App.java[tag=connect]` | `<!-- include: src/App.java tag=connect -->` |
| `[source,java,highlight=3;5..6]` | ```` ```java hl="3,5-6" ```` |
| `.App.java` (block title) | `title="App.java"` |
| `<1>` callouts + `<1> text` list | the same (§4.5) |
| `[tabs]` + `====` block of `[source]` blocks (Asciidoctor tabs) | `<!-- tabs -->` |
| `xref:install.adoc#ports[]` | `[…](install.md#ports)` |
| `NOTE:` / `TIP:` / `[IMPORTANT]` | `> [!NOTE]` … (exists) |

A guide may mix `.md` and `.adoc` pages; `_nav.md` links either.

## 7. `si check` for documents

Errors: missing include / code file, range or tag not found, broken internal
link or anchor, malformed `_nav.md`, unreachable `code:` / `diff:` with no
cache, callout marker without a list item (and the reverse).
Warnings: moving git ref, page not in `_nav.md`, unknown fence attribute, demo
minutes ≠ `duration`, deck-only directives in a document (exists).

## 8. Agent guide additions (`si --export agent-rules`)

- Which type to use (article / guide / lab / demo) and the required parts of each.
- Guides: write `_nav.md` first; link to `.md` files; one topic per page.
- Code: include from the repo instead of pasting; pin git refs to a commit;
  prefer `tag=` over `lines=`; explain code with callouts, not prose about line
  numbers; put alternatives in tabs.
- Demos: `> Say:` is spoken verbatim (the teleprompter rule); every command
  block followed by its expected output; a reset section.
- Run `si check` (and `--offline` in CI) before finishing.

## Phases

| Phase | Ships | Acceptance |
|---|---|---|
| 1 | `type: article`, `type: guide`, `_nav.md` (order, sidebar, Prev / Next, breadcrumb, edit link), `_header.md` / `_footer.md`, subfolders, link and anchor checks, `.adoc` pages in a guide | a 6-page example guide under `examples/`, and one CACI lab set rebuilt as a guide, read at 400 px, 1440 px and in print |
| 2 | include of code files (`lines`, `tag`, `lang`, `title`, `indent`), fence attributes, `hl` / line numbers, callouts, tabs (synced, print-expanded) | the example guide shows a tagged include, a 3-language tab group with callouts |
| 3 | `code:` and `diff:` (GitHub / GitLab / Bitbucket permalinks, local clone), build-time cache, tokens, `--offline`, moving-ref warning | a guide page pinned to a public repo commit builds offline from cache |
| 4 | `type: demo` + `--no-instructor`, cross-page search, side-by-side diffs, versioned guides | the CACI demos built by si match the pandoc output's content |

Each phase is useful alone; RULES.md and the agent guide grow with it.

## Non-goals

- A full AsciiDoc implementation (only the mapped subset).
- Fetching anything at read time; live "embed from GitHub" widgets.
- A static-site generator's plugin system, themes per page type, or a blog
  (dates, feeds, tag pages).
- Executing code from documents (the Run button stays in the `si play` terminal
  for demos and runbooks only).

## Decisions (2026-11-05)

1. **Hosting: both.** `si build` writes a static site (one self-contained file
   per page, any static host). A new **`si serve <dir>`** serves *everything*
   (decks, articles, guides with their nav, labs, demos) with live reload, the
   way `si play` serves decks; `si play` stays the presenter's command (slides,
   teleprompter, terminal, `--remote`). Both render on every request; `si serve`
   is loopback by default and never has the terminal.
2. **Code hosts: GitHub only for now, public and private.** Private access with
   `GITHUB_TOKEN` (or `gh auth token` when the GitHub CLI is logged in), or a
   local clone (`repo=<path>`). GitLab, Bitbucket and S3 are later; the
   `code:` / `diff:` syntax keeps the URL form so they slot in.
3. **Guides embed slides.** A body directive in any document:

   ```markdown
   <!-- slides: ../decks/c1-message-flows.md -->            the whole deck, in a 16:9 frame
   <!-- slides: ../decks/c1-message-flows.md 23-24 -->      slides 23 to 24 only
   <!-- slides: ../decks/c1-message-flows.md#live-a-first-broker -->   one slide by its heading id
   ```

   The deck is rendered at build time and inlined (the page stays
   self-contained): arrow keys and buttons step through slides and build
   steps, canvases animate, a link opens the full deck. Print shows each
   slide as a static frame. `si check` reports a missing deck or range.
4. **CACI demos move to `type: demo`** after all phases are done (not before).

## `si --export example --type`

`si --export example` takes `--type` to pick what to scaffold; the default stays
the slide deck, so today's command is unchanged.

| `--type` | Aliases | Writes (into `--output`, default `./si-example`) |
|---|---|---|
| `deck` (default) | `slides` | the showcase deck + appendix (today's export), now with a runbook and a demo |
| `article` | | one article using the lede, reading time, a cover, code includes, tabs, callouts |
| `guide` | `tutorial`, `guides` | a small guide: `_nav.md`, `_header.md`, 4 to 6 pages in sections, an embedded slide range, tagged includes, a `code:` permalink |
| `lab` | | a lab document (today's `order-lab.md`) |
| `demo` | | a demo script in the §5 structure, plus the deck front matter that lists it |
| `all` | | every type side by side, so `si serve si-example` shows them together |

Each example is real content about the order-#4711 story (CLAUDE.md rule 4),
and every rule it uses is in RULES.md (rule 3).
