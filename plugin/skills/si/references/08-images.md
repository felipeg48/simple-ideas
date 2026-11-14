# 8. Images

A paragraph that holds only one image becomes a figure; the image **title**
becomes the caption:

```markdown
![Warehouse floor plan](assets/floor.svg "Six aisles, one packing table")
```

Images inside a sentence stay inline. Size and place an image with the
`image` directive on the line before it (the same "applies to the next block"
rule as steps):

```markdown
<!-- image: width=46% align=right -->
![Floor plan](assets/floor.svg "Two pickers meet in the middle")

- Text flows beside the image and moves below it on a phone
```

| Option | Values | Effect |
|---|---|---|
| `width` | `40%`, `24em`, `30cqi` (never `px`) | figure width |
| `align` | `left`, `right`, `center` | `left`/`right` float beside the text; stacks on narrow screens |
| `ratio` | `16:9`, `4:3`, `1:1` | crops the image into an aspect-ratio box |
| `fit` | `cover`, `contain` | how the image fills the ratio box |
| `position` | `center`, `left-bottom` | which part stays visible when cropped |
| `poster` | `assets/still.jpg` | a video's still before it plays |

**Videos.** An image whose file is a video (`.mp4`, `.webm`, `.mov`, `.m4v`,
`.ogv`) is a player, on a slide and in a document alike: the title (else the
alt text) is the caption, `poster=` in the `image` directive its still.

```markdown
<!-- image: poster=assets/cluster.jpg -->
![The cluster, narrated](assets/cluster.mp4)
```

It renders `<figure class="si-image" data-video><video controls
preload="metadata" playsinline src="assets/cluster.mp4" …></video><figcaption>`.

A video without a `poster=` still opens on its first frame rather than on
black: `si build` uses `cluster.poster.jpg` beside the video, and when there is
none it writes one from the frame at 0.5 s with ffmpeg (skipped quietly
without ffmpeg). It is made once; delete it to make it again, or give
`poster=` to choose the picture. `si build talk.md --mp4` writes
`talk.poster.jpg` too: the deck's first slide at 1920×1080. A `poster=` file
that is missing is a `si check` problem.
A video is never inlined as a data URI: `si build` copies it beside the page
(in every media mode), and so does the `src` of a raw `<video>`, `<source>`
or `<audio>` you write yourself.

**Linked files.** A link to a local file si does not render (`[The slides
(PDF)](assets/talk.pdf)`, a `.zip`, a `.csv`, an `.mp4`) is copied beside the
built page the same way, keeping its path (a file outside the page's folder
goes to `media/`). A linked file that is missing fails `si check`. Links to
pages (`.md`, `.adoc`, `.html`), folders and outside URLs are left alone.

Side by side: put images in columns (`two-col` + `<!-- column -->`) or use
`split` (text left, image right). Full bleed: `<!-- layout: image -->`.

Every image, background, poster and SVG is embedded as a data URI, so one
deck is one self-contained file. A file used on several slides (the same
photo as two backgrounds and a figure) is embedded once. Files above 2 MB draw
a warning; build with
`--no-inline-media` to copy media next to the HTML instead. `si play` links
files from the deck directory directly. A missing file fails `si check`.
