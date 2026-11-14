# si publish

`si publish` sends a deck or an article to LinkedIn, dev.to, Hashnode, Ghost or
WordPress, and a course to Skilljar. It is
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
si publish article.md --to ghost --yes             # a draft on your Ghost site; --public publishes
si publish article.md --to wordpress --yes         # a draft on your WordPress site
si publish web-course --to skilljar --base-url https://files.example.com/web-course/   # a course; see below
si publish login linkedin|devto|hashnode|ghost|wordpress|skilljar …   # see below
si publish logout [linkedin|devto|hashnode|ghost|wordpress|skilljar]
```

## What is sent

| Source | `--to linkedin` | `--to devto` | `--to hashnode` | `--to ghost` | `--to wordpress` |
|---|---|---|---|---|---|
| deck | document post: the deck as PDF (a swipeable carousel in the feed), text from `linkedin:` / `description:` / the title | — | — | — | — |
| article, guide page, page | link post to `--url` (or `canonical:`); `--copy` for LinkedIn's article editor | article (draft unless `--public`), tags from `tags:` (first 4), `canonical_url` | post, tags from `tags:`, `originalArticleURL`, subtitle from `description:` | post as HTML (draft unless `--public`), tags, excerpt, `canonical_url` | post as HTML (draft unless `--public`), excerpt |

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

## Ghost (Admin API)

`POST <site>/ghost/api/admin/posts/?source=html` with `Accept-Version: v5.0`
and `Authorization: Ghost <token>`. The token is a JWT si makes for each
request from the Admin API key (`id:secret`, Settings › Integrations › Add
custom integration): HS256 with the hex-decoded secret, header `kid` = the
id, payload `iat`, `exp` five minutes later and `aud: /admin/`. The body is
`{"posts":[{title, html, status, custom_excerpt, canonical_url, tags}]}`;
`source=html` has Ghost convert the HTML. A draft unless `--public`
(`status: published`). Login: `si publish login ghost --url https://site
--token <Admin API key>` (si checks the key's shape; it is not sent until you
publish).

## WordPress (REST API)

`POST <site>/wp-json/wp/v2/posts` with basic auth: your user name and an
**application password** (Users › Profile › Application Passwords; WordPress
5.6+, over HTTPS). The body is `{title, content, status, excerpt}`: `content`
is the article as HTML, `status` `draft` unless `--public` (`publish`). Tags
are not sent: the API wants tag ids; si notes it. Login: `si publish login
wordpress --url https://site --user you --token "abcd efgh ijkl …"`.

## Skilljar (REST API, checked 2026-11-14)

A course folder (`_course.md`) goes to Skilljar:

```bash
si publish login skilljar --token <API key>      # Skilljar dashboard: Organization settings, API Credentials
si publish web-course --to skilljar --base-url https://files.example.com/web-course/   # dry run
si publish web-course --to skilljar --base-url https://files.example.com/web-course/ --yes
```

Built from Skilljar's public OpenAPI description
(<https://api.skilljar.com/docs/>, spec at
<https://api.skilljar.com/docs/schema.yml>) and its getting-started article
(<https://support.gainsight.com/Skilljar/Develop_and_Customize/API/Getting_started_with_the_Skilljar_API>):

- Base URL `https://api.skilljar.com/v1`; HTTP Basic with the API key as the
  user name and an empty password; JSON in and out.
- `POST /courses` (`title`, `short_description`,
  `enforce_sequential_navigation: false`, `manual_course_time` from
  `duration:`, 15 characters at most), then `PATCH /courses/{id}` on the next
  publish. Skilljar has no course level field, so `level:` is not sent.
- Lessons are one ordered list per course: `POST /lessons` with `course_id`,
  `title`, `order`, `type`, `time_seconds`; `PATCH /lessons/{id}` on the next
  publish; `DELETE /lessons/{id}` for a lesson no longer in the course.
- A module is a `SECTION` lesson; a reading is an `HTML` lesson
  (`content_html`: the page's text, pictures inlined); a video is an `ASSET`
  lesson (`POST /assets` with `content_url`, then `content_asset_id`); a
  slides, lab or quiz lesson, and a reading with a quiz in it, is a
  `WEB_PACKAGE` lesson (`POST /web-packages` with `content_url` and
  `web_package: {title, redirect_on_completion, sync_completion}`, then
  `content_web_package_id`): a SCORM 1.2 zip of that one page.
- **The API takes no file uploads**: Skilljar fetches each video and package
  from `content_url`. si writes them into `<course>/.si/publish/skilljar/`
  (in a dry run too); put that folder online and pass its address as
  `--base-url`. `--yes` refuses to run without it when the course has a video
  or a package.
- Ids are kept in `<course>/.si/publish/skilljar.json` (the course, each
  lesson, each asset or package with its file's SHA-256), so a re-publish
  updates in place and fetches only the files that changed. Instructor demos
  are never published.
- Not sent: publishing to a domain (`/domains/{name}/published-courses`); do
  that once in the Skilljar dashboard.

**Needs a check against a real account.** The client follows the published
spec and is tested against fakes only. The spec does not list the values of
a web package's or asset's processing `state`, so si does not wait for
processing; it does not say whether a `SECTION` must come first or whether
`order` must be contiguous (si sends 1, 2, 3 … with each section before its
lessons). Old assets and packages are left in Skilljar after a changed file
is fetched again (the API refuses to delete one still in use).

## Tests

`internal/publish` tests every provider against `httptest` fake servers
(headers, payload shapes, the upload chain, errors, OAuth state, dry runs that
send nothing and mask secrets, the 0600 credentials file). Nothing in the test
suite talks to a real service.
