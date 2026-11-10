package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// export a built-in as CSS and as Design Tokens, import both back, check.
func TestThemeExportImportCheck(t *testing.T) {
	dir := t.TempDir()
	var out, errb bytes.Buffer
	css := filepath.Join(dir, "paper.si-theme.css")
	if code := run([]string{"theme", "export", "paper", "-o", css}, &out, &errb); code != 0 {
		t.Fatalf("export: %d %s", code, errb.String())
	}
	js := filepath.Join(dir, "paper.tokens.json")
	if code := run([]string{"theme", "export", "paper", "--format", "dtcg", "-o", js}, &out, &errb); code != 0 {
		t.Fatalf("export dtcg: %d %s", code, errb.String())
	}
	if b, _ := os.ReadFile(js); !bytes.Contains(b, []byte(`"$type": "color"`)) {
		t.Errorf("dtcg: %s", b)
	}
	themes := filepath.Join(dir, "themes")
	conf := filepath.Join(dir, "si.conf")
	if code := run([]string{"theme", "import", css, "-o", themes, "--use", "--conf", conf}, &out, &errb); code != 0 {
		t.Fatalf("import: %d %s %s", code, out.String(), errb.String())
	}
	if b, _ := os.ReadFile(conf); !strings.Contains(string(b), "theme: "+filepath.ToSlash(filepath.Join(themes, "paper.css"))) {
		t.Errorf("si.conf: %s", b)
	}
	if code := run([]string{"theme", "import", css, "-o", themes}, &out, &errb); code == 0 {
		t.Error("import over an existing theme should fail")
	}
	if code := run([]string{"theme", "import", js, "-o", filepath.Join(dir, "t2")}, &out, &errb); code != 0 {
		t.Fatalf("import dtcg: %d %s", code, errb.String())
	}
	if code := run([]string{"check", filepath.Join(themes, "paper.css")}, &out, &errb); code != 0 {
		t.Errorf("check: %s", out.String())
	}
	bad := filepath.Join(dir, "bad.css")
	os.WriteFile(bad, []byte(":root { --si-h1-size: 48px; }\n"), 0o644)
	out.Reset()
	if code := run([]string{"theme", "check", bad}, &out, &errb); code == 0 || !strings.Contains(out.String(), "bad.css:1: --si-h1-size: px") {
		t.Errorf("check bad: %d %s", code, out.String())
	}
	if code := run([]string{"theme", "import", bad, "-o", themes}, &out, &errb); code == 0 {
		t.Error("a theme with a px size was imported")
	}
}
