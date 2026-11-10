package studio

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeView(t *testing.T) {
	f := newFx(t)
	code, m := f.do("GET", "theme/spec", nil)
	if code != 200 || len(m["tokens"].([]any)) < 50 || len(m["themes"].([]any)) < 5 {
		t.Fatalf("spec: %d", code)
	}
	for _, th := range m["themes"].([]any) {
		if n := th.(map[string]any)["name"]; n == "crdb" || n == "acemq" {
			t.Errorf("unlisted theme %s offered", n)
		}
	}
	code, m = f.do("POST", "theme/new", map[string]string{"name": "Acme Brand", "from": "paper"})
	if code != 201 || m["path"] != "themes/acme-brand.css" {
		t.Fatalf("new: %d %v", code, m)
	}
	if code, _ = f.do("POST", "theme/new", map[string]string{"name": "Acme Brand"}); code != 409 {
		t.Errorf("new over an existing theme: %d", code)
	}
	raw, _ := os.ReadFile(filepath.Join(f.root, "themes", "acme-brand.css"))
	text := string(raw)
	if !strings.HasPrefix(text, "/* si theme: Acme Brand · extends paper") {
		t.Fatalf("new file: %q", text)
	}
	// one op: the accent in dark mode; the answer carries the text, values, checks and the preview
	code, m = f.do("POST", "theme", map[string]any{"path": "themes/acme-brand.css", "text": text,
		"op": map[string]string{"token": "--si-accent", "mode": "dark", "value": "#333333"}})
	if code != 200 || !strings.Contains(m["text"].(string), `:root[data-si-mode="dark"] {`+"\n  --si-accent: #333333;") {
		t.Fatalf("op: %d %v", code, m["text"])
	}
	if m["effLight"].(map[string]any)["--si-bg"] != "#fbf7ee" || m["effDark"].(map[string]any)["--si-accent"] != "#333333" {
		t.Errorf("effective values: %v", m["effDark"])
	}
	if len(m["issues"].([]any)) == 0 {
		t.Error("a dark accent on a dark ground passed the checks")
	}
	if h := m["html"].(string); !strings.Contains(h, "Bread by Six") || !strings.Contains(h, "--si-accent: #333333") {
		t.Error("preview does not use the edited theme")
	}
	if code, _ = f.do("POST", "theme", map[string]any{"path": "themes/acme-brand.css", "text": text,
		"op": map[string]string{"token": "--si-h1-size", "value": "40px"}}); code != 422 {
		t.Errorf("px size: %d", code)
	}
	if code, _ = f.do("POST", "theme", map[string]any{"path": "../x.css", "text": text}); code != 403 {
		t.Errorf("outside the workspace: %d", code)
	}
	if code, _ = f.do("POST", "theme/use", map[string]string{"path": "themes/acme-brand.css"}); code != 200 {
		t.Fatalf("use: %d", code)
	}
	if b, _ := os.ReadFile(filepath.Join(f.root, "si.conf")); string(b) != "theme: themes/acme-brand.css\n" {
		t.Errorf("si.conf: %q", b)
	}
}

func TestThemeFromLogo(t *testing.T) {
	f := newFx(t)
	logo := `<svg viewBox="0 0 2 1"><rect fill="#e4572e" width="1" height="1"/><rect fill="#1d3a8a" x="1" width="1" height="1"/></svg>`
	os.MkdirAll(filepath.Join(f.root, "img"), 0o755)
	os.WriteFile(filepath.Join(f.root, "img", "acme-logo.svg"), []byte(logo), 0o644)
	code, m := f.do("POST", "theme/logo", map[string]any{"path": "img/acme-logo.svg"})
	if code != 200 || m["name"] != "Acme" || m["path"] != "themes/acme.css" || len(m["palette"].([]any)) != 2 {
		t.Fatalf("preview: %d %v", code, m)
	}
	if _, err := os.Stat(filepath.Join(f.root, "themes")); err == nil {
		t.Error("a preview wrote a file")
	}
	code, m = f.do("POST", "theme/logo", map[string]any{"data": base64.StdEncoding.EncodeToString([]byte(logo)), "name": "Acme Two", "create": true})
	if code != 201 || m["path"] != "themes/acme-two.css" {
		t.Fatalf("create: %d %v", code, m)
	}
	raw, _ := os.ReadFile(filepath.Join(f.root, "themes", "acme-two.css"))
	if !strings.HasPrefix(string(raw), "/* si theme: Acme Two · extends studio") {
		t.Errorf("file: %q", raw)
	}
	for _, c := range []struct {
		body map[string]any
		code int
	}{
		{map[string]any{"path": "../x.png"}, 403},
		{map[string]any{"path": ".git/x.png"}, 403},
		{map[string]any{"data": "!!"}, 400},
		{map[string]any{"data": base64.StdEncoding.EncodeToString([]byte("text"))}, 422},
		{map[string]any{}, 400},
	} {
		if code, _ := f.do("POST", "theme/logo", c.body); code != c.code {
			t.Errorf("%v: %d, want %d", c.body, code, c.code)
		}
	}
}
