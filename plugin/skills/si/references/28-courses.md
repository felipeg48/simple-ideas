# 28. Courses

A course is a **folder with a `_course.md`**: modules of lessons in order,
each lesson something si already makes. Readings are pages, slides are decks
that present themselves in the page, videos are `.mp4` files or filmed web
demos, labs are `type: lab`, instructor demos are `type: demo` runbooks, and
quizzes are `type: quiz`. It builds into a website bundle with a home page,
progress kept in the learner's browser and a completion page.

```text
web-course/
  _course.md          modules, lessons, course-wide settings
  01-web/browser.md   a reading
  01-web/journey.md   type: deck, a slides lesson
  01-web/page-load.mp4  a video lesson
  02-html/first-page.md  type: lab
  02-html/inspect.md  type: demo, an instructor demo
  02-html/quiz.md     type: quiz
```

`_course.md` is front matter plus a list: the top-level items are
**modules**, the links under them **lessons**.

```markdown
---
title: A first web page
description: How a browser turns a file into a page, then write one.
level: beginner
nav: left · accordion · sticky
theme: studio
instructor-demos: show-placeholder
certificate: on
prerequisites: a text editor and a web browser
---

- Module 1: How the web works
  - [What a browser does](01-web/browser.md)
  - [From address to pixels](01-web/journey.md)
  - [A page loads, step by step](01-web/page-load.mp4)
- Module 2: Writing HTML · prerequisites=Module 1
  - [Build your first page](02-html/first-page.md)
  - [Inspect a page live](02-html/inspect.md)
  - [Check yourself](02-html/quiz.md) · minutes=5
```

### Lesson kinds

A lesson's kind comes from what it links to; `· kind=reading` (or `slides`,
`video`, `lab`, `demo`, `quiz`) after the link sets it by hand.

| Links to | Kind | The page shows |
|---|---|---|
| a page, an article, a guide page | Reading | the page |
| `type: deck` | Slides | the deck in the page, with auto-play (`v`) |
| a `.mp4`, `.webm`, `.m4v` or `.mov` | Video | a player; the file is copied beside the page |
| `type: demo` with `app:` | Video | its `<script>.mp4`, filmed by `si build` when missing or older than the script |
| `type: demo` with `app:` and `· kind=demo` | Demo | the same filmed video, for learners too, labelled Demo |
| `type: lab` | Lab | the lab; its task-list boxes can be ticked |
| `type: demo` without `app:` | Demo | the runbook (instructor build only) |
| `type: quiz` | Quiz | the questions, scored |
| `type: deck` with `· kind=video` | Video | the deck's narrated `<deck>.mp4`, filmed by `si build` when missing or older than the deck |

**Demo lessons for learners.** A filmed web demo (`type: demo` with `app:`,
such as one that shows the deck with `si play`'s drawer and a console over
it, §27) under `· kind=demo` is a learner lesson labelled **Demo** in the
menu and in the next-lesson line: the player with its captions, its done
tick following the video, as a Video lesson. So a course can say Video for
the narrated slides and Demo for "let's see it in action". A demo without
`app:` stays the instructor's live runbook.

```markdown
- Module 2: First messages
  - [Publish and consume](02/first-messages.md) · kind=video · pdf
  - [See it in action](02/in-action.md) · kind=demo
```

**Video lessons from a deck.** For self-paced learners a slides lesson can be
the deck's narrated video, with the slides to keep:

```markdown
- Module 3: Exchanges and routing
  - [Exchanges and routing](03/exchanges.md) · kind=video · pdf · slides
```

