# si: every feature

`si` turns Markdown (or AsciiDoc) into talks, articles, guides, books, mind
maps, code shows and whole websites: fluid on every screen, one
self-contained HTML file each. This is the complete list; the site shows each
one with pictures and live examples: **[simple-ideas.org](https://simple-ideas.org/)**.
What changed in each release: [CHANGELOG.md](CHANGELOG.md). The format, with
an example for every rule: [RULES.md](docs/RULES.md).

## Writing

- **Markdown or AsciiDoc** (`.md`, `.adoc`): AsciiDoc is converted (headers,
  sections, lists, tables, source blocks with si's options, callouts,
  admonitions, includes, links) and every feature works the same.
- **One format, many documents:** `type: deck` (default), `article`, `page`,
  `lab`, `demo`, `mindmap`; folders with `_nav.md` (guide, or `type: book`) or
  `_site.md` (website).
- **Starters:** `si new <name> --type deck|article|guide|book|website|page|lab|demo|mindmap|mindmap3d|codeshow`
  (a website `--kind resume|portfolio|landing`),
  joined to the menu of the guide or site they land in; `si new-canvas` for an
  animated SVG.
- **Includes:** `<!-- include: sections/02-payment.md -->`, code from files by
  lines or tags, and code from GitHub permalinks; cycle-checked.
- **`si check`:** every problem with its file and line (directives, options,
  missing files, links and anchors, `[CLICK]` counts, canvases, budgets);
  `si fix` turns raw HTML into si syntax; `si migrate` prints an editorial
  migration prompt for an existing deck.

## Decks

- **Fluid slides:** no fixed canvas; type and layouts reflow from a phone to a
  4K projector, in portrait too.
- **Layouts:** `center`, `two-col`, `split`, `quote`, `big-number`, `image`;
  columns, cards, eyebrows, highlighter (`==text==`), badges and logos.
- **Build steps:** `<!-- step -->`, `<!-- steps -->`, reveal list items one
  by one.
- **Animated SVG canvas:** one picture whose parts appear, fail, recover and
  move slide after slide (`<!-- svg: journey.svg +declined -->`); helpers
  for show/hide, alerts, flows, packets, growing bars, ticking counters.
- **Diagrams:** ```` ```stage ```` animated flows, ```` ```mermaid ````,
  ```` ```mindmap ```` on a slide, tables, highlighted code with callouts and
  tabs.
- **Media:** colour, gradient, image and muted video backgrounds with
  overlays; sized, aligned, captioned and full-bleed images; `![Caption](clip.mp4)`
  plays as a video (with a `poster=`); videos and linked files (PDFs, zips)
  copied beside the page and checked; web fonts inlined.
- **Transitions:** fade, slide, zoom; the canvas and code windows morph
  between slides; `transition: morph` glides any heading, picture or words
  two slides share (or a block named `<!-- id: -->`) to its new place.

## Presenting

- **`si play`:** live reload on every save; the URL keeps your slide.
- **Keys:** overview `o`, find `/` (slides, other pages, `links.*`), draw `a`
  (drag, or hold ⌥ Option and move: the cursor draws; colours, sizes, undo),
  break timer `t` (digital or analog, animations), light/dark `d`, full
  screen `f`, help `?`; every key remappable in `si.conf`.
- **Teleprompter** (`p`): the `Notes:` script in large type, `[CLICK]` cues,
  auto-scroll at your speed, mirror, notes only, slide and total clocks with
  pace against your last run, previews of now and next, the `Points:` tab;
  works from `file://`.
- **Two or more presenters:** `speakers:` and `Notes[Name]:` /
  `Points[Name]:`: a teleprompter each, cue lines for the others, a hand-over
  banner, each speaker's own clock; `speakers.control: turn`.
- **Record your talk:** **● Record** in the teleprompter keeps your voice and
  the slide timings; `si build --narration latest` makes the narrated video,
  `si rehearse` reports planned vs spoken per slide and words per minute.
- **Auto-play** (`v`, **▶ Auto-play**): the deck presents itself, reading
  each slide's `Notes:` aloud and advancing at each `[CLICK]`, with captions;
  any key, click or the pen takes over. Voiced by `si play` (cached, Kokoro
  by default), by `si build --autoplay[=inline]` into `<deck>.voice/`, or the
  browser's voice; `voices:` per speaker; `#autoplay` and `si play --autoplay`
  open it ready; `si build --mp4 --voice` films it narrated. Pause and Resume
  carry on mid-part (the voice from where it stopped, the browser's voice
  from its sentence).
- **Browser windows** (`w`, `si play` on loopback): web pages in floating
  windows over the slides, each origin through its own loopback proxy so
  consoles that refuse framing still show; links and redirects to other
  sites stay proxied, the address box shows the real address, back and
  forward cross sites.
- **Terminal drawer** (`` ` ``, `si play` on loopback): real shells beside the
  slides, a runbook of commands, demo scripts with Copy and Run; Viewer tabs
  that read and edit files, a split view of up to four panes, ⌘P quick open, ⌘⇧F search in
  files, and line spotlight for the audience.
- **A talk that sets itself up:** front matter `setup:`, `start: … · wait for
  "ready"`, `stop:`, `cleanup:` (repeatable, in order, local commands or
  pinned sources): `si play` and `si serve` run them before the first page (a
  getting-ready page with each step), keep the programs running, and tear down
  on quit; `si build` never runs them.
- **Talks from anywhere:** `si play|serve|build github:owner/repo@REF/talk.md`
  (or a folder, or `https://… · sha256 HEX`), fetched once per pin into
  `~/.si/cache/repos`, asked about once, `--trust` for CI.

## From your phone

- **`si play --remote`:** a phone clicker, a tablet teleprompter (one per
  speaker), a draw pad; codes on the deck with `q`; reconnect after sleep.
- **`--https`:** si's own local certificate authority (each phone trusts it
  once), or `--https=tailscale`; the phone stays awake. `si ca trust|untrust`.
- **Si Studio.app menu bar:** every running `si play` on the Mac, with its
  phone codes.

## Your audience

- **`si play --audience`:** the room follows the slides on their phones (their
  own key: slides only, never your notes); `<!-- qr: audience -->` puts the
  code on a slide.
- **Live polls:** `<!-- poll: id · a · b · multiple · hidden -->`: phones vote,
  the bars grow on the slide; `hidden` reveals on your next click.
- **Word clouds:** `<!-- cloud: id -->`: a few words from each phone, sized by
  count; hide any word from the teleprompter.
- **Questions:** *Ask a question* on every phone; a Q&A tab in your
  teleprompter, *Show on screen* on the slide.
- **`--tunnel`:** the same on a video call, through a temporary public HTTPS
  link; results saved to `.si/audience/` (`audience.save: off` keeps nothing).
- **Reviews:** `si serve . --review [--https]`: colleagues comment on slides,
  sections or one paragraph; replies and resolve in the Studio.

## Documents

- **Articles:** a lede, reading time, a contents column, code with copy
  buttons, callouts, tables, embedded slides.
- **Guides:** a sidebar, breadcrumbs, Previous / Next, reading progress,
  checked links and anchors; labs with steps and checks.
- **Books:** a cover, parts and chapters, numbered figures and tables,
  cross-references, an index, a one-file edition and PDF.
- **Courses:** a `_course.md` of modules and lessons (readings, slides that
  present themselves, videos, labs, instructor demos, quizzes; a deck as its
  narrated, captioned video with a slides PDF to keep), a course home
  with Continue, progress rings and ticks kept in the browser, a completion
  page with a printable certificate; learner and instructor builds, a website
  bundle (`--zip`), a SCORM 1.2 package for any LMS (`--scorm`, one SCO per
  lesson or `--scorm=single`; status, score and progress reported), a learner
  workbook and an instructor guide as PDF (`--pdf`), and `si publish --to
  skilljar` (sections, HTML, video and SCORM lessons, updated in place).
- **Course results:** `webhook:` sends lessons done, quiz scores (with which
  questions were right, never the answers), lab checks, videos watched and
  the course finished to Slack, Teams, Discord, your own endpoint (JSON) or a
  learning record store (xAPI 1.0.3); a notice (or required consent) first,
  an offline queue, and `si serve --relay` to keep the address off the page
  and sign every event (HMAC-SHA256).
- **Labs that check themselves:** `<!-- check: command · expect "text" -->`
  under a lab step: "Run this to check" in every build, a ✓ Check button in
  `si play` that runs it and shows the result.
- **The live class view:** `si serve course/ --class` and the trainer's
  dashboard at `/_si/class` (loopback, or a keyed link with its QR code):
  learners by lessons (done, quiz scores, lab checks, videos watched), who's
  stuck (the same check failed twice, no progress for N minutes), per-lesson
  and per-question summaries, a module filter, CSV, live over SSE; events kept
  in `.si/class/<date>.jsonl`, names only when learners type one.
- **CI results:** `si demo test --webhook URL --webhook-format slack` posts
  the run's summary (passed, failed, the failed steps, the report).
- **Quizzes:** `type: quiz` pages and ```` ```quiz ```` fences: single and
  multiple choice, explanations, a score and a pass mark; learner pages keep
  the right answers only as hashes (a self-check, not a secure exam).
- **Navigator layouts:** `nav:` puts the menu of a course, guide, book or site
  left, right, top or bottom, sticky or not, as an accordion, collapsed or
  numbered; on a phone it folds into a Menu button.
- **Demo scripts:** facts, `> Say:` lines, commands with expected output.
- **Plugins:** alerts, badges, logos, QR codes, event countdowns, accordions,
  modals, carousels, cards, tabs, `embed:` (cards, sections, whole documents,
  live slides).

## Websites

- A folder with `_site.md`: a navigator (`topbar`, `topbar-centred`, `mega`,
  `sidebar`), a call to action, search, favicon and link previews, footer;
  pages, articles, talks, guides and books inside it. `si new <name> --type website`.
- Start from a kind: `--kind resume` (one page that prints on A4 or Letter,
  projects, about), `--kind portfolio` (project cards with pictures, a case
  study each) or `--kind landing` (a hero with its own button, features,
  questions, a QR code, pricing). A page's own `cta:` is a hero button; a
  card's title can be a link.
- `unlisted: true` on a page builds it and lets links to it resolve (the
  navigator `cta:`, `_footer.md`, another page's body) while keeping it out of
  the navigator menu — a "Talk to us" page with no redundant menu item.

## Mind maps

- A nested list becomes a map: layouts `map`, `right`, `org`, `radial`; line,
  topic and map styles; badges, pictures, sticky notes, code, links to slides,
  drill-down into other maps, folding, search, Play.
- **3D:** pills on a floor, linked maps as glass floors, fly-in, three kinds;
  a phone layout (branches on one side, level labels, icon tools, touch).
- **Import / export:** OPML, FreeMind, XMind, XMind 8, Mermaid; SVG, PNG, PDF,
  MP4 (a 3D map: PNG, PDF and MP4 of the 3D view; SVG stays 2D).

## Timelines

- `type: timeline`: eras (`## Era · color=`), milestones (`### date · title`,
  a description, a picture medallion with caption and credit), a closing line
  and sources; dates kept as written, read for order.
