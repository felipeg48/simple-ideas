---
title: Privacy and your data
description: What this site and the si tool collect (nothing), and what leaves your machine only when you ask for it.
---

# Privacy and your data

si is local-first. It runs on your machine, keeps your files there, and
sends nothing anywhere unless you ask it to. There is no account, no
telemetry and no usage tracking, in the tool or on this site.

## This site

simple-ideas.org is a static site hosted on GitHub Pages. It sets no
cookies, runs no analytics and has no forms. GitHub, as the host, keeps its
own server logs (such as IP addresses) under the
[GitHub Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement).
A few example pages load the pinned Mermaid script from cdn.jsdelivr.net to
draw their diagrams.

## The si tool

### What stays on your machine

| What | Where |
|---|---|
| Your Markdown, decks, pages and builds | your folders |
| Settings | `si.conf` in your project, and `~/.si/si.conf` |
| Logins and API keys for publishing and cloud voices | `~/.si` (files only you can read, mode 0600); never in a project's `si.conf` |
| Kokoro voices, the voice cache, the download caches | `~/.si` |
| Review comments, poll and Q&A results, class rosters | `.si/` in your project |
| Speed, voice and other viewer choices on a built page | the viewer's browser (localStorage) |

`si play`, `si serve` and si Studio listen on your own computer
(127.0.0.1) unless you choose otherwise (`--addr`, `--remote`,
`--audience`, `--tunnel`).

### What leaves your machine, and only when you ask

| When you… | si contacts | What is sent |
|---|---|---|
| use a cloud voice (`voice: openai:…`, `elevenlabs:…`) | that provider | the text to be spoken, with your API key |
| run `si publish` | LinkedIn, dev.to, Hashnode, Ghost, WordPress or SkillJar | the post or document you publish (always a dry run unless `--yes`) |
| start `--tunnel` | Cloudflare (a quick tunnel) | the pages and traffic of that session |
| use `--https=tailscale` | your Tailscale network | the session's traffic |
| set `fonts:` | Google Fonts, Bunny, Fontshare or Fontsource (jsDelivr) | a download request for the font, at build time |
| build a page with a Mermaid fence | cdn.jsdelivr.net, from the viewer's browser | a request for the pinned script |
| include `gist:`, `github:` or `https://` sources | GitHub or that address | a download request for the pinned file |
| set a results webhook (courses, labs, demo tests) | the address you give | the results you chose to send |
| run `si voice install kokoro` | Python's package index and GitHub | download requests for the voice and its models |
| run a sandbox lab or class | Docker registries | image pulls for the lab's box |
| run `si selfupdate`, or open si Studio for the Mac | api.github.com | a check for the latest release |

Every one of these services has its own terms and privacy policy, and they
apply to what you send them.

## Questions

Open an issue on [GitHub](https://github.com/felipeg48/simple-ideas/issues).
See also the [license and disclaimer](license.md).
