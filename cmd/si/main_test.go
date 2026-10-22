package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgsGNUStyle(t *testing.T) {
	tests := []struct {
		argv   []string
		args   string
		output string
		css    string
	}{
		{[]string{"--css", "css/", "--output", "html/", "md/"}, "md/", "html/", "css/"},
		{[]string{"md/", "-o", "html/"}, "md/", "html/", ""},
		{[]string{"build", "md/", "--output=out", "--css", "a.css", "--css", "b/"}, "build md/", "out", "a.css,b/"},
		{[]string{"play", "--open", "decks", "--addr", ":0"}, "play decks", "", ""},
	}
	for _, tc := range tests {
		o, err := parseArgs(tc.argv, os.Stderr)
		if err != nil {
			t.Fatalf("%v: %v", tc.argv, err)
		}
		if strings.Join(o.args, " ") != tc.args || o.output != tc.output || o.css.String() != tc.css {
			t.Errorf("%v: args=%q output=%q css=%q", tc.argv, o.args, o.output, o.css.String())
		}
	}
}

func TestRunBuildCheckExport(t *testing.T) {
	out := t.TempDir()
	var stdout, stderr bytes.Buffer
	examples := filepath.Join("..", "..", "examples")
	if code := run([]string{"--css", filepath.Join(examples, "css"), "--output", out, examples}, &stdout, &stderr); code != 0 {
		t.Fatalf("build exit %d: %s", code, stderr.String())
	}
	for _, f := range []string{"index.html", "showcase.html", "appendix.html"} {
		raw, err := os.ReadFile(filepath.Join(out, f))
		if err != nil {
			t.Fatal(err)
		}
		if f == "showcase.html" && !strings.Contains(string(raw), "Harbor") {
			t.Error("--css stylesheet not inlined")
		}
	}
	stdout.Reset()
	if code := run([]string{"check", examples}, &stdout, &stderr); code != 0 {
		t.Fatalf("check exit %d: %s", code, stderr.String())
	}
	stdout.Reset()
	if code := run([]string{"--export", "css-rules"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "--si-accent:") {
		t.Errorf("export css-rules: %d %.200s", code, stdout.String())
	}
	stdout.Reset()
	if code := run([]string{"rules"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "# ") {
		t.Error("rules prints RULES.md")
	}
	ex := filepath.Join(t.TempDir(), "talk")
	if code := run([]string{"new", ex}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(ex, "talk.md")); err != nil {
		t.Error("new writes <name>/<name>.md")
	}
	if code := run([]string{"check", ex}, &stdout, &stderr); code != 0 {
		t.Errorf("a new deck checks clean: %s", stderr.String())
	}
}

func TestCheckFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.md"), []byte("## no title\n\n<!-- wobble -->\n![x](missing.png)\n"), 0o644)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", dir}, &stdout, &stderr); code != exitProblems {
		t.Fatalf("check exit %d, want %d", code, exitProblems)
	}
	for _, want := range []string{"missing title", `unknown directive "wobble"`, "missing file: missing.png"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("check output lacks %q:\n%s", want, stderr.String())
		}
	}
}

func TestUsageErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	for _, argv := range [][]string{{}, {"nonexistent-path-xyz"}, {"--export", "nope"}, {"--bogus"}} {
		if code := run(argv, &stdout, &stderr); code != exitUsage {
			t.Errorf("%v: exit %d, want %d", argv, code, exitUsage)
		}
	}
}
