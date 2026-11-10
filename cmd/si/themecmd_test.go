package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/mcp"
)

// The MCP theme tools: spec, write (only known tokens of their kind), check
// with fixes, export as Design Tokens, import, and (with Chrome) a picture.
func TestMCPThemeTools(t *testing.T) {
	root := t.TempDir()
	tools := map[string]func(json.RawMessage) (mcp.Result, error){}
	for _, tl := range mcpTools(root) {
		tools[tl.Name] = tl.Call
	}
	res, err := tools["si_theme_spec"](json.RawMessage(`{}`))
	if err != nil || len(res.Structured.(map[string]any)["tokens"].([]assets.TokenMeta)) < 50 {
		t.Fatalf("spec: %v", err)
	}
	if _, err := tools["si_theme_write"](json.RawMessage(`{"name":"Acme","extends":"paper","light":{"h1-size":"40px"}}`)); err == nil {
		t.Error("a px size was written")
	}
	res, err = tools["si_theme_write"](json.RawMessage(`{"name":"Acme","extends":"paper","light":{"--si-accent":"#0b3d91","muted":"#c9c9c9"},"dark":{"accent":"#8fb3ff"}}`))
	if err != nil {
		t.Fatal(err)
	}
	st := res.Structured.(map[string]any)
	if st["path"] != "themes/acme.css" || st["passes"] != false || st["ok"] != true {
		t.Errorf("write: %+v", st)
	}
	res, _ = tools["si_theme_check"](json.RawMessage(`{"path":"themes/acme.css"}`))
	if !strings.Contains(res.Text, "--si-muted on --si-bg") || !strings.Contains(res.Text, "try #") {
		t.Errorf("check: %s", res.Text)
	}
	res, err = tools["si_theme_export"](json.RawMessage(`{"theme":"themes/acme.css","format":"dtcg","output":"acme.tokens.json"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tools["si_theme_import"](json.RawMessage(`{"source":"acme.tokens.json","use":true}`)); err == nil {
		t.Error("import over the theme it came from should fail (themes/acme.css exists)")
	}
	os.Remove(filepath.Join(root, "themes", "acme.css"))
	if _, err := tools["si_theme_import"](json.RawMessage(`{"source":"acme.tokens.json","use":true}`)); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "si.conf")); !strings.Contains(string(b), "themes/acme.css") {
		t.Errorf("si.conf: %s", b)
	}
	// a theme from a logo, then checked: no issue
	os.WriteFile(filepath.Join(root, "brand.svg"), []byte(`<svg><rect fill="#e4572e"/><rect fill="#1d3a8a"/></svg>`), 0o644)
	if _, err := tools["si_theme_from_logo"](json.RawMessage(`{"logo":"../brand.svg"}`)); err == nil {
		t.Error("a logo outside the folder was read")
	}
	if _, err = tools["si_theme_from_logo"](json.RawMessage(`{"logo":"brand.svg","name":"Logo Co"}`)); err != nil {
		t.Fatal(err)
	}
	res, _ = tools["si_theme_check"](json.RawMessage(`{"path":"themes/logo-co.css"}`))
	if res.Structured.(map[string]any)["passes"] != true {
		t.Errorf("from logo: %s", res.Text)
	}
	if findChrome() == "" {
		return
	}
	res, err = tools["si_theme_preview"](json.RawMessage(`{"path":"themes/acme.css","dark":true}`))
	if err != nil || len(res.Images) != 1 {
		t.Errorf("preview: %v", err)
	}
}

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
