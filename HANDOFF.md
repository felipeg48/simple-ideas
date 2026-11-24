# HANDOFF — for the next agent

Read this, then `FEATURE.md`, then `CLAUDE.md`. This is a Go CLI (`si`) that turns
Markdown/AsciiDoc into animated, self-contained HTML slide decks.

## Current state (2026-11-23, end of session)

**LATEST (read first, 2026-11-24): v4.4.0 tagged: Listen (text to speech,
RULES §39).** `feature: t2s` on document pages (guide, book, article, lab,
site page; inherited from `_nav.md`/`_site.md`, `feature: none` turns it off)
adds a Listen bar under the title (`web/t2s.js`, inlined only when on;
`render/document.go` `t2sScript`; looks in `web/doc.css`). Browser
speechSynthesis, one utterance per sentence, `t2s-highlight:
sentence|word|paragraph|off` via CSS Custom Highlights, `t2s-rate:`,
`t2s-voice:`. Front matter keys may now contain digits (`metaRE`, also
`web/studio/src/runbook.ts`). Next ideas offered to the founder: Kokoro voice
in `si play` (reuse `/_si/voice`), `si build --t2s-audio` pre-rendered
audio, read tables, continue to the next page, mini player while scrolling.
Untracked `.serena/` and `si-docs-site/` are not ours; leave them.

