---
title: si with your AI agent
description: Do you have an AI agent? Give it the si skill and ask for a talk, an article or a mind map; it writes the Markdown, checks it with si check and fixes what it finds.
---

# si with your AI agent

Do you have an AI agent (Claude Code, Codex, Cursor, Copilot, Gemini CLI)?
Ask it for the talk. si's format is plain Markdown, `si check` tells the
agent exactly what is wrong and where, and the si skill teaches it the rest:
the layouts, the teleprompter script, canvases, mind maps, code shows, QR
codes. You review the result in `si play`.

## Claude Code: the plugin

```text
/plugin marketplace add felipeg48/simple-ideas
/plugin install si@simple-ideas
```

The `si` skill loads by itself when you ask for a deck, slides, a talk, an
article or a mind map, or work in a folder with `si.conf` or `_nav.md`. You
also need the [si binary](download.md): the agent runs it to check and
build.

## Any agent with skills: the skill folder

```bash
si --export skill                        # into .claude/skills/si/ (this project)
si --export skill -o ~/.claude/skills/si # every project
```

A `SKILL.md` (when to use si, the work loop, the writing guide) and a
`references/` folder with one file per section of [the rules](rules/), so the
agent reads only the part it needs. Run it again after you update si.

## Codex, Cursor, Copilot, Gemini CLI: AGENTS.md

```bash
si --export agent-rules                  # CLAUDE.md and AGENTS.md in this folder
```

The same writing guide as a project file, the one those agents read on
their own. It never overwrites a file that is already there.

## What to ask

```text
Turn notes.md into a 20-minute talk. One idea per slide, a big number where
there is one, and end with a QR code to https://example.com/slides.
```

```text
Move deck.pptx (exported as outline.txt) to si: cut the duplicates, keep
every number and SQL snippet, and replace the screenshots with diagrams.
```

```text
Write an article from this README, with a mind map of the architecture and a
code show of the retry change in diff.patch.
```

```text
Add Notes: to every slide of talk.md: what I say to the room, word for word,
with a [CLICK] where I advance a step.
```

## How a good agent works with si

The skill makes this its loop, every time:

1. Write or edit the Markdown.
2. `si check`, and fix every problem it reports (a file, a line, what is wrong).
3. Look at it: build it and take a screenshot, or ask you to run `si play`.
   A deck that passes `si check` can still look wrong.
4. Only then say it is done.

And it keeps si's two rules about notes: `Notes:` is what you say to the
room, word for word, never stage directions; `Points:` is the short list for
you (pace, emphasis, likely questions).

## The MCP server: si mcp

`si mcp` gives an agent tools instead of commands, in the folder it starts
in (paths outside it are refused):

| Tool | What it does |
|---|---|
| `si_check` | checks a file or folder; the problems come back as data (file, line, message) |
| `si_rules` | one section of the format, by number, title word or search |
| `si_new` | a starter: deck, article, guide, book, site, mindmap, codeshow, … |
| `si_build` | self-contained HTML (and PDF) |
| `si_qr` | a QR code file |
| `si_screenshot` | a slide or page as a picture, light or dark, at any size: the agent sees what it made |

The Claude Code plugin above already includes it. Elsewhere:

```bash
claude mcp add si -- si mcp          # Claude Code, without the plugin
codex mcp add si -- si mcp           # Codex
```

Claude Desktop, Cursor and other MCP clients take the same JSON (use the
full path to `si`, from `which si`, where the app does not see your PATH):

```json
{ "mcpServers": { "si": { "command": "si", "args": ["mcp"] } } }
```

It speaks MCP 2026-07-28 and the versions before it, over stdio; nothing
listens on the network.
