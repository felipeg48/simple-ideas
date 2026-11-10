// Package themefile reads and writes si theme files: a CSS file of --si-*
// tokens with an optional header naming the theme and the built-in it
// extends. Parse finds the token blocks (:root, the dark-mode rule and its
// prefers-color-scheme twin); String writes them back in one layout, with
// any other CSS in the file kept verbatim, like the mind map model keeps
// unknown text. Standard library only.
package themefile

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// File is a theme file.
type File struct {
	Name    string // from the header: "si theme: Name · extends base"
	Extends string // a built-in theme name, or ""
	// HeaderRest is the header comment after its first line, up to "*/".
	HeaderRest string
	// Light holds :root values (both modes, unless Dark changes them).
	Light map[string]string
	// Dark holds dark-mode values.
	Dark map[string]string
	// Preamble is the CSS before the first token block (comments, mostly);
	// Extra is every other rule, in order. Both are kept verbatim.
	Preamble, Extra string
}

// New returns an empty theme named name that extends base ("" for none).
func New(name, base string) *File {
	return &File{Name: name, Extends: base, Light: map[string]string{}, Dark: map[string]string{}}
}

// Builtin returns a built-in theme as a full theme file (every token it
// sets), the starting point `si theme export <name>` writes.
func Builtin(name string) (*File, bool) {
	light, dark, ok := assets.ThemeTokens(name)
	if !ok {
		return nil, false
	}
	f := New(assets.Themes[assets.CanonicalTheme(name)].Label, "")
	f.Light, f.Dark = light, dark
	return f, true
}

// Selectors of the token blocks, the same ones assets.ThemeCSS emits.
const (
	selLight     = ":root"
	selDark      = `:root[data-si-mode="dark"]`
	selDarkMedia = "@media (prefers-color-scheme: dark)"
	selDarkAuto  = ":root:not([data-si-mode])"
)

var headerRE = regexp.MustCompile(`^/\*\s*si theme:([^\n]*?)(\*/|\n)`)

// Parse reads a theme file. It never fails: CSS it does not recognise as a
// token block is kept in Preamble or Extra.
func Parse(src string) *File {
	f := New("", "")
	rest := src
	if m := headerRE.FindStringSubmatchIndex(strings.TrimLeft(src, " \t\r\n")); m != nil {
		s := strings.TrimLeft(src, " \t\r\n")
		f.Name, f.Extends = parseHeaderLine(s[m[2]:m[3]])
		end := strings.Index(s, "*/")
		if s[m[4]:m[5]] == "\n" && end >= 0 {
			f.HeaderRest = s[m[4]:end]
		}
		if end < 0 {
			end = len(s) - 2
		}
		rest = strings.TrimLeft(s[end+2:], "\n")
	}
	var pre, extra strings.Builder
	seen := false
	var walk func(items []item)
	walk = func(items []item) {
		for _, it := range items {
			if it.body != "" || it.prelude != "" {
				p := norm(it.prelude)
				if p == "@layersi.theme" { // a file written with its own layer wrapper
					walk(scan(it.body))
					continue
				}
				if f.absorb(p, it.body) {
					seen = true
					continue
				}
			}
			if !seen {
				pre.WriteString(it.text)
			} else {
				extra.WriteString(it.text)
			}
		}
	}
	walk(scan(rest))
	f.Preamble = strings.TrimSpace(pre.String())
	f.Extra = strings.TrimSpace(extra.String())
	if !seen { // nothing recognised: all of it is extra, after the (empty) tokens
		f.Extra, f.Preamble = f.Preamble, ""
	}
	return f
}

func parseHeaderLine(l string) (name, base string) {
	l = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l), "*/"))
	if i := strings.Index(l, "extends "); i >= 0 {
		if w := strings.Fields(strings.TrimSuffix(l[i+len("extends "):], "*/")); len(w) > 0 {
			base = w[0]
		}
		l = l[:i]
	}
	name = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(l), "·|-"))
	return name, base
}

// absorb takes a rule's tokens into the file when its selector is one of
// the token blocks and its body holds only custom properties.
func (f *File) absorb(prelude, body string) bool {
	var into map[string]string
	switch prelude {
	case selLight:
		into = f.Light
	case norm(selDark):
		into = f.Dark
	case norm(selDarkMedia):
		in := scan(body)
		n := 0
		for _, it := range in {
			if strings.TrimSpace(it.text) == "" {
				continue
			}
			n++
			if p := norm(it.prelude); n > 1 || (p != norm(selDarkAuto) && p != norm(`:root:not([data-si-mode="light"])`)) {
				return false
			}
			body = it.body
		}
		if n != 1 {
			return false
		}
		d, ok := decls(body)
		if !ok {
			return false
		}
		for k, v := range d {
			if _, set := f.Dark[k]; !set { // the explicit dark rule wins over its twin
				f.Dark[k] = v
			}
		}
		return true
	default:
		return false
	}
	d, ok := decls(body)
	if !ok {
		return false
	}
	for k, v := range d {
		into[k] = v
	}
	return true
}

// decls splits a block body into custom properties; ok is false when it
// holds anything else (a comment, a normal property).
func decls(body string) (map[string]string, bool) {
	out := map[string]string{}
	for _, d := range split(body, ';') {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		i := strings.Index(d, ":")
		if i < 0 || !strings.HasPrefix(d, "--") || strings.Contains(d, "/*") {
			return nil, false
		}
		out[strings.TrimSpace(d[:i])] = strings.TrimSpace(d[i+1:])
	}
	return out, true
}

func norm(s string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(s), ""), "'", `"`)
}

