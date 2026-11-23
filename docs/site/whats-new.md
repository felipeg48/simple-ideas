---
title: What's new in si 4
description: si 4 teaches classes from your own computer, a Docker box per learner beside the lesson. si 3 finished web demos.
---

# What's new in si 4

## Flow metrics (4.2)

A team's numbers, drawn and revealed day by day: a ```` ```cfd ```` whose
bands fill as the days pass, a ```` ```burndown ```` against its ideal line,
a ```` ```cycle-time ```` scatter with its percentiles, a ```` ```throughput ````
column per week. Under each, what it says: the throughput, the time from
start to done, the day the pace points to, "85% within 7 days". See
[§38](rules/38-delivery-diagrams.md).

## Delivery diagrams (4.1)

The diagrams a team draws every week, in plain lines, and moving. A
```` ```kanban ```` board whose cards glide from column to column day by day,
with a column over its WIP limit turning amber. A ```` ```gantt ```` plan si
schedules itself, the critical path outlined, the today line moving as the
bars fill and late tasks turn amber. A ```` ```sequence ```` of calls and
replies that draw themselves one click at a time. In a deck, in an article,
or a page of their own (`type: diagram`), exported as PNG, JPG, PDF, SVG,
GIF or MP4. See [§38](rules/38-delivery-diagrams.md) and the
[delivery diagrams](features/diagrams.md) page.

## Classes on your computer (4.0)

A course with a `sandbox:` can now be taught to a whole class from your own
machine. `si class` opens the planner: pick the course and the image, give
each box its memory and CPUs, add seats from a roster CSV (or a count), and
print the codes sheet, one QR code per learner. Start it with `--tunnel` (a
public HTTPS address) or `--https` (your network). A learner opens their
link and gets their own Docker box beside the lesson: the terminal drawer's
shells are in it, every ✓ Check runs in it, its ports open in their
browser. They reach nothing of your computer: not a shell, not a file, not
si's own pages.

While it runs the planner shows every box (starting, ready, its memory),
**Reset** gives a learner a fresh box from the image, seats can be added on
the day, and **Progress** opens the live class view. A memory budget keeps
the class inside what the sandbox host can hold: on a 128 GB Mac, about 38
learners at 2 GB each. See [§37](rules/37-sandbox-labs.md).

```bash
si class new docker-course --name "Docker basics" --roster people.csv
si class start docker-basics --tunnel
```

## si 3

si 3 brings together a month of work on one idea: **decks that present
themselves, demos that film themselves.** One Markdown script drives a web
app, a terminal or a deck, checks every step as a test, and comes out as a
narrated video you would publish.

<!-- si-video: presenting -->

## Sandbox labs (3.44)

A lab can bring its own machine. Name a Docker image in `sandbox:` and
`si play` starts a box from it before the lab opens: Docker inside, the
lab's files in `/lab`, the terminal drawer's shells in the box, and every
✓ Check run there, so "three containers are running" is checked where they
run. Links written `sandbox:8080` open the box's own ports. The boxes live
in a Colima VM of their own, unprivileged under sysbox and with none of
your folders mounted, and they are gone when si play stops. See
[§37](rules/37-sandbox-labs.md).

```bash
si sandbox host init          # once
si new web-lab --type sandbox
si play web-lab/lab.md --open
```

## Architecture pages and status boards (3.43)

