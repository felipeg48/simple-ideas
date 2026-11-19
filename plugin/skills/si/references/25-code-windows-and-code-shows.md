# 25. Code windows and code shows

A fenced code block with `window=` is drawn as a window: macOS (traffic
lights and a centred title), Windows (title on the left, minimise, maximise,
close), a terminal (a tab; a file gets a `$ cat <title>` line above it, while a
shell block, `sh` `bash` `zsh` `console` `powershell`, shows only its own
lines, so write the prompt yourself: `$ go test ./...`) or `none` (a plain rounded
card), optionally on a backdrop, with numbered callouts you place by hand. A
run of code blocks with the same `id=` is a **show**: one window whose steps
morph into each other (the lines that stay glide to their new place, new lines
fade in, removed ones fade out). It works in every document type, on slides and
in mind maps.

````markdown
```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers step="One attempt"
func send(order Order) error {
	resp, err := client.Post(url, order)
	if err != nil {
		return err
	}
	return check(resp)
}
```
<!-- callout: 2 One attempt: a dropped connection loses the order. at=60,-54 -->

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers step="Retry five times" hl=3
func send(order Order) error {
	var last error
	for try := 1; try <= 5; try++ {
		resp, err := client.Post(url, order)
		if err == nil {
			return check(resp)
		}
		last = err
	}
	return last
}
```
<!-- callout: 3 Five tries, back to back. -->
````

### Options (on the fence line)

| Option | Values | What it does |
|---|---|---|
| `window=` | `mac`, `windows`, `terminal`, `none` | turns the block into a window; without `window=` or `id=` nothing changes |
| `title=` | text (quote it when it has spaces) | the title bar: `title="PaymentClient.cs - Gateway"` |
| `theme=` | `auto` (default), `night`, `paper`, `dusk` | `auto` uses the deck or page colours (light and dark); the others are code-window looks |
| `backdrop=` | `none` (default), `dusk`, `ocean`, `mint`, `graphite` | a gradient behind the window, with room around it |
| `shadow=` | `lifted` (default), `soft`, `deep`, `none` | how far the window floats off the page: a drop shadow and a faint edge that show in light and dark (a backdrop gives a deeper one unless `shadow=` is set) |
| `numbers` | (a flag; `linenos` works too) | line numbers in a gutter |
| `start=` | a number (default 1) | the gutter's first number: `numbers start=120` for a block cut from line 120 of a file |
| `hl=` | `3`, `3,5-7` | highlighted lines of this step |
| `id=` | a name | blocks with the same id in a row are one show; on slides, the same id on consecutive slides morphs |
| `step=` | text | the step's name, shown under the window (`2 / 4 · Back off`) and in the Studio |
| `transition=` | `morph` (default), `fade`, `slide` | how a show goes from step to step; on a slide the window keeps the tallest step's height (and room for the longest list of notes on a phone), so a step moves only the code, never the title or text around it |
| `as=` | `cat` | a shell script in a `terminal` window shows as a file, under `$ cat <title>` |

```xml window=windows title="Gateway.csproj" theme=paper
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>
```

```sh window=terminal title="checkout" backdrop=graphite
$ go test ./payments -count=100
ok  	payments	2.41s
```

An unknown value of `window=`, `theme=`, `backdrop=`, `transition=` or `as=`
is a `si check` problem.

### Code from a file

A window can show a real file: give the include (or a `<!-- code: -->`
permalink, §13) the same options as the fence. It draws exactly the window
the fence would, with the file's text, highlighted by its name (`.yaml`
`.yml`, `.sh` `.bash` `.zsh`, `.json`, `.toml`, `.ini` `.conf` `.properties`
`.env` (also `rabbitmq.conf`), `Dockerfile` `Containerfile`, `Makefile`,
`.sql`, `.xml`, `.md`, `nginx.conf`, `.tf`…; `lang=` overrides). The title is
the file name unless `title=` says otherwise, and `lines=` / `tag=` keep their
meaning: with `numbers`, the gutter counts from the first line shown
(`lines=40-52` numbers 40 to 52), as the file does.

```markdown
<!-- include: compose.yaml window=mac theme=night numbers hl=6 -->
<!-- callout: 6 The same five tries as production. -->

