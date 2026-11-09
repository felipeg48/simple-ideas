package skill

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite plugin/ from docs/agent-rules.md and docs/RULES.md")

// The plugin committed under plugin/ is exactly what the generator makes, so
// it never drifts from the rules (go test ./internal/skill -update rewrites it).
func TestPluginInSync(t *testing.T) {
	root := filepath.Join("..", "..", "plugin")
	src, err := os.ReadFile(filepath.Join("..", "..", "cmd", "si", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^const version = "([^"]+)"`).FindSubmatch(src)
	if m == nil {
		t.Fatal("no const version in cmd/si/main.go")
	}
	want := Plugin(string(m[1]))
	if *update {
		os.RemoveAll(root)
		for p, c := range want {
			os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
			if err := os.WriteFile(filepath.Join(root, p), []byte(c), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	var have []string
	filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			rel, _ := filepath.Rel(root, p)
			have = append(have, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(have)
	for _, p := range have {
		if _, ok := want[p]; !ok {
			t.Errorf("plugin/%s is not generated any more (go test ./internal/skill -update)", p)
		}
	}
	for p, c := range want {
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil || string(b) != c {
			t.Errorf("plugin/%s is out of date: go test ./internal/skill -update", p)
		}
	}
}

// The skill loads on the right requests and points at every reference.
func TestSkillShape(t *testing.T) {
	f := Files()
	s := f["SKILL.md"]
	if !strings.HasPrefix(s, "---\nname: si\ndescription: \"") || len(Description) > 1024 {
		t.Errorf("frontmatter: name si and a description of at most 1024 characters (%d)", len(Description))
	}
	for p := range f {
		if strings.HasPrefix(p, "references/") && !strings.Contains(s, "("+p+")") {
			t.Errorf("SKILL.md does not list %s", p)
		}
	}
	if !strings.Contains(s, "si check") || !strings.Contains(s, "[CLICK]") {
		t.Error("SKILL.md should carry the check loop and the agent guide (Notes:, [CLICK])")
	}
}
