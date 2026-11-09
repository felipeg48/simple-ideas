// Package skill makes si's Agent Skill: SKILL.md (when to use si, the work
// loop, and the agent guide docs/agent-rules.md) and references/, one file
// per RULES.md section, so an agent loads only the part it needs. The same
// files are `si --export skill` and the Claude Code plugin committed under
// plugin/ (TestPluginInSync keeps that copy from drifting).
package skill

import (
	"fmt"
	"strings"

	"github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/internal/docsite"
)

// Description decides when an agent loads the skill, so it names the things a
// user says when they want one.
const Description = "Write, check and build slide decks, talks and presentations in Markdown with si " +
	"(simple-ideas), and its other documents: articles, tutorials and guides, books, labs, sites, " +
	"mind maps (2D and 3D), code windows and code shows, QR codes. Use it when the user asks for a " +
	"deck, slides, a talk, a presentation, speaker notes or a teleprompter script from notes or an " +
	"outline; mentions si, simple-ideas or `si play` / `si build` / `si check`; works in a folder with " +
	"si.conf, _nav.md or _site.md; or wants an existing deck (PowerPoint, reveal.js, AsciiDoc) moved to Markdown."

// Files returns the skill as relative path → content: SKILL.md and
// references/NN-slug.md.
func Files() map[string]string {
	intro, secs := docsite.Split(docs.Rules)
	files := map[string]string{}
	var refs strings.Builder
	for _, s := range secs {
		name := "references/" + s.Slug + ".md"
		files[name] = fmt.Sprintf("# %s. %s\n\n%s\n", s.N, s.Title, strings.TrimSpace(s.Body))
		fmt.Fprintf(&refs, "- [%s. %s](%s)\n", s.N, s.Title, name)
	}
	files["references/00-overview.md"] = strings.TrimSpace(intro) + "\n"
	guide := strings.TrimSpace(docs.AgentRules)
	if i := strings.Index(guide, "\n"); strings.HasPrefix(guide, "# ") && i > 0 {
		guide = strings.TrimSpace(guide[i:]) // the skill has its own title
	}
	files["SKILL.md"] = fmt.Sprintf(`---
name: si
description: %q
---

# si: talks and documents from Markdown

si turns Markdown (or AsciiDoc) into fluid, animated, self-contained HTML:
decks with a teleprompter, articles, guides, books, mind maps, code shows.

## Before you start

- `+"`si --version`"+` must work. If it does not, tell the user to install si:
  https://simple-ideas.org/download.html (or `+"`go install github.com/felipeg48/simple-ideas/cmd/si@latest`"+`).
- Read the guide below once. For a directive or option, open the matching
  file in references/ (listed at the end) or run `+"`si rules`"+`; never guess
  syntax.

## The loop, every time

1. Write or edit the Markdown.
2. `+"`si check <file or folder>`"+`, and fix every problem it reports.
3. Look at it: `+"`si build <file> -o <tmp dir>`"+` and a headless screenshot
   (or ask the user to run `+"`si play <file> --open`"+`). A deck that passes
   `+"`si check`"+` can still look wrong.
4. Only then say it is done, with the command that shows it.

%s

## References (one per section of the full rules)

- [Overview](references/00-overview.md)
%s`, Description, guide, refs.String())
	return files
}

// Plugin is the Claude Code plugin under plugin/: its manifest (stamped with
// si's version) and the skill.
func Plugin(version string) map[string]string {
	files := map[string]string{
		".claude-plugin/plugin.json": `{
  "name": "si",
  "version": "` + version + `",
  "description": "Write slide decks, talks, articles, guides, books, mind maps and code shows in Markdown with si (simple-ideas): the format, the check-and-look loop, and every rule on demand.",
  "author": { "name": "Felipe Gutierrez", "url": "https://github.com/felipeg48" },
  "homepage": "https://simple-ideas.org/agents.html",
  "repository": "https://github.com/felipeg48/simple-ideas",
  "license": "Apache-2.0",
  "keywords": ["slides", "presentation", "markdown", "teleprompter", "mind-map", "docs"]
}
`,
		"README.md": "# si for Claude Code\n\nThe `si` skill: how to write decks and documents for si, the check-and-look\nloop, and every rule on demand. Generated from docs/agent-rules.md and\ndocs/RULES.md by `go test ./internal/skill -update`; do not edit by hand.\n\n```\n/plugin marketplace add felipeg48/simple-ideas\n/plugin install si@simple-ideas\n```\n\nThe si binary itself: https://simple-ideas.org/download.html\n",
	}
	for p, c := range Files() {
		files["skills/si/"+p] = c
	}
	return files
}
