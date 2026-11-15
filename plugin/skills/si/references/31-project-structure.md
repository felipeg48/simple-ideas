# 31. Project structure

A course, a talk series, a training: the slides, the labs, the website, the
videos and the zips you hand over each want a folder of their own. A
`structure:` block in the project's `si.conf` is that map: one folder per
line, nested by indentation, its name ending in `/`; after it, what goes
there.

```text
structure:
  rabbitmq/
    md/           markdown
    slides/       deck
      pdf/        deck:pdf
    labs/         lab
    html/         site
      images/     image
      animations/ animation
      pdfs/       deck:pdf
    videos/       video
    dist/         html/:zip, deck:zip, course:scorm
```

`si build rabbitmq` then writes the decks into `slides/`, their PDFs into
`slides/pdf/` **and** `html/pdfs/`, the labs into `labs/`, the website into
`html/` with its pictures in `html/images/`, the videos into `videos/`, and
last the zips into `dist/`: `rabbitmq-html.zip` (the whole `html/` folder)
and `rabbitmq-deck.zip` (every deck with what it links to, the index and
labs included). `index.html`
stays at the root, its tiles pointing into the folders. The example project
is `examples/structure/`.

**No block, no change.** Without `structure:` a build writes what it always
did, every page side by side in the output folder.

**The top folder** names the project (the `<project>` of the zip names) and
is the folder that holds `si.conf`; a map without one top folder uses the
folder's own name. Paths of `html/:zip` are relative to it.

**The words after a folder** (a comma list):

| Kind | What goes there |
|---|---|
| `markdown` | sources: where `si new` puts them (sources may live in any folder of the map) |
| `deck`, `lab`, `guide`, `course`, `site`, `article`, `timeline`, `infographic`, `mindmap`, `book`, `demo` | the pages of that type (`type:` in front matter; a folder with `_nav.md`, `_course.md` or `_site.md` for guide, course, site, book). One guide, course, site or book fills its folder; several get a subfolder each |
| `image`, `animation`, `video` | the pictures (`.png`, `.jpg`, `.webp`), animations (`.svg`, `.gif`, and every SVG canvas also as a file) and videos and sound the pages use. With such a folder, pictures are linked from it, not inlined (as `--no-inline-media`); canvases stay inline in the slides, they must for their states |
| `kind:format` | a derived file: `deck:pdf`, `deck:mp4`, `deck:handout`, `course:scorm`, `course:pdf` (the workbook), `book:pdf`, `timeline:png` / `pdf` / `svg`, `infographic:png` / `pdf` / `svg`, `mindmap:png` / `pdf` / `svg` / `mp4`. Naming one is the same as its flag (`--pdf`, `--mp4`, `--handout`, `--scorm`, `--png`, `--svg`): the map builds it |
| `kind:zip` | a bundle: every output of that kind (pages and their derived files) with the pictures and videos they link to, and the built pages they link to (the index, a lab), then the pages and files those link to, and so on (each once), paths as in the project, so every local link inside the bundle works: `<project>-<kind>.zip` |
| `folder/:zip` | a bundle: a folder of the map, zipped whole after it is built: `<project>-<folder>.zip` |

A kind listed in two folders is written to both (copies). A derived file the
map does not place is not kept, unless you asked for it with its flag (then
it goes where it always did); a page of a kind with no folder goes to the
project root.

**The build, in order:**

1. read the sources, wherever they sit in the project (by `type:`);
2. render each page into its kind's folder;
3. copy the pictures, animations and videos into theirs;
4. make the derived files (headless Chrome: PDF, MP4, PNG, SVG);
5. copy what is listed twice;
6. zip the bundles, last, from the finished folders.

**Links follow the files.** Every page's links are rewritten from its new
folder: pictures, videos, PDFs, the other decks and labs, the index tiles
and their PDF badges, find (`/`), the navigation of guides, courses and
sites, and the shared `_si/` of `--share-assets` (a copy in each top folder
that uses it). A page links the nearest copy of a file listed twice, so
`html/` zipped alone still finds its PDFs in `html/pdfs/`.

**The output folder.** The map is laid out in the project folder itself;
`-o out/` lays it out under `out/` instead.

**From inside the project.** `si build md/intro.md` or `si build md/site`
(a page, or a guide, course or site folder, in one of the map's folders or
at the project root) finds the project: the nearest `si.conf` above it with
a `structure:` block, looking no higher than the top of the git repository
or your home folder. It then builds the whole project with its map, as `si
build rabbitmq` does, so the page lands in its kind's folder and its links,
the index and the bundles stay whole. A page outside the map's folders (a
`notes/` the map does not name) builds as it always did.

**`si new rabbitmq --structure`** writes `rabbitmq/si.conf` with the map
above, makes its folders and puts the starter deck in `md/`. Plain `si new
rabbitmq` does the same when `rabbitmq/si.conf` already has a
`structure:` block (its folders, its markdown folder).

**`si check`** checks every folder of sources and warns about the map: an
unknown kind or format, an indentation that does not line up, a folder name
without its `/` (each with its line in `si.conf`), a bundle naming a folder
the map does not define, a folder bundle that would be empty, a source of a
kind with no folder (it builds into the project root), and a source sitting
in a folder meant for another kind (a lab in `slides/`). A kind bundle for a
kind the project has no source of yet (`course:scorm` before the
`_course.md`) is a note, not a warning: `note: … nothing to bundle yet`, so
the map can be written ahead of the content and a new `--structure` project
checks clean.

**`si play` and `si serve`** serve the sources as always, live, from where
they sit: the map is about what `si build` writes.