<!-- include: start.sh window=terminal as=cat numbers hl=4-5 -->
```

Includes with the same `id=` are a show like fences: two versions of a
file (`start-v1.sh`, `start-v2.sh`, both `title=start.sh`), or one file's
`tag=` regions, morph step to step, on slides (one click each) and in
documents (← → Play). Callouts go after the include line. In `si play` an
edit to the included file reloads the page, also for a file outside the
served folder (`../code/compose.yaml`). The Studio's Window view edits the
include's options and callouts (its code is the file: edit that), and
`si code` exports it.

A shell script in a `terminal` window shows only its lines, like a command
typed at the prompt; `as=cat` shows it as a file instead, under
`$ cat start.sh`. It works on a fence too.

### Callouts

`<!-- callout: LINE text -->` right after the block (blank lines between are
fine, one per line) puts a numbered dot at the end of that code line and a
note beside the window, joined by a dashed wire. The text takes inline
Markdown. `at=dx,dy` is where the note sits, in pixels from the end of the
line; without it the note goes to the right of the window. Drag a note to move
it: in `si play` and in the Studio's Window view the new place is written back
into the Markdown as `at=` (si play finds the callout by its line and text, in
the page's file or a file it includes); on a built page the move is local.

```markdown
<!-- callout: 8 Declines are final: never retry a refused card. at=70,-64 -->
<!-- callout: 12 Jitter spreads the retries of a thousand orders. -->
```

On a narrow screen (a phone) the notes move under the window as a numbered
list; the dots stay on the lines. `si check` reports a callout on a line the
block does not have and a callout with no window above it.

### Shows in documents, on slides, in mind maps

- **Documents** (articles, guides, labs, sites): a show is one window with
  `← →` and Play under it (Play steps every 2.6 s).
- **Slides:** several same-id blocks on one slide are build steps, one click
  each (write one `[CLICK]` per extra step in `Notes:`); the same id on
  consecutive slides morphs as the slide changes, without the slide's fade, like
  an animated canvas.
- **Mind maps:** a code topic takes `window=` and `title=` and gets a title bar
  (macOS lights or the title).
- Without JavaScript every step is a plain code block with its callouts as a
  list.

### Exports

```bash
si code retry.md --png --scale 2   # retry-retry-1.png … one per step, backdrop and callouts included
si code retry.md --svg             # retry-retry-1.svg … (fonts embedded)
si code retry.md --gif             # retry-retry.gif, the whole show: 2.2 s per step, a short morph between
si code retry.md --mp4             # retry-retry.mp4, the same show as H.264 video (needs ffmpeg)
si code talk.md --id charge --gif -o out/   # one show of a deck (its id), into out/
```

Files are named `<file>-<id>-<step>`; a window without an id is
`window-<n>`, counting the file's windows from 1. A show split across
slides exports as one show. Exports need Chrome or Chromium (headless). The
GIF uses one shared 256-colour palette, and the morph is drawn as five frames
between steps. The MP4 (`--mp4`, at `--scale`, 30 fps, H.264, yuv420p,
faststart: it plays in browsers and posts to social sites) needs ffmpeg on the
PATH (`brew install ffmpeg`, `apt install ffmpeg`); si says so when it is
missing.

### In the Studio and the terminal

Put the cursor in a code block and choose **Window** in the bar above it: a
preview of the window and panels for its look (window, theme, backdrop,
transition, title, line numbers), its steps (name, order, add a step as a copy,
delete, and each step's code in a small editor) and its callouts (add one on a
line, edit, delete, drag in the preview to place it), plus PNG, SVG and GIF
exports next to the file. The editor completes code block options after
```` ```go ```` (`window=`, `title=`, `theme=`, `backdrop=`, `numbers`, `hl=`,
`id=`, `step=`, `transition=` and their values), mind map options after
```` ```mindmap ````, `title=` and `lang=` after an include's path, and the
block's line numbers after `<!-- callout: `. Every change is
written into the Markdown as one undoable edit. `si play --terminal` shows a
window's title (`● ● ●  retry.go` for `mac`) in its box and the callouts as
numbered notes; a show advances one step per build (→), with the step in the
box title (`retry.go · 2/4 Back off`) and that step's callouts, and the
`--notes` teleprompter counts them like any build. In the terminal Studio
(`si --terminal`), `:` offers every code block's window style, line numbers
and title, written into the fence line the same way.

Examples: `examples/code/retry.md` (a four-step show on a backdrop, a Windows
window with C# and a `.csproj`, `compose.yaml` and `start.sh` shown from
their files, a terminal), the showcase slide "The retry
rule" (two steps, one click), and a terminal window in
`examples/article/retry.md`.

### Video player

A fenced code block whose language is `video` is a video player, authored
like a code window: the options go on the fence line, any body is the
caption. The player has its own controls (the HUD), themed with the page's
`--si-*` tokens: play and pause, a scrub bar with the buffered part, the time
under the pointer and chapter marks, the time, volume and mute, speed (1×,
1.25×, 1.5×, 2×), captions, a chapter list, picture in picture where the
browser has it, and fullscreen. Before the first play it shows the poster
with a big play button; while it plays the HUD hides itself until the
pointer moves. `si build` copies the video beside the page (as it does for
`![](clip.mp4)`, §8) and inlines the player's CSS and JS; nothing external
loads.

````markdown
```video src="assets/demo.mp4" poster="assets/demo.jpg" mode=modal title="The walkthrough"
Two minutes, end to end.
```
````

- **`mode=inline`** (default) — the player in place.
- **`mode=modal`** — the poster is a button; it opens the player large in a
  dialog over the page, which is dimmed and blurred behind it. Esc, the ✕
  button or a click on the dimmed page closes it (and pauses); focus stays in
  the dialog while it is open and goes back to the poster after. Every modal
  player on the page is one gallery: the dialog shows "2 / 9", the title and
  caption under the video, and ‹ › to step through them. `group=` makes
  separate galleries.
- **`mode=facade`** — like inline, but nothing loads until the first play;
  good for a page with many clips.

A block with only a `poster=` and no `src=` degrades: the poster shows with
the play control disabled, rather than erroring.

| Option | Values | What it does |
|---|---|---|
| `src=` | a video file or URL (mp4, webm, mov, m4v, ogv) | the video; omit it (with a `poster=`) for a disabled placeholder |
| `poster=` | an image file or URL | the still before play and the modal thumbnail; default `<video>.poster.jpg`, made from the first frame when ffmpeg is there |
| `title=` | text (quote it when it has spaces) | the accessible label, the poster's alt text, the title under the video in the dialog |
| `mode=` | `inline` (default), `modal`, `facade` | how it plays (above) |
| `captions=` | a `.vtt` or `.srt` file | the captions; default: `<video>.vtt` (or `.srt`) beside the video |
| `chapters=` | a `.txt` (`0:00 Title` lines) or `.vtt` file | marks on the scrub bar, the chapter list, the chapter's name in the HUD; default: `<video>.chapters.txt` (or `.chapters.vtt`) beside the video, or a `chapters.txt` beside a video alone in its folder (as `si demo run --with chapters` writes it) |
| `group=` | a name | `modal` only: players with the same group share the dialog's ‹ › (default: every modal player on the page) |
| `start=` | a time (`12`, `1:05`, `1:02:03`) | starts there |
| `hud=` | `on` (default), `native`, `off` | si's controls, the browser's own, or none (a click plays and pauses) |
| `autoplay` | a flag | `inline` only, always muted (browsers block sound) |
| `muted`, `loop` | flags | starts muted; loops |
| `lang=` | a language code | the captions' language (default: the page's `lang:`) |
| `width=` | a length (`40rem`, `70%`) | caps the player width |
| `aspect=` | a ratio (`16/9` default, `4/3`, `1/1`) | the player's aspect ratio (`ratio=` works too) |

Captions and chapters are read at build time and carried in the page (the
captions as a `data:` track), so a built page shows them from `file://` too.
`![caption](clip.mp4)` in an article, guide, book or lab (§8) and a course's
video lessons (§28) play in the same player and find the same files beside
the video, so the videos si films (`si demo run`, `si build --mp4 --voice`)
get their captions and chapters with nothing to write. On slides,
`![](clip.mp4)` keeps the browser's own player; use the fence for the HUD.

