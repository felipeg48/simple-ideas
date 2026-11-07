// Package docs embeds the documentation the si binary prints and renders:
// RULES.md is the single description of the markdown deck format; the user
// manual is rendered by `si manual`. Both live under docs/ so this embedder
// (which cannot reach parent directories) can reach them.
package docs

import "embed"

// Rules is RULES.md, printed by `si rules` and `si --export rules`.
//
//go:embed RULES.md
var Rules string

// AgentRules is the CLAUDE.md / AGENTS.md written by `si --export
// agent-rules`: how an AI agent should write and check si decks.
//
//go:embed agent-rules.md
var AgentRules string

// Manual is the user-manual deck, rendered by `si manual`.
//
//go:embed reference/user-manual.md
var Manual string

// Site is the hand-written part of the docs site (si --export docs-site):
// its settings, home page and examples page; the rules pages are generated
// from RULES.md so they never drift.
//
//go:embed all:site
var Site embed.FS

