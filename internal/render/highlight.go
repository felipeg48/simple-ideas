package render

import (
	"html"
	"regexp"
	"strings"
	"sync"
)

// A tiny dependency-free highlighter: one regexp per language, one CSS class
// per token kind (.si-tk-c comment, s string, k keyword, n number, f a call's
// name or a flag, t a type (PascalCase), v variable, m command, y key).
// Good enough for slides; unknown languages get strings, calls and numbers.

var keywords = map[string]string{
	"go": "break case chan const continue default defer else fallthrough for func go goto if import interface map " +
		"package range return select struct switch type var nil true false",
	"python": "def class return if elif else for while in not and or import from as with try except finally raise " +
		"lambda yield pass break continue True False None async await",
	"javascript": "const let var function return if else for while new class import export from async await true " +
		"false null undefined this of typeof",
	"java": "class public private protected static final void int long boolean new return if else for while try catch throw throws import package extends implements interface record var true false null",
	"rust": "fn let mut pub struct enum impl trait use mod match if else for while loop return self Self true false",
	"sql": "select distinct from where and or not in like between is null insert into values update set delete " +
		"join inner left right outer full cross on using group by having order limit offset as create table alter " +
		"drop truncate add column constraint primary key foreign references unique default check cascade index view " +
		"database schema sequence show explain analyze begin commit rollback savepoint release cluster setting role " +
		"user grant revoke to with recursive union all exists case when then else end asc desc returning upsert",
	"yaml":   "true false null yes no on off",
	"json":   "true false null",
	"bash":   "if then else fi for do done while case esac function in export local return",
	"css":    "",
	"html":   "",
	"toml":   "true false",
	"kotlin": "fun val var class object interface if else when for while return import package true false null",
	"csharp": "abstract as async await base bool break byte case catch char checked class const continue decimal default " +
		"delegate do double else enum event explicit extern false finally fixed float for foreach get goto if implicit in init " +
		"int interface internal is lock long namespace new null object operator out override params partial private protected " +
		"public readonly record ref return sbyte sealed set short sizeof stackalloc static string struct switch this throw true " +
		"try typeof uint ulong unchecked unsafe ushort using var virtual void volatile when where while yield",
	"c": "auto break case char const continue default do double else enum extern float for goto if inline int long register " +
		"return short signed sizeof static struct switch typedef union unsigned void volatile while NULL true false",
	"cpp": "auto bool break case catch char class const constexpr continue default delete do double else enum explicit extern " +
		"false float for friend if inline int long namespace new noexcept nullptr operator override private protected public " +
		"return short signed sizeof static struct switch template this throw true try typedef typename union unsigned using " +
		"virtual void volatile while",
	"php": "abstract and array as break case catch class const continue declare default do echo else elseif extends final " +
		"fn for foreach function global if implements interface match namespace new null or private protected public readonly " +
		"return static switch throw trait true false try use var while yield",
	"ruby": "begin def class module end if elsif else unless case when while until for in do return yield self nil true " +
		"false and or not require include extend attr_accessor puts",
	"swift": "let var func class struct enum protocol extension import return if else guard for while in switch case " +
		"default break continue true false nil self init deinit throws throw try catch async await public private static",
	"scala": "def val var class object trait extends with case match if else for while yield return import package new " +
		"true false null this",
	"powershell": "function param if elseif else foreach for while do switch return try catch finally throw begin process end",
	"dockerfile": "from run cmd label expose env add copy entrypoint volume user workdir arg onbuild stopsignal healthcheck shell as",
	"xml":        "",
	"hcl": "resource data variable output locals module provider terraform backend required_providers for_each count " +
		"depends_on lifecycle dynamic content for in if true false null",
	"protobuf": "syntax package import option message enum service rpc returns stream repeated optional required map oneof " +
		"reserved extend double float int32 int64 uint32 uint64 sint32 sint64 fixed32 fixed64 sfixed32 sfixed64 bool string bytes true false",
	"graphql": "query mutation subscription fragment on type interface union enum input scalar schema extend directive " +
		"implements true false null",
	"fsharp": "let mutable rec and if then else elif match with function fun type of module namespace open member override " +
		"abstract interface inherit new yield return async task do done for to downto while in try finally raise not true false " +
		"null use when as begin end struct val static private public internal",
	"elixir": "def defp defmodule defstruct defmacro defprotocol defimpl do end fn if else unless case cond with when in " +
		"import alias require use quote unquote receive after try catch rescue raise true false nil and or not",
}

