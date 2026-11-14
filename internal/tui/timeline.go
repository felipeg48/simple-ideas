package tui

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Timelines and infographics in the terminal: a timeline as a list of eras
// and their milestones (date, title, description, the picture's caption); an
// infographic as its blocks in order, a stat's number in block letters when it
// fits, every other block's Markdown drawn like a slide body.

func (r *textRenderer) timeline(src string) []string {
	t, _, _ := deck.ParseTimeline(src, "", 1)
	var out []string
	if t.Title != "" {
		out = append(out, r.st().Bold(true).Render("◷ "+stripMarks(t.Title)))
	}
	cols := []color.Color{r.pal.Accent, r.pal.Accent2, r.pal.OK, r.pal.Warn, r.pal.Crit}
	dw := 0
	for _, m := range t.Milestones() {
		dw = max(dw, ansi.StringWidth(m.Date))
	}
	dw = min(dw, max(6, r.w/4))
	for i, e := range t.Eras {
		c := cols[i%len(cols)]
		if len(out) > 0 {
			out = append(out, "")
		}
		if e.Name != "" {
			span := ""
			if n := len(e.Items); n > 0 {
				span = e.Items[0].Date
				if last := e.Items[n-1].Date; last != span {
					span += " – " + last
				}
			}
			out = append(out, ansi.Truncate(r.st().Bold(true).Foreground(c).Render("━━ "+strings.ToUpper(stripMarks(e.Name)))+"  "+r.st().Foreground(r.pal.Muted).Render(span), r.w, "…"))
		}
		pad := strings.Repeat(" ", dw+2)
		for _, m := range e.Items {
			date := ansi.Truncate(m.Date, dw, "…")
			head := r.st().Foreground(c).Bold(true).Render(date+strings.Repeat(" ", dw-ansi.StringWidth(date))) + "  " + r.st().Bold(true).Render(stripMarks(m.Title))
			out = append(out, ansi.Truncate(head, r.w, "…"))
			if m.Text != "" {
				for _, l := range r.wrap(r.st().Render(stripMarks(m.Text)), r.w-dw-2) {
					out = append(out, pad+l)
				}
			}
			if m.Img != "" && m.Caption != "" {
				out = append(out, ansi.Truncate(pad+r.st().Foreground(r.pal.Muted).Italic(true).Render("▨ "+m.Caption), r.w, "…"))
			}
		}
	}
	if len(t.Sources) > 0 {
		title := t.SourcesTitle
		if title == "" {
			title = "Sources"
		}
		out = append(out, "", r.st().Foreground(r.pal.Muted).Bold(true).Render(strings.ToUpper(title)))
		for _, s := range t.Sources {
			for _, l := range r.wrap(r.st().Foreground(r.pal.Muted).Render("· "+stripMarks(s)), r.w) {
				out = append(out, l)
			}
		}
	}
	return out
}

func (r *textRenderer) infographic(src string) []string {
	ig, _ := deck.ParseInfographic(src)
	var out []string
	if ig.Title != "" {
		out = append(out, r.st().Bold(true).Render(stripMarks(ig.Title)))
	}
	if ig.Subtitle != "" {
		out = append(out, r.wrap(r.st().Foreground(r.pal.Muted).Render(stripMarks(ig.Subtitle)), r.w)...)
	}
	for i, b := range ig.Blocks {
		c := r.pal.Accent
		if i%2 == 1 {
			c = r.pal.Accent2
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		if b.Kind == "stat" {
			num := stripMarks(b.Title)
			if lines, ok := bigLines(num, r.w); ok && len(num) <= 12 {
				for _, l := range lines {
					out = append(out, r.st().Foreground(c).Bold(true).Render(l))
				}
			} else {
				out = append(out, r.st().Foreground(c).Bold(true).Render(num))
			}
		} else {
			kind := b.Kind
			if b.Chart != "" {
				kind = b.Chart + " chart"
			}
			out = append(out, ansi.Truncate(r.st().Foreground(c).Render("▍")+r.st().Bold(true).Render(strings.ToUpper(stripMarks(b.Title)))+r.st().Foreground(r.pal.Muted).Render(" · "+kind), r.w, "…"))
		}
		sub := &textRenderer{pal: r.pal, dir: r.dir, w: r.w, maxH: r.maxH, k: 1 << 30, pics: r.pics}
		for j, bl := range sub.body(b.Body) {
			if j > 0 {
				out = append(out, "")
			}
			out = append(out, bl.lines...)
		}
	}
	if s := ig.Sources; s != nil && s.Body != "" {
		title := s.Title
		if title == "" {
			title = "Sources"
		}
		out = append(out, "", r.st().Foreground(r.pal.Muted).Bold(true).Render(strings.ToUpper(stripMarks(title))))
		for _, l := range strings.Split(s.Body, "\n") {
			if l = strings.TrimSpace(strings.TrimLeft(l, "-*+ ")); l != "" {
				out = append(out, r.wrap(r.st().Foreground(r.pal.Muted).Render("· "+stripMarks(l)), r.w)...)
			}
		}
	}
	return out
}