- `· kind=video` on a `type: deck` lesson shows `03/exchanges.mp4`. `si build`
  films it beside the deck (as `si build exchanges.md --mp4 --voice` would:
  narrated from its `Notes:` with `--voice`, else the deck's or si.conf's
  `voice:`) when it is missing or older than the deck; `--no-film` skips it.
  The deck's time is the newest of its file and the files its `<!-- include:
  -->` lines bring in (sections, code), so editing a section re-films it;
  pictures and canvases it uses do not count (touch the deck to re-film).
  The same build writes `exchanges.vtt` captions from the narration's words and
  timing, which the player shows (`<track kind="captions" default>`, the deck's
  `lang:`, else `en`).
- `· pdf` adds a **Download the slides (PDF)** button under the player
  (`exchanges.pdf` beside the deck, printed by the same build); `· handout`
  a **Download the handout (PDF)** one (`exchanges.handout.pdf`, the deck's
  `handout:` layout).
- `· slides` adds an **Open the interactive slides** link to
  `exchanges.slides.html`: the deck, built, in no menu.
- The done tick follows the video (watched to 90 %, or to the end).
- `· pdf`, `· handout` and `· slides` on any other lesson fail `si check`.
- **Poster.** A video lesson opens on a still, not on black: a deck's
  `exchanges.poster.jpg` is its first slide (1920×1080), written by the same
  build that films it; a `.mp4` lesson's `<video>.poster.jpg` is its frame at
  0.5 s (ffmpeg). `· poster=assets/cover.jpg` (beside `_course.md`) or
  `poster: cover.jpg` in the deck's front matter (beside the deck) picks
  another picture; `· poster=off` leaves the player without one. A missing
  poster file fails `si check`, and so does `· poster=` on a lesson that is
  not a video.

**Minutes.** A lesson's `minutes:` front matter (or `· minutes=` in
`_course.md`) sets its length; otherwise si estimates it: reading time at 230
words a minute, a deck's `(N min)` notes, a video's length, five minutes a
lab step, a minute a quiz question, a demo's `duration:`. Modules and the
course add them up; `duration:` in `_course.md` overrides the course total.

**Prerequisites** on the course (`prerequisites:`), a module (`·
prerequisites=…`) or a lesson (front matter) are shown, not enforced.

### Two audiences

- **Learner** (`si build course/`, `si serve course/`): readings, slides,
  videos, filmed demos, labs, quizzes. Instructor demos (no `app:`) are left out of the menu, or with
  `instructor-demos: show-placeholder` shown as "Your instructor shows this
  live". Decks carry no speaker notes; lab solutions are folded.
- **Instructor** (`si build course/ --instructor`, `si play course/`):
  everything, plus each demo's runbook, the decks' teleprompter (`p`), lab
  solutions open, and an **instructor box** above any lesson whose front
  matter says `instructor: …` (a note learners never see).

### The navigator: `nav:`

`nav:` in `_course.md` (or si.conf) takes a position and options, split by
`·`, commas or spaces:

| Value | What it does |
|---|---|
| `left` (default), `right` | the modules in a column beside the lesson |
| `top` | the modules as tabs, the open module's lessons in a row under them (`tabs`) |
| `bottom` | a bar docked at the foot of the window: where you are, Outline, Back, Next |
| `sticky` (default), `not-sticky` | stays while the page scrolls, or scrolls away |
| `accordion` (default) | one module open; opening another closes the rest; the current lesson's module stays open |
| `collapsed` | every module closed but the current one |
| `open` | every module open |
| `numbered` | lessons numbered within their module (`1.1`, `1.2`, `2.1`) in the menu and on the Previous / Next cards (modules are always numbered) |

Each module shows its number, a progress ring and its minutes; each lesson
its kind (Read, Slides, Video, Lab, Demo, Quiz), its minutes and a done tick.
Menu text is at full strength; a hovered lesson gets a tinted row and an
accent bar, the current one a solid accent bar. On a phone every layout folds
into a **Menu** button and a bottom bar. Everything is drawn with the theme's
`--si-*` tokens, so every theme, light and dark, works.

### Quizzes

A `type: quiz` page (or a ```` ```quiz ```` fence in any lesson, article or
guide page): `## ` questions, the answers as a task list (`- [x]` marks a
right one; more than one makes it multiple choice), an optional `>
Explanation:` shown after answering, and a pass mark.

````markdown
---
type: quiz
title: Check yourself
pass: 70%
---

# Check yourself

## Which tag makes the largest heading?
- [x] `<h1>`
- [ ] `<h6>`
- [ ] `<head>`
> Explanation: `h1` to `h6` are headings; `head` holds the page's settings.

