---
title: The si User Manual
author: simple-ideas
theme: studio
aspect: 16:9
---

# The si User Manual

Markdown in, animated fluid HTML slide decks out.

Notes: (1 min)
This manual is itself an si deck — build it with `si manual --slides` or `si manual --pdf`. For the exhaustive directive list, run `si rules`.

---

## What si is

- Write a talk in **plain Markdown**; `si` renders animated, fluid HTML slides
- `si play` serves them with live reload; `si build` writes one self-contained HTML file per deck
- The same Markdown also makes **articles** (`type: article`), multi-page **guides** (a folder with a `_nav.md`) and **labs** (`type: lab`); `si serve` shows them all live
- **Fluid, never fixed:** slides reflow from a phone to a 4K screen — no 1280×720 canvas
- Self-contained output: CSS, JS, SVG, images and video in a single file

---

## A deck is just Markdown

```markdown
---
title: My Talk
theme: studio
---

# My Talk
A subtitle under the title.

---

## A content slide
- a point
- another point

Notes: (1 min)
Order #4711 left the warehouse at 05:40 and reached the door on Thursday.

Points: Talking points
- a point to glance at
- another one
```

- Front matter sets `title`, `theme`, `aspect`, `mode`, and more
- `---` on its own line splits slides; `#` is the title slide, `##` a content slide

---

## The core syntax

| Write | You get |
|---|---|
| `# Title` | the title slide |
| `## Heading` | a content slide |
| `---` | a slide break |
| `- item` | a bullet list |
| ` ```lang ` | a highlighted code block |
| `Notes: (2 min)` | verbatim teleprompter script + a per-slide time budget |
| `Points:` | speaker-notes talking points (the teleprompter's second tab) |
| `Notes[Ana]:` / `Points[Ana]:` | one speaker's script and points, with `speakers: Ana, Felipe` in the front matter |
| `<!-- layout: two-col -->` | a layout directive (see next) |
| `![Caption](clip.mp4)` | a video player (`<!-- image: poster=still.jpg -->` for its still, else `clip.poster.jpg`, its first frame), copied beside the page |
| ` ```video src="clip.mp4" mode=modal title="…" ` | si's video player: its own controls, captions and chapters from `clip.vtt` and `clip.chapters.txt` beside the video, `mode=modal` for a dialog over the dimmed page (every modal player on the page is one gallery) |
| `[The slides (PDF)](talk.pdf)` | a link to a local file, copied beside the built page (missing: `si check` says so) |

Everything is in **`si rules`** (or RULES.md) with an example for each.

---

## Layouts

<!-- layout: two-col -->

Pick a layout per slide with `<!-- layout: name -->`:

- `two-col` — two columns (`<!-- column -->` splits them)
- `big-number` — one huge figure
- `quote` — a centred pull-quote
- `center` / `image` — centred / full-bleed image

<!-- column -->

```markdown
<!-- layout: big-number -->

**99.99%**

uptime across three regions.
```

---

## Build steps

Reveal content one keypress at a time:

<!-- steps -->
- First this appears
- then this
- then this — each on its own press

```markdown
<!-- steps -->
- First this appears
- then this
```

Notes: Steps also drive canvas and stage state (next slides).

---

## Things that arrive

<!-- animate: count -->
**12,480** orders on time

<!-- step -->
<!-- animate: highlight -->
Shoes at the door in ==two days==.

```markdown
<!-- animate: count -->
**12,480** orders on time
```

Notes: Seven build animations: type, words, highlight, count, draw, decode and confetti. Each one runs as its block arrives and ends with the block exactly as written. [CLICK] Lists can arrive in focus, dealt like cards, or re-sorted by their numbers.

---

## Stage diagrams

A ` ```stage ` fence generates a themed, animated flow — no external tool:

```stage
dir: LR
Draft --> Review --> Ship
dot Draft..Ship 2s
```

- Chains, fan-out and merges; node labels `A[Label]`, edge labels `-->|x|`
- `dot`/`flow A..B` animate a route; `step <node>` reveals on build steps
- Full guide: `docs/stage-diagrams.md`

---

## Mind maps

A nested list becomes a map you can fold, move and drill into:

```mindmap layout=right
# A mind map
- Page: type: mindmap
  - One file per map → linked with →
  - Or one file: maps between --- lines
- Fence on a slide or in a document
- Topics {badge=options}
  - [!sticky] Sticky notes, pictures, code
