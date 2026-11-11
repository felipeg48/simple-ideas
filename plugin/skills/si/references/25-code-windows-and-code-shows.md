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
| `numbers` | (a flag; `linenos` works too) | line numbers in a gutter |
| `hl=` | `3`, `3,5-7` | highlighted lines of this step |
| `id=` | a name | blocks with the same id in a row are one show; on slides, the same id on consecutive slides morphs |
| `step=` | text | the step's name, shown under the window (`2 / 4 · Back off`) and in the Studio |
| `transition=` | `morph` (default), `fade`, `slide` | how a show goes from step to step |

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

An unknown value of `window=`, `theme=`, `backdrop=` or `transition=` is a
`si check` problem.

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
window with C# and a `.csproj`, a terminal), the showcase slide "The retry
rule" (two steps, one click), and a terminal window in
`examples/article/retry.md`.
