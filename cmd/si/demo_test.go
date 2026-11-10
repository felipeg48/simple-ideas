package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"demo"}, &out, &errb); code != exitUsage {
		t.Errorf("si demo: exit %d", code)
	}
	errb.Reset()
	if code := run([]string{"demo", "--help"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "si demo test") || !strings.Contains(out.String(), "--captions") {
		t.Errorf("si demo --help: %d %s", code, out.String())
	}
	out.Reset()
	if code := run([]string{"--help"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "si demo test") {
		t.Error("si --help does not list si demo")
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.md")
	os.WriteFile(bad, []byte("---\ntype: demo\ntitle: Bad\napp: http://localhost:1\n---\n\n## A\n\n- clik \"Pay\"\n- type $SI_DEMO_TEST_UNSET into \"Password\"\n"), 0o644)
	errb.Reset()
	if code := run([]string{"check", bad}, &out, &errb); code != exitProblems || !strings.Contains(errb.String(), `bad.md:9: unknown step "clik`) || !strings.Contains(errb.String(), "$SI_DEMO_TEST_UNSET is not set") {
		t.Errorf("si check: %d %s", code, errb.String())
	}
	errb.Reset()
	if code := run([]string{"demo", "test", bad}, &out, &errb); code != exitProblems || !strings.Contains(errb.String(), "unknown step") {
		t.Errorf("si demo test on a bad script: %d %s", code, errb.String())
	}
	js := filepath.Join(dir, "ok.json")
	os.WriteFile(js, []byte(`{"type":"demo","app":"http://localhost:1","features":[{"name":"A","blocks":[{"steps":["go /"]}]}]}`), 0o644)
	out.Reset()
	if code := run([]string{"check", js}, &out, &errb); code != exitOK {
		t.Errorf("si check ok.json: %d %s", code, errb.String())
	}
}
