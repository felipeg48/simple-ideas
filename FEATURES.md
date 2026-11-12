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
  overlays; sized, aligned, captioned and full-bleed images; web fonts inlined.
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
  open it ready; `si build --mp4 --voice` films it narrated.
- **Terminal drawer** (`` ` ``, `si play` on loopback): real shells beside the
  slides, a runbook of commands, demo scripts with Copy and Run.

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
- **3D:** pills on a floor, linked maps as glass floors, fly-in, three kinds.
- **Import / export:** OPML, FreeMind, XMind, Mermaid; SVG, PNG, PDF, MP4.

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
  `press ctrl+c`, `note "…" at line "text"`, `browser` goes back to the app.
  Workloads for a feature or a few blocks: `start "…" as load`, `stop load`,
  `expect output of load`, `expect load exited 0`. Every program stopped by
  process group, also on Ctrl-C; logs beside the report; the terminal's last
  40 lines in the report on a failure; `app: none` for a terminal-only demo.
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