```

- `layout:` map, right, org, radial · `lines:` curved, straight, orthogonal, tapered
- `nodes:` box, pill, underline · `map-theme:` auto, harbor, blueprint, ink, garden
- On a page: `/` searches every map, Play walks the branches; full guide: RULES.md §24
- `view: 3d` · `camera:` iso, perspective · `floors:` on, off: pills on a floor, linked maps hover above
- `kind:` blocks, bricks, toy: pills, studded bricks on a baseplate, or a cartoon island
- On a slide, `steps` shows one branch per click; `flow: on` runs dots along the lines
- `si build map.md --svg --png --pdf`; `si mindmap import plan.opml`, `export --to xmind` (or `xmind8`, the legacy XMind 8 file)
- A `view: 3d` map: `--png` / `--pdf` picture the 3D view (`--scale`); no SVG of 3D (raster only: `--svg` writes the 2D map)
- 3D on a phone (≤ 480 px wide or upright): branches on one side, labels read level, icon tools; one finger orbits, two pinch and pan, a tap opens the card
- In the Studio, Map view: Tab adds a child, Enter a sibling, drag to move, typing renames

---

## Timelines

```timeline layout=line
## Cards
### 1890 · Hollerith tabulator
Punched cards sorted by machine for the census.
## Stored programs
### 1945 · Merge sort
John von Neumann merges sorted runs.
### 1961 · Quicksort
Tony Hoare partitions around a pivot.
## Libraries
### 2002 · Timsort
The sort Python still uses.
```

- Page: `type: timeline` · `## Era · color=` · `### date · title`, a description, `![caption](pic "credit")`
- `layout:` snake (rows, 30 fit one 16:9 page), line, vertical, eras (a column per era), cards (picture cards on a dated rail) · `orientation:` landscape (a slide), portrait (a poster)
- `alternate:` on, off · `scale:` even, time · `closing:` a last line · `## Fuentes · sources`
- A landscape page is one slide: each era a build step, `Notes:` for the teleprompter
- `si check` counts milestones, pictures, eras; `require: 20 dates, 15 images` checks a rubric
- `si build timeline.md --png --pdf --svg` (`--pdf`: 16:9 pages with selectable text; `--svg`: a vector picture, text as text); `si new history --type timeline`; full guide: RULES.md §29
## Infographics

```infographic steps=off
## 1 page · stat · accent
`type: infographic`, or this fence on a slide

## Block kinds · list
- stat, compare, steps, quote, icons, image
- chart=bar, donut, line, progress from a table

## Bento grid · text
`span=2` or `wide`, `tall`, `accent` after the kind
```

- Every `##` is a block: `## 2.4 s · stat`, `## Share · chart=donut · wide`; no kind: si guesses
- `orientation: landscape` is a 16:9 slide (`steps: on`: one block per click); `portrait` a poster that scrolls
- `layout:` grid, columns, flow · `aspect:` 16:9, 4:3, 9:16, a4 (or an exact frame: phone, square, 1080x1350) · `sources:` the footer
- A block's text shrinks to fit it (down to 60%; `si check` warns past that)
- `si build poster.md --png --pdf --svg` (`--scale 3` for print; `--svg` a vector picture); full guide: RULES.md §30

---

## Animated infographics

```infographic animate=build steps=off
## The path · flow
Producer -[publish]-> Queue -[deliver]-> Consumer

## 50,000 · stat · accent
messages a second, counted up as it comes in
```

- `animate: build` (once) or `loop`; `duration: 12s`; per block `· enter=fade|rise|grow|count · at=3s`
- By kind: numbers count up, bars grow, donuts sweep, lines draw, steps light one by one, flows draw and send packets
- `## Path · flow`: `A -> B -> C`, `A -[label]-> B`, one path per line; laid out for you, arrows numbered
- `aspect:` phone, story, portrait, square, landscape or `1080x1350`: an exact frame, fluid inside
- `si build x.md --gif --mp4 --png` (`--fps`, `--scale`, `--size phone,square`); slides play on arrival; the Studio preview has ▶ Play / ↺ Replay

---

## Animated charts

```chart bar race
| Type     | One | Two | Three |
|----------|-----|-----|-------|
| bar      | 3   | 5   | 9     |
| donut    | 6   | 4   | 7     |
| bar3d    | 1   | 6   | 8     |
```

