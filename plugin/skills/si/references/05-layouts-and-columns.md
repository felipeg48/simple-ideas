# 5. Layouts and columns

`<!-- layout: NAME -->` picks a slide layout. Every layout is fluid.

| Layout | What it does |
|---|---|
| `default` | heading at the top, content below, vertically centred |
| `center` | everything centred; lists stay left-aligned inside a centred box |
| `two-col` | equal columns; stack on narrow or portrait screens |
| `split` | 2 : 3 columns, the second on a panel (text + code, text + picture) |
| `quote` | the blockquote large, a paragraph after it as the attribution |
| `big-number` | the first paragraph huge in the accent colour, the rest as caption |
| `image` | the first image fills the slide; heading and text sit on it |

Columns: `<!-- column -->` starts the next column (any number). Using it
without a layout implies `two-col`.

```markdown
<!-- layout: two-col -->
## What changed

### Before
- Each team watched its own queue

<!-- column -->

### After
- One tracking screen
```

Without `<!-- column -->`, `two-col` and `split` put media blocks (an SVG
canvas, an image on its own line, a code block, a Mermaid diagram, a table)
in the second column and everything else in the first; a `two-col` slide with
no media flows its text into two columns.

```markdown
<!-- layout: big-number -->

**98.7%**

of Thursday promises kept in the first month.
```

```markdown
<!-- layout: quote -->

> The customer never saw the declined card. She saw Thursday.

Route 7 courier log
```