- Layouts `snake` (30 milestones on one 16:9 page), `line`, `vertical`,
  `eras` (a column per era under a coloured header, also 30 on one page) and
  `cards` (picture cards on a thin dated rail);
  landscape is a slide (one era per click, `Notes:`), portrait a scrolling
  poster; a ```` ```timeline ```` fence in any slide or page; every theme,
  light and dark; on a phone the line layouts turn vertical and eras and
  cards stack.
- `https://` pictures fetched and inlined at build; `si check` counts and
  checks order and a `require:` rubric; `si build --png --pdf --svg` (a
  paged PDF and a one-page one; a vector SVG with real text).
## Infographics

- `type: infographic`: every `## heading · kind` is a block of a bento grid:
  big stats, before/after comparisons with the change, numbered steps, quotes,
  icon rows, pictures and a sources footer.
- Charts drawn to scale as SVG from Markdown tables: bars, donuts, lines,
  progress; every built-in theme, light and dark.
- Landscape is a 16:9 slide (one block per click, a teleprompter script);
  portrait a tall poster; an ```` ```infographic ```` fence on any slide.
- A block's text shrinks to fit its cell (down to 60%); `si check` warns
  about a block that would need less.
- `si build --png --pdf --svg`: a high-resolution picture, a one-page PDF and
  a vector SVG with real text.
- Motion: `animate: build` (once) or `loop`, `duration:`; numbers count up,
  charts grow, steps light one by one; per block `· enter=fade|rise|grow|count`
  and `· at=3s`; slides play on arrival, a block per click with steps.
- `flow` blocks: `Client -> API -[SQL]-> Database`, one path per line, laid
  out for you (left to right, or top to bottom in a tall frame), arrows
  numbered, drawn in order, packets travelling along them.
- Exact frames: `aspect: phone`, `story`, `portrait`, `square`, `landscape`
  or `1080x1350`, fluid inside.
- `si build --gif --mp4`: frames from the page's own clock, a small GIF (one
  adaptive palette, only changed pixels; loops for `loop`, plays once for
  `build`) and a full-size MP4; `--fps`, `--scale`, `--size phone,square`;
  `si check` warns past about 8 MB.
- Both show in the Studio (New… lists them) and as text in the terminal
  (`si play --terminal`, `si --terminal`).

## Code

- Code windows (macOS, Windows, terminal, none) with titles, themes,
  backdrops, line numbers, highlighted lines, numbered callouts placed by hand.
- **Code shows:** steps that morph into each other, on slides and in
  documents; `si code` exports PNG, SVG, GIF and MP4.

## Web demos

- **A test from a demo script:** `type: demo` with `app:` (a URL, or the app's
  folder, served by si) drives a web app in headless Chrome, feature by
  feature, each in a fresh browser: `go`, `click "Pay"`, `type "…" into
  "Field"`, `select`, `check`, `press cmd+k`, `hover`, `scroll to`, `wait for`.
- Elements found the way a person names them (text, label, aria name,
  placeholder, title), or `#id`, `[data-testid=…]`, `role=button "Pay"`.
