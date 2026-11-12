# Changelog

What changed in each release of si, newest first. Every release, with its
binaries and the Mac app: https://github.com/felipeg48/simple-ideas/releases

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
