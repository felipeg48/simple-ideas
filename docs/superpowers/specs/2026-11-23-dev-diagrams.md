# Delivery diagrams: the software life cycle, drawn and animated

Status: phase 1 DONE v4.1.0 (kanban, gantt, sequence, type: diagram);
phase 2 DONE v4.2.0 (cfd, burndown, cycle-time, throughput, built on the
diagram engine rather than the chart fence: one player, one step model, one
export path); phase 3 DONE v4.3.0 (state, flowchart (`flow` was taken by
the stage fence), class, wbs, storymap). Next: phase 4 (Studio editors,
terminal views).

## The idea

The diagrams a software team draws every week (a board, a plan, a flow, a
sequence, a cumulative flow, a story map) written in a few plain lines,
friendlier than Mermaid, and **animated**: a card moves across the board, a
plan's bars grow day by day, a request travels down a sequence, the flow's
bands fill. Each is:

- a **fence** in any deck or document (```` ```kanban ````, ```` ```gantt ````,
  …): one build step per click on a slide, played on scroll in a page;
- a **page of its own**, `type: diagram` with `diagram: kanban` (or the type
  named directly: `type: kanban`), `orientation: landscape | portrait`, sized
  for a slide, a poster or a doc;
- **exported** like an infographic: `si build x.md --png --svg --jpg --pdf`,
  one picture per step with `--steps`, `--gif` / `--mp4` of the animation.

Markdown and AsciiDoc alike: in AsciiDoc the same lines go in
`[kanban]` / `----` blocks (`deck.MarkdownToAdoc` maps them both ways).

## What is different from Mermaid

1. **Words, not syntax.** `Ada moves LOGIN-12 → Review` rather than node
   ids and arrows with brackets. Indentation and `·` options, as si's other
   fences.
2. **Time is part of the diagram.** Every type has a notion of *step* (a
   day, a sprint, a message, an event); steps are build steps, so the room
   watches the work flow, not a still picture.
3. **Theme colours, fluid layout.** SVG drawn in `--si-*` tokens, light and
   dark, reflowing to the frame (rule #1), no network script (rule #2).
4. **The numbers are computed.** WIP over the limit, the critical path, cycle
   time percentiles, throughput: si works them out from what is written and
   `si check` says what does not add up.

## The types

### `kanban`: the board, moving

````markdown
```kanban title="Checkout squad" wip=Doing:3,Review:2
columns: Backlog, Doing, Review, Done
LOGIN-12 Password reset · Backlog · @ada · bug
CART-7 Saved carts · Doing · @ravi
PAY-3 Apple Pay · Review · @lin · blocked

day 2
LOGIN-12 → Doing
PAY-3 → Done

day 3
CART-7 → Review
LOGIN-12 blocked: waiting for the mail team
```
````

Cards glide between columns on each `day`/`step`; a column over its WIP
limit flashes amber; blocked cards get a red edge and the reason. Swimlanes:
`lanes: Expedite, Standard` and `· lane=Expedite`. Card ages count up.

### `gantt`: the plan, with today moving

````markdown
```gantt start=2026-12-01 unit=day
Design        · 5d
API           · 8d · after Design
Web           · 6d · after Design
Tests         · 4d · after API, Web
Launch        · milestone · after Tests

today 2026-12-04
today 2026-12-10 · API 60%, Web 100%
```
````

Bars grow in as the plan arrives; each `today` step moves the today line and
fills progress; the critical path lights; a late task turns amber. `sections:`
group rows; holidays and weekends (`workdays: mon-fri`).

### `flow` (flowchart) and `sequence`, `state`, `class` (UML)

- `flow`: si's stage fence (already there, RULES §13) with a token that
  travels the path one click at a time: `token Start → Check → Pay`.
- `sequence`:

  ````markdown
  ```sequence
  Browser → Shop: POST /orders
  Shop → Payments: authorise €84
  Payments --> Shop: ok · 380 ms
  Shop --> Browser: 201 Created
  ```
  ````

  one message per click, the arrow drawing itself, a timing badge;
  `alt`/`loop` boxes with indentation.
- `state`: `Placed → Paid on authorised`, then `event authorised` steps move
  the current-state marker.
- `class`: `Order has many LineItem`, `Order: id, total(), pay()`; steps reveal
  classes in order; no animation beyond builds.

### `cfd` (cumulative flow), `burndown`, `cycle-time`, `throughput`

Charts from a table, on the chart engine (RULES §33):

````markdown
```cfd unit=items
| Day | Backlog | Doing | Review | Done |
| 1   | 30 | 4 | 2 | 0 |
| 2   | 28 | 5 | 3 | 1 |
| 3   | 27 | 4 | 3 | 3 |
```
````

Bands grow day by day; the WIP and lead-time guides (vertical and horizontal
gaps) draw at the end. `burndown` adds the ideal line; `cycle-time` is a
scatter of finished items (`LOGIN-12 · 2026-12-03 · 4d`) with 50/85/95%
percentile lines drawing in; `throughput` columns per week.

### `wbs` and `storymap`

- `wbs`: an indented list → a top-down tree (boxes, numbered 1, 1.1, 1.1.2),
  unfolding a level per click, `· 3d` estimates rolled up.
- `storymap` (user story map):

  ````markdown
  ```storymap
  ## Find a product | Choose | Pay | Track
  Search · Filters | Compare · Wish list | Card · Apple Pay | Email · SMS
  release MVP: Search, Card, Email
  release 2: Filters, Apple Pay, SMS
  ```
  ````

  the backbone of activities, the stories under them, and each `release` line
  a slice drawn across, one per click.

## Where it lives in the code

- `internal/deck/diagram/` (or one file per type in `internal/deck/`): parse
  each fence into a model plus its steps, with problems for `si check`.
- `internal/render/diagram_*.go`: SVG per type, every step's state as JSON,
  build-step marks (as `scene.go`); `web/diagram.js` + `diagram.css` tween
  between steps (FLIP for cards, path drawing, number counting), only on pages
  that use one.
- `type: diagram` on the document path (as `type: infographic`), exports
  through `cmd/si/infographic.go` (`pageShot`), `igmotion.go` (GIF/MP4),
  `svgexport.go`; `--jpg` added there for every page kind.
- Terminal: text renderings in `internal/tui` later; the Studio: New…
  starters and the gallery.

## Phases

1. **v4.1.0:** `type: diagram`, `kanban` and `gantt` (the two most asked
   for), `sequence`; exports png/svg/jpg/pdf/gif/mp4; RULES §38; a showcase
   deck on the site's Motion gallery; si check.
2. **v4.2.0:** `cfd`, `burndown`, `cycle-time`, `throughput` (chart engine:
   a `scatter` type and percentile lines).
3. **v4.3.0:** `state`, `class`, `flow` tokens; `wbs`, `storymap`.
4. Studio editors (drag a card to write a `→` line), terminal views.

## Open questions for the founder

- Fence names: `kanban`, `gantt`, `sequence`, `cfd`, `wbs`, `storymap`,
  `cycle-time` as written, or under one `diagram kanban` info string?
  (Proposed: the short names; `diagram` only as the page type.)
- Which three first if not kanban, gantt, sequence?
- Imports later? (Jira/Linear CSV → kanban/cfd, Mermaid gantt/sequence →
  si.)
