// Package simpleideas embeds the documentation the si binary prints:
// RULES.md is the single description of the markdown deck format.
package simpleideas

import _ "embed"

// Rules is RULES.md, printed by `si rules` and `si --export rules`.
//
//go:embed RULES.md
var Rules string