- **`expect`** text, URL, `visible`, `hidden`, `disabled`, `enabled`,
  `checked`, `no errors` (console errors, failed requests); `$VAR` secrets
  from the environment, never printed.
- **`si demo test`:** ✓ / ✗ per step, `report.html` (expected, found, the
  closest elements, a screenshot, console errors) and `report.json`, exit 1 on
  a failure, `--junit` for CI. Markdown, AsciiDoc or JSON scripts.
- **`si demo run`:** the same steps filmed and narrated: a drawn cursor that
  glides, click ripples, `note "…" at "Field"` callouts, blurred `$VAR`
  fields; each `Say:` line voiced first so the words set the pace;
  one file by default, `<script>.mp4` (a title card and a chapter per
  feature); the rest with `--with` (or `with:`, `demo.with:`): `captions`
  (the picture shrunk above a band with the captions, so they never cover
  it), `silent` (picture only), `audio` (`.m4a`), `subtitles` (`.srt` +
  `.vtt`), `transcript` (`.txt`), `chapters` (`chapters.txt` for YouTube),
  `clips` (the same per feature), `report`, `readme` (README.md + README.html)
  or `all`; only what is asked for is made. Long waits play faster with a ⏩
  badge, the voice never (`--fast-forward 3s`, `--no-fast-forward`);
  `--viewport phone`; a failing step ends its clip. `--region "<css>"` (or a feature's `Region:` line, or front matter
  `region:`) crops every clip and the screenshot step to one element's box;
  `--crop x,y,w,h` crops a fixed box; default is the whole page.