## Which of these are block elements?
- [x] `<p>`
- [ ] `<em>`
- [x] `<ul>`
````

A fence takes the mark as an option: ```` ```quiz pass=80% ````. A question
is right when exactly its right answers are picked; the score is the share
of right questions. Without a pass mark, answering every question completes
the quiz. `si check` reports a question with fewer than two answers or none
marked right.

**A quiz is a self-check, not a secure exam.** A learner page keeps no right
answers in its source: each question carries only a SHA-256 hash of a
per-quiz salt and the ids of its right answers, and the page hashes the
learner's picks the same way to check them. Explanations are shown after
answering. A question has few answers, so anyone can try them all, or read
the explanations in the source; use an LMS's own exam for grades that matter.
The instructor build (`--instructor`, `si play`) keeps the plain answers.

### Lab solutions

In a lab, `<!-- solution -->` up to `<!-- end -->` is the answer: folded behind
**Show the solution** for learners, open in the instructor build.

````markdown
## Step 3: Write the content

- [ ] The heading, the paragraphs, the list and the link all show

<!-- solution -->

```html
<h1>Sourdough at home</h1>
```

<!-- end -->
````

A solution can hold anything a lab can, includes too. `si check` reports a
solution without its `<!-- end -->` at the `<!-- solution -->` line; an
include inside it that fails is reported on its own (the solution still
finds its end).

### Progress

Progress is kept in the learner's browser (localStorage, per course; nothing
is sent anywhere unless the course has a `webhook:`, below). A lesson is done when it is read to the end, its video
watched to 90 %, its lab's boxes all ticked, or its quiz passed. The menu
ticks done lessons and fills each module's ring; the top bar shows the
course's percentage; the home page's **Start** becomes **Continue: <the
next lesson>**. `complete.html` lists what is left and, when everything is
done, says so; with `certificate: on` the learner types a name and prints a
certificate.

### Results to your team: `webhook:`

A course with `webhook:` in `_course.md` sends small result events from the
learner's browser to an address you choose: a Slack, Teams or Discord
channel, a learning record store, or your own endpoint (a Zap, a sheet, a
small server). It works in the website bundle, from `file://` and inside a
SCORM package (as well as the LMS calls).

```markdown
---
title: A first web page
webhook: https://hooks.slack.com/services/T000/B000/XXXX
webhook-format: slack
webhook-events: quiz.submitted, course.completed
webhook-learner: ask
webhook-consent: required
webhook-notice: Your quiz scores go to your trainer, Ana.
webhook-base: https://learn.example.com/web101
---
```

| Key | What |
|---|---|
| `webhook:` | `https://…` (si check warns on anything else), or `relay` (below). si.conf `course.webhook:` is the default for every course |
| `webhook-format:` | `json` (default), `slack`, `teams`, `discord`, `xapi` |
| `webhook-events:` | which events (default all five) |
| `webhook-token:` | json: a `"token"` in every event, to tell classes apart. It is in the page: not a secret |
| `webhook-auth:` | xapi: the `Authorization` header (`Basic …`) of the learning record store. In the page too: use a write-only key, or the relay |
| `webhook-learner:` | `random` (default): a random id per browser; `ask`: the notice has a name field |
| `webhook-consent:` | `notice` (default): events are sent, the notice stays until OK; `required`: nothing is sent before OK (events wait), **No thanks** sends nothing, ever |
| `webhook-notice:` | the notice's text (default: "This course sends your progress (lessons done, quiz scores) to its organizers.") |
| `webhook-base:` | `https://…` where the course is published: `course.url` and `lesson.url` (the xAPI activity ids) are under it wherever the page is opened (`…/web101/` and `…/web101/02-html/quiz.html`). Without it: the page's own address when it is `http(s)`, else `urn:si:course:<id>` and `urn:si:course:<id>:<page>` (never a `file://` path). si.conf `course.base:` is the default for every course |

The events:

