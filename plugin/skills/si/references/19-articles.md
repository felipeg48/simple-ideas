# 19. Articles

`type: article` is one long read: a blog post, a write-up, a case study.

```markdown
---
type: article
title: One retry fixed nine in ten night-time declines
description: Order #4711 failed at 21:03 and was paid at 21:05.
author: Jonas Berg · Payments team
date: 5 November 2026
---

# One retry fixed nine in ten night-time declines

For a year we treated every night-time card decline as a fraud signal …

## The declines were timeouts
```

- The header holds the title, the **lede** (`description:`, one or two
  sentences; it is also the page's `<meta name="description">` and the
  subtitle of its index tile), the byline (`author`, `date`, `version`) and
  the **reading time**, counted at 230 words a minute on the text outside code
  blocks.
- "On this page" (the `##` / `###` headings) appears on the right when the
  article has three or more `##` sections, and above the text on narrow
  screens; with fewer sections there is none.
- The text column is narrower than a lab's (88% of `--si-doc-measure`) and
  slightly larger. Everything else is as in a lab (§18): callouts, code with
  copy, key/value tables, figures, includes, print, theme and dark mode.
- `## Step N:` headings stay plain in an article.
- `si build` writes `<name>.html` with an ARTICLE badge on the index tile.

Example: `si --export example --type article`.
