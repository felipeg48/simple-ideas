# 24. Mind maps

A mind map is a Markdown list of topics under a `# Title`. It is a page of
its own (`type: mindmap`) or a ```` ```mindmap ```` fence inside a slide, an
article, a guide page or any other document. The browser lays it out, draws
the lines, and lets the reader fold branches, move topics, and open linked
maps with a zoom. Without JavaScript (a screen reader, a feed, a print of the
source) the map is the nested list it was written as.

### A mind map page

```markdown
---
type: mindmap
title: The Thursday Promise
description: Every hand-off between "Buy now" and the doorstep.
layout: map
lines: curved
nodes: box
map-theme: auto
watermark: Northwind Logistics
watermark-position: corner
---
# The Thursday Promise

- Checkout
  - Cart & address
  - Payment → payments.md
  - Promise date {badge=Thu}
- Warehouse → warehouse.md
  - Picking
  - Label printed → slides: talk.md#7
- Delivery
  - Dusk departure ![The first vans](img/dusk.jpg)
  - [!sticky] Drivers asked for a later cut-off on Thursdays.
```

`si build` writes one self-contained HTML file per map file (into `--output`
like every other page) and lists it on the index as *N maps*; `si play` and
`si serve` serve it live; `si check` reports its problems. The page fills the
window.

| Front matter | Values | Default |
|---|---|---|
| `type` | `mindmap` | |
| `layout` | `map` (branches on both sides), `right` (a tree to the right), `org` (top-down; leaves stack under their branch), `radial` (rings around the centre) | `map` |
| `lines` | `curved`, `straight`, `orthogonal` (elbows with round corners), `tapered` (thick at the parent, thin at the child) | `curved` |
| `nodes` | `box`, `pill`, `underline` (deep topics are text on a line in their branch colour) | `box` |
| `map-theme` | `auto` (the page theme: `theme:` and light/dark), `harbor` (navy and amber), `blueprint` (blue grid, white lines, monospace), `ink` (black on white), `garden` (greens) | `auto` |
| `watermark` | text drawn on the map (and its exports); a picture stays a page layer | none |
| `watermark-position` | `corner` (`© text` bottom right), `across` (faint diagonal text over the map) | `corner` |
| `view` | `2d`, `3d` (topics as pills on a floor; see *In 3D*) | `2d` |
| `camera` | `iso` (isometric), `perspective` (3D only) | `iso` |
| `floors` | `on` (linked maps hover above their topic), `off` (3D only) | `on` |
| `kind` | `blocks` (pills), `bricks` (studded bricks on a baseplate), `toy` (cartoon shapes on an island); 3D only, see *Kinds* | `blocks` |
| `flow` | `on` (dots run along the lines, from each topic to its children), `off` | 2D `off`, 3D `on` |

`title`, `description`, `theme`, `mode`, `accent`, `fonts`, `lang` work as on
any page. `layout`, `lines`, `nodes`, `map-theme`, `view`, `camera`, `kind`,
`floors` and `flow` in any other file draw a warning: a fence takes them as options
instead.

### Topics

One list item is one topic; a nested list holds its children; the order is
the order on the map. The central topic is the `# Title` (without one, the
page title).

| Write | You get |
|---|---|
| `- Picking` | a topic |
| `- Promise date {badge=Thu}` | a badge after the text (`_` is a space: `{badge=two_days}`) |
| `- Delivery {collapsed}` | the branch starts folded, with a `+3` chip |
| `- Route 7 {pos=40,-12}` | the topic moved by hand (x, y from its automatic place); only this topic moves |
| `- Warehouse {branch-pos=52,104}` | the topic and its whole branch moved by hand |
| `- Dusk departure ![alt](img/dusk.jpg)` | a picture in the topic (resolved and inlined like any image; the alt text names it when the topic has no text) |
| `- [!sticky] Check with ops.` | a sticky note on a dashed line, handwritten |
| a fenced code block inside an item | a code topic, highlighted like any code block |
| `- Payment → payments.md` | `map ›`: opens that mind map file with a zoom (drill-down) |
| `- Refunds → #refunds` | `map ›`: opens the map `# Refunds {#refunds}` of the same page |
| `- Label → slides: talk.md#7` | a `slide 7` chip: opens the deck at that slide |
| `- Docs → https://example.com` | a `link` chip: opens the page in a new tab |
| `- Retry → ../article/retry.md` | a `page` chip: opens that page (`.md` becomes `.html`) |

Options combine: `- Checkout {badge=Thu collapsed} → checkout.md`. `->`
works as well as `→`. A code topic:

