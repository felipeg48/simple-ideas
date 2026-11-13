# 14. Presenting: keys, touch, overview, print

| Key | Action |
|---|---|
| → ↓ Space PgDn Enter | next step, then next slide |
| ← ↑ Shift+Space PgUp | previous step or slide |
| Home / End | first / last slide |
| `o` | overview of every slide: one column per section (`#` slide) or included module, headed by its title; a deck with neither gets a plain grid. ← → ↑ ↓ move, Enter or click opens, Esc closes |
| `p` or `Shift+S` | teleprompter window |
| `t` | break timer HUD: pick minutes, countdown to project during a break |
| `/` | find a slide: search titles and text, or type a number to jump |
| `a` | draw: a pen layer over the slide (drag, or hold ⌥ Option / Alt and move: the cursor draws, no click; si.conf `draw.hold` changes the key) |
| `d` | light / dark (remembered) |
| `f` | full screen |
| `v` | auto-play: the deck reads its `Notes:` aloud and advances at each `[CLICK]`; again pauses, Esc stops (§11) |
| `?` | key help |
| swipe left / right | next / previous on touch screens |

- **Break timer** (`t`): opens a full-screen HUD that floats over the deck.
  Number keys pick a preset and start the countdown; `Space` pauses and resumes;
  `t` or `Esc` closes it. At zero the digits turn critical and flash. While it is
  open the deck does not advance, so it is safe to leave projected during a
  break. It needs no network and works from `file://`. The presets default to
  `5,10,15,20,30,45,60` minutes; override them in `si.conf` with `timer.time` and
  `timer.unit` (minutes or seconds). The countdown shows as digits or, with
  `timer.face: analog`, an analog sport-clock dial — toggle it live in the HUD.
  A `timer.animation` runs a themed animation layer behind the translucent HUD
  (SLIDE │ ANIMATION │ TIMER) — built-ins `waves`, `orbit`, `bokeh`, or your own;
  list several (comma-separated) for a toggle (a **None** option is always
  offered). Once the countdown starts the HUD hides its presets and toggles so
  only the clock and the animation show — `Space` still pauses; any other key or
  a click brings the controls back. See §16 and `docs/timer-animations.md`.

- **Find a slide** (`/`): opens a search palette over the deck. Type to search
  slide titles and body text — results list the slide number, title and the
  matched snippet, ranked title-matches first; `↑`/`↓` move the selection and
  `Enter` (or a click) jumps. Type a **number** to go straight to that slide
  (e.g. `/12` → slide 12). `Esc` closes. Disable with `feature.find: off`.
  Below the slides it also lists the **other pages** of the folder, matched
  loosely (the letters in order, not necessarily together: `idx` or `inde`
  finds `index.html`, `lab3` finds `lab-03-quorum.html`): the index, the other
  decks and labs, their PDFs (`--pdf`), any other `.html` already in the
  output folder, and the `links.*` of `si.conf`. `Enter` opens the page.
  `si build` writes the list into each deck (names and titles only, so it works
  from `file://`); `si play` reads the folder on every request.

- **Draw** (`a`): a transient pen over the current slide, for circling or
  underlining as you talk. Drag to draw (mouse, touch, or pen — pen pressure
  varies the stroke), or hold ⌥ Option (Alt) and move: the cursor draws where
  it goes, no click, so a trackpad draws with one finger (`draw.hold` in
  si.conf picks another key: shift, ctrl, meta, or off); the toolbar picks the colour (accent, red, green, ink) and
  the brush size (thin / medium / thick); `c` cycles colour, `e` erases strokes,
  `z` undoes the last, `x` clears all, `Esc` turns it off. Strokes use the theme
  colours and clear when you change slides; they are not saved and do not print.

- The URL hash is the position (`#7`, `#7.2`); reloads keep it, so live
  reload in `si play` stays on your slide.
- A progress bar and a slide counter (with the step) sit at the bottom; the
  controls appear when the mouse moves.
