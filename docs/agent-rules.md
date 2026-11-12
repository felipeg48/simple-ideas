# Writing slide decks with si

This project writes talks as markdown and turns them into animated, fluid HTML
slide decks with `si` (simple-ideas). These instructions are for the AI agent
that writes or edits the decks. The full, authoritative format is printed by
`si rules`; read it before your first edit and whenever a directive is unclear.

## Commands

```bash
si rules                        # the complete markdown format (RULES.md)
si check <dir|file.md>          # validate; fix every problem before you finish
si check <dir> --budget         # also fail pages over 1 MB (3D maps: 1.5 MB); budget.* in si.conf
si doctor                       # which optional programs (Chrome, ffmpeg, git, …) are installed, and how to add one
si serve <dir> --review         # a keyed link where reviewers comment on slides and sections (--https: over HTTPS)
si play <dir|file.md> --open    # live preview with reload (http://127.0.0.1:7070/)
si build <dir> -o html/         # one self-contained HTML file per deck + index.html
si build <dir> -o html/ --pdf   # also a PDF per deck (needs headless Chrome)
si build <dir> --handout        # handouts/<deck>.pdf for the audience: slides only (=points / =notes add speaker text)
si build talk.md --narration latest  # talk.mp4 paced and voiced by a teleprompter recording (● Record in si play)
si build talk.md --autoplay     # auto-play (v) voiced: talk.voice/<slide>-<part>.m4a beside the page (=inline: inside it)
si build talk.md --mp4 --voice kokoro  # talk.mp4 narrated from its own Notes: (needs ffmpeg)
si rehearse talk.md             # a recording's pace: planned vs spoken per slide, words per minute
si theme list                   # built-in themes for theme: / --theme
si theme check themes/acme.css  # a .css theme: wrong values, WCAG contrast light and dark (with a passing colour)
                                # (a theme file starting /* si theme: Acme · extends paper */ sets only what it changes)
si theme from logo.svg --use    # a theme from a logo's colours (themes/<name>.css), light and dark, AA; then pick fonts
si new <name>                   # a scaffold deck that uses every rule
si --export example             # the showcase decks, to copy patterns from
si play talk.md --remote --https  # teleprompter / draw pad / clicker on a tablet or phone (keyed LAN port, HTTPS)
si qr https://… --png --svg     # a QR code file (on a slide: <!-- qr: https://… · caption=… -->)
si play talk.md --audience      # the room follows on their phones; <!-- qr: audience --> shows the code,
                                # <!-- poll: id · option · option --> asks them (live bars; · hidden: on the next click);
                                # <!-- cloud: id --> collects a few words each, shown as a word cloud;
                                # their questions arrive in the teleprompter's Q&A tab
si play talk.md --tunnel        # the same for a remote audience (a video call): a temporary public link
si demo test shop.md            # a web demo script (type: demo + app:) run against the web app in headless Chrome:
                                # report.html + report.json in demo-out/shop/, exit 1 on a failure, --junit junit.xml for CI
si demo run shop.md             # the same steps filmed and narrated: ONE file, demo-out/shop/shop.mp4 (title cards,
                                # chapters); more only with --with (comma list, may repeat; or with: in front matter,
                                # demo.with: in si.conf): captions, silent, audio, subtitles (.srt + .vtt), transcript
                                # (.txt), chapters (chapters.txt), clips (per feature), report, readme, all;
                                # long waits play faster (--fast-forward 3s, --no-fast-forward); --voice, --viewport
si demo test docs/demos/        # every demo script in a folder: demo-out/<script>/ each, one combined report, one --junit
si demo test github:owner/repo@v2/demos/ --trust   # demos from a repo at a pin (or "https://…/x.md · sha256 HEX");
                                # --trust only in CI: a person is shown new code and asked once per pin
si play github:owner/repo@v2/talks/talk.md         # a talk from a repo (si build and si serve too)
```

## A deck

