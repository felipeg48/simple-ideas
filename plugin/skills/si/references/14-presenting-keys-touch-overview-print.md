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
  A deck with `[TERM …]` / `[BROWSER …]` cues (§11) is filmed on `si play`'s
  page, with its `setup:` and `start:` run first and `stop:` / `cleanup:`
  after (§14; `--no-live` skips them). Two builds may film the same deck at
  once: each narrates into a file of its own.
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
  Drag a window by its title bar, resize it from any edge or corner, **←** /
  **→** go back and forward in that window's page (dimmed when there is no
  page that way; the address box follows the links you click inside), **−** /
  **+** zoom that window's page alone (click the percentage for 100%),
  double-click the bar (or **□**) to maximise, **–** to minimise it to a pill in
  the bottom-left dock, **✕** to close; several windows at once, the one you
  touch comes to the front, and their places survive a live reload. Each page
  is shown through a loopback proxy si play starts for its origin, so consoles
  that refuse to be framed (X-Frame-Options, CSP `frame-ancestors`) still
  show. A window keeps that across sites: a link to another origin (the
  console's "Docs", a login page) and a redirect to one open through that
  origin's own proxy, the address box always shows the real address, and
  **←** / **→** go back and forward across the sites. A page's own script
  that sends it to another site stays in the window too: `location.assign`,
  `location.replace`, `location.href = …`, `window.open` and forms (in
  Chrome, Edge and other browsers with the Navigation API; elsewhere only
  `window.open` and GET forms are caught, since a page cannot replace
  `location`'s methods). A form that POSTs to another site (a login page)
  arrives there with its fields, in every browser: si sends it on through
  that site's proxy. Type the address in the box when a page still leaves.
  Keys typed in a window never drive the deck.

  ```markdown
  ---
  browser: RabbitMQ=http://localhost:15672, Grafana=http://localhost:3000
  ---
  ```

- **The presenter tools API, `window.siPlay`** (`si play` on a loopback
  address only, never in a built deck): what web demos (§27 `embed-terminal`,
  `embed-browser`, `slide`), a deck's `[TERM …]` / `[BROWSER …]` cues (§11)
  and auto-play use to drive the drawer, the browser windows and the deck by
  name. It is a contract (version 1; si's markup and CSS classes are not):
  a script of your own in the page, or a test, may call it too. Calls that
  open or type return a Promise; `output`, `frame`, `names` and `at` answer
  at once.

  | Call | What it does |
  |---|---|
  | `siPlay.terminal.open(name, {split})` | opens the drawer on the shell tab `name` (created when missing); `split: 'right'` or `'down'` puts it beside the tab in front; resolves when its shell has drawn its prompt |
  | `siPlay.terminal.run(name, command, {typing})` | runs a command in that tab (pasted; `typing: 55` types it, 55 ms a character) |
  | `siPlay.terminal.type(name, keys, {typing})` | sends keys as they are (`'\r'` Enter, `'\x03'` Ctrl+C) |
  | `siPlay.terminal.output(name)` | the tab's screen and scrollback as text (`''` for no such tab) |
  | `siPlay.terminal.state(s)` | the drawer: `'min'` (hidden, the shells run on), `'max'`, `'restore'`, `'close'` |
  | `siPlay.terminal.names()` | the open shell tabs |
  | `siPlay.browser.open(name, {x, y, w, h, zoom})` | a window with the `browser:` link `name` (or a URL), placed in CSS pixels, zoom `1.25` = 125 %; an open one comes to the front; resolves when its page has loaded |
  | `siPlay.browser.go(name, url)` | takes the open window `name` to `url` (relative to its page: `'#/queues'`); resolves when it has loaded (5 s at most) |
  | `siPlay.browser.close(name)` · `.min(name)` | closes (no name: every window) or minimises it; the keys go back to the deck |
  | `siPlay.browser.frame(name)` | the window's `<iframe>`, also `iframe[data-si-window="name"]` |
  | `siPlay.browser.names()` | the open windows |
  | `siPlay.slide(n, step)` · `.next()` · `.prev()` | moves the deck (slide from 1) like the keys; resolve with `siPlay.at()` |
  | `siPlay.at()` | `{slide, step, slides}` |
  | `siPlay.act(cues)` | runs cues as `[TERM …]` / `[BROWSER …]` do: `[{tool: 'term', name, cmd}, {tool: 'browser', name, o: {x, y, w, h, zoom}, url}, {tool: 'term', state: 'close'}]` |

- **The tools bar**: the play terminal's tools sit in a narrow bar of icons
  on its right edge, as in an editor: **Demos** (▷), **Runbook** (the book)
  and **Viewer** (the page). Demos and Runbook open as a side panel (one at a
  time; click the icon again to close it) that resizes by dragging its left
  edge; the width is remembered. **+** (new shell) and **✕** stay at the top.
  At the top left, **–** minimises the drawer (**`** brings it back, shells
  still running) and **□** maximises it to the whole window (or double-click
  the bar; **❐** restores; remembered). Drag the drawer's top edge to set its
  height (a fifth of the window up to all of it; remembered; maximise comes
  back to it).

- **Text size per tab**: **−** **100%** **+** in the top bar size the active
  tab only, a shell's text or a Viewer's tree, file and editor, never the
  other tabs (click the percentage for 100%); **⌘=** / **⌘-** / **⌘0**
  (Ctrl on Windows and Linux) do the same while you are anywhere in the
  drawer, instead of zooming the page (Chrome, Edge and Firefox let the page
  keep these keys; a browser that zooms anyway still has **−** / **+**). The
  size you used last is where the next tab of that kind starts.

- **Shells and Viewers survive a reload.** Saving the deck reloads the page;
  the drawer comes back as it was: open or closed, the same tabs in the same
  order, the active one, each shell with its recent screen (the last 256 KB
  of its output) and still running (a `sleep 600` keeps sleeping), each Viewer
  on its folder and file, its tree with the same folders open and scrolled
  where it was, in the editor with unsaved changes kept, Follow on or off,
  each tab's text size. A shell whose program ended is dropped; a
  shell no page has shown for 10 minutes is closed; all close when si play
  stops.

- **Viewer tabs** (the Viewer icon in the tools bar): each click opens
  a new tab beside the shells, a tree of a folder and the file you pick
  (read-only until **Edit**), so several files can be open at once (the tab shows `Viewer · name`).
  A Viewer opens on the last folder used (first time: the deck's folder);
  type a path (`~` works) and Enter, **↑** for the parent, double-click a
  folder to make it the top, **⌂ Shell** for the folder the last shell you
  used is in now. Files show highlighted with line numbers (the first MB;
  pictures shown; binary files named, not shown); **Copy path** copies it.
  Runbook and Demos Run buttons still type into the last shell.
  **Edit** turns a text file (up to 1 MB) into an editor: syntax colours by
  file name, line numbers, bracket matching, ⌘F search, undo, Tab indents,
  **Save** or ⌘S (refused when the file changed on disk since you opened
  it: **Reload** or **Overwrite**), **Revert**, **Done** for the read-only
  view again. Unsaved changes put a dot in the tab (`Viewer · app.yaml ●`);
  closing the tab or opening another file asks **Save**, **Discard** or
  **Keep editing**. A save never reloads the deck page. Colours for JS, TS,
  JSX, TSX, JSON (with errors marked), Markdown, CSS, HTML, XML, YAML, shell
  (`.sh`, `.zsh`, `.env`, Makefile), Python, Go, Java, Kotlin, Scala,
  Groovy, Rust, C, C++, C#, Swift, Ruby, Perl, R, Lua, SQL, TOML,
  INI/properties/`.conf` (`rabbitmq.conf`), Erlang (`.erl`,
  `advanced.config`), Dockerfile, nginx, PowerShell, protobuf, CMake and
  diff. It colours and checks JSON; it is not a language server (no
  completion or go-to-definition). `.conf`, INI, `.properties` and
  `.cfg` files colour their values in both views: keys, `[sections]`,
  numbers, sizes and durations (`512MB`, `30s`), booleans
  (`true`/`off`/`yes`), quoted strings, paths and URLs, `#` and `;`
  comments.
  The tree has a toolbar (and a right-click menu on its rows): **New file**
  and **New folder** (in the folder you picked, or the top; a new file opens
  in the editor), **Rename** (F2, in place), **Delete** (Delete key; a bar in
  the pane names what goes, a folder with the number of items in it; a link
  is deleted, never what it points to), **Copy path** and **Reveal path**. A
  name already taken is refused. **Follow** (read-only view, off by default)
  shows the file as it changes on disk, once a second, scrolled to the end
  when you were at the end, like `tail -f`; a file over 1 MB shows its last
  MB, numbered from its real line (the header says `lines 48210–71034`). It
  pauses while the tab is hidden. While you edit, a change on disk
  shows a notice (**Reload** or **Keep editing**), never a refresh under
  you. `.log` files never live-reload the deck.
- **Split view, quick open, search, spotlight** (the workbench, v3.9.0).
  **◫** in the top bar (or right-click a tab: **Split right**, **Split
  down**) shows two tabs at once: the active one and the next (a new shell
  when there is only one). **Shift-click ◫** (or right-click a tab: **Show
  in a new pane**, **Add a pane**) adds a pane, up to four: three are one
  big pane and two beside it (or under it, **Stacked**), four a 2×2 grid.
  Each pane shows one tab; clicking a tab shows it in the focused pane (the
  one with the blue edge; click inside a pane to focus it), dragging a tab
  onto a pane puts it there, the right-click menu has **Show in the other
  pane** (with more panes, **Show in pane 3**), **Swap the panes** (**Rotate
  the panes**), **Stacked** / **Side by side**, **Close this pane** and
  **Close the split**; **◫** again closes it. Drag the dividers to size the
  panes (shells refit as you drag). A narrow drawer (a phone) stacks them.
  The split, its sizes and which tab is where survive a reload.
  **⌘P** (Ctrl+P; in a shell on Linux and Windows Ctrl+P stays the shell's)
  or the **Go to file** icon: type letters of a file's path (`chkgo` finds
  `src/orders/checkout.go`) in the active Viewer's folder (from a shell: the
  deck's folder); **Enter** opens it in the active Viewer (from a shell: a
  new one), **⌘Enter** in a new Viewer. **⌘⇧F** or the **Search** icon:
  search the text of the files in the same folder (**Aa** match case, **.\***
  a regular expression); results grouped by file with the matches marked;
  click one to open the file in a Viewer at that line, marked. Results come
  in as they are found; **Stop** (or Esc) ends a long search and keeps what
  came. Both skip
  `.git`, `node_modules`, `vendor`, `dist`, `build` and dot-folders (start
  the query with `.` to look in dot-folders); search reads text files up to 1 MB and
  shows the first 1000 matches. **Spotlight**: in the read-only view, click a
  line number to light that line for the audience (the rest dims),
  Shift-click another to light the lines between, the same number again or
  **Esc** to clear; in the editor, select lines and press **Spotlight**
  (**Esc** clears). On screen only: nothing is saved.

  ```text
  shell 1 │ Viewer · checkout.go                ◫ +
  $ go test ./orders │ 75  // the order is paid
  ok  orders 0.4s    │ 76  // so we publish it
                     │ 77  func Checkout(o Order) error {   ← lit
  ```

  ```text
  Viewer · rabbitmq.conf ●     Save  Revert  Done  Copy path
  1  # rabbit
  2  listeners.tcp.default = 5672
  3  vm_memory_high_watermark.absolute = 512MB

  Viewer · app.log      [Follow]  Edit  Copy path
  87  2026-11-13T10:01:00 WARN appended 87
  ```

- **Demos** (front matter `demos: ../demos/`): demo scripts, plain markdown, in a
  second side panel of the play terminal (the Demos icon, one panel open at a time).
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
  exits 1. `si build` does not run them (a built deck is a file), except
  to film the deck's `[TERM …]` / `[BROWSER …]` cues (§11): `si build
  talk.md --mp4` of a deck with cues runs `setup:` and `start:` (in order,
  with their waits) before it films, so the cues show the live system, then
  `stop:`, the programs' end and `cleanup:` (always, a failure too), as
  `si play` would; `--no-live` films without them. A course build filming a
  stale `kind=video` deck does the same (`--no-live` and `--trust` pass on).
  `si check` checks each line and the pin of each source. A web demo script (§27) has its
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
  (with the pictures its `files:` key lists, each with its own fingerprint;
  `gist:ID@REV/talk.md` brings the whole gist). The same trust rules hold:
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