| Event | When | `data` |
|---|---|---|
| `lesson.completed` | a lesson is done (as for the ticks above) | `{}` |
| `quiz.submitted` | Check my answers | `score` (0-100), `pass`, `passMark`, `attempt`, `questions: [{"id": "q1", "correct": true}, …]` |
| `lab.step.checked` | a lab's **✓ Check** ran (below) | `step`, `title`, `pass`, `exit`, `ms` |
| `demo.watched` | a lesson's video ends, or the learner leaves after watching some | `percent`, `seconds` |
| `course.completed` | the last lesson that counts is done | `lessons` |

`json` events (`version: 1`; a field may be added, none changes without
version 2):

```json
{
  "version": 1, "event": "quiz.submitted", "id": "5f0c9a6b-2d41-4e8f-9a3c-0d1e2f3a4b5c",
  "time": "2026-11-14T10:02:11.418Z", "si": "3.18.0", "audience": "learner",
  "course": { "id": "web101", "title": "A first web page", "url": "https://learn.example.com/web101/index.html" },
  "lesson": { "id": "02-html/quiz.html", "title": "Check yourself", "kind": "quiz", "url": "https://learn.example.com/web101/02-html/quiz.html" },
  "learner": { "id": "l-3f9a0c71b2e4", "name": "Ana" },
  "data": { "score": 80, "pass": true, "passMark": 70, "attempt": 2,
            "questions": [ { "id": "q1", "correct": true }, { "id": "q2", "correct": false } ] },
  "token": "c42-7f3a"
}
```

`id` is a UUID per event (drop duplicates by it: an event queued offline may
arrive twice); `course.id` is the course folder's name, `lesson.id` the
page's path in the course (`course.completed` has no `lesson`);
`audience` is `instructor` in the instructor build, so you can ignore the
trainer's own clicks. `slack`, `teams` (an Adaptive Card) and `discord` get
one sentence: "Ana passed the quiz “Check yourself” in A first web page:
80 % (attempt 2)." `xapi` sends an xAPI 1.0.3 statement: the learner as an
account (`homePage` the course URL, `name` the learner id), ADL verbs
(`completed`, `passed` / `failed`, `answered` for a lab check,
`experienced` for a video), the lesson's URL as the activity, the course as
its parent, `result.score.scaled` and `success`; point `webhook:` at the
LRS's `…/statements` endpoint.

How it is sent: `json` and `slack` with `navigator.sendBeacon` (a
`text/plain` JSON body: no CORS preflight, and it leaves even as the page
closes); `teams`, `discord` and `xapi` with `fetch` (keepalive), which needs
the receiver to allow CORS. What cannot leave (offline, refused, a 5xx)
waits in localStorage (the newest 100) and the page tries again: after 2 s,
then 4 s, 8 s … up to a minute apart, and at once when the browser is back
online; what is still waiting when the learner leaves goes with the next page.

**Privacy.** Events never carry the answers a learner picked, a lab's output,
or a name the learner did not type; the learner id is random and kept in
that browser. A notice at the bottom of the course says results are sent
before anything else happens.

**The relay.** `webhook: relay` keeps the address (and `webhook-auth:`) off
the page: `si serve` (or `si play`) answers `POST /_si/relay`, forwards each
event to the course's address (`course.webhook:` in si.conf) and signs it:
`X-Si-Timestamp: <Unix seconds>` and `X-Si-Signature: sha256=<hex
HMAC-SHA256 of "<timestamp>.<body>">` with `webhook-secret:` from your own
si.conf or `SI_WEBHOOK_SECRET`. A receiver checks both: recompute the HMAC
over the timestamp header, a dot and the raw body, compare in constant time,
and refuse a timestamp more than **5 minutes** from its own clock, so a
captured post cannot be replayed later (drop a repeated event `id` inside the
window too). A page served by si sends through the relay whenever it is on:
always on a loopback address, and on any address with `--relay` (`si serve
course/ --addr 0.0.0.0:8080 --relay`). With `si play --remote` the network
listener relays too, behind the run's key like the other remote pages. The
relay only ever forwards to a course's own configured address. A static
build of a `webhook: relay` course sends nothing.