- **Edited like a video:** every scene change (a `go`, `terminal` ↔
  `browser`, a `window`, the feature joins) is a transition, `fade 0.4s` by
  default (`transition:`, `--transition`, a step's `· slide`; `cut`, `fade`,
  `slide`, `wipe`, `pop`, `zoom`, `circle`), drawn from the take's frames with
  the voice, captions and chapters untouched. Camera moves `zoom to "…"`,
  `pan to "…"`, `zoom out` (and `Region: … · zoom`); `spotlight "…"` (alias
  `focus`: dark or `· blur`, `· circle`, `· label "…"`, `· for 3s`);
  `window "si-presenter"` films a popup or tab (`window main` returns);
  `drag from "A" to "B"` / `by DX,DY` is a real drag with the drawn cursor.
- **Promo kit:** `si demo draft <url>` reads a site headless (same origin,
  robots.txt obeyed, a page limit, nothing clicked or submitted) and writes a
  first promo script from its own words (marked to polish), with selectors
  `si demo test` finds; `session: keep` (one browser profile for the run: a
  cookie banner accepted once in `## Before`); `header-offset: auto|72px|none`
  (scrolls land below a sticky header, nothing in view is nudged); `click
  "Buy" in frame .preview` (same- and cross-origin iframes); `intro:` /
  `outro:` cards with `logo:`, `url:` and a built-in `theme:`; `music: bed.mp3
  · 20%` ducked under the voice; `Lower-third: Name · Role` and the
  `lower-third` step; `--with vertical` (1080×1920, blurred fill, large
  captions) and `--with highlight` (about 60 s: each feature's first sentence).
- **Voices:** `kokoro:af_heart` (local, the default when installed),
  `system` (say, espeak-ng, Windows), `elevenlabs`, `openai` (`voice-style:`);
  `pronounce: si = ess eye`; lines cached, only changed lines re-voiced.
- **`si voice`:** `si voice install kokoro` sets Kokoro up in one command (a
  Python venv, kokoro-onnx, the model files checked by SHA-256);
  `si voice say "text"` speaks a line aloud like macOS `say` (`-o` saves a
  WAV); `si voice list`; `si voice login openai|elevenlabs` keeps API keys in
  your own si.conf (0600), never a project's; `voice:` and `voice.style:` in
  si.conf set the default voice and tone.
- **With a terminal:** `start: <command> · wait for "ready"` (or a port, or a
  delay) runs a broker or a server in the background for the run, `stop:`
  tears down; `terminal "Name"` opens a real shell filmed as a code window
  (`window=mac`, `theme=dusk`, `backdrop=ocean`, `size=100x30`), `run
  "command"` types it (`· wait` for its exit status, `· exit 1` or `· fails`
  when it must fail), `expect output "text"`,
  `press ctrl+c`, `note "…" at line "text"`, `browser` goes back to the app;
  `terminal split right|down` puts a second shell beside the first, both
  filmed (`run "…" in 2`, `expect output "…" in 1`, `focus 2`).
  Workloads for a feature or a few blocks: `start "…" as load`, `stop load`,
  `expect output of load`, `expect load exited 0`. Every program stopped by
  process group, also on Ctrl-C; logs beside the report; the terminal's last
  40 lines in the report on a failure; `app: none` for a terminal-only demo.
- **Before and after:** `start:`, `prepare:` (alias `step:`) in the order
  written, `setup:` before each feature, `stop:` while the programs run,
  `cleanup:` after they stop, each repeatable; `stop:` and `cleanup:` always
  run (a failure, a failed `prepare:`, Ctrl-C). `## Before` and `## After`
  sections: any step, never filmed, not counted; a failed `## Before` skips
  the features. `si demo test docs/demos/`: a folder as one suite, one
  combined report and one JUnit file (a suite per script).
- **Scripts from anywhere:** each of those lines takes `gist:ID@REV`,
  `github:owner/repo@TAG/path.sh` (or a folder of scripts), or `https://… ·
  sha256 HEX`; demos run from `github:owner/repo@REF/demos/` or a pinned URL.
  Pinned only (`si check`), shown and asked once per pin (`~/.si/trusted`),
  `--trust` for CI, refused without a terminal; cached by pin in `~/.si/cache`.
- **si's own videos:** `docs/demos/` (decks, presenting, the Studio, themes,
  mind maps, the terminal) tested on every push, filmed on every minor
  release, shown on the site.