var aliases = map[string]string{
	"js": "javascript", "ts": "javascript", "typescript": "javascript", "jsx": "javascript", "tsx": "javascript",
	"py": "python", "sh": "bash", "shell": "bash", "zsh": "bash", "console": "bash", "yml": "yaml", "golang": "go",
	"rs": "rust", "kt": "kotlin", "json5": "json",
	"cs": "csharp", "c#": "csharp", "dotnet": "csharp", "h": "c", "c++": "cpp", "cc": "cpp", "hpp": "cpp", "rb": "ruby",
	"ps1": "powershell", "pwsh": "powershell", "docker": "dockerfile", "containerfile": "dockerfile",
	"csproj": "xml", "fsproj": "xml", "vbproj": "xml", "props": "xml", "targets": "xml", "nuspec": "xml", "resx": "xml",
	"xaml": "xml", "svg": "xml", "plist": "xml", "pom": "xml", "xsd": "xml", "xsl": "xml", "xslt": "xml", "wsdl": "xml",
	"htm": "html", "xhtml": "html", "vue": "html",
	"tf": "hcl", "terraform": "hcl", "tfvars": "hcl", "proto": "protobuf", "proto3": "protobuf", "gql": "graphql",
	"fs": "fsharp", "f#": "fsharp", "fsx": "fsharp", "ex": "elixir", "exs": "elixir",
}

var (
	hashComment  = map[string]bool{"bash": true, "python": true, "yaml": true, "toml": true, "ruby": true, "powershell": true, "dockerfile": true, "graphql": true, "elixir": true}
	slashComment = map[string]bool{"go": true, "javascript": true, "java": true, "rust": true, "kotlin": true, "css": true, "json": true,
		"csharp": true, "c": true, "cpp": true, "php": true, "swift": true, "scala": true, "protobuf": true}
	markup  = map[string]bool{"html": true, "xml": true} // tags as keywords, attribute names as keys
	noCalls = map[string]bool{"fsharp": true, "bash": true, "yaml": true, "toml": true, "json": true, "css": true, "dockerfile": true, "powershell": true}
	typed   = map[string]bool{"csharp": true, "java": true, "go": true, "kotlin": true, "swift": true, "scala": true,
		"javascript": true, "cpp": true, "rust": true, "php": true, "protobuf": true, "graphql": true, "fsharp": true, "elixir": true}
	hlCache sync.Map
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
	case lang == "hcl":
		parts = append(parts, `(?P<c>#.*|//.*|/\*[\s\S]*?\*/)`)
	case lang == "fsharp":
		parts = append(parts, `(?P<c>//.*|\(\*[\s\S]*?\*\))`)
	case markup[lang]:
		parts = append(parts, `(?P<c><!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>)`, `(?P<m><\?[\s\S]*?\?>|<![A-Za-z][^>]*>)`)
	}
	parts = append(parts, "(?P<s>\"(?:\\\\.|[^\"\\\\\\n])*\"|'(?:\\\\.|[^'\\\\\\n])*'|`[^`\\n]*`)")
	if lang == "bash" {
		parts = append(parts, `(?P<m>(?m:^[ \t]*(?:\$ )?[a-z][\w.-]*))`, `(?P<v>\$\{?[A-Za-z_]\w*\}?)`, `(?P<f>(?:^|[ \t])--?[A-Za-z][\w-]*)`)
	}
	if markup[lang] {
		parts = append(parts, `(?P<k></?[A-Za-z][\w:.-]*|/?>)`, `(?P<y>[A-Za-z_:][\w:.-]*=)`)
	}
	if lang == "hcl" { // attribute = value
		parts = append(parts, `(?P<y>(?m:^[ \t]*[\w-]+[ \t]*=))`)
	}
	if lang == "graphql" {
		parts = append(parts, `(?P<v>\$[A-Za-z_]\w*)`)
	}
	if lang == "elixir" { // atoms
		parts = append(parts, `(?P<v>:[a-z_]\w*[?!]?)`)
	}
	if lang == "yaml" || lang == "toml" {
		parts = append(parts, `(?P<y>(?m:^[ \t]*-?[ \t]*[\w.\-/]+[ \t]*[:=]))`)
	}
	if kw := strings.Fields(keywords[lang]); len(kw) > 0 {
		group := `\b(?:` + strings.Join(kw, "|") + `)\b`
		if lang == "sql" || lang == "dockerfile" {
			// SQL keywords are case-insensitive; most CockroachDB SQL is UPPERCASE.
			group = `(?i:` + group + `)`
		}
		parts = append(parts, `(?P<k>`+group+`)`)
	}
	if !noCalls[lang] && !markup[lang] { // a name followed by ( is a call: the name only
		parts = append(parts, `(?P<f>\b[A-Za-z_]\w*)\s*\(`)
	}
	if typed[lang] { // PascalCase: a type or a class
		parts = append(parts, `(?P<t>\b[A-Z][A-Za-z0-9]*[a-z][A-Za-z0-9]*\b)`)
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
		kind, from, to := "", m[0], m[1]
		for g := 1; g < len(names); g++ {
			if m[2*g] >= 0 {
				kind, from, to = names[g], m[2*g], m[2*g+1] // only the group: a call's ( stays plain
				break
			}
		}
		b.WriteString(html.EscapeString(code[pos:from]))
		b.WriteString(`<span class="si-tk-` + kind + `">` + html.EscapeString(code[from:to]) + `</span>`)
		pos = to
	}
	b.WriteString(html.EscapeString(code[pos:]))
	return b.String()
}