````markdown
- Retry with backoff
  ```go
  for try := 1; try <= 5; try++ {
      time.Sleep(backoff(try))
  }
  ```
````

### Many maps

One file per map is the usual structure: link them with `→ other.md`. The
linked mind map files (and the ones they link, cycles included) are inlined
into the page, so drill-down works offline from one HTML file, and a
breadcrumb leads back. One big file works too: separate the maps with `---`
and give each a `# Title {#id}`:

```markdown
# Payment

- Refunds → #refunds

---

# Refunds {#refunds}

- Partial
- Full
```

A map's id is its `{#id}`, or its title as a slug. On a page, `page.html#refunds`
opens that map first.

### In a slide or a document

````markdown
## Where an order goes

```mindmap layout=right lines=tapered theme=harbor
# Order
- Checkout
  - Payment
- Delivery {badge=Thu}
```
````

The fence takes `layout=`, `lines=`, `nodes=`, `theme=`, `view=`, `camera=`,
`floors=`, `kind=` and `flow=` (the same values as the front matter; `theme=`
is the map theme): ```` ```mindmap view=3d camera=perspective ````. Topics, links and drill-down work
as on a page; `→ file.md` inlines that map file. In a slide the map is as tall
as `--si-figure-max`; in a document `--si-mm-height`. The map grows in each time
its slide comes on screen. Two fences with the same title on one page keep
their own ids.

**One branch per click.** On a slide, `steps` (or `reveal=branches`) makes each
first-level branch a build step: → shows them one by one, in 2D and in 3D, and
they count as the slide's steps, so put one `[CLICK]` per branch in `Notes:`:

````markdown
## Where a promise breaks

```mindmap steps flow=on layout=right
# A late parcel
- Payment
  - Bank timeout
- Warehouse
  - One short-staffed shift
- Road
  - A cut-off that leaves too early
```

Notes: (1 min)
A late parcel almost always has one of three causes. [CLICK] Payment… [CLICK] The warehouse… [CLICK] And the road…
````

In a document `steps` is reported (there are no clicks there); every branch shows.

### Reading a map

| Do | What happens |
|---|---|
| click a topic | folds or opens its branch; a topic that only links opens its link |
| click `map ›` | drills into that map with a zoom; the breadcrumb zooms back out |
| drag a topic | moves only that topic; its lines stretch to follow |
| Shift-drag a topic | moves the topic and its branch |
| arrow keys on a focused topic | nudge it (Shift: its branch); Enter or Space acts like a click |
| drag the canvas | pans |
| `+` `−`, the wheel (a page), Ctrl + wheel or a pinch (a map in a slide or a document) | zoom |
| Fit | the whole map |
| Tidy | puts every moved topic back in its automatic place |
| `/` (a page) or Find (any map) | searches every topic, sticky note and code topic of every map (linked and inlined maps too); ↑ ↓ choose, Enter unfolds the path, opens the right map with the breadcrumb, and zooms to the hit (in 3D the camera flies there and the pill lifts). In a slide `/` stays the deck's search: use the Find button |
| Play (a page) | the overview, then one branch per step with the others dimmed; ← → Space PgUp PgDn step, it advances by itself every 3 s, Esc stops |

Moves made in the browser last until the page reloads; write them down as
`{pos=…}` / `{branch-pos=…}` to keep them. On a narrow portrait screen (a
phone) any layout is drawn as `right`, which stays readable.

### In 3D

```markdown
---
type: mindmap
title: The Thursday Promise
view: 3d
camera: iso
floors: on
---
```

`view: 3d` (or `view=3d` on a fence) draws the same maps as rounded pills on a
floor: the central topic is the tallest, deeper topics lower, each a matte
shade of its branch colour with a soft shadow. Lines are cables on the floor
with dots flowing along them (`tapered` draws as `curved`). Pictures, badges,
link chips, sticky notes, code topics and the watermark are printed on top of
the pills; colours and fonts come from the same `--si-mm-*` tokens, so the page
theme, dark mode and the map themes all apply.

| Do (3D) | What happens |
|---|---|
| drag | orbits the camera; right-drag or two fingers pan |
| the wheel (a page), Ctrl + wheel or a pinch (a slide or a document), `+` `−` | zoom |
| hover a pill | lifts it |
| click a pill | a card: *Open map ›* (drill-down), the slide or page link, *Fold*/*Unfold*, *Focus* |
| double-click a `map ›` pill (or one on a floor) | the camera flies into it and the next map grows out of the floor; the breadcrumb flies back |
| the camera button | Isometric (no perspective, the default) or Perspective |
| Fit, Find, Play | as in 2D: Play flies branch by branch with the rest dimmed (← → Esc) |
| **2D** / **3D** | switches the view; the Markdown only sets where it starts |

With `floors: on`, each map a topic links to (`→ payments.md`, `→ #refunds`)
hovers as a glass floor above that topic, joined by a dashed beam, so a map and
the maps it opens read as one building.