## Themes

- Nine built-in themes, each in light and dark: `studio`, `paper`, `mono`,
  `slate`, `twilight`, `forest`, `sunrise`, `neon`, `newsprint`.
- **Your own:** a CSS file of `--si-*` tokens; `/* si theme: Name · extends
  twilight */`; `si theme check` (values, contrast in light and dark),
  `si theme export|import` (CSS or W3C Design Tokens); a theme from a URL;
  **a theme from a logo** (`si theme from logo.svg`, the Studio's *From a
  logo…*: its colours, light and dark, every pair WCAG AA);
  your CSS in its own cascade layer.

## Output

- **One self-contained HTML file** per page (CSS, JS, images, video, fonts
  inlined); works offline, from a USB stick; `--share-assets` for sites.
- **`--pdf`**, **`--handout[=slides|lines|points|notes]`**, **`--mp4`** (the
  real motion, or `--dissolve`; `--narration` for your voice).
- **Project structure:** a `structure:` block in si.conf maps each kind of
  output to a folder (`slides/ deck`, `pdf/ deck:pdf`, `html/ site`,
  `images/ image`, `dist/ html/:zip, deck:zip, course:scorm`); `si build`
  writes each kind there, copies what is listed twice, rewrites every link,
  zips the bundles last; `si new <name> --structure`; `si check` warns about
  empty bundles and misplaced sources. No block: the flat layout.
