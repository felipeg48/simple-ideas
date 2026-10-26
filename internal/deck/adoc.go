package deck

import (
	"regexp"
	"strings"
)

// adocToMarkdown converts the si-relevant subset of AsciiDoc into the Markdown
// that Parse understands, so `.adoc`/`.asciidoc` decks feed the same pipeline.
// It is a focused preprocessor, not a full AsciiDoc implementation (stdlib only,
// no dependency): document header + attributes, section headings (as slides),
// lists, source/listing blocks, admonitions, links, images, inline bold and
// comments. Anything else is passed through unchanged; see docs/asciidoc.md.
//
// A deck that already begins with an si `---` front-matter block is left alone
// (that block passes through); otherwise an AsciiDoc `= Title` header and its
// `:attr: value` lines become front matter + an `# Title` slide.
func adocToMarkdown(src string) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var out []string
	i := adocHeader(lines, &out)

	inFence := false // inside a converted code fence (pass body verbatim)
	inComment := false
	pendingLang := "" // from a preceding [source,lang]
	wrote := len(out) > 0

	for ; i < len(lines); i++ {
		line := lines[i]

		// block comment ////
		if strings.TrimSpace(line) == "////" {
			inComment = !inComment
			continue
		}
		if inComment {
			continue
		}

		// listing / literal block delimiters -> ``` fences
		if !inFence && (line == "----" || line == "....") {
			out = append(out, "```"+pendingLang)
			pendingLang, inFence, wrote = "", true, true
			continue
		}
		if inFence {
			if line == "----" || line == "...." {
				out = append(out, "```")
				inFence = false
			} else {
				out = append(out, line) // code body verbatim
			}
			continue
		}

		// [source,go] / [source] / [,go] attribute line before a block
		if m := adocSourceRE.FindStringSubmatch(line); m != nil {
			pendingLang = m[1]
			continue
		}
		// a stray block attribute like [quote] or [.role] we don't handle: drop it
		if adocAttrRE.MatchString(line) {
			continue
		}
		// line comment //
		if strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "///") {
			continue
		}

		// section headings -> slides
		if m := adocHeadingRE.FindStringSubmatch(line); m != nil {
			level := len(m[1]) // number of '='
			text := adocInline(m[2])
			if level >= 2 { // == and deeper
				if level == 2 && wrote {
					out = append(out, "", "---", "") // new slide
				}
				out = append(out, strings.Repeat("#", level)+" "+text)
			} else { // level-0 '=' in the body (rare): a top title
				out = append(out, "# "+text)
			}
			wrote = true
			continue
		}

		// thematic break (not an si slide break, which is ---)
		if strings.TrimSpace(line) == "'''" {
			out = append(out, "***")
			wrote = true
			continue
		}

		// admonitions
		if m := adocAdmonRE.FindStringSubmatch(line); m != nil {
			label := m[1][:1] + strings.ToLower(m[1][1:])
			out = append(out, "> **"+label+":** "+adocInline(m[2]))
			wrote = true
			continue
		}

		// lists
		if m := adocULRE.FindStringSubmatch(line); m != nil {
			out = append(out, strings.Repeat("  ", len(m[1])-1)+"- "+adocInline(m[2]))
			wrote = true
			continue
		}
		if m := adocOLRE.FindStringSubmatch(line); m != nil {
			out = append(out, strings.Repeat("  ", len(m[1])-1)+"1. "+adocInline(m[2]))
			wrote = true
			continue
		}

		// inline: images, links, bold
		line = adocInline(line)
		if strings.TrimSpace(line) != "" {
			wrote = true
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

var (
	adocHeadingRE = regexp.MustCompile(`^(=+)\s+(.*)$`)
	adocSourceRE  = regexp.MustCompile(`^\[source(?:%[^,\]]*)?\s*,?\s*([a-zA-Z0-9]*)\]\s*$`)
	adocAttrRE    = regexp.MustCompile(`^\[[^\]]*\]\s*$`)
	adocAdmonRE   = regexp.MustCompile(`^(NOTE|TIP|IMPORTANT|WARNING|CAUTION):\s+(.*)$`)
	adocULRE      = regexp.MustCompile(`^(\*+)\s+(.*)$`)
	adocOLRE      = regexp.MustCompile(`^(\.+)\s+(.*)$`)
	adocBlockImg  = regexp.MustCompile(`^image::(\S+)\[([^\]]*)\]\s*$`)
	adocInlineImg = regexp.MustCompile(`image:(\S+)\[([^\]]*)\]`)
	adocLinkRE    = regexp.MustCompile(`(?:link:)?(https?://[^\[\s]+)\[([^\]]*)\]`)
	adocBoldRE    = regexp.MustCompile(`(^|[\s(])\*(\S[^*]*?\S|\S)\*([\s).,;:!?]|$)`)
)

// adocInline converts block/inline images, links and bold on one line.
func adocInline(line string) string {
	if m := adocBlockImg.FindStringSubmatch(line); m != nil {
		return "![" + m[2] + "](" + m[1] + ")"
	}
	line = adocInlineImg.ReplaceAllString(line, "![$2]($1)")
	line = adocLinkRE.ReplaceAllString(line, "[$2]($1)")
	line = adocBoldRE.ReplaceAllString(line, "$1**$2**$3")
	return line
}

// adocHeader consumes an AsciiDoc document header (= Title plus :attr: value
// lines) at the top of lines, appending an si front-matter block and the title
// slide to out, and returns the index to continue from. If the deck already
// starts with an si `---` front-matter block it is copied verbatim; if there is
// no header at all, nothing is consumed and 0 is returned.
func adocHeader(lines []string, out *[]string) int {
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return i
	}
	// existing si front matter: copy through the closing ---
	if strings.TrimSpace(lines[i]) == "---" {
		*out = append(*out, lines[i])
		for i++; i < len(lines); i++ {
			*out = append(*out, lines[i])
			if strings.TrimSpace(lines[i]) == "---" {
				return i + 1
			}
		}
		return i
	}
	// AsciiDoc document title
	m := adocHeadingRE.FindStringSubmatch(lines[i])
	if m == nil || len(m[1]) != 1 {
		return 0 // no recognizable header; let the main loop handle everything
	}
	title := m[2]
	i++
	fm := map[string]string{}
	attr := regexp.MustCompile(`^:([a-zA-Z0-9_-]+):\s*(.*)$`)
	for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
		if a := attr.FindStringSubmatch(lines[i]); a != nil {
			switch strings.ToLower(a[1]) {
			case "author":
				fm["author"] = a[2]
			case "revdate", "date":
				fm["date"] = a[2]
			case "theme", "mode", "accent", "transition", "aspect", "mermaid", "lang", "background":
				fm[strings.ToLower(a[1])] = a[2]
			}
		}
		i++
	}
	*out = append(*out, "---", "title: "+title)
	for _, k := range []string{"author", "date", "theme", "mode", "accent", "transition", "aspect", "mermaid", "lang", "background"} {
		if v, ok := fm[k]; ok && v != "" {
			*out = append(*out, k+": "+v)
		}
	}
	*out = append(*out, "---", "", "# "+title, "")
	return i
}
