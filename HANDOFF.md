# HANDOFF — for the next agent

Read this, then `FEATURE.md`, then `CLAUDE.md`. This is a Go CLI (`si`) that turns
Markdown/AsciiDoc into animated, self-contained HTML slide decks.

## Current state (2026-11-09)

- **v2.32.0: si doctor, dependencies, website kinds.** `internal/tools` (the
  optional programs: Bins, per-OS Paths, Install per platform mac/debian/
  fedora/arch/windows/other from /etc/os-release, `Need` = the message a feature
  prints); `cmd/si/doctor.go`; Chrome, ffmpeg, cloudflared, Tailscale go through
  it; the Studio History shows git's install line. Site: `internal/docsite/deps.go`
  (External dependencies sections from the same table: `featureDeps` for the
  generated pages, `<!-- si-deps: … -->` lines in hand-written ones). Website
  kinds (`--kind resume|portfolio|landing`, `internal/studio/starters/<kind>/`,
  `studio.StarterFolder`; a page `cta:` via `ctaLink` in render/site.go; card
  titles keep links). CI's `studio` job: vitest, the bundle up to date, every
  e2e (zsh installed for the terminal drawer).
- **v2.31.0: theme from a logo.** `internal/themefile/logo.go`
  (`LogoPalette`/`FromLogo`: PNG/JPEG/GIF pixels and SVG fills, OKLab
  quantising, accent = weightiest chromatic colour, accent-2 ≥ 40° away, every
  `Pairs` pair fixed with `FixContrast`, extends studio, deterministic);
  `si theme from`; Studio `POST theme/logo` + New theme › From a logo…; MCP
  `si_theme_from_logo`. Skipped: a brand-logo token (`url()` is refused in
  token values), a `--from-url` option.
- **Feature tours (after v2.30.1):** Articles, Guides, Books, Mind Maps and Code
  pages get a picture tour before "Try it live": `internal/docsite/tours.go`
  (Markdown + figure grids, `§` = backtick, code spans in captions become
  `<code>`), pictures in `docs/site/features/img/<slug>/`.
