package tui

import (
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"html"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// blk is one block of a slide body: its lines, its column (<!-- column -->),
// whether it is media (an image, code, a diagram: the right-hand column of an
// automatic two-col layout) and its kind for the cover (badge, logo).
type blk struct {
	lines []string
	col   int
	media bool
	kind  string
}

// textRenderer turns markdown into terminal lines for one width.
type textRenderer struct {
	coN   int // callouts drawn since the last code block
	pal   Palette
	dir   string // base for images
	w     int
	maxH  int       // tallest an image may be (rows)
	k     int       // build steps shown: a block of step n shows when n <= k
	steps int       // build steps counted so far
	pics  *Pictures // real pictures (Kitty / iTerm2), nil for half blocks
	// canvases: their state on this slide (by the path the directive names),
	// the state they come from (a tween while p < 1), and the time
	opts *SlideOpts
	// collect: only record the slide's canvases and build-step ops (canvasStates)
	collect    bool
	svgs       []svgRef
	stepOps    []string
	tokens     []string // the state tokens of the steps list being rendered
	pendingOps string   // the state ops of the next <!-- step: ops -->
	src        []byte
}

var md = render.NewParser()

func (r *textRenderer) st() lipgloss.Style { return lipgloss.NewStyle().Foreground(r.pal.FG) }

// body renders a slide body (or any markdown) into blocks.
func (r *textRenderer) body(markdown string) []blk {
	r.src = []byte(markdown)
	doc := md.Parser().Parse(text.NewReader(r.src))
	var nodes []ast.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		nodes = append(nodes, c)
	}
	var out []blk
	col, step, stepsList := 0, false, false
	add := func(b blk) {
		b.col = col
		out = append(out, b)
	}
	// visible counts one build step (and its state ops) and says whether it is shown
	visible := func() bool {
		r.steps++
		r.stepOps = append(r.stepOps, r.pendingOps)
		r.pendingOps = ""
		return r.steps <= r.k
	}
	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		if hb, ok := n.(*ast.HTMLBlock); ok {
			raw := blockText(hb, r.src)
			if notes := r.callouts(raw); notes != nil { // a code window's callouts: numbered notes under it
				add(blk{lines: notes})
				continue
			}
			name, val, isDir := deck.ParseDirective(raw)
			if !isDir {
				if t := strings.Join(strings.Fields(stripTags(raw)), " "); t != "" {
					add(blk{lines: r.wrap(r.st().Foreground(r.pal.Muted).Render(t), r.w)})
				}
				continue
			}
			switch name {
			case "step":
				step = true
				r.pendingOps = val
			case "steps":
				stepsList = true
				r.tokens = strings.Fields(val)
			case "column":
				col++
			case "badge":
				add(blk{lines: []string{r.badge(val)}, kind: "badge"})
			case "logo":
				add(blk{lines: []string{r.logo(val)}, kind: "logo"})
			case "cards":
				end := i + 1
				for end < len(nodes) && !endsGroup(nodes[end], r.src) {
					end++
				}
				lines := r.cards(val, nodes[i+1:end])
				i = end - 1
				if step {
					step = false
					if !visible() {
						continue
					}
				}
				add(blk{lines: lines})
			case "svg":
				f := strings.Fields(val)
				if len(f) > 0 {
					r.svgs = append(r.svgs, svgRef{f[0], f[1:]})
				}
				add(blk{lines: r.canvasLines(f), media: true})
			}
			continue
		}
		if l, ok := n.(*ast.List); ok && stepsList {
			stepsList = false
			add(blk{lines: r.list(l, 0, true)})
			r.tokens = nil
			continue
		}
		if step || stepsList {
			if stepsList && len(r.tokens) > 0 { // steps: ops on a block that is not a list: one step
				r.pendingOps = r.tokens[0]
			}
			step, stepsList = false, false
			r.tokens = nil
			if !visible() {
				continue
			}
		}
		lines, media := r.block(n)
		if len(lines) > 0 {
			add(blk{lines: lines, media: media})
		}
	}
	return out
}

