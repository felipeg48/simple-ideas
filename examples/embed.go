// Package examples embeds the showcase decks so `si new` and
// `si --export example` can write them out.
package examples

import "embed"

// FS holds the example decks, their assets and the example theme, plus the
// example article (article/), guide (guide/, embedded with its _ files) and
// mind maps (mindmap/).
//
//go:embed *.md *.json si.conf assets css night-shift demos article code all:guide all:book all:site all:course mindmap infographic
var FS embed.FS

// Main is the showcase deck that includes the others.
const Main = "showcase.md"
