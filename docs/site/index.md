---
title: Write Markdown, present anything
description: si turns plain Markdown into fluid slide decks, articles, tutorials, sites, mind maps and code shows: one self-contained HTML file each, no build server, no lock-in.
---

You write a Markdown file. si gives you back a deck that fills any screen, a
teleprompter that scrolls with you, an article with a table of contents, a
mind map you can fly into, or a code window whose lines glide from one step
to the next. One file in, one HTML file out, and it still opens from a USB
stick in ten years.

## A deck, live

<!-- embed: _examples/showcase.md 1-3 -->

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

## Where to go next

- [The rules](rules/): every directive, front matter key and fence option, each with an example.
- [The manual](manual.md): the same, as a deck you can present.
- [Examples](examples.md): the files behind the pages above, to copy from.
