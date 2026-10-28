// Package examples embeds the showcase decks so `si new` and
// `si --export example` can write them out.
package examples

import "embed"

// FS holds the example decks, their assets and the example theme.
//
//go:embed *.md *.json si.conf assets css night-shift
var FS embed.FS

// Main is the showcase deck that includes the others.
const Main = "showcase.md"
