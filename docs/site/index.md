---
title: Write Markdown or AsciiDoc, present anything
description: si turns plain Markdown or AsciiDoc into fluid slide decks, articles, tutorials, books, mind maps and code shows: one self-contained HTML file each, no build server, no lock-in.
---

You write a Markdown or AsciiDoc file. si gives you back a deck that fills any
screen, a teleprompter that scrolls with you, an article with a table of
contents, a tutorial with steps you can tick, a book with an index, a mind map
you can fly into, or a code window whose lines glide from one step to the
next. One file in, one HTML file out, and it still opens from a USB stick in
ten years.

## What you can make

<!-- cards: 3 -->

### Decks · talks · hi

Talks that fill any screen, with build steps, animated canvases and a
teleprompter. [See decks](features/decks.md)

### Articles · posts · teal

One readable page with a table of contents, code, callouts and mind maps in
the text. [See articles](features/articles.md)

### Tutorials and guides · courses · ok

A folder of pages with a sidebar, numbered steps and a runbook, in Markdown or
AsciiDoc. [See guides](features/guides.md)

### Mind maps · 2D and 3D · orange

A nested list you can fold, search and fly into, as boxes, bricks or toys.
[See mind maps](features/mindmaps.md)

### Books · references · teal

Chapters, numbered figures and tables, an index and a glossary, printed as one
volume. [See books](features/books.md)

### Code windows and shows · code · hi

Code in a macOS, Windows or terminal window, with callouts and lines that
glide between steps. [See code](features/code.md)

## A deck, live

<!-- embed: live/showcase.md 1-3 -->

## A mind map, live

```mindmap
# The Thursday Promise

- Checkout
  - Cart & address
  - Payment
- Warehouse
  - Picking
  - Packing
- Delivery
  - Route 7
  - Doorstep photo
```

## A code show, live

```go id=send window=mac title="send.go" step="One attempt"
func send(order Order) error {
    return client.Post(url, order)
}
```

```go id=send step="Retry"
func send(order Order) error {
    var last error
    for try := 1; try <= 5; try++ {
        if last = client.Post(url, order); last == nil {
            return nil
        }
    }
    return last
}
```

## Around them

<!-- cards: 4 -->

### Studio · web and desktop

Write, preview, rehearse and present in the browser or the Mac app, with a
map editor and a code window builder.

### Terminal · si play --terminal

The same deck in the terminal, with real pictures, animated canvases and a
teleprompter in a second pane.

### Presenting · from your phone

Your phone as the clicker, an iPad as the teleprompter, over HTTPS after a
one-time setup. [Present from your phone](features/remote.md)

### Exports · PDF PNG SVG GIF MP4

Every deck, map and code show as a file you can send, print or post.

### Publish · LinkedIn, dev.to, Hashnode

A deck becomes a carousel post, an article a post or a full article, always a
dry run first.

### Themes and fonts · your look

Nine built-in themes in light and dark, your own CSS, fonts from Google,
Bunny, Fontshare or a file. [See the themes](themes.md)

### Sites · many pages

A folder becomes a site with a navigator, cards, embeds and search, decks and
articles side by side.

### QR codes · si qr

A code on a slide, or as PNG, SVG or PDF with your logo in the middle.
[See QR codes](features/qr.md)

### Checks · si check

Broken links, missing files, click cues that do not match, and pages over
their size budget.

## Do you have an AI agent?

Ask it for the talk. With the si skill it writes the Markdown, runs
`si check`, fixes what it finds and looks at the result. In Claude Code:
`/plugin marketplace add felipeg48/simple-ideas`, then
`/plugin install si@simple-ideas`. [Codex, Cursor and the others](agents.md)

## Where to go next

- [The features](features/decks.md): each kind of document with its example running and the Markdown behind it.
- [The rules](rules/): every directive, front matter key and fence option, each with an example.
- [The manual](manual.md): the same, as a deck you can present.
- [Download](download.md): the `si` binary and the Mac app.
