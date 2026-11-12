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
marked right. The answers are in the page, so a quiz checks understanding;
it is not an exam.

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
  of lesson; `si --export example --type course` the same example.
