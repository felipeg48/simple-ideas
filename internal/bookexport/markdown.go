package bookexport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteMarkdown writes the book as plain CommonMark with GitHub tables into
// dir: <whole>.md (the whole book, a pandoc title block first), one <page>.md
// per chapter, and images/ with every picture they show. It returns the files
// written, the whole book first.
func WriteMarkdown(b *Book, dir, whole string) ([]string, error) {
	order, name := names(b)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if len(order) > 0 {
		if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
			return nil, err
		}
	}
	for _, src := range order {
		data, err := os.ReadFile(src)
		if err != nil {
			continue // a missing picture: si check reports it; the link stays
		}
		if err := os.WriteFile(filepath.Join(dir, "images", name[src]), data, 0o644); err != nil {
			return nil, err
		}
	}
	var files []string
	write := func(file, text string) error {
		p := filepath.Join(dir, file)
		files = append(files, p)
		return os.WriteFile(p, []byte(text), 0o644)
	}
	if err := write(whole+".md", Markdown(b, name)); err != nil {
		return nil, err
	}
	used := map[string]bool{whole + ".md": true}
	for _, c := range b.Chapters {
		if len(b.Chapters) == 1 {
			break // one article: the whole is the chapter
		}
		file := c.Name + ".md"
		for i := 2; used[file]; i++ {
			file = fmt.Sprintf("%s-%d.md", c.Name, i)
		}
		used[file] = true
		if err := write(file, chapterMarkdown(c, name, 1)); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// Markdown is the whole book as one Markdown text; name maps image files to
// their names under images/ (nil keeps the paths as they are).
func Markdown(b *Book, name map[string]string) string {
	var w strings.Builder
	w.WriteString("---\n")
	for _, kv := range [][2]string{{"title", b.Title}, {"subtitle", b.Subtitle}, {"author", b.Author}, {"date", b.Date}, {"rights", b.Rights()}, {"lang", b.Lang}} {
		if kv[1] != "" {
			fmt.Fprintf(&w, "%s: %q\n", kv[0], kv[1])
		}
	}
	w.WriteString("---\n\n")
	if len(b.Colophon) > 0 {
		w.WriteString(mdBlocks(b.Colophon, name))
	}
	for _, c := range b.Chapters {
		if c.Part != "" {
			fmt.Fprintf(&w, "# %s\n\n", mdText(c.Part))
		}
		w.WriteString(chapterMarkdown(c, name, 1))
	}
	return strings.TrimRight(w.String(), "\n") + "\n"
}

func chapterMarkdown(c Chapter, name map[string]string, level int) string {
	return strings.Repeat("#", level) + " " + mdText(c.Heading()) + "\n\n" + mdBlocks(c.Blocks, name)
}

func mdBlocks(bs []Block, name map[string]string) string {
	var w strings.Builder
	for _, b := range bs {
		w.WriteString(mdBlock(b, name))
		w.WriteString("\n")
	}
	return w.String()
}

func mdBlock(b Block, name map[string]string) string {
	switch b.Kind {
	case Heading:
		t := mdRuns(b.Runs)
		if b.Num != "" {
			t = b.Num + " " + t
		}
		return strings.Repeat("#", b.Level) + " " + t + "\n"
	case Para:
		return mdRuns(b.Runs) + "\n"
	case Code:
		out := fence(CodeLines(b), b.Lang)
		if len(b.Callouts) > 0 {
			out += "\n"
			for _, co := range b.Callouts {
				out += co.N + ". " + mdRuns(co.Runs) + "\n"
			}
		}
		return out
	case Placeholder:
		if b.Lang == "mermaid" { // GitHub and many editors draw it
			return fence(b.Text, b.Lang)
		}
		return mdRuns(b.Runs) + "\n"
	case List:
		var w strings.Builder
		for i, it := range b.Items {
			marker := "- "
			if b.Ordered {
				marker = fmt.Sprintf("%d. ", b.Start+i)
			}
			var parts []string
			for j, k := range it {
				if j > 0 && k.Kind != List { // a paragraph after the first: a loose item
					parts = append(parts, "")
				}
				parts = append(parts, strings.TrimRight(mdBlock(k, name), "\n"))
			}
			body := strings.Join(parts, "\n")
			pad := strings.Repeat(" ", len(marker))
			lines := strings.Split(body, "\n")
			for j, l := range lines {
				switch {
				case j == 0:
					w.WriteString(marker + l + "\n")
				case l == "":
					w.WriteString("\n")
				default:
					w.WriteString(pad + l + "\n")
				}
			}
		}
		return w.String()
	case Table:
		var w strings.Builder
		cols := 0
		for _, r := range b.Rows {
			cols = max(cols, len(r))
		}
		for i, r := range b.Rows {
			w.WriteString("|")
			for j := 0; j < cols; j++ {
				cell := ""
				if j < len(r) {
					cell = strings.ReplaceAll(strings.ReplaceAll(mdRuns(r[j]), "|", `\|`), "\n", " ")
				}
				w.WriteString(" " + cell + " |")
			}
			w.WriteString("\n")
			if i == 0 {
				w.WriteString("|" + strings.Repeat("---|", cols) + "\n")
			}
		}
		if b.Caption != "" {
			w.WriteString("\n*" + mdText(b.Caption) + "*\n")
		}
		return w.String()
	case Image:
		src := filepath.ToSlash(b.Src)
		if n := name[b.Src]; n != "" {
			src = "images/" + n
		}
		alt := b.Alt
		if alt == "" {
			alt = b.Caption
		}
		out := fmt.Sprintf("![%s](%s)\n", mdText(alt), strings.ReplaceAll(src, " ", "%20"))
		if b.Caption != "" {
			out += "\n*" + mdText(b.Caption) + "*\n"
		}
		return out
	case Admonition, Quote:
		inner := strings.TrimRight(mdBlocks(b.Blocks, name), "\n")
		if b.Kind == Admonition {
			if len(b.Blocks) > 0 && b.Blocks[0].Kind == Para {
				inner = "**" + b.Label + ":** " + inner
			} else {
				inner = "**" + b.Label + ":**\n\n" + inner
			}
		}
		var w strings.Builder
		for _, l := range strings.Split(inner, "\n") {
			if l == "" {
				w.WriteString(">\n")
			} else {
				w.WriteString("> " + l + "\n")
			}
		}
		return w.String()
	case Rule:
		return "---\n"
	}
	return ""
}

func fence(code, lang string) string {
	ticks := "```"
	for strings.Contains(code, ticks) {
		ticks += "`"
	}
	return ticks + lang + "\n" + code + "\n" + ticks + "\n"
}

// mdRuns writes runs as Markdown inline text.
func mdRuns(runs []Run) string {
	var w strings.Builder
	for _, r := range runs {
		if r.Text == "\n" {
			w.WriteString("\\\n")
			continue
		}
		t := r.Text
		if r.Code {
			ticks := "`"
			for strings.Contains(t, ticks) {
				ticks += "`"
			}
			pad := ""
			if strings.HasPrefix(t, "`") || strings.HasSuffix(t, "`") {
				pad = " "
			}
			t = ticks + pad + t + pad + ticks
		} else {
			t = mdText(t)
		}
		lead, core, trail := splitSpace(t)
		if core == "" {
			w.WriteString(t)
			continue
		}
		if r.Italic {
			core = "*" + core + "*"
		}
		if r.Bold {
			core = "**" + core + "**"
		}
		if r.Link != "" {
			core = "[" + core + "](" + r.Link + ")"
		}
		w.WriteString(lead + core + trail)
	}
	return w.String()
}

func splitSpace(t string) (lead, core, trail string) {
	core = strings.TrimLeft(t, " ")
	lead = t[:len(t)-len(core)]
	c2 := strings.TrimRight(core, " ")
	return lead, c2, core[len(c2):]
}

// mdText escapes the characters Markdown would read as markup.
func mdText(t string) string {
	var w strings.Builder
	rs := []rune(t)
	for i, r := range rs {
		switch r {
		case '\\', '`', '*', '[', ']', '<':
			w.WriteRune('\\')
		case '_':
			if i > 0 && i < len(rs)-1 && isWord(rs[i-1]) && isWord(rs[i+1]) {
				break // snake_case stays as it is: GitHub never reads it as emphasis
			}
			w.WriteRune('\\')
		case '#':
			if i == 0 {
				w.WriteRune('\\')
			}
		}
		w.WriteRune(r)
	}
	return w.String()
}

func isWord(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 127
}