```text
X-Si-Timestamp: 1763114531
X-Si-Signature: sha256=hex(HMAC-SHA256(secret, "1763114531." + body))
```

### Labs that check themselves: `<!-- check: -->`

In a lab, `<!-- check: <command> -->` under a step says how to tell the step
works: `· expect "text"` (the output must contain it) and `· exit N` (the
exit code that passes, 0 by default).

```markdown
## Step 1: Create the file

Make a folder called `my-page` and, inside it, a file called `index.html`.

<!-- check: ls my-page/index.html · expect "index.html" -->
```

Every build shows it as **Run this to check:** with the command and what
passes. In `si play` on a loopback address (where the terminal drawer
works) it gets a **✓ Check** button: si runs the command in a process of its
own (`sh -c`, `cmd /C` on Windows; never the drawer's shell), with the
page's folder as the working folder and a 30 s limit, and shows Passed or Not
yet with the exit code and the output. si runs only commands written in a
`<!-- check: -->` of that page. In a course the result is a
`lab.step.checked` event (through the relay, signed). `si check` reports a
check with no command or an option it does not know.

### The live class view: `si serve --class`

For a class in the room (or on a call), serve the course with `--class` and
watch it from a laptop or a tablet:

```text
si serve web-course/ --class --addr 0.0.0.0:8000
  class view (this machine): http://127.0.0.1:8000/_si/class
  class view on another device (keyed link, keep it to yourself):
  http://192.168.1.20:8000/_si/class?k=…   (and its QR code)
```

The course needs no `webhook:` for it: a page served this way sends every
result event, as JSON, to the server itself (`POST /_si/class/event`), beside
whatever goes to the course's webhook (which keeps its own
`webhook-events:` and format). Learners are asked their name
(`webhook-learner: ask` is the default with `--class`; write
`webhook-learner: random` to keep them anonymous ids), and the notice and
`webhook-consent:` work as above: a learner who says No thanks never shows up.

```markdown
---
title: A first web page
webhook-learner: ask
webhook-notice: Your progress shows on the trainer's screen during the class.
---
```

The dashboard (`/_si/class`) opens on this machine's loopback, or anywhere
with the run's key (the printed link; a new key each run). It shows, live
(server-sent events, no reload):

- **Learners by lesson:** one row per learner (the typed name, else
  "Learner 3f9a"), one column per lesson under its module: ✓ done, the quiz
  score (green passed, red not yet; ×2 for attempts), a dot per lab check
  (red failing) with passed/total, ▶ the percent of a video watched.
- **Who's stuck:** a learner who failed the same check twice in a row, or
  made no progress (a lesson done, a quiz or check passed, a video watched)
  for N minutes on the lesson of their last event (N: 5, 10, 15 or 30, 10
  by default).
