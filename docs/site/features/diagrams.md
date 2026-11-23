---
title: Delivery diagrams
description: Kanban boards, gantt plans and sequence diagrams written in plain lines, and animated. Cards glide between columns, bars fill as the today line moves, messages draw themselves. In decks, in documents, or as pages exported to PNG, JPG, PDF, SVG, GIF and MP4.
---

# Delivery diagrams

The diagrams a software team draws every week, in a few plain lines, and
**moving**. Scroll down: each one plays as it comes into view, and ← → step
through it.

## A board, day by day

```kanban title="Checkout squad"
columns: Backlog, Doing, Review, Done
wip: Doing 2, Review 2
LOGIN-12 Password reset · Backlog · @ada · bug
CART-7 Saved carts · Doing · @ravi
PAY-3 Apple Pay · Review · @lin
SHIP-9 Parcel lockers · Backlog · @mo

day 2
LOGIN-12 → Doing
PAY-3 → Done

day 3
SHIP-9 → Doing
CART-7 blocked: waiting for the mail team

day 4
CART-7 unblocked
CART-7 → Review
+ SHIP-10 Locker map · Backlog · @mo
```

```markdown
columns: Backlog, Doing, Review, Done
wip: Doing 2, Review 2
LOGIN-12 Password reset · Backlog · @ada · bug
CART-7 Saved carts · Doing · @ravi

day 2
LOGIN-12 → Doing

day 3
CART-7 blocked: waiting for the mail team
+ SHIP-10 Locker map · Backlog · @mo
```

Cards glide to their new column and are outlined as they arrive; a blocked
card turns red with its reason; a column over its WIP limit turns amber, and
the caption says when.

## A plan, with today moving

```gantt start=2026-12-01 title="Parcel lockers"
section Build
Design · 5d
Locker API · 8d · after Design
Web map · 6d · after Design
section Ship
Tests · 4d · after Locker API, Web map
Launch · milestone · after Tests

today 2026-12-04 · Design 80%
today 2026-12-10 · Design done, Locker API 60%, Web map done
today 2026-12-16 · Locker API 90%, Tests 20%
```

```markdown
Design · 5d
Locker API · 8d · after Design
Tests · 4d · after Locker API, Web map
Launch · milestone · after Tests

today 2026-12-10 · Design done, Locker API 60%
```

si schedules the tasks, outlines the critical path and says when the plan
ends. Each `today` line moves the today line and fills the bars; a task past
its end and not done turns amber.

## A request, message by message

```sequence
Browser → Shop: POST /orders
Shop → Payments: authorise €84 · 380 ms
Payments --> Shop: ok
note Shop: stock reserved
Shop --> Browser: 201 Created · 520 ms
```

```markdown
Browser → Shop: POST /orders
Shop → Payments: authorise €84 · 380 ms
Payments --> Shop: ok
note Shop: stock reserved
```

## Anywhere, and as files

- **In a deck:** each step is a click.
- **In an article or a guide:** it plays as it scrolls into view, with ← → and Play.
- **A page of its own:** `type: diagram`, landscape or portrait, with a
  title and description: [a board](../live/diagram/board.md),
  [a plan](../live/diagram/plan.md), [a request](../live/diagram/request.md).
- **Files:** `si build board.md --png --jpg --pdf --svg --gif --mp4`, and a
  picture of every step with `--png`.

Every option: [§38 Delivery diagrams](../rules/38-delivery-diagrams.md).