- ```` ```chart bar|column|line|area|donut|pie|progress|bar3d|donut3d|funnel3d ````, then a table: labels, number columns
- `steps`: a column per click, the header big in the corner · `sort` · `race` (both, rescaled each step) · `stacked` · `unit=%` · `title=` · `explode="Late"`
- Bars grow and glide to their new rank, arcs sweep, lines morph, numbers count; same `id=` in a row: one show that turns bars into a donut; same `id=` on the next slide carries on
- In a document: plays once scrolled into view, ← → Play under it; full guide: RULES.md §33

---

## Video edits

```text
---
type: video
source: demo.mp4
---
- 0:45–0:50 · blur 120,80 400x60
- 1:34 · bubble "Log in as the admin user" · to 820,410 · 4s · pop
- 1:52 · freeze 2s
- 2:40 · zoom to 600,300 900x500 · 3s
- 4:00–4:04 · cut
- end · card "Next: clustering" · 3s
```

- Times are the source's: an edit never moves another line; `at chapter "Install" + 5s`, `at said "management UI"` (from the `.vtt` beside it)
- Edits: `cut`, `trim start|end`, `freeze`, `speed 2x`, `replace with`, `insert clip|image|card`, `mute`; `· fade|dissolve|wipe|slide`
- Callouts in the theme: `title`, `text`, `bubble`, `arrow`, `box`, `circle`, `underline`, `spotlight`, `blur`, `pixelate`, `keys`, `badge`, `lower-third`; the camera: `zoom to`, `pan to`, `zoom out`
- Moving callouts: `· to 100,200 → 400,300 over 2s`, `· path A B C` (a bubble's tail, an arrow's head, a text, a rectangle, a blur or pixelate); a rectangle's size changes too: `· to 10,10 200x60 → 400,300 300x90`
- Sound and takes: `say "…" · duck 30%`, `music`, `captions on` or `captions words` (the word being said marked), `refilm feature "Name"` (the demo script runs that feature again)
- `si check edit.md` · `si play edit.md` (a live preview) · `si build edit.md --with captions,vertical,highlight` (the lines marked `· highlight`, else each chapter's first sentence); full guide: RULES.md §32
- In the Studio: the Video view, markers that select their line, callouts dragged into place (written back in source pixels), a moving callout's places and a zoom's rectangle dragged too, ⌘B to build

Notes: A recording gets its finishing touches from a Markdown file beside it: each line is a time and what happens there, and every time is the one you see in the original video.

---

## Code windows

```go id=send window=mac title="send.go" numbers step="Once"
func send(o Order) error {
	return post(o)
}
```
<!-- callout: 2 One try only -->

```go id=send window=mac title="send.go" numbers step="Retry" hl=2-4
func send(o Order) error {
	for try := 1; try <= 5; try++ {
		if post(o) == nil { return nil }
	}
	return errGaveUp
}
```

- `window=` mac, windows, terminal, none · `theme=` night, paper, dusk · `backdrop=` dusk, ocean, mint, graphite
- Same `id=` in a row: a show that morphs; on a slide, one click per step
- A real file: `<!-- include: compose.yaml window=mac numbers hl=3 -->` (every option; `as=cat` puts a script under `$ cat start.sh`)
- `si code file.md --png --svg --gif`; in the Studio, put the cursor in a block and choose Window

Notes: A code block can be a window. [CLICK] Blocks with the same id become one window whose lines glide into the next step.

---

## Animated canvases

One inline SVG whose parts change state slide by slide:

```markdown
<!-- svg: journey.svg paid -->
<!-- steps: +shipped -->
- the order ships
```

- Helper attributes: `data-show/hide/ok/alert/flow/packets/grow/move/count`
- State words in the markdown drive CSS transitions in the SVG
- Scaffold one with **`si new-canvas`**; recipe in `docs/authoring-canvases.md`
- Or draw it in the Studio's **Motion view**: frame 1, then frames where you
  drag, resize and recolour parts; it writes the canvas and the step lines

Or let si move the slide itself: with `<!-- transition: morph -->` what two
slides in a row share (the same heading, picture or words, or a block named
`<!-- id: name -->` on both) glides from its old place and size to the new;
the rest fades. `morph 0.9s` sets the length.

Ten more transitions are played by the page itself: `cube`, `flip`, `iris`,
`wipe`, `shatter`, `depth`, `blinds`, `portal`, `glitch` and `stack`
(`<!-- transition: cube 1.2s -->`); going back plays each one in reverse.
Six build animations add depth: `tilt`, `steps: carousel`, `explode`,
`flap`, `flyin` and `orbit` (RULES.md §9).

---

## Themes

- Nine built-ins: `studio`, `paper`, `mono`, `slate`, `twilight`, `forest`, `sunrise`, `neon`, `newsprint`; `si theme list` prints them
- `theme: my.css` loads a CSS token file; `theme: https://…/x.css` fetches one
- `/* si theme: Acme · extends paper */` as a theme file's first comment puts paper's tokens underneath: the file sets only what it changes
- Make one in the Studio: ⌘K › *New theme*, then colours (light and dark), fonts and sizes beside a live specimen deck, with contrast checks and **Fix**
- `si theme from logo.svg [--name Acme] [--accent '#e4572e'] [--use]` makes `themes/acme.css` from a logo's colours, light and dark, every pair WCAG AA (the Studio: *New theme* › **From a logo…**)
- `si theme check acme.css` (contrast in light and dark, wrong values); `si theme export acme.css [--format dtcg]` and `si theme import file.css|file.tokens.json [--use]` share one as CSS or W3C Design Tokens
- `si --export themes` writes a gallery; `si --export css-rules` a starter stylesheet
- Your own CSS layers in `si.user` after the theme — tokens are `--si-*`

---

## Presenting

| Key | Action |
|---|---|
| → ← Space | next / previous step or slide |
| `o` | overview — one column per topic (section); ← → across topics, ↑ ↓ within one, Enter opens the selected slide |
| `p` | teleprompter — script + speaker-notes tabs, timers, rehearsal pacing, ● Record (Shift+R) |
| `t` | break timer (configurable presets + animation layer) |
| `a` | draw over the slide |
| `d` / `f` / `?` | light-dark / full screen / help |
| `v` | auto-play — the deck reads its `Notes:` aloud and advances at each `[CLICK]`; any key that moves the deck, a click or the pen pauses it |
| `` ` `` | terminal drawer — run local commands live (`si play` on a loopback address only) |
| `a`, then ⌥ + move | in draw mode, hold Option (Alt) and the cursor draws where it goes: one finger on a trackpad, no click (si.conf `draw.hold: shift`, `ctrl`, `meta` or `off` picks the key) |
| `q` | `si play --remote`: the phone and tablet links as QR codes, on the projected screen (`--https`: over HTTPS, so the phone stays awake) |

Open on a phone: `si play --addr 0.0.0.0:7070` prints a LAN URL + QR.

Teleprompter or pen on a tablet: `si play talk.md --remote` keeps the deck on
loopback and opens a keyed second port (play port + 1) with one QR per role:
`tp` (the teleprompter, drives the deck) and `draw` (a pen pad whose strokes
show on the projected slide) and `clicker` (a phone remote: big Next / Prev,
where you are, what comes next, a stopwatch). `--remote=tp` picks roles, `--remote-addr` the
address. See `docs/remote.md`.

Two presenters: `speakers: Ana, Felipe` in the front matter and
`Notes[Ana]:` / `Notes[Felipe]:` on the slides. Each teleprompter picks its
speaker (the picker in the bar, `s`, or `?speaker=Ana`; `--remote` prints one
teleprompter code per speaker): your script in full, the other's lines as a
short cue, a **→ Felipe** banner when the turn passes, and your own clock.
`speakers.control: turn` in `si.conf` lets only the speaker whose turn it is
advance. In the terminal: `si play talk.md --terminal --notes --speaker Ana`.

Auto-play: **▶ Auto-play** (or `v`) lets the deck present itself, for a
kiosk, a rehearsal you listen to, or a talk to share. Under `si play` each
part is voiced with the deck's voice (Kokoro `af_heart` when installed,
`voice:` and `voices: Ada=af_heart, Ravi=am_michael` to choose) and cached;
`si build talk.md --autoplay` writes the voice beside the page
(`talk.voice/`), `--autoplay=inline` inside it; a plain build speaks with the
browser's voice. Captions show the line being said; `#autoplay` in the
address opens it ready. Pause and Resume on the same slide carry on
mid-part (the voice from where it stopped, the browser's voice from its
sentence); another slide starts its part from the beginning.