```markdown
---
title: The promise we print at checkout
author: Ada Lovelace
theme: studio
---

# The promise we print at checkout

Following one parcel, order #4711.

Notes: (0.5 min)
Welcome. This talk follows a single parcel from the click on "buy" to a
doorstep, and the two moments it nearly broke the promise: delivered by Thursday.

---

## Three teams, one promise

- Warehouse, payments and courier each hit their own target
- One order in eleven is still late

Notes: (1 min)
Each team is green on its own dashboard, and the customer still waits. The
promise lives between the teams, so today we follow one order across all of them.

Points: Talking points
- Three hand-offs, three dashboards, no end-to-end owner
- One order in eleven late: the number to remember
```

- `---` on its own line separates slides; front matter comes first.
- `# H1` makes a title or section slide; `## H2` starts a content slide.
- Directives are HTML comments on their own line: `<!-- layout: two-col -->`,
  `<!-- column -->`, `<!-- steps -->`, `<!-- svg: assets/flow.svg words -->`,
  `<!-- background: … -->`, `<!-- class: … -->`, `<!-- eyebrow: … -->`,
  `<!-- footer: off -->`, `<!-- include: sections/01-intro.md -->`. An unknown
  directive is an error in `si check`; `si rules` lists them all.
- `<!-- transition: morph -->` moves what two slides in a row share (the same
  heading, picture or words, or a block named with `<!-- id: name -->` on
  both) from its old place to its new one; use it to show one idea growing or
  moving, not on every slide.
- A long talk is a thin main deck plus one file per section under
  `sections/`, pulled in with `<!-- include: -->`.

## Teleprompter vs Speaker Notes: the most important rule

Every slide can carry two blocks. They are different things.

**`Notes:` is the teleprompter.** It is what the speaker says to the public,
word for word, about what is on the slide. The speaker reads it aloud from the
teleprompter, so write it exactly as it will be spoken: full sentences, in the
speaker's voice, addressed to the audience.

- Never put instructions to the speaker in `Notes:`. No "pause here",
  "mention the retry", "tell them about…", "explain that…", "press next",
  "click to reveal", "skip this in a short slot", "read slowly".
- Never describe the slide or the tool: no "this slide shows…", "on the left
  you can see the layout…", "the canvas stays on screen", "on a phone the
  columns stack".
- Talk about the content itself. Bad: `Mention that the retry fixed most
  declines.` Good: `One retry thirty seconds later fixed most of the declines.`
- `(N min)` after `Notes:` is the slide's planned time: `Notes: (1.5 min)`.
  Give every slide one; the deck total is their sum.
- **`[CLICK]` is the one exception (recommended):** on a slide with build
  steps, a canvas or a stage animation, write `[CLICK]` in the script exactly
  where the speaker advances, one per step (and at most one more for the next
  slide). The teleprompter shows it as a ▶ CLICK badge; it is never read
  aloud. `The token turns red. [CLICK] Thirty seconds later the retry goes
  through.` Auto-play (`v`, `si build --autoplay`) reads the script aloud
  and advances at each cue, so the cues must match the steps (`si check`
  warns when they do not); that is one more reason the script must be words
  a voice can say, with no stage directions.

**`Points:` is the Speaker Notes tab.** It holds the main points of the
topic, as short bullets the speaker glances at to talk freely instead of
reading. Reminders meant only for the speaker (pace, emphasis, what to skip,
likely questions and their answers) go here, never in `Notes:`.

```markdown
Notes: (1.5 min)
The token reaches payments and turns red: the bank declined the card.
Thirty seconds later one retry goes through, and the order is paid.

Points: Payment retry
- Declined at 21:03, paid at 21:05: one idempotent retry
- Most night-time declines are timeouts, not fraud
- Pause after the red tile
```

