package render

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// sandboxLinks points [Console](sandbox:15672) links (a port of the lab's
// Docker box, RULES §37, and an optional path: sandbox:15672/api) at si
// play, which sends them on to where the box publishes that port.
func sandboxLinks(doc ast.Node) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if l, ok := n.(*ast.Link); ok && entering {
			if rest, ok := strings.CutPrefix(string(l.Destination), "sandbox:"); ok {
				port, path, _ := strings.Cut(rest, "/")
				l.Destination = []byte("/_si/sandbox/port/" + port + "/" + path)
			}
		}
		return ast.WalkContinue, nil
	})
}