- When a slide is too tall for the window its canvases, diagrams and images
  shrink first (down to about 60 %) so a figure plus bullets clears the
  footer, then the fluid type shrinks a little, then the slide scrolls;
  nothing is cut off. The figure cap itself is the `--si-figure-max` token
  (default `60cqb`; the `center` layout adds `6cqb`, columns `14cqb`), so a
  theme with a taller footer band can lower it for every slide:

  ```css
  :root { --si-figure-max: 52cqb; }
  ```
- **Print** (browser print, or print to PDF): one slide per page at the deck
  `aspect`, every step revealed, canvases in their final state. `si build --pdf`
  writes a `<name>.pdf` next to each deck using headless Chrome (needs Chrome or
  Chromium installed; without it, HTML is still written).
- **Handouts**: `si build talk.md --handout` writes `handouts/<name>.pdf` for
  the audience: a header (title, author, date) and every slide in its final
  state, eight to a page. By default a handout carries **no speaker text**;
  ask for it by name:

  ```bash
  si build talk.md --handout              # slides only (also: --handout=slides)
  si build talk.md --handout=lines        # each slide with ruled lines for notes
  si build talk.md --handout=points       # each slide with its Points:
  si build talk.md --handout=notes        # each slide with its Notes: script
  ```

  Front matter: `handout:` sets a deck's layout for a plain `--handout`
  (`slides`, `lines`, `points` or `notes`); `handout-url: https://…` prints a
  QR code to that address in the header, so the reader can open the live
  deck. si.conf `handout.paper: letter` changes the paper (A4 by default).
  The index tile of a deck gets a HANDOUT download badge. Needs Chrome, like
  `--pdf`.
