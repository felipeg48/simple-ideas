# 35. Architecture pages

`type: architecture` turns one Markdown file into a design document: the
`##` sections are numbered, the first paragraph after each is its lead, and
the blocks under it take a design-doc look. It is an article (§19) with a
section grammar; everything an article holds still works.

````markdown
---
type: architecture
title: Orders — Architecture & Design
subtitle: One writer, events for everyone else
---

## The orders service owns the data

Only one service writes an order; the others learn about it from the stream.

- `invariant` **One writer** — only the orders service writes the orders table.
- `invariant` **Events, not calls** — the warehouse hears about an order from the stream.
- `boundary` **Public edge** — the gateway is the only thing the internet reaches.

> Honest read: the stream is one broker today; it gets a replica before launch.

## How a checkout flows

From the browser to the warehouse, one hop at a time.

```flow
dir: LR
B[Browser] --> G[Gateway] -->|POST /orders| O[Orders] -->|order.created| W[Warehouse]
dot B..W 2s
```

## The data

**Order** — what was bought, by whom, and its state
**Parcel** — one box of an order, with its tracking number

| Region | Orders / day | Brokers |
|--------|--------------|---------|
| EU | 40 000 | 1 |
| US | 65 000 | 1 |

## Phases

1. One region, one broker. (now)
2. A broker replica and a second region.
3. Each region takes the writes of its own customers.
````

| You write | You get |
|---|---|
| `## Section` | a numbered section: a badge (1, 2, 3 …) and the title. `number: off` in the front matter drops the numbers. |
| the first paragraph after a `##` | the section's lead: a quieter introductory line (unless it is a definition list, below) |
| a list whose **every** item reads `` - `tag` **Title** — body `` | a grid of cards: the tag chip, the title, the body. The tag is optional (`- **Title** — body`); `—`, `–`, `--`, `-` or `:` separate the title from the body. A list where only some items read that way stays a list, and `si check` says so. |
| a table | a bordered table on a card |
| a quote (`> …`) | a note: a tinted box for caveats ("Honest read: …") |
| a ` ```flow ` fence | the stage fence (§13) by another name, on a card |
| a paragraph of `**Name** — what` lines | a definition list, two columns on a wide page |
| the ordered list of a section titled `Phases` | a row of phase cards (Phase 1, Phase 2 …); the item ending in `(now)` is highlighted and the mark itself is not shown |

The page has no contents column: the numbered sections are the outline.
`subtitle:` is the line under the title (`description:` wins when both are
set). Light and dark, the theme toggle, `--css`, `si build --pdf` (one
printed page per sheet, cards kept whole) and `--md` / `--docx` / `--rtf` /
`--epub` (§21) all work as for an article.
