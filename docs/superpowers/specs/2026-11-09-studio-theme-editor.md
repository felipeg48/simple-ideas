# The Theme editor in the Studio (web and desktop)

Status: spec, not started. Order: 6 (after or with "theme from a logo",
which it uses as one way to start). Requested by the founder, 2026-11-09:
"create a theme, and the best way is the Studio".

## Why

A theme in si is a CSS file of `--si-*` tokens (`theme: brand.css`). It works,
but making one means knowing ~100 token names, which look light and which
dark, and checking contrast by eye. The Studio already edits mind maps (Map
view) and code windows (Window view) with live previews; a Theme view does
the same for the look of everything si renders.

## What the author does

1. **Start**: Studio › New… › Theme (or ⌘K "New theme"), then pick a start:
   - a built-in theme (studio, paper, mono, crdb, acemq, acemq-pro) to copy,
   - **From a logo…** (the algorithm of `2026-11-09-theme-from-logo.md`),
   - or blank (studio's values underneath, nothing changed yet).
   It creates `themes/<name>.css` in the workspace and opens it in the
   Theme view.
2. **Edit** in three panes:
   - **Tokens** (left), grouped: Colours, Type, Headings, Space, Shape,
     Motion, Code, Mind maps, Code windows, QR. Each control fits its token:
     a colour pair (light | dark) with a picker and hex field; a font field
     with system stacks and the `fonts:` providers (Google, Bunny, Fontshare,
     Fontsource, a local file); a slider for sizes and spaces (in `em`,
     `cqi`, `cqb`, never `px`); a select for enumerations (heading case).
     Changed tokens show a dot; "Reset" goes back to the base theme's value.
   - **Preview** (centre): a specimen deck rendered with the theme as it is
     being edited: title, two-col, a quote, a big number, a table, code with
     callouts, a mind map, a code window, a stage diagram, a QR code; a grid
     of slides or one large, light / dark / both side by side, four screen
     sizes (desktop, phone, portrait tablet, 4K). A switch shows the user's
     own open deck instead of the specimen.
   - **Checks** (right, collapsible): every text/background pair with its
     WCAG ratio (body 4.5:1, large and UI 3:1), light and dark; a failing pair
     has **Fix** (moves the text colour in OKLab until it passes, keeping its
     hue). The same checks `si check` runs on the theme file.
3. **Use it**: "Use in this deck" writes `theme: themes/<name>.css` into the
   open deck's front matter (the inspector's writer, one undoable edit) or
   "Use in this folder" writes it into `si.conf`. The file is plain CSS,
   readable and editable as text in the normal editor too; the Theme view
   and the text editor stay in step (like Map view and its Markdown).

## What a saved theme looks like

```css
/* si theme: Acme · extends studio
   made in si Studio; edit here or in the Theme view */
@layer si.theme {
  :root {
    --si-accent: #e4572e;
    --si-font-heading: "Fraunces", Georgia, serif;
    --si-h1-size: 3.4em;
  }
  :root[data-si-mode="light"], :root:not([data-si-mode]) { --si-bg: #fbf8f4; }
  :root[data-si-mode="dark"] { --si-bg: #17130f; --si-accent: #ff7a54; }
  @media (prefers-color-scheme: dark) { :root:not([data-si-mode="light"]) { --si-bg: #17130f; --si-accent: #ff7a54; } }
}
```

Only the changed tokens are written; `extends studio` names the base.
(The exact light/dark selectors follow what `assets.ThemeCSS` emits today;
phase 1 settles them in one helper used by both.)

## Engine changes first (phase 1, in si itself)

These fix gaps found earlier with CSS-file themes and make the editor's
small files work:

1. **`extends`**: a theme file's header comment `si theme: … · extends
   <built-in>` makes the renderer emit that built-in theme's tokens first and
   the file after (both in `@layer si.theme`, the file wins). Without the
   header, a `.css` theme keeps today's behaviour (nothing underneath), so no
   existing theme changes. Today a partial theme loses `--si-h1-size`, pad,
   gap and shadow, and headings fall back to body size.
2. **The index page** of `si build` follows a `.css` theme (today it falls
   back to studio unless `--css` is passed).
3. **Token metadata** in `internal/assets/contract.go`: each `Token` gains a
   group, a kind (`color`, `length`, `font`, `number`, `enum`, `duration`,
   `shadow`), its allowed units and enum values. The editor's controls and
   the validation both come from it; `TestContractCoversBuiltinCSS` keeps it
   complete.
4. **A theme reader/writer** (`internal/themefile`, stdlib): parse a theme
   file (header, base, light/dark/base values per token), write it back with
   only changed tokens; unknown CSS in the file (a user's extra rules) is
   kept verbatim, like the mind map model keeps unknown text.
5. **`si check` on a theme**: unknown tokens, values of the wrong kind, a
   failing contrast pair: warnings with the line.

## Studio architecture

- **API** (`internal/studio/theme.go`, behind the Studio gateway: token,
  Host, Origin; paths confined to the workspace):
  - `GET theme/spec`: groups, tokens with kind and units, every built-in
    theme's values (for Start and Reset).
  - `POST theme/preview {css, deck?}`: the specimen deck (or the open deck)
    rendered with that CSS; returns the page for the preview frame. Debounced
    like the Map view's redraw.
  - `POST theme/check {css}`: the contrast and validity report.
  - `POST theme/new {name, from: builtin|logo|blank, logo?}` and saving
    through the normal file API, so History, undo and autosave just work.
- **Front-end** (`web/studio/src/theme-editor.tsx`, Preact + the existing
  CodeMirror text editor for the file): controls are plain inputs
  (`<input type="color">`, range, select), which also work in the desktop
  app's WKWebView. Edits go through `EditorCtl.edit` as text changes, one
  undoable step per gesture.
- **Specimen deck**: `internal/assets/web/themes/specimen.md`, embedded,
  every feature a theme touches; also used by `si --export themes`.
- **Desktop**: nothing extra; File › New Theme… in the menu calls the
  Studio command (`window.siStudio`), like Help does.
- **Terminal Studio**: out of scope (it can show the specimen preview later).

## Import and export (founder, 2026-11-09: "easy to share")

A theme should travel as one file and come back the same.

- **Export** (`si theme export themes/acme.css -o acme.si-theme.css`, and
  Export… in the Theme view): one self-contained CSS file: the header
  (`si theme: Acme · extends studio`, `fonts:` line), every token it changes,
  its logo and local font files inlined as data URIs. A built-in theme exports
  too (`si theme export paper`), as a full starting point.
- **Import** (`si theme import acme.si-theme.css|https://…|folder/` and
  Import… / drop a file on the Theme view): validated with the theme checker
  (known tokens, values of their kind, the Safety rules below), copied into
  `themes/`, data URIs kept or written out as files (`--unpack`), then opened
  in the Theme view; `--use` also writes `theme:` into si.conf.
