# 38. Delivery diagrams

The diagrams a software team draws every week, written in plain lines and
**animated**: a board whose cards move from column to column, a plan whose
bars grow and fill as the today line moves, a request that travels from one
party to the next. Each is a fence, in a deck (one build step per click) or
in any document (it plays as it is scrolled into view, with ← → and Play
under it), or a page of its own with `type: diagram`.

````markdown
## The board this week

```kanban
columns: Backlog, Doing, Review, Done
wip: Doing 2, Review 2
LOGIN-12 Password reset · Backlog · @ada · bug
CART-7 Saved carts · Doing · @ravi
PAY-3 Apple Pay · Review · @lin

day 2
LOGIN-12 → Doing
PAY-3 → Done

day 3
CART-7 blocked: waiting for the mail team
+ SHIP-9 Parcel lockers · Backlog · @mo
```
````

**`kanban`.** `columns:` names the columns, left to right. A card is one
line: an id and a title (`LOGIN-12 Password reset`, or just a title), then
after `·` its column, `@who`, `blocked: why` and a tag. A line starting
with `day`, `step`, `sprint`, `week` or `iteration` begins a step; under
it `ID → Column` moves a card (it glides there and is outlined),
`ID blocked: why` and `ID unblocked`, `ID done` (to the last column), and
`+ ID Title · Column` adds one. A `wip: Doing 2, Review 2` line sets
limits: a column over its limit turns amber and says `3/2`, and the
caption names when it happened. Cards in the last column are tinted done.

````markdown
```gantt start=2026-12-01
section Build
Design · 5d
API · 8d · after Design
Web · 1w · after Design
section Ship
Tests · 4d · after API, Web
Launch · milestone · after Tests

today 2026-12-04 · Design 80%
today 2026-12-10 · Design done, API 60%, Web done
```
````

**`gantt`.** `start=` is the first day. A task is a name, then a length
(`5d`, `2w`), `after` the tasks it waits for (comma separated), `from
YYYY-MM-DD` for a fixed start, a `%` done, or `milestone`; `section Name`
groups the rows. si schedules the tasks (calendar days), draws the critical
path (the tasks with no slack) with an outline, and says when the plan ends
and its critical path under it. Each `today DATE · Task 60%, Task done`
line is a step: the today line moves there, the bars fill, and a task past
its end and not done turns amber (late), a done one green. The bars grow in
as the slide arrives.

````markdown
```sequence
Browser → Shop: POST /orders
Shop → Payments: authorise €84 · 380 ms
Payments --> Shop: ok
note Shop: stock reserved
Shop --> Browser: 201 Created
```
````

**`sequence`.** The parties are named as they first appear (or in a
`participants:` line). `A → B: what` is a call, `A --> B: what` a reply
(dashed), `A → A: what` a call to itself, and `· 380 ms` a timing badge;
`note A: text` puts a note on a lifeline. One message per step: the arrow
draws itself and the two parties light up.

**Flow metrics.** Four more fences draw a team's numbers on a plot, and
reveal them left to right, one click per day (or per finished item, per
week):

````markdown
```cfd
| Day | Backlog | Doing | Review | Done |
|-----|---------|-------|--------|------|
| Mon | 30 | 4 | 2 | 0 |
| Tue | 28 | 5 | 3 | 1 |
| Wed | 27 | 5 | 3 | 3 |
```

```burndown days=10 unit=points
| Day | Remaining |
|-----|-----------|
| 1 | 40 |
| 2 | 37 |
```

```cycle-time
LOGIN-12 · 2026-12-03 · 4d
CART-7 · 2026-12-04 · 6d
```

```throughput
| Week | Done |
|------|------|
| 48 | 11 |
| 49 | 8 |
```
````

