package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Slide renders slide i of d with k build steps shown into w × h cells (the
// bar under it is not part of it). It also returns the slide's number of
// build steps.
func Slide(d *deck.Deck, i, k, w, h int, pal Palette, options ...*SlideOpts) (string, int) {
	if i < 0 || i >= len(d.Slides) || w < 10 || h < 3 {
		return "", 0
	}
	s := d.Slides[i]
	px := max(2, w/16) // side margin
	cw := w - 2*px
	r := &textRenderer{pal: pal, dir: s.Dir, w: cw, maxH: max(4, h-8), k: k}
	opts := &SlideOpts{P: 1}
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}
	if opts.States == nil {
		opts.States, opts.Prev = canvasStates(d, i, k)
	}
	r.pics, r.opts = opts.Pics, opts
	st := r.st()

	var head []string
	if s.Eyebrow != "" {
		head = append(head, st.Foreground(pal.Accent).Bold(true).Render(strings.ToUpper(s.Eyebrow)))
	}
	blocks := r.body(s.Body)
	steps := r.steps

	cover := s.Kind == deck.KindTitle || s.Kind == deck.KindSection || s.Kind == deck.KindClosing
	var top []string // a cover's logo and badge sit above its title
	if cover {
		var rest []blk
		for _, b := range blocks {
			if b.kind == "logo" || b.kind == "badge" {
				top = append(top, b.lines...)
				top = append(top, "")
			} else {
				rest = append(rest, b)
			}
		}
		blocks = rest
	}
	title := s.Title
	if title != "" {
		tr := &textRenderer{pal: pal, w: cw, k: 1 << 30}
		titleText := strings.TrimSpace(stripMarks(title))
		if s.Kind == deck.KindSection && s.Part > 0 {
			head = append(head, st.Foreground(pal.Muted).Render("PART "+itoa(s.Part)))
		}
		if lines, ok := bigLines(titleText, cw); cover && ok {
			for _, l := range lines {
				head = append(head, st.Bold(true).Render(l))
			}
		} else {
			hs := st.Bold(true)
			head = append(head, tr.wrap(tr.inline(parseInline(title, tr), hs), cw)...)
		}
		head = append(head, st.Foreground(pal.Accent).Render("━━━━━━"))
	}

	// columns: render the body again at the column width, so lists keep their
	// hanging indent and boxes their borders
	if n := columnCount(blocks, s.Layout); n > 1 && cw >= 60 {
		colW := (cw - 4*(n-1)) / n
		cr := &textRenderer{pal: pal, dir: s.Dir, w: colW, maxH: max(4, h-8), k: k, pics: r.pics, opts: opts}
		blocks = cr.body(s.Body)
		if cover {
			var rest []blk
			for _, b := range blocks {
				if b.kind != "logo" && b.kind != "badge" {
					rest = append(rest, b)
				}
			}
			blocks = rest
		}
	}
	body := compose(r, blocks, s.Layout, cw)
	if s.Layout == "big-number" {
		body = bigNumber(r, blocks, cw)
	}
	for _, l := range s.Labs {
		body = append(body, st.Foreground(pal.Accent).Render("▶ Lab: ")+st.Bold(true).Render(l.Text)+st.Foreground(pal.Muted).Render("  "+l.URL))
	}
	var lines []string
	lines = append(lines, top...)
	lines = append(lines, head...)
	if len(head) > 0 && len(body) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, body...)
	if i == 0 && s.Kind == deck.KindTitle {
		if by := strings.Trim(d.Meta.Author+" · "+d.Meta.Date, " ·"); by != "" {
			lines = append(lines, "", st.Foreground(pal.Muted).Render(by))
		}
	}
	center := s.Layout == "center" || s.Layout == "quote" || s.Layout == "big-number"
	if len(lines) > h {
		lines = append(lines[:h-1], st.Foreground(pal.Muted).Render("▼ more below: make the terminal taller"))
	}
	padTop := max(0, (h-len(lines))/2)
	var out []string
	for range padTop {
		out = append(out, "")
	}
	indent := 0 // a centred layout centres the block, so a list keeps its left edge
	if center {
		wide := 0
		for _, l := range lines {
			wide = max(wide, ansi.StringWidth(l))
		}
		indent = max(0, (cw-wide)/2)
	}
	for _, l := range lines {
		if ansi.StringWidth(l) > cw-indent { // a safety net: nothing runs past the right margin
			l = ansi.Truncate(l, cw-indent, "…")
		}
		out = append(out, strings.Repeat(" ", px+indent)+l)
	}
	for len(out) < h {
		out = append(out, "")
	}
	return strings.Join(out, "\n"), steps
}

// columnCount is how many columns the blocks fill: <!-- column --> breaks, or
// text and media side by side in a two-col / split layout.
func columnCount(blocks []blk, layout string) int {
	cols := 1
	for _, b := range blocks {
		cols = max(cols, b.col+1)
	}
	if cols > 1 || (layout != "two-col" && layout != "split") {
		return cols
	}
	media, text := false, false
	for _, b := range blocks {
		media = media || b.media
		text = text || !b.media
	}
	if media && text {
		return 2
	}
	return 1
}

