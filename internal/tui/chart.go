package tui

import (
	"math"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// chart draws a ```chart fence as text: one bar per row, to scale, for the
// last number column (where a show with steps ends; the others are listed
// under it), whatever the type (a donut or a 3D chart reads as bars too).
func (r *textRenderer) chart(src string, attrs map[string]string) []string {
	c, _ := deck.ParseChart(attrs, src)
	if len(c.Head) == 0 {
		return r.placeholder("▤ chart (needs a table: labels, then numbers)", nil)
	}
	col := len(c.Head) - 1
	head := c.Type + " chart · " + stripMarks(c.Head[col])
	if c.Title != "" {
		head = stripMarks(c.Title) + " · " + head
	}
	out := []string{r.st().Bold(true).Render(head)}
	top, lw, vw := 0.0, 0, 0
	for i, l := range c.Labels {
		top = math.Max(top, c.Vals[i][col])
		lw = max(lw, ansi.StringWidth(stripMarks(l)))
		vw = max(vw, ansi.StringWidth(c.ChartValue(i, col)))
	}
	lw = min(lw, max(6, r.w/3))
	room := max(4, r.w-lw-vw-3)
	for i, l := range c.Labels {
		n := 0
		if top > 0 && c.Vals[i][col] > 0 {
			n = max(1, int(math.Round(c.Vals[i][col]/top*float64(room))))
		}
		colour := r.pal.Accent
		if i%2 == 1 {
			colour = r.pal.Accent2
		}
		label := ansi.Truncate(stripMarks(l), lw, "…")
		out = append(out, label+strings.Repeat(" ", lw-ansi.StringWidth(label))+" "+
			r.st().Foreground(colour).Render(strings.Repeat("█", n))+" "+r.st().Bold(true).Render(c.ChartValue(i, col)))
	}
	if col > 0 {
		var steps []string
		for _, h := range c.Head {
			steps = append(steps, stripMarks(h))
		}
		out = append(out, r.st().Foreground(r.pal.Muted).Render(strings.Join(steps, " › ")))
	}
	return out
}