// endsGroup reports whether a node ends a cards group: a ## heading, a ---,
// or another directive.
func endsGroup(n ast.Node, src []byte) bool {
	switch x := n.(type) {
	case *ast.Heading:
		return x.Level <= 2
	case *ast.ThematicBreak:
		return true
	case *ast.HTMLBlock:
		_, _, ok := deck.ParseDirective(blockText(x, src))
		return ok
	}
	return false
}

// block renders one top-level node; media marks images, code and diagrams.
func (r *textRenderer) block(n ast.Node) ([]string, bool) {
	switch x := n.(type) {
	case *ast.Heading:
		return r.wrap(r.inline(x, r.st().Bold(true).Foreground(r.pal.Accent2)), r.w), false
	case *ast.Paragraph:
		if img := soleImage(x); img != nil {
			return r.image(string(img.Destination), render.TextOf(img, r.src)), true
		}
		return r.wrap(r.inline(x, r.st()), r.w), false
	case *ast.TextBlock:
		return r.wrap(r.inline(x, r.st()), r.w), false
	case *ast.List:
		return r.list(x, 0, false), false
	case *ast.FencedCodeBlock:
		return r.fence(x), true
	case *ast.CodeBlock:
		return r.code(blockText(x, r.src), "", nil), true
	case *ast.Blockquote:
		return r.quote(x), false
	case *ast.ThematicBreak:
		return []string{r.st().Foreground(r.pal.Line).Render(strings.Repeat("─", min(r.w, 40)))}, false
	case *east.Table:
		return r.table(x), false
	}
	return nil, false
}

// inline renders the inline content of n with the style st carried down, so a
// bold word inside a highlight keeps the band.
func (r *textRenderer) inline(n ast.Node, st lipgloss.Style) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch x := c.(type) {
		case *ast.Text:
			b.WriteString(st.Render(string(x.Segment.Value(r.src))))
			if x.HardLineBreak() {
				b.WriteString("\n")
			} else if x.SoftLineBreak() {
				b.WriteString(st.Render(" "))
			}
		case *ast.String:
			b.WriteString(st.Render(string(x.Value)))
		case *ast.Emphasis:
			if x.Level >= 2 {
				b.WriteString(r.inline(x, st.Bold(true)))
			} else {
				b.WriteString(r.inline(x, st.Italic(true)))
			}
		case *ast.CodeSpan:
			b.WriteString(st.Background(r.pal.Surface).Foreground(r.pal.Accent2).Render(" " + render.TextOf(x, r.src) + " "))
		case *ast.Link:
			b.WriteString(r.inline(x, st.Foreground(r.pal.Accent).Underline(true).Hyperlink(string(x.Destination))))
		case *ast.AutoLink:
			u := string(x.URL(r.src))
			b.WriteString(st.Foreground(r.pal.Accent).Underline(true).Hyperlink(u).Render(u))
		case *ast.Image:
			b.WriteString(st.Foreground(r.pal.Muted).Render("▣ " + render.TextOf(x, r.src)))
		case *ast.RawHTML:
			// inline tags are dropped; a <br> becomes a line break
			for i := 0; i < x.Segments.Len(); i++ {
				seg := x.Segments.At(i)
				if strings.HasPrefix(strings.ToLower(string(seg.Value(r.src))), "<br") {
					b.WriteString("\n")
				}
			}
		case *east.Strikethrough:
			b.WriteString(r.inline(x, st.Strikethrough(true)))
		case *east.TaskCheckBox:
			box := "☐ "
			if x.IsChecked {
				box = "☑ "
			}
			b.WriteString(st.Foreground(r.pal.Accent).Render(box))
		default:
			if c.Kind() == render.KindHighlight {
				b.WriteString(r.inline(c, st.Background(r.pal.HL)))
			} else {
				b.WriteString(r.inline(c, st))
			}
		}
	}
	return b.String()
}