Both blocks are hidden on the slides and run to the end of the slide (or to
the other block's line).

**Two or more presenters.** List them in the front matter (`speakers: Ana,
Felipe`) and give each one a script: `Notes[Name]:` is that speaker's
verbatim script (spoken sentences, `[CLICK]` included, the same rules as
`Notes:`), `Points[Name]:` their own points and reminders. Plain `Notes:` is
the first speaker's. Order on a slide is speaking order; put a `[CLICK]` where
the turn passes, and count the cues of every speaker together (one per build
step, plus at most one for the next slide). Each teleprompter shows its
speaker's script, the others as cue lines.

```markdown
Notes[Ana]: (1 min)
The token reaches payments and turns red. [CLICK] Felipe built what happens next. [CLICK]

Notes[Felipe]: (0.5 min)
Thank you, Ana. One retry, thirty seconds later, and the order is paid.

Points[Felipe]:
- If asked: a real fraud decline is never retried
```

## Documents: articles, guides, labs

Not everything is a slide deck. Pick the type by what the reader does:

| They will | Write | How |
|---|---|---|
| watch a talk | a deck | the default; `---` between slides |
| read one long piece | an article | `type: article`, a `description:` lede, `##` sections |
| follow steps over several pages | a guide | a folder with `_nav.md` (the page order), one topic per page |
| do hands-on work on one page | a lab | `type: lab`, `## Step N:` headings |
| look things up, read cover to cover | a book | a guide folder whose `_nav.md` says `type: book`: parts, chapters, appendices |
| run a live demo (the presenter) | a demo script | `type: demo`, `> Say:` lines, every command followed by its expected output |
| show and test a web app, feature by feature | a web demo | `type: demo` with `app:` (a URL or the app's folder), `## Feature`, `What it does:`, `Say:` lines, then steps as list items (`go`, `click "Pay"`, `type "…" into "Field"`, `expect text "Paid"`); run `si demo test` |
| browse several kinds of content | a site | a folder with `_site.md` (a `navigator:`, a menu list; `logo:`, `favicon:`, `touch-icon:`, `og-image:` are opt-in branding), plain pages, plus articles, talks and guide folders |
| a résumé, a portfolio, a product or event page | a site of that kind | start with `si new <name> --type website --kind resume\|portfolio\|landing`, then replace the content; a landing hero is the page's title, `description:` and its own `cta:` |
| show code changing, step by step, in a window | a code show | fences with `window=mac` and the same `id=`, one per step, `<!-- callout: LINE text -->` after a fence |
| explore a topic and how its parts connect | a mind map | `type: mindmap`, a `# Title`, then a nested list of topics; or a ```` ```mindmap ```` fence on a slide or in a document; `view: 3d` (or `view=3d`) for pills on a floor with linked maps as floors above; `kind: bricks` (studded bricks) or `kind: toy` (cartoon island) for other 3D looks |

- **Guides:** write `_nav.md` first (front matter for the guide-wide
  settings, then a nested list: `- [Title](page.md)` pages, plain items as
  section headings). Each page has its own `# Title` and a one-sentence
  `description:`. Link to other pages by their `.md` file, with an anchor when
  useful (`install.md#step-2-start-the-stack`). Shared bands go in
  `_header.md` / `_footer.md`. Run `si check <guide>`: it reports missing
  pages, broken links and anchors, and pages `_nav.md` forgot.
- **Navigator layout:** `nav:` in `_nav.md`, `_site.md` or `_course.md` (or
  si.conf) places the menu: `left` (default), `right`, `top`, `bottom`, plus
  `sticky` / `not-sticky`, `accordion`, `collapsed`, `numbered` (RULES.md §20,
  §28). Do not hand-style the menu: themes draw it from the `--si-*` tokens.
- **Courses** (RULES.md §28): a folder with `_course.md`; top-level items are
  modules (`- Module 1: Title`), the links under them lessons. A lesson's kind
  comes from its target: a page is a reading, `type: deck` slides, a `.mp4` (or
  a `type: demo` with `app:`) a video, `type: lab` a lab, `type: demo` without
  `app:` an instructor demo, `type: quiz` a quiz; `· kind=…`, `· minutes=…`,
  `· prerequisites=…` after a link override. Give each lesson a one-sentence
  `description:` and, when the estimate is wrong, `minutes:`. Quizzes: `## `
  questions, `- [x]` right answers, `> Explanation: …`, `pass: 70%`; a
  ```` ```quiz ```` fence works in any lesson. A lab's answer goes between
  `<!-- solution -->` and `<!-- end -->`. A note only the instructor sees is
  `instructor: …` in a lesson's front matter, never in its text. Learner build
  `si build course/`, instructor build `--instructor`; `si check course/`.
- **Steps:** in a lab or a guide, `## Step 1: Start the stack` shows a
  numbered badge. Put the time and prerequisites in a key/value table at the
  top (`| | |` header, then `| **Time** | 10 minutes |`).
- **Every command block** is followed by what the reader should see, in a
  `text` block.
- **Code:** include real files instead of pasting them
  (`<!-- include: src/App.java tag=retry -->`; add `lang=xml` when the file's
  name does not say its language, e.g. `NuGet.config`; mark the region with
  `// tag::retry[]` … `// end::retry[]`), prefer `tag=` over `lines=`, explain
  lines with `<1>` callouts (a `// <1>` marker on the line, a `<1> text` line
  after the block) instead of prose about line numbers, highlight what matters
  with `hl=`, and put the same code in several languages in `<!-- tabs -->`.
  Code that lives in a repo comes from the repo: paste a GitHub **permalink**
  (`<!-- code: https://github.com/OWNER/REPO/blob/COMMIT/path#L5-L10 -->`), pin
  a commit (never `main`), and show a change with
  `<!-- diff: https://github.com/OWNER/REPO/commit/SHA path=src/x.go -->`.
  `si check --offline` in CI proves the page builds from the cache.
- **No raw HTML for common pieces:** a highlighted phrase is `==text==`; a
  status pill is `<!-- badge: text · ok -->`; a logo lockup is
  `<!-- logo: path/logo.svg × Partner -->` (or `<!-- logo: × Partner -->` for
  the theme's own logo); a QR code is `<!-- qr: https://… · caption=… -->`
  (never a pasted QR picture); a card grid is `<!-- cards: 2 -->` then `### Title · tag ·
  teal` per card. They work on slides and in documents (RULES.md §13, §23).
  `si fix <dir>` converts existing raw HTML.
- **Fonts:** never link a font stylesheet; set `fonts: heading=Family:600,
  body=Family:400;700` (Google by default; `bunny:`, `fontshare:`,
  `fontsource:` or a local `.woff2`) and si inlines it (RULES.md §15).
- **Sites:** pages are plain pages (no `---` slides); a talk inside a site
  says `type: deck`. Use the plugins (`alert`, `badge`, `event`, `accordion`,
  `modal` … `end`, `carousel`, `embed`) for interaction instead of raw HTML or
  script, and `embed:` to show an article, deck slides or a guide on a page.
  A `·` in a plugin's text starts an option; reword instead (RULES.md §23).
- **Books:** number nothing by hand: si numbers chapters, sections, figures
  (an image with a title) and tables (a `Table: caption` line after the
  table). Refer with empty links (`[](#fig-…)`, `[](02-x.md#section-id)`) so the
  numbers stay right, and mark index terms with `((term))` / `(((term)))`.
- **Mind maps** (RULES.md §24): one list item per topic, nested lists for
  children, short topic text (a few words). Link maps with `→ other.md` (one
  file per map; they are inlined and open with a zoom) or, in one file,
  `---` + `# Title {#id}` and `→ #id`. `→ slides: talk.md#7` opens a slide,
  `→ https://…` a page. Options go in braces at the end:
  `{badge=Thu collapsed pos=40,-12 branch-pos=0,30}`; `[!sticky] text` is a
  sticky note; a fenced code block inside an item is a code topic; `![alt](img)`
  puts a picture in a topic. Look: `layout:` map/right/org/radial, `lines:`
  curved/straight/orthogonal/tapered, `nodes:` box/pill/underline, `map-theme:`
  auto/harbor/blueprint/ink/garden, `flow: on` for dots along the lines (a
  fence: `layout= lines= nodes= theme= flow=`). On a slide, ```` ```mindmap steps ````
  reveals one first-level branch per click: write one `[CLICK]` per branch in
  `Notes:`. Exports: `si build map.md --svg --png --pdf`; other tools:
  `si mindmap import file.opml|.mm|.xmind|.mmd`, `si mindmap export map.md --to
  opml|freemind|xmind|mermaid`. The Studio edits maps visually (Map view);
  the Markdown it writes is the same you would write by hand.
  Keep one idea per branch and at most about seven branches per map; move
  detail into a linked map instead of a deep tree.
- **Code windows** (RULES.md §25): `window=mac|windows|terminal|none` on a
  code fence draws it as a window (`title="file.go"`, `theme=night|paper|dusk`,
  `backdrop=dusk|ocean|mint|graphite`, `numbers`, `hl=`); callouts go right
  after the fence, one per line: `<!-- callout: LINE text at=dx,dy -->`. Fences
  in a row with the same `id=` (and `step="name"`) are a show whose steps morph;
  on a slide each extra step is a click (one `[CLICK]` each), and the same id on
  consecutive slides morphs between them. Keep each step small (what changed is
  the point) and use callouts for the why, not to repeat the code.
  `si code file.md --png --svg --gif` exports them. Leave `at=` out unless the
  author placed the note: dragging it in `si play` or the Studio writes it.
- **Publishing** (RULES.md §26): `linkedin:` (the post text), `tags:`,
  `canonical:` and `cover:` (a link post's picture) in front matter feed `si publish` (LinkedIn, dev.to, Hashnode, Ghost, WordPress); it is a dry run unless `--yes`.
  Never run it with `--yes` for the author.
- **Web demos** (RULES.md §27): each `##` is a feature run in a fresh browser; steps
  name elements as a person reads them (`click "Sign in"`, `type "demo" into
  "Username"`, `#id`, `[data-testid=pay]`, `role=button "Pay"`). Every feature
  ends with an `expect` (`text`, `url`, `"X" visible|hidden|disabled|enabled|checked`,
  `no errors`). Secrets are `$VAR` from the environment, never written in the
  script. `Say:` is narration spoken to the viewer (the teleprompter rule), never
  a direction. Run `si demo test <script>`; when a step fails, read expected,
  found and the closest elements in the output, and fix the step or the app.
  `si demo run <script>` films the same steps: keep each `Say:` short (one or
  two sentences: the words set the pace), use `note "…" at "Field"` for what
  the viewer must read (a demo login), and `pronounce: word = how to say it`
  for names the voice gets wrong. It writes one file, `<script>.mp4`; add
  `--with` (or `with:`) only for what the author asked for (`captions`,
  `subtitles`, `clips`, `report`…), and `--with report` when you need the
  report to debug a run. To film one panel, not the page, add `region:
  ".panel"` (or a feature's `Region:` line, `--region`, `--crop x,y,w,h`);
  keep notes inside that region, or they are cut. Leave `voice:` out unless the author asks for
  one (si picks Kokoro af_heart, else the computer's voice); never put an API
  key in the script or in a project si.conf (si ignores it there): the author
  runs `si voice login openai|elevenlabs`. Kokoro installs with `si voice
  install kokoro`; `si voice say "line" --voice kokoro:af_heart` tries a voice.
- **Web demos with a terminal** (RULES.md §27, Terminals and programs): a
  server the app needs goes in front matter `start: <command> · wait for
  "ready text"` (or `· wait for port N`), never in a `run` step of a feature
  the others depend on. Show a command with `terminal "Name"` then `run
  "command"` (`· wait` when it must finish first; `· exit 1` or `· fails` when
  it must fail, an error on purpose) and check it with `expect output "text"`; write `- browser` before any browser step that follows a
  terminal. A workload for one feature is `start "…" as NAME`, then `stop NAME`
  and `expect NAME exited 0`. Every program is stopped for you; do not add
  `kill` steps. Run `si check` (it follows the scenes), then `si demo test`;
  a failed `expect output` shows the terminal's last 40 lines.
- **Web demos, before and after** (RULES.md §27, Before and after; Scripts
  from anywhere): the run's own set-up goes in front matter, in the order it
  must happen: `start:` (a program), `prepare:` (alias `step:`, a command that
  must exit 0; `· within 5m` when it is slow), more `start:` lines; `setup:`
  runs before each feature, `stop:` after the last (programs still up),
  `cleanup:` last (removes what was made). They always run, so never add
  clean-up steps to a feature. Checks before or after the features go in `##
  Before` and `## After` (never filmed, not features: no `Say:` there). A
  script from elsewhere is always pinned: `gist:ID@REV`,
  `github:owner/repo@TAG/path.sh`, `https://… · sha256 HEX`; never write one
  without its pin, and never add `--trust` to a command a person runs (CI
  only).
- **A talk that sets itself up** (RULES.md §14): a deck that shows something
  live sets it up in front matter, `setup:` (must finish) and `start: <cmd> as
  NAME · wait for "ready"` in the order needed, `stop:` and `cleanup:` to tear
  down; `si play` and `si serve` run them, `si build` never does. Keep them
  local commands unless the author names a pinned source.
- **Web demo videos that edit themselves** (RULES.md §27, Transitions, camera,
  spotlight, windows and drag): every scene change already fades (0.4 s); set
  `transition: slide` (or `cut`) in front matter, or `· pop` after one `go`,
  `browser`, `terminal "…"` or `window …` step, only when the author asks.
  Use `zoom to "Element"` … `zoom out` for the one or two moments the viewer
  must read something small, and `spotlight "Element"` (`· label "…"`) when
  the narration points at one control; both are checks in `si demo test`.
  `window "name"` films a popup or tab the app opened (`window main` comes
  back); `drag from "A" to "B"` (or `from X,Y to X,Y`, `by DX,DY`) is a real
  drag. `What it does:` is shown as written: start it lower case only when the
  word is (si).
- **Sites** take opt-in branding in `_site.md`: `logo: brand/mark.svg Name` (a
  picture with the name), `favicon:`, `touch-icon:`, `og-image:` (an https URL)
  and `version: v1.2.0 → <release notes URL>`. Never add si's own logo to a
  page you build for someone. A page only a button or a link should reach (a
  contact page behind `cta:`) takes `unlisted: true`: built as a full site page
  (same navigator, header, footer, shell and theme as the listed pages),
  linkable, only kept out of the menu; a page in no menu and not unlisted is an
  orphan and is not built.
- Problems with si itself: `si report` prints a prefilled GitHub issue link;
  `si --version --verbose` shows the build, links and credits.
- `si --export example --type guide` (or `article`, `lab`, `book`, `course`, `all`)
  writes a working example to copy from.

## Slide content

- Slides carry content, never instructions: no "press the arrow", "this slide
  demonstrates the two-column layout". A deck is a real talk.
- One idea per slide; a claim as the heading beats a topic label. No wall of
  text: prefer a layout (`two-col`, `big-number`, `quote`), an inline SVG
  diagram coloured with `--si-*` tokens, a `stage` flow, or a short code block.
- Keep it fluid: image sizes in `%`, `em` or `cqi` (never `px`); SVGs need a
  `viewBox`. The layout reflows from a phone to 4K by itself.
- A deck is self-contained: local images, fonts and media are inlined at
  build time. Do not link external scripts, fonts or images.
- Do not edit the generated HTML; change the markdown and rebuild.

## Before you finish

1. `si check <dir>` reports no problems.
2. Look at the result: `si build` and screenshot a few slides headless
   (`chrome --headless=new --screenshot=out.png --window-size=1440,900
   file://…/deck.html#3`), at a wide and a narrow size. A deck that passes
   `si check` can still look wrong.
3. Read the `Notes:` of every slide you touched aloud in your head: if a line
   tells the speaker what to do instead of saying something to the audience,
   rewrite it, or move it to `Points:`.
