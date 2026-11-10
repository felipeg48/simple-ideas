# 26. Publishing

`si publish` posts a deck to LinkedIn as a document post (the deck as PDF,
which the feed shows as a swipeable carousel) and an article to LinkedIn as a
link post, or to dev.to, Hashnode, Ghost or WordPress. It is a **dry run** that prints the exact
requests unless you add `--yes`.

```markdown
---
title: The Thursday Promise
description: How one order crosses five hand-offs and still arrives on Thursday.
linkedin: Our checkout keeps a promise across five hand-offs. Here is how, in 22 slides. #logistics #retries
tags: go, reliability, payments
canonical: https://northwind.dev/talks/thursday
---
```

```bash
si publish talk.md --to linkedin                 # dry run
si publish talk.md --to linkedin --yes           # renders the PDF, uploads, posts
si publish article.md --to devto --base-url https://northwind.dev/retry --yes   # a draft; --public publishes
si publish article.md --to linkedin --url https://northwind.dev/retry --yes
si publish article.md --to ghost --yes           # a draft on your Ghost site; --public publishes
si publish article.md --to wordpress --public --yes
si publish article.md --to linkedin --copy       # rich text for LinkedIn's article editor
si publish login linkedin --client-id ID --client-secret SECRET   # or --token
si publish logout
```

- The post text is `linkedin:`, else `description:`, else the title.
- An article's body is its Markdown with includes expanded and the title
  heading dropped; relative links and images become absolute against
  `--base-url` (si warns about each relative image without it).
- Ghost and WordPress get the article as HTML (si renders the Markdown).
- `tags:` go to dev.to (the first four), Hashnode and Ghost (WordPress wants
  tag ids, so add them there); `canonical:` (or
  `--canonical`) becomes `canonical_url` / `originalArticleURL`, and is the
  link of a LinkedIn link post when `--url` is not given.
- A LinkedIn link post carries a picture: `cover:` (a PNG, JPEG or GIF next
  to the article, e.g. `cover: ../assets/dusk.jpg`), else the article's first
  local picture. si uploads it through LinkedIn's Images API first; the dry
  run shows that request and the `thumbnail` it attaches.
- Logins are stored in your config dir with mode 0600, never in the repo.
  LinkedIn logins last 60 days; the Hashnode API needs a Pro plan. Ghost:
  `si publish login ghost --url https://site --token <Admin API key>`;
  WordPress: `si publish login wordpress --url https://site --user you
  --token "<application password>"`.

Details, scopes and the exact requests: `docs/publish.md`.
