# 13. Code, Mermaid, stage diagrams, tables, links

- Fenced code with a language gets a small built-in highlighter (Go, Python,
  JavaScript/TypeScript, Java, Kotlin, C# (`csharp`, `cs`, `c#`), C, C++,
  PHP, Ruby, Swift, Scala, Rust, SQL, YAML, TOML, JSON, Bash, PowerShell,
  Dockerfile, Terraform/HCL (`hcl`, `tf`), Protobuf (`proto`), GraphQL (`gql`),
  F# (`fsharp`, `fs`), Elixir (`ex`, `exs`), CSS, HTML, XML (also `csproj`, `props`, `xaml`, `svg`, `pom`,
  `nuspec`: tags, attributes, comments)) and a language label written as a
  reader expects it (C#, JavaScript, Shell, XML…). Keywords, strings,
  comments and numbers are coloured, and so are calls (the name before a `(`)
  and, in C#, Java, Go, Kotlin, Swift, Scala, JavaScript, C++, Rust and PHP,
  types (PascalCase names); other languages get strings, calls and numbers.
- ```` ```mermaid ```` fences render with Mermaid, themed from the deck tokens
  and re-rendered on light/dark switch. **This is the only network
  resource:** the pinned Mermaid script from jsDelivr is added only to decks
  that have a Mermaid fence. Offline, the diagram source shows as text. Set
  `mermaid: off` to always keep the source, or `mermaid: <url>` to load the
  script from elsewhere.
- ```` ```stage ```` fences generate a themed, animated flow diagram — a
  native, offline, themeable alternative to Mermaid for **simple flows**
  (chains, fan-out, merges). Boxes, arrows, labels and a travelling dot or
  marching dashes are emitted as self-contained SVG coloured with the deck
  tokens; no runtime script, and motion pauses under reduced motion. Use
  `mermaid` for dense graphs instead.

  ````markdown
  ```stage
  dir: LR                              # LR (default) or TB
  Read --> Cache
  Cache -->|hit| Fast[Return fast]     # id[Label] renames; -->|x| labels an edge
  Cache -->|miss| Disk --> Fill --> Fast
  dot Read..Fast 2s                    # a dot loops along the route Read->Fast
  flow Read..Disk                      # marching dashes along a route
  ```
  ````

  (The `#` notes explain the lines; leave them out of a real fence.)

  Nodes appear in first-seen order; `dot`/`flow A..B [dur]` take the shortest
  route A→B. Labels do not wrap: a long one (`Q[Matching queue]`) is set
  smaller so it stays inside its box in every built-in theme, a short one
  keeps the full size. Add `step <node...>` lines to walk the flow on build steps — each
  `step` line is one click. An optional op leads the line:
  `highlight` (default, accent outline), `ok` (green), `alert` (red, pulsing),
  `show` (the node starts hidden and fades in), `hide` (fades out):

  ```stage
  dir: LR
  Submit --> Check --> Deploy --> Done
  step Submit
  step ok Check
  step alert Deploy
  step show Done
  ```

  A bad block (unknown node, no route, malformed `dir`, unknown `step` node) is
  a `si check` problem. The generator lays out chains, fan-out and merges with
  longest-path layering (no crossing minimisation); diagrams size like any
  figure, so give a wide one its own `##` slide. Full reference with many
  examples: `docs/stage-diagrams.md`.
- GitHub tables, strikethrough (`~~x~~`), autolinks and task lists work.
- Relative links to `.md` files point at the generated `.html`:
  `[appendix](appendix.md)` → `appendix.html`.
- Raw HTML is allowed.


### Highlighted words: `==text==`

Two `=` on each side mark a phrase to stand out, in a heading or in body text:

```markdown
## Five things ==to keep==

Every team was green; ==the customer still waited==.
```

It renders as `<span class="hl">…</span>`, so a theme decides the look: the
built-in themes underline it with a band of the accent colour, and a theme can
sweep the band in as the slide arrives. It nests (`==**bold** words==`), stays
on one line, and leaves code, a single `=` and `===` alone. Plain-text copies
of a title (the tab, the teleprompter, search) drop the marks.

### Code from files, highlighted lines, callouts and tabs

These work on slides and in documents (labs, articles, guides).

**Code from a file.** An include of anything that is not `.md` / `.adoc`
becomes a code block, highlighted by the file's extension:

```markdown
<!-- include: src/App.java -->                     the whole file
<!-- include: src/App.java lines=5-10 -->          lines 5 to 10 (also 5-, -10, 5-10,20-24)
<!-- include: src/App.java tag=retry -->           between tag::retry[] and end::retry[]
<!-- include: src/App.java tag=retry hl=3-6 title=Java lang=java -->
```

- **Prefer `tag=` to `lines=`**: a tag moves with the code, line numbers do
  not. Mark the region in a comment of the file's language
  (`// tag::retry[]` … `// end::retry[]`, `# tag::retry[]`); marker lines are
  never shown; `tag=a,b` joins several in file order.
- A `lines=` or `tag=` include shows the file's own line numbers. The
  indentation every shown line shares is removed (`indent=keep` keeps it). The
  title is the file name unless `title=` says otherwise.
- The language comes from the file's name (`.cs` → C#, `.csproj`, `.config`,
  `.props`, `.xaml` → XML, `Dockerfile` → dockerfile, `.sh` → bash, `.yml` →
  YAML), then from its first characters (`<?xml` or a tag → XML, `{` or `[` →
  JSON, a `#!` line → bash or Python). `lang=` overrides both, here and on
  `<!-- code: -->`: `<!-- include: ../code/NuGet.config title=NuGet.config lang=xml -->`.
