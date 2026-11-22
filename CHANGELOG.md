# Changelog

What changed in each release of si, newest first. Every release, with its
binaries and the Mac app: https://github.com/felipeg48/simple-ideas/releases

- **v3.42.0:** edge labels follow their edges (fixes #27). In a `stage`
  diagram, a label on a slanted edge (a fan-out, a merge, a diagonal) is
  turned to the edge's angle, centred on it and set just above the line;
  it is always upright, so a label on an edge heading left is turned half
  round. Level and upright edges, and the middle of a bowed skip edge, keep
  the level label they had, byte for byte. `labels: flat` in the fence keeps
  every label level. New `dir: RL` runs a stage diagram right to left. The
  labels of infographic `flow` blocks turn the same way, beside the arrow's
  number. The gap widening for long labels and the sibling spacing are
  unchanged; still one static SVG, no script.

- **v3.41.0:** books for publishers (fixes #23). `si build <book> --pdf`
  now prints a paginated book: cover and title page unnumbered, contents
  with page numbers (two prints: the first measures, the second writes the
  numbers), a page per part, each chapter on a new page with a running title
  and page numbers, a PDF outline. New `--md` (the whole book and one file
  per chapter, images copied beside), `--docx` (Word, written by hand:
  styles, embedded pictures, a contents field Word updates on open, page
  numbers), `--rtf` and `--epub` (EPUB 3). One model of the book
  (`internal/bookexport`) feeds every format; guides and articles export too.

- **v3.40.0:** a code block and its `<N>` callout list are one card
  (issues #24, #25): a `<figure class="si-codecallout">` holds both, with one
  border, one background and one dashed line between code and list (no
  double border at the seam). The list takes the code's width, so the edges
  always line up, even when the code is narrower than a wide book column; a
  long explanation wraps instead of widening the card, a long code line
  scrolls inside it, and on paper the card stays on one page. It no longer
  needs `:has()`, so static viewers and PDFs show the same card, in slides,
  articles, labs, guides, books and tabs. Hovering a badge lights up only
  its own card's line.

- **v3.39.2:** code callouts, as AsciiDoc books write them (issue #22): an
  explanation that wraps onto a second line stays one item (before, the whole
  list showed as literal `<1>` text), `<.>` numbers itself in the code and
  in the list, and `/* <1> */` is a marker too. Markdown takes the same forms.

- **v3.39.1:** a scene's coordinates are rounded to two decimals, so a deck
  with a scene builds to the same bytes on Intel and Apple silicon (the
  layout's float sums differed in their last digits between the two).

- **v3.39.0:** scenes: objects that move (RULES §34). A ```` ```scene ````
  fence in a slide or a document names blocks (`box Shop`, `row A, B, C`,
  `column …`, `grid …`, `box X at 30,60`; laid out for you when unplaced,
  the blocks a split or a fan-out reaches stacked in a column) and then
  moves things between them, one build step per line: `parcel flies Shop
  → Customer · arc|straight`, `token hops A → B → C`, `Order morphs →
  Event circle → Message diamond`, `Order splits → Berlin, Lyon, Milan`,
  `parcels merge → Delivery`, `swap A ↔ B`, `tiles assemble 3×3`, `tiles
  scatter` / `Plan scatters`, `queue orders · Billing takes 3` and `Billing
  takes 2`, `Exchange fans out → Email, Stock`, `Warehouse opens → Pick,
  Pack, Ship`, `route Depot → Stop 1 → Stop 2`, `parcel drops into Locker
  12`; `->` and `<->` work too; `· slow`, `· fast`, `· 2s`. One inline SVG
  in theme tokens, played by `web/scene.js` (only on pages with a scene);
  every step's end is worked out in Go, so reduced motion, print, the
  overview, thumbnails, handouts, steps back and `#N.k` jumps show it at
  once. In documents a scene plays once in view, with ← → Play. `si build
  --png --svg` writes `talk-scene-1-step-1.png` per step and
  `talk-scene-1.png`; `si play --terminal` lists the blocks and steps; `si
  check` reports unknown verbs, names and options with the line in the
  fence. The showcase's order journey uses four scenes.
- **v3.38.1:** a new unlisted brand theme. It works by name like the other
  brand themes and stays out of the theme list, the docs and the gallery.

- **v3.38.0:** charts and motion, finished. `si build talk.md --png --svg`
  writes every ```` ```chart ```` fence of a deck or document as a picture,
  numbered in page order: `talk-chart-1.png` / `.svg` of its final step and,
  for a chart with more than one step, `talk-chart-1-step-2.png` / `.svg` per
  step (the SVG keeps real text; a 3D chart is PNG only). In a same-`id=`
  show, bars and columns now fold into a donut's or a pie's arcs, each bar
  bending round the ring into its own slice and unrolling back the other
  way, and a line turns into an area with its points in place; other pairs
  still cross-fade. Infographic blocks take `chart=bar3d|donut3d|funnel3d`,
  drawn by `--png` and `--pdf` (`--svg` leaves them out). Articles, guides
  and books play `animate:` blocks and `steps: focus|deal|reorder|carousel`
  lists once when scrolled into view (a reorder list is written in its new
  order; `tilt` stays slide-only); `si check` no longer warns about them;
  print, reduced motion and no script show the final state. A jump to a
  slide that is not the next one (a `#N` link, the overview, find, Home and
  End) plays the transition of the slide it lands on instead of a fade;
  morph and canvas transitions stay neighbour-only. The build animations
  live in one script shared by decks and documents (`web/build.js`). Fixed:
  `--svg` of an infographic or timeline lost the swatch of the widest legend
  item; a tipped-back 3D donut is drawn larger.

- **v3.37.0:** animated charts. A ```` ```chart ```` fence in any slide or
  document page draws a Markdown table (labels, then number columns) as
  `bar`, `column`, `line`, `area`, `donut`, `pie`, `progress`, or in CSS
  3D as `bar3d` (one row of solid bars, or a bar city with a row per
  column), `donut3d` (tipped back, ten layers deep) and `funnel3d`
  (stacked layers narrowing by value), on the theme's colours in light and
  dark. `steps` makes every number column a build step, its header shown
  big in the corner: bars grow and shrink, columns rise, arcs sweep, lines
  morph and the numbers count; `sort` keeps rows in order of value and
  they glide to their new rank; `race` is both, each step rescaled;
  `stacked`, `title=`, `unit=`, `explode="Label"` (a donut, pie or 3D
  donut slice lifts out on its own step). Charts with the same `id=` in a
  row are one show whose steps turn one kind into another (a cross-fade
  while the new one grows), and the same `id=` on the next slide carries on
  without the slide fade. A chart grows in when its slide or build step
  arrives (3D bars rise as the camera turns, funnel layers drop in), and in
  a document when scrolled into view, then plays its steps with ← → Play.
  Reduced motion, print, the overview, thumbnails, handouts and `#N.k`
  links show the step's final values. Infographic blocks take
  `chart=column|area|pie` too. `si check` reports non-numbers, unknown
  types and options, `steps` with one column, a bad `explode=`.
  `si play --terminal` draws the last column as text bars. RULES.md §33.
  Fixed: a slide's build-step count no longer resets after inline
  Markdown is rendered inside a fence.

- **v3.36.1:** six build animations in 3D, CSS 3D only (no library):
  `<!-- animate: tilt -->` (the heading, the marked block and the rest as
  layers at different depths; the camera tilts once as the slide arrives,
  then follows the pointer while it is shown), `<!-- steps: carousel -->`
  (a list's items as cards in a ring, one turn to the front per step),
  `<!-- animate: explode -->` on a list (one tilted plate per item, pulling
  apart), `<!-- animate: flap -->` (split-flap tiles that flip through the
  alphabet until the word lands), `<!-- animate: flyin -->` (words from far
  behind the slide, out of a blur) and `<!-- animate: orbit -->` on a list
  (the first item in the middle, the rest on a turning tilted ring that
  always faces you, while the slide is shown). The ring, plates and tiles
  keep their look at rest, so print, PDF and the overview show them as they
  end; reduced motion, thumbnails and jumps never play the motion.

- **v3.36.0:** motion. Ten slide transitions the page plays itself with the
  Web Animations API, no library: `cube`, `flip`, `iris` (from the click),
  `wipe`, `shatter`, `depth`, `blinds`, `portal` (into the block named with
  `<!-- id: -->` or the first picture), `glitch` and `stack`; going back
  plays the mirror, and `<!-- transition: cube 1.2s -->` sets a length.
  Build animations on the next block, run as its step arrives (or with the
  slide): `<!-- animate: type|words|highlight|count|draw|decode|confetti -->`;
  lists: `<!-- steps: focus -->`, `deal`, and `reorder` (one step that
  re-sorts by each item's last number, or `reorder 2 3 1`, with FLIP). Every
  block ends exactly as written; reduced motion, print, PDF, the overview,
  thumbnails, handouts, `#N.k` jumps and the teleprompter previews show the
  final state; real-motion `--mp4` films them. Documents show the final
  state (`si check` warns that `animate:` has no effect there). The Studio
  completes the new names.

- **v3.35.1:** Windows fixes found by running si on a Windows 11 machine.
  Every export through headless Chrome (`--pdf`, `--png`, `--svg`, `--mp4`,
  mind maps, timelines, infographics, `si code`) opened `file://C:/…`,
  which Chrome reads as a host named `C:` and loads nothing; they now open
  `file:///C:/…`. A deck's PDF path is absolute, so Chrome writes it where
  si looks for it. A local font in `fonts:` given as an absolute Windows path
  (`heading=C:\talk\fonts\Brand.woff2:600`) keeps its drive letter. si
  sets the console to UTF-8, so `—`, `✓` and the emoji look show as written.

- **v3.35.0:** more languages coloured, and a shadow for code windows.
  Groovy and Gradle (`groovy`, `gradle`, `jenkinsfile`; `build.gradle` and
  `Jenkinsfile` by file name, with Gradle and Jenkins words such as
  `implementation` and `stage` coloured), Kotlin scripts (`.kts`), Erlang
  (RabbitMQ's `advanced.config`, `enabled_plugins`), Makefile, nginx,
  Markdown and Windows batch. TypeScript has its own words now (`interface`,
  `readonly`, `keyof`…), Kotlin and Rust have their full sets (Rust lifetimes
  and `println!` macros), and annotations (`@Override`, `@Bean`,
  `@Component`) are coloured in Java, Kotlin, TypeScript, Groovy and Scala. A
  code window takes `shadow=lifted|soft|deep|none` (RULES §25); the default is
  now `lifted`, a clear drop shadow with a faint edge that shows in dark too.

- **v3.34.1:** a code show on a slide keeps its window at the tallest step's
  height (and, on a phone, room for the longest list of notes under it). A
  step that adds lines used to grow the window, and the centred slide moved
  its title and text with it; now a step moves only the code.

- **v3.34.0:** the control room. `si play` prints a keyed link to a **web
  monitor** at `/_si/monitor` (this machine only; `--remote` offers it to a
  tablet as the `monitor` role): uptime, CPU and memory, viewers **by role**
  (deck, teleprompter, remote teleprompter, draw pad, clicker, audience
  phones, review and class pages, monitors), the slide with time on it and
  the talk's time, the links with their QR codes, shells and the recent
  reloads, live over server-sent events. Under `--audience` it moderates:
  close, reopen or reset a poll, reveal a `· hidden` poll's results;
  approve, show on screen, mark answered or hide a question; hide a word
  from a cloud or clear it. Its states go into the `.si/audience/` save.
  The terminal monitor shows the same numbers (one source) with the roles
  and timers; `p` shows the network and audience links as QR codes, `w`
  opens the web monitor. The page needs the run's key, a loopback peer and
  Host, and the same origin for every change.
- **v3.33.1:** the terminal looks of v3.32.0 keep the words scripts wait
  for: `si play` and `si serve` say `serving <dir>` and `live reload on`,
  and a clean `si check` ends in `OK` (`1 deck OK · no warnings, no
  problems`), as the plain lines do; `si demo test` scripts that wait for
  them pass again on a terminal.
- **v3.33.0:** video edits, improved. A **moving blur**: `blur` and
  `pixelate` take `· to A → B over 2s` and `· path A B C` (eased or
  `· linear`) like the other moving callouts, and ffmpeg hides the pixels
  under the place of each moment (the place is an expression of time in
  the graph, the same curve the preview draws). A rectangle's place can
  carry a **size** (`· to 100,100 300x80 → 700,400 500x200`) and the size
  changes along the way, for a blur and for a box, circle, underline or
  spotlight. In the Studio's **Video view**, a moving callout's (or blur's)
  places are handles while paused, with its path drawn between them: drag
  one and that place is rewritten in the line, one undo away; the line
  under the cursor that is a `zoom to` or `pan to` shows its rectangle,
  which drags and resizes like a box. **Burned per-word captions scale**:
  the captions variant (`captions words`, `--with words`, and plain
  captions too) plays all its pictures as one ffmpeg input (an ffconcat
  list, a transparent picture in the gaps) instead of one input and one
  overlay per picture. On a 3-minute 1280×720 clip with 600 words, the
  ffmpeg pass went from 601 inputs and 33.2 s to 2 inputs and 10.1 s (the
  600 pictures, drawn by headless Chrome in 45.8 s, are unchanged). The
  look is the same pictures: in frames compared at seven times, the
  caption band matches pixel for pixel; only encoding noise in the moving
  picture differs. The ffmpeg si finds here has no libass (nor drawtext),
  so ASS karaoke was not an option.
- **v3.32.0:** terminal looks. `si play`, `si serve`, `si build` and `si
  check` print in colour on a terminal, in one of four looks: **ledger** (the
  default: a dim label column, the value, a green ✓ per line), **badges** (a
  coloured block names each line), **checklist** (emoji and plain words) and
  **steps** (a tree with timings and a bar per output). Pick one with `--view
  ledger|badges|checklist|steps`, `SI_VIEW`, or `ui.view:` in si.conf (the
  flag wins, then `SI_VIEW`, the project's si.conf, your own); a wrong name is
  an error that lists the four. Problems always show their file and line.
  `si play` on a terminal has a **live monitor**: `m` shows uptime, the si
  process's CPU and memory with sparklines or bars, the drawer's shells, the
  links (and the phone's with `--addr 0.0.0.0`), the pages open, the slide and
  step the deck last reported, and the recent reloads and warnings, redrawn
  every second on the alternate screen; `m` again goes back, `o` opens the
  browser, `q` quits, and the terminal is restored on quit and on Ctrl-C.
  Piped output, CI logs, `NO_COLOR=1` and `TERM=dumb` keep the plain lines,
  byte for byte. Standard-library ANSI only (`internal/ui`).
- **v3.31.0:** web demos and videos, finished. The Studio has a **Video
  view** for a `type: video` edit: the preview with its timeline strip above
  the text; a marker (or a row under the player) selects its line, and the
  cursor's line is marked in the player; pause and drag a callout on the
  frame (a bubble's tail, a text, key caps, a badge, a lower third, an arrow
  by either end, a box, circle, underline, spotlight, blur or pixelate, the
  rectangles resized by their corner) and its place is written into that
  line in source pixels, one undo away; ⌘B builds the file and plays the
  result in the panel. Callouts can **move**: `· to 100,200 → 400,300 over
  2s` or `· path A B C` (eased or `linear`) moves a bubble's tail, an
  arrow's head, a text or a rectangle. **Per-word captions**: `captions
  words` in an edit and `--with words` for `si demo run` mark the word being
  said, in the captions variant (a picture per word) and in the `.vtt` (a
  time before each word; si's player marks the word too). **`--with
  highlight`** works on an edit: the lines marked `· highlight`, else each
  chapter's first sentence, joined into `<out>.highlight.mp4`. The `si play`
  strip puts lines at the same time in lanes. Windows CI now runs a split
  terminal demo (`terminal split right|down`, `run … in 2`, `expect output …
  in 2`) under ConPTY and cmd, and the `si play` drawer's split view. Fixed
  on the way: headless Chrome now starts on Windows (its DevTools pipe is
  handed over as inherited handles, `--remote-debugging-io-pipes`), so web
  demos, `--mp4` filming and video edits run there; a file opened by quick
  open or search in the play Viewer gets its name on Windows.
- **v3.30.0:** a playable video player. A ```` ```video ```` fence (PR #21)
  draws a player in place (`mode=inline`), as a dialog over the dimmed and
  blurred page (`mode=modal`: Esc, ✕ or the backdrop closes it, focus stays
  inside and returns to the card) or as a poster that builds the player on
  the first click (`mode=facade`). The player has its own HUD: a scrub bar
  with the buffered range, hover time and chapter marks, time, volume,
  speed (1×–2×), captions, a chapter list, picture in picture and full
  screen; keys Space/K, ← →, J/L, M, F, C, Esc; tap and double-tap on
  phones; the HUD hides while playing. Modal players on a page share one
  dialog as a gallery (‹ ›, "2 / 9", `group=`). Captions and chapters come
  from `captions=` / `chapters=` or the files beside the video
  (`<video>.vtt`, `.chapters.txt`, si's own `chapters.txt`), so si's filmed
  videos get them for free; `![](x.mp4)` in documents and course video
  lessons use the same player. `si --export player` writes `si-player.css`
  and `si-player.js` for pages si did not build: any element with
  `data-video="clips/intro.mp4"` becomes a player, its image the poster.
  Also: a drawer reloaded twice in quick succession keeps its shells.
- **v3.29.0:** improvements. Chrome runs share one machine-wide limit
  (lock files in the temp folder, `flock` / `LockFileEx`), and a run that
  times out on a busy machine before writing its file gets one more try,
  with the wait in the error. Code-window gutters start at the right line
  (`start=N` on a fence, the first included line for `lines=`). The class
  view is in the Studio (a Class mode with the share link and QR) and in
  the terminal (`si play --terminal --class`). Paged timeline PDFs are
  vector PDFs with selectable text. `--svg` cuts embedded fonts to the
  characters shown and draws conic gradients, inset shadows and text
  shadows.
- **v3.28.0:** video edits from Markdown (`type: video`, RULES §32). A file
  beside a recording lists its finishing touches, one line each at the
  source's own times (`1:34`, `1:36–1:46`, `end`, `at said "words"` from the
  `.vtt` beside it, `at chapter "Name" + 2s`): an edit never moves another
  line's time. **Edits:** `cut`, `trim start|end`, `freeze`, `speed` (pitch
  kept; `· mute`, `· badge`), `replace with` (`· keep audio`), `insert
  clip|image|card`, `card`, `mute`, transitions `fade`, `dissolve`, `wipe`,
  `slide`; several sources joined (`· from`, `· to`, `join:`). **Callouts**
  drawn in the theme by headless Chrome: `title`, `text` (font, size, colour,
  box, shadow), `bubble` (`round`, `think`), `arrow`, `box`, `circle`,
  `underline`, `spotlight`, `keys`, `badge`, `lower-third`; entrances `pop`,
  `fade`, `slide-in`, `type`; `blur` and `pixelate` by ffmpeg; the camera
  `zoom to`, `pan to`, `zoom out`, eased. **Sound and takes:** `say` (the
  configured voice, the original ducked), `music`, `captions on` (the
  source's captions moved by the edits, plus the said lines), `refilm feature
  "Name"` (one feature of the demo script filmed again; alone, the rest of
  the video is stream-copied). `si check edit.md` reports every problem with
  its line; `si play edit.md` previews it live over the source with a
  timeline strip; `si build edit.md [--with captions,vertical]` writes it.
  Example: `examples/video/`.
- **v3.27.0:** code windows from files. `<!-- include: compose.yaml
  window=mac numbers hl=3 -->` and `<!-- code: … -->` take every code-window
  option (`window=`, `theme=`, `backdrop=`, `numbers`, `id=`, `step=`,
  `transition=`, the title defaulting to the file name) and draw exactly as
  the same fence would: shows of several files morph step to step, callouts
  follow, and Studio's Window view, `si code` exports and the terminal view
  know them. The language comes from the file name (`rabbitmq.conf`, `.env`,
  `Makefile`, `Containerfile`, …). `as=cat` shows a script in a terminal
  window under `$ cat <name>`. `si play` reloads when an included file
  changes, also outside the served folder.
- **v3.26.0:** eighteen improvements. **Courses and build:** posters are
  remade when their video's content changes (a fingerprint inside the JPEG);
  `nav: … numbered` numbers lessons 1.1, 1.2; the example course has a deck
  video lesson; a deck is re-filmed when an included file changes; `kind:zip`
  follows links all the way (every link inside resolves); `si build` of a
  page or guide inside a project with a `structure:` map builds the project
  with its map; SCORM progress over 4096 characters is packed to fit.
  **Demos, workbench, visuals:** `shell:` in a deck picks the drawer's shell
  (`clean` for the clean prompt); a framed page's `fetch`/XHR to another site
  goes through the proxy (the forwarder now needs this run's key);
  `[TERM]` cues take `· split`, `· max|min`, `· opacity`, `· typing`;
  without a voice, cues land where their words would be spoken; `files:`
  in JSON demo scripts; the Viewer follows `cd` on Windows (and from OSC 7);
  Studio icons for timelines and infographics; flow boxes take icons
  (`[db] Orders`) and `· dir=lr|tb`; `--svg` keeps gradients and shadows and
  embeds the fonts used; XMind 8 notes carry HTML; a 3D page's `--png`
  writes one picture per map as well.
- **v3.25.0:** steadier tests, webhook and class view follow-ups.
  **Headless Chrome:** every one-shot Chrome run (screenshots, PDFs,
  handouts, mind map and code exports, `--dump-dom`) goes through
  `cdp.Command`: at most max(2, CPUs/4) at once per process, a fresh
  `--user-data-dir` each (DevTools Chromes too), a 3-minute bound that starts
  once it has its slot (the MCP screenshot had none). The shared cause of the
  flakes: under load headless Chrome writes the screenshot, PDF or DOM and
  then dies in teardown ("Teardown watchdog expired", exit status 2); a run
  whose product is written now succeeds, and a Chrome that dies before gets
  one more try. Test harnesses poll for what they wait on instead of fixed
  sleeps. **Webhooks:** `webhook-base: https://…` (si.conf `course.base:`)
  puts event urls and xAPI activity ids under where the course is published;
  without it a page opened from disk says `urn:si:course:<id>[:<page>]`,
  never `file://`. A refused or offline send is tried again on the same page
  (2 s doubling to a minute, and when the browser is back online). Signed
  posts carry `X-Si-Timestamp`, and `X-Si-Signature` now covers
  `"<timestamp>.<body>"`: receivers refuse a timestamp over 5 minutes off (a
  change for existing receivers). The relay answers on the `si play --remote`
  listener too, behind its key. **Class view:** quiz questions are named by
  their titles from the quiz source; a Day picker opens earlier
  `.si/class/<date>.jsonl` days (and their CSV). The class view stays a
  browser page: none in the Studio or the terminal.
- **v3.24.0:** the play terminal drawer. **Long lines never overwrite**:
  the garbling seen in films made with si 3.17–3.22 came from the clean `sh`
  prompt's hidden marker, counted as 10 columns by bash (fixed in v3.23.0);
  now also a new shell starts at its tab's size, hidden or empty panes are
  never fitted, one observer refits on every drawer size change, and `run` /
  `type` wait until the shell knows its size (a committed e2e plays a
  minimise → reopen → long `curl` sequence). **Opacity**: a slider in the
  drawer bar (40–100 %, remembered) lets the slide show through while the
  text stays solid; `terminal.opacity: 85%` and `terminal.blur: 6px` in
  si.conf, `[TERM Ops · opacity 80%]`, `embed-terminal "Ops" · opacity 80%`,
  `siPlay.terminal.opacity(n)`.
- **v3.23.0:** si on Windows, and nine issues. **The terminal on Windows:**
  the play drawer and web-demo terminals run through ConPTY
  (`internal/termpty`; PowerShell, else cmd; a clean PowerShell prompt), with
  a Windows CI job; on Windows the Viewer reads `/` as the drive root and
  never deletes the served folder. **Web demos:** CSS selectors like `form
  .name input`; `files:` fetched beside a pinned script and `gist:ID@REV/file`
  targets; `vertical-fill: blur|theme|color #hex`; lower thirds drawn after
  the `--region` crop; card `theme:` takes a `.css` theme file; `shell: sh`
  long commands wrap right; `si build --mp4 --dissolve` says cues need the
  motion film. **Issues:** a page that reloads itself while loading no
  longer leaves demo steps "still loading" (#14); deck posters, PDFs and
  handouts are taken with reduced motion, so animated themes are never blank
  (#15); a film advances through `siPlay.next()`, and closing the drawer gives
  the keys back to the deck (#16); filming a deck with cues runs its
  `setup:`/`start:`/`stop:`/`cleanup:` (`--no-live` skips) (#17); a demo with
  `app:` a deck waits for the talk and runs its `prepare:` after the deck's
  setup (#18); `[BROWSER Name · at X,Y · size WxH · zoom N%]` and `[BROWSER
  Name: URL]` (#19); concurrent narrated builds of one deck never share a temp
  file (#20).
- **v3.22.0:** the live class view. `si serve course/ --class` (or `si
  play … --class`) collects the course's events at `/_si/class/event` (same
  origin, validated, no answers stored) into `.si/class/<date>.jsonl` and a
  live instructor dashboard: learners × lessons (done, quiz score and
  attempts, lab checks, video watched), who's stuck (a check failed twice,
  or no progress for 5–30 min), per-lesson and per-question summaries, a
  module filter and a CSV export. Open on this machine, or from elsewhere
  with the printed keyed link and QR code. Learners give a name or stay
  anonymous.
- **v3.21.0:** mind maps. `si mindmap export map.md --to xmind8` writes
  the XMind 8 format (and its import reads labels and sheet links back).
  3D maps on a phone or an upright tablet lay their branches down the screen
  seen from the front, with finger-sized tools, one-finger orbit, pinch and
  a tap to open a topic. `si build map.md --png --pdf` of a `view: 3d` map
  exports the 3D view (the first map with its linked maps); `--svg` stays
  2D (the 3D view is raster only).
- **v3.20.0:** five improvements. Top-to-bottom stage diagrams widen the
  gap between sibling boxes so their edge labels never overlap. Infographic
  line charts place numeric and date x values by their real value (round
  ticks). A `kind:zip` bundle also carries the built pages its outputs link
  to (index, labs) with their assets. A new `--structure` project checks
  clean: a bundle of a kind with no sources yet is a note, not a warning. A
  portrait infographic printed over several pages keeps its background on
  every page and never cuts a block.
- **v3.19.0:** course results to your team. `webhook: https://…` in
  `_course.md` sends `lesson.completed`, `quiz.submitted` (score, pass, per
  question right or wrong, never the answers), `lab.step.checked`,
  `demo.watched` and `course.completed` (schema version 1), as JSON, Slack,
  Teams, Discord or xAPI 1.0.3 statements, after a short notice
  (`webhook-consent: required` sends nothing before OK), queued while
  offline. **Labs that check themselves:** `<!-- check: cmd · expect "text"
  · exit 0 -->` on a lab step shows "Run this to check" in every build and a
  **✓ Check** button in `si play` that runs it (30 s, the page's own checks
  only) and sends a signed event. `/_si/relay` (loopback, or `--relay`)
  signs browser events with `X-Si-Signature` (HMAC-SHA256, secret only in
  the user's si.conf or `SI_WEBHOOK_SECRET`). `si demo test --webhook URL`
  posts the run summary. Spec of the next step, a live class view:
  `docs/superpowers/specs/2026-11-14-results-webhooks.md`.
- **v3.18.0:** the workbench, finished, and a privacy fix for filming.
  **A clean prompt when si films the drawer:** `si demo test` / `run` (an
  `app:` deck) and `si build --mp4` with `[TERM …]` cues start the drawer's
  shells with si's neutral prompt, the folder's name and `$` (zsh with its
  own `ZDOTDIR`, bash `--noprofile --rcfile`, sh `ENV`; `PATH` and the
  environment kept, history off), so no email, host or cloud account from
  your own prompt reaches a video. A live `si play` keeps your shell unless
  si.conf says `terminal.prompt: clean`. **A POST form to another site keeps
  its fields** in a browser window (a login page on another origin): the
  page's proxy hands it on, method and body, to that site's proxy (a 307),
  in every browser. **The terminal teleprompter** (`si play --terminal
  --notes`, and the terminal Studio's rehearse view) shows `[TERM …]` and
  `[BROWSER …]` cues as badges, not raw text; the Studio's rehearse view
  also draws its `[CLICK]` badges again.
- **v3.17.0:** the deck on stage (#13). Web demo steps drive `si play`'s
  own tools by name: `embed-terminal "Lab 1"` (then `run`, `type`, `press`,
  `expect output` go to that drawer tab; `· split right|down`;
  `min|max|restore|close`), `embed-browser "RabbitMQ"` (a `browser:` link or
  a URL, `· at X,Y · size WxH · zoom N%`, `close|min`), `click "…" in
  embed-browser "Name"`, and `slide N` / `next` / `prev`; an `app:` that is a
  deck starts `si play` itself. A public `window.siPlay` API (version 1) is
  the contract behind them. `Notes:` cues `[TERM Name: cmd]` and `[BROWSER
  Name]` act during auto-play and `si build --mp4 --voice`, show as
  teleprompter badges and are never read aloud. A filmed demo with
  `· kind=demo` is a learner Demo lesson.
- **v3.16.0:** three issues. **Video posters (#12):** a filmed deck writes
  `<deck>.poster.jpg` (its first slide, 1920×1080) and video lessons open on
  it instead of a black player; `· poster=path`, `poster:` in the deck or
  `· poster=off`; plain videos get their first frame through ffmpeg when
  `si build` runs; `si check` reports a missing poster. **Stage diagrams
  (#11):** the gap after each column widens to fit its longest edge label, so
  labels like `order.*.created` no longer cover arrows or boxes (a label past
  twice a box's width is set smaller). **Lab solutions (#10):** a failed
  include inside a `<!-- solution -->` block no longer adds false "no end"
  problems, and an unquoted `title=` with spaces gets a hint to quote it.
- **v3.15.0:** animated infographics. **Motion in the Markdown:**
  `animate: build` builds an infographic once, `animate: loop` over and over,
  `duration: 12s` sets its length; numbers count up, bars and progress grow,
  donuts sweep, line charts draw, steps light one by one, quotes fade; per
  block `· enter=fade|rise|grow|count` and `· at=3s`. Reduced motion,
  thumbnails and print show it finished. **Flow blocks:** `## The path ·
  flow` with `Client -> Load balancer -> API`, labels as `A -[HTTP]-> B`, one
  path per line, an arrow back allowed: boxes and arrows laid out by the
  stage fence's engine, left to right or top to bottom by the frame, arrows
  numbered, drawn in order, then packets travel along them. **Exact frames:**
  `aspect: phone` (9:19.5), `story`, `portrait` (4:5), `square`, `landscape`
  or `1080x1350`, fluid inside (container units, rows shared by what they
  hold). **GIF and MP4:** `si build x.md --gif --mp4 --png` films the page
  frame by frame from its own clock (exact, the same on every run): a 720 px
  GIF at 15 fps with one adaptive palette and only the changed pixels of each
  frame (loops for `loop`, plays once for `build`), a full-size MP4;
  `--fps`, `--scale`, `--size phone,square,...` for several sizes in one run;
  `si check` warns when the GIF would pass about 8 MB. **In slides:** an
  animated ```` ```infographic ```` fence plays when its slide becomes the
  current one (again on every visit), a block per click with steps, and
  `si build deck.md --mp4` films it building. **The Studio preview** (and any
  page, on hover) has ▶ Play / ↺ Replay. `si code --gif` uses the same
  smaller encoder. Example: `examples/infographic/broker-flow.md`.
- **v3.14.0:** the play workbench, six improvements. **The Viewer's tree
  survives a reload:** the folders you opened come back open and the tree
  scrolled where it was. **⌘= ⌘- ⌘0 anywhere in the drawer** (Ctrl+ off
  macOS) size the active tab and are cancelled before the browser zooms the
  page (Chrome, Edge and Firefox let a page keep them; − / + stay for any
  browser that does not). **Search streams:** results show as they are found,
  file by file, with **Stop** (or Esc) to end a long search and keep what
  came; still the first 1000 matches. **Up to four panes:** Shift-click ◫ (or
  a tab's **Show in a new pane** / **Add a pane**) adds a pane: three are one
  big pane and two beside or under it, four a 2×2 grid, two dividers to drag,
  **Close this pane**, **Rotate the panes**; all kept across a reload.
  **Browser windows catch script navigations:** a page that sends itself to
  another site with `location.assign`, `location.replace`, `location.href =`,
  `window.open` or a form stays in the window, through that site's proxy (via
  the Navigation API; without it only `window.open` and GET forms are
  caught; a POST form opens its address without the fields). **Follow on a
  file over 1 MB** numbers its last MB from the real line (`lines
  48210–71034`), counted once and then only the new bytes as the log grows.
  Also: keys typed fast into a shell can no longer arrive out of order. The
  Viewer and workbench browser checks are committed e2e tests
  (`viewer.e2e.mjs`, `workbench.e2e.mjs`) run in CI.
- **v3.13.0:** **Project structure.** A `structure:` block in a project's
  `si.conf` maps each kind of output to a folder: one folder per line,
  nested by indentation, names ending in `/`, then what goes there (`md/
  markdown`, `slides/ deck`, `pdf/ deck:pdf`, `html/ site`, `images/
  image`, `animations/ animation`, `videos/ video`, `dist/ html/:zip,
  deck:zip, course:scorm`). `si build` reads the sources wherever they sit
  in the project, renders each page into its kind's folder, copies
  pictures, animations (canvases too) and videos into theirs, makes the
  derived files the map names (as if `--pdf`, `--mp4`, `--handout`,
  `--scorm`, `--png`, `--svg` were given), copies a kind listed twice into
  both folders, and zips the bundles last (`<project>-html.zip`,
  `<project>-deck.zip`). Every link is rewritten from its page's new folder:
  pictures, videos, PDFs, other pages, index tiles and their PDF badges,
  find (`/`), guide, course and site navigation, `_si/`. `si new <name>
  --structure` writes the map and its folders, the deck in `md/`; `si check`
  checks every folder of sources and warns about unknown kinds or formats
  (with the line), empty bundles, kinds with no folder and sources in
  another kind's folder. No block: the build is byte for byte what it was.
  RULES.md §31, `examples/structure/`.
- **v3.12.0:** fixes #8, #9. **Courses whose pages are not written yet:**
  `si check` and `si build` on a `_course.md` (or a `_nav.md`) whose listed
  pages do not exist report `missing page …` and `lists no pages` instead of
  panicking; the messages no longer repeat the file name
  (`_course.md:7: missing page missing.md`). **Videos in pages:**
  `![Caption](clip.mp4)` (`.webm`, `.mov`, `.m4v`, `.ogv`) is a player on a
  slide and in a document, the title or alt text its caption, `poster=` in the
  `image` directive (which documents now honour too) its still; never inlined
  as a data URI. **Linked files are copied:** a link to a local file si does
  not render (a PDF, a zip, a CSV, a video) and the `src` of a raw `<video>`,
  `<source>` or `<audio>` are copied beside the built page in every media
  mode, for decks, documents, guides, courses and sites; a missing one fails
  `si check`. **Video lessons from a deck:** `- [Title](03/deck.md) ·
  kind=video · pdf` in `_course.md` shows the deck's narrated video, filmed by
  `si build` beside the deck when missing or older than it (`--voice`, else
  the deck's or si.conf's voice; `--no-film` skips it), with a **Download the
  slides (PDF)** button under the player; `· handout` adds the handout PDF,
  `· slides` a link to the interactive slides (built, in no menu). `si build
  --mp4 --voice` now also writes `<deck>.vtt` captions from the narration's
  words and timing, which the lesson's player shows.
- **v3.11.0:** three open items. **Auto-play resumes mid-part:** Pause (a
  click, a key, the pen) and Resume on the same slide and build step go on
  where the voice stopped, the voice file from its second and the browser's
  own voice from its sentence (it now speaks a part one sentence at a time),
  instead of starting the part over; another slide or step starts its part
  from the beginning. **Browser windows keep the proxy across sites:** a
  link to another origin and a redirect to one open through that origin's
  own loopback proxy, so consoles that send you to their docs or login page
  stay framed; the address box shows the real address on every site, and ←
  → go back and forward across them. **Web demos: split terminals:**
  `terminal split right` (or `down`) opens a second shell beside the first
  in the same window, both filmed; `focus 2` picks the pane `run`, `press`
  and `expect output` go to, `run "…" in 2` and `expect output "…" in 1`
  pick one for a step; `si check` follows the panes. The broker example
  sends a message from a pane under the broker. ConPTY (Windows) stays out:
  terminals there are `cmd` on pipes.
- **v3.10.0:** timelines and infographics, the four open items. Two new
  timeline layouts: **`layout: eras`**, one column per era (a coloured header
  with the era's name and dates, its milestones stacked under it, pictures as
  small medallions; the 30-milestone example fits one 16:9 frame), and
  **`layout: cards`**, a grid of picture cards on a thin dated rail in each
  era's colour; both landscape (a frame, one era per click) and portrait (a
  poster), every theme, light and dark, the paged PDF too. **`si build
  --svg`** writes `<name>.svg` of a timeline or an infographic: a vector
  picture of the frame as shown, its text as real text, pictures inside,
  theme colours resolved (soft glows and shadows left out), made from the
  page Chrome laid out. **Infographic text fits its block:** a block whose
  text does not fit gets a smaller text size step by step, down to 60%; `si
  check` warns when a block would need less. **The Studio** lists Timeline
  and Infographic in New… and the start gallery (14 starters), and
  **the terminal** (`si play --terminal`, `si --terminal`) shows a timeline
  as its eras and milestones and an infographic as its blocks, a stat's
  number in block letters.
- **v3.9.0:** the play workbench grows. **Split view:** ◫ (or right-click
  a tab: Split right / Split down) shows two tabs side by side or stacked,
  with a draggable divider (shells refit), tabs dragged onto a pane, swap,
  and the split kept across a reload. **⌘P** (Ctrl+P) quick open: fuzzy
  file names in the Viewer's folder, Enter opens in the active Viewer, ⌘Enter
  in a new one. **⌘⇧F** search in files: plain text, match case or regular
  expression, results grouped by file; a click opens the file at that line,
  marked. Both skip `.git`, `node_modules`, `vendor`, `dist`, `build` and
  dot-folders. **Spotlight:** click (Shift-click) line numbers in the
  read-only view, or select lines in the editor and press Spotlight, to
  light those lines for the audience; Esc clears.
- **v3.8.0:** a promo kit for web demos. `session: keep` keeps one browser
  (cookies, local storage) across a run's features; `header-offset:
  auto|<px>|none` keeps spotlights and scrolling clear of sticky headers and
  stops the closing spotlight from nudging the page; `si demo draft <url>`
  crawls a site (same origin, robots.txt, no form submits) and writes a first
  promo script from its own words; `intro:` / `outro:` cards with `logo:`,
  `music:` ducked under the voice, `Lower-third:` captions; `--with vertical`
  (1080×1920) and `--with highlight` (about 60 s) cuts; steps `… in frame
  <css>` reach into same-origin and cross-origin iframes. `si check` accepts
  `region:` in Markdown demo scripts.
- **v3.7.0:** courses, finished. `si build course/ --scorm` writes a SCORM
  1.2 package (one SCO per lesson, `--scorm=single` for one; status, quiz
  scores and progress reported to the LMS, still working with none);
  `--pdf` writes a learner workbook and `--instructor --pdf` an instructor
  guide; `si publish login skilljar` and `si publish course/ --to skilljar`
  (dry run unless `--yes`; sections, HTML, video and SCORM lessons, updated
  in place; files are fetched by SkillJar from `--base-url`). Studio's New
  lists courses. Learner quiz pages carry hashed answers only; a quiz is a
  self-check, not a secure exam.
- **v3.6.0:** the play terminal's shells and Viewers survive a live
  reload: saving the deck brings the drawer back as it was (open or closed,
  the same tabs, the active one), each shell still running with its recent
  screen replayed (the last 256 KB of output), each Viewer on its folder and
  file, in the editor with unsaved changes kept. A shell whose program ended
  is dropped; one no page has shown for 10 minutes is closed. **Text size per
  tab:** − 100% + in the top bar (or ⌘= ⌘- ⌘0 in the tab) size the active
  shell or Viewer alone; the last size is where the next tab starts. Drag the
  drawer's top edge to set its height (remembered). Browser windows get **←**
  and **→**: back and forward in the page, the address box following the
  links you click. `si demo`: a CSS selector keeps its spaces (`click
  .hm-eng.big .hm-open`), and a selector can start with a tag
  (`button.pay`, `nav > a`).
- **v3.5.0:** the play terminal's Viewer tree manages files: **New
  file**, **New folder**, **Rename** (F2) and **Delete** from a toolbar
  above the tree or a right-click menu, plus **Copy path** and **Reveal
  path**. A new file opens in the editor; a taken name is refused; Delete
  asks in the pane and names what goes (a folder with its item count); a
  link is deleted, never what it points to. `.conf`, INI, `.properties`
  and `.cfg` values are coloured in the read-only view and the editor
  (keys, sections, numbers, `30s`, `512MB`, booleans, strings, paths,
  URLs). **Follow** keeps the read-only view in step with a file that
  changes on disk, scrolled to the end like `tail -f` (the last MB of a big
  file); while you edit, a change on disk is a notice, never a refresh.
  `.log` files no longer live-reload the deck.
- **v3.4.0:** the play terminal's Viewer tabs edit: **Edit** opens the
  file in an editor with syntax colours for about 40 languages and formats
  (JS/TS, JSON with errors marked, YAML, Markdown, shell, Python, Go, Java,
  Rust, C/C++, SQL, TOML, `.conf`, Erlang, Dockerfile, nginx and more), ⌘F
  search, undo and **Save** (⌘S). A file changed on disk since you opened it
  is not overwritten unless you choose **Overwrite** (or **Reload**);
  unsaved changes show as a dot in the tab, and closing the tab asks first.
  Saving never reloads the deck page. Colouring and JSON checks, not a
  language server.
- **v3.3.5:** the play terminal drawer has **–** (minimise; ` brings it
  back) and **□** (maximise to the whole window, or double-click its bar;
  remembered) at its top left.
- **v3.3.4:** the play terminal's Demos, Runbook and Viewer are icons in a
  tools bar on its right edge, as in an editor, so the top bar keeps only the
  tabs, **+** and **✕**; the Demos and Runbook panels resize by dragging
  their left edge (the width is remembered).
- **v3.3.3:** the file viewer is a tab: **▤ Viewer** in the play terminal
  opens a new `Viewer` tab beside the shells each time, so several files can
  be open at once; Run buttons and **⌂ Shell** use the last shell.
- **v3.3.2:** a **Files** panel in the `si play` terminal: a read-only
  tree of a folder (the deck's, a typed path, or the active shell's with
  **⌂ Shell**) and a viewer that shows a file highlighted with line numbers,
  or a picture. Browser windows (w) resize from every edge and corner, and
  **− / +** zoom each window's page on its own. The `si play` bar gets
  **Terminal** and **Window** buttons beside Auto-play.
- **v3.3.1:** a landscape timeline's `--pdf` is readable: `pdfs/<name>.pdf`
  has several 16:9 pages in the same look (whole eras, about 12 milestones a
  page, title on each, closing line and sources on the last) and
  `pdfs/<name>-compact.pdf` is the whole timeline on one page, like the PNG.
  Before, the PDF printed only part of the timeline.
- **v3.3.0:** timelines and infographics, both landscape (16:9, a slide you
  present) or portrait (a tall poster). `type: timeline`: `## Era · color=`,
  `### date · title` milestones with a description and a captioned picture
  (local or https, inlined), `## … · sources`, `closing:`; layouts `snake`
  (fits 30 milestones on one 16:9 frame), `line`, `vertical`; one era per
  click; dates as written (a.C./BC, c., ranges) and checked for order; `si
  check` counts dates, pictures and eras and verifies `require: 20 dates, 15
  images`. `type: infographic`: every `## heading · kind` is a block of a
  bento grid (stat, compare, steps, `chart=bar|donut|line|progress` from a
  Markdown table, quote, icons, list, text, image, sources; `wide`, `tall`,
  `accent`, `span=`). Both: a fence for deck slides (```` ```timeline ````,
  ```` ```infographic ````), every theme in light and dark, `si build --png
  --pdf` (`--scale`), `si new --type timeline|infographic`, RULES §29 and §30.
- **v3.2.2:** browser windows in `si play`: **w** opens web pages (a
  broker's console, a dashboard) in floating windows over the slides; drag,
  resize, minimise to a dock, maximise, close, several at once, kept across a
  live reload. `browser: Name=URL, …` in the front matter lists the links;
  consoles that refuse framing show through a loopback proxy. Never in a
  built deck.
- **v3.2.1:** the ▶ Demos panel of the play terminal opens on **Commands
  only**: the headings that hold commands and their Copy / Run blocks, so
  `> Say:` lines and notes stay in the speaker's own window; **Full script**
  shows everything, and the choice is remembered.
- **v3.2.0:** courses. A folder with a `_course.md` is a course: modules of
  lessons whose kind comes from what they link to (a page is a reading, a
  deck slides that present themselves, a `.mp4` or a web demo a video,
  `type: lab` a lab, `type: demo` an instructor demo, `type: quiz` a quiz;
  `· kind=` overrides), with minutes (front matter or estimated) and
  prerequisites. `si build course/` writes the learner's website bundle (a
  course home with Start / Continue and module cards, one page per lesson,
  videos copied beside them, a completion page, shared scripts once; `--zip`
  as well; filmed web demo lessons re-filmed when stale unless `--no-film`);
  `--instructor` (and `si play`) the instructor's, with demo runbooks, the
  teleprompter, instructor notes and open lab solutions. The course shell
  follows the approved mockup: a module accordion with progress rings and
  minutes, lesson kind labels and done ticks, full-strength menu text with
  tinted hovers and accent bars, a top bar with the course's progress, and on
  a phone a Menu button and a bottom bar. Quizzes: `type: quiz` pages and
  ```` ```quiz ```` fences (`- [x]`, `> Explanation:`, `pass: 70%`), scored
  in the page. Progress (ticks, rings, Continue, completion, `certificate:
  on`) is kept in the learner's browser. Labs: `<!-- solution -->` … `<!--
  end -->` folds an answer for learners. `nav:` (left, right, top, bottom ·
  sticky · accordion, collapsed, numbered) also places the menu of guides,
  books and sites, and si.conf takes a default. `si new <name> --type
  course`, `si --export example --type course`, `si check` on courses;
  RULES.md §28, a Courses page and a live example course on the docs site.
- **v3.1.5:** documents: the "On this page" contents column shows its links
  at full strength (they were muted grey, hard to read); hovering one gives
  it a tinted row and an accent bar instead of only darkening the text, and
  the section you are reading is tinted too.
- **v3.1.4:** a `pronounce:` rule whose words contain the word itself
  (`VEX = vex`) no longer makes `si demo run` loop for ever before filming;
  each rule repeats at most 3 times and stops when nothing changes.
- **v3.1.3:** web demos: `wait for "…" · within 40s` waits longer than
  `timeout:`, also while filming (where a step otherwise gives up after 3 s);
  si's presenting demo waits that way for a deck reading its own notes.
- **v3.1.2:** web demo videos never show a list of files: a folder `app:`
  serves a plain page where it would list a folder, and a feature whose
  first page step is a `go` opens on a plain page, not the app's root. si's
  own demos show the talk's files with `ls` and `si play` in a terminal
  instead.
- **v3.1.1:** `si demo run` with a voice this computer cannot use (a script
  asking for Kokoro where it is not installed) speaks with the system voice
  and says so, instead of stopping, like auto-play and `si voice say`.
  Ctrl-C during a run always exits 130 after `stop:` and `cleanup:` (it could
  return as a normal end when the interrupted `prepare:` failed first). CI
  installs ffmpeg and espeak-ng for the demos job.
  Fixed: zooms are crisp (issue #6). `si demo run` films
  the page, its terminals and windows at 2 device pixels per CSS pixel
  (Chrome's window and the page: 2880×1800 frames for 1440×900) and scales
  every video back down to the viewport with lanczos; camera crops, region
  crops, transitions, title cards, captions and fast-forward badges all work
  on the 2× frames, so a 3× `zoom to` magnifies at most 1.5× instead of 3×.
  Videos and `screenshot` steps keep the viewport's size. `--scale 1` (or
  `scale: 1` in front matter) films at 1× as before. On the shop example:
  45 s instead of 42 s, 61 MB of temporary frames instead of 23 MB, same
  file sizes.
- **v3.1.0:** demos and talks that set themselves up, from anywhere
  (spec `docs/superpowers/specs/2026-11-12-demo-lifecycle.md`). Web demos:
  `start:`, `prepare:` (alias `step:`) run once, interleaved in the order
  written (`· within 5m` for a slow one); `setup:` may repeat (before each
  feature); `stop:` may repeat (programs still up); new `cleanup:` (after
  every program stopped). `stop:` and `cleanup:` always run: a pass, a
  failure, a failed `prepare:`, Ctrl-C (a running `prepare:` is stopped
  first). `## Before` and `## After` sections: any step, never filmed, not
  features; a failed `## Before` skips the features, `## After` always runs
  (its failure is exit 1); programs it starts live for the run. The report
  shows "Before the features" and "After the features"; `lifecycle.log`
  beside it. `si demo test|run <folder>`: every demo script in name order,
  one combined `report.html`/`report.json`, one JUnit file with a suite per
  script. Sources for those lines: `gist:ID@REV[/file]`,
  `github:owner/repo@REF/path.sh` or `…/dir/` (every script, name order),
  `https://… · sha256 HEX`, arguments after; `· in <dir>`. Pinned only (`si
  check`); asked once per pin with the source, size and first lines
  (`~/.si/trusted`); `--trust` for CI; refused without a terminal; a pin whose
  contents changed stops the run; cached in `~/.si/cache/scripts`; GitHub via
  the API (`$GITHUB_TOKEN`) or `gh` for private repos. `si demo test
  github:owner/repo@REF/demos/` (or one script, or a pinned URL): the repo
  fetched once at that pin into `~/.si/cache/repos`, layout kept; output in
  `demo-out/<repo>/<script>/`. Decks: `si play`, `si serve`, `si build` take
  the same sources; a deck's front matter `setup:`, `start:`, `stop:`,
  `cleanup:` (repeatable, in order) run around `si play`/`si serve` with a
  getting-ready page, programs kept running (logs in `$SI_RUN_LOGS` for the
  terminal drawer), torn down on quit; `si build` never runs them; `si check`
  checks them. `within`, `timeout:` and other durations also take minutes
  (`5m`, `1m30s`). Examples: `examples/webdemo/morning.md`,
  `examples/live/talk.md`. Title cards render `` `code` `` spans in `What it
  does:` and the feature name (accent colour or `--si-code`, mono font, no
  visible backticks; `` \` `` and `\\` escape; issue #7).

- **v3.0.1:** `si check` no longer reads a web demo's `transition:` as a
  deck's: `transition: fade 0.5s` or `wipe` in a demo script was an error
  ("unknown transition") with a warning that it had no effect. A deck keeps
  the deck values; a demo is checked against its own (cut, fade, slide, wipe,
  pop, zoom, circle).
- **v3.0.0 (si 3):** decks that present themselves, demos that film
  themselves; the release that completes web demos (a "What's new in si 3"
  page on the site). Web demo videos are edited like a video. Every scene
  change (a `go` that loads a page, `terminal` ↔ `browser`, a `window`
  switch, card ↔ clip in `<script>.mp4`) is now a transition, `fade 0.4s` by
  default (it was a hard cut; while a page loads its last frame holds):
  `transition: slide 0.6s` in front matter, `--transition`, or `· pop` after
  one step; `cut`, `fade`, `slide[-left|-right|-up|-down]`, `wipe[-…]`, `pop`,
  `zoom`, `circle` (from the cursor); `si check` reports an unknown name. The
  video keeps its length, so the voice, captions and chapters keep their
  times. New steps: `zoom to "…"`, `pan to "…"`, `zoom out` (`· 0.8s`; a
  crop of each frame, eased, inside the region when there is one) and
  `Region: … · zoom` (open on the page, zoom into the region); `spotlight
  "…"` (alias `focus`; `· circle`, `· blur`, `· label "…"`, `· for 3s`,
  `spotlight off`); `window "name"` / `window 2` / `window main` (perform and
  film a popup or tab, such as si's teleprompter); `drag from "A" to "B"`,
  `drag from X,Y to X,Y`, `drag by DX,DY`. Title cards keep `What it does:`
  as written: "si reads…" is no longer "Si reads…". Fixed: a window that is
  not in front (a popup, a tab behind) keeps drawing, so a filmed teleprompter
  no longer freezes; a page written by script after it opened gets the
  overlay back; the spotlight always darkens (it took the page's text colour,
  which brightens a dark page). si's own demos film the teleprompter, drawing
  on a slide, a spotlight and a zoom.
- **v2.43.0:** si has one folder of its own, `~/.si` (`$SI_HOME` moves it):
  your si.conf (`~/.si/si.conf`), logins (`publish.json`), the certificate
  authority (`ca/`), running sessions (`sessions/`), Kokoro (`kokoro/`) and the
  caches (`cache/voice`, `cache/fetch`, `cache/tailscale`). They used to live
  in the system's config and cache folders (`~/Library/Application Support/si`
  and `~/Library/Caches/si` on a Mac); each moves into `~/.si` the first time
  si uses it, so nothing is lost. `si voice install kokoro` now also writes
  `voice: kokoro:af_heart` into `~/.si/si.conf` when it names no voice, so
  Kokoro is the default everywhere.
- **v2.42.1:** voices: a Kokoro voice name alone means Kokoro:
  `--voice af_heart` (and `voice: am_michael`) is `kokoro:af_heart`, as in
  `voices:`.
- **v2.42.0:** web demos: `si demo run` and the `screenshot` step can film a
  region of the page instead of the whole viewport. `--region "<css>"` crops
  every clip to that element's box (the cursor, clicks and notes still play;
  only the output frame is cropped), `--crop x,y,w,h` crops to a fixed pixel
  box, a `Region:` line under a `##` feature crops that feature's clip, and
  `region:` in the front matter is the script's default. The captions variant
  shrinks the region into the frame with its aspect kept, and `<script>.mp4`
  letterboxes region clips to the viewport so the features join. Default is
  unchanged: the whole page when no region is given.
- **v2.41.0:** `si demo run` writes **one file by default**:
  `demo-out/<script>/<script>.mp4`, the whole run with its voice, a title card
  and a chapter per feature. What used to be `all.mp4` is now `<script>.mp4`.
  Everything else is asked for with `--with`, a comma list that may repeat:
  `captions` (`<script>.captions.mp4`), `silent` (`<script>.silent.mp4`, was
  `all.video.mp4`), `audio` (`<script>.m4a`), `subtitles` (`<script>.srt` +
  `.vtt`), `transcript` (`<script>.txt`, new: what the `Say:` lines say, as
  prose), `chapters` (`chapters.txt`), `clips` (the same kinds per feature:
  `<feature>.mp4`, `.captions.mp4`, `.silent.mp4` (was `.video.mp4`), `.m4a`
  (was `.voice.m4a`), `.srt`, `.vtt`), `report`, `readme` and `all`. `with:`
  in the script's front matter and `demo.with:` in si.conf set a default
  (`--with` > `with:` > the project's si.conf > yours); `si check` reports an
  unknown name. The run still checks every step and exits 1 on a failure, but
  writes the report only with `--with report`. Only the work asked for is
  done (no captions rendered unless wanted), in a temporary folder that goes
  away. `--captions` is now `--with captions`; **`--no-captions` is gone**
  (a usage error: captions are opt-in). The `.voice.wav` is no longer kept.
  `si demo test` is unchanged. The docs site reads `<area>/<area>.mp4`, and the
  demos workflow films with `--with clips,subtitles,chapters,report`.
- **v2.40.3:** sites: an `unlisted: true` page now renders through the same
  site-page path as a listed page, so it carries the full site shell — the
  navigator (logo, menu, `cta:`), `_header.md`, `_footer.md`, the `si-doc`
  body wrapper and the theme — instead of a standalone document. It still
  stays out of the navigator menu and raises no orphan warning; that remains
  its only difference from a listed page. Listed and unlisted pages now share
  one render path, so the two cannot drift.
- **v2.40.2:** sites: a page with `unlisted: true` in its front matter is
  built and is a valid link target — the navigator `cta:`, `_footer.md` and
  links from other pages resolve in `si check` — but it stays out of the
  navigator menu. A page that is in no menu list and is not `unlisted:` is
  still an orphan: warned and not built, as before. Lets a "Talk to us" page
  live behind a cta button with no redundant menu item.
- **v2.40.1:** code windows on slides: a link to a step (`#5.2`) shows that
  step of the show; callout notes wrap into the room beside the window instead
  of dropping to the list under it (the list is for phones); a window repaints
  when the slide's type settles after load (lines no longer overlap). A
  terminal window running a shell block (`sh`, `bash`, `console`…) shows only
  what was typed, no `$ cat <title>` line; write the `$` prompt yourself.
- **v2.40.0:** auto-play: a deck that presents itself. **▶ Auto-play** in the
  deck's buttons (or `v`) reads each slide's `Notes:` aloud, advances at each
  `[CLICK]` (a build step, a canvas change, the next slide) and moves on after
  the last part; a slide with no notes stays its `(N min)` share or
  `autoplay.dwell` (3 s). A caption shows the line being said (CC hides it);
  the bar pauses, resumes and stops; any key that moves the deck, a click on
  the slide, the pen or a teleprompter's Next pauses it, and the teleprompter
  shows *auto-play*. Under `si play` each part is voiced on demand
  (`/_si/voice/<slide>/<part>.wav`) with the deck's voice through the voice
  cache, one part ahead; `si build --autoplay` writes
  `<deck>.voice/<slide>-<part>.m4a` beside the page, `--autoplay=inline` puts
  them inside it (warns over 20 MB); with no audio the browser's own voice
  speaks, labelled. `voices: Ada=af_heart, Ravi=am_michael` (or si.conf
  `voice.<Name>:`) gives each speaker of `speakers:` a voice; `pronounce:`
  applies. `si play --autoplay` and `#autoplay` open decks ready (the first
  click or key starts them). `si build --mp4 --voice kokoro` films the deck
  narrated from its own notes. si.conf: `keys.autoplay`, `feature.autoplay`,
  `autoplay.dwell`. The presenting demo gains "It presents itself".
- **v2.39.1:** web demos: `select "…" in "…"` waits for an option a page adds
  late (a list filled from the network), as other steps wait for elements;
  the shop example now offers Express a moment after the page shows. The
  Studio demo works on Linux too.
- **v2.39.0:** si's own demo videos: six scripts in `docs/demos/` (a deck
  from the command line to the room, presenting, the Studio, themes, mind maps,
  the terminal) are checked by `si demo test` on every push and filmed with
  the Kokoro voice on every minor release; the site shows each area's video
  with its chapters (Decks, Studio, Themes, Mind maps, Terminal), and the
  README has a "Watch it" row of posters. Fixed: a page with an iframe that
  reloads (the Studio's preview) no longer makes a click wait for "still
  loading".
- **v2.38.0:** `si demo run` makes three videos from one take: `<feature>.video.mp4`
  (the picture alone), `<feature>.mp4` (picture and voice) and
  `<feature>.captions.mp4` (picture, voice and captions); the same for the
  whole run (`all.video.mp4`, `all.mp4`, `all.captions.mp4`, title cards and
  chapters in each). Captions never cover the picture any more: in the
  captions video the picture shrinks into the top of the frame and the
  captions sit in a dark band under it (they used to hide the command just
  typed in a terminal). A `.vtt` is written beside every `.srt` (and
  `all.srt`/`all.vtt`), and README.html plays each feature with a captions
  track and links all three videos; so does the report. Long waits play
  faster: after its first 3 s a command, a program starting or a wait for
  output is sped up to last 2 s at most, with a ⏩ badge; the voice is never
  sped up, captions and chapters follow the shorter video (`--fast-forward
  5s`, `--no-fast-forward`). `--no-captions` skips the captions videos;
  `--captions` still runs and does nothing. Web demos keep backslashes in
  quoted step values except `\"` and `\\`. Also: `type "text"` with no field types where the focus is
  (an editor) instead of replacing a field; an `app:` with a query (the
  Studio's `?token=`) opens as written; `SI_STUDIO_TOKEN` (32+ characters,
  still loopback only) fixes the Studio's token for scripts; `si theme list`
  lists only the nine public themes. Six demo scripts of si itself in
  `docs/demos/` (decks, presenting, the Studio, themes, mind maps, the
  terminal).

- **v2.37.0:** `si voice`: `si voice install kokoro` sets Kokoro up in one
  command (a Python venv with kokoro-onnx 0.6.1 and the two model files,
  SHA-256 checked, in the user cache; idempotent; `--dir`; `si voice uninstall
  kokoro`), and si uses that venv on its own. `si voice say "text"` speaks a
  line aloud like macOS `say` (cached; `-o` saves a WAV; an engine that cannot
  speak falls back to the computer's voice with a note); `si voice list
  [engine]`. API keys: `si voice login openai|elevenlabs` asks without echo and
  keeps the key in a new user-level si.conf (`<config dir>/si/si.conf`, 0600),
  `si voice logout` removes it; environment variables still win; a key in a
  project si.conf is ignored and `si check` warns with file and line. The user
  si.conf is read before the project's (project values win). `voice:` and
  `voice.style:` in si.conf set the default voice and tone of `si demo run`.
  `si doctor` shows Kokoro's venv and files and whether each cloud key is set.
  si.conf warnings now carry their file. Fixed: headless Chrome (exports, screenshots, web demos) no longer asks
  macOS for a keychain ("Keychain Not Found") when HOME is not your own.

- **v2.36.1:** web demos: a terminal command that must fail, `run "…" ·
  exit 1` or `· fails` (any non-zero status), passes only when it does. Fixed:
  `run … · wait` after a `run` that was not waited for read the earlier
  command's exit status.
- **v2.36.0:** web demos with a terminal: front matter `start: <command> ·
  wait for "text"` (or `· wait for port N`, `· wait 5s`; `as NAME`) runs
  programs in the background for the whole run, `stop:` tears down at the
  end, `shell:` picks the terminals' shell. New steps: `terminal "Name"` (a
  real shell under a pseudo-terminal, filmed as a code window: `window=`,
  `title=`, `theme=`, `backdrop=`, `size=`), `run "command"` (typed at a
  person's pace; `· wait` waits for it to end and fails on a non-zero exit),
  `press ctrl+c` in a terminal, `expect output "text"`, `expect no output`,
  `wait for output`, `browser`, `start "…" as NAME` for one feature, `stop
  NAME` (`· wait`), `expect output of NAME`, `expect NAME running|exited 0`,
  `note "…" at line "text"`. Every program runs in its own process group and
  is stopped (SIGTERM, SIGKILL after 5 s) at its feature's or the run's end,
  on a failure and on Ctrl-C; logs `<name>.log` beside the report with
  secrets masked. The video follows the scene in front (terminal or browser);
  `si check` follows the scenes; the report and JUnit show the terminal's last
  40 lines on a failure. `app: none` for a terminal-only demo.
  `examples/webdemo/broker.md` films a tiny Go broker.
- **v2.35.0:** web demos on film: `si demo run script.md` performs each
  feature in headless Chrome and films it, with a drawn cursor that glides to
  each element, a ripple on clicks, `note "…" at "Field"` callouts and a blur
  over fields typed from `$VAR` secrets; every `Say:` line is voiced first and
  sets the pace. Voices: Kokoro `af_heart` (local, the default when
  installed), the computer's own (`say`, espeak-ng, Windows), ElevenLabs and
  OpenAI (`voice-style:`); `pronounce: si = ess eye`; voiced lines cached.
  Writes `<feature>.mp4` (and a silent one, the narration, `.srt` captions),
  `all.mp4` with a title card per feature and chapters, `chapters.txt` for
  YouTube, the report and README.md + README.html; `--captions` burns the
  captions in, `--viewport phone|tablet|WxH`, `--voice`, `--feature`. A
  failing step ends its clip and lands in the report. `si doctor` lists Kokoro
  and espeak-ng.
- **v2.34.0:** web demos, phase 1: `si demo test script.md` runs a web app's
  features in headless Chrome from a `type: demo` script with `app:` (a URL or
  a folder si serves), finds elements as a person would, clicks and types for
  real, checks text, URL, state and `no errors`, and writes report.html,
  report.json and JUnit (`--junit`); exit 1 on a failure; `$VAR` secrets
  masked. Markdown, AsciiDoc or JSON scripts; `examples/webdemo/` (a small shop
  app); RULES §27; a Web Demos page on the site. Video and voice come next.
- **v2.33.0:** `transition: morph`: what two slides share (a heading, a
  picture, a block named with `<!-- id: name -->`) glides from its old place to
  the new, the rest cross-fades (`morph 0.9s`; no motion when reduced motion is
  on); the Studio's Motion view for animated SVG canvases (drag parts, frames,
  timing, easing, colours as theme tokens, Play, the slide lines to copy);
  `--mp4` films the morph; stage diagrams get an aria-label.
- **v2.32.0:** `si doctor` lists every optional program (Chrome, ffmpeg, git,
  tmux, cloudflared, Tailscale, gh), found or not, with the install command for
  your computer (macOS, Debian/Ubuntu, Fedora, Arch, Windows); a feature that
  needs a missing one says how to install it instead of failing; every feature
  page of the site has an External dependencies section with macOS | Linux |
  Windows tabs. Website kinds: `si new me --type website --kind
  resume|portfolio|landing` (also in the Studio); a page's own `cta:`; links in
  card titles; sites print without their navigator. The Themes page shows how
  to make your own (the Theme view, a theme from a logo). CI runs the Studio's
  browser tests.
- **v2.31.0:** theme from a logo: `si theme from logo.svg` (PNG, JPEG, GIF or
  SVG) finds the logo's colours and writes a whole theme, light and dark, with
  every text and background pair passing WCAG AA; the Studio's New theme ›
  From a logo…; the MCP tool `si_theme_from_logo`. The docs site's Articles,
  Guides, Books, Mind Maps and Code pages open with picture tours.
- **v2.30.1:** a teleprompter opened after questions or cloud words arrived
  now shows them (the deck replays them on its hello); the docs site's Decks
  page is rewritten as a visual tour: layouts, canvases, overview, find, draw,
  break timer, the teleprompter, two presenters, recording, the phone remote,
  the audience QR, polls, word clouds and questions, each with a picture and
  its command.
- **v2.30.0:** make your own theme: the Studio's Theme view (⌘K › New theme:
  colour pickers for light and dark, fonts, a live specimen, contrast checks
  with Fix, Use in this folder; File › New Theme… in the Mac app);
  `/* si theme: Name · extends <built-in> */` headers; `si theme
  check|export|import` (CSS or W3C Design Tokens); MCP theme tools; the index
  page follows a `.css` theme.
- **v2.29.1:** drawing on a slide is ⌥ Option in draw mode, nothing else to
  install; an old, unused `si play` flag is removed.
- **v2.29.0:** two presenters: `speakers:` + `Notes[Name]:` / `Points[Name]:`; a
  teleprompter per speaker (picker, cue lines for the others, a → hand-over
  banner, your own clock), `speakers.control: turn`, one remote teleprompter
  code per speaker, `--notes --speaker`, handout labels, `si rehearse` per speaker.
- **v2.28.1:** `si new <name> --type website` (an alias of `site`); the docs
  site's Features menu is keywords only, with a Website page, Studio in the
  top bar, and Presenting and Audience inside Decks.
- **v2.28.0:** record your talk: **● Record** in the teleprompter keeps your
  voice and the slide timings (`.si/recordings/`); `si build --narration latest`
  makes the narrated video, paced as you spoke; `si rehearse` reports planned
  vs spoken per slide and your words per minute.
- **v2.27.0:** the docs site gains **Studio** (web and Mac app) and **si in
  the terminal** feature pages with screenshots; the terminal Studio's map
  tree indents (`>`), outdents (`<`) and undoes (`u`).
- **v2.26.0:** Studio History: switch or make branches, Pull (fast-forward
  only) and Push with ahead/behind counts.
- **v2.25.0:** `si publish --to ghost` (Admin API, a JWT from your Admin API
  key) and `--to wordpress` (REST API, an application password): the article
  as HTML, a draft unless `--public`.
- **v2.24.0:** word clouds: `<!-- cloud: id -->` collects a few words from
  each phone under `si play --audience`; the slide sizes each word by its
  count, and the teleprompter can hide any word.
- **v2.23.0:** `--review --https` (and `--https=tailscale`): the review link
  over HTTPS, like the remote pages.
- **v2.22.0:** every example on the docs site shows its result under the
  Markdown and AsciiDoc tabs; `<!-- embed: … bare -->` embeds without a caption;
  the Decks page covers presenting (keys, overview, find, draw with the
  trackpad); a Your audience page.
- **v2.21.0:** the docs site shows every example of the rules, the manual and
  the feature pages in Markdown and AsciiDoc (tabs; the choice is remembered);
  AsciiDoc decks gain si's fence options in `[source,…]`, real `<1>` callouts,
  alerts from `NOTE:` / `[NOTE]` blocks and relative `link:` targets.
- **v2.20.0:** `<!-- poll: … · hidden -->` keeps a poll's bars for the next
  click, so the first votes do not sway the rest; `audience.save: off` in
  `si.conf` keeps no votes or questions; a Buy me a coffee badge.
- **v2.19.1:** the docs list nine built-in themes; three client themes stay in
  the code but leave the docs and site (see [TRADEMARKS.md](TRADEMARKS.md)).
- **v2.19.0:** remote audiences: `si play talk.md --tunnel` puts the audience
  side (follow along, polls, questions) on a temporary public HTTPS link
  (Cloudflare's free quick tunnel, `cloudflared`) for a talk on a video call;
  only the audience side, gone when `si play` stops.
- **v2.18.0:** questions from the room: under `si play --audience` every
  phone can ask; questions land in a Q&A tab on your teleprompter (never on the
  other phones), *Show on screen* puts one over the slide; votes and
  questions are saved beside the talk when `si play` stops.
- **v2.17.0:** live polls: `<!-- poll: id · option · option -->` on a slide;
  under `si play --audience` the room votes on their phones and the bars grow
  on your screen (one vote per phone, changeable); elsewhere a plain list.
- **v2.16.0:** the room follows along: `si play talk.md --audience` and
  `<!-- qr: audience -->` on a slide; phones show the current slide as you
  present, with their own key (never your controls) and never your notes.
- **v2.15.0:** six new themes: `slate`, `twilight`, `forest`, `sunrise`,
  `neon`, `newsprint` (each light and dark, contrast-checked), and a Themes
  page with a picture of every theme; a bottom-right footer no longer
  overlaps the slide counter on short screens.
- **v2.14.0:** handouts. `si build talk.md --handout` prints every slide,
  eight to a page, with a header and (with `handout-url:`) a QR code back to
  the live deck; no speaker text unless you ask: `--handout=lines` (space to
  write), `=points` (Points:), `=notes` (the Notes: script).
- **v2.13.0:** the key that draws with the cursor is yours to pick:
  `draw.hold: shift` (or `alt`, the default, `ctrl`, `meta`, `off`) in si.conf.
- **v2.12.0:** a menu-bar icon in si Studio for Mac: every running `si play`
  / `si serve` (from the app or a terminal) with Open in Browser,
  Teleprompter, Phone QR Codes…, Copy Link, Show in Finder and Stop; the app
  stays in the menu bar after its last window closes.
- **v2.11.0:** draw with the cursor: in draw mode (`a`), hold ⌥ Option (Alt)
  and move, and the cursor draws where it goes, no click; a trackpad draws with
  one finger.
- **v2.9.1:** `si mcp` tool schemas every client accepts (Claude Code listed
  no tools with v2.9.0).
- **v2.9.0:** `si mcp`, an MCP server for AI agents (stdio, MCP 2026-07-28
  and earlier): `si_check` (problems as data), `si_rules`, `si_new`,
  `si_build`, `si_qr` and `si_screenshot` (the agent sees the slide); the
  Claude Code plugin includes it.
- **v2.8.0:** si for AI agents. The `si` skill (`SKILL.md` + one reference
  file per rules section) as a Claude Code plugin
  (`/plugin marketplace add felipeg48/simple-ideas`, `/plugin install
  si@simple-ideas`) and as `si --export skill` for any agent with skills;
  `AGENTS.md` for the others; https://simple-ideas.org/agents.html.
- **v2.7.0:** `si play talk.md --remote --https`: the phone and tablet pages
  over HTTPS, so a phone keeps its screen on and the notes stay private on any
  network. si makes a certificate authority of its own once (it can only vouch
  for local addresses) and each phone trusts it once, from a "first time" QR
  code; `--https=tailscale` uses a real certificate for your Tailscale name.
- **v2.6.0:** QR codes. `si qr https://…` writes a QR code as PNG, JPG, SVG
  or PDF, for a link, any text, a Wi-Fi network (`--wifi`) or a contact
  (`--vcard`), with your logo in the middle (`--logo`; a logo too big to scan
  past is refused, with the size that fits), round dots (`--style dots`) and
  your colours; `<!-- qr: https://… · caption=The slides -->` puts one on a
  slide or page; `q` in `si play --remote` shows the phone links as codes on
  the projected screen. si's own encoder now does every QR size (versions
  1-40) at every error correction level, each checked by a real reader.
- **v2.5.2:** the QR codes `si play` prints scan: they were unreadable
  before (a layout bug in si's own encoder), and the `--remote` ones are now
  short links (`/clicker?k=…`) that fit any address.
- **v2.5.1:** your phone as the remote: `si play talk.md --remote` gives an
  iPhone a teleprompter laid out for its screen (the script fills it, big Prev /
  Next) or the clicker; a phone that slept reconnects to the current slide, and
  the page keeps the screen on where the browser allows it (else it says to set
  Auto-Lock to Never); the docs site's rules pages carry the site's icon and
  footer.
- **v2.5.0:** a logo (the Stack, `docs/brand/`) on the app icon, the Studio
  and the docs site; Help and **About** in the Studio, the desktop app and
  `si --version --verbose` point to https://simple-ideas.org/; **Report a
  Problem / Request a Feature** open GitHub issue forms with the version, mode
  and OS filled in (`si report [--feature]`); the desktop build signs with a
  self-signed identity when you have one (`make desktop-cert`); `--mp4` films
  the real motion (transitions, canvas morphs, 3D camera flights; `--dissolve`
  for stills); review comments pin to a paragraph, bullet, code block or
  figure; the map editor edits in 3D; History, Review and About are in the
  accessibility check; sites take `favicon:`, `touch-icon:`, `og-image:`,
  `version:` and a picture-plus-name `logo:`.
- **v2.4.0:** the Studio's **History** panel (changed files and their diff,
  commit, a file's earlier versions with restore; git); **review links**
  (`si serve . --review`): reviewers on your network comment on slides and
  sections, the Studio's Review panel follows them live; `si check --budget`
  (and `budget.*` in si.conf) fails pages over 1 MB (3D maps 1.5 MB), and CI
  checks the examples; the map editor selects several topics and copies, cuts
  and pastes branches as Markdown lists; the terminal Studio edits mind map
  trees and code window looks.
- **v2.3.1:** the desktop Help menu opens the start gallery and the tour;
  `--mp4` also films decks (every slide and step; `--dwell` for a teaser) and 3D
  mind map Play; a LinkedIn link post carries a picture (`cover:`); ⌘K reaches
  the Studio from the map editor and previews; `.github/workflows/pages.yml`
  publishes the docs site to GitHub Pages once Pages is on (Settings ▸ Pages ▸
  Source: GitHub Actions) and the repository variable `SI_PAGES` is `on`
  (`gh variable set SI_PAGES --body on`).
- **v2.3.0:** `si publish` posts a deck to LinkedIn as a document post (its
  PDF, a swipeable carousel in the feed) and an article as a LinkedIn link or to
  dev.to / Hashnode, always as a dry run first (`--yes` sends; see
  `docs/publish.md`); `--mp4` films a code show or a mind map's Play (needs
  ffmpeg); `make docs-site` builds the rules into a browsable site with si itself
  (one page per RULES.md section, live examples on the home page); the Studio
  opens an empty folder with a start gallery of live starter thumbnails and a
  skippable tour.
- **v2.2.0:** in `si play --terminal` a code show advances one step per click,
  with its title and callouts; Terraform/HCL, Protobuf, GraphQL, F# and Elixir
  are highlighted; `si build --share-assets` writes three.js once for a folder
  of 3D maps instead of into every page; pointer tests click, double-click and
  drag like a person on 3D maps, 2D maps and code windows.
- **v2.1.1:** the desktop app runs a newer si you installed and tells you when a
  new release is out; the Studio editor completes code block and mind map
  options, include options and callout lines; a step's code is edited inside
  the Window view; a callout dragged in `si play` keeps its place in the
  Markdown; 3D mind maps on a phone start closer and offer a List view.