- **Video**: `si build talk.md --mp4` films the deck to `<name>.mp4` (1920×1080,
  H.264): every slide and every build step as the audience sees it, each held
  for its share of the slide's `(N min)` (else 3 s), then the slide transition,
  the build step and the canvas morph filmed as they move (30 fps). `--dwell 2`
  holds every view 2 s instead, for a short teaser:

  ```bash
  si build talk.md --mp4 --dwell 2   # talk.mp4: a 2 s look at every slide and step
  ```

  With a recording from the teleprompter's **● Record** button, `--narration`
  holds every view exactly as long as you spoke on it and puts your voice
  under the video (`latest`: the newest recording of that deck; or a
  recording's folder). A view you never showed gets the usual time, and si
  warns when the deck changed since the recording:

  ```bash
  si build talk.md --narration latest   # talk.mp4: the talk, paced and voiced as you gave it
  si rehearse talk.md                   # per slide: planned, spoken, and your words per minute
  ```

  `si rehearse` reads the same recording: each slide's `(N min)` against the
  time you spent on it (`← long` past 1.25×, `← fast` under 0.75×) and the
  words per minute its `Notes:` script implies at that pace. Recordings stay
  on your computer; keep `.si/` out of git.

  No recording? `--voice` narrates the video from the `Notes:` themselves,
  the way auto-play reads them (§11): each view stays while its parts are
  said, a view with nothing to say its `(N min)` share or `--dwell` (needs
  ffmpeg):

  ```bash
  si build talk.md --mp4 --voice kokoro:af_heart   # talk.mp4, narrated by the deck's own script
  ```

  It needs headless Chrome and ffmpeg (`brew install ffmpeg`). si runs the page
  slowed down while it films, so the motion is smooth whatever the machine;
  `--dissolve` films still views that dissolve into each other in 0.4 s
  instead (faster, and what si falls back to when filming the motion fails).
- **Open on a phone**: run `si play <dir> --addr 0.0.0.0:7070` to serve on your
  network; `si play` then prints a LAN URL and a QR code to scan. (The default
  address is loopback-only, so it prints the `--addr` hint instead.)

- **Play terminal** (`` ` ``, backtick): a drawer of real shells on your machine
  for live demos — play-only and loopback-only, never shipped in a built deck.
  Full reference: `docs/terminal.md`.

- **Runbook** (front matter `runbook: runbook.json`): a command cheat-sheet in a
  right drawer of the play terminal. The JSON is a list of categories, each with
  entries (`label`, `description`, `command`, `target: sql|bash`):

  ```json
  { "categories": [ { "name": "Database", "entries": [
      { "label": "Sessions", "description": "who is connected",
        "command": "SELECT pid, state FROM pg_stat_activity;", "target": "sql" } ] } ] }
  ```

  Each command is syntax-highlighted (by `target`) so you can read it at a glance;
  there's a filter box. Every entry has **Copy** (puts the command at the shell
  prompt without running it, and on the clipboard) and **Run** (the same, then
  Enter); clicking the entry itself is Copy, Shift-click Run. A command waits
  until the shell has drawn its prompt, and is sent as a paste where the shell
  supports it (zsh, bash 5.1+), so a multi-line command shows whole and runs only
  on Enter. Like the terminal, it
  appears only in `si play` on a loopback address — a built deck never loads it, so
  rules #2/#7 still hold for shipped HTML. `si check` validates the path and JSON.

  A command may hold **placeholders**: `{{name}}` or `{{name:default}}` (name
  `[A-Za-z_][A-Za-z0-9_-]*`, default any text without `}`). The drawer asks for a
  value for each distinct name (first occurrence and its default win), then Copy
  or Run the command with every occurrence replaced; Esc cancels and types nothing.
  Placeholders are highlighted in the command.

- **Browser windows** (key **w**, or **Window** in the bar; `si play` on a
  loopback address only, never in a built deck; the bar also gets
  **Terminal** for the drawer): web pages in floating windows over the slides, for a
  broker's console or a dashboard on a wide screen. **w** opens a picker with
  the deck's `browser:` links, the recent addresses and a box to type one.
  Drag a window by its title bar, resize it from any edge or corner, **−** /
  **+** zoom that window's page alone (click the percentage for 100%),
  double-click the bar (or **□**) to maximise, **–** to minimise it to a pill in
  the bottom-left dock, **✕** to close; several windows at once, the one you
  touch comes to the front, and their places survive a live reload. Each page
  is shown through a loopback proxy si play starts for its origin, so consoles
  that refuse to be framed (X-Frame-Options, CSP `frame-ancestors`) still
  show. Keys typed in a window never drive the deck.

  ```markdown
  ---
  browser: RabbitMQ=http://localhost:15672, Grafana=http://localhost:3000
  ---
  ```

- **Files** (`▤ Files` in the play terminal's bar): a read-only tree of a
  folder beside the shells, for showing what a file says. It opens on the
  deck's folder; type a path (`~` works) and Enter, **↑** for the parent,
  double-click a folder to make it the top, **⌂ Shell** for the folder the
  active shell is in now. Click a file to read it, highlighted, with line
  numbers (the first MB; pictures shown; binary files named, not shown);
  **Copy path** copies it. The folder and the open panel are remembered.

- **Demos** (front matter `demos: ../demos/`): demo scripts, plain markdown, in a
  second side panel of the play terminal (`▶ Demos`, one panel open at a time).
  A drop-down lists the demos by their `# ` title (else the file name). The panel
  opens on **Commands only**: the headings that hold commands and the runnable
  blocks, so `> Say:` lines and notes stay in the speaker's own window; **Full
  script** shows the whole script (headings, prose, tables, `>` quotes), and the
  choice is remembered. Every
  ```` ```bash ```` / `sh` / `shell` / `zsh` / `console` / `sql` / `powershell`
  block gets **Copy** and **Run**, as in the runbook. Blocks in any other
  language (```` ```text ```` for expected output) stay plain. The value is a
  comma-separated list of `.md` files or folders, relative to the deck; a folder
  gives its `*.md` except `README.md`, `index.md` and `_`-prefixed files. Like
  the runbook: play and loopback only, never in a built deck; `si check`
  reports a missing path. The last demo picked is remembered.

  ```markdown
  ---
  title: Session 1
  demos: ../demos/
  ---
  ```

  ```json
  { "label": "Pods", "command": "kubectl -n {{ns:messaging}} get pods {{pod}}", "target": "bash" }
  ```

  Not placeholders, typed verbatim: `{{.Server.Version}}`, `{{json .}}`,
  `${HOME}`, `{print $1}`, `{a: .b}`, and the Go-template keywords `{{end}}`,
  `{{else}}`, `{{break}}`, `{{continue}}`, `{{nil}}`, `{{true}}`, `{{false}}`. The file may also carry a top-level `name`
  (the drawer title) and `description`.

