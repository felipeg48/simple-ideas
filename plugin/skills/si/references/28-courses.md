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
| `type: lab` | Lab | the lab; its task-list boxes can be ticked |
| `type: demo` without `app:` | Demo | the runbook (instructor build only) |
| `type: quiz` | Quiz | the questions, scored |
| `type: deck` with `· kind=video` | Video | the deck's narrated `<deck>.mp4`, filmed by `si build` when missing or older than the deck |

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
  videos, labs, quizzes. Instructor demos are left out of the menu, or with
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
| `numbered` | numbers on guide sections (modules are always numbered) |

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
is sent anywhere). A lesson is done when it is read to the end, its video
watched to 90 %, its lab's boxes all ticked, or its quiz passed. The menu
ticks done lessons and fills each module's ring; the top bar shows the
course's percentage; the home page's **Start** becomes **Continue: <the
next lesson>**. `complete.html` lists what is left and, when everything is
done, says so; with `certificate: on` the learner types a name and prints a
certificate.

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
lesson; so a learner continues on another device), `LMSCommit` and
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