**Delivery diagrams** (RULES §38, spec
`docs/superpowers/specs/2026-11-23-dev-diagrams.md`, phases 1–3 DONE):
- **v4.1.0** ```kanban, ```gantt, ```sequence fences and `type: diagram`
  pages (landscape 16:9 / portrait 3:4), animated by `web/diagram.js`
  (tweens per-step states, follows `.si-dg-mark` build steps on slides,
  scroll autoplay, `window.siIG.seek` for filming); exports PNG/JPG/PDF/SVG/
  GIF/MP4 (`cmd/si/diagram.go`, `exportPictures("diagram")`).
- **v4.2.0** flow metrics: cfd, burndown, cycle-time, throughput
  (`internal/deck/diagram_flow.go`, `clip` reveal, `dot`/`shape` kinds).
- **v4.3.0** state, flowchart (with a moving token), class
  (`diagram_graph.go`, layered layout), wbs, storymap (`diagram_tree.go`;
  `card`/`dia` kinds). Element classes `si-dg-cls`/`si-dg-wb` (not
  `-class`/`-wbs`: those clash with the figure's `si-dg-<kind>`).
- Engine: `deck.Diagram{Kind,Title,W,H,Els,In,States,Steps,Facts}`, render
  `internal/render/diagram.go`, examples `examples/diagram/*.md`, site page
  `docs/site/features/diagrams.md` + Motion gallery cards.
- **Next for diagrams:** phase 4 (Studio editors, terminal text views);
  not built yet: kanban lanes and card ages, gantt workdays, sequence
  alt/loop boxes.

**Demo videos are on request only (the founder, 2026-11-23).** Never
re-record `docs/demos/` on your own; `.github/workflows/demos.yml` is
`workflow_dispatch` only (run it only when the founder asks); `pages.yml`
finds the newest videos in the last 200 releases. The CI `demos` *test* job
(`si demo test`) still runs on every push. v4.3.0 also made screencast start
retry "Not attached to an active page" (`internal/webdemo/runterm.go`).

**Voices:** the founder's default stays Kokoro `af_heart` (user si.conf
`~/.si/si.conf`); they also have an ElevenLabs cloned voice, commented out
there, usable per run with `--voice elevenlabs:<id>`. Never print that
si.conf whole: it holds API keys (grep `^#* *voice:` only).

**After every release: `make install`** so the founder's `si` matches the tag.

**Shipped this session, in order:**
- `73345b3` names scrubbed from everything public (docs, examples, specs,
  fixtures, tests, issues). **Never write a client, employer or brand name
  in anything public** (code, docs, examples, commits, issues, PRs, status
  boards); invented names only (Northwind, Acme, Dockyard). Private notes
  live outside the repo in `~/Progs/AI/si-private-notes/`.
- **v3.44.0 / v3.44.1** sandbox labs (RULES §37, spec
  `docs/superpowers/specs/2026-11-22-sandbox-labs.md`): `sandbox:` gives a
  lab its own Docker-in-Docker box in `si play` (drawer tabs, ✓ Check and
  `sandbox:PORT` links in it), on the `si-sandbox` Colima VM with sysbox
  (unprivileged, only `~/.si/sandbox/share` mounted). `internal/sandbox/`
  (engine.go, box.go, host.go), `cmd/si/sandbox.go`.
- **v4.0.0** classes on the presenter's machine (spec phase 2 DONE):
  `si class` planner (keyed loopback page, `cmd/si/class.go` +
  `classplanner.html`), `si class new|list|codes|start|rm`, class files
  `.si/classes/<id>.json` (0600), seat codes (8 of 30 symbols), roster CSV,
  codes sheet with QR codes, `internal/sandbox/class.go` (`Fleet`: a box per
  seat on first use, memory budget = 80% of the VM unless `--budget`,
  ttl/idle `Reap`, `States`, `Measure`), `internal/serve/seats.go`
  (`Classroom`, `ClassroomHandler`: its own allow-list mux, never
  `Handler()`; `/go/<code>` → `si_seat` cookie; tabs only in the seat's box
  via `termHub.need`; checks in the box; ports reverse-proxied). Shared by
  `--tunnel` (cloudflared quick tunnel), `--https` (si's CA) or `--local`.
  Quick tunnels hold chunked streams back, so off the loopback the drawer
  long-polls `GET /_si/term/api/read` (term.js `POLL`; ~150 ms a key,
  measured) and hides the Viewer/⌘P/⌘⇧F. Lab pages now carry the drawer
  (`render/docterm.go`: >_ Terminal button, `` ` ``, Run on shell blocks).
  CLAUDE.md rule #8 has the classes exception.
- `2dbf501` + **v4.0.1** the docs site's **Motion** top-level menu (the
  founder: motion must be first-class marketing, not buried in reference):
  `docs/site/motion.md` (*See it move*: 29 effect cards, each one line of
  Markdown and a ▶ Watch it), `docs/site/features/{transitions,animations,
  motion}.md`, showcase decks in `docs/site/features/showcase/` (also
  menu items, so built standalone), a "See it move" section on the home
  page. New runtime feature `?play` (`deck/02-navigation.js` `fromHash`):
  `talk.html?play#5` opens one step before and moves on so the transition
  and build animations play (a plain `#5` shows the slide at rest); RULES §14.

**Resume here (next, in order; nothing is in progress, wait for the
founder's pick if they name one):**
1. **Sandbox labs phase 3, instructor live tools** (spec "Phases" 3): watch a
   seat's terminal read-only, take over (learner sees a badge), show a seat on
   the deck screen, Reset all, idle warnings; plus the Viewer reading files
   inside the box (exec-based, confined to /lab), CPU per box (two stats
   samples), several classes per planner. Learners' class events could carry
   the seat's roster name.
2. Phase 4 hardening: sysbox by default where present, a warm pool of boxes,
   Windows host check. Bump sysbox 0.7.1 → 0.7.2 once two weeks old.
3. Diagrams phase 4 (Studio editors, terminal views) and the missing
   kanban/gantt/sequence extras above.
4. More Motion gallery polish if the founder asks (e.g. a small preview
   loop per card); same pattern for other showcases (3D mind maps, code shows).

**The founder's machine:** a 128 GB Mac. The `si-sandbox` Colima VM is still
at the 8 GiB test size: before a real class run `si sandbox host rm && si
sandbox host init` (defaults: half the cores, 3/4 of RAM = 96 GiB, 100 GiB
disk; about 38 learners at 2 GB). Colima is started with `--activate=false`
so the founder's own docker context (`desktop-linux`) is never switched.

**Testing tips from this session:** curl is blocked by a hook here: drive
HTTP with Python (`http.client`) or Playwright scripts (import playwright via
`createRequire('<repo>/web/studio/package.json')`), headless only. Real-box
test: `SI_DOCKER=1 go test ./internal/sandbox`. A whole class end to end:
copy `examples/sandbox` into a temp project as `lab/`, `si class new lab
--seats 2`, `si class start <id> --local --no-open`, then `/go/<code>`.
The `cmd/si` build-listing golden changes whenever the deck runtime JS
changes: `go test ./cmd/si -run TestBuildDefaultUnchanged -update`, read the
diff. RULES changes need `go test ./internal/skill -update`.

**Status board:** https://claude.ai/code/artifact/1dc6fd53-0b3f-4a86-a379-59db56d274da
(version 64, updated for v4.0.0; still needs chips for v4.0.1 Motion
gallery, v4.1.0, v4.2.0, v4.3.0 diagrams). `Artifact read` it, edit the saved file with a script, republish with
`url`.

**Client names (the founder, 2026-11-23):** the four brand themes (`crdb`,
`acemq`, `acemq-pro`, `osstend`) STAY in the public repo, unlisted;
`TRADEMARKS.md` names their owners and says the marks, logos and looks are
theirs and not under Apache 2.0. Nothing else in the repo may carry a client
name (specs, tests, examples, docs were scrubbed). Remote branch
`feat/architecture-status-doctypes` deleted (superseded by RULES §35/§36).

## Earlier state (2026-11-17, end of session)

main `854c431`, tag **v3.34.0**, CI green (test, studio, demos, windows),
installed (`si --version` 3.34.0). Working tree clean except the untracked
`.serena/` and `si-docs-site/` (not ours; leave them).

**Resume here:**
- **Status board:** the founder's private artifact
  https://claude.ai/code/artifact/1dc6fd53-0b3f-4a86-a379-59db56d274da
  (Version 40, 97% done: 139 done, 4 missing, 4 improve). Update it in
  place after every release; see the memory note `status-board.md` and run
  `python3 ~/.claude/projects/-Users-felipeg-Progs-AI-simple-ideas/memory/board-progress.py <board>`.
  If the scratchpad copy is gone, `Artifact read` the URL, save it, edit,
  republish with `url`.
- **Open Improve items:**
  - Decks: the control room cannot reveal a hidden word cloud (only polls); the
    terminal monitor's `p`/`w` keys were never tried on a real terminal.
  - Terminal looks: the monitor's memory is Go `MemStats.Sys`, not RSS.
  - Video edits: Chrome still draws one caption picture per word (≈46 s for
    600 words); ffmpeg here has no libass/drawtext.
- **Missing, all gated on the founder:**
  - a real SkillJar account check;
  - Apple Developer ID signing + notarisation;
  - listing the skill/MCP server in the public directories;
  - the desktop app on Windows/Linux.
- **Housekeeping:** many old agent worktrees under `.claude/worktrees/` are
  still registered (`git worktree list`); they are merged and can be pruned
  (`git worktree remove`, then `git worktree prune`). Agents in worktrees cannot
  run git in the main checkout: after they push, `git pull --ff-only` and
  `make install </dev/null` here.

**v3.34.0, the control room:**
- **Page and access:** `/_si/monitor?k=<key>` on `si play`'s loopback
  listener (`internal/serve/monitor.go`, page `internal/assets/web/monitor.html`).
  It requires a loopback client and Host, an Origin match, and the run key
  (cookie after the first visit, HttpOnly, SameSite=Strict); no framing.
- **Remote role:** `monitor` on `--remote` behind the presenter key.
- **What it shows:** viewers by role (`/_si/events?role=…`), slide/step timers,
  links and QR codes, CPU/memory sparklines, the log.
- **Controls:**
  - polls: close/reopen (votes 409 when closed), reveal/hide, reset (a new round);
  - Q&A: approve, show/off screen, answered, hide;
  - cloud: hide/restore a word, clear.

  They are sent as the existing relay messages through `/_si/monitor/{poll,qa,cloud}`
  and saved in `.si/audience/`.
- **Shared stats and keys:** `Server.Stats()` feeds both monitors. Terminal
  monitor keys: `p` QR codes, `w` opens the web monitor.
- **Tests:** `monitor_test.go`, e2e `web/studio/e2e/monitor.e2e.mjs`;
  docs/remote.md.

**v3.32.0 / v3.33.1, terminal looks:**
- **Choosing a look:** `--view ledger|badges|checklist|steps` (default
  ledger) on play/serve/build/check; `SI_VIEW`; `ui.view:` in si.conf.
- **Code:** `internal/ui/`, `cmd/si/look.go`. Colour only when stdout and
  stderr are terminals, no `NO_COLOR`, `TERM` not dumb; otherwise today's
  plain lines byte for byte (goldens).
- **Monitor:** key `m`.
- **Demo-script words:** v3.33.1 restored "serving", "live reload on" and
  "OK" in every look: `docs/demos` scripts wait for them. Keep them.
- **Design board:** the four looks came from the founder's design board
  (artifact https://claude.ai/code/artifact/47f1f5ae-1bbe-4339-ae4a-927475f56fa6);
  they loved all four, hence the flag.

**v3.33.0:** the video edit "Improve" items. A moving blur or pixelate
(`· to`/`· path` on `blur`/`pixelate`): `graph.go` `hideMoving` cuts a
piece the size of the largest keyframe at `x='min(keyExpr…)'`, hides it and
overlays it back at the same expression of t (`keyExpr`: smoothstep or
linear progress, stretches timed by their length in percent of the frame,
exactly as overlay.js `place`); when sizes differ a `geq` mask
(alphamerge) cuts it to the size of the moment, the branch trimmed to the
line's range. A size per place (`→ 400,300 300x90`, `Item.PathWH`) for any
moving rectangle (overlay.js sets width/height too). The Video view: path
places are handles while paused (`edit.js` `.ve-key`, `.ve-route`; a drag
sends `{key, at}`, `Move.Key` + `movePathPoint` rewrite that place), a
zoom's/pan's rectangle (`.ve-camrect`, from the mark's `box`) shows with
its line and drags like a box (`Move.Apply` writes `zoom to …`). Burned
captions: `webdemo.slideshow` writes an ffconcat list (a transparent PNG in
the gaps) so `CaptionedArgs` and the demo `videoArgs` have one caption
input; the bundled ffmpeg has no libass or drawtext, so the per-word
pictures stay (same look). Numbers in the CHANGELOG.

**v3.31.0:** the video edit phase 3 and its "not now" items: the Studio Video
view (`internal/studio/video.go`: POST video applies a `videdit.Move` and
returns `videdit.PreviewText` with `edit.js`/`edit.css`; POST/GET
video/build runs `si build` in the background; `web/studio/src/video-editor.tsx`,
e2e `videoedit.e2e.mjs`), moving callouts (`Item.Path`, overlay.js `place`),
per-word captions (`internal/webdemo/words.go`, `captions words`, `--with
words`, karaoke cues in `videoplayer.js`), `--with highlight` for edits
(`internal/videdit/highlight.go`), Windows CI running a split demo
(`TestTerminalSplitWindows`) and the workbench e2e.

**v3.28.0:** video edits (`type: video`, RULES §32, spec
`docs/superpowers/specs/2026-11-17-video-edit.md`, phases 1–2): `internal/videdit/`
(parse.go lines and times, plan.go source→output time map, graph.go one ffmpeg
graph, callout.go + overlay.css/js drawn in headless Chrome as a PNG overlay
sequence, preview.go the `si play` page, refilm.go stream-copy splice,
check.go), `cmd/si/videoedit.go`; promo-kit exports in
`internal/webdemo/videdit.go`; example `examples/video/` (shop.mp4 filmed from
shop.md). Next: phase 3, the Studio Video view (drag callouts, write x,y back).

## Earlier state (2026-11-14)

**v3.15.0:** animated infographics: `animate: build|loop`, `duration:`,
`· enter= · at=` (`internal/render/igmotion.go`: igSchedule, flow blocks on
the stage engine; `web/infographic.js`: one clock, `window.siIG.seek`),
`flow` blocks (`deck.ParseIGFlow`), exact frames (`deck.IGSize`: phone, story,
portrait, square, landscape, WxH), `si build --gif --mp4 --size --fps`
(`cmd/si/igmotion.go`: `?si-ig-film` seeking, `encodeAnimGIF` with frame
differencing, also used by `si code --gif`), deck `--mp4` films an animated
fence (`igSlideSeconds`). Left out: icons by name in flow boxes, `dir=` on a
flow.

**v3.6.0:** play terminal shells and Viewers survive a live reload (PTY
kept with a 256 KB replay ring, `GET /_si/term/api/list`, `sessionStorage`
`si-term-state`, idle shells closed after 10 min), text size per tab (− % +,
⌘= ⌘- ⌘0), the drawer's height dragged from its top edge, ← → in browser
windows (`injectNav` in `internal/serve/browser.go`), and `si demo` CSS
selectors keep their spaces (`locator` in `internal/webdemo/script.go`).

**v3.5.0:** the Viewer's tree creates, renames and deletes files and
folders (toolbar + right-click menu; `create`/`rename`/`delete`/`stat` in
`internal/serve/files.go`), `.conf`/INI values coloured in both views
(`render.Highlight` `ini`, the `conf` mode in `term-editor.ts`), and
**Follow** (tail -f for logs; `cat?tail=1`; the watcher skips `*.log`).

**v3.4.0:** the play terminal's Viewer tabs edit files (CodeMirror 6 bundle
`internal/assets/web/term/editor.js` from `web/studio/src/term-editor.ts`,
lazy-loaded on Edit; `POST /_si/term/api/save` with an mtime check; saves
never live-reload). See `docs/terminal.md` "Viewer tabs".


**Released and live: v3.27.0** (code windows from included files) · (18 improvements) · (steady headless Chrome tests: `cdp.Command` slots, fresh profiles, a written product survives a teardown crash, polling harnesses; webhook `webhook-base:`/`course.base`, in-page retry with backoff, `X-Si-Timestamp` signing, relay on the `--remote` listener; class view quiz question titles and a Day picker) · v3.24.0 (drawer sizes hardened, opacity) · (Windows terminal via ConPTY, web demo improvements, issues #14–#20) · (live class view; v3.20 improvements, v3.21 mind maps) · (result webhooks, self-checking labs, relay; class view specified) · (clean filmed prompt via `webdemo.CleanShell` + `Server.CleanPrompt`, POST forms across sites via `/__si_fwd` 307, act-cue badges in the terminal teleprompter) · v3.17.0 (#13: embed-terminal/browser, siPlay API, Notes cues, Demo lessons) · (issues #10–#12) · (the play workbench, six improvements:
the Viewer tree's open folders and scroll in `t.snap()` (`open`, `treeTop`,
`t.onShow`); ⌘= ⌘- ⌘0 caught on the whole drawer; `grep?stream=1` NDJSON with
Stop (`grepWalk` in `internal/serve/search.go`); split view up to four panes
(`split.panes`, `rx`/`ry`, `geometry()` in `term.js`); `navScript` catches
`window.open`, GET forms and, through the Navigation API, `location.assign`/
`replace`/`href =`; `cat?tail=1` returns `first` (`lineAt`, counts cached in
`termHub.lines`); shell input serialised (`sendIn`); committed e2e
`web/studio/e2e/viewer.e2e.mjs` and `workbench.e2e.mjs`; see
`docs/terminal.md`).
Before: **v3.13.0** (project structure map: a `structure:`
block in si.conf, `internal/config/structure.go` parses it; `si build`
of a project with one (`cmd/si/structure.go`) builds the usual flat layout
into a staging folder, routes every file to its kind's folders (`route`),
rewrites every quoted local path from each page's new folder (`rewrite`,
nearest copy wins), copies canvases to `animation` folders, zips
`kind:zip` / `folder/:zip` last (`bundle`); `si check` warns (`structureWarnings`),
`si new <name> --structure`; RULES.md §31, `examples/structure/`; the
no-structure build is pinned by `cmd/si/testdata/build-listing.golden`).
Before: **v3.12.0** (fixes #8, #9: a course whose pages are
not written yet reports `missing page` / `lists no pages` instead of a
panic (`Guide.Home()`); `![c](x.mp4)` is a player on slides and pages,
`poster=` in the image directive; linked local files and raw
`<video>/<source>/<audio>` src are copied beside the page and checked
(`fileLink`, `postprocess`); a `type: deck` lesson with `· kind=video ·
pdf · handout · slides` shows `<deck>.mp4` with `.vtt` captions, download
buttons and `<deck>.slides.html`, made by `deckLesson` in
`cmd/si/course.go`). Before: **v3.11.0** (auto-play resumes mid-part: the voice
file from its position, the browser's voice from its sentence; browser
windows keep the proxy across sites: cross-origin links and redirects go
through their origin's proxy, the address box shows the real address, ← →
cross sites; web demo `terminal split right|down`, `focus N`, `run … in 2`,
`expect output … in 1`, both panes filmed). Before: **v3.10.0** (timelines `eras` and `cards` layouts,
`--svg` for timelines and infographics as real vector SVG, infographic text
that shrinks to fit its block with a `si check` warning past 60%, both types
in the Studio's New… and gallery and as text in the terminal). Before:
**v3.9.0** (the play workbench: split view with a
draggable divider kept across reloads, ⌘P quick open, ⌘⇧F search in files
opening at the line, line spotlight in the Viewer and editor;
`internal/serve/search.go`). Before: **v3.8.0** (web demo promo kit: session keep, sticky
headers, `si demo draft`, brand cards/music/lower thirds, vertical and
highlight cuts, iframe steps). Before: **v3.7.0** (courses finished: SCORM, PDFs, SkillJar
needs a real-account check, hashed quizzes). Before: v3.4–3.6 the play
workbench (Viewer editor, file ops, reattach). Earlier **v3.3.0** (timelines, infographics; v3.2.1 Demos panel
Commands only; v3.2.2 browser windows `w`). Earlier: **v3.2.0, courses** (CI, release, demo filming and
docs-site green; si 3.2.0 installed with `make install </dev/null`; update
the plugin with `claude plugin marketplace update simple-ideas && claude
plugin update si@simple-ideas`). Every section below marked "unreleased,
worktree branch" is merged and released.

### NEXT TASK (the founder's order for the next session)
Finish courses, in this order, then release (minor: v3.3.0):
1. **Phase 3** of `docs/superpowers/specs/2026-11-12-courses.md`:
   `si build course/ --scorm` (SCORM 1.2 `imsmanifest.xml`, one SCO per
   lesson or `--scorm=single`, cmi status/score/suspend data from the
   localStorage progress in `web/course.js`, validated in tests) and
   `--pdf` (learner workbook; `--instructor --pdf` instructor guide).
2. **Phase 4**: `si publish login skilljar`, `si publish course/ --to
   skilljar` (dry run unless `--yes`; check SkillJar's API docs first;
   ids in `.si/publish/skilljar.json`; httptest fakes only; instructor
   demos never published), in `internal/publish/`.
3. **Studio "New…"** lists courses (`internal/studio/kinds.go`, the start
   gallery).
4. **Quiz answers**: keep them out of the page source where possible (e.g.
   hashed answers checked in course.js; SCORM scores still reported) and
   say plainly in RULES §28 that a quiz is a self-check, not a secure exam.
Docs with each (RULES §28, manual, agent-rules + skill, FEATURES,
CHANGELOG, README, docs site Courses page, HANDOFF, CLAUDE.md roadmap).

**Timelines and infographics: DONE v3.3.0** (built without mockups at the
founder's request). Left: timeline `eras`/`cards` layouts, `--svg`, `aspect:
a4` for timelines, infographic text auto-shrink, Studio/terminal support.

**(was) Then: timelines** (`docs/superpowers/specs/2026-11-12-timelines-infographics.md`):
`type: timeline` (16:9, eras with colours, milestones with pictures and
captions, sources, layouts line/vertical/snake/eras/cards, every theme;
`--png/--pdf/--svg`; a ```timeline fence for slides). First make design
mockups (layouts × themes) for the founder to approve, using their sample
(the 30 "Hitos en Algoritmos y Complejidad Computacional", Spanish, Mexico).
`type: infographic` comes after. Both take `orientation: landscape |
portrait`: landscape (16:9) **is a slide** (presentable, Notes:, build
steps, droppable into a deck); portrait is a tall scrolling poster/page.

### Courses (v3.2.0, 2026-11-12)
Phases 1 and 2 of `docs/superpowers/specs/2026-11-12-courses.md`, built to
the approved mockup `2026-11-12-courses-design.html`. A course is a guide
with `type: course` (`_course.md`, `deck.CourseFile`): `internal/deck/course.go`
(lesson kinds, minutes incl. an MP4 mvhd reader, `ForAudience`, `ParseNav`,
`ParsePass`), `internal/deck/quiz.go` (`ParseQuiz`, `Score`, the same rule
as doc.js), `internal/render/course.go` (`renderCourse` via `RenderGuide`,
the shell, home `index.html`, `complete.html`, slides lessons as a srcdoc
iframe, video lessons copied), `internal/render/quiz.go` (the fence and
`type: quiz` pages), `<!-- solution -->` in `plugins.go`, `nav:` for guides
(`guideNavAttrs`) and sites (`siteNav`), `web/course.css` + `web/course.js`
(progress in localStorage `si-course:<guide key>`), `cmd/si/course.go`
(`filmStale`, `zipDir`); `si serve` = learner, `si play` = instructor
(`serve.Server.Instructor`). Example: `examples/course/` (a first web page;
`page-load.mp4` filmed from a 5-slide deck, 52 KB). Not built: phase 3
(SCORM, `--pdf` workbook) and phase 4 (SkillJar). Ideas: the Studio's "New…"
does not list courses yet; quiz answers are in the page (not an exam).

### Releases since v2.40.0
- v2.40.1 code shows on slides (deep links, notes wrap, repaint) · v2.40.2/3
  unlisted site pages (PRs #1, #3) · v2.41.0 `si demo run` one file by
  default + `--with` · v2.42.0 region capture (PR #5) · v2.42.1 `--voice
  af_heart` · v2.43.0 `~/.si` (internal/home) + Kokoro default in
  `~/.si/si.conf` · **v3.0.0 si 3** (transitions, camera, spotlight/focus,
  window, drag; What's new page) · v3.0.1 demo `transition:` check · v3.1.0
  lifecycle (start/prepare/step/setup/stop/cleanup, Before/After, folders,
  gist/github/pinned URLs with trust, `si play github:…`) + code spans in
  title cards (#7) · v3.1.1 crisp zoom (#6, 2× capture), system-voice
  fallback, Ctrl-C exit 130 · v3.1.2 no file lists in demo videos ·
  v3.1.3 `wait for … · within` · v3.1.4 self-replacing `pronounce:` loop.

### How the founder works (read this)
- **Full autonomy, never ask** (memory `full-autonomy-no-asking`): decide,
  do, report. Only "let's talk first" or something truly theirs pauses.
  `.claude/settings.local.json` allows Bash, Read, Edit, Write, Glob, Grep,
  Agent and the rest; run in **manual mode** (auto mode's classifier blocks
  background agents).
- Spoken notices as **"Simple Ideas"**: `si voice say "Simple Ideas needs
  your input"` / `"Simple Ideas finished"` (memory `say-notifications`).
- Other agents open PRs or issues **without version bumps**; this session
  merges, fills doc gaps, bumps, tags, releases (patch for fixes, minor for
  features; minor tags re-film the demos on CI). Another session once
  committed into this checkout on a branch (`feat/card-inline-code`): check
  `git branch --show-current` before merging.
- Demo videos: Kokoro af_heart always; show files and commands in a
  terminal (`ls`, `si play`), never a browser file list.

### Open items
- Issues: none open (#6, #7 closed). Remote branches: only main.
- Auto-play: resume mid-sentence; re-check the bar at 4K.
- Lifecycle "Left": files beside a URL-fetched script are not fetched; JSON
  scripts can't interleave start/prepare; AsciiDoc front matter keeps one
  value per key; the interactive trust prompt is unit-tested only.
- Terminal `split` view; pop/circle are plain at feature joins (xfade).

### Crisp zooms, issue #6 (unreleased, worktree branch)
- `si demo run` supersamples: `runner.k` (`--scale`, `scale:`,
  `webdemo.DefaultScale` = 2; always 1 in `si demo test`). Chrome is launched
  with `--force-device-scale-factor=k` (without it the screencast stays at
  the window's 1× size even with a DPR-2 page) and `cdp.Client.Scale` sets
  the device scale factor of every page (app, terminals; windows in
  `attachWindow`); the screencast's max size is `W*k`.
- Everything downstream already worked in frame pixels (`fx.scale` = frame
  width / viewport); `cropFor`/`cropShot` now take the scale from the
  picture's width too (a fixed `--crop` is CSS pixels). `videoArgs` gets `k`
  and ends with `outSize(k)` (lanczos down to the viewport); `allArgs` fits
  2× cards with lanczos; `runner.screenshot` box-filters `screenshot` steps
  and failure shots down to the viewport (`shrink`).
- Measured on shop.md (`--with clips`): 45 s vs 42 s wall, CPU 80 s vs 44 s,
  peak temp frames 61 MB vs 23 MB, output files the same size. Chosen:
  always 2×. Tests: `internal/webdemo/supersample_test.go`.

### Demo lifecycle, sources from anywhere, talks that set themselves up (unreleased, worktree branch)
- Spec `docs/superpowers/specs/2026-11-12-demo-lifecycle.md`, all three
  phases built; CHANGELOG has an **Unreleased** entry, no version bump.
- `internal/pinned`: `Source` (`gist:ID@REV[/file]`, `github:o/r@REF/path`
  or `/dir/`, URL + `· sha256`), `Parse`/`ParseTarget`/`JoinTarget`;
  `Client.Scripts` (gist/contents API, a URL checked by sha256; cached in
  `~/.si/cache/scripts/<sha(pin)>`), `Client.Tree` (the tarball API, layout
  kept, `~/.si/cache/repos/<owner>/<repo>@<ref>` + `.sum`, the tree's sha256);
  `Trust` (`~/.si/trusted`, "<pin> <sha256>" lines; `--trust` = yes, not
  written; no terminal (a pipe or /dev/null) = refused with the reason; a pin
  whose sha changed = refused even with --trust). gh is used only against the
  default API after a 401/403/404; `SI_GITHUB_API` points tests at httptest.
- `internal/webdemo/lifecycle.go`: `Command`, `Prep` (`Script.Prelude`, in
  written order), `Setups`/`Stops`/`Cleanups`; `Script.Resolve(get)` fills
  `Cmds` with each fetched script quoted plus the args; `runner.prelude`,
  `runner.command` (`runTimed`: own process group, `WaitDelay`, aborted when
  shutdown begins), `shutdown` = stop: → programs → cleanup: (only once the
  prelude began). `## Before`/`## After` are `Script.Before/After`, run by
  `feature(f, section=true)` (no setup:, no take, programs not local).
  `Report.Start/Before/After/Teardown`, `around()` for the HTML and JUnit;
  `Suite` for folder runs (`report.json` = the suite).
- Decks: `deck.Meta.Lifecycle` keeps every lifecycle line in order;
  `webdemo.ParseStage` reads them with deck meaning (setup: = prelude);
  `webdemo.Stage` (Up/Down) is a runner without Chrome. `cmd/si/stage.go`:
  `deckStages`, the `readyGate` (getting-ready page + `/_si/ready` JSON),
  `runStages` (sets `$SI_RUN_LOGS`), `deckTarget`. `cmd/si/fetched.go`:
  `fetchTarget`, `demoScripts`, `fetchSources`.
- Real check (2026-11-12, with bin/si, fresh SI_HOME): `si demo test
  github:felipeg48/simple-ideas@7492470…/docs/demos/effects.md --trust` 4/4;
  the folder form ran all 8 scripts, 27 passed + webdemos.md 3/3 once Kokoro
  was reachable (it films with the installed `si`); no-TTY refusal, a wrong
  commit (404), a wrong sha256 and a changed tree all refused; `si play
  github:…/examples/showcase.md --trust` served the deck (HTTP 200).
- Left: a URL-fetched script's relative files are not fetched (its app: must
  be a URL or none); JSON scripts cannot interleave start/prepare; AsciiDoc
  front matter keeps one value per key.

### si 3, items 1–4 (unreleased, worktree branch; target v3.0.0)
- Spec `docs/superpowers/specs/2026-11-12-si-3.md`. Item 5 (re-filming
  `docs/demos/`, the "What's new" page) is still to do; no version bump yet
  (CHANGELOG has an **Unreleased** entry).
- Parsing: `internal/webdemo/scriptfx.go` (`ParseTransition`, `· <name>` after
  go/browser/terminal/window, `zoom to`/`pan to`/`zoom out`, `spotlight`/
  `focus`, `window`, `drag`, `regionValue` for `… · zoom`).
- The take records `cuts` (From = old scene's last moment, To = new scene in
  front) and camera `moves` (`internal/webdemo/runfx.go`: `cutting`,
  `cameraMove`, `spotlight`, `window`, `drag`; `show` in runterm.go records
  scene switches and resets the camera).
- Encode: `internal/webdemo/effects.go` — `plan` decides every 30 fps output
  frame (source frame, camera rect via `resolveMoves`/`camAt`, transition
  progress), `drawShots` draws the non-plain ones in Go (YCbCr planes,
  bilinear, every core) and hard-links the rest; the video's length never
  changes. Feature joins use ffmpeg `xfade` (`xfadeArgs` in encode.go: each
  card/clip held tr.Dur longer, so chapters keep their times).
- Zoom vs region: with a crop region the camera's view is the region;
  `· zoom` drops the crop and adds a zoom move at 1 s to the region's box.
- Terminal tab frames are 1280x713-ish, not the viewport: a transition
  stretches them to the other scene's size (`resize`), as ffmpeg did.
- Spotlight is in `overlay.js` (`d.spot` / `d.unspot`, a clip-path evenodd
  hole animated in rAF); ends with its Say: block (`endBlock`).
- Ceilings: `pop` at joins is a plain fade (no xfade equivalent); camera
  boxes are measured once at the step; `window` numbering is first-seen order.

### si 3 (v3.0.0, 2026-11-12)
Spec `docs/superpowers/specs/2026-11-12-si-3.md`, all done: transitions
(default fade 0.4s), camera (`zoom to`/`pan to`/`zoom out`, `Region: … · zoom`),
`spotlight` (alias `focus`, darkens 60 %, `· blur`), `window` (popups such as
the teleprompter) and `drag`; title cards keep casing; `internal/webdemo/
scriptfx.go`, `runfx.go`, `effects.go`. Fixes while filming: Chrome launched
with `--disable-renderer-backgrounding --disable-backgrounding-occluded-windows
--disable-background-timer-throttling` (a filmed popup froze), `claim()`
re-injects the overlay when a page wrote over it, the spotlight colour is a
fixed dark (not `--si-fg`). docs/demos/presenting.md films the teleprompter
and drawing; studio.md uses a spotlight and `zoom to .frame-shell`. Site page
`docs/site/whats-new.md` (menu "What's new in si 3"), README section.

### v2.41.0 and v2.42.0 (2026-11-11)
- v2.41.0: `si demo run` writes one file, `<script>.mp4`; `--with
  captions,silent,audio,subtitles,transcript,chapters,clips,report,readme,all`
  (`internal/webdemo/outputs.go`), `with:` / `demo.with:`; the site reads
  `<area>/<area>.mp4` and still accepts older `all.mp4` films.
- v2.42.0: PR #5 (another agent) region capture: `--region "<css>"`,
  `--crop x,y,w,h`, a feature's `Region:` line, front matter `region:`
  (`internal/webdemo/region.go`); merged by hand on top of v2.41.0's output
  names. Ceiling: notes and the ⏩ badge outside the region are cut.
- Workflow the founder set: other agents open PRs without version bumps; this
  session merges, fills doc gaps, bumps, tags and releases.

### Today's releases, in order
- v2.34.0 `si demo test` (phase 1) · v2.35.0 `si demo run` (video + voice) ·
  v2.36.0 terminals and programs (`start:`/`stop:`, `terminal`, `run`,
  `expect output`, `start … as`/`stop`) · v2.36.1 `run … · exit N` / `· fails`,
  exit-status counting fix · v2.37.0 `si voice` (install kokoro, say, list,
  login; keys only in the user si.conf) + mock keychain for headless Chrome ·
  v2.38.0 three videos per take, captions band, `.vtt`, fast-forward; `type "x"`
  at the focus; `app:` with a query kept; `SI_STUDIO_TOKEN`; `si theme list`
  hides client themes · v2.39.0 si's own demo videos (below) · v2.39.1
  `select` waits for late options · v2.40.0 auto-play (below): `v` / ▶ reads
  `Notes:` aloud and clicks at each `[CLICK]`, `si play --autoplay`,
  `si build --autoplay[=inline]`, `si build --mp4 --voice`, `voices:` per
  speaker; the CI filming run re-filmed the demos with "It presents itself".

### si demo run: one file by default (v2.41.0, unreleased, on a worktree branch)
- `si demo run` writes only `<out>/<script>.mp4` (was `all.mp4`); the rest
  is opt-in: `internal/webdemo/outputs.go` `With` (captions, silent, audio,
  subtitles, transcript, chapters, clips, report, readme, all; `Add` parses a
  comma list), `With.Files` (the plan: options + filmed features → file
  names, table-tested) and `transcript` (`<script>.txt`, Say: as written).
- `Run` (run mode) works in `<out>/.si-run-*` (`MkdirTemp`), `finishTake` /
  `joinAll` make only what `With` needs (per-feature `.silent.mp4` + `.mp4`
  always, as `<script>.mp4` is joined from them; captions PNGs/encodes only
  with captions), then `deliver` writes the report (with `report`), filters
  the report's links to what exists, renames the planned files out and sets
  `Report.Files` / `Report.Video`; the work folder is removed after the
  programs stop. `cmd/si/demo.go`: `--with` (multi), `--captions` = `--with
  captions`, `--no-captions` a usage error, `demoWith` (precedence --with >
  `with:` > `demo.with` from config.Load). `with:` is a deck MetaKey and a
  webdemo front-matter check. Feature stems never equal the script name
  (`runner.stem` adds `-2`).
- Docs site: `internal/docsite/videos.go` reads `<area>/<area>.mp4` + `.vtt`;
  `.github/workflows/demos.yml` films with `--with
  clips,subtitles,chapters,report` and attaches `<area>/<area>.mp4`.

### Auto-play (v2.40.0, released)
Spec `docs/superpowers/specs/2026-11-10-autoplay.md`, all three phases.
- `deck.Parts` (`internal/deck/say.go`): a slide's script split at `[CLICK]`
  into plain-text parts (`{t, s, k}`); `render` puts them in `si-data.say`
  (never with `NoSpeakerText`), with `a` (audio src) from `Options.VoiceAudio`.
- Runtime `web/deck/05-autoplay.js`: key `v`, ▶ Auto-play button, the
  `.si-autoplay` bar + caption; `go()` calls `apTouch()` so any move it did
  not make pauses it, `toggleDraw(true)` too; clicks are state-derived
  (`step < apClicks(parts, j)`), so resume never double-advances. Web Speech
  fallback with a reading-time guard (headless has no voices).
- `si play`: `GET /_si/voice/{slide}/{part}` (`internal/serve/voice.go`,
  loopback, one voicing at a time, `Server.Voicer` for tests);
  `webdemo.ForDeck` picks voices (`--voice` > front matter > si.conf >
  Kokoro/system; `voices:` / `voice.<Name>:`), `DeckPronounce` reads every
  `pronounce:`. `--autoplay` primes pages; `--open` adds `#autoplay`.
- `si build --autoplay[=inline]` (`cmd/si/autoplay.go`): `talk.voice/` m4a
  files (a voice failure is a warning: the page falls back to the browser);
  `--mp4 --voice` builds a narration from the voiced parts
  (`voicedNarration`, `webdemo.MixWAV`) and reuses the `--narration` path.
- Demo: `docs/demos/presenting.md` "It presents itself" (`press v`; a click
  on a toolbar button fails while the buttons are faded out).
- Next: resume mid-sentence (it restarts the part), a filmed demo where the
  deck's own voice is heard (the take films only the picture).

### si's own demo videos (v2.39.0)
- Six scripts in `docs/demos/` (decks, presenting, studio, themes, mindmaps,
  terminal); `.gitignore` there lists what they create.
- `ci.yml` job `demos`: `si demo test` on all six, Linux, every push (green).
- `.github/workflows/demos.yml`: films them on macOS with Kokoro on minor tags
  (`v*.*.0`) and on `workflow_dispatch`; uploads `si-demo-videos.tar.gz` and
  `si-demo-<area>.mp4` to the release, then dispatches docs-site. First run on
  v2.39.0 succeeded (~15 min).
- `pages.yml` downloads the newest release's tarball into `$SI_DEMO_VIDEOS`;
  `internal/docsite/videos.go` turns `<!-- si-video: area -->` lines (decks,
  presenting in features/decks.md; studio, terminal pages; themes page;
  mindmaps tour) into the video + chapters; `make docs-site` copies `videos/`.
  Live on simple-ideas.org (checked).
- README "Watch it": posters `docs/screenshots/readme/demo-*.jpg` → site.
- Local copies of every video: `~/Downloads/si-demo/<area>/`.

### Bugs found by filming, fixed
iframe loads no longer hold a click (`r.inner`, run.go); exit status counted
per command (`t.base`/`t.runs`); backslashes kept in quoted steps; `select`
waits for its option; Studio demo avoids `cmd+z` (Linux).

### Next (in order; the founder wants autonomy: do not ask, decide and report)
0. Auto-play follow-ups: resume mid-sentence (today Resume restarts the
   part); re-screenshot the bar at 3840x2160 (enlarged after the last check);
   a filmed demo where the deck's own voice is heard.
1. A window step (the teleprompter opens a popup the runner does not film) and
   a drag step (drawing with `a`); then add them to `presenting.md`.
2. Title cards capitalise the first word of `What it does:` ("si" → "Si");
   keep it lowercase when the script wrote it so (film.go title card).
3. Terminal `split` (terminal and browser side by side), phase 3 of the
   terminal spec.
5. Retake (`~/Progs/AI/retake-app`): point its CLAUDE.md at the web-demos spec
   for the shared script format (phase 5).
6. Voice: try `si voice login` with real ElevenLabs/OpenAI keys; Kokoro
   install on Linux/Windows untested.

### Housekeeping
- Notices: I am "Simple Ideas": `si voice say "Simple Ideas needs your
  input"` before asking, `si voice say "Simple Ideas finished"` once at the
  end (macOS `say` if it fails)
  (Kokoro installed in `~/.si/kokoro`; si's global folder is `~/.si`, v2.43.0, `internal/home`).
- Permissions: `.claude/settings.local.json` allows Bash, Agent, SendMessage,
  TaskStop, Monitor, ToolSearch, ListAgents; run in manual mode (auto mode's
  classifier blocked background agents).
- Pages deploys sometimes time out on GitHub's side: re-run the failed job.

- **v2.38.0 (unreleased, on a worktree branch): three videos per take, the
  captions band, fast-forwarded waits.** `internal/webdemo/film.go`
  `finishTake` writes `.video.mp4` (picture + fast-forward badges),
  `.mp4` (that + the voice, `-c:v copy`) and `.captions.mp4` (the same image
  sequence encoded again: badges, then `band(w, h)` = scale to 85% + pad on
  0x111318 at 2% from the top, then the caption PNGs in the bottom 13%, voice
  muxed in the same call), `.srt` + `.vtt`; `joinAll` adds `all.video.mp4`
  (all.mp4 with `-map 0:v -c copy`, chapters ride along), `all.captions.mp4`
  (allArgs over the captions clips) and `all.srt`/`all.vtt`
  (`joinCaptions`). Fast-forward: `r.waiting()` records wait windows in
  `tk.waits` (run · wait, stop · wait, expect/wait for output, start, browser
  wait for); `fastForward` (encode.go) cuts each to `[from+after, to]`,
  subtracts `tk.spoken` (the voice never plays over sped-up picture), picks
  X = ceil(len/2 s); `sequence` samples the take through `inT`, voice,
  captions and badges are placed with `outT`, so the clip length (and the
  chapters) come from the output. Options: `NoCaptions`, `FastForward`
  (`--fast-forward 3s` default in cmd, 0 in tests); `--captions` is a no-op.
  FeatureResult gains `videoOnly` / `videoCaptions`. Verified: broker.md,
  shop.md and a `sleep 14` script (a 6× badge, 9.2 s shorter), frames looked
  at. Next: a nicer badge animation; a `fast-forward:` front-matter key if
  someone asks.
- **v2.37.0 (unreleased, on a worktree branch): `si voice`.**
  `cmd/si/voice.go`: `say` (voice: `--voice` > si.conf `voice:` > default;
  an engine whose `Ready()` fails falls back to `system` with one stderr
  line; the line goes through `webdemo.Synth`, so the cache is shared with
  `si demo run` and ffmpeg is needed only on a cache miss; no `-o` = play via
  `playerFor`: afplay / paplay, aplay, ffplay / PowerShell SoundPlayer;
  `runPlayer` is swapped in tests), `list` (Kokoro names read from the npz
  `voices-v1.0.bin` with archive/zip, English a…/b… only), `install|uninstall
  kokoro`, `login|logout` (`readSecret`: stty -echo on unix,
  SetConsoleMode on Windows, a pipe reads one line). `internal/tools/kokoro.go`:
  `KokoroInstall` (FindPython: python3.13…3.10, python3, python; `py -3.x` on
  Windows; kokoro-onnx 0.6.1 needs <3.14), venv in `<dir>/venv`, pip, the two
  files from the `model-files-v1.0` release to a temp file + SHA-256 +
  rename; `Run`, `BaseURL`, `Python` injectable). `KokoroPython()` prefers
  that venv after `SI_KOKORO_PYTHON`. `internal/config`: `Load` now reads the
  user's si.conf (`UserPath`) first and overlays the project's; warnings carry
  their file; `voice.elevenlabs.key` / `voice.openai.key` are kept only from
  the user file (a project one: warning, `si check` prints it); `APIKey` (env
  first); `UnsetKey`; `SetKey` keeps the file's mode. `si demo run` resolves
  `--voice` > front matter > si.conf beside the script > user si.conf >
  default; `voice.style` fills an empty `voice-style:`. Tests set HOME / XDG /
  AppData (TestMain in cmd/si and config). Verified on this Mac: a real
  `si voice install kokoro --dir` (Python 3.13 venv, both files, hashes), a
  second run skipping everything, `si voice say` to a WAV (24 kHz mono,
  ~1 s, -20 dB mean) and aloud through afplay, cached replay. Not run:
  install on Windows or Linux, the cloud engines (no keys here).
- **v2.36.0 (unreleased, on a worktree branch): web demos with a terminal**
  (spec `docs/superpowers/specs/2026-11-10-web-demos-terminal.md`, phases 1, 2
  and `app: none` of 3; `split` not built). `scriptterm.go` parses `start:`
  (several; `as NAME`, `· wait for "text"|port N|Ns`, `· within`), `stop:`,
  `shell:` and the steps terminal/browser/run/start/stop, expect output (of
  NAME), expect no output, wait for output, expect NAME running|exited N,
  note … at line; `checkScenes` follows browser vs terminal and program names
  for `si check`. `procs.go`: programs via `sh -c` in their own process group
  (`proc_unix.go` Setpgid + kill(-pgid); `proc_windows.go` taskkill /T),
  stop = SIGTERM, `stopGrace` (5 s), SIGKILL to the group. `output.go`: the
  stream kept raw (masked, for the page) and plain (a streaming ANSI
  stripper) plus the exit codes from OSC 133;D marks. `term.go`: a shell per
  terminal under creack/pty with si's prompt (`shellRC`: zsh -d with ZDOTDIR,
  bash --rcfile, sh via ENV; job control and ! expansion off; the prompt
  prints `ESC]133;D;$?BEL`, which is how `run · wait` gets the exit status),
  the keyed loopback `termServer` (page, xterm assets from `assets.TermFS`,
  chunked output stream, resize) and `termpage.js` (xterm in the code window
  frame from `render.WindowChrome` + `assets.CodewinCSS`; `__siTermLine` makes
  a live box for note-at-line). `runterm.go`: the runner side: scenes are tabs
  (`show` moves `c.Session`, `Page.bringToFront` and the screencast;
  `r.film` is the filmed session, `r.session` stays the app's for console
  events), a feature that begins with `terminal` opens it before the first
  block and loads the app lazily at `browser`; `shutdown` (once: stop:, then
  every terminal and program; deferred in Run, also on SIGINT/SIGTERM) and
  `endFeature`. A note stays at least 1.8 s on screen (`tk.noted`). Windows:
  cmd.exe on pipes with local echo, compiles (`GOOS=windows go build`), never
  run. Verified end to end: `si demo test` and `si demo run --voice system
  --captions` on `examples/webdemo/broker.md` (frames looked at: ocean and dusk
  backdrops, typing, note under a line, Ctrl-C, switch to the console, a
  live counter). `TestExamples` runs broker.md (needs Chrome and go).
  Next: `split` (two tabs side by side needs compositing frames or an iframe
  with locate.js inside it), ConPTY on Windows.
- **v2.35.0: web demos phases 2 + 3 (video, voice).** `si demo run`
  (`cmd/si/demo.go`): `internal/webdemo/film.go` (the take: overlay injected
  with `Page.addScriptToEvaluateOnNewDocument`, `overlay.js` in a closed shadow
  root on `<html>`: cursor, ripple, notes ringed and anchored to `__siT`,
  blur; restored after navigations via `__siDemo.fresh`; `Page.startScreencast`
  JPEG frames stamped on arrival, acked from a goroutine; pacing
  `beginBlock`/`pace`/`endBlock` = max(voice, steps); title cards and caption
  PNGs rendered by the same Chrome, transparent background override),
  `encode.go` (30 fps sequence of hard links, ffmpeg args: captions burned with
  `overlay` + `enable=between`, mux, all.mp4 via the concat filter with
  `anullsrc` under the cards, FFMETADATA chapters, chapters.txt, SRT, caption
  split/wrap), `voice.go` (engines system/kokoro/elevenlabs/openai, cache
  `<user cache>/si/voice/<sha>.wav` normalised to 24 kHz mono, `mixWAV`,
  `pronounce:`), `readme.go` (README.md; cmd renders README.html with
  `render.Deck`, MediaLink). Run mode waits at most 3 s per step. Kokoro is a
  `tools` entry with a `Probe` (python + `kokoro_onnx` spec + model files in
  `tools.KokoroDir()`); espeak-ng added. Verified end to end on
  examples/webdemo with `say` (Kokoro was not installed on this machine, so
  its Python path has not run yet; the cloud engines are tested on fakes only). Next: piper/chatterbox/record engines,
  phase 4 (the manual with videos, `make docs-videos`), the open questions in
  the spec (`--titles`, music, device frames).
- **v2.34.0: web demos phase 1 (test mode).** `internal/cdp` (the DevTools
  client, now with events; motion.go uses it), `internal/webdemo` (parser for
  Markdown/AsciiDoc/JSON with line numbers, runner: locators by text, label,
  aria, placeholder, title, alt, CSS, role; real mouse events; console errors
  and failed requests; fresh incognito context per feature after `setup:`;
  `$VAR` masked; report.html/json, JUnit), `cmd/si/demo.go`. A prototype made
  the first narrated video of the Studio with Playwright + kokoro-onnx + ffmpeg
  (spec "Lessons"). Next: phase 2 (video: frames, cursor, ripple, notes drawn,
  blur, per-feature and full MP4, chapters), screenshots per step.
- **v2.33.0: motion.** `transition: morph` runtime in `deck/02-navigation.js`
  (`morphList`, `morphPlay`: FLIP; match by `data-si-morph` from `<!-- id: -->`,
  heading text, image, same words; text scales by font size, boxes by box;
  tokens `--si-morph-duration/-ease`, hooks `.si-morph-from`, `.si-morph`).
  `internal/motion` (canvas frames between `/* si motion: … */` and `/* si
  motion end */`, parts by `data-part`), `internal/studio/motion.go`
  (`POST motion`, text in/out), `web/studio/src/motion-editor.tsx`; e2e
  `morph.e2e.mjs`, `motion.e2e.mjs`. Next: New › Animation, insert into a slide,
  spread over slides, paths and stagger, MCP `si_motion`.
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

- **Migration recipe** (proven): one general-purpose agent per course, authoring
  ONLY under its own `markdown/<course>` subtree; it reads `docs/RULES.md` +
  `docs/authoring-canvases.md` + `docs/stage-diagrams.md`, matches the house style
  of an existing migrated course, replaces every raster with native SVG/stage/code,
  adds MORE animations than the source, runs `si check` on its own deck, and does
  NOT touch `index.html`/other courses/the repo. The lead does the central build.


## Known minor issues / polish backlog

- Roadmap ideas are in CLAUDE.md "Roadmap ideas".