### A talk that sets itself up, from anywhere

A talk that shows something live (a broker, a database, a cluster) can set it
up itself. Front matter `setup:`, `start:`, `stop:` and `cleanup:` may each
repeat; `setup:` and `start:` run **in the order written**:

```markdown
---
title: The orders broker, live
setup:   mkdir -p .live && echo 3 > .live/orders     # must finish, exit 0
start:   sh broker.sh as broker · wait for "ready"    # keeps running while you present
setup:   echo 4 > .live/orders                        # after the broker is up
stop:    echo "draining" > .live/state                # when si play quits, the programs still running
cleanup: rm -rf .live                                 # last, after the programs are stopped
---
```

| Key | When | Notes |
|---|---|---|
| `setup:` | before the first page, in order with `start:` | must exit 0 (within 10 minutes; `· within 30m` for longer); runs in the deck's folder (`· in <dir>` for another) |
| `start:` | before the first page, in order with `setup:` | a program kept running while si serves the talk: `as NAME`, `· wait for "text"`, `· wait for port 5672`, `· wait 3s`, `· within 60s`, as in a web demo (§27); its output goes to `<name>.log` in `$SI_RUN_LOGS` |
| `stop:` | when `si play` quits (Ctrl-C too), while the programs still run | |
| `cleanup:` | last, after si has stopped every program (SIGTERM, then SIGKILL after 5 s, the whole process group) | |

- **When.** `si play` and `si serve` run them before they serve the first
  page; meanwhile every page is a **getting ready** page that shows each step
  (waiting, running, ✓, ✗) and opens the talk once they are done. A failed step
  is shown there and in the terminal; si then runs `stop:` and `cleanup:` and
  exits 1. `si build` never runs them (a built deck is a file); `si check`
  checks each line and the pin of each source. A web demo script (§27) has its
  own meaning for these keys.
- **The programs.** The play terminal drawer's shells have `$SI_RUN_LOGS`, the
  folder of the programs' logs (`tail -f $SI_RUN_LOGS/broker.log`); the
  programs run on this machine, so the drawer, the runbook's commands and the
  talk's own `code:` reach them as they would any local program.
- **Sources.** A line may name a script from elsewhere instead of a command,
  with its arguments after it: `setup: github:acme/ops@v1.4/scripts/install-broker.sh --version 3`.
  The sources, their pins and the trust question are those of a web demo (§27,
  Scripts from anywhere): asked before running, once per pin, never silently.