#### Kinds

```markdown
---
type: mindmap
title: Payment
view: 3d
kind: toy
---
```

````markdown
```mindmap view=3d kind=bricks
# One safe retry
- Wait thirty seconds {badge=30_s}
- Same idempotency key
- One try only
```
````

| `kind` | Topics | Lines | Ground | Linked maps |
|---|---|---|---|---|
| `blocks` (default) | matte rounded pills, text on top | cables | a soft shadow | glass floors |
| `bricks` | studded plastic bricks snapped to a stud grid; the text is printed on the front face; sticky notes and code are smooth printed tiles | rows of round tiles | a studded baseplate | small studded plates |
| `toy` | cartoon shapes with ink outlines and three-tone shading: a round drum for the central topic, a hexagon for a topic that opens another map, pills for the rest, flat cards for notes and code | light roads | a floating grass island with trees on its rim (never on a topic) and clouds | small floating islands |

The **kind** button in the 3D tools cycles Blocks → Bricks → Toy; the Markdown
only sets the start. `bricks` and `blocks` take their colours from the
`--si-mm-*` tokens; `toy` keeps its own candy palette and turns to a night
scene on a dark page or a dark map theme. Search, Play (the dimmed branches
include studs and outlines), drill-down, the cards and the flow dots work the
same in every kind.

The 3D view uses three.js (r128, MIT), inlined into the page like everything
else (no network): about 620 KB, only on pages that have a map starting in 3D.
Pages with 2D maps only stay small and show no 3D button. A folder or site
with several 3D maps can carry three.js once: `si build maps/ --share-assets`
(and `--no-inline-media`, which already puts files beside the pages) writes it
to `_si/three.min.js` next to the pages and links it, so each 3D page is about
600 KB smaller (the example maps: 2.1 MB → 1.5 MB); those pages then need the
`_si/` folder beside them. Without the flag every page stays one
self-contained file. A browser without
WebGL shows the map in 2D with a short note. Reduced motion skips the grow-in,
the flow and the camera flights.

On a phone held upright, the 3D view starts closer, on the central topic and
its first ring of branches; pan, zoom or Fit for the rest. Its **List** tool
shows the map as the nested list it was written as, and **Map** brings the 3D
view back.

### Looks

Branch colours come from `--si-mm-1` … `--si-mm-5` (by default the theme's
accent, second accent, ok, warn and a mix of accent and crit), so every
built-in theme and dark mode work. A `map-theme` (or `theme=` on a fence) sets
every `--si-mm-*` token at once; your CSS can set any of them:

```css
.si-mindmap { --si-mm-1: #0f766e; --si-mm-grid: rgb(0 0 0 / .05); }
.si-mindmap { --si-mm-hand: "Caveat", cursive; }  /* load it with fonts: google:Caveat */
```

The tokens (`si --export css-rules` lists them all): `--si-mm-bg`,
`--si-mm-fg`, `--si-mm-muted`, `--si-mm-surface`, `--si-mm-line`,
`--si-mm-accent`, `--si-mm-on-accent`, `--si-mm-1` … `--si-mm-5`,
`--si-mm-font`, `--si-mm-mono`, `--si-mm-hand`, `--si-mm-grid`,
`--si-mm-sticky`, `--si-mm-sticky-fg`, `--si-mm-height`.

### Editing in the Studio