Two documents that were kept by hand as hundreds of lines of HTML are now
one Markdown file each. `type: architecture` numbers the sections, makes
the first paragraph of each its lead, turns a list of `` `tag` **Title** —
body `` items into cards, a quote into a note, a ```` ```flow ```` fence
into a diagram and the Phases list into a row of phase cards.
`type: status` draws a status board from a checklist: pills and tiles from
the front matter, then one card per area with Done, Missing and Improve
columns. Every percentage and bar is counted from the lists when the page
is built, so the board never says 80% when the lists say 60%. See
[§35](rules/35-architecture-pages.md) and [§36](rules/36-status-boards.md).

```bash
si new design --type architecture
si new board --type status
si build board.md --pdf
```

## Books for publishers (3.41)

Publishers each want their own format: Word, Markdown, PDF. A book in si now
builds into all of them from the same chapters. `--pdf` prints it as a
book: a cover and title page, contents with the page of every chapter and
section, a page for each part, the chapter's title at the top of each page
and the number at the bottom. `--docx` writes a Word document with real
styles (headings, code, captions, lists, quotes, tables), the pictures in
it and a contents field Word fills when it opens. `--md` writes clean
Markdown, the whole book and one file per chapter, with the pictures
beside; `--rtf` and `--epub` cover the rest. Numbers, cross-references and
index terms come out as the HTML book shows them. See
[§21](rules/21-books-references.md).

```bash
si build book/ -o out/ --pdf --docx --md --rtf --epub
```

## Scenes: objects that move (3.39)

A `scene` fence names a few blocks and then, one line per click, moves
things between them, in words anyone can read: a parcel flies from the
shop to the customer, a token hops down the line, an order splits into
parcels that merge again at the door, two blocks swap, tiles assemble or
scatter, a queue fills and its consumer takes from the front, a message
fans out to every subscriber, a block opens on its parts, a route draws
itself and drops pins, a parcel drops into a locker. si lays the blocks out
for you, draws them in the theme's colours and shows the end of a step at
once for print, reduced motion and jumps. Every verb is in
[§34](rules/34-scenes-objects-that-move.md).

```scene title="order.placed"
box Shop
box Exchange
column Email, Stock, Analytics
message flies Shop → Exchange · straight
Exchange fans out → Email, Stock, Analytics
```

## Charts and motion, finished (3.38)

A chart show whose kind changes now folds: each bar bends round the ring
into its own slice of the donut, and unrolls back into a bar the other
way. `si build talk.md --png --svg` writes every chart as a picture,
`talk-chart-1.png` for its last step and `talk-chart-1-step-2.png` for each
step, the SVG with real text. Infographic blocks take the 3D charts
(`chart=donut3d`). In an article, guide or book, `animate:` blocks and the
`steps: deal`, `focus`, `reorder` and `carousel` lists play once as you
scroll to them. A jump to another slide, from a link or the overview, plays
the transition of the slide it lands on.

```chart bar id=region title="Orders by region"
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```

```chart donut id=region
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```

## Charts that move (3.37)

A ```` ```chart ```` fence draws a Markdown table as bars, columns, a line,
an area, a donut, a pie, progress bars, or solid 3D bars, a 3D donut and a
3D funnel, on the theme's colours. With `steps` each number column is one
click: bars grow and shrink, arcs sweep, lines bend and the numbers count.
`race` keeps the bars in order of value, so they glide past each other.
Charts with the same `id=` in a row are one show: a click turns the bars
into a donut, and the same `id=` on the next slide carries on.

```chart bar race unit=% title="Late orders by cause"
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
```

[The rules for charts →](rules/33-animated-charts.md)

## Motion (3.36)

Ten new slide transitions, played by the page itself with no library:
`cube`, `flip`, `iris` (it opens from where you clicked), `wipe`, `shatter`,
`depth`, `blinds`, `portal` (the camera flies into a picture on the slide,
and the picture becomes the next slide), `glitch` and `stack`. Each one
plays in reverse when you go back, and a length sets it per slide:

```markdown
<!-- transition: portal 1.4s -->
## The receipt
```

Build animations run as their step arrives: `type`, `words`, `highlight`
(the marker of a `==mark==` paints in), `count` (an odometer), `draw` (an SVG
canvas or stage diagram draws itself on), `decode` and `confetti`. A list
can arrive in focus (`steps: focus`), dealt like cards (`steps: deal`), or
shown as written and then re-sorted by its numbers (`steps: reorder`).

