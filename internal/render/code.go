package render

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
)

// Code blocks, phase 2 of the document features: fence attributes
// (```java title="App.java" hl="3,5-6" linenos start=12), per-line rendering for
// highlighted lines and line numbers, <N> callouts and <!-- tabs --> groups.
// Line numbers and callout markers are drawn by CSS from data attributes, so
// copying a block copies only the code.

var fenceAttrRE = regexp.MustCompile(`([\w-]+)(?:="([^"]*)"|=(\S+))?`)

// fenceAttrs parses the info string after the language.
func fenceAttrs(info string) map[string]string {
	_, rest, _ := strings.Cut(strings.TrimSpace(info), " ")
	out := map[string]string{}
	for _, m := range fenceAttrRE.FindAllStringSubmatch(rest, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		out[m[1]] = v
	}
	return out
}

// lineSet parses "3,5-6" into a set of 1-based line numbers.
func lineSet(spec string) map[int]bool {
	out := map[int]bool{}
	for _, r := range strings.Split(spec, ",") {
		a, b, ranged := strings.Cut(strings.TrimSpace(r), "-")
		from, err1 := strconv.Atoi(a)
		to := from
		var err2 error
		if ranged {
			to, err2 = strconv.Atoi(b)
		}
		if err1 != nil || err2 != nil {
			continue
		}
		for i := from; i <= to; i++ {
			out[i] = true
		}
	}
	return out
}

// calloutMarkRE finds <N> markers at the end of a code line, bare or as a
// trailing comment (// <1>, # <1>, -- <1>, ; <1>, <!-- 1 -->).
var calloutMarkRE = regexp.MustCompile(`^(.*?)\s*(?:(?://|#|--|;)\s*)?((?:<\d+>\s*)+)$|^(.*?)\s*<!--\s*(\d+)\s*-->\s*$`)
var calloutNumRE = regexp.MustCompile(`\d+`)

// splitHTMLLines splits highlighted HTML at newlines, closing the spans open
// at the end of a line and reopening them on the next, so every line is valid
// on its own.
var htmlTokRE = regexp.MustCompile(`<span[^>]*>|</span>|\n|[^<\n]+|<`)

func splitHTMLLines(h string) []string {
	var lines []string
	var cur strings.Builder
	var open []string
	for _, t := range htmlTokRE.FindAllString(h, -1) {
		switch {
		case t == "\n":
			cur.WriteString(strings.Repeat("</span>", len(open)))
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(strings.Join(open, ""))
		case strings.HasPrefix(t, "<span"):
			open = append(open, t)
			cur.WriteString(t)
		case t == "</span>":
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
			cur.WriteString(t)
		default:
			cur.WriteString(t)
		}
	}
	return append(lines, cur.String())
}

