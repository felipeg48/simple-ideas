package deck

import (
	"fmt"
	"regexp"
	"strings"
)

// MarkdownToAdoc writes si Markdown as the AsciiDoc that adocToMarkdown reads
// back: the inverse of the subset in docs/asciidoc.md. Front matter, si
// directives (<!-- … -->), Notes:/Points:, raw HTML and anything it does not
// know pass through unchanged, as they do in a .adoc deck. The docs site uses
// it to show every example in both syntaxes.
func MarkdownToAdoc(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var out []string
	i := 0
	// front matter: verbatim (a .adoc deck may start with it)
	if len(lines) > 1 && strings.TrimSpace(lines[0]) == "---" && mdKeyRE.MatchString(lines[1]) {
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				out = append(out, lines[:j+1]...)
				i = j + 1
				break
			}
		}
	}
	inComment := false
	var quote []string // > lines waiting to become a ____ block
	flushQuote := func() {
		if len(quote) == 0 {
			return
		}
		if m := mdAlertRE.FindStringSubmatch(quote[0]); m != nil { // > [!NOTE]
			if len(quote) == 2 {
				out = append(out, m[1]+": "+mdInline(quote[1]))
			} else {
				out = append(out, "["+m[1]+"]", "====")
				for _, q := range quote[1:] {
					out = append(out, mdInline(q))
				}
				out = append(out, "====")
			}
			quote = nil
			return
		}
		out = append(out, "____")
		for _, q := range quote {
			out = append(out, mdInline(q))
		}
		out = append(out, "____")
		quote = nil
	}
	for ; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(line)
		if inComment { // a multi-line directive or comment: verbatim
			out = append(out, line)
			inComment = !strings.Contains(line, "-->")
			continue
		}
		if m := mdQuoteRE.FindStringSubmatch(line); m != nil {
			quote = append(quote, m[1])
			continue
		}
		flushQuote()
		if strings.HasPrefix(trim, "<!--") {
			out = append(out, line)
			inComment = !strings.Contains(line, "-->")
			continue
		}
		// a fenced code block: [source,lang,…] + ---- (its body verbatim)
		if m := mdFenceRE.FindStringSubmatch(line); m != nil && m[1] == "" {
			fence, info := m[2], strings.TrimSpace(m[3])
			j := i + 1
			for j < len(lines) && !(strings.HasPrefix(strings.TrimSpace(lines[j]), fence) && strings.Trim(strings.TrimSpace(lines[j]), "`~") == "") {
				j++
			}
			out = append(out, adocSource(info)...)
			out = append(out, "----")
			out = append(out, lines[i+1:min(j, len(lines))]...)
			out = append(out, "----")
			i = j
			continue
		}
		if m := mdHeadingRE.FindStringSubmatch(line); m != nil {
			out = append(out, strings.Repeat("=", len(m[1]))+" "+mdInline(m[2]))
			continue
		}
		if trim == "***" || trim == "___" || trim == "* * *" {
			out = append(out, "'''")
			continue
		}
		if mdTableRow(trim) && i+1 < len(lines) && mdTableSepRE.MatchString(strings.TrimSpace(lines[i+1])) {
			out = append(out, "|===")
			out = append(out, adocRow(trim))
			out = append(out, "")
			for i += 2; i < len(lines) && mdTableRow(strings.TrimSpace(lines[i])); i++ {
				out = append(out, adocRow(strings.TrimSpace(lines[i])))
			}
			out = append(out, "|===")
			i--
			continue
		}
		if m := mdListRE.FindStringSubmatch(line); m != nil {
			depth := len(strings.ReplaceAll(m[1], "\t", "  "))/2 + 1
			mark := "*"
			if m[2] != "-" && m[2] != "*" && m[2] != "+" {
				mark = "."
			}
			out = append(out, strings.Repeat(mark, depth)+" "+mdInline(m[3]))
			continue
		}
		if m := mdImageLineRE.FindStringSubmatch(trim); m != nil {
			out = append(out, "image::"+m[2]+"["+m[1]+"]")
			continue
		}
		out = append(out, mdInline(line))
	}
	flushQuote()
	return strings.Join(out, "\n")
}

var (
	mdKeyRE       = regexp.MustCompile(`^\w[\w.-]*:`)
	mdFenceRE     = regexp.MustCompile("^( *)(`{3,}|~{3,})(.*)$")
	mdHeadingRE   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	mdQuoteRE     = regexp.MustCompile(`^>\s?(.*)$`)
	mdAlertRE     = regexp.MustCompile(`^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$`)
	mdListRE      = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)
	mdTableSepRE  = regexp.MustCompile(`^\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)*\|?$`)
	mdImageLineRE = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)\s]+)\)$`)
	mdImageRE     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	mdLinkRE      = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	mdBoldRE      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalicRE    = regexp.MustCompile(`(^|[^*\w])\*([^*\s][^*]*?)\*([^*\w]|$)`)
	mdFenceAttrRE = regexp.MustCompile(`(\w[\w-]*)(?:=("[^"]*"|\S+))?`)
)

// adocSource turns a fence's info string (go window=mac title="a.go" hl=2)
// into the AsciiDoc lines before the ---- block: a .Title line and
// [source,go,highlight=2,window=mac].
func adocSource(info string) []string {
	if info == "" {
		return nil
	}
	lang, rest, _ := strings.Cut(info, " ")
	var title string
	parts := []string{"source", lang}
	for _, m := range mdFenceAttrRE.FindAllStringSubmatch(rest, -1) {
		k, v := m[1], m[2]
		switch k {
		case "title":
			title = strings.Trim(v, `"`)
		case "hl":
			parts = append(parts, "highlight="+strings.NewReplacer("-", "..", ",", ";").Replace(strings.Trim(v, `"`)))
		case "linenos":
			parts = append(parts, "linenums")
		default:
			if v == "" {
				parts = append(parts, k)
			} else {
				parts = append(parts, k+"="+v)
			}
		}
	}
	var out []string
	if title != "" {
		out = append(out, "."+title)
	}
	return append(out, fmt.Sprintf("[%s]", strings.Join(parts, ",")))
}

func mdTableRow(s string) bool { return strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|") }

// adocRow is one GFM table row as an AsciiDoc row: | a | b.
func adocRow(s string) string {
	cells := strings.Split(strings.Trim(s, "|"), "|")
	var b strings.Builder
	for k, c := range cells {
		if k > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("| " + mdInline(strings.TrimSpace(c)))
	}
	return b.String()
}

// mdInline converts images, links, bold and italic on one line, leaving code
// spans alone.
func mdInline(line string) string {
	parts := strings.Split(line, "`")
	for k := 0; k < len(parts); k += 2 { // even parts are outside `code`
		p := parts[k]
		p = mdImageRE.ReplaceAllString(p, "image:$2[$1]")
		p = mdLinkRE.ReplaceAllStringFunc(p, func(s string) string {
			m := mdLinkRE.FindStringSubmatch(s)
			if strings.HasPrefix(m[2], "http://") || strings.HasPrefix(m[2], "https://") {
				return m[2] + "[" + m[1] + "]"
			}
			return "link:" + m[2] + "[" + m[1] + "]"
		})
		p = mdItalicRE.ReplaceAllString(p, "${1}_${2}_$3")
		p = mdBoldRE.ReplaceAllString(p, "*$1*")
		parts[k] = p
	}
	return strings.Join(parts, "`")
}