- **`cfd`** (cumulative flow): a column per state, the first (the backlog)
  first and the last (done) last; the bands stack with done at the bottom
  (green), the state before it in the accent. Under it: the throughput (done
  per day) and, with work in progress, the time from start to done it means
  (Little's law). `unit=` and `per=` name the items and the day.
- **`burndown`**: `| Day | Remaining |`, an optional third column for the
  scope; `days=` is the plan's length (the ideal line runs to zero there).
  Under it: what was burned, and the day the pace points to.
- **`cycle-time`**: one finished item per line, `ID · YYYY-MM-DD · days`
  (or a table `| Item | Finished | Days |`): a dot per item, by the day it
  finished; the 50th, 85th and 95th percentiles drawn last; items past the
  85th in amber. Under it: "85% of the items finished within N days".
- **`throughput`**: `| Week | Done |`: a column per row, growing one per
  click, then the average.

More than twelve rows reveal in about eight steps.

**Graphs, trees and maps.** Five more, each one line per thing:

````markdown
```state
Placed → Paid on authorised
Paid → Shipped on picked
Shipped → Delivered on scanned
event authorised
event picked
```

```flowchart
Start → In stock?
In stock? → Pay: yes
In stock? → Backorder: no
Backorder → Pay
Pay → End
token Start → In stock? → Pay → End
```

```class
Order: id, total(), pay()
LineItem: sku, qty
Order has many LineItem
CardPayment is a Payment
Order uses Payment
```

```wbs
- Parcel lockers
  - Design
    - Research · 2d
    - Screens · 3d
  - Build · 8d
```

```storymap
## Find a locker | Send a parcel | Pick it up
Map · Opening hours | Label · Size check | Code by SMS · QR code
release MVP: Map, Label, Code by SMS
release 2: Opening hours, QR code
```
````

- **`state`**: `A → B on event` transitions (`[*] → A` names the first
  state; else the first written). Each `event name` line is a step: the
  current state (lit) follows its transition, which lights up. The first
  state is a pill.
- **`flowchart`**: `A → B` or `A → B: label` (a box can be named alone on
  a line); a name ending in `?` is a decision (a diamond), `Start`, `End`,
  `Done` and `Stop` are pills. `token A → B → C` lines are steps: a token
  moves box by box, lighting the box and the edge it came along.
- **`class`**: `Name: field, method()` for a class and its members;
  `A has many B` (1 → *), `A has one B`, `A is a B` (inheritance, a hollow
  triangle), `A uses B` (dashed) and `A → B`. One class per step, a
  relation as soon as both its classes are shown.
- **`wbs`**: an indented list, the project first: its parts in a row under
  it, their parts stacked below, numbered 1, 1.1, 1.1.2; `· 3d` (or `2w`,
  `5h`, `8pts`) estimates added up to every parent and the total. One level
  per step.
- **`storymap`**: `## Activity | Activity` is the backbone; a row of `|`
  cells puts stories (split by `·`) under each activity; each `release
  Name: story, story` line is a slice, one per step, and stories in no
  release come last, under *later*.

Graphs are laid out left to right from the first box, a layer per step away
from it; a step back goes under the chart, or over it to the top box of its
layer.

| Fence option | What it does |
|---|---|
| `title="…"` | a small caption over the diagram |
| `steps=off` | the last state only, no build steps |
| `facts=off` | no caption of what si worked out (critical path, end, WIP) |
| `wip=Doing:2;Review:2` | kanban: the WIP limits, as the `wip:` line does |
| `start=YYYY-MM-DD` | gantt: the first day |
| `days=10` | burndown: the plan's length in rows |
| `unit=points`, `per=week` | metrics: what is counted, and per what |
| `width=600` | the layout width in viewBox units (1000; a portrait page uses 600) |

**A page of its own.** `type: diagram` makes one frame around the page's
diagrams, with the `title:` and `description:` over them:
`orientation: landscape` (16:9, the default) or `portrait` (3:4, the
diagrams laid out narrower so their text is larger).

````markdown
---
type: diagram
title: The checkout board, week 49
description: Four days of the squad's work, one click a day
---

```kanban
columns: Backlog, Doing, Done
…
```
````

**Exports.** `si build board.md --png --jpg --pdf --svg --gif --mp4` writes
the frame at its last step as `board.png` (1600 × 900 CSS pixels, or 1200
× 1600 portrait, × `--scale`), `board.jpg`, a one-page `pdfs/board.pdf`,
`board.svg` (text kept as text), and the whole animation as `board.gif` and
`board.mp4` (ffmpeg). In any deck or document `--png` / `--svg` also write
a picture of every diagram and of each of its steps:
`<page>-diagram-<n>-step-<k>.png`. In AsciiDoc the same lines go in a
`[source,kanban]` (or `gantt`, `sequence`) block. Reduced motion, print and
PDF show each diagram at its last step. On a narrow page (a phone) a
diagram keeps a readable width and scrolls sideways inside its own box. `si check` names what does not add
up: a card or column that does not exist, a task with no length or one
that waits for itself, a progress for a task that is not there.