// compose lays the blocks out: one column, or two (explicit <!-- column -->
// breaks, or text left and media right in a two-col / split layout).
func compose(r *textRenderer, blocks []blk, layout string, w int) []string {
	cols := 1
	for _, b := range blocks {
		cols = max(cols, b.col+1)
	}
	auto := cols == 1 && (layout == "two-col" || layout == "split")
	if auto {
		hasMedia, hasText := false, false
		for _, b := range blocks {
			hasMedia = hasMedia || b.media
			hasText = hasText || !b.media
		}
		if !(hasMedia && hasText) {
			auto = false
		}
	}
	if cols == 1 && !auto || w < 60 {
		var out []string
		for i, b := range blocks {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, b.lines...)
		}
		return out
	}
	if auto {
		cols = 2
	}
	gap := 4
	cw := (w - gap*(cols-1)) / cols
	sub := *r
	sub.w = cw
	sub.steps = 0
	// re-render each block at the column width: wrap and boxes depend on it
	var colLines = make([][]string, cols)
	for _, b := range blocks {
		c := b.col
		if auto {
			c = 0
			if b.media {
				c = 1
			}
		}
		if len(colLines[c]) > 0 {
			colLines[c] = append(colLines[c], "")
		}
		colLines[c] = append(colLines[c], refit(b.lines, cw)...)
	}
	// join by hand: images are raw colour codes that a styled re-render would mangle
	rows := 0
	for _, c := range colLines {
		rows = max(rows, len(c))
	}
	out := make([]string, rows)
	for y := range out {
		var b strings.Builder
		for i, c := range colLines {
			l := ""
			if y < len(c) {
				l = c[y]
			}
			b.WriteString(l)
			if i < cols-1 {
				b.WriteString(strings.Repeat(" ", max(0, cw-ansi.StringWidth(l))+gap))
			}
		}
		out[y] = b.String()
	}
	return out
}

// refit re-wraps lines rendered for a wider column (boxes are cut, text wraps).
func refit(lines []string, w int) []string {
	var out []string
	for _, l := range lines {
		if ansi.StringWidth(l) <= w {
			out = append(out, l)
			continue
		}
		if strings.ContainsAny(l, "│╭╰┌└├") {
			out = append(out, ansi.Truncate(l, w-1, "…"))
			continue
		}
		out = append(out, strings.Split(ansi.Wrap(l, w, ""), "\n")...)
	}
	return out
}

// bigNumber draws a big-number slide: its first short line in block letters.
func bigNumber(r *textRenderer, blocks []blk, w int) []string {
	var out []string
	done := false
	for i, b := range blocks {
		if i > 0 {
			out = append(out, "")
		}
		if !done && len(b.lines) == 1 {
			plain := strings.TrimSpace(ansi.Strip(b.lines[0]))
			if lines, ok := bigLines(plain, w); ok && len(plain) <= 12 {
				for _, l := range lines {
					out = append(out, r.st().Foreground(r.pal.Accent).Bold(true).Render(l))
				}
				done = true
				continue
			}
		}
		out = append(out, b.lines...)
	}
	return out
}

// Bar is the line under a slide: progress, slide and build, the footer, the timer.
func Bar(d *deck.Deck, i, k, steps, w int, timer string, pal Palette) string {
	st := lipgloss.NewStyle().Foreground(pal.FG)
	n := len(d.Slides)
	right := st.Foreground(pal.Muted).Render("  " + timer + "   ? keys ")
	count := " " + itoa(i+1) + "/" + itoa(n)
	if steps > 0 {
		count += st.Foreground(pal.Muted).Render(" · build " + itoa(k) + "/" + itoa(steps))
	}
	footer := d.Meta.Footer
	if f := d.Slides[i].Footer; f == "off" {
		footer = ""
	} else if f != "" {
		footer = f
	}
	if footer != "" {
		footer = st.Foreground(pal.Muted).Render("   " + footer)
	}
	room := w - ansi.StringWidth(count) - ansi.StringWidth(right) - ansi.StringWidth(footer) - 2
	if room < 10 {
		footer = ""
		room = w - ansi.StringWidth(count) - ansi.StringWidth(right) - 2
	}
	room = max(4, room)
	done := room * (i + 1) / max(1, n)
	bar := st.Foreground(pal.Accent).Render(strings.Repeat("━", done)) + st.Foreground(pal.Line).Render(strings.Repeat("─", room-done))
	return " " + bar + count + footer + right
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// stripMarks drops ==highlight== marks and inline markdown from a title.
func stripMarks(s string) string {
	return strings.NewReplacer("==", "", "**", "", "__", "", "`", "").Replace(stripTags(s))
}

// parseInline parses one line of inline markdown (a title) for tr.inline.
func parseInline(s string, tr *textRenderer) ast.Node {
	tr.src = []byte(s)
	if p := md.Parser().Parse(text.NewReader(tr.src)).FirstChild(); p != nil {
		return p
	}
	return ast.NewParagraph()
}
