# Changelog

What changed in each release of si, newest first. Every release, with its
binaries and the Mac app: https://github.com/felipeg48/simple-ideas/releases

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
