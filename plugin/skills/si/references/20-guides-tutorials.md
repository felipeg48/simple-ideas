# 20. Guides (tutorials)

A guide is a **folder with a `_nav.md`**: a tutorial of several pages read in
order, with a navigation sidebar, a breadcrumb, Previous / Next links and
shared header and footer.

```text
order-journey/
  _nav.md          the order of the pages, and the guide-wide settings
  _header.md       optional: a band at the top of every page
  _footer.md       optional: a band above the footer of every page
  index.md
  install.md
  happy/place.md   pages may sit in subfolders
  reset.adoc       and may be AsciiDoc
```

`_nav.md` is front matter plus a markdown list. An item with a link is a
page; an item without one is a section heading; indentation nests.

```markdown
---
title: The order journey
description: Follow order #4711 from checkout to the doorstep.
version: v1.0
theme: paper
footer: Northwind Logistics engineering
edit-url: https://github.com/acme/order-journey/edit/main/docs
---

- [Introduction](index.md)
- Getting started
  - [Install the lab](install.md)
- The happy path
  - [Place an order](happy/place.md)
  - [Pay for the order](happy/pay.md)
- [Reset the lab](reset.adoc)
- [Status page](https://status.example.com)
```

- **Inherited settings.** Every page starts from the front matter of
  `_nav.md` (theme, mode, accent, footer, year, copyright, edit-url, author,
  version, …) and its own front matter overrides it key by key. Pages are
  `type: guide` unless they say otherwise (a `type: lab` page keeps its LAB
  badge). The guide's `title` and `description` stay the guide's: each page has
  its own `# Title` and `description:`.
- **Navigation.** The sidebar follows `_nav.md`: sections as small headings,
  the current page marked, pages the reader has opened marked as read (kept in
  that browser), and an "N of M read" count. On narrow screens it folds above
  the page. A link to an outside URL opens it.
- **Page.** A breadcrumb (guide › section › page), the title, the
  `description:` as a lede, the page, then **Previous / Next** in nav order
  and, with `edit-url:`, an "Edit this page" link (`edit-url` + the page's path
  in the guide). "On this page" sits on the right on wide screens.
- **Bar and partials.** A top bar shows the guide title (a link to the first
  page), the `version` and the light / dark button. `_header.md` and
  `_footer.md` are markdown, rendered on every page with their links resolved
  from the guide folder; `header: off` or `footer-page: off` in a page's front
  matter drops one on that page.
- **Links.** Link to the `.md` (or `.adoc`) file, with an anchor if you like
  (`[ports](install.md#step-2-start-the-stack)`); it becomes `.html`. Heading
  anchors are GitHub-style (lower case, spaces to hyphens, punctuation dropped).
- **`si check <guide>`** reports a page that `_nav.md` lists but does not
  exist (or is listed twice, or is outside the folder), a link between pages
  to a missing page or to an anchor the target does not have, and, as a
  warning, a page in the folder that `_nav.md` does not list (it is not built).
  `_`-prefixed files and `README.md` are never pages.
- **`si build <guide> -o out/`** writes one self-contained HTML file per page,
  mirroring the folders (`happy/place.md` → `out/happy/place.html`), and an
  `index.html` (the `index.md` page, or a redirect to the first page). In
  `si build <folder>`, each subfolder that holds a `_nav.md` is built into
  `out/<subfolder>/` and becomes one GUIDE tile on the index.
- **`si serve <folder>`** serves decks, articles, guides and labs with live
  reload; a guide folder URL opens its first page.

Example: `si --export example --type guide` (or `--type all` for every kind
side by side, then `si serve si-example`).
