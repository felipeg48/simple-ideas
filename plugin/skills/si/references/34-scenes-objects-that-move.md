# 34. Scenes: objects that move

A ```` ```scene ```` fence draws a few blocks and makes things move between
them, one click at a time: a parcel flies from the shop to the customer, a
token hops down a line of teams, an order splits into parcels, messages
queue up and a consumer takes them. It is written in plain words, in a slide
or a document page.

````markdown
## Tuesday to Thursday

```scene title="Tuesday → Thursday"
box Shop
box Customer
parcel flies Shop → Customer
```
````

**The first lines name the blocks.** `box Shop` is one block (its name is
its label; quote a name with a comma or an arrow in it: `box "A → B"`).
`row A, B, C` names several in a row; `column A, B, C` stacks them in one
column; `grid A, B, C, D` makes a small grid. `box Shop at 20,60` puts a
block at 20% across and 60% down the stage. With no places, si lays the
blocks out for you, left to right in the order they are named, and stacks
the blocks a split or a fan-out reaches in a column after it. Parcels,
tokens and messages need no line: write `parcel`, `token` or `message` in a
step and one appears; `parcel P2` names a second one.

**Every other line is one build step** (one click, one `[CLICK]` in
`Notes:`). `->` works as well as `→`, and `<->` as `↔`.

| Step | What moves |
|---|---|
| `parcel flies Shop → Customer` | a parcel flies along a curve from one block to the other, turning with the path and leaving a dashed trail; `parcel flies → Courier` flies on from where it is |
| `token hops Shop → Payments → Warehouse → Courier` | a token hops from block to block; each block lights up as it lands |
| `Order morphs → Event circle → Message diamond` | the block changes shape (`box`, `circle`, `diamond`, `pill`) and name, one after the other; the new name finds the block in later steps |
| `Order splits → Berlin, Lyon, Milan` | one block sends a parcel to each of the blocks named |
| `parcels merge → Delivery` | every parcel on the stage flies into the block and fuses with it, which lights up (with none on the stage, three wait at the left); or name them: `P1, P2 merge → Delivery` |
| `swap Courier ↔ Stock drift` | two blocks trade places, one over and one under |
| `tiles assemble 3×3` | tiles fly in from everywhere and snap into a 3 by 3 grid; `Plan assembles 3×3` builds the block `Plan` out of them |
| `tiles scatter` / `Plan scatters` | the tiles, or a block in pieces, burst outward and fade |
| `queue orders · Billing takes 3` | five messages line up in a lane called orders (`queue orders 8` for eight) and the consumer takes three from the front while the rest shuffle forward; a later `Billing takes 2` takes more. The consumer needs no box line |
| `Exchange fans out → Email, Stock, Analytics` | a copy of a message leaves the block for each block named; what flew into it goes in |
| `Warehouse opens → Pick, Pack, Ship` | the block grows to fill the stage and opens on its parts, which pop in; the parts need no box line |
| `route Depot → Stop 1 → Stop 2 → Stop 3` | a road draws itself under the blocks with a token riding its tip, and drops a pin at each stop |
| `parcel drops into Locker 12` | the parcel falls into the block, squashes as it lands and kicks up a little dust |

After a `·`, a step takes options: `arc` (the default for a flight) or
`straight`, `slow` or `fast`, or a time for the whole step: `· 2s`,
`· 800ms`. Lines starting with `//` are notes to yourself.

````markdown
## One order, three parcels

```scene
box Order €84
column Berlin, Lyon, Milan
Order €84 splits → Berlin, Lyon, Milan
parcels merge → Lyon · slow
```
````

````markdown
## orders.new

```scene title="orders.new"
queue orders · Billing takes 3
Billing takes 2
```
````

````markdown
## Who hears about an order

```scene title="order.placed"
box Shop
box Exchange
column Email, Stock, Analytics
message flies Shop → Exchange · straight
Exchange fans out → Email, Stock, Analytics
```
````

````markdown
## Inside the warehouse

```scene
row Shop, Warehouse
token hops Shop → Warehouse
Warehouse opens → Pick, Pack, Ship
```
````

````markdown
## Route 7, Thursday

```scene
row Depot, Stop 1, Stop 2, Stop 3
route Depot → Stop 1 → Stop 2 → Stop 3
parcel drops into Stop 3
```
````

The scene is one SVG that scales with the slide or the page (its stage is
16:9, wider when many blocks stand in a row). Blocks are the theme's
`--si-surface-2` with `--si-fg` text, lit blocks `--si-accent`; parcels,
tokens and pins are `--si-accent-2`, in light and dark. On a slide the
scene starts before its first step and each line is a click; in a document
it plays once when it is scrolled into view, with ← → and Play under it.
Reduced motion, print and PDF, the overview, thumbnails, handouts and a
`#5.2` link show the end of the step at once.

**Pictures of a scene.** `si build talk.md --png --svg` writes each scene
after each step, `talk-scene-1-step-1.png`, … and `talk-scene-1.png` of its
end, as charts do (§33). `si play --terminal` shows the blocks as text and
lists the steps.

`si check` reports, with the line in the fence, a line with no verb, a
block that was never named (with the names there are), an unknown option, a
block used where a parcel, token or message should be (`Shop flies …`), a
`takes` with no queue, and a grid or a route that cannot be drawn.