**Keys** while a player has focus (click it or Tab to it):

| Key | What |
|---|---|
| Space, K | play / pause |
| ← → | back / forward 5 s |
| J L | back / forward 10 s |
| Home End | start / end |
| M | mute |
| F | fullscreen |
| C | captions on / off |
| Esc | close the chapter list, leave fullscreen, close the dialog; on a slide, give the arrow keys back to the deck |
| ← → in the dialog | the previous / next video while paused; Shift+← → while playing |

On a touch screen a tap shows or hides the HUD and a double tap on the left
or right third seeks 10 s. Buttons carry labels for screen readers, focus is
always visible, and `prefers-reduced-motion` turns the HUD's animations off.

An unknown option, `mode=` or `hud=`, a `start=` that is not a time and a
missing captions or chapters file are `si check` problems. The player's look
is in `--si-vp-*` tokens (`si --export css-rules` lists them).

**On pages si did not build.** `si --export player [dir]` writes
`si-player.css` and `si-player.js` (the same two files si inlines), a README
and `demo.html`. Load the two files on any page and every element with
`data-video` becomes a player: a card with an `<img>` uses it as the poster,
its first heading as the title and its first paragraph as the caption, and a
click (or Enter) opens the dialog, with ‹ › through every card on the page.

```html
<link rel="stylesheet" href="si-player.css">
<script src="si-player.js" defer></script>

<figure class="video-card" data-video="clips/01-intro.mp4" data-captions="clips/01-intro.vtt"
        data-chapters="clips/01-intro.chapters.txt">
  <img src="shots/01-intro.png" alt="The complete film">
  <figcaption><h3>The complete film</h3><p>The whole story, end to end.</p></figcaption>
</figure>
```

| Attribute | What |
|---|---|
| `data-video` | the video file |
| `data-poster` | the still (default: the `<img>` inside) |
| `data-title`, `data-caption` | the title and text under the video (default: the heading and paragraph inside) |
| `data-captions` | a WebVTT file |
| `data-chapters` | `0:00 Title` lines or a WebVTT chapters file |
| `data-mode` | `modal` (default) or `inline` (plays in place of the picture) |
| `data-group` | separate galleries |
| `data-start` | a start time |

The drop-in fetches captions and chapters, so serve the page over http. Its
colours follow `--si-accent` when the page sets it; `window.siPlayer.init(el)`
wires players added after the page loaded.