- **`si publish`:** LinkedIn (a deck as a swipeable document post, an article
  as a link post), dev.to, Hashnode, Ghost, WordPress; always a dry run unless
  `--yes`.
- **`si qr`:** PNG, JPG, SVG or PDF codes, a logo in the middle, dots, Wi-Fi
  networks and contacts.

## Studio

- **`si --web [dir]`** or **si Studio.app** (macOS): files, an editor that
  completes directives, a live preview at Fit, 16:9, phone, portrait and 4K,
  `si check` as you type; Write, Rehearse, Present and Lab modes; ⌘K.
- **Views:** the Map editor (drag, reparent, 2D and 3D), the Window view for
  code, the Theme view (make a theme with live preview and contrast checks),
  the Motion view (draw frame 1, add frames, drag parts; si writes the
  animated canvas and the steps that play it).
- **History** (git: diff, commit, earlier versions, branches, pull, push) and
  **Review** (comments from a review link).
- **`si --terminal [dir]`:** the Studio as a keyboard app: preview, storyboard,
  inspector, mind map editing, Lab with tmux panes.

## Terminal

- **`si play --terminal`:** the deck in a terminal: block-letter titles,
  layouts, animated canvases, real pictures (Kitty, iTerm2) or half-blocks;
  `--notes [--speaker NAME]` for the teleprompter in a second pane.

## AI agents

- **The si skill and plugin** (`/plugin install si@simple-ideas`),
  `si --export skill`, `si --export agent-rules` (AGENTS.md, CLAUDE.md).
- **`si mcp`:** check, rules, new, build, QR, a screenshot of a slide, and
  theme tools, confined to the folder it serves.

## Built to last

- Go standard library plus goldmark; nothing fetched at run time except the
  pinned Mermaid script for decks that use it.
- Loopback by default; keys for every network listener; confined file access
  in the Studio and the MCP server.
- `si doctor`: the optional programs (Chrome, ffmpeg, git, tmux, cloudflared,
  Tailscale, gh), found or not, with the install command for macOS, Linux or
  Windows; a feature that needs a missing one says how to install it.
- `si selfupdate` (checksum-verified), `si report` for issues, `si manual`.
