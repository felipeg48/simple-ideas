package render

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// Demos are instructor demo scripts (plain markdown) shown in the `si play`
// terminal drawer: a drop-down picks one, the panel shows the whole document,
// and each shell code block gets Copy and Run. The front matter
// `demos: ../demos/, extra.md` names files or folders (relative to the deck);
// a folder contributes its *.md files except README.md, index.md and _-prefixed
// ones. Like the runbook, it is resolved on every render (so `si check`
// validates the paths) and injected only with the play terminal.

type demoDoc struct {
	Title string `json:"title"`
	File  string `json:"file"`
	HTML  string `json:"html"`
}

// runLangs are the fence languages whose blocks get Copy / Run; everything
// else (text, output, json, …) is shown as plain output.
var runLangs = map[string]bool{"bash": true, "sh": true, "shell": true, "zsh": true, "console": true, "sql": true, "powershell": true, "ps1": true, "pwsh": true, "fish": true}

var demoMD = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(gmhtml.WithUnsafe()))

var demoCodeRE = regexp.MustCompile(`(?s)<pre><code(?: class="language-([^"]+)")?>(.*?)</code></pre>`)

// demoHTML renders a demo script: GFM markdown, code blocks highlighted, the
// runnable ones marked for the drawer's Copy / Run buttons.
func demoHTML(src []byte) string {
	var buf bytes.Buffer
	if err := demoMD.Convert(src, &buf); err != nil {
		return "<pre>" + html.EscapeString(string(src)) + "</pre>"
	}
	return demoCodeRE.ReplaceAllStringFunc(buf.String(), func(m string) string {
		sm := demoCodeRE.FindStringSubmatch(m)
		lang, code := sm[1], strings.TrimRight(html.UnescapeString(sm[2]), "\n")
		cls := "si-dm-code"
		if runLangs[lang] {
			cls += " si-dm-run"
		}
		return `<pre class="` + cls + `"><code>` + highlight(code, lang) + `</code></pre>`
	})
}

// demoTitle is the first `# ` heading, else the file name.
func demoTitle(src []byte, path string) string {
	for _, l := range strings.SplitN(string(src), "\n", 40) {
		if strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(l[2:])
		}
	}
	return strings.TrimSuffix(filepath.Base(path), ".md")
}

// demoFiles expands the comma-separated demos: value into markdown files.
func demoFiles(base, spec string) ([]string, []string) {
	var files, missing []string
	for _, p := range strings.Split(spec, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		path := p
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, p)
		}
		fi, err := os.Stat(path)
		if err != nil {
			missing = append(missing, p)
			continue
		}
		if !fi.IsDir() {
			files = append(files, path)
			continue
		}
		mds, _ := filepath.Glob(filepath.Join(path, "*.md"))
		sort.Strings(mds)
		for _, f := range mds {
			n := strings.ToLower(filepath.Base(f))
			if n == "readme.md" || n == "index.md" || strings.HasPrefix(n, "_") {
				continue
			}
			files = append(files, f)
		}
	}
	return files, missing
}

// resolveDemos loads the deck's `demos:` scripts into dr.demosJSON.
func (dr *deckRenderer) resolveDemos() {
	spec := dr.d.Meta.Demos
	if spec == "" {
		return
	}
	files, missing := demoFiles(filepath.Dir(dr.d.Path), spec)
	for _, m := range missing {
		dr.problem(nil, "demos: cannot read %s", m)
	}
	var docs []demoDoc
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			dr.problem(nil, "demos: cannot read %s", f)
			continue
		}
		docs = append(docs, demoDoc{Title: demoTitle(src, f), File: filepath.Base(f), HTML: demoHTML(src)})
	}
	if len(docs) == 0 {
		if len(missing) == 0 {
			dr.problem(nil, "demos: no markdown files in %s", spec)
		}
		return
	}
	b, _ := json.Marshal(docs)
	dr.demosJSON = string(b)
}