- **v2.30.1:** Decks page hand-written (`docs/site/features/decks.md`, images in
  `features/img/decks/`, from the showcase: file:// for keys, `si play --remote
  --audience` with Playwright phones for the live shots; navigate the deck by
  `#N` then a key so phones follow). The `decks` generated feature and its
  `More` field are gone from features.go. Fix: `replayRoom()` on the
  teleprompter's hello re-sends the last questions and clouds.
- **v2.30.0: theme editor.** `internal/themefile` (read/write theme files,
  extends, checks, DTCG export/import), `render.ThemeLayer`/`ThemeFileLayer`/
  `Options.ThemeCSS`, token metadata `assets.AllMeta`, `cmd/si/themecmd.go`,
  `cmd/si/mcptheme.go`, `internal/studio/theme.go`, `web/studio/src/theme-editor.tsx`,
  specimen `internal/assets/web/themes/specimen.md`, `e2e/theme.e2e.mjs`. Left
  out: start from a logo (next: theme-from-logo spec), use your own deck in the
  preview, Use in this deck, compare two themes.
- **README redesign (2026-11-10):** feature sections with screenshots
  (`docs/screenshots/readme/`, the site's `features/img/`), theme thumbnails
  linking to simple-ideas.org/themes.html#<name>; release notes moved to
  `CHANGELOG.md` (newest first). New releases: add the note to CHANGELOG.md.
- **v2.29.0: two presenters.** Voices parsed in `deck.voices` (`Slide.Voices`,
  speaking order; plain `Notes:` = first speaker; `Slide.Notes/Points` keep the
  joined text). A part starts at the click count of the parts before it
  (`data-step` on `.si-voice`); who speaks = the part starting on this step,
  else the last begun: same rule in presenter.html `turn()` and narration.go
  `voiceAt`. e2e `web/studio/e2e/speakers.e2e.mjs`. Next: per-speaker notes in
  the Studio inspector.
- **v2.28.0: record your talk.** Teleprompter ● Record (Shift+R; shown only when
  `GET /_si/record` answers, i.e. si play on loopback): MediaRecorder chunks
  every 2 s to `POST /_si/record/chunk`, marks on each view (`/mark`, audio
  clock = Date.now, not the pausable talk timer), `/stop` writes meta.json
  (`internal/serve/record.go`). `cmd/si/narration.go`: `findRecording`,
  `loadNarration` (spoken per view), `hold` (minus the transition into it:
  1.3 s motion, 12 frames dissolve), `muxAudio` (AAC), `si rehearse` +
  `paceReport`. Deck page path reaches the teleprompter as `D.path`.
- **v2.27.0: site Studio + terminal pages; TUI map indent/undo.**
  `docs/site/features/{studio,terminal}.md` with `img/*.jpg`. Terminal shots:
  run si in a pty (Python `pty`, TERM=xterm-256color, COLORTERM=truecolor),
  keep the bytes, replay them into the bundled `internal/assets/web/term/xterm.js`
  in headless Playwright and screenshot. TUI map: `>` / `<` (move op), `u`
  (mmUndo, 100 deep).
- **v2.26.0: History branches, pull, push.** `git/branches`, `git/branch`
  (`switch`/`switch -c`, `branchRE` + check-ref-format), `git/pull` (`--ff-only`),
  `git/push` (`-u origin <branch>` the first time); branch bar in history.tsx;
  `TestHistoryBranchesPullPush` against a bare remote.
- **v2.25.0: Ghost and WordPress.** `internal/publish/blogs.go`: Ghost Admin
  API (`ghostToken`: HS256 JWT, kid, aud /admin/, 5 min; `?source=html`) and
  WordPress REST (basic auth, application password; tags not sent). The CLI
  renders the article body to HTML (`Article.HTML`); `--user` for WordPress login.
- **v2.24.0: word clouds.** `<!-- cloud: id · hidden -->` (render/poll.go);
  `serve/cloud.go` (3 answers per phone, case-folded counts, top 60, hidden
  words), `cloud` relay messages (audienceView drops `hidden`), the phone's
  answer form and `cloudShow` in 06-media.js, the teleprompter's Q&A tab lists
  words with hide/show (`POST /_si/cloud`). Saved with the audience results.
- **v2.23.0: --review --https.** `httpsFor` (local CA leaf or Tailscale) and
  `serveSplit` (TLS + the plain `/ca` page on one port) in cmd/si/main.go, used
  by --remote and --review.
- **v2.22.0: results under every example.** `docsite.liveExamples` writes each
  ```markdown example of the rules and site pages as `live/ex-NNN.md` (a deck,
  with a `# Example` title slide shown from slide 2; else an article for
  document-only plugins), renders it first in a scratch copy of examples/ and
  embeds it (`embed: … bare`) only when it has no problems; examples with
  relative page links are skipped. `embed:` gained `bare`. Decks feature page:
  `More` (keys, overview, find, draw with Option).
- **Site: Your audience page** (`docs/site/features/audience.md`, menu, home card, Decks options row; remote.md points to it); hand-written site pages get AsciiDoc tabs too.
- **v2.21.0: AsciiDoc tabs on the site.** `deck.MarkdownToAdoc`
  (`internal/deck/mdadoc.go`, the inverse of `adocToMarkdown`);
  `docsite.withAdoc` turns each ```markdown example (rules pages, manual,
  feature pages) into `<!-- tabs -->` Markdown | AsciiDoc.
  `TestAdocExamplesRenderTheSame` renders every RULES/manual example from both
  and compares the HTML. Reader fixes it needed: fence options pass through
  `[source,…]`, `<n>` callouts kept, admonitions as `> [!NOTE]`, `[NOTE]` +
  `====`, `link:` any target, no extra slide for `==` after `---`, no slide
  breaks in documents.
- **v2.20.0: hidden polls, audience.save, coffee.** `· hidden` on a poll adds
  `<span class="si-step si-poll-reveal">` (render/poll.go, slides only); layout.css
  hides `.si-poll-n`/`.si-poll-bar` until it is `.is-visible` (deck and phones,
  which follow the step). `audience.save: off` (config `AudienceOff`, checked in
  `SaveAudience`). README badge, `.github/FUNDING.yml`, site footer pill.
- **v2.19.1: client themes off the docs.** `crdb`, `acemq`, `acemq-pro` stay
  in the code (decks keep building) but left RULES, manual, README, help text,
  site (`internal/docsite/themes.go` `unlisted`, pictures deleted) and the
  skill; `TRADEMARKS.md` + a README Trademarks section.
- **v2.19.0: `si play --tunnel` for remote audiences.** `cmd/si/tunnel.go`:
  `AudienceOnlyHandler` (audience key and routes only) on a loopback port,
  `cloudflared tunnel --url` (found on PATH or Homebrew), the trycloudflare
  address read from its stderr; the link printed, in the `q` overlay and in
  `<!-- qr: audience -->`. Found with a real tunnel: quick tunnels carry no
  server-sent events (the stream connected and stayed silent), so audience
  pages fall back to polling `GET /_si/audience/now` after 4 s silent; streams
  also got `no-transform`, `X-Accel-Buffering: no` and a 20 s heartbeat. Real
  test from outside: follow (#1 → #2), a vote (deck 0,0,1, viewer 0,0,1), a
  question; presenter routes 404 through the tunnel; cloudflared exits with
  si play. `TestAudienceOnlyHandler`. Not done: hosted relay (founder: later,
  if donations allow). The founder is setting up Buy Me a Coffee for a
  donate link (GitHub FUNDING.yml + site) — waiting for the URL.

- **v2.18.0: Q&A (audience mode phase 3).** `serve/qa.go`: `ask` (tidy text ≤ 500,
  name ≤ 40, one per voter per 20 s, 200 open max), `set` (open, shown,
  answered, hidden; one shown at a time), `questions` messages (presenter pages
  only: `audienceView` drops them), `/_si/audience/ask` (audience mux),
  `/_si/qa` (loopback `localOnly` + presenter mux), `SaveAudience` on exit →
  `.si/audience/<talk>-<date-time>.json`. Teleprompter: Q&A tab (`3`, hidden
  until a question), Show on screen / Off screen / Answered / Hide. Deck:
  forwards the list over BroadcastChannel to a local teleprompter window,
  `.si-qa-show` band, Esc → open. Phone: Ask a question sheet. Tests: `TestQA`;
  live: ask → tab "Q&A (1)" → Show → band on the slide → Esc clears; phones
  never get the question; a second question in 20 s is refused; the saved file.
  The founder asked how a remote (Zoom) audience could join: see the answer in
  the session (a tunnel now, a relay on simple-ideas.org later).

- **v2.17.0: live polls (audience mode phase 2).** `<!-- poll: id · a · b … ·
  multiple -->` (`render/poll.go`; slides and documents). `serve/poll.go`:
  `POST /_si/audience/vote` on the audience mux only; ballots per poll per
  voter (checked ids, ≤ 6 distinct choices, 300 ms between changes, caps),
  empty ballot withdraws; `relay.publish` sends `{type: poll, id, counts,
  total}` to every page, kept as `poll:<id>` for late joiners; `audienceView`
  passes polls. Runtime: `pollShow` (bars via `--_share`), audience phones
  get buttons (aria-pressed, localStorage voter id and choices; results after
  voting); counts only with `data-live` (audience phone, or the deck when
  `SI_REMOTE_QR.audience`). Tests: `TestPolls`; live: three phones, counts
  1,2,0 → 0,3,0 after a change, a late phone sees 0,3,0, results hidden
  before voting; a built deck shows the plain list. The showcase has a poll
  slide ("Which habit would your team start with?"). Gotcha: saving files
  into the folder si play serves triggers live reload (a test artefact).

- **v2.16.0: audience mode, phase 1 (follow along).** `si play --audience`
  (starts the remote listener if needed): `Server.AudienceKey`, cookie
  `si_audience`; `RemoteHandler` admits the presenter key to the presenter mux
  and the audience key to `audienceHandler` (`/a`, the deck with
  `render.Options.NoSpeakerText`, files, `relay.audienceEvents` through
  `audienceView`: state cut to name/i/k/kmax/n/title). Runtime role
  `audience` (follows like draw; navigation and commands but full screen /
  mode ignored; controls hidden by CSS; no wake-lock hint). `qr: audience`
  → hidden `.si-qr-audience` filled from `SI_REMOTE_QR.audience`. Tests:
  `TestAudience` (routes, no notes in page or stream, no sends, presenter key
  intact). Live: presenter deck shows the code (Vision decodes it), an
  iPhone-size audience page follows, its keys/taps move nothing, no notes in
  its HTML. Also: the v2.15.0 docs-site failed (the Download page left the
  build with its menu item); it is under Help now, and CI builds the docs
  site on every push.

- **v2.15.0: six new themes and a Themes page** (the founder: pictures on the
  site and GitHub, the name to use, at least five more themes). `slate`,
  `twilight`, `forest`, `sunrise`, `neon`, `newsprint` in `assets.Themes`;
  `TestNewThemesContrast` (fg, muted, accent, accent-2 ≥ 4.5:1 on bg and
  surface, on-accent on accent, light and dark); added to the a11y suite.
  Pictures: `make theme-shots` builds the showcase without its `accent:` /
  `watermark:` (they hid each theme's accent) in every theme, slide 5.2, light
  beside dark → `docs/site/themes/<name>.jpg` (~40 KB each). Site: `themes.md`
  generated from `assets.Themes` (top menu "Themes"; the duplicate "Download"
  item went, the "Get si" button links there); README "Themes" grid. Found on
  the way: a bottom-right footer overlapped the slide counter on short
  screens (720 px); it now stays ≥ 2.2rem up.

- **v2.14.0: handouts.** `si build … --handout[=slides|lines|points|notes]`
  → `handouts/<name>.pdf` (index tile: HANDOUT badge). The founder asked
  mid-build for no speaker text by default: `slides` (8 a page) is the default;
  `lines` adds ruled lines, `points` the Points:, `notes` the Notes: script
  (cues hidden). How: `pagePDF` prints the built page with
  `?si-handout=<mode>&si-paper=a4|letter#print`; the runtime sets
  `data-si-handout`, light mode, and a portrait `@page` appended after the
  deck's own landscape `@page` (in layout.css it lost to it); slides are
  `container-type:size` boxes, so content scales by its own container units.
  `handout-url:` → a QR code in `.si-handout-head` (render/qr.go
  `handoutHead`). Tests: `TestHandout` (modes; a real PDF when Chrome is
  there), `TestHandoutHead`; the four layouts looked at page by page, the
  header QR decoded by Vision. Not done: a Handout choice in the Studio's
  Build dialog. Next (the founder): 5+ new built-in themes, screenshots of
  every theme on the site and in the README.

- **Specs for the next five features** (2026-11-09): `docs/superpowers/specs/2026-11-09-next-five.md`
  and one spec each. Not started; the founder picks the order, one at a time.

- **v2.13.0: `draw.hold` in si.conf** (the founder: the drawing key must be
  configurable). `config.DrawHold` (alt/option, shift, ctrl/control,
  meta/cmd/command, off/none; anything else warns and keeps alt), sent as
  `SI_CONFIG.drawHold` only when set (golden unchanged); `05-annotate-keys.js`
  maps it to the event flag and key name. Tests: `TestDrawHold`, pointer e2e
  "draw.hold: shift makes Shift the drawing key, not Option". Docs: RULES §14
  and §16, manual, site features/remote, `si --export conf`.

- **v2.12.0: si Studio's menu-bar icon.** The founder asked for an icon to
  see where talks run, open them, quit; only while the app is open. `si play`
  / `si serve` write `<config dir>/si/sessions/<pid>.json` and remove it on
  exit (`serve.WriteSession`; `$SI_SESSIONS_DIR` for tests); with `--remote`
  the loopback-only `GET /_si/qr` shows the phone codes. `app_darwin.m`:
  NSStatusItem (SF Symbol `rectangle.stack`, template), menu filled in
  `menuNeedsUpdate` from `goSessions` (`desktop/sessions.go`: loopback URLs
  only, dead pids' files deleted); per session Open in Browser, Teleprompter
  (`?si-view=tp`), Phone QR Codes…, Copy Link, Show in Finder, Stop
  (`goStopSession`: SIGINT, listed pids only); Open Folder…, Quit. The app no
  longer quits on its last window; the Dock icon reopens the folder picker.
  Tests: `TestSessionAndQRPage` (si), `TestSessions` (desktop). The menu itself
  was NOT seen on screen (no windows on the founder's screen): the founder's to
  check. The founder confirmed Option-drawing works on the trackpad (v2.11.0).

- **v2.11.0: draw with the cursor.** In draw mode (`a`), holding ⌥ Option
  (Alt) draws where the cursor goes, no click (`05-annotate-keys.js`: a
  pointermove listener starts/ends the stroke on `e.altKey`, keydown Alt starts
  it at the last pointer position, keyup Alt / blur end it); pointer e2e "draw:
  in draw mode, the cursor draws while Option is held, and only then". The
  v2.10 trackpad helper (absolute mapping) was removed after the founder's feel
  test; the `--trackpad` flag itself is gone since v2.29.1.

- **v2.9.1:** `claude mcp list` rejected v2.9.0's tools/list: `si_rules` sent
  `"required": null` (must be an array or absent); `obj` now omits it, and
  `TestMCPTools` checks every schema the way a client does. The headless
  `--mcp-config` run had tolerated it, so check new MCP work with
  `claude mcp list` too.
- **v2.9.0 (2026-11-09): `si mcp`.** `internal/mcp` (protocol checked against
  the 2026-07-28 spec through Context7: stateless, `server/discover`, version
  in `_meta`, `-32022` UnsupportedProtocolVersionError, `resultType`,
  `ttlMs`/`cacheScope`; `initialize` kept for clients still on 2025-xx).
  `cmd/si/mcp.go`: si_check (runs the CLI in-process, parses `file:line: msg`
  into structuredContent), si_rules (section by number/title/search),
  si_new, si_build, si_qr, si_screenshot (build to a temp dir + headless
  Chrome, PNG content); paths confined to the served folder; one CLI call at
  a time (`mcpMu`). Tests: `TestProtocol`, `TestMCPTools`. Real client: a
  headless `claude -p --mcp-config` session made a 3-slide deck using
  si_rules ×3, si_new, si_check, si_screenshot; `si check` clean. The plugin
  gained `.mcp.json` (`si mcp`). The plugin was installed for the founder from
  GitHub (`claude plugin install si@simple-ideas`, user scope).

- **v2.8.0 (2026-11-09): si for AI agents.** `internal/skill` makes the skill:
  `SKILL.md` (quoted YAML description that triggers on decks/slides/talks/si
  folders; the check-and-look loop; `docs/agent-rules.md`) and
  `references/NN-slug.md` per RULES.md section (`docsite.Split`). Output: `si
  --export skill [-o dir]` (default `.claude/skills/si/`) and `plugin/` (the
  Claude Code plugin, version stamped from cmd/si; regenerate with `go test
  ./internal/skill -update`; `TestPluginInSync` reads the version from
  cmd/si/main.go), listed by `.claude-plugin/marketplace.json`. `claude plugin
  validate .` passes. Tested for real: a headless `claude -p` in an empty
  project with only the exported skill wrote a 4-slide deck with spoken notes,
  points, big-number and a qr directive, ran `si check`, fixed a problem, built
  it. Docs: `docs/site/agents.md` (top-level "AI agents" menu, home section),
  manual, README, `brand.Pages["agents"]` + Studio Help. Next: `si mcp` (v2.9.0).

- **v2.7.0 (2026-11-09): `--https` for `si play --remote`.**
  `internal/serve/tls.go`: `LoadCA` (ECDSA P-256, `<config dir>/si/ca/`,
  key 0600, `$SI_CA_DIR` for tests; name constraints: private + 100.64/10 +
  169.254/16 + 127/8 IPs and `.local`, so it cannot vouch for a public site),
  `Leaf` (7 days, SAN, serverAuth), `SplitTLS` (one port: first byte 0x16 →
  TLS, else plain HTTP), `CAHandler` (plain side: `/ca` steps + fingerprint,
  `/ca.crt` DER as application/x-x509-ca-cert, else 301 to https).
  `cmd/si/main.go` `httpsFlag` (`--https`, `--https=tailscale`; refused without
  `--remote`), `startRemote` prints the fingerprint and a "first time" QR and
  adds it to the `q` overlay. `cmd/si/tailscale.go`: `tailscale status --json`
  → MagicDNS name, `tailscale cert` into `<cache>/si/tailscale/`. Tested:
  `TestLocalCA` (8.8.8.8, example.com, bank.ts.net refused), `TestSplitTLS`,
  `TestTailscaleName`; curl against a live run; headless Chrome on HTTPS is a
  secure context with `navigator.wakeLock` and no hint, clicker ↔ deck sync over
  TLS. NOT seen on a real device yet: iOS profile install + full trust with the
  name-constrained authority, the actual wake lock (headless denies it), and
  `--https=tailscale` (this Mac's tailscale CLI had no running daemon).
  The founder's first iPhone try still warned after installing the profile:
  Apple's own trust evaluation (`security verify-cert -c leaf -r ca -p ssl -s
  IP`, as iOS runs it) accepts the chain, so the cause is the full-trust switch
  (Settings › General › About › Certificate Trust Settings) — confirmed: with it on,
  the iPhone opens the remote pages over HTTPS with no warning. `si ca
  [trust|untrust]` (`cmd/si/ca.go`) trusts it on the laptop (login keychain via
  `security add-trusted-cert`; Windows `certutil -user`; Linux prints steps). The loopback port (7070) stays HTTP on purpose: localhost
  is already a secure context, and the authority does not cover `localhost`.
  Docs: `docs/site/features/remote.md` (Present from your phone: the one-time
  setup per laptop and per phone, every talk, Tailscale, troubleshooting) and
  `features/qr.md`, in the Features menu and home cards; a manual slide;
  `brand.Pages` remote/qr + Studio ⌘K Help commands (next binary). Next:
  `--https` for `--review` too.

- **v2.6.0 (2026-11-09): QR codes.** `internal/qr` (new; `cmd/si/qr.go` is
  gone): byte mode, versions 1-40, levels L/M/Q/H (tables from ISO 18004 table
  9), block interleaving with the short-block placeholder, version information
  (v7+), all four mask penalty rules, a module→codeword map, `Cover` / `Hole`
  (a centred picture is allowed when, per block, the hidden codewords stay
  within 80 % of half its error correction: hidden modules are unknown errors,
  not erasures), renderers `ANSI`, `SVG` (class mode for CSS colours),
  `Image` (3x3 supersampled, logo box-scaled), `PDF` (hand-written, vector,
  logo as an image XObject with an SMask). Tests: spec anchors (format words
  for every level, capacities, version-information words), `TestGolden` (64
  codes: versions 1-40 × 4 levels, module hashes in `testdata/golden.txt`),
  `TestVision` (macOS, `QR_VISION=1`: every golden code, the dots style and
  logo holes decoded by Apple Vision through `testdata/vision.swift`; refresh
  the golden file with `QR_UPDATE=1` only after it passes). Found by Vision:
  dots of radius 0.45 stopped reading above version 20, so dots are 0.5
  (touching) and alignment/timing stay square. `si qr` (`cmd/si/qrcmd.go`):
  `-o`/`--name`, `--png --jpg --svg --pdf` / `--type a,b` / the extension,
  `--ec`, `--style square|dots`, `--fg/--bg` (contrast and inverted warnings),
  `--quiet`, `--size` (px or cm/mm/in/pt), `--logo` (PNG/JPEG everywhere, SVG
  into SVG only) + `--logo-size`, `--wifi/--password/--security`, `--vcard`,
  `-` for stdin, `--terminal`; every output decoded by Vision end to end.
  `<!-- qr: -->` (`render/qr.go`, slides and documents, RULES §23; tokens
  `--si-qr-fg/-bg/-size`, hooks `.si-qr`, `.si-qr-code`, `.si-qr-cap`; dark on
  light in every theme), used on the showcase's closing slide. `si play
  --remote`: `q` overlay (`window.SI_REMOTE_QR`, loopback pages only,
  `TestRemoteQROnlyOnLoopback`; `keys.remote`), and `lanIPs()` lists every
  usable address (private, then 100.64/10) with the others printed and shown.
  Skipped on purpose: a `share:` front matter key (a `qr` on the closing slide
  does it). Next (v2.7.0): `si play --remote --https` with a local CA
  (crypto/x509; the phone installs a profile once) and `--https=tailscale`.

- **v2.5.2 (2026-11-08): QR codes that scan.** Every QR code `si play` ever
  printed was unreadable: `cmd/si/qr.go` wrote the 15 format bits transposed
  (the spec's (x, y) used as (row, column)). Found by decoding the printed codes
  with macOS Vision (`VNDetectBarcodesRequest` in a Swift script); now all decode.
  `TestQRFormatInfoPlacement` reads both format copies back from the printed
  code. Remote role links are short (`/tp?k=…`, `/clicker?k=…`, a 302 to the
  deck in that role; `TestRemoteShortRoleLink`) so they fit the encoder's
  version 1-4 limit (78 bytes; `TestRemoteLinkFitsAQR`); before, a long link
  silently printed no QR. `printQR` says why when there is none.
- **v2.5.1 (2026-11-08): the iPhone as a remote.** The `tp` teleprompter has a
  phone layout (`presenter.html`, `(max-width:40rem),(max-height:30rem)`): script
  full screen, previews hidden, one clock line, big Prev / Next first in a
  swipeable button row; portrait wraps title / position / clocks. `RelayJS`
  (`render/page.go`): a page hidden > 5 s reopens its EventSource when shown (the
  relay replays the last state); role pages hold a screen wake lock, and where
  there is none (plain HTTP on the LAN, so every phone at `http://192.168.…`)
  show an 8 s hint to set Auto-Lock to Never. Clicker checked at iPhone SE / 13 /
  13 landscape. e2e harnesses kill their `si --web` on process exit (no leaked
  servers when a step throws). Docs site: `rules/` (its own guide) gets the
  favicon, touch icon, og:image and footer. Next: HTTPS (or a pairing code) for
  `--remote`, which would also give phones a real wake lock; a native iPhone
  clicker (volume buttons); a direct test for a 3D pill click in the map editor.
- **Session wrap-up (2026-11-07), read first:**
  - **Released and installed:** v2.5.0 (tag pushed, `si --version` 2.5.0, CI + release + docs-site runs green).
  - **Website:** https://simple-ideas.org (GitHub Pages from `.github/workflows/pages.yml`, repo
    variable `SI_PAGES=on`, Pages source "GitHub Actions", custom domain set with
    `gh api -X PUT repos/felipeg48/simple-ideas/pages -f cname=simple-ideas.org`, HTTPS enforced,
    Let's Encrypt cert for apex + www, auto-renewed). DNS at Squarespace: 4 A `@` →
    185.199.108–111.153, 4 AAAA `@` → 2606:50c0:8000–8003::153, CNAME `www` → felipeg48.github.io,
    TXT `_github-pages-challenge-felipeg48` (domain verified). Squarespace only published records
    after delete + re-add; check with `dig @nsc1.squarespacedns.com`, not the sandboxed resolver
    (run dig with the sandbox disabled). If a record looks right in the panel but `dig` disagrees,
    re-add it. The www CNAME was once saved as `.github.com` (browser autofill).
  - **Founder's browser "Not Secure"** after HTTPS went live: incognito and phone were fine; the
    founder's profile runs the CSS Peeper extension (Plasmo/React inspector, injects into every page, sends
    analytics). Not an si problem; suggested disabling it to test.
  - **Process notes:** the auto-mode classifier refuses background agents that push/tag/release
    on their own and parallel worktree agents that merge to main; run agents as "commit locally
    only", then the main thread runs the gate, pushes, tags, installs. It also refused killing
    processes we did not clearly own.
  - **Open:** founder to run `make desktop-cert` once and open the desktop app (icon, Help
    menu, About, update banner never seen on screen). `si publish` needs the founder's LinkedIn app
    client id/secret, dev.to key, Hashnode token.
  - **Separate projects:** Retake (screen demo recorder) lives in `~/Progs/AI/retake-app`
    (CLAUDE.md there); the old spec is `docs/superpowers/specs/2026-11-06-screen-demo-recorder.md`.
  - **Notifications:** in this repo the agent is "Ideas": `say "Ideas needs your input"` before
    asking, `say "Ideas finished"` at the very end (saved in auto-memory).
- **v2.5.0 (released 2026-11-07):** the Stack
  logo (`docs/brand/`, `desktop/icon/gen.go` → `AppIcon.icns`, Studio favicon and
  header, docs site logo/favicon/touch icon/og:image); `internal/brand` (site URL,
  issue URLs, credits, build info) used by `si --version --verbose`, `si report
  [--feature] [--open]`, the Studio's About dialog and Help commands (`#si-about`
  JSON, `window.siStudio.about/help`), the desktop Help menu (Documentation, Rules,
  Manual, Release Notes, Report a Problem…, Request a Feature…) and About (the
  Studio's, with the binary in use); `.github/ISSUE_TEMPLATE/{bug,feature,config}.yml`;
  docs site Help menu + `_footer.md` with the issue links and a version badge
  (`version:` site key, from cmd/si's version; `TestDocsSiteShowsThisVersion`);
  new front matter keys `favicon:`, `touch-icon:`, `og-image:` (opt-in, users'
  pages carry no si branding) and `logo: picture.svg Name`; `make desktop-cert`
  + build-mac.sh self-signed signing (falls back to ad-hoc); real-motion `--mp4`
  (`cmd/si/motion.go`; `--dissolve` and automatic fallback); review pins on
  blocks (`slide:3/b2-1a2b3c`, `quote`); map editor in 3D; History/Review/About in
  the a11y e2e (dialogs take focus; Escape closes from outside).
- **v2.4.1: the docs site's feature pages.** The site (https://simple-ideas.org/, GitHub Pages with a custom domain)
  is titled "Write Markdown or AsciiDoc, present anything". Home: six main-feature cards (Decks, Articles, Tutorials
  and guides, Mind maps, Books, Code windows and shows) and eight secondary ones; menus Features, Examples, Rules,
  Manual, Download. `internal/docsite/features.go` writes one page per feature from `examples/` (live example, its
  Markdown, options, commands, rules links); the examples are copied to `live/` and built as site pages
  (`live/guide/`, `live/book/` as nested guide/book). `TestEveryFeatureHasAWorkingPage`. Two fixes found on the way:
  `<!-- embed: x.md full -->` (and `#section`) now makes the page load the mind map, 3D and code window runtimes the
  embedded body needs (`dr.adopt`, `TestEmbedFullLoadsRuntimes`; before, those parts were inert), and a guide/site
  link to a deck's slide (`deck.md#7`, `#7.2`) is no longer reported as a missing anchor. `make docs-site` uses
  `--share-assets`.
- **v2.4.0: History, review links, budgets, map clipboard, terminal editing.**
  - Studio History (`internal/studio/git.go`, `web/studio/src/history.tsx`): `git/status|diff|log|show`
    GET and `git/commit|restore|init` POST; git with fixed args and no shell in the workspace root,
    paths through `resolve` and after `--`, revisions must be hex. Restore in the UI goes through
    `EditorCtl.edit` (one undoable edit). Go test with a temp repo; e2e (init → commit → edit → diff →
    commit → history → restore → undo).
  - Review links (`internal/serve/review.go`, `web/studio/src/review.tsx`): `--review [--review-addr]` on
    play/serve, keyed listener (port + 2) serving pages with `ReviewJS` (no dot paths, no `/_si/*` but
    events/review); comments in `.si/review/<page>.json`; reviewer POSTs need this Origin; loopback author
    routes (list, events, reply, resolve) behind `localOnly`; SSE also polls file mtimes so another si's
    writes show live. Go test (key, origin, limits, blocked paths, author vs rebinding host); e2e
    (reviewer comments → Studio sees it live → resolve).
  - `si check --budget` / `budget.page`, `budget.3d` in si.conf (config.Budget); CI checks 4 example folders.
  - Map editor: multi-select (⌘/Ctrl/Shift-click), ⌘C ⌘X (`copy` via the API → clipboard, then
    `delete-many`), paste event → `paste` op (Markdown list under the topic); `mindmap.Branches`.
  - Terminal Studio (`internal/tui/edit.go`): focused mind map preview is an editable tree; `:` has
    window style / numbers / title per code block.
- **v2.3.1: v2.3.0's gaps closed.**
  - Desktop: Help ▸ Start Gallery / Take the Tour run `window.siStudio.gallery()/.tour()`
    (set by app.tsx; `#gallery` / `#tour` open them on load) via `goHelpJS` (main.go, tested).
  - `si build deck.md --mp4 [--dwell N]` (video.go `exportDeckMP4`): one Chrome screenshot per
    slide view (`#N.k`, 1920×1080), held `stepHold` (dwell, else minutes / views, else 3 s), 12-frame
    crossfades (`withFades`). 3D maps (`view: 3d`): `#si-play3d-K` makes mindmap.js show Play view K
    still (mindmap3d.js treats it as reduced motion) and report the count in `<pre id="si-play3d">`;
    `exportMindmap3DMP4` shoots each with SwiftShader WebGL. Encodes checked with ffprobe: appendix
    5 views 8.97 s, showcase 37 views 35.4 s at --dwell 0.6, thursday 3D 5 views 13.97 s 1600×1000.
  - `cover:` front matter (else the first local picture) → LinkedIn Images API upload → the link post's
    `content.article.thumbnail` (`LinkedIn.upload` now serves documents and images).
  - Pages: `.github/workflows/pages.yml` (skipped until `vars.SI_PAGES == 'on'`; the owner must turn
    Pages on, Source: GitHub Actions — enabling it was left to the owner, it creates a public site).
  - Studio ⌘K root causes: (1) keys pressed in a same-origin frame (map editor, window preview,
    preview) never reached the Studio's window listener — app.tsx now hooks every frame's keydown
    on load and forwards ⌘K/⌘S; (2) mindmap-edit.js `select()` focused a topic for every selection
    the Studio pushed (each `si-mm-data`), stealing focus from the palette — quiet selections focus
    only when the frame already has focus. start e2e now types for real (20/20), and its thumbnail
    check polls instead of a fixed wait.
- **v2.3.0: publish, video, docs site, Studio first run.**
  - `si publish` (`cmd/si/publish.go`, `internal/publish/`): LinkedIn document post
    (deck → PDF via `si build --pdf` → Documents API initializeUpload → PUT → Posts
    API), link post (article, `--url` or `canonical:`), `--copy` (textutil RTF +
    pbcopy), dev.to (draft unless `--public`), Hashnode (GraphQL; needs a Pro plan).
    Dry run unless `--yes`; logins in `os.UserConfigDir()/si/publish.json` (0600,
    `SI_PUBLISH_CONFIG` overrides); LinkedIn login = loopback OAuth on
    `http://localhost:8723/callback` or `--token`. New front matter: `linkedin`,
    `tags`, `canonical`. Never tested against real services (httptest only).
  - MP4 (`cmd/si/video.go`): frames → hard-linked 30 fps PNG sequence → ffmpeg
    (H.264, yuv420p, faststart); `si code --mp4`, `si build map.md --mp4` (runtime
    `#si-export-play`: overview, each branch dimmed, tweened zoom). Concat-demuxer
    durations drifted on the last frame, hence the sequence.
  - Docs site (`internal/docsite`, `docs/site/`, `si --export docs-site`,
    `make docs-site`): 26 rules pages from RULES.md (CommonMark fence-aware split),
    the manual as a deck, live showcase embed. RULES §13's stage example is now
    shown as code (its `#` notes are not valid stage syntax).
  - Studio: start gallery (`gallery.tsx`, `GET api/gallery/{kind}` renders starters
    in a temp dir, cached) and tour (`tour.tsx`); starters `mindmap3d`, `codeshow`
    (also `si new --type`); `e2e/start.e2e.mjs`; a11y audits both.
- **v2.2.1: quality (no new features).**
  - **Security:** the play terminal endpoints (`/_si/term/api/*`) and `/_si/callout` go through
    `serve.localOnly`: a loopback Host and the same Origin. Before, any web page open in the
    browser could blind-POST a shell tab and keystrokes to 127.0.0.1 (no CORS preflight for a
    text/plain POST), and a DNS-rebinding host passed the Origin check. `TestTerminalRefusesOtherSites`.
    The desktop update banner opens only this project's GitHub release page (`releasePage`).
  - **Flaky e2e fixed:** "reload keeps unsaved text" unblocked saves before the reload, so the
    unload flush (visibilitychange) saved for real whenever the 700 ms autosave had not fired;
    and `openFile` clicked a folder before the tree loaded (closing it). 20/20 alone, 3/3 full runs.
  - **Accessibility:** `web/studio/e2e/a11y.e2e.mjs` (axe-core 4.14.0, dev dependency, injected by
    the harness only) audits 45 views: slides, an article, a guide page, 2D and 3D maps, a code
    window article, the teleprompter, every built-in theme light and dark, and the Studio (Write,
    Map, Window); it fails on any serious or critical violation and runs in `npm run e2e`.
    Before: 22 of 45 failed (87 nodes: contrast 49, aria-required-attr 12, label-content-name-mismatch 10,
    aria-prohibited-attr 6, aria-required-children 6, scrollable-region-focusable 2, aria-input-field-name 2).
    After: 0. Fixes: code token, link, nav and callout-title colours are the token mixed 20-32% toward
    `--si-fg` (darker in light, lighter in dark, every theme); mind map topics are named by what
    they show and carry `aria-expanded`; callout marks `role="img"`; the code window body is a
    focusable region; Studio: editor label, splitter values, the tab row a group, gutter/comment/green contrast.
  - **3D sizes:** each topic's holder keeps its 3D size (`holder.userData.size`); the 2D view
    rewrites the shared `n.w/n.d/n.h`. The pointer test aims at the real top face.
  - **Files split:** `web/deck.js` → `web/deck/01-core.js … 06-media.js`, joined in name order into
    `assets.DeckJS` (byte-identical, checked with cmp); `render.go` → `render.go` (deck, slides),
    `page.go`, `theme.go`, `index.go`, `chrome.go` (same declarations and code; golden unchanged).
  - **Review notes (not changed):** `/_si/position` and `/_si/position/cmd` stay same-origin only
    (navigation, used by LAN-opened decks); a missing Origin is allowed (non-browser local clients);
    the Studio export passes `--id <id>` to the in-process CLI (an id starting with `-` would be
    misread; ids come from the user's own file); callout and topic text is the author's Markdown
    with raw HTML allowed, the same trust as any deck; 2D and 3D share topic objects (x, w, h) and
    each recomputes them when it draws; maps never tear down their observers (a page never removes a map).
- **v2.2.0: terminal code shows, five languages, shared three.js, real-pointer tests.**
  `tui/text.go` `show()`: fences with the same id in a row become one box; each extra step is a build (`visible()`),
  the title gets " · k/N name", the shown step's callouts follow. `render/highlight.go`: hcl (tf, terraform, tfvars),
  protobuf (proto), graphql (gql, $vars), fsharp (fs, fsx; (* *) comments, no call colouring), elixir (ex, exs; :atoms);
  names in `langNames`, include detection in deck `extLang` and `codeshow.LangOf`. `si build --share-assets` (or
  `--no-inline-media`): `Options.SharedAssets` + per-page `AssetPrefix` (guide pages by depth) link
  `_si/three.min.js` + orbit controls, written once by `render.WriteShared` (with three's license); the example maps
  2.1 MB → 1.5 MB. `web/studio/e2e/pointer.e2e.mjs` (npm run e2e runs it too): page.mouse on built pages with
  SwiftShader: click a 3D pill opens its card, double-click Payment drills in, drag a 2D topic moves only it, drag a
  callout moves it. It found and fixed a callout drag that started from the unclamped place (codewin.js now starts
  from where the note is drawn).
- **v2.1.1: staying current, completions, step code, saved callout drags, 3D on phones.**
  `desktop/update.go`: `pickSi` runs an installed si (~/.bin, ~/bin, ~/go/bin, Homebrew, PATH) when newer than
  the bundled one (About says so); `latestRelease` asks the GitHub API at most once a day (cache in
  ~/Library/Caches/si-studio/release.json, 3 s timeout, silent offline) and the app adds "Update Available — vX"
  and a banner in the Studio window. Studio `si-lang.ts`: completions after ```lang (code window options and
  values; mind map options after ```mindmap), `title=`/`lang=` after an include path, callout line numbers.
  Window view: a CodeMirror editor per step (`codeshow` op `set-code`). `serve/callout.go`: POST /_si/callout
  (loopback Handler only, same-origin) finds a callout by line+text in the page's file or any .md under the
  served folder and rewrites `at=` with `codeshow.Apply`; codewin.js sends it when `window.SI_LIVE` (set by
  PositionJS, live pages only) and the page is not in a frame. 3D on a portrait phone: Fit frames the root and
  first ring; a List tool (`.si-mm-listview`) shows the fallback list.
- **v2.1.0: code windows and code shows.** `internal/render/codewin.go` groups fences with
  `window=` / `id=` (and their `<!-- callout: -->` lines, now a known directive, also in documents) into shows and
  writes a figure with each step's highlighted lines as JSON; `web/codewin.js` draws the chrome (mac / windows /
  terminal / none), the backdrop, the lines (morph keyed by text + occurrence), callouts (dot, wire, draggable note;
  a numbered list when the note does not fit), ← → Play in documents, build-step markers (`.si-cw-mark`) on slides,
  and a slide-to-slide morph (`deck.js` canvasKey `cw:<id>`). `internal/codeshow` edits the Markdown by lines
  (Studio Window view, `POST codewin`) and builds the export article (`Source`). `si code --png --svg --gif`
  (`cmd/si/code.go`): Chrome dumps `#si-code-export` (SVG frames: foreignObject with the window's CSS, tokens and
  data-URI fonts), rasterises each, Go encodes the GIF (a popularity palette, no dithering: 0.9 MB for the four-step example). Mind map code topics
  take `window=`/`title=` (and keep them through the map editor). Terminal: title bar in the code box, callouts as
  numbered notes. Examples: `examples/code/retry.md`, showcase "The retry rule", article terminal window.
  Fix: a fence's language and options are read by one helper (`codeshow.Attrs`, used by render, tui, studio): a
  first word with `=` is an option, not the language (``` title="config.csproj" ``` used to show the label
  `TITLE="CONFIG.CSPROJ"`), pandoc's `{.xml title="x"}` and `xml {title="x"}` work, and without a language the
  title's extension names it (`.csproj` → xml).
  Highlighter: calls (the name before `(`, the group's span only) as `si-tk-f` and PascalCase types as the new
  `si-tk-t` (C#, Java, Go, Kotlin, Swift, Scala, JS/TS, C++, Rust, PHP); colours in theme.css, doc.css, mindmap.css,
  codewin.css (`--si-cw-t`) and the terminal. Code labels use display names (`langName`: C#, JavaScript, Shell, XML…)
  in document headers, tabs, code window titles and the terminal; doc.css no longer uppercases them.
  Code includes (`deck/codeinc.go` `fileLang`): the language from the file name (`codeshow.LangOf`: `.config`,
  `.csproj`, `.props`, … → xml, Dockerfile), then the content (`<?xml` / a tag, JSON, `#!`); `lang=` wins (it
  already did, now documented).
- **v2.0.1: highlighter languages.** `render/highlight.go` adds C# (`csharp`/`cs`/`c#`/`dotnet`),
  XML with tags, attributes, comments, `<?xml?>` and CDATA (`xml`, `csproj`, `props`, `targets`, `nuspec`, `resx`,
  `xaml`, `svg`, `pom`, …; HTML shares it), C, C++, PHP, Ruby, Swift, Scala, PowerShell, Dockerfile (case-insensitive).
  Before, C# and XML fences had a label but no colour. `TestHighlightLanguages`.
- **v2.0.0: the mind map builder.** `internal/mindmap` (new):
  `Parse`/`String` model (topic line `[!sticky] text ![alt](img) {attrs} → target`,
  fenced code under an item = code topic, `# Title {#id}` = map, other text kept as
  `Part.Text`; the examples round-trip byte for byte), `Apply(Op)` (add-child,
  add-sibling, delete, rename, set, move, up/down, pos, tidy, meta, add-map),
  `FenceAt`/`Fence.Replace`/`SetInfo`, `Import`/`Export` (OPML, FreeMind, XMind
  content.json (+ content.xml read), Mermaid). Studio: `internal/studio/mindmap.go`
  (`POST mindmap` {path, text, line, op} → {text, data, opts, maps, front, select};
  `GET mmcanvas`), `render.MindmapEdit`/`MindmapCanvas`, `web/mindmap-edit.js` (the
  canvas: selection, keys, rename box, drag with drop zones, postMessage ops),
  `web/studio/src/map-editor.tsx` (iframe + panels; text → map redraw debounced;
  ops → `ctl` one undoable change via `editMap` in editor.tsx), e2e "map editor".
  CLI: `cmd/si/mindmap.go` (`--svg --png --scale --pdf` via Chrome `--dump-dom` of
  `#si-export`, which `mindmap.js exportSVG` fills; `si mindmap import|export`).
  Render: fence `steps`/`reveal=branches` (`.si-mm-step` marks with data-step,
  `sc.steps += branches`; runtime `applyReveal`, 3D `revealBranches`), `flow:`/
  `flow=`, fallback ids prefixed `mmN-` per fence, `"own"` count in the map JSON.
  Terminal Studio: mind maps stay a read-only tree (`e` edits in `$EDITOR`).
- **v1.14.0: 3D kinds.** `kind: blocks|bricks|toy` (front matter,
  fence `kind=`; `deck.MapKinds`, `mmOpts.Kind` -> `data-kind`). `mindmap3d.js`:
  `snap` (stud pitch 0.4), `P.brick` (studs share the body material so Play dims
  them), `studPath` (instanced round tiles, `inst` grow anim), `baseplate`; toy:
  `toon` (3-tone gradient), `outlined` (back-face hull, kept in
  `mesh.userData.outline`, dimmed by Play), `toySize` (drum / hex / pill / note),
  `P.scenery` (island, trees off topics, clouds on the far side, sky) in
  `this.deco`, left out of Fit; `P.setKind`; mindmap.js Kind button cycles.
  Examples: `examples/mindmap/payments.md` (`view: 3d`, `kind: toy`), a
  `kind=bricks` fence in `examples/article/retry.md`.
- **v1.13.0: 3D mind maps.** `deck`: keys `view` (2d|3d),
  `camera` (iso|perspective), `floors` (on|off), warned outside mind maps; the
  fence takes `view= camera= floors=`. `render/mindmap.go`: `data-view="3d"`,
  `data-camera`, `data-floors` only when set (2D markup unchanged);
  `dr.mindmap3d` -> `mindmapScripts()` inlines `assets.ThreeJS` +
  `ThreeOrbitJS` + `Mindmap3DJS` before `MindmapJS`. `web/mindmap3d.js`:
  `window.SIMindmap3D(mm)` shares the 2D instance's data, `st` (map, stack) and
  chrome; theme from `--si-mm-*` (colours resolved through a 1-pixel canvas);
  `enter/leave/rebuild/fly/drill/drillTo/back/reveal/tour*`. `mindmap.js`:
  `set3d`, `active()`, Find on every map (`/` only on a page), `chainTo` shared by
  2D and 3D search. Verified headless (SwiftShader WebGL): 24 runs (4 sizes ×
  light/dark × page, slide fence, article fence), search across inlined maps,
  breadcrumb back, Play, camera and 2D/3D toggles, drill-down, web Studio preview.
- **v1.12.0: mind maps.** `deck`: type `mindmap`, keys `layout`,
  `lines`, `nodes`, `map-theme`, `watermark-position` (warned outside mind maps).
  `render/mindmap.go`: goldmark list → `mmMap`/`mmNode` JSON (topic line:
  `[!sticky] text ![](img) {badge= collapsed pos= branch-pos=} → target`; targets
  `#id`, `slides: x.md#N`, URL, mind map file (queued, inlined, cycles resolve),
  other page); `Mindmap()` page (Document dispatches), `mindmapFence` (options
  validated); `component()` writes `.si-mindmap` + data script + nested-list
  fallback. `assets/web/mindmap.js` (one `Mindmap` per element: layouts map,
  right, org, radial; phone portrait → right; lines; chips; foreignObject code;
  drag/Shift-drag/arrows; drill-down with breadcrumb; `/` search and Play on
  pages; regrow when its slide becomes current) + `mindmap.css` (mechanics
  unlayered, looks and `data-mm-theme` palettes in si.theme; `--si-mm-*` in
  contract.go). Terminal: `textRenderer.mindmap` trees (fences, mind map docs in
  the Studio). Studio: `mindmap` starter/kind (web + terminal). Examples:
  `examples/mindmap/` (3 files, 4 maps), a fence in `examples/article/retry.md`.
  Tested: deck/render/tui unit tests, golden, headless 4 sizes × light/dark
  (page, slide fence, article fence), drill/search/Play, web Studio preview,
  studio e2e.
- **v1.11.0: phone clicker + pace vs last run.** `--remote` role
  `clicker` (`serve.RemoteRoles`; deck.js `role === 'clicker'`: a fixed `#si-clicker`
  overlay with inline token styles, posts `cmd` next/prev, follows `state`, sends
  `hello` on open, stopwatch resets on tap). Teleprompter `#lastpace`: elapsed vs the
  last run's per-slide times (`prevSpent`), "vs last run M:SS behind/ahead/on pace".
  Tested headless: a real `si play --remote`, deck + 400×800 clicker in light and
  dark (#1 → #2.1); the teleprompter popup with a seeded last run.
- **v1.10.2: shared media.** `render.shareMedia` (media.go): a
  data URI of 1 KB or more used in two or more src / href / poster / xlink:href
  attributes is written once in a small script after the slides (deck) or before
  doc.js (documents); each use carries `data-si-m-<attr>="N"`. Showcase build
  1495 KB → 614 KB. Checked headless: no broken images. Backlog cleaned: the
  `data-count` tween was already bounded (650 ms) and no canvas overflows at 400 px
  (showcase + the 7 CACI decks; only wide tables scroll, in their own box).
- **v1.10.1: canvases animated in the terminal.** `tui/canvas.go`
  (SVG → character grid: rects as boxes or bands, wires, text runs in order, dots;
  helper attributes; CSS state rules for opacity / display / fill / transform),
  `tui/canvasstate.go` (`canvasStates`, `SlideOpts`, canvas cache), presenter
  `frameMsg` at 10 fps while `moving`, 0.9 s ease between builds; stage `dot A..B`
  moves. Wires go under box edges and text; pill text snaps onto its band.
- **v1.10.0: real pictures and a teleprompter for browser decks.**
  `tui/pictures.go`: `PictureProtocol()` (env guess, never in tmux; `SI_PICTURES`),
  reserved cells with a U+E100+n marker, `placements` after rendering, `draw` (Kitty
  graphics chunks / iTerm2 OSC 1337) sent with `tea.Raw` 80 ms after a cleared frame.
  `serve/position.go`: `POST /_si/position` (deck.js `announce()` when `cfg.live`),
  `GET /_si/position/events` (state + cmd events), `POST /_si/position/cmd`
  (same-origin); `tui` follower `followBrowser`. Tested: unit tests, a headless
  browser (positions in, `next` moved #2.1→#2.2). Pictures not seen on screen (no
  graphics terminal here): the sequences are unit-tested.
- **v1.9.0: Lab in `si --terminal`** (`internal/tui/lab.go`):
  runbook entries + demo shell blocks (`render.DemoFiles`, `render.RunLang`); in tmux a
  shell pane below (`split-window -P`, `send-keys -l`, `x` adds Enter); outside tmux
  `sh -c` output streamed (`labLine` channel pattern), `!` full shell; placeholders
  prompt. Tested: unit test outside tmux; a detached tmux session (`-L sitest`) drove
  it headless (works with /bin/sh panes; the founder's zsh showed no prompt in a bare
  tmux within 15 s). Next: 1.10 real pictures (Kitty / iTerm2) and a teleprompter for
  a browser deck.
- **v1.8.0: `si --terminal [dir]`, the Studio in the terminal**
  (`internal/tui/studio.go`): files, live preview (slides, documents, images),
  problems (`studio.Workspace.Check`), `e` `$EDITOR` (tmux pane beside), `n`
  starters, `b` build (CLI in-process), `s` storyboard, `i` inspector
  (`deck.SetDirective`, the Go port of the web inspector's writer), `:` palette,
  Rehearse, Present (hands the terminal to `si play --terminal`; tmux opens
  `--notes` beside). Lab is a stub until 1.9. `docs/terminal-studio.md`.
- **v1.7.0: si in the terminal.** `internal/tui` (Bubble Tea v2 +
  Lip Gloss v2): `si play deck.md --terminal` (every layout, builds, block-letter
  titles, code/tables/cards/badges/logos, half-block images, stage diagrams; live
  reload; keys like the browser), `--notes` (teleprompter in a second terminal over a
  Unix socket: script with [CLICK] badges, timers and pace, NOW/NEXT, auto-scroll),
  `--dump N --size WxH`. QA: dumps converted to HTML and screenshotted; a pty driver
  ran presenter + teleprompter together. Plan: 1.8 `si --terminal` Studio, 1.9 Lab,
  1.10 real pictures (Kitty/iTerm2) and a teleprompter for a browser deck.
- **v1.6.3.** `<!-- cards -->` (`render/cards.go`, a pass in
  `codeFeatures` so slides and documents share it; card class `.si-cell` because
  `.si-card` is the index tile), `si fix [--write]` (`deck.FixComponents`), `si new
  <name> --type …` (`studio.WriteStarter`, shared with the Studio), Studio titles drop
  `==` (`plainTitle`), editor colours `==highlight==` (`HL_RE`), `## Title {#id}` in
  documents, `fonts-subset:`, desktop File ▸ Open Recent (own list, NSMenuDelegate),
  CACI: cards converted, the two [CLICK] warnings fixed.
- **v1.6.2: components without raw HTML.** `==text==` highlight
  (`render/highlight_mark.go`, goldmark delimiter like ~~strike~~; `plain()` drops the
  marks); `badge` and new `logo` work on slides (CACI pill / lockup markup, with
  `si-*` hooks; generic looks under `:where()` so a theme's `.pill`/`.lockup`/`.hl` wins);
  a `· part` that is no kind stays text. AceMQ Pro sample and CACI converted (CACI
  backup `_backup-2026-11-06-pre-components/`).
- **v1.6.1** (fix: the deck teleprompter `p` opened window.open(''), which the shell sent to NSWorkspace → "The application can't be opened. -50"; only http(s) pages of other sites leave the app now)
- **v1.6.0 release:** (tagged; the release workflow publishes it, plus
  `si-Studio-<v>-macos.zip` from a macOS runner).
- **v1.6.0: si Studio for macOS + storyboard thumbnails.** `desktop/` (own module,
  cgo + Cocoa/WebKit, no Wails): one WKWebView window per folder, each running the
  bundled `si --web --addr 127.0.0.1:0`; window.open → app windows; external links →
  browser; ad-hoc signed universal bundle (`make desktop-mac`); `docs/desktop.md`.
  The window UI was not visually tested (no windows on the founder's screen);
  `desktop/main_test.go` tests start/stop headless. Storyboard thumbnails:
  deck.js `?si-view=thumb` (role `thumb`: no channel, all steps), live reload skips
  it. Next: Windows/Linux desktop, `si publish` (LinkedIn).
- **v1.5.0: Studio S4.** Inspector (`inspector.tsx`, `inspector-edit.ts`
  `setDirective`, `EditorCtl.edit` one undoable minimal change; check API slides
  carry layout/transition/background/class/eyebrow/words/clicks/steps), storyboard
  (`storyboard.tsx`), focus view, Write view switch, narrow layout (< 900 px: nav
  drawer, stacked panes), preview re-syncs the slide after every load. Bundle left as
  is (measured; see docs/studio.md). Next: the desktop shell (Wails), slide
  thumbnails in the storyboard, `si publish` (LinkedIn).
- **v1.4.0: Studio S3 (modes + ⌘K).** `modes.tsx` (Rehearse, Present, Lab on the
  previewed deck), `palette.tsx` (⌘K, `fuzzy`). deck.js: `?si-view=tp` makes the page
  the teleprompter of the same deck elsewhere in the browser (role `tp-local`, no
  BroadcastChannel of its own), `window.SI.run(cmd)` (Lab opens the terminal with
  it). Next (S4): inspector, storyboard, focus mode, a smaller bundle, narrow layouts;
  then the desktop shell.
- **v1.3.0: Studio S2 (writing).** Completion for every directive/key with help,
  values and workspace paths (`rules.go` help/values/paths; `si-lang.ts`
  `valueOptions`, `pathOptions`, `setCompletionFiles`). **New** dialog
  (`new-dialog.tsx`) + `POST starter` (`starters.go`, `starters/`): eight
  starters, never overwrite, auto-link into the enclosing `_nav.md`/`_site.md`.
  `GET checkall` (one pass like `si check <dir>`) replaces one check per deck on
  refresh; `POST build` runs the CLI in-process (`studio.Options.Run`) into
  `html/` (`build-dialog.tsx`). Next: S3 modes (Rehearse, Present, Lab) and ⌘K.
- **v1.2.0: Studio S1 (documents in `si --web`).** `internal/studio/kinds.go`:
  `guides` cache, `parsePage` (guide defaults like serve), `pageKind`, `guideKind`.
  Tree nodes carry kinds (article, guide, book, demo, site, page, nav) and `guide`
  on guide folders; `GET decks` adds `kind`, `url`, `guide` plus one entry per
  `_nav.md`/`_site.md`; `GET check` renders a guide/book/site whole
  (`RenderGuide`) and returns `kind`. Front-end: icons, badges, kind labels,
  Open vs Play, slide-follow only for decks. Fixed in `si serve`/`si play`: an
  `index.md` beats the generated folder index at `/` and `/index.html`.
  Next (S2): completion for every directive and key, New… starters per type,
  Build panel, workspace check; S3 modes (Rehearse, Present, Lab) and ⌘K.
- **v1.1.0: web fonts.** `fonts:` (front matter, `_nav.md`/`_site.md` via
  `Guide.Defaults` with file paths made absolute, `si.conf`): roles heading / body /
  mono, sources google (default), bunny, fontshare, fontsource, or a local font file
  (`render/fonts.go`). Stylesheets and files go through `deck.FetchAsset` (cached for
  good, `--offline`), latin + latin-ext subsets only, woff2 preferred, inlined as data
  URIs in `@layer si.theme` after the theme. The example site uses a local Fraunces
  (OFL, `examples/site/fonts/`). RULES §15 "Fonts".
- **v1.0.0: sites.** A folder with `_site.md` (`deck.SiteFile`, `Guide.IsSite()`):
  pages default to `type: page`, navigators `topbar` / `topbar-centred` / `mega` /
  `sidebar` with dropdowns and a phone menu (`render/site.go`), folder links to
  sub-guides/books/sites (`NavItem.Sub`, built and checked on their own). Plugins in
  every document (`render/plugins.go` + doc.js/doc.css): `alert` (dismiss remembered,
  `until=`), `badge`, `event` (HUD toast countdown, 48 h window), `accordion`,
  `modal` … `end`, `carousel`, `embed` (card / `#section` / `full` / deck slides /
  folder card, `render/embed.go`). `--site-title` (old `--site <text>` warns; `--site`
  is free for site features). `examples/site/` (Northwind Engineering),
  `--export example --type site`, RULES §23. Fixed: a folder link in a nav list
  stopped parsing the rest of the list. Next: v1.1.0 Google Fonts (inlined at build
  time), then publishing integrations (Medium).
- **v0.9.10: search (`/`) in every document**: `render/search.go` writes a `#si-search`
  JSON block (a guide/book page: every other page, its sections with a 220-char snippet,
  a book's index terms, filled by `fillGuideSearch` after RenderGuide; a lone page: the
  build's page list); doc.js searches the page DOM first; Search button in the header /
  guide bar; `si serve` renders guides whole (like books) so live pages get the data.
- **v0.9.9:** `watermark:` / `watermark-opacity:` in every document type (fixed
  `.si-doc-wm` layer, inherited from `_nav.md`, printed on every page); `[CLICK]`
  cues in `Notes:` (teleprompter ▶ CLICK badge `.si-cue`; `si check` warns when the
  count is < steps or > steps + 1) — CLAUDE.md rule 5, RULES §11, agent guide.
- **v0.9.8: demo scripts** (phase 4): `type: demo` (`session`, `slides`, `duration`
  facts, INSTRUCTOR ONLY badge, `> Say:` speech callouts, `text` blocks labelled
  expected output, `### … (N min)` steps checked against `duration`), `si build
  --no-instructor`, the ▶ Demos panel strips front matter, `examples/demos/order-demo.md`
  (the showcase lists it), `--export example --type demo`, RULES §22. Deferred from
  phase 4: search across a guide, side-by-side diffs, versioned guides.
- **v0.9.7: books** (`type: book` in `_nav.md`; `render/book.go`): `Guide.BookNumbers`
  (front matter / chapters / lettered appendices), `## ` sections numbered N.k,
  captioned figures (image title) and `Table:` captions numbered and given ids,
  empty-text links filled from a whole-book label map (`bookLinks`), `((term))` /
  `(((term)))` index (`indexTerms`), generated `index.html` (cover), `contents.html`,
  `book-index.html`, `book.html` (one file, ids prefixed `c-<page>-`), `book.pdf`
  with `--pdf`; `si serve` renders the whole book per request; RULES §21;
  `--export example --type book|reference`. Mockups: "si Documents", "si Books".
- **v0.9.6: documents phase 3**: `<!-- code: GitHub permalink | path= ref= repo= -->`
  and `<!-- diff: commit/compare URL | repo= ref=a..b path= -->` (`internal/deck/remote.go`:
  GitHub API with GITHUB_TOKEN / GH_TOKEN / `gh auth token`, else raw URLs; local clones via
  `git show` / `git diff`; disk cache in os.UserCacheDir()/si/fetch, commits forever, refs 1 h,
  stale copy on network errors; `--offline`; moving-ref warning), diff rendering with old/new
  line numbers (`diffHTML`), `repo:` front matter, `<!-- slides: deck.md 2-3 | #id -->` in
  documents (iframe srcdoc, `render/embed.go`; deck.js guards replaceState for about:srcdoc).
- **v0.9.5: documents phase 2** (code, slides and documents alike): code-file
  includes `<!-- include: f lines=5-10 | tag=x [hl= title= lang= linenos indent=keep] -->`
  (`internal/deck/codeinc.go`, become fenced blocks at parse time; same-line runs
  stay one document part), fence attributes `title= hl= linenos start=`,
  `<N>` callouts + `<N> text` list, `<!-- tabs -->` groups (synced by label and
  remembered in documents) — `internal/render/code.go`; AsciiDoc `include::f[lines=..|tag=..]`,
  `.Title`, `[source,lang,highlight=..]`, `%linenums`. Tokens `--si-code-hl`,
  `--si-tab-active`. RULES §13 "Code from files, highlighted lines, callouts and tabs".
- **v0.9.4: documents phase 1** (spec `docs/superpowers/specs/2026-11-05-documents-design.md`,
  design mockups approved by the founder): `type: article` (lede from `description:`,
  reading time, contents on the right), **guides** = a folder with `_nav.md`
  (`deck.LoadGuide`, pages via `deck.ParseWith` inherit the nav's front matter;
  sidebar with read marks, breadcrumb, Prev/Next in nav order, `edit-url`,
  `_header.md` / `_footer.md` partials, subfolders, `.adoc` pages;
  `render.RenderGuide` checks links + anchors between pages), `## Step N:` badges
  in labs and guides, document theme tokens (`--si-font-prose`, `--si-prose-size`,
  `--si-prose-leading`, `--si-doc-measure`, `--si-lede-size`, `--si-nav-bg`,
  `--si-nav-active`, `--si-step-badge`; unset by default, CSS falls back), RULES
  §15 "Themes for documents", §19 Articles, §20 Guides; **`si serve`** (play
  without the terminal, guide-aware); **`si --export example --type
  deck|slides|article|guide|tutorial|lab|all`** (examples in `examples/article/`,
  `examples/guide/`, embedded with `all:guide`). Next: phase 2 (code includes
  with lines/tag, tabs, callouts, hl), phase 3 (GitHub permalinks, diffs), phase 4
  (`type: demo`, CACI demos last).
- **v0.9.3:** find (`/`) also searches the other pages of the folder, fuzzily
  (`idx` → index.html): `render.Pages` (index, decks, labs, PDFs, other `.html`
  in the output dir, si.conf `links.<label>: <url>`) → `SI_CONFIG.pages`;
  `si build` bakes it in (works from `file://`), `si play` reads it per request.
  Terminal drawer: runbook entries get **Copy** (prompt + clipboard, not run) and
  **Run**; `sendToShell` waits for the prompt (output quiet 250 ms) and uses
  bracketed paste when the shell enabled it (fixes "clicked command not visible").
  New front matter **`demos: ../demos/`**: markdown demo scripts in a `▶ Demos`
  panel (drop-down + the whole script, Copy/Run on shell blocks),
  `internal/render/demos.go` → `window.SI_DEMOS`, play + loopback only.
- **v0.9.2:**
  - **`theme: acemq-pro`** ("AceMQ Pro"): the acemq tokens + `acemq.css` +
    `internal/assets/web/themes/acemq-pro.css`, the full AceMQ deck style-guide
    catalogue (`.heard`, `.inc`+`.verdict`, `.split2`/`.blist`/`.topo`, `.zones`,
    `.opts`, `table.tbl`, `.facts`, `.feat`, `.bench`/`.bks`, `.flowline`/`.mnotes`,
    `.poc`/`.term2`, `.floaters`, `.icon-chip`, `.btn`, motion `.r .f .pop .pulse
    .grow-x`). `Theme.Extras` chains extra stylesheets; `acemq-pro.sample.md` is
    its gallery deck (`si --export themes`). RULES.md §15.
  - **`si theme list`**; theme names match loosely (`CanonicalTheme`: case,
    spaces/underscores → `-`, aliases), so `--theme "AceMQ Pro"` works; an
    unknown `--theme` is a usage error.
  - **`si play --remote[=tp,draw]`** (`--remote-addr`, default 0.0.0.0:play+1):
    keyed LAN listener (`?k=` → HttpOnly cookie) for a tablet: `tp` teleprompter
    and `draw` pen pad, synced by a relay (`internal/serve/remote.go`, SSE + POST,
    last state replayed). Transport `render.RelayJS` is injected only under
    `--remote`; built decks contain none of it. `docs/remote.md`, RULES.md §11.
  - **`si --export agent-rules`**: writes `CLAUDE.md` + `AGENTS.md` from
    `docs/agent-rules.md` (never overwrites; `-o -` prints).
  - **Teleprompter rule** (CLAUDE.md rule #5): `Notes:` = what the speaker says
    to the audience, verbatim, no stage directions; `Points:` = main points +
    speaker reminders. Examples, scaffolds, samples and the migrate prompt follow it.
- **v0.9.1:** built-in **`theme: acemq`** (AceMQ brand: Urbanist/Mona Sans + logo
  embedded from `internal/assets/web/themes/acemq/`, extra CSS in
  `internal/assets/web/themes/acemq.css`, emitted only for that theme by `ThemeCSS`;
  component classes `.hl .cards .card .tag .tl .steps .callout on-ink us-N` documented
  in RULES.md §15 "acemq theme extras"; `--si-brand-logo` hides/swaps the logo);
  **`<!-- eyebrow: … -->`** slide directive (`.si-kicker` above the heading, not in
  the title); CSS-file themes inline relative `url()`s (`inlineURLs` in
  `internal/render/css.go`); stage labels shrink to fit (`labelFit`); `--si-figure-max`
  token and `fit()` shrinks figures before type; lab docs: key/value tables wrap,
  byline `·` split, wide tables/code use the column, Previous/Next pager between labs
  built from one folder (`.si-doc-pager`); `make install` does rm+cp (macOS kills
  in-place binary overwrites — that is what killed `si selfupdate` once).
- **v0.9.0:** Studio runbook generator + visual JSON editor + `{{name}}` /
  `{{name:default}}` placeholders (Go-template words `end else break continue nil
  true false` are excluded; `{{.X}}` never matches); **lab documents** (`type: lab`,
  `render.Document`, `doc.css`/`doc.js`, RULES.md §18; `si build` lists labs with a
  LAB badge); Studio `si --web` (v0.8.0 work: Preact + CodeMirror bundle committed in
  `internal/assets/web/studio/assets/`, rebuild with `make studio-ui`, e2e with
  `make studio-e2e`). Spec: `docs/superpowers/specs/2026-10-28-si-studio-design.md`;
  plans 1/2a/2b in `docs/superpowers/plans/` are done and merged.
- Older: v0.7.x runbook drawer (`runbook:` JSON → `si play` terminal drawer,
  RULES.md §14, `docs/terminal.md`); v0.6.5 Points/teleprompter tabs; v0.6.3
  `--dev` hot reload + lab slides; v0.6.2 footer/watermark; v0.6.1 topic overview;
  v0.6.0 terminal drawer (play + loopback only).

## Non-negotiable rules

In CLAUDE.md ("Non-negotiable rules"); do not break one without the founder.

## How to work

- `make test` = gofmt check + vet + test. `make build` → `bin/si`.
  `make play-example`. **Run `go test ./...` and check its real exit code — never
  pipe it through grep (that masks the exit code and has shipped a red test before).**
- **Release:** bump `const version` in `cmd/si/main.go`, commit, `git tag -a vX.Y.Z`,
  push the tag — the workflow builds + publishes. README badges are dynamic.
- **Visual QA is mandatory for any visual change** and must be **headless only**
  (never open a window on the founder's screen): headless Chrome
  (`"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new
  --screenshot=out.png --window-size=1440,900 --virtual-time-budget=2000 file://…`)
  or a headless Playwright script. **Never the Playwright MCP browser tools.**
  For deck JS, `cat internal/assets/web/deck/*.js > /tmp/d.js && node --check /tmp/d.js` catches syntax errors (the files are fragments of one script, so check them joined).
- **Commits:** plain messages, **no AI/Claude attribution** of any kind.
- **Autonomy:** the founder wants full autonomy — pick sensible defaults, say what
  you chose, don't ask for confirmation on routine work. DO surface decisions that
  break a non-negotiable rule (that's the founder's call), e.g. the terminal.
- Adding a directive end-to-end: see CLAUDE.md "Adding a directive".

## The courses (separate from this repo)

- Course **sources** live at `~/Desktop/simple-ideas-course/markdown/` (one thin
  `<course>.md` + `<course>/sections/*.md` + `<course>/images/{anim,diagrams}`),
  with a shared `si.conf` and `anim/` (globe/ranges/roaches — the CockroachDB-brand
  timer animations that do NOT live in this repo).
- **Built site** ("CockroachDB University") is written alongside, into
  `~/Desktop/simple-ideas-course/`. Central build:
  `si build ~/Desktop/simple-ideas-course/markdown/ --site "CockroachDB University" --pdf -o ~/Desktop/simple-ideas-course`
- ILT source material (for migrations) is at
  `~/Roach/Courses/cockroach-university-ilt-courses/courses/` — **only** look there.
- **Migration recipe** (proven): one general-purpose agent per course, authoring
  ONLY under its own `markdown/<course>` subtree; it reads `docs/RULES.md` +
  `docs/authoring-canvases.md` + `docs/stage-diagrams.md`, matches the house style
  of an existing migrated course, replaces every raster with native SVG/stage/code,
  adds MORE animations than the source, runs `si check` on its own deck, and does
  NOT touch `index.html`/other courses/the repo. The lead does the central build.

## Founder decks outside this repo (built with si 0.9.1)

- **CACI RabbitMQ course** `~/Downloads/CACI/` (5 sessions + ref decks A/B, labs
  0–8, handouts, instructor demos). `theme: acemq`, footer `CACI/AceMQ 2026`,
  eyebrows + highlighter titles; labs/handouts are `type: lab`. Builds:
  `si build decks --output slides --site "CACI Training" --pdf`,
  `si build labs --output labs/html --site "CACI Training · Labs"`,
  `si build handouts --output handouts --site "CACI Training · Session outputs"`.
  Runbook source of truth: `labs/runbook.md` → `python3 labs/runbook-to-json.py`
  → `labs/runbook.json` (founder does not use Studio for CACI). Backups:
  `_backup-2026-10-29-pre-v0.9/`, `_backup-2026-10-30-pre-acemq-theme/`.
- **PagoNxt pitch** `~/Downloads/PagoNxtAceMQBroker/` (original, untouched) and
  `~/Downloads/PagoNxtAceMQBrokerV2/` (approved restyle per `STYLE-GUIDE.md`,
  `theme: acemq`; "Trestle" never on slides → "AceMQ Enterprise Distribution for
  RabbitMQ" / "AceMQ Distribution"). Build:
  `si build md/ --output html/ --site "AceMQ for PagoNxt" --pdf && cp html/pdfs/pagonxt-acemq.pdf PagoNxt-AceMQ-V2.pdf`.
  Open question to the founder: the "Free trial today" slide vs the guide's
  no-free-offer rule.

## Known minor issues / polish backlog

- Roadmap ideas are in CLAUDE.md "Roadmap ideas".
