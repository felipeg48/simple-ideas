---
title: Website
description: A whole website from a folder of Markdown files: a menu, four navigators, pages, articles, talks and guides inside it, search, light and dark, ready to upload anywhere.
---

# Website

A folder of Markdown files becomes a website: a top bar with your logo and
menu, pages, blog posts, talks and whole guides inside it, search, light and
dark. `si build` writes plain HTML files you can put on any web host (GitHub
Pages, Netlify, a bucket, a USB stick).

```bash
si new mysite --type website     # mysite/ with _site.md, a home and an about page
si play mysite                   # preview it live, reloads on save
si build mysite -o out/          # the site as HTML, ready to upload
```

![A website built with si: the top bar, the home page with an announcement and a badge](img/site-home.jpg)

## The folder

```
mysite/
  _site.md          the menu and the look
  index.md          Home
  about.md
  blog/retries.md   type: article: a byline and a contents column
  talks/journey.md  type: deck: a talk, built as slides
  docs/_nav.md      a guide inside the site, with its own sidebar
```

`_site.md` is front matter plus the menu, a nested list:

```markdown
---
title: Northwind Engineering
description: How Northwind Logistics moves forty thousand orders a day.
navigator: topbar
logo: Northwind Engineering
cta: Run the lab → docs/index
theme: studio
---

- [Home](index.md)
- Writing
  - [Retries without the stampede](blog/retries.md)
  - [The order journey, the talk](talks/journey.md)
- [Docs](docs/)
- [About](about.md)
```

Add a page by writing its file and a menu line; `si new post --type article`
inside the folder adds the line for you.

![An article page of the same site, in dark mode, with its contents column](img/site-blog.jpg)

## The look

| Key | What |
|---|---|
| `navigator:` | `topbar` (logo left, menu, a button right), `topbar-centred`, `mega` (sections open as wide panels) or `sidebar` (a column on the left); on a phone every navigator folds into a **Menu** button |
| `logo:` | your name, a picture, or both: `logo: brand/mark.svg Northwind` |
| `cta:` | the highlighted button: `Label → page` or `→ https://…` |
| `theme:` · `fonts:` | any theme, light and dark, and your own fonts |
| `favicon:` · `og-image:` | the tab icon and the picture link previews show |
| `footer:` | your footer, or a `_footer.md` with links |

Pages use everything else si has: cards, badges, alerts, accordions, tabs,
carousels, code, mind maps, QR codes and embedded talks.

![The same home page on a phone, its menu folded behind a button](img/site-phone.jpg)

[See the whole example on GitHub](https://github.com/felipeg48/simple-ideas/tree/main/examples/site),
or this very site: simple-ideas.org is built with si.

## The rules

- [23. Sites](../rules/23-sites.md)