- **Lessons:** done, quiz average and passed, learners failing a check,
  average watched, per lesson. **Quiz questions:** for each quiz, how many
  got each question wrong (each learner's last attempt), each question
  named by its title from the quiz's source.
- **Module** filter, **Export CSV** (one row per learner and lesson), light
  and dark, the course's theme. More than one course in the folder: a Course
  picker.
- **Day:** today, live, or any earlier day with a saved
  `.si/class/<date>.jsonl`, read back as it stood at that day's last event
  (CSV too).

Every event is appended to `.si/class/<date>.jsonl` beside the served folder
as it arrives; a restarted server starts again from today's file. Kept: the
event, the lesson, the score, the step and the random learner id, and the
name the learner typed: no address, no browser, no answers, no lab output.
Delete the files when the class is over. `si play course/ --class` does the
same for the instructor's build (its events say `instructor`). The class
view is a browser page of `si serve` / `si play` only: the Studio (`si
--web`) and the terminal (`si --terminal`, `si play --terminal`) have none.

### Build, serve, check

- **`si build course/ -o html/`** writes the website bundle: `index.html` (the
  course home: level, length, modules, Start / Continue, a card per module),
  one page per lesson mirroring the folders, `complete.html`, and the videos
  copied beside their pages (never inlined). Shared scripts are written once
  (`--share-assets` is on for courses). `--zip` also writes `<course>.zip`
  beside the output; `--instructor` builds the instructor's edition;
  `--no-film` skips filming web demo lessons.
- **`si serve course/`** serves the learner's course with live reload;
  **`si play course/`** the instructor's.
- **`si check course/`** reports a missing page or video, a lesson outside a
  module, an unknown `kind=`, bad `minutes=`, an unfilmed web demo (a
  warning), quiz problems and broken links between lessons.
- **`si new web-course --type course`** writes a small course with every kind
  of lesson; `si --export example --type course` the same example. The
  Studio's **New…** and start gallery offer it as **Course**.

### SCORM packages: `--scorm`

`si build course/ -o html/ --scorm` also writes **`<course>-scorm12.zip`**
beside the output: the learner build plus an `imsmanifest.xml` (SCORM 1.2,
which SkillJar, Moodle, Canvas, Docebo and Cornerstone all import).

| Flag | SCOs | What the LMS shows |
|---|---|---|
| `--scorm` | one per lesson (instructor demos left out) | its own menu of modules and lessons; the course menu in the pages is hidden |
| `--scorm=single` | one for the whole course (`scorm.html`) | the course as on the web, menu and all |

The pages find the LMS's SCORM API in a parent or opener window and report
to it: `LMSInitialize`, `cmi.core.lesson_status` (`incomplete`, then
`completed`; a quiz `passed` or `failed`), `cmi.core.score.raw` from the
quizzes (0 to 100; one SCO for the course reports their mean),
`cmi.suspend_data` with the progress (the done lessons, quiz scores, the last
lesson; so a learner continues on another device; SCORM 1.2 keeps 4096
characters of it, so a course whose progress is longer sends it packed, its
lessons by their place in the menu with a hash of the menu: a republished
course with other lessons starts that copy afresh, and progress that does
not fit even packed stays in the browser), `LMSCommit` and
`LMSFinish`. A quiz lesson with `pass:` gets that mark as its
`adlcp:masteryscore`. The package has no network dependencies and still
works with no LMS (progress in the browser, as on the web). `--scorm` packs
the learner build; it refuses `--instructor`.

```text
si build web-course -o html --scorm          # html/ and web-course-scorm12.zip
si build web-course -o html --scorm=single   # one SCO: scorm.html
```

### Printed handouts: `--pdf`

`si build course/ -o html/ --pdf` (needs Chrome) also writes the **learner
workbook**, `html/<course>-workbook.pdf` (and `workbook.html`): a cover with
the module plan and timings, then every lesson in order: the readings, the
lab steps without their solutions, the quiz questions without answers or
explanations, a deck's slide titles, and a lined **Notes** box after each
lesson. With `--instructor` it writes the **instructor guide**,
`<course>-instructor-guide.pdf`: the same order with each lesson's minutes
and instructor note, every demo's runbook (commands and expected output),
the decks' spoken script, the quiz answers ticked with their explanations,
and the lab solutions.

### Publish to SkillJar: `si publish --to skilljar`

```text
si publish login skilljar --token <API key>
si publish web-course --to skilljar --base-url https://files.example.com/web-course/
si publish web-course --to skilljar --base-url https://files.example.com/web-course/ --yes
```

The learner build becomes a Skilljar course: each module a section, each
reading an HTML lesson, each video an asset lesson, and each slides, lab or
quiz lesson (and a reading with a quiz in it) a SCORM 1.2 web-package lesson
of its own. Instructor demos are never published. Skilljar's API takes no
uploads: it fetches videos and packages from a URL, so si writes them into
`web-course/.si/publish/skilljar/` and you put that folder online at
`--base-url` before `--yes`. Without `--yes` it is a dry run that prints
every request (the key masked). The ids land in
`web-course/.si/publish/skilljar.json`, so the next publish updates the same
course in place: lessons patched and reordered, only changed files fetched
again, removed lessons deleted. Publish the course to your domain in the
Skilljar dashboard. See `docs/publish.md`.
