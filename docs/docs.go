// Package docs embeds the documentation the si binary prints and renders:
// RULES.md is the single description of the markdown deck format; the user
// manual is rendered by `si manual`. Both live under docs/ so this embedder
// (which cannot reach parent directories) can reach them.
package docs

import _ "embed"

// Rules is RULES.md, printed by `si rules` and `si --export rules`.
//
//go:embed RULES.md
var Rules string

// Manual is the user-manual deck, rendered by `si manual`.
//
//go:embed reference/user-manual.md
var Manual string