// String writes the theme: the header, the preamble, the token blocks
// (tokens in contract order, then any others by name) and the extra CSS.
func (f *File) String() string {
	var b strings.Builder
	if f.Name != "" || f.Extends != "" {
		b.WriteString("/* si theme: " + f.Name)
		if f.Extends != "" {
			b.WriteString(" · extends " + f.Extends)
		}
		rest := f.HeaderRest
		if rest == "" {
			rest = "\n   tokens and looks only; edit here or in the Theme view of si Studio "
		}
		b.WriteString(rest + "*/\n")
	}
	if f.Preamble != "" {
		b.WriteString(f.Preamble + "\n\n")
	}
	block := func(sel, indent string, m map[string]string) {
		b.WriteString(indent + sel + " {\n")
		for _, k := range Order(m) {
			fmt.Fprintf(&b, "%s  %s: %s;\n", indent, k, m[k])
		}
		b.WriteString(indent + "}\n")
	}
	block(selLight, "", f.Light)
	if len(f.Dark) > 0 {
		block(selDark, "", f.Dark)
		b.WriteString(selDarkMedia + " {\n")
		block(selDarkAuto, "  ", f.Dark)
		b.WriteString("}\n")
	}
	if f.Extra != "" {
		b.WriteString("\n" + f.Extra + "\n")
	}
	return b.String()
}

// Set changes one token in light ("" or "light") or dark mode; an empty
// value removes it (back to the extended theme's value). The value is
// checked with CheckValue first.
func (f *File) Set(name, mode, value string) error {
	m := f.Light
	if mode == "dark" {
		m = f.Dark
	} else if mode != "" && mode != "light" {
		return fmt.Errorf("mode is light or dark, not %q", mode)
	}
	if strings.TrimSpace(value) == "" {
		delete(m, name)
		return nil
	}
	if err := CheckValue(name, value); err != nil {
		return err
	}
	m[name] = strings.TrimSpace(value)
	return nil
}

// Order returns the keys of m in contract order, unknown ones last by name.
func Order(m map[string]string) []string {
	pos := map[string]int{}
	for i, t := range assets.Tokens {
		pos[t.Name] = i + 1
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := pos[keys[i]], pos[keys[j]]
		switch {
		case a > 0 && b > 0:
			return a < b
		case a > 0 || b > 0:
			return a > 0
		}
		return keys[i] < keys[j]
	})
	return keys
}

// Effective returns the values a page sees with this theme in light and in
// dark mode: the extended built-in's, then the file's, in cascade order.
func (f *File) Effective() (light, dark map[string]string) {
	bl, bd, _ := assets.ThemeTokens(f.Extends)
	light, dark = map[string]string{}, map[string]string{}
	for _, m := range []map[string]string{bl, f.Light} {
		for k, v := range m {
			light[k], dark[k] = v, v
		}
	}
	for _, m := range []map[string]string{bd, f.Dark} {
		for k, v := range m {
			dark[k] = v
		}
	}
	return light, dark
}

// item is one top-level piece of CSS: a rule (prelude + body), or other text
// (whitespace, a comment, a statement). text is the piece verbatim.
type item struct {
	text, prelude, body string
}

// scan splits css into top-level items, respecting comments and strings.
func scan(css string) []item {
	var out []item
	i := 0
	for i < len(css) {
		j := i
		for j < len(css) && strings.ContainsRune(" \t\r\n", rune(css[j])) {
			j++
		}
		if j > i {
			out = append(out, item{text: css[i:j]})
			i = j
			continue
		}
		if strings.HasPrefix(css[i:], "/*") {
			e := strings.Index(css[i+2:], "*/")
			if e < 0 {
				out = append(out, item{text: css[i:]})
				return out
			}
			out = append(out, item{text: css[i : i+e+4]})
			i += e + 4
			continue
		}
		// a prelude up to "{" or a statement up to ";"
		k, open := i, -1
		for k < len(css) && open < 0 {
			switch c := css[k]; {
			case c == '"' || c == '\'':
				k = skipString(css, k)
			case strings.HasPrefix(css[k:], "/*"):
				if e := strings.Index(css[k+2:], "*/"); e >= 0 {
					k += e + 4
				} else {
					k = len(css)
				}
			case c == ';':
				out = append(out, item{text: css[i : k+1]})
				i = k + 1
				open = -2
			case c == '{':
				open = k
			default:
				k++
			}
		}
		if open == -2 {
			continue
		}
		if open < 0 {
			out = append(out, item{text: css[i:]})
			return out
		}
		depth, k := 0, open
		for k < len(css) {
			c := css[k]
			if c == '"' || c == '\'' {
				k = skipString(css, k)
				continue
			}
			if strings.HasPrefix(css[k:], "/*") {
				if e := strings.Index(css[k+2:], "*/"); e >= 0 {
					k += e + 4
					continue
				}
				k = len(css)
				break
			}
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					break
				}
			}
			k++
		}
		if k >= len(css) { // unbalanced: keep the rest as text
			out = append(out, item{text: css[i:]})
			return out
		}
		out = append(out, item{text: css[i : k+1], prelude: strings.TrimSpace(css[i:open]), body: css[open+1 : k]})
		i = k + 1
	}
	return out
}

func skipString(s string, i int) int {
	q := s[i]
	i++
	for i < len(s) && s[i] != q {
		if s[i] == '\\' {
			i++
		}
		i++
	}
	return i + 1
}

// split splits s on sep outside strings and brackets.
func split(s string, sep byte) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			i = skipString(s, i) - 1
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}