The **terminal drawer** (`` ` ``) slides up a tabbed, real shell for running
commands mid-demo (`docker ps`, `psql`, …). It exists only under
`si play` bound to a loopback address — never in a built deck, never on
`--addr 0.0.0.0`. Viewer tabs read and edit files beside the shells; **◫**
splits the drawer into two panes (drag the divider), **⌘P** opens a file by
a few letters of its path, **⌘⇧F** searches the files' text and opens a
result at its line, and clicking line numbers spotlights lines for the
audience (Esc clears). **w** opens web pages (a broker's console) in floating
windows over the slides, each origin through a loopback proxy so consoles
that refuse framing still show; links and redirects to other sites stay
proxied (a login form's POST to another site keeps its fields), the address
box shows the real address, and ← → cross sites. When si films the drawer
(`si demo`, `si build --mp4` with `[TERM …]` cues) its shells start with a
clean prompt, the folder and `$`, never your own; si.conf `terminal.prompt:
clean` asks for it in a live talk too. The slider in the drawer's bar sets
its opacity (40 % to 100 %, remembered): the slide shows through, dimmed,
and the text stays opaque; si.conf `terminal.opacity: 85%`, a cue's
`· opacity 80%` and `embed-terminal "Lab 1" · opacity 80%` set it too. See
`docs/terminal.md`.

A talk that shows something live can set it up itself: front matter
`setup:` (must finish), `start:` (keeps running, `· wait for "ready"`),
`stop:` and `cleanup:`, each repeatable, in the order written. `si play` and
`si serve` run them before the first page (a **getting ready** page shows each
step), keep the programs running while you present, and stop and clean up
when they quit; `si build` runs them only to film the talk's `[TERM …]` /
`[BROWSER …]` cues (`si build talk.md --mp4`, `--no-live` to skip). A talk can also come from a repo
or a URL: `si play github:acme/talks@v2.1/orders/talk.md`, asked once per pin
(`--trust` for CI). Example: `examples/live/talk.md`.

---

## Present from your phone

```bash
si play talk.md --remote --https
```

**Once per laptop:** nothing; the first run makes its certificate authority
(local addresses only, ten years, the key stays on the laptop).

**Once per phone or tablet:** press `q`, scan *First time on this phone*,
install the certificate, then on an iPhone turn on **Settings › General ›
About › Certificate Trust Settings**. Optional on the laptop: `si ca trust`.

**The room:** `--audience` gives everyone a follow-along link
(`<!-- qr: audience -->` on a slide): slides only, never your notes;
`<!-- poll: id · a · b -->` lets them vote, live bars on your screen
(`· hidden`: the bars wait for your next click); `<!-- cloud: id -->` turns
their few-word answers into a word cloud (you can hide a word from the
teleprompter's Q&A tab); they
ask questions from their phones into the teleprompter's Q&A tab (`3`).
On a video call: `--tunnel` gives them a temporary public link instead.

**Every talk:** press `q`, scan the Clicker or Teleprompter code. The phone
stays awake and the notes travel encrypted. With Tailscale on both:
`--https=tailscale`, nothing to install.

Notes: (1 min)
Your phone can be the clicker and your iPad the teleprompter. The first time,
each phone trusts this laptop once: scan the setup code, install the
certificate, and switch on full trust. From then on, every talk is one scan. [CLICK]

Points:
- The trust switch on iPhone is the step people miss: without it Safari warns
- One authority per laptop; a phone that trusts yours trusts no one else's
- Full guide: simple-ideas.org/features/remote.html

---

## Web demos

A demo script with `app:` drives a web app in headless Chrome, step by step:

```markdown
## Pay for an order

What it does: a customer pays with a card and sees "Paid".

Say: A test card goes through in a second.
- click "Order #4711"
- type "4242 4242 4242 4242" into "Card number"
- click "Confirm"
- expect text "Paid"
```

- Elements by what a person reads: text, label, placeholder, `#id`, `role=button "Pay"`
- `si demo test shop.md`: ✓ / ✗ per step, `report.html`, exit 1 on a failure, `--junit` for CI
- `$VAR` values come from the environment and are never printed

Notes: (1 min)
The same Markdown can drive your own web app. Each feature is a list of
steps a person would take: open a page, click Pay, type a card number, and
expect the word Paid. si runs them in a browser you never see, and when a
step fails it shows what it expected, what it found and a picture of the page. [CLICK]

Points:
- `app:` is a URL, or the app's folder: si serves it on loopback
- The same script makes the videos: next slide
- Example: `examples/webdemo/` (shop.md passes, broken.md fails on purpose)

---

## Web demos, on film

```bash
SHOP_PASSWORD=shop-demo si demo run shop.md
```

- A drawn cursor, click ripples, `note "…" at "Field"` callouts, blurred `$VAR` fields
- Each `Say:` line voiced first: the narration sets the pace
- `voice: kokoro:af_heart` (the default when installed), `system`, `elevenlabs`, `openai`
- `pronounce: si = ess eye`: the voice says it, the captions keep the word
- One file: `shop.mp4`, every feature after a title card, chapters inside
- More on request: `--with captions,subtitles` (or `with:` in front matter, `demo.with:` in si.conf)
- `captions` `silent` `audio` `subtitles` `transcript` `chapters` `clips` `report` `readme` `vertical` `highlight` `all`
- Long waits play faster, with a ⏩ badge; the voice never speeds up
- Scenes fade into each other; `zoom to`, `spotlight`, `window` and `drag` steps

Notes: (1 min)
The same script makes the video. Every Say line is spoken first, so the
words set the pace and the steps fit between them. A cursor glides to each
button, notes point at the fields that matter, and passwords are blurred.
You get one file: the whole run, a title card and a chapter per feature.
Ask for more when you need it: captions in a band under the picture,
subtitles, the narration alone, a transcript, a clip per feature, the report.
A long wait plays faster, and the voice stays at its own pace.
Scenes fade into each other, the picture can zoom in on what matters, a
spotlight dims the rest, and the teleprompter window is filmed too. [CLICK]

Points:
- Default is one file; `--with all` makes today's full set; `--no-captions` is gone (captions are opt-in)
- One panel only: `--region ".agents"` (or `region:`, a feature's `Region:`), or `--crop x,y,w,h`
- `--fast-forward 5s` or `--no-fast-forward` for waits
- Cuts fade by default (0.4 s): `transition: slide`, `--transition cut`, or `· pop` after a `go`, `browser`, `terminal`, `window` step
- `zoom to "Total"` … `zoom out`, `pan to "…"`; `Region: ".agents" · zoom` zooms in instead of cropping
- Filmed at 2× and scaled down, so zooms stay crisp; `--scale 1` (or `scale: 1`) films at 1×: faster, less disk
- `spotlight "Pay"` (alias `focus`), `· circle`, `· blur`, `· label "…"`, `· for 3s`, `spotlight off`
- `window "si-presenter"` films the teleprompter popup; `window main` comes back; `drag from "A" to "B"` draws
- Kokoro is si's own voice: `si voice install kokoro`, one command; without it, the computer's voice
- Try a voice aloud: `si voice say "Hello" --voice kokoro:bf_emma`; `si voice list`
- Cloud keys: `si voice login openai` keeps the key in your own si.conf, never the project's
- `voice:` in si.conf sets the default voice
- Cloud voices send only the Say: text; lines are cached, re-runs re-voice only changes
- A failing step ends that clip and lands in the report
- Needs Chrome and ffmpeg: `si doctor`

---

## Web demos, as a promo

```bash
si demo draft https://harbor.example --out harbor.md
si demo run harbor.md --with vertical,highlight
```

- `si demo draft`: a first script from the site's own words, robots.txt obeyed
- `session: keep`: the cookie banner accepted once, in `## Before`
- Sticky headers stay clear: `header-offset: auto | 72px | none`
- `click "Buy" in frame .preview`: steps inside framed pages
- `intro:`, `outro:`, `logo:`, `url:`, `music: bed.mp3 · 20%`, `theme:` (a built-in or a `.css` file)
- `Lower-third: Ana Ruiz · Head of Operations`
- `--with vertical` (9:16) and `--with highlight` (about a minute)

Notes: (1 min)
A promo starts from the site itself. Draft reads the pages, keeps to the site
and to its robots file, and writes a script from the site's own words, ready
to test. One browser profile for the whole film means the cookie banner is
accepted once, before the first chapter. Spotlights and zooms land below a
sticky header, and steps reach into framed pages. The brand goes around the
film: an intro card with the logo, an outro with the address, music that
steps back while the voice speaks, a name caption for the speaker. And the
same take gives a vertical cut for phones and a highlight of about a minute.
[CLICK]

Points:
- Draft never clicks or submits; `--pages 6` by default; polish every `<!-- draft -->` line before filming
- `session: keep` profile is si's own, removed at the end; default stays a fresh one per feature
- An element in view is no longer scrolled: the closing spotlight does not nudge the page
- Frames: same-origin and cross-origin; `in frame "name"` matches name, title or id
- Music is ducked by the voice (sidechain); `<script>.m4a` stays narration only
- Highlight: each feature's first sentence; vertical: captions large, `vertical-fill: blur | theme | color #hex`
- Lower thirds are drawn after a `--region` crop, never cut off
- Example: `examples/webdemo/promo.md` (a sticky header, a cookie banner, a framed shop)

---

## Web demos with a terminal

```markdown
start: ./bin/broker --port 5672 · wait for "ready"

Say: First we send one message.
- terminal "Client" · backdrop=ocean
- run "./bin/send orders hello" · wait
- expect output "sent 1 message"
- browser
- expect text "1 message waiting"
```

- `start:` programs in the background for the run; `start "…" as load` for a feature; `stop load`
- `terminal`, `run`, `expect output`, `browser`: the shell filmed as a framed window
- a failure on purpose: `run "si check bad.md" · exit 1` (or `· fails`), then `expect output "problem"`
- `note "…" at line "ready"`, `press ctrl+c`, `expect load exited 0`
- `terminal split right` (or `down`): a second shell in the same window, both filmed; `run "…" in 2`, `expect output "…" in 1`, `focus 2`

Notes: (1 min)
Many apps are a terminal and a web page together. The script starts the
broker before the first feature and waits until it is ready. Then a terminal
opens on screen, the command is typed for real, and its output is checked.
One more line goes back to the browser, where the console shows the message.
When the run ends, everything it started is stopped. [CLICK]

Points:
- Three lifetimes: the run (`start:`, `stop:`), a feature (`start` step), some blocks (`start` … `stop`)
- Programs stop by process group: SIGTERM, SIGKILL after 5 s, also on Ctrl-C
- A failure shows the terminal's last 40 lines in the report and JUnit
- Windows: PowerShell under ConPTY (colours, `· wait`, full-screen programs); `shell: cmd` runs on pipes
- Example: `examples/webdemo/broker.md` (a tiny Go broker, `go run`)

---

## The deck on stage

```markdown
app: onstage/talk.md

Say: Let's see it in action.
- slide 2
- embed-terminal "Lab 1"
- run "docker compose up -d --wait"
- embed-browser "Console" · at 700,60 · size 640x420
- click "Queues" in embed-browser "Console"
```

- `app:` a deck: si demo starts `si play` on it, and stops it at the end
- `embed-terminal "Name"`: the play drawer's shell tab, by name; `· split right`; `embed-terminal min|max|close`
- `embed-browser "Name"`: a `browser:` link (or a URL) in a window over the slides; `· at`, `· size`, `· zoom`
- `slide 4`, `next`, `prev`; inside the window: `… in embed-browser "Name"`
- Or in the deck's own `Notes:`: `[TERM Lab 1: make up]` and `[BROWSER Console · at 560,40 · size 1010x640 · zoom 125%]` (`[BROWSER Console: http://…/#/queues]` moves that window), filmed by `si build --mp4 --voice` with the deck's `setup:` / `start:` run first (not with `--dissolve`: stills have no drawer, and si build says so)

Notes: (1 min)
For a course, the best minute is often the one where the slide comes alive.
The demo can play the deck itself: the slide stays on screen, the presenter's
drawer opens under it and runs the lab's commands, and the broker's console
opens in a window right over the slide. The same moments can be written in
the deck's own script, so the narrated video shows them where they are said. [CLICK]

Points:
- Everything goes through `window.siPlay`, the play page's public API (RULES §14), never si's CSS classes
- `si demo test` runs the same steps headless as a check
- Teleprompter: `[TERM …]` and `[BROWSER …]` are badges, never read aloud; auto-play acts on them
- Example: `examples/webdemo/onstage.md`

---

## Before, after, and from anywhere

```markdown
start:   make demo-up · wait for port 22310
prepare: ./scripts/create-queues.sh
step:    github:acme/ops@v1.4/scripts/seed.sh --orders 50
setup:   acme queue purge orders
stop:    make demo-down
cleanup: rm -rf data/
```

- `start:`, `prepare:`, `step:` in the order written; `setup:` before each feature
- `## Before` and `## After`: steps around the features, never filmed, not counted
- `stop:` and `cleanup:` always run: a pass, a failure, Ctrl-C
- `si demo test docs/demos/`: every script, one report, one JUnit
- `gist:`, `github:…@tag`, `https://… · sha256`: pinned, asked once, `--trust` for CI
- A script from a URL brings what its `files:` lists, each with its own sha256; a gist brings all its files

Notes: (1 min)
A real demo needs more than one program. The script starts the broker,
creates its queues, seeds the data and starts a consumer, in the order you
wrote them. Before and After sections run steps around the features without
filming them. When the run ends, whatever happened, it stops what it started
and cleans up. A whole folder of demos runs as one suite, and the scripts it
needs can come from a gist, a repo or a URL, each pinned and trusted once. [CLICK]

Points:
- A failed `prepare:` fails every feature with its output; a failed `## Before` skips them
- `~/.si/trusted` remembers a yes per pin; a pin whose contents changed stops the run
- Demos from a repo: `si demo test github:owner/repo@v2/demos/` (output in `demo-out/<repo>/`)
- Example: `examples/webdemo/morning.md`

---

## Courses

```markdown
---
title: A first web page
level: beginner
nav: left · accordion · sticky
---
- Module 1: How the web works
  - [What a browser does](01-web/browser.md)
  - [From address to pixels](01-web/journey.md)
  - [A page loads](01-web/page-load.mp4)
- Module 2: Writing HTML
  - [Build your first page](02-html/first-page.md)
  - [Check yourself](02-html/quiz.md)
```

- Readings, slides, videos, labs, instructor demos and quizzes, in modules
- A deck lesson as its narrated video, captioned, with the slides as a PDF to keep
- `si build course/ -o html/`: a website bundle with a home page; `--zip`, `--instructor`
- `--scorm` for any LMS, `--pdf` for a workbook, `si publish --to skilljar`
- Progress stays in the learner's browser: ticks, rings, Continue, a certificate

Notes: (1 min)
A course is what si already makes, put in order. The list in the course file
holds the modules, and each link under a module is a lesson: a page is a
reading, a deck becomes slides that present themselves, a video file plays
in the page, a lab can be ticked off, and a quiz is scored. The learner's
build leaves the instructor demos out, and the instructor's build keeps them
with the teleprompter and the lab solutions. The same course packs as SCORM
for any learning platform, prints as a workbook for learners and a guide for
the instructor, and publishes straight to SkillJar. [CLICK]

Points:
- Lesson kinds come from the link target; `· kind=` overrides
- A deck as a video lesson: `· kind=video · pdf · handout · slides`; `si build` films it, with captions and its first slide as the poster (`· poster=` to change it)
- A filmed web demo with `· kind=demo`: a learner lesson labelled Demo; without `app:` it stays the instructor's runbook
- `nav:` also places the menu of guides, books and sites: left, right, top, bottom
- `si new web-course --type course` writes an example with every kind
- `--scorm` one SCO per lesson, `--scorm=single` one for the course
- `--pdf` workbook; `--instructor --pdf` the guide with answers and runbooks
- Skilljar fetches videos and packages from `--base-url`; dry run unless `--yes`
- A quiz is a self-check: answers only as hashes, not a secure exam
- RULES.md §28

---

## Course results, to your team

```markdown
---
title: A first web page
webhook: https://hooks.slack.com/services/…
webhook-format: slack
webhook-learner: ask
---
```

- Lessons done, quiz scores, lab checks, videos watched, the course finished
- As JSON, a Slack, Teams or Discord message, or an xAPI statement for an LRS
- A notice tells learners first; `webhook-consent: required` waits for their OK
- `<!-- check: command · expect "text" -->` in a lab: **✓ Check** in `si play`

Notes: (1 min)
A course can tell you how the class is doing. With one line it sends small
events as learners go: a lesson done, a quiz score with which questions were
right, a lab step that passed, a video watched, the whole course finished.
They arrive in your Slack, Teams or Discord channel, in your own endpoint, or
in a learning record store as xAPI. Learners see a notice first, and the
events never carry their answers. [CLICK] A lab step can check itself too:
the command written under it runs with one click, and the result comes back
as one more event.

Points:
- Sent by the page itself (sendBeacon, a queue offline, retried on the page with backoff): website bundle, file://, SCORM
- `webhook-base: https://…`: stable xAPI activity ids wherever the course is opened (else `urn:si:course:…` from disk)
- `webhook: relay` + `si serve --relay` (and the `--remote` listener): the address off the page, events signed (`X-Si-Timestamp`, `X-Si-Signature` over timestamp.body, 5-minute window; `webhook-secret:` in your own si.conf)
- `audience: instructor` marks the trainer's own clicks
- `si demo test --webhook URL --webhook-format slack`: a CI run's summary in the channel
- RULES.md §28 (Results to your team; Labs that check themselves), §27

---

## The class, live

```text
si serve web-course/ --class --addr 0.0.0.0:8000
```

- Learners by lesson: ✓ done, quiz scores, lab checks, videos watched
- Who's stuck: the same check failed twice, no progress for 10 minutes
- The quiz questions most got wrong, per lesson summaries, CSV

Notes: (1 min)
While you teach, the course can report to the screen in front of you. Serve
it with class on, and every learner's page sends its progress to your
laptop: a grid of learners by lesson, filling in as they work. The ones who
failed the same lab check twice, or have not moved for ten minutes, rise to
the top. Under the grid, the quiz questions most of the room got wrong, so
you know what to explain again.

Points:
- `/_si/class` on the loopback, or the keyed link + QR si prints for a tablet
- Names only when learners type one (`webhook-learner: ask`, the default with `--class`); No thanks keeps them out
- Kept in `.si/class/<date>.jsonl`; a restart picks up today's class; the Day picker opens earlier days
- Quiz questions named by their titles
- The Studio's Class mode: Start the class, the dashboard, link and QR beside it
- `si play course/ --terminal --class`: the same class as text in the terminal
- Works beside a `webhook:`; no webhook needed
- RULES.md §28 (The live class view)

---

## Project structure

```text
structure:
  rabbitmq/
    md/       markdown
    slides/   deck
      pdf/    deck:pdf
    html/     site
      images/ image
    dist/     html/:zip, deck:zip
```

- A `structure:` block in si.conf: one folder per line, what goes there after it
- `si build rabbitmq` writes each kind into its folder, links rewritten, zips last
- No block: the flat layout as always; `si new rabbitmq --structure` makes one
- RULES.md §31

---

## The CLI

```bash
si play <dir>              # serve with live reload (--dev: in-place hot reload)
si build <dir> --pdf       # self-contained HTML (+ PDF)
si build talk.md --handout # handouts/talk.pdf: slides only; =lines, =points, =notes add space or speaker text
si check <dir>             # validate; exit 1 on a problem
si demo test shop.md       # run a web demo script against a web app as a test (report.html, --junit, --webhook URL)
si demo run shop.md        # the same, filmed and narrated: shop.mp4 with chapters; --with captions,subtitles,clips,report,… for more
si doctor                  # the optional programs (Chrome, ffmpeg, git…): found or not, how to install each
si new <name>              # scaffold a deck (the showcase, every rule)
si new <name> --structure  # a project with a structure map: folders per kind, sources in md/
si new <name> --type article   # a small starter: deck, article, guide, book, course, site, page, lab, demo, mindmap, mindmap3d, codeshow
si build course/ -o html/ --zip   # a course as a website bundle (+ zip); --instructor for the instructor's edition
si build course/ -o html/ --scorm --pdf   # + <course>-scorm12.zip for an LMS and the learner workbook PDF
si publish course/ --to skilljar --base-url URL   # a dry run; --yes sends it (si publish login skilljar first)
si new <name> --type website --kind resume   # a website made for one job: resume, portfolio, landing
si publish talk.md --to linkedin   # a dry run of the post; --yes sends it (also devto, hashnode, ghost, wordpress)
si code show.md --mp4      # a code show as video (si build map.md --mp4: a mind map's Play; needs ffmpeg)
si fix <dir> [--write]     # raw-HTML components (hl, pill, lockup, cards) into si syntax
si qr https://… --png --svg --pdf  # a QR code file (--logo, --style dots, --wifi, --vcard)
si --export skill          # the si skill for your AI agent (Claude Code: /plugin install si@simple-ideas)
si --export player site/   # si's video player for a page si did not build (data-video="clip.mp4" cards)
si mcp                     # MCP tools for agents: check, rules, new, build, qr, screenshot
si new-canvas <name>       # scaffold an animated canvas
si manual --slides         # this manual
si rules                   # the full format reference
si migrate <src>           # an AI migration prompt for an existing deck
```

---

## Terminal looks

```bash
si play talk.md --view checklist   # ledger (default), badges, checklist or steps
SI_VIEW=steps si build talks/      # or ui.view: steps in si.conf
```

- `si play`, `si serve`, `si build` and `si check` speak in one of four looks on a terminal: **ledger** (aligned label and value, a green ✓ per line), **badges** (a coloured block per line), **checklist** (emoji and plain words), **steps** (a tree with timings)
- Problems always show the file and line; green is done, yellow a warning, red a problem, cyan a link
- Pick it with `--view`, `SI_VIEW`, or `ui.view:` in si.conf (the flag wins, then `SI_VIEW`, the project's si.conf, your own)
- Piped, in CI, with `NO_COLOR=1` or `TERM=dumb`: the plain lines, unchanged
- **`si play` monitor:** press `m` for CPU, memory, shells, links, the pages open by role, the current slide with its timers and recent reloads, redrawn every second; `m` again goes back, `p` shows the network and audience links as QR codes, `w` opens the web monitor, `o` the deck, `q` quits

---

## The control room

```text
si play talk.md --audience
  monitor (this machine; the control room for polls and questions):
  http://127.0.0.1:7070/_si/monitor?k=<key>
```

- `si play` prints a keyed link to the **web monitor**: the same numbers as `m` (uptime, CPU, memory, viewers by role, the slide and its timers, the links with their QR codes, shells, the recent reloads and warnings), live in the browser
- It runs the room: **polls** (close and reopen voting, reveal a `· hidden` poll's results, reset), **questions** (approve, show on screen, mark answered, hide), **word clouds** (hide a word, clear)
- This machine only, with the run's key; with `--remote` (or `--remote=monitor`) a tablet opens it from the `Monitor` link and code
- What you decide is kept: the polls, words and questions, with their states, go to `.si/audience/` when `si play` stops

---

## Share it

- **Post it:** `si publish talk.md --to linkedin` turns the deck into a
  document post, its PDF swipeable in the feed; an article goes to LinkedIn as a
  link, or to dev.to, Hashnode, Ghost or WordPress. Always a dry run first;
  `--yes` sends.
- **Get reviews:** `si serve . --review` prints a keyed link for people on
  your network: they leave comments on slides, sections or one paragraph,
  bullet or figure (click it to pin the note), and the Studio's
  Review panel shows them live, to reply and resolve; `--https` puts the
  link on HTTPS. **History** in the Studio
  shows what changed, commits, restores earlier versions, switches or makes
  branches, and pulls and pushes (git).
- **Film it:** `--mp4` on `si code`, on a mind map build (in 3D when the map
  is) and on a deck build (every slide and step, `--dwell 2` for a teaser)
  writes video that plays anywhere, with the transitions and morphs as they
  move, made with ffmpeg if it is installed.
- **Record your talk:** **● Record** in the teleprompter (under `si play`)
  records your voice with every slide change; `si build talk.md --narration
  latest` turns it into the video, paced and voiced as you gave it, and `si
  rehearse talk.md` shows where you ran long or rushed. No recording?
  `si build talk.md --mp4 --voice kokoro` narrates the video from the
  `Notes:` themselves.
- **Point to it:** `<!-- qr: https://… · caption=The slides -->` puts a QR
  code on a slide or page; `si qr https://… --logo mark.png --pdf` makes one as
  a file for a poster or a badge.
- **Tell us:** Help ▸ Report a Problem (or `si report`) opens an issue with
  your version filled in; the rules and examples live on simple-ideas.org.
- **Start fast:** an empty folder in the Studio opens a gallery of starters
  with live thumbnails, and a six-stop tour of the window.

Notes: (1 min)
When the talk is done, it does not have to stay on your laptop. si
publish posts the deck to LinkedIn as a document people swipe through, and
sends an article to dev.to, Hashnode, Ghost or WordPress; it always shows you the requests
before it sends anything. A QR code on the last slide takes the room to the
slides. Code shows and mind maps become short videos. And
when you open an empty folder in the Studio, a gallery shows what you can
start from, with a tour of the window if you want one. [CLICK]

Points:
- Publish: dry run by default, `--yes` to send; logins in the config dir, never the repo
- MP4 needs ffmpeg; si says how to install it
- Gallery + tour: first run, or ⌘K any time

---

## Where to go next

- **`si rules`** — the complete, authoritative directive reference
- `make docs-site` — the rules as a browsable site, built with si itself
- `docs/publish.md` — LinkedIn, dev.to, Hashnode, Ghost and WordPress: logins, scopes, requests
- `docs/stage-diagrams.md` — the stage flow DSL
- `docs/authoring-canvases.md` — hand-authored animated canvases
- `docs/timer-animations.md` — the break-timer animation layer
- `docs/reference/developer-reference.md` — architecture and how to extend si

Notes: (0.5 min)
Thanks — now go build a deck.
