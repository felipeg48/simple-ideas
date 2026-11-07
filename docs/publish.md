# si publish

`si publish` sends a deck or an article to LinkedIn, dev.to or Hashnode. It is
always a **dry run** (it prints the exact requests, secrets masked) unless you
add `--yes`. Logins live in your config dir (`~/Library/Application
Support/si/publish.json` on macOS, `~/.config/si/publish.json` on Linux), file
mode 0600, never in the repo. Stdlib only, no SDKs.

```bash
si publish talk.md --to linkedin                  # dry run: a document post with the deck's PDF
si publish talk.md --to linkedin --yes            # send it (renders the PDF with headless Chrome first)
si publish article.md --to linkedin --url https://me.dev/retry --yes   # a link post
si publish article.md --to linkedin --copy        # the article as rich text on the clipboard (macOS)
si publish article.md --to devto --base-url https://me.dev/retry      # dry run; a draft unless --public
si publish article.md --to hashnode --canonical https://me.dev/retry --yes
si publish login linkedin|devto|hashnode …        # see below
si publish logout [linkedin|devto|hashnode]
```

## What is sent

| Source | `--to linkedin` | `--to devto` | `--to hashnode` |
|---|---|---|---|
| deck | document post: the deck as PDF (a swipeable carousel in the feed), text from `linkedin:` / `description:` / the title | — | — |
| article, guide page, page | link post to `--url` (or `canonical:`); `--copy` for LinkedIn's article editor | article (draft unless `--public`), tags from `tags:` (first 4), `canonical_url` | post, tags from `tags:`, `originalArticleURL`, subtitle from `description:` |

The article body is the Markdown with its includes expanded and the title
heading dropped. Relative links and images become absolute against
`--base-url` (where the article's files are online, e.g. the `si build`
output you deployed); without it si warns about every relative image.

Front matter keys: `linkedin:` (the post text), `tags:` (comma-separated),
`canonical:` (the original URL), `cover:` (the picture of a LinkedIn link
post; default: the article's first local PNG, JPEG or GIF). The picture goes
up through the Images API (`POST /rest/images?action=initializeUpload`, then
a `PUT` of the bytes) and the post's `content.article.thumbnail` is its
`urn:li:image:…`.

## LinkedIn (checked 2026-11-07)

- **App**: create one at linkedin.com/developers, add the products *Share on
  LinkedIn* (scope `w_member_social`) and *Sign In with LinkedIn using
  OpenID Connect* (scopes `openid profile`, to learn your member id).
- **Login**: `si publish login linkedin --client-id ID --client-secret SECRET`
  (or `SI_LINKEDIN_CLIENT_ID` / `SI_LINKEDIN_CLIENT_SECRET`) runs the
  3-legged OAuth flow: si opens
  `https://www.linkedin.com/oauth/v2/authorization` and waits on
  `http://localhost:8723/callback` (`--port` changes it; the app must list
  that exact redirect URL), checks `state`, exchanges the code at
  `/oauth/v2/accessToken`, then reads your id from `GET /v2/userinfo` (`sub`
  → `urn:li:person:<sub>`). Access tokens last **60 days**; refresh tokens are
  only for approved partners, so log in again when si says it expired.
  Alternatively `--token <token>` stores a token from the developer portal's
  token generator (with the same three scopes).
- **Headers** on every REST call: `Authorization: Bearer …`,
  `Linkedin-Version: 202610` (YYYYMM; `SI_LINKEDIN_VERSION` overrides; LinkedIn
  sunsets a version about a year after release), `X-Restli-Protocol-Version: 2.0.0`.
- **Document post** (the deck): `POST /rest/documents?action=initializeUpload`
  with `{"initializeUploadRequest":{"owner":"urn:li:person:…"}}` → `uploadUrl`
  and `urn:li:document:…`; `PUT` the PDF to `uploadUrl` (PDF, PPT(X), DOC(X);
  at most 100 MB and 300 pages); then `POST /rest/posts` with
  `content.media {title, id}`. Organic "carousel" posts are not in the API
  (carousels are sponsored only); a document post shows as the swipeable
  carousel in the feed.
- **Link post** (an article): `POST /rest/posts` with
  `content.article {source, title, description}`. The API does not scrape the
  page, so the title and description come from the front matter; there is no
  thumbnail unless one is uploaded (not done).
- The post id comes back in the `x-restli-id` header; si prints
  `https://www.linkedin.com/feed/update/<id>/`.
- `commentary` uses LinkedIn's "little" text format: si escapes
  `| { } @ [ ] ( ) < > * _ ~ \` so the text shows as written (hashtags stay).
- There is no API for LinkedIn articles (newsletters); `--copy` puts the
  article on the clipboard as rich text to paste into the editor.

## dev.to (Forem API)

`POST https://dev.to/api/articles`, header `api-key: <key>` (create it at
dev.to/settings/extensions; `si publish login devto --token KEY`), body
`{"article":{"title","body_markdown","published","tags","canonical_url","description"}}`.
`tags` is a comma-separated string of at most 4. `published: false` keeps a
draft (si's default; `--public` publishes). There is no image upload
endpoint: images must be absolute URLs (`--base-url`). 201 returns the
article with its `url`.

## Hashnode (GraphQL)

`POST https://gql.hashnode.com`, header `Authorization: <personal access
token>` (hashnode.com/settings/developer). Since 2026 **every API request
needs a Pro plan on the publication** (free API access was retired). Login:
`si publish login hashnode --token PAT --url https://yourblog.hashnode.dev`
looks the blog up once with `publication(host:) { id }`. Publishing runs the
`publishPost(input: PublishPostInput!)` mutation with `title`,
`contentMarkdown`, `publicationId`, `tags [{slug,name}]`, `subtitle`,
`originalArticleURL` and returns `post { id url }`. `publishPost` publishes
immediately (there is no draft flag on it).

## Tests

`internal/publish` tests every provider against `httptest` fake servers
(headers, payload shapes, the upload chain, errors, OAuth state, dry runs that
send nothing and mask secrets, the 0600 credentials file). Nothing in the test
suite talks to a real service.
