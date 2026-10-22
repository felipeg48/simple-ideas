package render

import (
	"html"
	"regexp"
	"strings"
	"sync"
)

// A tiny dependency-free highlighter: one regexp per language, one CSS class
// per token kind (.si-tk-c comment, s string, k keyword, n number, f flag,
// v variable, m command, y yaml key). Good enough for slides; unknown
// languages get strings and numbers only.

var keywords = map[string]string{
	"go": "break case chan const continue default defer else fallthrough for func go goto if import interface map " +
		"package range return select struct switch type var nil true false",
	"python": "def class return if elif else for while in not and or import from as with try except finally raise " +
		"lambda yield pass break continue True False None async await",
	"javascript": "const let var function return if else for while new class import export from async await true " +
		"false null undefined this of typeof",
	"java":   "class public private protected static final void int long boolean new return if else for while try catch throw throws import package extends implements interface record var true false null",
	"rust":   "fn let mut pub struct enum impl trait use mod match if else for while loop return self Self true false",
	"sql":    "select from where and or not in like between is null insert into values update set delete join on group by order limit as create table",
	"yaml":   "true false null yes no on off",
	"json":   "true false null",
	"bash":   "if then else fi for do done while case esac function in export local return",
	"css":    "",
	"html":   "",
	"toml":   "true false",
	"kotlin": "fun val var class object interface if else when for while return import package true false null",
}

var aliases = map[string]string{
	"js": "javascript", "ts": "javascript", "typescript": "javascript", "jsx": "javascript", "tsx": "javascript",
	"py": "python", "sh": "bash", "shell": "bash", "zsh": "bash", "console": "bash", "yml": "yaml", "golang": "go",
	"rs": "rust", "kt": "kotlin", "json5": "json",
}

var (
	hashComment  = map[string]bool{"bash": true, "python": true, "yaml": true, "toml": true}
	slashComment = map[string]bool{"go": true, "javascript": true, "java": true, "rust": true, "kotlin": true, "css": true, "json": true}
	hlCache      sync.Map
)

func langRE(lang string) *regexp.Regexp {
	if re, ok := hlCache.Load(lang); ok {
		return re.(*regexp.Regexp)
	}
	var parts []string
	switch {
	case hashComment[lang]:
		parts = append(parts, `(?P<c>#.*)`)
	case slashComment[lang]:
		parts = append(parts, `(?P<c>//.*|/\*[\s\S]*?\*/)`)
	case lang == "sql":
		parts = append(parts, `(?P<c>--.*)`)
	case lang == "html":
		parts = append(parts, `(?P<c><!--[\s\S]*?-->)`)
	}
	parts = append(parts, "(?P<s>\"(?:\\\\.|[^\"\\\\\\n])*\"|'(?:\\\\.|[^'\\\\\\n])*'|`[^`\\n]*`)")
	if lang == "bash" {
		parts = append(parts, `(?P<m>(?m:^[ \t]*(?:\$ )?[a-z][\w.-]*))`, `(?P<v>\$\{?[A-Za-z_]\w*\}?)`, `(?P<f>(?:^|[ \t])--?[A-Za-z][\w-]*)`)
	}
	if lang == "yaml" || lang == "toml" {
		parts = append(parts, `(?P<y>(?m:^[ \t]*-?[ \t]*[\w.\-/]+[ \t]*[:=]))`)
	}
	if kw := strings.Fields(keywords[lang]); len(kw) > 0 {
		parts = append(parts, `(?P<k>\b(?:`+strings.Join(kw, "|")+`)\b)`)
	}
	parts = append(parts, `(?P<n>\b\d[\w.]*\b)`)
	re := regexp.MustCompile(strings.Join(parts, "|"))
	hlCache.Store(lang, re)
	return re
}

func highlight(code, lang string) string {
	if a, ok := aliases[lang]; ok {
		lang = a
	}
	if lang == "" || lang == "text" || lang == "txt" || lang == "plain" || lang == "output" {
		return html.EscapeString(code)
	}
	re := langRE(lang)
	names := re.SubexpNames()
	var b strings.Builder
	pos := 0
	for _, m := range re.FindAllStringSubmatchIndex(code, -1) {
		kind := ""
		for g := 1; g < len(names); g++ {
			if m[2*g] >= 0 {
				kind = names[g]
				break
			}
		}
		b.WriteString(html.EscapeString(code[pos:m[0]]))
		b.WriteString(`<span class="si-tk-` + kind + `">` + html.EscapeString(code[m[0]:m[1]]) + `</span>`)
		pos = m[1]
	}
	b.WriteString(html.EscapeString(code[pos:]))
	return b.String()
}
