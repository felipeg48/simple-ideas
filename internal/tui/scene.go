package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// scene draws a ```scene fence as text: its blocks as [boxes] where the
// scene ends, then its steps, numbered.
func (r *textRenderer) scene(src string, attrs map[string]string) []string {
	sc, _ := deck.ParseScene(attrs, src)
	head := "scene"
	if sc.Title != "" {
		head = stripMarks(sc.Title) + " · scene"
	}
	out := []string{r.st().Bold(true).Render(head)}
	end := sc.States[len(sc.States)-1]
	var row []string
	for i, o := range sc.Objs {
		if (o.Kind != "box" && o.Kind != "queue") || !end[i].V {
			continue
		}
		b := "[" + end[i].L + "]"
		if o.Kind == "queue" {
			b = "(" + end[i].L + " ▭▭▭)"
		}
		c := r.pal.Muted
		if end[i].On {
			c = r.pal.Accent
		}
		row = append(row, r.st().Foreground(c).Render(b))
	}
	if len(row) > 0 {
		out = append(out, ansi.Truncate(strings.Join(row, " "), max(8, r.w), "…"))
	}
	for k, s := range sc.Steps {
		out = append(out, r.st().Foreground(r.pal.Accent2).Render(fmt.Sprintf("%d.", k+1))+" "+ansi.Truncate(s.Text, max(8, r.w-4), "…"))
	}
	return out
}
