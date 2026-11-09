# 16. Configuring keys and features (si.conf)

An optional `si.conf` next to the deck (or `--conf <path>`) sets project-wide
defaults without editing each deck. It uses the same `key: value` style as
front matter — no new syntax. Blank lines and `#` comments are ignored.

| Key | Effect |
|---|---|
| `theme: <name>` | default theme when a deck sets no `theme:` and there is no `--theme` |
| `keys.<action>: <key>` | remap a command key |
| `feature.<name>: off` | turn a feature off (its key stops working and its toolbar button is hidden) |
| `links.<label>: <url>` | a page outside the build that find (`/`) offers, e.g. `links.Team wiki: https://wiki.example.com`; relative paths work too |
| `timer.time: 10,15,30` | break-timer presets (default `5,10,15,20,30,45,60`) |
| `timer.unit: seconds` | preset unit: `minutes` (default) or `seconds` |
| `timer.face: analog` | countdown face: `digital` (default) or `analog` (sport-clock dial) |
| `timer.animation: waves,orbit` | animation(s) behind the timer HUD: `waves`, `orbit`, `bokeh`, or custom names; a comma list gives a toggle |
| `timer.animation.path: ./anim` | directory of custom `<name>.js` / `<name>.svg` animations |

Nesting is written as dotted keys (`timer.animation.path`), not YAML.

Remappable **actions**: `overview`, `presenter`, `draw`, `timer`, `find`,
`mode`, `fullscreen`, `help`. Toggleable **features**: `draw`, `timer`,
`presenter`, `overview`, `find`.

```
# si.conf — flat key: value, same style as front matter (not YAML)
theme: crdb
keys.draw: w          # draw with w instead of a
feature.timer: off    # no break-timer HUD for these decks
```

Run `si --export conf` to print a starter `si.conf` with every key and feature
at its default. Theme precedence: `--theme` flag > a deck's `theme:` front
matter > `si.conf` > the built-in default (`studio`). `si play` re-reads
`si.conf` on every request, so edits show up on reload. An unknown key or a bad
value is a warning printed to the terminal, never a failed build.
