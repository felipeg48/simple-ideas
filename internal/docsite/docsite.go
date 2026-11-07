// Package docsite writes the si docs site (si --export docs-site): a site
// whose Rules section is generated from RULES.md, one page per numbered
// section, so the site cannot drift from the rules; the manual and the
// examples ride along, and the home page runs live examples.
package docsite

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/examples"
)

// Section is one numbered section of RULES.md ("## 13. Code, …").
type Section struct {
	N     string // "13"
	Title string // "Code, Mermaid, stage diagrams, tables, links"
	Slug  string // "13-code-mermaid-stage-diagrams-tables-links"
	Body  string // the Markdown under the heading
}

var (
	secRE  = regexp.MustCompile(`^## (\d+)\. (.+)$`)
	refRE  = regexp.MustCompile(`§(\d+)\b`)
	anchorRE = regexp.MustCompile(`\]\(#(\d+)-[^)]*\)`)
	fenceR = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
)

// fenceStep follows CommonMark fences: an opening fence (a backtick fence's
// info has no backtick, so "```` ```x ```` text" is prose) and a closing fence
// of the same character, at least as long, with nothing after it.
func fenceStep(open, line string) string {
	m := fenceR.FindStringSubmatch(line)
	if m == nil {
		return open
	}
	if open == "" {
		if m[1][0] == '`' && strings.Contains(m[2], "`") {
			return ""
		}
		return m[1]
	}
	if m[1][0] == open[0] && len(m[1]) >= len(open) && strings.TrimSpace(m[2]) == "" {
		return ""
	}
	return open
}

// Split cuts RULES.md into its intro (before section 1) and its numbered
// sections. Headings inside fenced code (example slides) are not sections.
func Split(rules string) (intro string, secs []Section) {
	var cur *Section
	var b strings.Builder
	fence := ""
	flush := func() {
		if cur == nil {
			intro = b.String()
		} else {
			cur.Body = strings.TrimSpace(b.String())
			secs = append(secs, *cur)
		}
		b.Reset()
	}
	for _, line := range strings.Split(rules, "\n") {
		fence = fenceStep(fence, line)
		if fence == "" {
			if m := secRE.FindStringSubmatch(line); m != nil {
				flush()
				cur = &Section{N: m[1], Title: strings.TrimSpace(m[2])}
				cur.Slug = fmt.Sprintf("%02s-%s", m[1], slug(cur.Title))
				continue
			}
		}
		b.WriteString(line + "\n")
	}
	flush()
	return strings.TrimSpace(intro), secs
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// linkRefs turns "§13" outside code into a link to that section's page.
func linkRefs(body string, pages map[string]string) string {
	var out []string
	fence := ""
	for _, line := range strings.Split(body, "\n") {
		was := fence
		if fence = fenceStep(fence, line); fence != "" || was != "" {
			out = append(out, line)
			continue
		}
		if fence == "" {
			parts := strings.Split(line, "`") // leave inline code alone
			for i := 0; i < len(parts); i += 2 {
				parts[i] = anchorRE.ReplaceAllStringFunc(parts[i], func(m string) string { // RULES.md's own contents links
					if p, ok := pages[anchorRE.FindStringSubmatch(m)[1]]; ok {
						return "](" + p + ".md)"
					}
					return m
				})
				parts[i] = refRE.ReplaceAllStringFunc(parts[i], func(m string) string {
					if p, ok := pages[m[len("§"):]]; ok {
						return "[" + m + "](" + p + ".md)"
					}
					return m
				})
			}
			line = strings.Join(parts, "`")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// Generate writes the docs site source into dst (which must be empty or new).
func Generate(dst string) (pages int, err error) {
	if entries, err := os.ReadDir(dst); err == nil && len(entries) > 0 {
		return 0, fmt.Errorf("%s exists and is not empty", dst)
	}
	write := func(rel, text string) error {
		p := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte(text), 0o644)
	}
	copyFS := func(fsys fs.FS, root, to string, skip func(string) bool) error {
		return fs.WalkDir(fsys, root, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || skip(p) {
				return err
			}
			raw, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
			return write(filepath.ToSlash(filepath.Join(to, rel)), string(raw))
		})
	}
	if err := copyFS(docs.Site, "site", "", func(string) bool { return false }); err != nil {
		return 0, err
	}
	// the files the home page's live deck needs, in a folder the site does not list
	keep := func(p string) bool {
		top, _, _ := strings.Cut(p, "/")
		return p == "showcase.md" || p == "appendix.md" || p == "runbook.json" || p == "si.conf" || top == "assets" || top == "css" || top == "demos"
	}
	if err := copyFS(examples.FS, ".", "_examples", func(p string) bool { return !keep(p) }); err != nil {
		return 0, err
	}
	if err := write("manual.md", strings.Replace(docs.Manual, "---\n", "---\ntype: deck\n", 1)); err != nil {
		return 0, err
	}
	intro, secs := Split(docs.Rules)
	refs := map[string]string{}
	for _, s := range secs {
		refs[s.N] = s.Slug
	}
	var nav strings.Builder
	nav.WriteString("---\ntitle: The rules\ndescription: Every rule of the si Markdown format, one page per section, each with an example.\n---\n\n- [Overview](index.md)\n")
	for _, s := range secs {
		fmt.Fprintf(&nav, "- [%s. %s](%s.md)\n", s.N, s.Title, s.Slug)
		page := fmt.Sprintf("---\ntitle: %s. %s\n---\n\n# %s. %s\n\n%s\n", s.N, yamlSafe(s.Title), s.N, s.Title, linkRefs(s.Body, refs))
		if err := write("rules/"+s.Slug+".md", page); err != nil {
			return 0, err
		}
	}
	if err := write("rules/_nav.md", nav.String()); err != nil {
		return 0, err
	}
	overview := "---\ntitle: The rules\n---\n\n# The rules\n\n" + linkRefs(stripH1(intro), refs) + "\n\n## Sections\n\n"
	for _, s := range secs {
		overview += fmt.Sprintf("- [%s. %s](%s.md)\n", s.N, s.Title, s.Slug)
	}
	if err := write("rules/index.md", overview); err != nil {
		return 0, err
	}
	return len(secs) + 1, nil
}

func stripH1(s string) string {
	if first, rest, ok := strings.Cut(s, "\n"); ok && strings.HasPrefix(first, "# ") {
		return strings.TrimSpace(rest)
	}
	return s
}

// yamlSafe keeps a title on one front matter line (si's front matter is
// key: value; quotes would show).
func yamlSafe(s string) string { return strings.ReplaceAll(s, "\n", " ") }
