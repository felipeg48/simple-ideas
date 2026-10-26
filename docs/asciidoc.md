# AsciiDoc decks

`si` reads AsciiDoc as well as Markdown: name a deck `*.adoc` or `*.asciidoc`
and `si play` / `si build` / `si check` treat it like any other deck. There is
nothing to configure — the extension is detected.

Under the hood si converts the AsciiDoc to the Markdown it already understands
(a focused, dependency-free preprocessor — stdlib only, not a full Asciidoctor).
So a converted deck supports **every si feature** (layouts, steps, canvases,
`stage` fences, themes, notes) via si's own directives; the AsciiDoc layer just
covers the prose and structure. For rich Asciidoctor features outside the subset
below, author those parts with si directives or raw HTML.

## The document header → front matter

An AsciiDoc document title and its attributes become si front matter + the title
slide:

```asciidoc
= My Talk
:author: Ada Lovelace
:theme: crdb
:aspect: 16:9

== First section
```

becomes a `title`/`author`/`theme`/`aspect` front-matter block and a `# My Talk`
title slide. Mapped attributes: `author`, `revdate`/`date`, `theme`, `mode`,
`accent`, `transition`, `aspect`, `mermaid`, `lang`, `background`. (You can also
just write an si `---` front-matter block at the top of a `.adoc` file; it passes
through untouched.)

## Supported subset

| AsciiDoc | Becomes | Notes |
|---|---|---|
| `= Title` | title slide (`# Title`) | the document title (header) |
| `== Section` | a new slide (`## Section`) | each `==` starts a slide |
| `=== Sub` / `==== ` | `### ` / `#### ` headings | within the current slide |
| `* item`, `** item` | `- item` (nested) | unordered lists |
| `. item` | `1. item` | ordered lists |
| `Term:: definition` | `**Term** — definition` | description lists (inline form) |
| `` `mono` `` | `` `mono` `` | inline code (unchanged) |
| `*bold*` | `**bold**` | inline bold |
| `_italic_` | `_italic_` | italic (Markdown already understands it) |
| `[source,go]` + `----` block | ```` ```go ```` fenced code | highlighted; `....` works too |
| `<1>` marker + `<1> text` list | marker kept; list → numbered list | code callouts |
| `[cols=…]` + `\|===` … `\|===` | a GFM table | see tables, below |
| `[quote]` + `____` … `____` | a `>` blockquote | quote blocks |
| `include::file.adoc[]` | `<!-- include: file.adoc -->` | included AsciiDoc is converted too |
| `url[label]`, `link:url[label]` | `[label](url)` | links |
| `image::path[alt]`, `image:path[alt]` | `![alt](path)` | block and inline images |
| `NOTE:`/`TIP:`/`WARNING:`/… | `> **Note:** …` blockquote | admonition lines |
| `'''` | a horizontal rule | not an si slide break (that is `---`) |
| `//` line, `////` block | dropped | comments |

Speaker notes use si's own convention inside the `.adoc` — a `Notes: (N min)`
line followed by the note text — and si directives (`<!-- layout: … -->`,
`<!-- svg: … -->`, `<!-- steps -->`, ` ```stage `) work verbatim.

### Tables

```asciidoc
[cols="1,1"]
|===
| Name | Role

| Ada | Engineer
| Bob | Designer
|===
```

becomes a GFM table (first row is the header). The column count comes from
`[cols=…]` when present (`"1,1,2"` → 3, `3*` → 3, `4` → 4); otherwise it is the
number of cells in the header row. Each cell starts with `|`; put rows one per
line (cells may share a line, as above). Cell spans, nested blocks and
`a|`/`m|` cell styles are not supported — for a complex table, write raw HTML.

### Includes

`include::sections/intro.adoc[]` becomes si's `<!-- include: sections/intro.adoc -->`
directive, so paths are relative to the including file and cycles are refused.
An included `.adoc`/`.asciidoc` file is converted before it is spliced, so you
can mix `.md` and `.adoc` modules freely. (The target's own document header is
ignored — the including deck owns the front matter.)

### Code callouts

```asciidoc
[source,go]
----
db.Exec(stmt) // <1>
----
<1> runs the statement
```

The `<1>` markers stay verbatim in the code block (they render as `<1>`), and
the `<1> …` list below becomes a numbered list. Auto-numbered `<.>` markers are
not expanded — number them explicitly.

## Not converted (use si directives or raw HTML)

Block attributes/roles (`[.lead]`), attribute substitutions (`{name}`),
multi-line description-list definitions, advanced table cell styles, and other
Asciidoctor features outside the subset above are passed through unchanged (they
will not render as intended). For those, write the slide with si's directives or
raw HTML. Use `si check` to catch anything that didn't convert cleanly.
