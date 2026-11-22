package render

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Architecture pages (`type: architecture`, RULES.md §35): an article whose
// ## sections are numbered and whose blocks take a design-doc look. The
// first paragraph after a ## is its lead; a list whose every item reads
// "`tag` **Title** — body" is a grid of cards; a "**Name** — what" line per
// line paragraph is a definition list; the ordered list of a section titled
// Phases is a timeline whose "(now)" item is highlighted. Tables, quotes
// (notes) and ```flow (the stage fence) are styled by doc.css.

var (
	archCardRE  = regexp.MustCompile("^(?:`([^`]+)`\\s+)?\\*\\*(.+?)\\*\\*\\s*(?:—|–|--|:|-)\\s*(.*)$")
	archDefRE   = regexp.MustCompile(`^\*\*(.+?)\*\*\s*(?:—|–|--|:|-)\s*(.+)$`)
	archNowRE   = regexp.MustCompile(`(?i)\s*\(now\)\s*$`)
	archPhaseRE = regexp.MustCompile(`(?i)^phases?\b`)
)

// architecture rewrites the top-level blocks of one part of an
// architecture page (see above); problems name what does not fit.
func (dr *deckRenderer) architecture(doc ast.Node, src []byte, sc *slideCtx) {
	sec, lead := "", false
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		isLead := lead
		lead = false
		switch n := c.(type) {
		case *ast.Heading:
			if n.Level == 2 {
				sec, lead = strings.TrimSpace(textOf(n, src)), true
			}
		case *ast.Paragraph:
			if dl := dr.archDefs(n, src); dl != nil {
				doc.ReplaceChild(doc, n, dl)
			} else if isLead {
				n.SetAttributeString("class", "si-arch-lead")
			}
		case *ast.List:
			if n.IsOrdered() && archPhaseRE.MatchString(sec) {
				dr.archPhases(n, src)
			} else if cards := dr.archCards(n, src, sc); cards != nil {
				doc.ReplaceChild(doc, n, cards)
			}
		}
		c = next
	}
}

// lineTexts are a block's source lines, trimmed.
func lineTexts(n ast.Node, src []byte) []string {
	var out []string
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		if t := strings.TrimSpace(string(seg.Value(src))); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// archCards is the grid of cards a card list becomes; nil when the list is
// not one (an item without its **Title** keeps the whole list a list, and
// a list where only some items are cards is reported).
func (dr *deckRenderer) archCards(l *ast.List, src []byte, sc *slideCtx) ast.Node {
	if l.IsOrdered() {
		return nil
	}
	var cards []string
	items, matched := 0, 0
	for it := l.FirstChild(); it != nil; it = it.NextSibling() {
		items++
		if it.ChildCount() != 1 {
			continue
		}
		m := archCardRE.FindStringSubmatch(strings.Join(lineTexts(it.FirstChild(), src), " "))
		if m == nil {
			continue
		}
		matched++
		var b strings.Builder
		b.WriteString(`<article class="si-arch-card">`)
		if m[1] != "" {
			fmt.Fprintf(&b, `<span class="si-arch-tag">%s</span>`, html.EscapeString(m[1]))
		}
		fmt.Fprintf(&b, `<h3 class="si-arch-card-title">%s</h3>`, dr.inline(m[2]))
		if strings.TrimSpace(m[3]) != "" {
			fmt.Fprintf(&b, "<p>%s</p>", dr.inline(m[3]))
		}
		b.WriteString("</article>")
		cards = append(cards, b.String())
	}
	if matched == 0 {
		return nil
	}
	if matched < items {
		dr.warn(sc, "architecture: %d of %d items in a card list read `tag` **Title** — body; the list stays a list", matched, items)
		return nil
	}
	return &rawNode{html: fmt.Sprintf(`<div class="si-arch-cards" data-n="%d">%s</div>`, len(cards), strings.Join(cards, "")) + "\n"}
}

// archDefs is the definition list a paragraph of "**Name** — what" lines
// becomes; nil for any other paragraph.
func (dr *deckRenderer) archDefs(p *ast.Paragraph, src []byte) ast.Node {
	lines := lineTexts(p, src)
	if len(lines) < 2 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`<dl class="si-arch-defs">`)
	for _, l := range lines {
		m := archDefRE.FindStringSubmatch(l)
		if m == nil {
			return nil
		}
		fmt.Fprintf(&b, "<div><dt>%s</dt><dd>%s</dd></div>", dr.inline(m[1]), dr.inline(m[2]))
	}
	b.WriteString("</dl>\n")
	return &rawNode{html: b.String()}
}

// archPhases marks the Phases list: the timeline class, and the item
// ending in "(now)" (the mark itself is dropped).
func (dr *deckRenderer) archPhases(l *ast.List, src []byte) {
	l.SetAttributeString("class", "si-arch-phases")
	for it := l.FirstChild(); it != nil; it = it.NextSibling() {
		blk := it.FirstChild()
		if blk == nil {
			continue
		}
		last := blk.LastChild()
		t, ok := last.(*ast.Text)
		if !ok {
			continue
		}
		v := string(t.Segment.Value(src))
		if loc := archNowRE.FindStringIndex(v); loc != nil {
			t.Segment = t.Segment.WithStop(t.Segment.Start + loc[0])
			if p, ok := t.PreviousSibling().(*ast.Text); ok && loc[0] == 0 { // "(now)" was a text node of its own
				p.Segment = p.Segment.WithStop(p.Segment.Start + len(strings.TrimRight(string(p.Segment.Value(src)), " \t")))
			}
			it.SetAttributeString("class", "si-arch-now")
		}
	}
}
