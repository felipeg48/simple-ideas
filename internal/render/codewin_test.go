package render

import (
	"regexp"
	"strings"
	"testing"
)

func TestCodeWindows(t *testing.T) {
	show := "```go id=r window=mac numbers title=\"retry.go\" step=\"One\"\na()\n```\n<!-- callout: 1 one call at=40,-20 -->\n\n```go id=r window=mac numbers hl=2\na()\nb()\n```\n"
	// a document: one window with two steps, ← → under it, the scripts loaded
	res, _ := renderFiles(t, map[string]string{"a.md": "---\ntype: article\n---\n# A\n\n" + show}, "a.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 || strings.Count(h, `<figure class="si-cw"`) != 1 || !strings.Contains(h, `data-window="mac"`) || !strings.Contains(h, `data-numbers`) || !strings.Contains(h, `data-cw-nav`) {
		t.Fatalf("document: %s", problemsText(res))
	}
	if !strings.Contains(h, `"notes":[{"line":1,"html":"one call","src":"one call","at":[40,-20]}]`) || !strings.Contains(h, `"hl":[2]`) || !strings.Contains(h, `"name":"One"`) || !strings.Contains(h, "si code windows:") {
		t.Errorf("data or script missing")
	}
	if strings.Contains(h, "callout:") {
		t.Error("callout comments must leave the page")
	}
	// a slide: the second step is a build step
	res, _ = renderFiles(t, map[string]string{"d.md": "# D\n\n---\n\n## Retry\n\n" + show + "\nNotes: One. [CLICK] Two.\n"}, "d.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 || strings.Count(h, `si-step si-cw-mark`) != 1 || regexp.MustCompile(`<figure class="si-cw"[^>]*data-cw-nav`).MatchString(h) {
		t.Errorf("slide: %s", problemsText(res))
	}
	// plain fences stay plain; no scripts
	res, _ = renderFiles(t, map[string]string{"p.md": "# P\n\n---\n\n## Code\n\n```go\na()\n```\n"}, "p.md")
	if h := string(res.HTML); strings.Contains(h, `class="si-cw`) || strings.Contains(h, "si code windows:") {
		t.Error("a plain fence must not become a window")
	}
	// problems: bad values, a callout off the code, a callout with nothing above it
	res, _ = renderFiles(t, map[string]string{"b.md": "---\ntype: article\n---\n# B\n\n```go window=linux backdrop=sunset\na()\n```\n<!-- callout: 7 far -->\n\nText.\n\n<!-- callout: 1 alone -->\n"}, "b.md")
	got := problemsText(res)
	for _, want := range []string{`unknown code window window "linux"`, `unknown code window backdrop "sunset"`, "callout on line 7: the code block has 1 lines", "a callout goes right after a code block"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing problem %q in %s", want, got)
		}
	}
}

func TestFenceInfoForms(t *testing.T) {
	src := "---\ntype: article\n---\n# T\n\n``` title=\"config.csproj\"\n<Project/>\n```\n\n```{.xml title=\"app.config\"}\n<Project/>\n```\n\n```xml {title=\"x.props\"}\n<Project/>\n```\n"
	res, _ := renderFiles(t, map[string]string{"t.md": src}, "t.md")
	h := string(res.HTML)
	for _, want := range []string{`<span class="si-code-title">config.csproj</span><span class="si-code-lang">XML</span>`,
		`<span class="si-code-title">app.config</span><span class="si-code-lang">XML</span>`,
		`<span class="si-code-title">x.props</span><span class="si-code-lang">XML</span>`, `<span class="si-tk-k">&lt;Project</span>`} {
		if !strings.Contains(h, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(strings.ToLower(h), `title=&#34;config`) || strings.Contains(h, `{.xml`) {
		t.Error("an option or a pandoc class leaked into the language label")
	}
}
