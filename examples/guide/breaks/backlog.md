---
title: A warehouse backlog
description: Flood the warehouse with orders and watch a second picker drain the queue.
---

# A warehouse backlog

On Tuesday morning a promotion put 2,400 orders on a shift planned for 900.
Order #4711 waited in that queue.

| | |
|---|---|
| **Time** | 15 minutes |
| **You need** | Order #4711 [paid](../happy/pay.md) |

## Step 1: Fill the queue

```bash
curl -X POST localhost:8083/admin/flood -d '{"orders": 2400}'
```

The tracking page for #4711 turns the promise tile amber: *Friday, late*.

## Step 2: Lend a picker

```bash
curl -X POST localhost:8083/admin/pickers -d '{"count": 2}'
```

Two pickers walking towards each other halve the walk for every order. The
queue drains in about four minutes, and the promise tile goes back to
*Thursday*.

> [!IMPORTANT]
> In the real warehouse the second picker comes from the returns desk, and
> only when the queue passes 500 orders. The lab skips the threshold.
