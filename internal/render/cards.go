package render

import (
	"fmt"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
)

// <!-- cards: 2 --> turns the ### headings that follow into a grid of cards,
// on a slide or in a document:
//
//	<!-- cards: 2 -->
//
//	### High availability · In seconds · teal
//	One cluster, one site, three hosts.
//
// A heading part that is a colour (teal, orange, ok, crit, hi = the
// highlighted card) styles the card; any other part is its tag. The value is
// the number of columns (2, 3 or 4); without one the cards fill the width.
// The grid ends at the next ## heading, --- or directive.

var cardColours = map[string]bool{"teal": true, "orange": true, "ok": true, "crit": true, "hi": true}

// cardsPass replaces every cards directive and the cards it owns with one
// grid node, so a <!-- step --> before it reveals the whole grid.
func (dr *deckRenderer) cardsPass(doc ast.Node, src []byte, sc *slideCtx) {
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		hb, ok := c.(*ast.HTMLBlock)
		if !ok {
			c = next
			continue
		}
		name, val, isDir := deck.ParseDirective(blockText(hb, src))
		if !isDir || name != "cards" {
			c = next
			continue
		}
		class := "si-cards cards"
		switch v := strings.TrimSpace(val); v {
		case "":
		case "2", "3", "4":
			class += " c" + v
		default:
			dr.problem(sc, "cards: %q is not a column count (2, 3 or 4)", v)
		}
		grid := &divNode{class: class}
		var card *divNode
		k := next
		for k != nil {
			kn := k.NextSibling()
			if h, ok := k.(*ast.Heading); ok && h.Level <= 2 {
				break
			}
			if _, ok := k.(*ast.ThematicBreak); ok {
				break
			}
			if kh, ok := k.(*ast.HTMLBlock); ok {
				if _, _, ok := deck.ParseDirective(blockText(kh, src)); ok {
					break
				}
			}
			if h, ok := k.(*ast.Heading); ok && h.Level == 3 {
				card = dr.cardOf(strings.TrimSpace(blockText(h, src))) // the source, so a title can be a link
				grid.AppendChild(grid, card)
				doc.RemoveChild(doc, k)
			} else if card != nil {
				doc.RemoveChild(doc, k)
				card.AppendChild(card, k)
			} else {
				break
			}
			k = kn
		}
		if grid.ChildCount() == 0 {
			dr.problem(sc, "cards: must be followed by ### headings, one per card")
			doc.RemoveChild(doc, c)
		} else {
			doc.ReplaceChild(doc, c, grid)
		}
		c = k
	}
}

// cardOf makes one card from its heading: "Title · tag · colour".
func (dr *deckRenderer) cardOf(heading string) *divNode {
	parts := strings.Split(heading, "·")
	title, tag, colour := strings.TrimSpace(parts[0]), "", ""
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if cardColours[strings.ToLower(p)] && colour == "" {
			colour = strings.ToLower(p)
		} else if p != "" {
			tag = strings.TrimSpace(strings.TrimPrefix(tag+" · "+p, " · "))
		}
	}
	class := "si-cell card"
	if colour != "" {
		class += " " + colour
	}
	open := ""
	if tag != "" {
		tc := "si-card-tag tag"
		if colour != "" && colour != "hi" {
			tc += " " + colour
		}
		open += fmt.Sprintf(`<span class="%s">%s</span>`, tc, dr.inline(tag))
	}
	open += fmt.Sprintf("<h4>%s</h4>\n", dr.inline(title))
	return &divNode{class: class, open: open}
}
