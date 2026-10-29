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
		fmt.Fprintf(&b, "/* %s */\n%s\n", filepath.Base(s.Path), inlineURLs(filepath.Dir(s.Path), s.Text))
	}
	b.WriteString("}\n")
	return b.String()
}

// inlineURLs replaces relative url() references in css (resolved against dir)
// with data URIs; external, #fragment and missing references stay as written.
func inlineURLs(dir, css string) string {
	return cssURLRE.ReplaceAllStringFunc(css, func(u string) string {
		m := cssURLRE.FindStringSubmatch(u)
		if isExternal(m[2]) || strings.HasPrefix(m[2], "#") {
			return u
		}
		if uri, ok := dataURI(filepath.Join(dir, filepath.FromSlash(cleanSrc(m[2])))); ok {
			return `url("` + uri + `")`
		}
		return u
	})
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