- **Design tokens** (`--format dtcg`): export to and import from the W3C
  Design Tokens format (`.tokens.json`: colours, dimensions, font families,
  durations), so a theme moves to and from Figma (Tokens Studio), Style
  Dictionary and other tools. Unmapped tokens are reported, never guessed.
- **Gallery**: `si --export themes` includes the workspace's themes beside
  the built-ins; a theme can be shared as its gallery page plus the file.

## With AI agents (the MCP server and the skill)

An agent can design a theme the way it writes a deck: propose, check, look,
refine. The skill teaches the method; the MCP tools do the work and return
data the agent can act on.

- **MCP tools** (`si mcp`):
  - `si_theme_spec`: groups, tokens, kinds, units, built-in values.
  - `si_theme_write {name, extends, tokens{light,dark,base}, fonts}`: writes
    the theme through `themefile` (only known tokens, values of their kind);
    returns the file and its check report.
  - `si_theme_check {path}`: contrast pairs (light and dark) and validity,
    with the ratio of each failing pair and the nearest passing value.
  - `si_theme_preview {path, deck?, dark?}`: the specimen (or a deck) as a
    picture, like `si_screenshot`.
  - `si_theme_from_logo {image}`: the palette of the logo spec, as data.
  - `si_theme_import` / `si_theme_export`.
- **Skill reference** `references/themes.md` (generated with the others):
  the token groups and what each controls, colour roles (accent for emphasis
  and links, accent-2 sparingly, tinted neutrals for surfaces), contrast
  minimums, font pairing (one display face for headings, a readable body
  face, a mono that matches), light and dark designed together, and the loop:
  write → `si_theme_check` → fix → `si_theme_preview` light and dark → done.
- **Prompts that should work**: "make a theme from our logo, warm and calm,
  serif headings"; "our brand colours are #0B3D91 and #FC3D21: a theme that
  passes contrast in dark mode"; "import brand.tokens.json and use it for
  this folder".

## Safety

Theme CSS is author code that ends up in every built deck. The writer only
writes known tokens with values of their kind; `url()` only to files inside
the workspace (inlined at build, as today); no `}` / `;` / `@import` /
`expression` inside a value. Hand-written extra rules are kept but flagged in
the Checks pane ("not edited here").

## Rules check

1. Fluid: lengths only in `em`, `rem`, `cqi`, `cqb`, `%` (the slider offers
   no `px`); the preview shows four sizes.
2. Self-contained: fonts are inlined at build (`fonts:` providers, local
   files); nothing fetched at view time.
3. RULES.md §15 gains "Make a theme in the Studio" and the `extends` header,
   with an example; `examples/css/harbor.css` gains the header.
6. Themes change tokens and looks only: the editor cannot write layout
   mechanics (layout.css stays unlayered and untouched).
8. Stdlib and goldmark; the Studio's existing Preact/CodeMirror bundle.

## Tests

- Go: `themefile` round trip (a file with comments and extra rules survives
  an edit byte for byte outside the changed lines), `extends` rendering, the
  index page theme, unsafe values refused, `si check` on themes.
- vitest: controls ↔ text in step, reset, light/dark pairs.
- e2e (Playwright): New theme from studio → change the accent and the
  heading font → the preview updates → a failing contrast pair → Fix →
  save → "Use in this deck" → the deck renders with it; light and dark; the
  a11y suite audits the Theme view.
- The founder: make a real brand theme in the desktop app.

## Phases

1. Engine: `extends`, index theme, token metadata, `themefile`, `si check`,
   `si theme export` / `import` (CSS), the MCP theme tools.
2. Theme view: Colours + Type + Headings groups, preview, save, use.
3. All groups, the Checks pane with Fix, Start from a logo (shares
   `internal/themegen` with the logo spec).
4. Polish: Design Tokens (`--format dtcg`) import/export, compare two themes
   side by side, duplicate a theme, custom themes in `si --export themes`.

## Open questions

- Should a theme also carry `fonts:` (today a front matter / si.conf key)?
  Proposed: yes, as a header line (`fonts: heading=Fraunces:600`), so a
  theme file is the whole look.
- Share a theme between projects (a theme folder in the config dir, offered
  in Start)? Proposed: phase 4.
