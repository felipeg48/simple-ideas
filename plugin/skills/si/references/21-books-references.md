# 21. Books (references)

A book is a guide (§20) whose `_nav.md` says **`type: book`**: a whole
reference with a cover, a numbered contents page, parts, chapters,
appendices, numbered sections, figures and tables, cross-references and an
index, as one page per chapter **and** as one file (and PDF).

```markdown
---
type: book
title: The Northwind Operations Reference
subtitle: How an order moves through the stations, and how each break is repaired.
author: Ada Lindqvist · Northwind Logistics operations
edition: 2nd edition
date: November 2026
theme: paper
---

- [Preface](preface.md)
- Part I · Foundations
  - [The order journey](01-journey.md)
  - [Promises and clocks](02-promises.md)
- Part II · When it breaks
  - [Payments that time out](03-payments.md)
- Appendices
  - [Glossary](a-glossary.md)
```

- **Numbering.** Pages listed before the first section are front matter
  (preface, foreword): not numbered. A section of `_nav.md` is a **part**.
  Every page after the first section is a chapter, numbered 1, 2, 3 in nav
  order; pages under a section called "Appendices" (or "Appendix …") are
  lettered A, B. A chapter's `##` headings are numbered 1.1, 1.2 (in the text,
  the sidebar and "On this page"); `## Step N:` headings get no badge in a book.
- **Figures and tables.** An image with a title is a captioned figure:
  `![Stations](images/stations.svg "The five stations")` becomes "Figure 1.1 ·
  The five stations". A paragraph `Table: Who owns each station` right after a
  table is its caption: "Table 1.1 · Who owns each station". (Outside a book
  both captions work too, without numbers.) Their ids are `fig-` / `tbl-` plus
  the caption in GitHub style: `#fig-the-five-stations`, `#tbl-who-owns-each-station`.
- **Cross-references.** A link with empty text is filled with the target's
  label: `[](#tbl-who-owns-each-station)` → "Table 1.1",
  `[](02-promises.md#the-promise-tile)` → "Section 2.1", `[](01-journey.md)` →
  "Chapter 1". `si check` reports a cross-reference to something that has no
  number.
- **Index.** `((idempotency key))` shows the term (lightly marked) and indexes
  it; `(((payment, retry)))` indexes without showing anything; a comma makes a
  sub-entry (`((promise, computed at checkout))` shows "computed at
  checkout", filed under "promise"). This is AsciiDoc's own syntax. The index
  page lists every term A–Z with the section numbers where it appears (or the
  chapter, or the page title for front matter), each a link to the spot.
- **Generated pages.** `index.html` is the **cover** (title, subtitle,
  edition, author, date, links to the contents, the first chapter, the PDF and
  the one-file book, and `_cover.md` as the colophon), `contents.html` lists
  front matter, parts, numbered chapters with their sections and reading
  times, appendices and the index, `book-index.html` is the index, and
  `book.html` is the **whole book in one file** (cover, contents, every
  chapter, index; links between chapters become in-page links). Every page has
  a bar with Cover · Contents · Index (· PDF). A book has no `index.md`: name the
  first page `preface.md` or similar.
- **PDF.** `si build <book> --pdf` also prints `book.html` to `book.pdf`
  (each chapter starts a new page) and links it from the cover and the bar.
- `si serve` serves a book live (the whole book is numbered on every
  request); in a folder, a book is one BOOK tile on the index.

Example: `si --export example --type book` (also `--type reference`).