// codeHTML is the inner HTML of a code block: highlighted, and line by line
// when it has highlighted lines, line numbers or callout markers.
func (dr *deckRenderer) codeHTML(body, lang string, attrs map[string]string, callouts bool) string {
	if lang == "diff" || lang == "patch" {
		return diffHTML(body)
	}
	lines := strings.Split(body, "\n")
	marks := map[int][]string{}
	if callouts {
		for i, l := range lines {
			m := calloutMarkRE.FindStringSubmatch(l)
			switch {
			case m == nil:
			case m[2] != "":
				lines[i], marks[i] = m[1], calloutNumRE.FindAllString(m[2], -1)
			case m[4] != "":
				lines[i], marks[i] = m[3], []string{m[4]}
			}
		}
	}
	h := highlight(strings.Join(lines, "\n"), lang)
	hl := lineSet(attrs["hl"])
	_, nums := attrs["linenos"]
	if len(hl) == 0 && !nums && len(marks) == 0 {
		return h
	}
	start := 1
	if n, err := strconv.Atoi(attrs["start"]); err == nil {
		start = n
	}
	var b strings.Builder
	for i, l := range splitHTMLLines(h) {
		cls := "si-line"
		if hl[i+1] {
			cls += " si-hl"
		}
		b.WriteString(`<span class="` + cls + `">`)
		if nums {
			fmt.Fprintf(&b, `<span class="si-ln" data-n="%d"></span>`, start+i)
		}
		b.WriteString(l)
		for _, n := range marks[i] {
			fmt.Fprintf(&b, `<span class="si-co" data-co="%s" aria-label="callout %s"></span>`, n, n)
		}
		b.WriteString("</span>")
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

var hunkRE = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// diffHTML renders a unified diff line by line: file headers, hunk headers,
// added and removed lines, each with its old and new line number (drawn by
// CSS, so Copy copies the patch as it is).
func diffHTML(body string) string {
	var b strings.Builder
	old, nw := 0, 0
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		cls, o, n := "si-diff-meta", "", ""
		switch {
		case strings.HasPrefix(l, "@@"):
			cls = "si-diff-hunk"
			if m := hunkRE.FindStringSubmatch(l); m != nil {
				old, _ = strconv.Atoi(m[1])
				nw, _ = strconv.Atoi(m[2])
			}
		case strings.HasPrefix(l, "diff --git"), strings.HasPrefix(l, "index "), strings.HasPrefix(l, "--- "), strings.HasPrefix(l, "+++ "),
			strings.HasPrefix(l, "new file"), strings.HasPrefix(l, "deleted file"), strings.HasPrefix(l, "similarity"), strings.HasPrefix(l, "rename "), strings.HasPrefix(l, "\\"):
		case strings.HasPrefix(l, "+"):
			cls, n = "si-diff-add", strconv.Itoa(nw)
			nw++
		case strings.HasPrefix(l, "-"):
			cls, o = "si-diff-del", strconv.Itoa(old)
			old++
		default:
			cls, o, n = "si-diff-ctx", strconv.Itoa(old), strconv.Itoa(nw)
			old++
			nw++
		}
		fmt.Fprintf(&b, `<span class="si-line %s"><span class="si-ln" data-n="%s"></span><span class="si-ln" data-n="%s"></span>%s</span>`, cls, o, n, html.EscapeString(l))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

var calloutLineRE = regexp.MustCompile(`^<(\d+)>\s+(.+)$`)

// calloutList reports whether n is a paragraph whose every line is `<N> text`
// and returns the items.
func calloutList(n ast.Node, src []byte) ([][2]string, bool) {
	p, ok := n.(*ast.Paragraph)
	if !ok || p.Lines().Len() == 0 {
		return nil, false
	}
	var items [][2]string
	for i := 0; i < p.Lines().Len(); i++ {
		seg := p.Lines().At(i)
		m := calloutLineRE.FindStringSubmatch(strings.TrimSpace(string(seg.Value(src))))
		if m == nil {
			return nil, false
		}
		items = append(items, [2]string{m[1], m[2]})
	}
	return items, true
}

var langNames = map[string]string{"bash": "bash", "sh": "bash", "shell": "bash", "js": "JavaScript", "javascript": "JavaScript",
	"ts": "TypeScript", "typescript": "TypeScript", "py": "Python", "python": "Python", "go": "Go", "java": "Java",
	"kotlin": "Kotlin", "csharp": "C#", "cs": "C#", "cpp": "C++", "c": "C", "ruby": "Ruby", "rust": "Rust",
	"sql": "SQL", "yaml": "YAML", "json": "JSON", "powershell": "PowerShell", "php": "PHP", "swift": "Swift"}

// codeFeatures runs on the top-level blocks of a slide or a document part:
// a paragraph of `<N> text` lines after a code block becomes its callout
// list (and the block's markers are drawn), and `<!-- tabs -->` groups the
// code blocks that follow it into tabs. It runs before the slide transform,
// which never sees the tabs directive.
func (dr *deckRenderer) codeFeatures(doc ast.Node, src []byte, sc *slideCtx) {
	defer dr.cardsPass(doc, src, sc) // cards group blocks too, on slides and in documents
	dr.codeWindows(doc, src, sc)
	if dr.callouts == nil {
		dr.callouts = map[ast.Node]bool{}
	}
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		fc, ok := c.(*ast.FencedCodeBlock)
		if !ok || c.NextSibling() == nil {
			continue
		}
		items, ok := calloutList(c.NextSibling(), src)
		if !ok {
			continue
		}
		dr.callouts[fc] = true
		marked := map[string]bool{}
		for i := 0; i < fc.Lines().Len(); i++ {
			seg := fc.Lines().At(i)
			if m := calloutMarkRE.FindStringSubmatch(strings.TrimRight(string(seg.Value(src)), "\n")); m != nil {
				for _, n := range calloutNumRE.FindAllString(m[2]+m[4], -1) {
					marked[n] = true
				}
			}
		}
		var b strings.Builder
		b.WriteString(`<ol class="si-callouts">`)
		listed := map[string]bool{}
		for _, it := range items {
			listed[it[0]] = true
			if !marked[it[0]] {
				dr.problem(sc, "callout <%s> has no <%s> marker in the code block above it", it[0], it[0])
			}
			fmt.Fprintf(&b, `<li data-co="%s"><span class="si-co" data-co="%s" aria-hidden="true"></span><span>%s</span></li>`, it[0], it[0], dr.inline(it[1]))
		}
		for n := range marked {
			if !listed[n] {
				dr.problem(sc, "code marker <%s> has no `<%s> …` line in the callout list below the block", n, n)
			}
		}
		b.WriteString("</ol>")
		list := &rawNode{html: b.String()}
		doc.ReplaceChild(doc, c.NextSibling(), list)
	}
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		hb, ok := c.(*ast.HTMLBlock)
		if !ok {
			c = next
			continue
		}
		name, _, isDir := deck.ParseDirective(blockText(hb, src))
		if !isDir || name != "tabs" {
			c = next
			continue
		}
		group := &divNode{class: "si-tabs"}
		var bar strings.Builder
		bar.WriteString(`<div class="si-tabs-bar" role="tablist">`)
		n := 0
		for k := next; k != nil; {
			kn := k.NextSibling()
			fc, ok := k.(*ast.FencedCodeBlock)
			if !ok {
				break
			}
			attrs := map[string]string{}
			if fc.Info != nil {
				attrs = fenceAttrs(string(fc.Info.Segment.Value(src)))
			}
			label := attrs["title"]
			if label == "" {
				lang := strings.ToLower(string(fc.Language(src)))
				if label = langNames[lang]; label == "" {
					label = lang
				}
			}
			if label == "" {
				label = fmt.Sprintf("Tab %d", n+1)
			}
			sel, hidden := "true", [][2]string{{"data-label", label}}
			if n > 0 {
				sel = "false"
				hidden = append(hidden, [2]string{"hidden", ""})
			}
			fmt.Fprintf(&bar, `<button type="button" role="tab" aria-selected="%s" data-label="%s">%s</button>`, sel, html.EscapeString(label), html.EscapeString(label))
			tab := &divNode{class: "si-tab", attrs: hidden}
			doc.RemoveChild(doc, k)
			tab.AppendChild(tab, k)
			if r, ok := kn.(*rawNode); ok && strings.HasPrefix(r.html, `<ol class="si-callouts">`) {
				kn2 := kn.NextSibling()
				doc.RemoveChild(doc, kn)
				tab.AppendChild(tab, kn)
				kn = kn2
			}
			group.AppendChild(group, tab)
			n++
			k = kn
		}
		bar.WriteString("</div>")
		if n == 0 {
			dr.problem(sc, "<!-- tabs --> must be followed by code blocks")
			doc.RemoveChild(doc, c)
			c = next
			continue
		}
		group.InsertBefore(group, group.FirstChild(), &rawNode{html: bar.String()})
		after := c.NextSibling()
		doc.ReplaceChild(doc, c, group)
		c = after
	}
}
