package deck

import (
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Code includes: `<!-- include: src/App.java lines=5-10 -->` turns a code file
// (anything that is not .md / .adoc) into a fenced code block, so the rest of
// the pipeline (highlighter, copy, callouts, tabs) treats it like any fence.

var includeOptRE = regexp.MustCompile(`(\w+)(?:="([^"]*)"|=(\S+))?`)

// splitIncludeValue splits an include value into the path and its options
// (key=value, key="a value", or a bare key).
func splitIncludeValue(val string) (string, map[string]string) {
	path, rest, _ := strings.Cut(strings.TrimSpace(val), " ")
	opts := map[string]string{}
	for _, m := range includeOptRE.FindAllStringSubmatch(rest, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		opts[m[1]] = v
	}
	return path, opts
}

// extLang maps a file extension to the fence language.
var extLang = map[string]string{
	".java": "java", ".kt": "kotlin", ".py": "python", ".go": "go", ".js": "javascript", ".mjs": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".jsx": "javascript", ".rb": "ruby", ".rs": "rust", ".c": "c", ".h": "c",
	".cpp": "cpp", ".cc": "cpp", ".cs": "csharp", ".php": "php", ".swift": "swift", ".scala": "scala",
	".sh": "bash", ".bash": "bash", ".zsh": "bash", ".ps1": "powershell", ".sql": "sql",
	".yml": "yaml", ".yaml": "yaml", ".json": "json", ".toml": "toml", ".xml": "xml", ".html": "html", ".css": "css",
	".tf": "hcl", ".hcl": "hcl", ".proto": "protobuf", ".graphql": "graphql", ".conf": "text", ".txt": "text",
}

var tagMarkRE = regexp.MustCompile(`\b(tag|end)::([\w.-]+)\[\]`)

// fileLang is the fence language of a code file: its name (codeshow.LangOf:
// .csproj and .config are XML, Dockerfile is dockerfile), the extensions only
// includes know, then a look at the content (<?xml or a tag, JSON, a #! line).
func fileLang(file string, raw []byte) string {
	if l := codeshow.LangOf(filepath.Base(file)); l != "" {
		return l
	}
	if l := extLang[strings.ToLower(filepath.Ext(file))]; l != "" && l != "text" {
		return l
	}
	s := strings.TrimSpace(string(raw[:min(len(raw), 512)]))
	switch {
	case strings.HasPrefix(s, "<?xml"), xmlStartRE.MatchString(s):
		return "xml"
	case (strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")) && jsonStartRE.MatchString(s):
		return "json"
	case strings.HasPrefix(s, "#!"):
		first, _, _ := strings.Cut(s, "\n")
		switch {
		case strings.Contains(first, "python"):
			return "python"
		case strings.Contains(first, "node"):
			return "javascript"
		case strings.Contains(first, "sh"):
			return "bash"
		}
	}
	return extLang[strings.ToLower(filepath.Ext(file))]
}

var (
	xmlStartRE  = regexp.MustCompile(`^<[A-Za-z][\w:.-]*[\s>/]`)
	jsonStartRE = regexp.MustCompile(`^[\[{]\s*(?:"|\{|\[|\]|\}|-?\d|true|false|null)`)
)

// codeInclude reads file and returns the fence lines of a code block, after
// applying the options. Options it does not know are errors, so a typo is
// caught by si check.
func codeInclude(file string, opts map[string]string) ([]string, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("file not found")
	}
	return codeBlock(raw, file, opts, "")
}

// codeBlock turns file content (named name, for its extension and default
// title) into fence lines; src, when set, is the source link of the block.
func codeBlock(raw []byte, file string, opts map[string]string, src string) ([]string, error) {
	for k := range opts {
		switch k {
		case "lines", "tag", "lang", "title", "hl", "linenos", "indent":
		default:
			return nil, fmt.Errorf("unknown option %q (lines, tag, lang, title, hl, linenos, indent=keep)", k)
		}
	}
	all := strings.Split(strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n"), "\n")
	type ln struct {
		no   int
		text string
	}
	var keep []ln
	switch {
	case opts["tag"] != "" && opts["lines"] != "":
		return nil, fmt.Errorf("use tag= or lines=, not both")
	case opts["tag"] != "":
		want := map[string]bool{}
		for _, t := range strings.Split(opts["tag"], ",") {
			want[strings.TrimSpace(t)] = true
		}
		open, found, closed := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for i, t := range all {
			if ms := tagMarkRE.FindAllStringSubmatch(t, -1); ms != nil {
				for _, m := range ms {
					if m[1] == "tag" {
						open[m[2]], found[m[2]] = true, true
					} else {
						open[m[2]], closed[m[2]] = false, true
					}
				}
				continue // marker lines are never shown
			}
			for name := range want {
				if open[name] {
					keep = append(keep, ln{i + 1, t})
					break
				}
			}
		}
		for name := range want {
			switch {
			case !found[name]:
				return nil, fmt.Errorf("tag %q not found (mark it with tag::%s[] ... end::%s[] in a comment)", name, name, name)
			case !closed[name]:
				return nil, fmt.Errorf("tag %q is not closed (end::%s[])", name, name)
			}
		}
	case opts["lines"] != "":
		for _, r := range strings.Split(opts["lines"], ",") {
			a, b, _ := strings.Cut(strings.TrimSpace(r), "-")
			from, to := 1, len(all)
			var err error
			if a != "" {
				if from, err = strconv.Atoi(a); err != nil {
					return nil, fmt.Errorf("lines=%s: want ranges like 5-10, 5-, -10 or 5-10,20-24", opts["lines"])
				}
			}
			if strings.Contains(r, "-") && b != "" {
				if to, err = strconv.Atoi(b); err != nil {
					return nil, fmt.Errorf("lines=%s: want ranges like 5-10, 5-, -10 or 5-10,20-24", opts["lines"])
				}
			} else if !strings.Contains(r, "-") {
				to = from
			}
			if from < 1 || to > len(all) || from > to {
				return nil, fmt.Errorf("lines=%s: the file has %d lines", r, len(all))
			}
			for i := from; i <= to; i++ {
				keep = append(keep, ln{i, all[i-1]})
			}
		}
	default:
		for i, t := range all {
			keep = append(keep, ln{i + 1, t})
		}
	}
	if len(keep) == 0 {
		return nil, fmt.Errorf("nothing to show")
	}
	if opts["indent"] != "keep" { // drop the indentation every shown line shares
		common := -1
		for _, k := range keep {
			if strings.TrimSpace(k.text) == "" {
				continue
			}
			n := len(k.text) - len(strings.TrimLeft(k.text, " \t"))
			if common < 0 || n < common {
				common = n
			}
		}
		for i := range keep {
			if len(keep[i].text) >= common && common > 0 {
				keep[i].text = keep[i].text[common:]
			}
		}
	}
	lang := opts["lang"]
	if lang == "" {
		lang = fileLang(file, raw)
	}
	title := opts["title"]
	if title == "" {
		title = filepath.Base(file)
	}
	fence, body := "```", make([]string, 0, len(keep))
	for _, k := range keep {
		for strings.Contains(k.text, fence) {
			fence += "`"
		}
		body = append(body, k.text)
	}
	info := fmt.Sprintf("%s%s title=%q", fence, lang, title)
	if opts["hl"] != "" {
		info += fmt.Sprintf(" hl=%q", opts["hl"])
	}
	_, ranged := opts["linenos"]
	if opts["lines"] != "" || opts["tag"] != "" || ranged {
		// numbered as in the file; ponytail: lines=5-10,20-24 numbers on from 5, add per-line numbers if mixed ranges matter
		info += fmt.Sprintf(" linenos start=%d", keep[0].no)
	}
	if src != "" {
		info += fmt.Sprintf(" src=%q", src)
	}
	return append(append([]string{info}, body...), fence), nil
}
