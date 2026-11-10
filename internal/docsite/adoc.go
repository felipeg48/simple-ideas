package docsite

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/examples"
	"github.com/felipeg48/simple-ideas/internal/brand"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// mdExampleRE opens a Markdown example: ```markdown (or ````markdown, for one
// with fences inside), maybe indented under a list item.
var mdExampleRE = regexp.MustCompile("^( *)(`{3,})markdown\\s*$")

// withAdoc shows every Markdown example in md beside its AsciiDoc: the fence
// becomes a group of two tabs, Markdown and AsciiDoc (si's tabs; a reader's
// choice carries to every page). With result, what the example looks like
// follows the tabs (result returns "" when it cannot show one).
func withAdoc(md string, result func(example string) string) string {
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
		if result != nil {
			if r := result(strings.Join(body, "\n")); r != "" {
				out = append(out, "", indent+"*Result:*", "", indent+r)
			}
		}
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

// liveExamples writes the examples of the site's pages as small pages in
// live/ (beside the examples si ships, so their assets/ paths resolve), each
// checked first in a scratch copy of examples/: only one that renders without
// a problem gets a result.
type liveExamples struct {
	write func(rel, text string) error
	tmp   string // a copy of examples/, to render in
	n     int
}

// result is the result callback for a page in dir ("rules", "features", "").
func (l *liveExamples) result(dir string) func(string) string {
	up := ""
	if dir != "" && dir != "." {
		up = strings.Repeat("../", strings.Count(dir, "/")+1)
	}
	return func(md string) string {
		if pageLinkRE.MatchString(md) { // a menu or contents of pages that are not there
			return ""
		}
		md = audienceQRRE.ReplaceAllString(md, "<!-- qr: "+brand.Site+"features/audience.html") // a live link only under si play --audience: a sample here
		forms := []string{asLivePage(md)}
		if !strings.HasPrefix(md, "---\n") { // slides, or a document's part (alert, carousel)
			forms = append(forms, "---\ntitle: Example\ntype: article\n---\n\n"+md)
		}
		for _, src := range forms {
			if r := l.try(src, up); r != "" {
				return r
			}
		}
		return ""
	}
}

// try renders one form of an example; "" when it has a problem.
func (l *liveExamples) try(src, up string) string {
	l.n++
	name := fmt.Sprintf("ex-%03d.md", l.n)
	p := filepath.Join(l.tmp, name)
	if os.WriteFile(p, []byte(src), 0o644) != nil {
		return ""
	}
	d, err := deck.Parse(p)
	if err != nil {
		return ""
	}
	res := render.Deck(d, render.Options{})
	if len(res.Problems) > 0 || (!d.IsDoc() && len(d.Slides) == 0) || l.write("live/"+name, src) != nil {
		return ""
	}
	if d.IsDoc() {
		return "<!-- embed: " + up + "live/" + name + " full bare -->"
	}
	if strings.Contains(src, titleSlide) { // only the example's own slides
		return fmt.Sprintf("<!-- embed: %slive/%s 2-%d bare -->", up, name, len(d.Slides))
	}
	return "<!-- embed: " + up + "live/" + name + " bare -->"
}

var audienceQRRE = regexp.MustCompile(`<!--\s*qr:\s*audience\b`)

var pageLinkRE = regexp.MustCompile(`\]\([^)#:\s]+(\.md|/)(#[^)]*)?\)`)

// titleSlide is put before an example that is slides without a deck's
// opening # Title (the result shows from the slide after it).
const titleSlide = "# Example\n\n---\n\n"

// asLivePage makes an example a page of its own: a deck unless its front
// matter names a type (pages of the site are documents by default).
func asLivePage(md string) string {
	fm, body := "---\ntitle: Example\ntype: deck\n---\n\n", md
	if f, rest, ok := strings.Cut(md, "\n---\n"); ok && strings.TrimSpace(strings.SplitN(md, "\n", 2)[0]) == "---" {
		fm, body = f+"\n---\n", rest
		if !regexp.MustCompile(`(?m)^type:`).MatchString(f) {
			fm = "---\ntype: deck" + strings.TrimPrefix(f, "---") + "\n---\n"
		} else if !regexp.MustCompile(`(?m)^type:\s*deck\s*$`).MatchString(f) {
			return md // a document
		}
	}
	if !strings.HasPrefix(strings.TrimLeft(body, "\n"), "# ") {
		body = titleSlide + body
	}
	return fm + body
}

// newLiveExamples copies examples/ into a scratch folder; done removes it.
func newLiveExamples(write func(rel, text string) error) (*liveExamples, func(), error) {
	tmp, err := os.MkdirTemp("", "si-docsite-")
	if err != nil {
		return nil, nil, err
	}
	done := func() { os.RemoveAll(tmp) }
	err = fs.WalkDir(examples.FS, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		raw, err := fs.ReadFile(examples.FS, p)
		if err != nil {
			return err
		}
		to := filepath.Join(tmp, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		return os.WriteFile(to, raw, 0o644)
	})
	if err != nil {
		done()
		return nil, nil, err
	}
	return &liveExamples{write: write, tmp: tmp}, done, nil
}