`si --web` (and the desktop app, which runs it) opens a mind map page in the
**map editor**: the map itself, drawn by the same runtime, with a panel for
the selected topic and one for the page. Text / Map in the bar above switches
between the Markdown and the map; Map is the default for a mind map page, and
a new map from New → Mind map opens there. In any other file, put the cursor
in a ```` ```mindmap ```` fence and press Map in the bar that appears.

The editor's 2D / 3D button edits in 3D too: click a pill to select it (it
stands lifted), and Tab, Enter, Delete, Space, F2 or typing work as in 2D
(renaming opens a box at the top of the canvas). Moving a topic to another
parent is done in 2D or through the topic panel's "what it opens" fields; in 3D
there is no drag to reparent.

| Do | What it writes |
|---|---|
| click a topic | selects it (↑ ↓ ← → move the selection; Esc clears it) |
| Tab / Enter | a child / a sibling, named *New topic*, selected |
| F2, a double-click, or start typing | renames the topic in place (Enter keeps, Esc cancels) |
| Delete or Backspace | removes the topic and its branch |
| drag a topic onto another | makes it that topic's last child; dropped on the top or bottom edge, it goes before or after it |
| Alt + ↑ / ↓ | moves the topic up or down among its siblings |
| Shift-drag / Alt-drag | places the topic (`{pos=x,y}`) / its branch (`{branch-pos=x,y}`) freely |
| Space | folds or opens the branch (`{collapsed}`) |
| Tidy | removes every `pos=` and `branch-pos=` of the map |
| ⌘-click / Ctrl-click / Shift-click | adds a topic to the selection, or takes it out |
| ⌘C / ⌘X | copies / cuts the selected branches as a Markdown list (`- Checkout` / `  - Cart & address` …) to the clipboard |
| ⌘V | pastes a Markdown list (a copy from any map or file, or a list typed anywhere) under the selected topic |
| ⌘Z / Ctrl+Z, ⇧⌘Z / Ctrl+Y | undo / redo (the editor's own history) |

The topic panel edits the text, the badge, what it opens (`#map`, a mind map
file, `slides: deck.md#N`, a URL; the list suggests them), a picture from the
workspace and its alt text, sticky note, folded, and a code topic's language
and code (Add code makes one). The page panel sets every front matter key of
the table above (on a fence: its options), switches between the maps of the
file, and adds a map (`---` and `# New map {#new-map}`). Every change is
written back into the Markdown by si itself, as one undoable change, and saved
like any edit; the rest of the file (front matter, comments, other text) is
kept as it was. Topics of a linked map that comes from another file show, but
are edited in their own file.

### Exports

```bash
si build map.md --svg              # every map: map-<id>.svg (standalone, fonts inside)
si build map.md --png --scale 3    # map-<id>.png at 3× (default 2×)
si build map.md --pdf              # pdfs/map.pdf: one page per map, sized to it
si build map.md --mp4              # map.mp4: the first map's Play, branch by branch (needs ffmpeg)
```

Exports draw the 2D map (a `view: 3d` map too), with its theme, lines, moved
topics and watermark; the PDF also has the linked maps, and a `map ›` topic
there jumps to its page. They need headless Chrome, like `--pdf` for decks.
`--mp4` films Play: the overview, then each first-level branch with the rest
dimmed (2.5 s each, a short zoom between), and the overview again; it also
needs ffmpeg (`brew install ffmpeg`), an optional tool like Chrome. A
`view: 3d` page is filmed in 3D: each Play view is rendered by Chrome's
software WebGL in the page's kind, camera and theme, held 2.5 s, with the
camera's flight to the next branch filmed as it moves (`--dissolve`: still
views dissolving into each other).

### Other mind map tools

```bash
si mindmap import plan.opml          # → plan.md (also .mm FreeMind, .xmind XMind, .mmd Mermaid)
si mindmap import plan.xmind -o maps/plan.md
si mindmap export map.md --to xmind  # → map.xmind (opml, freemind, xmind, mermaid)
si mindmap export map.md --to mermaid --map refunds -o -   # one map, to stdout
```

Import keeps the topics, their order and web links; every XMind sheet becomes
a map. Export writes one map (the first, or `--map id`); XMind takes every
map, one sheet each. Pictures, badges, sticky notes and code stay in si.
Import never overwrites a file.

### In the terminal

`si play talk.md --terminal` and `si --terminal` draw a mind map (a fence on a
slide, or a mind map page in the preview) as a tree: `✲` the central topic,
`├─ └─` its branches, `→ target` for links, `[badge]`, `▨` a picture, `✎` a
sticky note, `‹go›` a code topic. In the terminal Studio, focus the preview
(Tab) to edit the tree: ↑ ↓ pick a topic, Enter renames, `c` adds a child, `o`
a sibling, `x` deletes, `[` `]` move it; `e` still opens the file in `$EDITOR`.

### What si check reports

A missing file (`→ other.md`, a picture), a map id that does not exist
(`→ #ghost`), an unknown topic option (`{size=3}`), a `pos=` without two
numbers, an unknown fence option or value, `steps` outside a slide, a map with
no topics, and unknown front matter values. A slide with `steps` counts one
step per branch, so its `[CLICK]` cues are checked like any build.

Example: `examples/mindmap/` (three linked maps: `thursday.md` in 3D with
its linked maps as floors, `payments.md` with two maps in one file,
`warehouse.md` in 2D, `org` layout with `orthogonal` lines), and the article
`examples/article/retry.md` has a fence.
`si new ideas --type mindmap` writes a starter (`--type mindmap3d` one that starts in 3D).
