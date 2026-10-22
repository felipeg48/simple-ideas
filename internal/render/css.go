package render

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Stylesheet is a user CSS file: its text and path (base for relative url()).
type Stylesheet struct {
	Path string
	Text string
}

var cssURLRE = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)['"]?\s*\)`)

// UserCSS wraps user stylesheets in the si.user layer, inlining relative
// url() references so the output stays self-contained.
func UserCSS(sheets []Stylesheet) string {
	if len(sheets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("@layer si.user{\n")
	for _, s := range sheets {
		dir := filepath.Dir(s.Path)
		text := cssURLRE.ReplaceAllStringFunc(s.Text, func(u string) string {
			m := cssURLRE.FindStringSubmatch(u)
			if isExternal(m[2]) || strings.HasPrefix(m[2], "#") {
				return u
			}
			if uri, ok := dataURI(filepath.Join(dir, filepath.FromSlash(cleanSrc(m[2])))); ok {
				return `url("` + uri + `")`
			}
			return u
		})
		fmt.Fprintf(&b, "/* %s */\n%s\n", filepath.Base(s.Path), text)
	}
	b.WriteString("}\n")
	return b.String()
}

// LoadCSS reads user stylesheets: files as given, directories as their *.css
// files in name order.
func LoadCSS(paths []string) ([]Stylesheet, error) {
	var out []Stylesheet
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			return out, fmt.Errorf("--css %s: %w", p, err)
		}
		files := []string{p}
		if st.IsDir() {
			files, _ = filepath.Glob(filepath.Join(p, "*.css"))
		}
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				return out, err
			}
			out = append(out, Stylesheet{Path: f, Text: string(raw)})
		}
	}
	return out, nil
}