Since 3.36.1, six more give a slide depth with CSS 3D: `tilt` turns the slide into layers
that follow the pointer, `steps: carousel` stands a list's items in a ring
and turns one to the front per click, `explode` pulls a list apart into
plates, `orbit` circles the first item with the rest, `flap` shows a word
on split-flap tiles, and `flyin` brings words from far behind the slide.

Reduced motion, print, PDF, the overview and the teleprompter previews show
the final state.

## The control room (3.34)

`si play` prints a keyed link to a **web monitor** at `/_si/monitor`: uptime,
CPU and memory, who is watching by role (the deck, the teleprompter, the
remote teleprompter, draw pad and clicker, the audience's phones, review and
class pages), the slide with its timers, the links with their QR codes, and
the recent reloads. Under `--audience` it runs the room: close, reopen or
reset a poll and reveal a hidden poll's results; approve the questions,
put one on screen, mark it answered or hide it; hide a word from a cloud or
clear it. It answers on this machine only, with the run's key; `--remote`
offers it to a tablet as the `monitor` role. In the terminal monitor, `p`
shows the network and audience links as QR codes and `w` opens the page.

## Terminal looks (3.32)

`si play`, `si serve`, `si build` and `si check` print in colour on a
terminal, in one of four looks: **ledger** (the default), **badges**,
**checklist** or **steps**. Pick one with `--view`, `SI_VIEW` or `ui.view:` in
si.conf. In `si play`, press `m` for a live monitor: CPU, memory, shells,
links, open pages, the current slide and the recent reloads. Piped output and
`NO_COLOR=1` stay plain.

```text
$ si play talk.md --view checklist

🎬  si play 3.32.0  serving talk.md · theme studio

✅  Deck     The Thursday Promise  (26 slides)
✅  Live     live reload on  (163 files watched)

👉  Open     http://127.0.0.1:7070/talk.html

m monitor · p codes · w web monitor · o open · q quit
```

## Demos edited like a video

- **Transitions** at every scene change: `fade` by default, or `slide`,
  `wipe`, `pop`, `zoom`, `circle`, per script or per step.
- **Camera moves:** `zoom to "Agents panel"`, `pan to`, `zoom out`, and a
  region that opens wide and zooms in.
- **Spotlight** (`focus`): everything but one element dims, with a label.
- **Windows and dragging:** film a popup such as the teleprompter, and draw
  on a slide with the mouse.

```markdown
Say: The script is in large type, with the clocks and the next slide beside it.
- press p
- window "si-presenter" · pop
- spotlight #prompt · label "What you say, word for word"
```

## A voice, and decks that present themselves

- **Kokoro `af_heart`**, an open-source voice that runs on your computer, is
  si's voice: `si voice install kokoro` sets it up in one command and makes it
  the default in `~/.si/si.conf`. The computer's own voice and cloud voices
  work too.
- **Auto-play** (`v` or ▶ Auto-play): a deck reads its `Notes:` aloud and
  clicks at every `[CLICK]`. `si build --mp4 --voice af_heart` turns the same
  notes into a narrated video.

## Demos of anything on screen

- **Web apps** (`app:`), **terminals** (`terminal`, `run`, `expect output`)
  and programs in the background (`start:`, `start … as`), in one script.
- **A test as well as a video:** `si demo test` checks every step, writes a
  report and JUnit for CI; `si demo run` films the same steps.
- **One file by default** (`<script>.mp4`), the rest with `--with captions,
  subtitles, transcript, clips, report…`; crop to a region with `--region`.

## si's own videos

Every video on this site is a demo script in `docs/demos/`, tested on every
push and filmed with Kokoro on every release.

<!-- si-video: studio -->

[Web demos, in full](features/webdemos.md) · [Decks](features/decks.md) ·
[The changelog](https://github.com/felipeg48/simple-ideas/blob/main/CHANGELOG.md)
