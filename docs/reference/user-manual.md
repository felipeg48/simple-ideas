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
- Or draw it in the Studio's **Motion view**: frame 1, then frames where you
  drag, resize and recolour parts; it writes the canvas and the step lines

Or let si move the slide itself: with `<!-- transition: morph -->` what two
slides in a row share (the same heading, picture or words, or a block named
`<!-- id: name -->` on both) glides from its old place and size to the new;
the rest fades. `morph 0.9s` sets the length.

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
SHOP_PASSWORD=shop-demo si demo run shop.md --captions
```

- A drawn cursor, click ripples, `note "…" at "Field"` callouts, blurred `$VAR` fields
- Each `Say:` line voiced first: the narration sets the pace
- `voice: kokoro:af_heart` (the default when installed), `system`, `elevenlabs`, `openai`
- `pronounce: si = ess eye`: the voice says it, the captions keep the word
- `<feature>.mp4` each, `all.mp4` with chapters, `chapters.txt`, `.srt`, README.html

Notes: (1 min)
The same script makes the videos. Every Say line is spoken first, so the
words set the pace and the steps fit between them. A cursor glides to each
button, notes point at the fields that matter, and passwords are blurred.
You get one video per feature, one with chapters for YouTube, captions, and
a page that lists it all. [CLICK]

Points:
- Kokoro is si's own voice: `si voice install kokoro`, one command; without it, the computer's voice
- Try a voice aloud: `si voice say "Hello" --voice kokoro:bf_emma`; `si voice list`
- Cloud keys: `si voice login openai` keeps the key in your own si.conf, never the project's
- `voice:` in si.conf sets the default voice
- Cloud voices send only the Say: text; lines are cached, re-runs re-voice only changes
- A failing step ends that clip and lands in the report
- Needs Chrome and ffmpeg: `si doctor`

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
- Example: `examples/webdemo/broker.md` (a tiny Go broker, `go run`)

---

## The CLI

```bash
si play <dir>              # serve with live reload (--dev: in-place hot reload)
si build <dir> --pdf       # self-contained HTML (+ PDF)
si build talk.md --handout # handouts/talk.pdf: slides only; =lines, =points, =notes add space or speaker text
si check <dir>             # validate; exit 1 on a problem
si demo test shop.md       # run a web demo script against a web app as a test (report.html, --junit)
si demo run shop.md        # the same, filmed and narrated: <feature>.mp4, all.mp4 with chapters, .srt (--captions)
si doctor                  # the optional programs (Chrome, ffmpeg, git…): found or not, how to install each
si new <name>              # scaffold a deck (the showcase, every rule)
si new <name> --type article   # a small starter: deck, article, guide, book, site, page, lab, demo, mindmap, mindmap3d, codeshow
si new <name> --type website --kind resume   # a website made for one job: resume, portfolio, landing
si publish talk.md --to linkedin   # a dry run of the post; --yes sends it (also devto, hashnode, ghost, wordpress)
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
  rehearse talk.md` shows where you ran long or rushed.
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
