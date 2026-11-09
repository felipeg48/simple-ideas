# 18. Lab documents

`type: lab` in the front matter turns the file into one readable guide page,
for hands-on material that people follow at their own pace: a header with the
title and byline, a table of contents beside the text, the guide, and a footer.
There are no slides, steps, canvases or teleprompter.

```markdown
---
type: lab
title: Trace an Order Through the Hand-offs
---

# Trace an Order Through the Hand-offs

> [!NOTE]
> Everything here is read-only until the last step.

## Step 1: read the timeline
```

- The `# H1` is the page title: the first `# H1` whose text equals the title
  (compared as plain text, so `**bold**` or `` `code` `` in either still
  matches) is dropped from the body; any other H1 stays. `##` and `###`
  headings get stable GitHub-style ids and anchor links, and fill the table of
  contents. The contents sit beside the text on wide screens and collapse
  above it under about 900 px.
- A `---` line inside a lab is a horizontal rule, never a slide break.
- **Callouts** use GitHub alert syntax: `> [!NOTE]`, `> [!TIP]`,
  `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`. These five are the only
  kinds (there is no separate checkpoint kind; use `[!IMPORTANT]` or
  `[!TIP]`). Each becomes a callout with an icon and a title. Callouts are
  recognised at the top level only (not inside lists or other quotes); any
  other blockquote stays a quote.
- Code blocks keep the highlighter, a language label and a copy button. The
  prose keeps a readable measure (78ch); on a wide screen a code block or a
  markdown table that needs more room grows to the whole text column, and
  only beyond that scrolls sideways inside its own box; the page never does.
  Raw HTML tables are not wrapped. A table whose header row is all empty
  (`| | |`, as pandoc metadata tables have) shows no header and is a
  key/value table: it stays at the prose measure and its value cells wrap
  (long words and `code` break) so it fits a phone without scrolling.

  ```markdown
  | | |
  |---|---|
  | **Time** | 30 minutes |
  | **Folder** | `labs/files/lab01` |
  ```
- Images become figures, links to other `.md` files become `.html`, and the
  theme, `mode:`, `accent:`, `--theme` and `--css` apply as they do for a deck
  (the `d` key toggles dark mode). The print stylesheet prints the whole guide.
- An include (`<!-- include: parts/setup.md -->`) pastes the file in place
  without a slide break. Keep included parts in a subfolder: `si build` builds
  every top-level `.md` of a folder (except `README.md`, which is never built)
  as its own deck or lab. A split lab keeps paths relative to each module, as
  a split deck does: `sections/01-setup.md` writes `![x](../images/x.svg)`,
  and problems point at the module's own file and line.
- Slide and body directives (`layout`, `steps`, `svg`, and so on) have no
  effect in a lab. Each one is reported by `si check` as a warning and
  otherwise ignored; an unknown directive (`<!-- columns -->`) is also only a
  warning in a lab (in a deck it is an error). A `Notes:` line is not special
  in a lab: it renders as ordinary text.
- Deck-only front matter keys (`aspect`, `transition`, `background`,
  `labs-label`, `footer-position`, `runbook`, `demos`) have no effect in a
  lab; `si check` warns about each. `watermark` and `watermark-opacity` work
  in every document type (labs, articles, guides, books, demos): a faint
  fixed layer over the page, on every printed page; guide and book pages
  inherit it from `_nav.md`. `footer`,
  `year` and `copyright` fill the page footer; `author`, `date` and `version`
  the byline. A ` · ` inside one of them splits it into separate byline items
  (`date: Harbor Labs · day 1` is two), and the dots between items are drawn
  so that a wrapped line never starts or ends with a lone separator.
- When `si build` builds a folder with more than one lab, each lab page ends
  with **Previous / Next** links to its neighbours in file-name order
  (`lab-01.md`, `lab-02.md`, … — number the files to set the order); decks in
  the same folder are skipped. A single lab, `si build file.md` and `si play`
  show no such links. Other links are ordinary links (`[Lab 2](lab-02.md)`
  becomes `lab-02.html`).
- `si build` writes one HTML file per lab and lists it in `index.html` with a
  `LAB` badge and its section count instead of slides and time. A slides PDF
  (`--pdf`) is not made for labs; print the page instead.
- In Studio, labs show a lab icon in the tree and open in the preview like
  decks.

The showcase has one: `examples/order-lab.md`.

**Search (`/`).** Every document (lab, article, guide page, book page, demo)
has a Search button and opens a search box on `/`, in `si serve` and in the
built files alike (it works from `file://`). It finds the words on this page
(a heading first, then the text, with the match marked), then the other
pages: in a guide or book every page, every section with a snippet of its
text, and a book's index terms; for a page on its own, the other pages of the
build. ↑ ↓ choose, Enter opens, Esc closes. Nothing is fetched: the other
pages' data is written into each page when it is built.

Labs share the document engine with articles (§19) and guides (§20). A
`## Step N:` (or `## N.`) heading in a lab or a guide page shows N in a
numbered badge; the contents list keeps the full text.
