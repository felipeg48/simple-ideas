# si in the terminal

Everything si does in a browser, in a terminal: over SSH, in tmux, on a machine
with no desktop. It is the `internal/tui` package (Bubble Tea and Lip Gloss, a
documented exception to the stdlib-only rule; a built deck never uses it).

## Present: `si play talk.md --terminal`

The deck drawn with text and colour: the theme's palette, block-letter titles,
builds, every layout, code, tables, cards, badges, logos, images as half-block
pixels and stage diagrams as boxes. RULES.md §14 "In the terminal" lists what
draws and what shows a placeholder.

| Key | |
|---|---|
| → Space `n` `l` | next build or slide |
| ← `p` `h` | back (a slide shows whole when you come back to it) |
| `g` / `G` | first / last |
| `o` | overview |
| `t` | restart the timer |
| `?` | keys |
| `q` | quit |

The deck reloads when a file changes; resize the terminal and the slide reflows.

`--dump N --size WxH` prints one slide and exits (ANSI colours).

## Teleprompter: `si play talk.md --terminal --notes`

In a second terminal: the `Notes:` script, `[CLICK]` cues as badges, timers
with pace, NOW / NEXT, auto-scroll (`a`, `+` `-`), `Tab` for the `Points:`.
It follows the presenter over a Unix socket (`$TMPDIR/si-<hash>.sock`, one per
deck) and Space / ← drive it back.

## Studio: `si --terminal [dir]`

The Studio as a keyboard app, the twin of `si --web [dir]`.

| Key | |
|---|---|
| ↑ ↓ / j k | move in the focused pane |
| Tab | focus: files, preview, problems |
| Enter | open a folder, edit a file, go to a problem's line |
| ← → | slides, when the preview has focus |
| `e` | edit in `$VISUAL` / `$EDITOR` (vi when neither is set); inside tmux it opens in a pane beside, so the preview follows every save |
| `n` | new slides, article, guide, book, site, page, lab or demo (the same starters as the web Studio; it joins the guide or site it lands in) |
| `b` | build the folder to `html/` |
| `s` | storyboard: every slide with its minutes, layout, builds, and the ones without a script |
| `i` | inspector: layout, transition, background, classes, eyebrow, written into the slide as directives (empty removes) |
| `:` or Ctrl+K | files and commands, fuzzy |
| `1` Write, `2` Rehearse, `3` Present, `4` Lab | modes |
| `r` | check again |
| `?` / `q` | keys / quit |

- **Write:** files (kinds and titles, guide / book / site folders marked), the
  live preview (slides, or a document as one scrolling page, images as pixels),
  and the problems of the same whole-workspace check as `si check`.
- **Rehearse:** the slide and its script side by side; → steps through builds,
  the next `[CLICK]` highlighted.
- **Present:** hands the terminal to `si play --terminal` until you quit it;
  inside tmux the teleprompter opens in a pane beside.
- **Lab:** see below.

## Lab (`4`)

The slide (← → step through builds), the commands of the deck, and a shell:

- **Commands** are the deck's runbook entries (`runbook:` JSON) and the shell
  blocks (`bash`, `sh`, `zsh`, `console`, `sql`, …) of its demo scripts
  (`demos:`), grouped by category and by demo, each with its heading.
- **Inside tmux** Lab opens a shell pane below (Ctrl+T opens another beside):
  Enter types the selected command into it, `x` types and runs it. The pane is
  your normal shell, so interactive tools (`cockroach sql`, `rabbitmqctl`) work
  there.
- **Outside tmux** Enter runs the command and streams its output into the
  Output pane (`c` clears it); `!` opens your shell full screen until you exit.
- **Placeholders** (`{{name}}`, `{{name:default}}`) ask for their values first,
  prefilled with the last value used this session, or the default.
- Tab switches between the commands and the slide; 1 2 3 leave Lab (and close
  its pane).

The shell pane runs your login shell: if your shell takes long to start (a heavy
`.zshrc`), wait for its prompt before running the first command.

It reuses the web Studio's backend (`studio.Workspace`: tree, kinds, titles,
check, starters) and the CLI's own entry point for Build.

## Testing it without a window

`go test ./internal/tui` renders slides, the teleprompter and the Studio at
fixed sizes. For a look, `si play x.md --terminal --dump N --size WxH` prints
ANSI that can be turned into HTML and screenshotted headless; a pty driver
(Python `pty`) can press keys in the real programs.