// wrap breaks styled text to w columns.
func (r *textRenderer) wrap(s string, w int) []string {
	if w < 4 {
		w = 4
	}
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

// list renders a list; with each true, every item is its own build step.
func (r *textRenderer) list(l *ast.List, depth int, each bool) []string {
	var out []string
	num := l.Start
	if num == 0 {
		num = 1
	}
	item := 0
	for it := l.FirstChild(); it != nil; it = it.NextSibling() {
		if each {
			r.steps++
			ops := "" // this item's state token from <!-- steps: a b . c -->
			if item < len(r.tokens) && r.tokens[item] != "." {
				ops = r.tokens[item]
			}
			item++
			r.stepOps = append(r.stepOps, ops)
			if r.steps > r.k {
				num++
				continue
			}
		}
		mark := "•"
		if depth > 0 {
			mark = "◦"
		}
		if l.IsOrdered() {
			mark = strconv.Itoa(num) + "."
		}
		num++
		indent := strings.Repeat("  ", depth)
		markW := ansi.StringWidth(mark) + 1
		w := r.w - len(indent) - markW
		first := true
		for c := it.FirstChild(); c != nil; c = c.NextSibling() {
			var lines []string
			if sub, ok := c.(*ast.List); ok {
				out = append(out, r.list(sub, depth+1, false)...)
				continue
			}
			lines = r.wrap(r.inline(c, r.st()), w)
			for _, ln := range lines {
				if first {
					out = append(out, indent+r.st().Foreground(r.pal.Accent).Render(mark)+" "+ln)
					first = false
				} else {
					out = append(out, indent+strings.Repeat(" ", markW)+ln)
				}
			}
		}
	}
	return out
}

func (r *textRenderer) quote(q *ast.Blockquote) []string {
	bar := r.st().Foreground(r.pal.Accent).Render("▌ ")
	var out []string
	for c := q.FirstChild(); c != nil; c = c.NextSibling() {
		for _, ln := range r.wrap(r.inline(c, r.st().Italic(true)), r.w-2) {
			out = append(out, bar+ln)
		}
	}
	return out
}

// ---------- code ----------

var (
	stageDotRE = regexp.MustCompile(`(?m)^\s*dot\s+([^.\s]+)\.\.([^\s]+)(?:\s+([\d.]+)s)?`)
	tokRE      = regexp.MustCompile(`(?s)<span class="si-tk-(\w)">(.*?)</span>`)
	calloutRE  = regexp.MustCompile(`\s*(?://|#|--|;)?\s*<(\d+)>\s*$`)
	attrRE     = regexp.MustCompile(`(\w+)(?:=("[^"]*"|\S+))?`)
	circled    = []rune("❶❷❸❹❺❻❼❽❾❿")
)

func fenceAttrs(info string) (string, map[string]string) {
	lang, attrs := codeshow.Info(info)
	return strings.ToLower(lang), attrs
}

func (r *textRenderer) fence(f *ast.FencedCodeBlock) []string {
	info := ""
	if f.Info != nil {
		info = string(f.Info.Segment.Value(r.src))
	}
	lang, attrs := fenceAttrs(info)
	code := blockText(f, r.src)
	r.coN = 0
	switch lang {
	case "stage":
		return r.stage(code)
	case "mindmap":
		return r.mindmap(code)
	case "mermaid":
		lines := strings.Split(strings.TrimRight(code, "\n"), "\n")
		if len(lines) > 8 {
			lines = append(lines[:8], "…")
		}
		return r.placeholder("◆ Mermaid diagram (drawn in the browser)", lines)
	}
	return r.code(code, lang, attrs)
}

// code draws a code block in a box: highlighted, with a title, line numbers
// (linenos), highlighted lines (hl="2,4-5") and callouts (<1> → ❶).
func (r *textRenderer) code(code, lang string, attrs map[string]string) []string {
	code = strings.TrimRight(code, "\n")
	lines := strings.Split(code, "\n")
	for i, l := range lines {
		if m := calloutRE.FindStringSubmatchIndex(l); m != nil {
			n, _ := strconv.Atoi(l[m[2]:m[3]])
			mark := "●"
			if n >= 1 && n <= len(circled) {
				mark = string(circled[n-1])
			}
			lines[i] = l[:m[0]] + " \x00" + mark // \x00: placeholder kept through highlighting
		}
	}
	hl := lineSet(attrs["hl"])
	start := 1
	if s, err := strconv.Atoi(attrs["start"]); err == nil {
		start = s
	}
	_, nums := attrs["linenos"]
	if _, ok := attrs["numbers"]; ok {
		nums = true
	}
	numW := len(strconv.Itoa(start + len(lines)))
	inner := r.w - 4
	var body []string
	for i, l := range lines {
		h := r.hlTokens(render.Highlight(strings.ReplaceAll(l, "\x00", ""), lang))
		h = strings.Replace(h, "●", r.st().Foreground(r.pal.Accent).Render("●"), 1)
		for _, c := range circled {
			h = strings.Replace(h, string(c), r.st().Foreground(r.pal.Accent).Render(string(c)), 1)
		}
		if nums {
			h = r.st().Foreground(r.pal.Muted).Render(padLeft(strconv.Itoa(start+i), numW)+" ") + h
		}
		if ansi.StringWidth(h) > inner {
			h = ansi.Truncate(h, inner-1, "…")
		}
		if hl[start+i] {
			h = lipgloss.NewStyle().Background(r.pal.HL).Render(h + strings.Repeat(" ", max(0, inner-ansi.StringWidth(h))))
		}
		body = append(body, h)
	}
	w := 0
	for _, b := range body {
		w = max(w, ansi.StringWidth(b))
	}
	title := attrs["title"]
	if title == "" && lang != "" {
		title = render.LangName(lang)
	}
	if win := attrs["window"]; win == "mac" { // a code window: its title bar in the box's top line
		title = "● ● ●  " + title
	} else if win == "terminal" {
		title = "$ " + firstNonEmpty(title, "terminal")
	}
	w = max(w, ansi.StringWidth(title)+4)
	return r.box(title, body, w, r.pal.Line)
}

// callouts renders <!-- callout: LINE text --> lines as numbered notes, or nil.
func (r *textRenderer) callouts(raw string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(raw), "\n") {
		c, ok, err := codeshow.ParseCallout(l)
		if !ok || err != nil {
			return nil
		}
		r.coN++
		mark := "●"
		if r.coN <= len(circled) {
			mark = string(circled[r.coN-1])
		}
		out = append(out, r.wrap(r.st().Foreground(r.pal.Accent).Render(mark)+" "+r.st().Foreground(r.pal.Muted).Render(fmt.Sprintf("line %d", c.Line))+"  "+r.st().Render(c.Text), r.w)...)
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// hlTokens turns si's highlighted HTML into ANSI colours.
func (r *textRenderer) hlTokens(h string) string {
	col := map[string]lipgloss.Style{
		"c": r.st().Foreground(r.pal.Muted).Italic(true), "s": r.st().Foreground(r.pal.Accent2),
		"k": r.st().Foreground(r.pal.Accent).Bold(true), "n": r.st().Foreground(r.pal.Warn),
		"f": r.st().Foreground(r.pal.Accent2), "v": r.st().Foreground(r.pal.Warn),
		"m": r.st().Foreground(r.pal.Accent).Bold(true), "y": r.st().Foreground(r.pal.Accent2),
		"t": r.st().Foreground(r.pal.OK),
	}
	var b strings.Builder
	last := 0
	for _, m := range tokRE.FindAllStringSubmatchIndex(h, -1) {
		b.WriteString(r.st().Render(html.UnescapeString(h[last:m[0]])))
		b.WriteString(col[h[m[2]:m[3]]].Render(html.UnescapeString(h[m[4]:m[5]])))
		last = m[1]
	}
	b.WriteString(r.st().Render(html.UnescapeString(h[last:])))
	return b.String()
}

// lineSet reads "2,4-5" into a set of line numbers.
func lineSet(spec string) map[int]bool {
	out := map[int]bool{}
	for _, p := range strings.Split(spec, ",") {
		a, b, rng := strings.Cut(strings.TrimSpace(p), "-")
		from, err := strconv.Atoi(a)
		if err != nil {
			continue
		}
		to := from
		if rng {
			if t, err := strconv.Atoi(b); err == nil {
				to = t
			}
		}
		for i := from; i <= to; i++ {
			out[i] = true
		}
	}
	return out
}

// box draws lines (w columns of content) in a rounded box, with an optional
// title in the top border.
func (r *textRenderer) box(title string, lines []string, w int, border color.Color) []string {
	bs := r.st().Foreground(border)
	top := bs.Render("╭" + strings.Repeat("─", w+2) + "╮")
	if title != "" {
		t := " " + ansi.Truncate(title, w-2, "…") + " "
		top = bs.Render("╭─") + r.st().Foreground(r.pal.Muted).Render(t) + bs.Render(strings.Repeat("─", max(0, w+1-ansi.StringWidth(t)))+"╮")
	}
	out := []string{top}
	for _, l := range lines {
		if ansi.StringWidth(l) > w { // a wrapped line can keep a trailing space: keep the border straight
			l = ansi.Truncate(l, w, "")
		}
		out = append(out, bs.Render("│ ")+l+strings.Repeat(" ", max(0, w-ansi.StringWidth(l)))+bs.Render(" │"))
	}
	return append(out, bs.Render("╰"+strings.Repeat("─", w+2)+"╯"))
}

// placeholder is a dim box for what the terminal cannot draw.
func (r *textRenderer) placeholder(head string, lines []string) []string {
	body := []string{r.st().Foreground(r.pal.Accent).Render(head)}
	for _, l := range lines {
		body = append(body, r.st().Foreground(r.pal.Muted).Render(ansi.Truncate(l, r.w-6, "…")))
	}
	w := 0
	for _, b := range body {
		w = max(w, ansi.StringWidth(b))
	}
	return r.box("", body, min(w, r.w-4), r.pal.Line)
}

// ---------- stage diagrams: the chain of nodes as boxes and arrows ----------

func (r *textRenderer) stage(src string) []string {
	var nodes []string
	seen := map[string]bool{}
	for _, l := range strings.Split(src, "\n") {
		l = strings.TrimSpace(l)
		if !strings.Contains(l, "->") || strings.HasPrefix(l, "dot ") {
			continue
		}
		for _, p := range regexp.MustCompile(`-+>`).Split(l, -1) {
			p = strings.TrimSpace(p)
			if i := strings.IndexAny(p, "[("); i > 0 { // A[Label] → Label
				if lbl := strings.Trim(p[i:], "[]()\" "); lbl != "" {
					p = lbl
				}
			}
			if p != "" && !seen[p] {
				seen[p] = true
				nodes = append(nodes, p)
			}
		}
	}
	if len(nodes) == 0 {
		return r.placeholder("◆ stage diagram", strings.Split(strings.TrimSpace(src), "\n"))
	}
	bs := r.st().Foreground(r.pal.Accent)
	arrow := r.st().Foreground(r.pal.Muted).Render("──▶")
	var rows [][3]string
	var cur [3]string
	curW := 0
	centers := map[string]int{} // node -> its column, while there is one row
	for i, n := range nodes {
		bw := ansi.StringWidth(n) + 4
		need := bw
		if curW > 0 {
			need += 3
		}
		if curW > 0 && curW+need > r.w {
			rows = append(rows, cur)
			cur, curW = [3]string{}, 0
			need = bw
		}
		if curW > 0 {
			cur[0] += "   "
			cur[1] += arrow
			cur[2] += "   "
		}
		centers[n] = curW + need - bw + bw/2
		cur[0] += bs.Render("╭" + strings.Repeat("─", bw-2) + "╮")
		cur[1] += bs.Render("│ ") + r.st().Bold(i == 0).Render(n) + bs.Render(" │")
		cur[2] += bs.Render("╰" + strings.Repeat("─", bw-2) + "╯")
		curW += need
	}
	rows = append(rows, cur)
	// the travelling dot (dot A..B 3s) rides the arrows of a one-row diagram
	if m := stageDotRE.FindStringSubmatch(src); m != nil && len(rows) == 1 {
		a, aok := centers[m[1]]
		b, bok := centers[m[2]]
		if aok && bok && b > a {
			dur, _ := strconv.ParseFloat(m[3], 64)
			if dur <= 0 {
				dur = 3
			}
			t := 0.0
			if r.opts != nil {
				t = r.opts.T
			}
			x := a + int(float64(b-a)*math.Mod(t/dur, 1))
			row := rows[0][2]
			w := ansi.StringWidth(row)
			rows[0][2] = ansi.Cut(row, 0, x) + r.st().Foreground(r.pal.Accent).Bold(true).Render("●") + ansi.Cut(row, x+1, w)
		}
	}
	var out []string
	for i, row := range rows {
		if i > 0 {
			out = append(out, r.st().Foreground(r.pal.Muted).Render("  ↓"))
		}
		out = append(out, row[0], row[1], row[2])
	}
	return out
}

// ---------- tables ----------

func (r *textRenderer) table(t *east.Table) []string {
	var rows [][]string
	var header int
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for c := row.FirstChild(); c != nil; c = c.NextSibling() {
			st := r.st()
			if _, ok := row.(*east.TableHeader); ok {
				st = st.Bold(true)
			}
			cells = append(cells, strings.ReplaceAll(r.inline(c, st), "\n", " "))
		}
		if _, ok := row.(*east.TableHeader); ok {
			header = 1
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil
	}
	ncol := len(t.Alignments)
	widths := make([]int, ncol)
	for _, row := range rows {
		for i := 0; i < ncol && i < len(row); i++ {
			widths[i] = max(widths[i], ansi.StringWidth(row[i]))
		}
	}
	// shrink the widest column until the table fits
	for sum(widths)+3*ncol+1 > r.w {
		wi := 0
		for i := range widths {
			if widths[i] > widths[wi] {
				wi = i
			}
		}
		if widths[wi] <= 6 {
			break
		}
		widths[wi]--
	}
	ls := r.st().Foreground(r.pal.Line)
	line := func(l, m, rt string) string {
		parts := make([]string, ncol)
		for i, w := range widths {
			parts[i] = strings.Repeat("─", w+2)
		}
		return ls.Render(l + strings.Join(parts, m) + rt)
	}
	var out []string
	out = append(out, line("┌", "┬", "┐"))
	for ri, row := range rows {
		var b strings.Builder
		b.WriteString(ls.Render("│"))
		for i := 0; i < ncol; i++ {
			c := ""
			if i < len(row) {
				c = row[i]
			}
			if ansi.StringWidth(c) > widths[i] {
				c = ansi.Truncate(c, widths[i], "…")
			}
			pad := widths[i] - ansi.StringWidth(c)
			if t.Alignments[i] == east.AlignRight {
				c = strings.Repeat(" ", pad) + c
			} else if t.Alignments[i] == east.AlignCenter {
				c = strings.Repeat(" ", pad/2) + c + strings.Repeat(" ", pad-pad/2)
			} else {
				c += strings.Repeat(" ", pad)
			}
			b.WriteString(" " + c + " " + ls.Render("│"))
		}
		out = append(out, b.String())
		if ri == 0 && header == 1 {
			out = append(out, line("├", "┼", "┤"))
		}
	}
	return append(out, line("└", "┴", "┘"))
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// ---------- components ----------

func (r *textRenderer) badge(val string) string {
	parts := strings.Split(val, "·")
	kind := "info"
	var text []string
	for i, p := range parts {
		p = strings.TrimSpace(p)
		switch {
		case i > 0 && (p == "ok" || p == "warn" || p == "crit" || p == "info"):
			kind = p
		case i > 0 && strings.HasPrefix(p, "kind="):
			kind = strings.TrimPrefix(p, "kind=")
		case i > 0 && strings.Contains(p, "=") && !strings.Contains(p, " "):
		default:
			text = append(text, p)
		}
	}
	dot := map[string]color.Color{"ok": r.pal.OK, "warn": r.pal.Warn, "crit": r.pal.Crit}[kind]
	if dot == nil {
		dot = r.pal.Accent
	}
	return r.st().Foreground(dot).Render("●") + " " + r.st().Render(strings.Join(text, " · "))
}

func (r *textRenderer) logo(val string) string {
	head, _, _ := strings.Cut(val, "·")
	img, name, _ := strings.Cut(head, "×")
	img, name = strings.TrimSpace(img), strings.TrimSpace(name)
	alt := ""
	if _, a, ok := strings.Cut(val, "alt="); ok {
		alt = strings.TrimSpace(strings.Split(a, "·")[0])
	}
	if alt == "" && img != "" {
		alt = strings.TrimSuffix(img[strings.LastIndex(img, "/")+1:], img[strings.LastIndex(img, "."):])
	}
	if alt == "" {
		alt = "Logo"
	}
	s := r.st().Foreground(r.pal.Accent).Bold(true).Render("▲ " + alt)
	if name != "" {
		s += r.st().Foreground(r.pal.Muted).Render("  │  ") + r.st().Bold(true).Render(name)
	}
	return s
}

// cards draws the ### headings in nodes as a grid of boxes.
func (r *textRenderer) cards(val string, nodes []ast.Node) []string {
	type card struct {
		title, tag, colour string
		body               []ast.Node
	}
	var cs []card
	for _, n := range nodes {
		if h, ok := n.(*ast.Heading); ok && h.Level == 3 {
			parts := strings.Split(render.TextOf(h, r.src), "·")
			c := card{title: strings.TrimSpace(parts[0])}
			for _, p := range parts[1:] {
				p = strings.TrimSpace(p)
				switch strings.ToLower(p) {
				case "teal", "orange", "ok", "crit", "hi":
					c.colour = strings.ToLower(p)
				default:
					c.tag = strings.TrimSpace(strings.TrimPrefix(c.tag+" · "+p, " · "))
				}
			}
			cs = append(cs, c)
		} else if len(cs) > 0 {
			cs[len(cs)-1].body = append(cs[len(cs)-1].body, n)
		}
	}
	if len(cs) == 0 {
		return nil
	}
	cols, _ := strconv.Atoi(strings.TrimSpace(val))
	if cols <= 0 {
		cols = len(cs)
	}
	for cols > 1 && (r.w-(cols-1)*2)/cols < 22 {
		cols--
	}
	cw := (r.w - (cols-1)*2) / cols
	var boxes [][]string
	for _, c := range cs {
		border := map[string]color.Color{
			"teal": r.pal.Accent2, "ok": r.pal.OK, "orange": r.pal.Warn, "crit": r.pal.Crit, "hi": r.pal.Accent}[c.colour]
		if border == nil {
			border = r.pal.Line
		}
		sub := &textRenderer{pal: r.pal, dir: r.dir, w: cw - 4, maxH: r.maxH, k: 1 << 30, src: r.src}
		var body []string
		if c.tag != "" {
			body = append(body, r.st().Foreground(r.pal.Muted).Render(strings.ToUpper(c.tag)))
		}
		body = append(body, sub.wrap(r.st().Bold(true).Render(c.title), cw-4)...)
		for _, n := range c.body {
			lines, _ := sub.block(n)
			body = append(body, lines...)
		}
		boxes = append(boxes, r.box("", body, cw-4, border))
	}
	var out []string
	for i := 0; i < len(boxes); i += cols {
		row := boxes[i:min(i+cols, len(boxes))]
		h := 0
		for _, b := range row {
			h = max(h, len(b))
		}
		for y := 0; y < h; y++ {
			var line []string
			for _, b := range row {
				l := strings.Repeat(" ", cw)
				switch {
				case y < len(b)-1:
					l = b[y]
				case y == h-1:
					l = b[len(b)-1] // the bottom border, on the last row of the tallest card
				default:
					l = sideLine(b[len(b)-2], cw) // a shorter card keeps its sides down to the row's bottom
				}
				line = append(line, l)
			}
			out = append(out, strings.Join(line, "  "))
		}
		out = append(out, "")
	}
	return out[:len(out)-1]
}

// ---------- helpers ----------

func blockText(n ast.Node, src []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return html.UnescapeString(tagRE.ReplaceAllString(s, " ")) }

func soleImage(p *ast.Paragraph) *ast.Image {
	if p.ChildCount() != 1 {
		return nil
	}
	img, _ := p.FirstChild().(*ast.Image)
	return img
}

// sideLine is an empty line between a card's side borders, coloured like row.
func sideLine(row string, w int) string {
	left := ansi.Truncate(row, 1, "")
	right := ansi.Cut(row, ansi.StringWidth(row)-1, ansi.StringWidth(row))
	return left + strings.Repeat(" ", w-2) + right
}

func padLeft(s string, n int) string { return strings.Repeat(" ", max(0, n-len(s))) + s }

// ---------- mind maps: each map as a tree ----------

var (
	mmTargetRE = regexp.MustCompile(`\s*(?:→|\s->)\s*(\S.*)$`)
	mmAttrsRE  = regexp.MustCompile(`\s*\{([^{}]*)\}\s*$`)
	mmBadgeRE  = regexp.MustCompile(`(?:^|\s)badge=(\S+)`)
	mmPicRE    = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
)

// mindmap draws a ```mindmap fence (or a mind map page's body) as trees: a
// `# Title` is a root, list items its branches; links end in →, pictures
// are ▨, sticky notes ✎, code ‹lang›.
func (r *textRenderer) mindmap(src string) []string {
	type item struct {
		depth int
		text  string
	}
	var out []string
	var items []item
	flush := func() {
		open := map[int]bool{}
		for i, it := range items {
			last := true
			for j := i + 1; j < len(items); j++ {
				if items[j].depth < it.depth {
					break
				}
				if items[j].depth == it.depth {
					last = false
					break
				}
			}
			prefix := ""
			for d := 0; d < it.depth; d++ {
				if open[d] {
					prefix += "│  "
				} else {
					prefix += "   "
				}
			}
			branch := "├─ "
			if last {
				branch = "└─ "
			}
			open[it.depth] = !last
			line := r.st().Foreground(r.pal.Muted).Render(prefix+branch) + it.text
			out = append(out, ansi.Truncate(line, r.w, "…"))
		}
		items = nil
	}
	fence := false
	for _, l := range strings.Split(src, "\n") {
		t := strings.TrimSpace(l)
		depth := (len(l) - len(strings.TrimLeft(l, " \t"))) / 2
		switch {
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			if !fence {
				lang := strings.Fields(strings.TrimLeft(t, "`~") + " code")[0]
				items = append(items, item{depth, r.st().Foreground(r.pal.Accent2).Render("‹" + lang + "›")})
			}
			fence = !fence
		case fence:
		case strings.HasPrefix(t, "# "):
			flush()
			if len(out) > 0 {
				out = append(out, "")
			}
			title := strings.TrimSpace(customIDTail.ReplaceAllString(t[2:], ""))
			out = append(out, r.st().Bold(true).Foreground(r.pal.Accent).Render("✲ "+stripMarks(title)))
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ "):
			items = append(items, item{depth, r.mmTopic(t[2:])})
		}
	}
	flush()
	return out
}

var customIDTail = regexp.MustCompile(`\s*\{#[^}]*\}\s*$`)

// mmTopic is one topic line as text: sticky ✎, picture ▨, badge, link →.
func (r *textRenderer) mmTopic(t string) string {
	pre := ""
	if rest, ok := strings.CutPrefix(t, "[!sticky]"); ok {
		pre, t = "✎ ", strings.TrimSpace(rest)
	}
	target := ""
	if m := mmTargetRE.FindStringSubmatch(t); m != nil {
		target, t = m[1], mmTargetRE.ReplaceAllString(t, "")
	}
	badge := ""
	if m := mmAttrsRE.FindStringSubmatch(t); m != nil {
		t = mmAttrsRE.ReplaceAllString(t, "")
		if b := mmBadgeRE.FindStringSubmatch(m[1]); b != nil {
			badge = " " + r.st().Foreground(r.pal.Muted).Render("["+strings.ReplaceAll(b[1], "_", " ")+"]")
		}
	}
	if mmPicRE.MatchString(t) {
		pre += "▨ "
		t = mmPicRE.ReplaceAllString(t, "")
	}
	s := pre + stripMarks(strings.TrimSpace(t)) + badge
	if target != "" {
		s += " " + r.st().Foreground(r.pal.Accent).Render("→ "+target)
	}
	return s
}
