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
| `<!-- layout: two-col -->` | a layout directive (see next) |

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
- `si build map.md --svg --png --pdf`; `si mindmap import plan.opml`, `export --to xmind`
- In the Studio, Map view: Tab adds a child, Enter a sibling, drag to move, typing renames

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

---

## Themes

- Nine built-ins: `studio`, `paper`, `mono`, `slate`, `twilight`, `forest`, `sunrise`, `neon`, `newsprint`; `si theme list` prints them
- `theme: my.css` loads a CSS token file; `theme: https://…/x.css` fetches one
- `si --export themes` writes a gallery; `si --export css-rules` a starter stylesheet
- Your own CSS layers in `si.user` after the theme — tokens are `--si-*`

---

## Presenting

| Key | Action |
|---|---|
| → ← Space | next / previous step or slide |
| `o` | overview — one column per topic (section); ← → across topics, ↑ ↓ within one, Enter opens the selected slide |
| `p` | teleprompter — script + speaker-notes tabs, timers, rehearsal pacing |
| `t` | break timer (configurable presets + animation layer) |
| `a` | draw over the slide |
| `d` / `f` / `?` | light-dark / full screen / help |
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

The **terminal drawer** (`` ` ``) slides up a tabbed, real shell for running
commands mid-demo (`docker ps`, `psql`, …). It exists only under
`si play` bound to a loopback address — never in a built deck, never on
`--addr 0.0.0.0`. See `docs/terminal.md`.

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

## The CLI

```bash
si play <dir>              # serve with live reload (--dev: in-place hot reload)
si build <dir> --pdf       # self-contained HTML (+ PDF)
si build talk.md --handout # handouts/talk.pdf: slides only; =lines, =points, =notes add space or speaker text
si check <dir>             # validate; exit 1 on a problem
si new <name>              # scaffold a deck (the showcase, every rule)
si new <name> --type article   # a small starter: deck, article, guide, book, site, page, lab, demo, mindmap, mindmap3d, codeshow
si publish talk.md --to linkedin   # a dry run of the post; --yes sends it (also devto, hashnode)
si code show.md --mp4      # a code show as video (si build map.md --mp4: a mind map's Play; needs ffmpeg)
si fix <dir> [--write]     # raw-HTML components (hl, pill, lockup, cards) into si syntax
si qr https://… --png --svg --pdf  # a QR code file (--logo, --style dots, --wifi, --vcard)
si --export skill          # the si skill for your AI agent (Claude Code: /plugin install si@simple-ideas)
si mcp                     # MCP tools for agents: check, rules, new, build, qr, screenshot
si new-canvas <name>       # scaffold an animated canvas
si manual --slides         # this manual
si rules                   # the full format reference
si migrate <src>           # an AI migration prompt for an existing deck
```

---

## Share it

- **Post it:** `si publish talk.md --to linkedin` turns the deck into a
  document post, its PDF swipeable in the feed; an article goes to LinkedIn as a
  link, or to dev.to or Hashnode. Always a dry run first; `--yes` sends.
- **Get reviews:** `si serve . --review` prints a keyed link for people on
  your network: they leave comments on slides, sections or one paragraph,
  bullet or figure (click it to pin the note), and the Studio's
  Review panel shows them live, to reply and resolve; `--https` puts the
  link on HTTPS. **History** in the Studio
  shows what changed, commits, and restores earlier versions (git).
- **Film it:** `--mp4` on `si code`, on a mind map build (in 3D when the map
  is) and on a deck build (every slide and step, `--dwell 2` for a teaser)
  writes video that plays anywhere, with the transitions and morphs as they
  move, made with ffmpeg if it is installed.
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
sends an article to dev.to or Hashnode; it always shows you the requests
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
- `docs/publish.md` — LinkedIn, dev.to and Hashnode: logins, scopes, requests
- `docs/stage-diagrams.md` — the stage flow DSL
- `docs/authoring-canvases.md` — hand-authored animated canvases
- `docs/timer-animations.md` — the break-timer animation layer
- `docs/reference/developer-reference.md` — architecture and how to extend si

Notes: (0.5 min)
Thanks — now go build a deck.
