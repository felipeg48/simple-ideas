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
| `` `mono` `` | `` `mono` `` | inline code (unchanged) |
| `*bold*` | `**bold**` | inline bold |
| `_italic_` | `_italic_` | italic (Markdown already understands it) |
| `[source,go]` + `----` block | ```` ```go ```` fenced code | highlighted; `....` works too |
| `url[label]`, `link:url[label]` | `[label](url)` | links |
| `image::path[alt]`, `image:path[alt]` | `![alt](path)` | block and inline images |
| `NOTE:`/`TIP:`/`WARNING:`/… | `> **Note:** …` blockquote | admonition lines |
| `'''` | a horizontal rule | not an si slide break (that is `---`) |
| `//` line, `////` block | dropped | comments |

Speaker notes use si's own convention inside the `.adoc` — a `Notes: (N min)`
line followed by the note text — and si directives (`<!-- layout: … -->`,
`<!-- svg: … -->`, `<!-- steps -->`, ` ```stage `) work verbatim.

## Not converted (use si directives or raw HTML)

Tables, block attributes/roles (`[.lead]`, `[cols=…]`), AsciiDoc includes
(`include::`), substitutions/attributes (`{name}`), callouts, description lists,
and other advanced Asciidoctor syntax are passed through unchanged (they will not
render as intended). For those, write the slide with si's directives or raw HTML.
Use `si check` to catch anything that didn't convert cleanly.