- `si check` reports a missing file, a range past the end, a tag that is not
  found or not closed, and an unknown option.

**Fence attributes.** After the language of a code fence:

````markdown
```java title="Retry.java" hl="3,5-6" linenos start=12
````

`title=` shows in the code block's header bar (documents), `hl=` highlights
lines (counted from the first line shown), `linenos` shows line numbers, from
`start=` (default 1). Copy never copies the numbers.

The language may be left out or written the pandoc way; all of these read
the same: ```` ```xml title="config.csproj" ````, ```` ``` title="config.csproj" ````
(no language: the title's extension names it, `.csproj` → `xml`, `.go` → `go`,
`Dockerfile` → `dockerfile`), ```` ```{.xml title="config.csproj"} ```` and
```` ```xml {title="config.csproj"} ````.

**Callouts.** End a code line with `<N>`, bare or as its comment (`// <1>`,
`# <1>`, `-- <1>`, `<!-- 1 -->`), and write the explanations right after the
block, one `<N> text` line each:

````markdown
```java
var key = order.id();                     // <1>
sleep(Duration.ofSeconds(30));            // <2>
```
<1> The order id is the idempotency key.
<2> One retry only, thirty seconds later.
````

The markers become numbered badges (and leave the code, so Copy copies clean
code); hovering a badge or its line lights up both. `si check` reports a
marker without a line and a line without a marker.

**Tabs.** `<!-- tabs -->` turns the code blocks that follow it (with their
callouts, up to the first other block) into one group of tabs, labelled by
`title=` or the language:

````markdown
<!-- tabs -->
<!-- include: code/Retry.java tag=retry title=Java -->
<!-- include: code/retry.py tag=retry title=Python -->
```bash title="curl"
curl -X POST localhost:8082/orders/4711/pay
```
````

In a document, choosing a tab switches every group on the page that has a tab
with the same label, and the choice is remembered for the next page. On a
slide the group switches on its own. Print shows every tab, labelled. ← / →
move between tabs.

**Code and diffs from git.** Paste a GitHub permalink (on GitHub: the
file, then **Copy permalink**), or name the repo, path and commit:

```markdown
<!-- code: https://github.com/acme/app/blob/3f2a9c1/src/App.java#L5-L10 -->
<!-- code: https://github.com/acme/app/blob/3f2a9c1/src/App.java tag=retry hl=2 -->
<!-- code: path=src/App.java ref=3f2a9c1 tag=retry -->        repo: from the front matter
<!-- code: repo=../app path=src/App.java ref=HEAD -->         a local clone (git show)
<!-- diff: https://github.com/acme/app/commit/3f2a9c1 path=src/App.java -->
<!-- diff: https://github.com/acme/app/compare/v1.1...v1.2 path=src/ -->
<!-- diff: repo=../app ref=v1.1..v1.2 path=src/App.java -->
```

- `code:` takes every option of a code include (`lines=`, `tag=`, `hl=`,
  `title=`, `lang=`, `linenos`); a `#L5-L10` in the permalink is `lines=5-10`.
  The code block's header links to the source.
- `diff:` shows a commit's changes, or the changes from A to B, filtered to
  `path=` (a file, or a folder: everything under it): file headers, hunk
  headers, removed and added lines with their old and new line numbers. Copy
  copies the patch. A ```` ```diff ```` fence renders the same way.
- `repo:` in the front matter (or `_nav.md` for a whole guide) is the default
  repo: `OWNER/REPO`, a github.com URL, or a local folder holding a clone.
- Everything is **fetched when you build** (and on every `si play` / `si
  serve` request), cached on disk, and inlined: a reader never fetches it. A
  commit is cached for good; a branch or tag (`main`, `v1.2`) for an hour, and
  `si check` warns that it can move: with `lines=` the shown code would drift,
  so pin a commit.
- **Private repos**: set `GITHUB_TOKEN` (or `GH_TOKEN`), or log in with the
  GitHub CLI (`gh auth login`); si then uses the GitHub API. A local clone
  needs only `git`.
- `--offline` (with `si build`, `si check`, `si serve`) uses the cache only and
  reports anything that was never fetched; without it, a network failure falls
  back to a cached copy when there is one.

**Slides in a document.** A document (lab, article, guide page) can show a
deck, or some of its slides, playable in place:

```markdown
<!-- slides: ../decks/talk.md -->                     the whole deck
<!-- slides: ../decks/talk.md 23-24 -->               slides 23 to 24 (or one: 23)
<!-- slides: ../decks/talk.md#live-a-first-broker --> the slide with that heading id
```

The deck is rendered when the page is built, self-contained in a frame of the
deck's `aspect`: click it, then the arrow keys step through slides and build
steps; canvases animate. The caption names the deck. `si check` reports a
missing deck, a range past its end and an unknown heading id. On a slide this
directive is a problem (decks do not nest).

**AsciiDoc** maps onto the same features: `include::App.java[lines=5..10]`,
`include::App.java[tag=retry]` (`tags=a;b`), `.Retry.java` before a listing
(the title), `[source,java,highlight=3;5..6]` and `%linenums`; si's own
fence options go in the same brackets (`[source,go,window=mac,numbers]`);
`<1>` callouts are written the same way.
