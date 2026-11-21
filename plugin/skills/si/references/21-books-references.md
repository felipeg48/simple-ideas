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
  and links it from the cover and the bar: A4 pages, the cover and a title
  page (`_cover.md` and the `copyright:` line) unnumbered, the contents with
  the page of every chapter and section (si prints twice: once to learn the
  pages, once to write them in), a page of its own for each part, every
  chapter on a new page with its title at the top of each page and the page
  number at the bottom, figures, tables and code kept whole, a PDF outline.
  Code callouts, charts and scenes print in their final state. Needs Chrome.

  ```bash
  si build book/ -o html/ --pdf        # html/book.pdf
  ```
- **Export for a publisher.** The same book, read once, as Markdown, Word,
  RTF or EPUB (any of them, together; written beside the HTML). Chapters,
  appendices, sections, figures and tables keep their numbers, empty
  cross-references are filled ("Table 1.1"), `((index terms))` become the
  plain word, `<!-- … -->` directives and raw HTML are dropped, code callouts
  become `(1)` markers and a numbered list after the block, alerts (and
  AsciiDoc admonitions) a quote with a bold **Note:** label. Works the same
  from `.adoc` chapters.

  ```bash
  si build book/ -o out/ --md      # out/md/book.md (the whole book, a pandoc title block first),
                                   # out/md/<chapter>.md one per chapter, out/md/images/
  si build book/ -o out/ --docx    # out/book.docx: Word styles (Title, Heading 1-3, Code, Caption,
                                   # List Bullet/Number, Quote, Table Grid), a contents field Word
                                   # fills on open, each chapter on a new page, page numbers
  si build book/ -o out/ --rtf     # out/book.rtf: headings, lists, tables, code, pictures
  si build book/ -o out/ --epub    # out/book.epub: EPUB 3, one file per part and chapter
  ```

  Pictures travel as files (`image::` and `![]()` alike, and the SVG of a
  `<!-- svg: -->` canvas); Word and RTF get an SVG as a PNG Chrome draws
  (without Chrome, a placeholder line). Figures si draws itself (`chart`,
  `scene`, `stage`, `mindmap`, `quiz`, `video` fences) become a placeholder
  line naming the figure; a `mermaid` fence stays a fence in Markdown. A
  guide (`_nav.md` without `type: book`) exports the same way as `guide.*`,
  without numbers, and an article as `<name>.md`, `<name>.docx` and so on
  (`si build note.md --docx`). A deck, a site or a course says the flags are
  not for it.
- `si serve` serves a book live (the whole book is numbered on every
  request); in a folder, a book is one BOOK tile on the index.

Example: `si --export example --type book` (also `--type reference`).
