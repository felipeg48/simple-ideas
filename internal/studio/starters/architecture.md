---
type: architecture
title: {{title}}
subtitle: What this system is, in one line
---

## The shape of it

One service owns the data; everything else reads through it.

- `invariant` **One writer** — only the orders service writes the orders table.
- `invariant` **Events, not calls** — other services learn about an order from the stream.
- `boundary` **Public edge** — the gateway is the only thing the internet can reach.

> Honest read: the stream is a single broker today; it gets a replica before launch.

## How a request flows

A checkout, from the browser to the warehouse.

```flow
dir: LR
B[Browser] --> G[Gateway] --> O[Orders] --> W[Warehouse]
dot B..W 2s
```

## Phases

1. One region, one broker. (now)
2. A broker replica and a second region.
3. Regions take writes for their own customers.