- **A talk from a repo or a URL.** `si play`, `si serve` and `si build` take
  `github:owner/repo@REF/talks/talk.md` (or a folder, `…/talks/`): the repo is
  fetched once at that tag or commit into `~/.si/cache/repos`, its layout kept
  (a deck's `../images/…` still resolve), and served from there. A plain URL
  takes its fingerprint: `si play "https://example.com/talk.md · sha256 9f2a…"`
  (one file; pictures beside it are not fetched). The same trust rules hold:
  the first time, si shows where it comes from, its size and first lines, and
  asks; `--trust` says yes for one run (CI). A talk from a repo that sets
  nothing up just plays.

```bash
si play examples/live/talk.md                    # sets up, presents, cleans up on Ctrl-C
si play github:acme/talks@v2.1/orders/talk.md    # asked once for v2.1
si build github:acme/talks@v2.1/orders/ --trust  # CI: build a pinned talk
```

Example: `examples/live/talk.md` (a stand-in broker, `broker.sh`, started
before the title slide and stopped after the last).

### Studio (`si --web`)

`si --web [dir] [--addr 127.0.0.1:7070] [--no-open]` opens a browser editor for
the decks in `dir`: file tree, markdown editor, live preview and the `si check`
report. It listens on a loopback address only and is never part of a built deck.
See `docs/studio.md`.

```bash
si --web examples --no-open
```

### In the terminal

`si play talk.md --terminal` presents the deck in the terminal itself: no
browser, over SSH, in tmux. Every rule above draws as text and colour from the
same markdown:

- **The theme's colours** (its dark values, or light with `mode: light`): accent,
  muted text, the `==highlight==` band, card colours.
- **Titles of title, section and closing slides** in block letters made of
  half-block characters (falling back to bold text where a letter has no glyph
  or a word does not fit); the big number of a `big-number` slide too.
- **Builds** (`<!-- step -->`, `<!-- steps -->`) reveal one at a time; going back
  to a slide shows it whole.
- **Layouts:** columns (`<!-- column -->`, two-col and split with text left and
  media right) side by side when the terminal is 60 columns or wider, stacked
  below that; `center`, `quote` and `big-number` centred.
- **Code** highlighted with si's own highlighter, in a box with its title,
  `hl=` lines, `linenos` and callouts (❶); **tables** with borders and the
  column alignment; **cards**, **badges** and **logo lockups** as text.
- **Images** (PNG, JPEG, GIF) as real pictures in terminals that can show them
  (Kitty, Ghostty and WezTerm through the Kitty graphics protocol; iTerm2 through
  its inline images), and as half-block pixels everywhere else, inside tmux too
  (`SI_PICTURES=kitty|iterm|halfblock` overrides the guess); **stage diagrams**
  as boxes and arrows, the travelling dot moving along them.
- **Animated canvases** (`<!-- svg: -->`) drawn with characters: boxes, wires,
  text and dots from the SVG's shapes, in the state the deck has reached (the
  same words, `+w` / `-w` and build-step ops as in the browser). The helper
  attributes work (`data-show`, `data-hide`, `data-ok`, `data-alert`,
  `data-flow`, `data-packets`, `data-grow`) and so do the canvas's own CSS state
  rules for `opacity`, `display`, `fill` and `transform` (translate, scaleX).
  They move: packets travel along their wires, flows march, alerts pulse, and a
  part that moves between two states slides there when you press →. Other CSS
  (`offset-path`, counters, filters) shows the end state. SVG images, Mermaid
  diagrams and video show a labelled placeholder: the browser draws them.

Keys are the browser's: → Space `n` next, ← `p` back, `g` / `G` first and
last, `o` overview, `t` restart the timer, `?` keys, `q` quit. The deck reloads
when a file changes, and the slides reflow when the terminal is resized.

`si play talk.md --terminal --notes` in a second terminal (another tmux pane,
or the laptop screen while the projector shows the first) is the
teleprompter: it follows the presenter over a local socket and drives it back
(Space and ← move the slides from the notes). It shows the `Notes:` script with
the `[CLICK]` cues as badges (done ones ticked, the next one highlighted),
slide and total timers against the `(N min)` plan with the pace (on time,
ahead, behind), the clock, and the current and next slide. It scrolls by itself
(`a` on / off, `+` / `-` speed, ↑ ↓ by hand); Tab switches to the `Points:`.
It also follows a deck open in a browser: with `si play` (or `si serve`)
running, `--notes` listens to the server at `--addr` (default
`127.0.0.1:7070`) for the page of the same name, and Space / ← move the browser
deck. Without either it moves on its own.

`si play talk.md --terminal --dump 4 --size 100x30` prints slide 4 with every
build shown (ANSI colours) and exits: for scripts, READMEs and checks.
