package docsite

import (
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// mdExampleRE opens a Markdown example: ```markdown (or ````markdown, for one
// with fences inside), maybe indented under a list item.
var mdExampleRE = regexp.MustCompile("^( *)(`{3,})markdown\\s*$")

// withAdoc shows every Markdown example in md beside its AsciiDoc: the fence
// becomes a group of two tabs, Markdown and AsciiDoc (si's tabs; a reader's
// choice carries to every page).
func withAdoc(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		m := mdExampleRE.FindStringSubmatch(lines[i])
		if m == nil {
			out = append(out, lines[i])
			continue
		}
		indent, fence := m[1], m[2]
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) != fence {
			j++
		}
		if j == len(lines) { // unclosed: leave it be
			out = append(out, lines[i:]...)
			break
		}
		body := make([]string, 0, j-i-1)
		for _, l := range lines[i+1 : j] {
			body = append(body, strings.TrimPrefix(l, indent))
		}
		adoc := deck.MarkdownToAdoc(strings.Join(body, "\n"))
		out = append(out, indent+"<!-- tabs -->", indent+fence+"markdown title=Markdown")
		out = append(out, lines[i+1:j]...)
		out = append(out, indent+fence, indent+fence+"asciidoc title=AsciiDoc")
		for _, l := range strings.Split(adoc, "\n") {
			if l != "" {
				l = indent + l
			}
			out = append(out, l)
		}
		out = append(out, indent+fence)
		i = j
	}
	return strings.Join(out, "\n")
}

// mdExamples returns the Markdown examples in md (dedented), for tests.
func mdExamples(md string) []string {
	lines := strings.Split(md, "\n")
	var ex []string
	for i := 0; i < len(lines); i++ {
		m := mdExampleRE.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		var body []string
		j := i + 1
		for ; j < len(lines) && strings.TrimSpace(lines[j]) != m[2]; j++ {
			body = append(body, strings.TrimPrefix(lines[j], m[1]))
		}
		ex = append(ex, strings.Join(body, "\n"))
		i = j
	}
	return ex
}
