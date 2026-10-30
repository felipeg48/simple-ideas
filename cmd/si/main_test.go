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

func TestWebRefusesNonLoopback(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--web", "--addr", "0.0.0.0:0", "--no-open", "."}, &out, &errb); code != 2 || !strings.Contains(errb.String(), "loopback-only") {
		t.Fatalf("code %d: %s", code, errb.String())
	}
}

func TestWebFlagParses(t *testing.T) {
	o, err := parseArgs([]string{"--web", "decks"}, os.Stderr)
	if err != nil || !o.web || strings.Join(o.args, " ") != "decks" {
		t.Fatalf("%v %+v", err, o)
	}
}

const labMD = "---\ntype: lab\ntitle: Quorum Lab\n---\n\n# Quorum Lab\n\n## One\n\ntext\n\n```\n## not a section\n```\n\n## Two\n\n> [!NOTE]\n> hi\n"

func TestBuildWritesLab(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\nhello\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "lab.md"), []byte(labMD), 0o644)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "-o", out, dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "lab  2 sections") {
		t.Errorf("summary lacks lab line:\n%s", stdout.String())
	}
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"lab.html", "talk.html", `data-kind="lab"`, "2 sections"} {
		if !strings.Contains(string(idx), want) {
			t.Errorf("index lacks %q", want)
		}
	}
}

func TestBuildLinksLabs(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\nhello\n"), 0o644)
	for _, n := range []string{"lab-1", "lab-2", "lab-3"} {
		os.WriteFile(filepath.Join(dir, n+".md"), []byte("---\ntype: lab\ntitle: T "+n+"\n---\n\n## One\n"), 0o644)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "-o", out, dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	read := func(n string) string { b, _ := os.ReadFile(filepath.Join(out, n)); return string(b) }
	first, mid, last := read("lab-1.html"), read("lab-2.html"), read("lab-3.html")
	if strings.Contains(first, `rel="prev" href`) || !strings.Contains(first, `rel="next" href="lab-2.html"`) {
		t.Error("first lab: want only a next link to lab-2")
	}
	if !strings.Contains(mid, `rel="prev" href="lab-1.html"`) || !strings.Contains(mid, `rel="next" href="lab-3.html"`) || !strings.Contains(mid, "T lab-3") {
		t.Error("middle lab: want prev lab-1 and next lab-3 with its title")
	}
	if strings.Contains(last, `rel="next" href`) || strings.Contains(read("talk.html"), `class="si-doc-pager"`) {
		t.Error("last lab has no next link; decks get no pager")
	}
	// one lab alone: no pager
	one, out1 := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(one, "lab.md"), []byte(labMD), 0o644)
	if code := run([]string{"build", "-o", out1, one}, &stdout, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if b, _ := os.ReadFile(filepath.Join(out1, "lab.html")); strings.Contains(string(b), `class="si-doc-pager"`) {
		t.Error("a single lab must not get a pager")
	}
}

func TestCheckLab(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lab.md"), []byte(labMD+"\n![x](missing.png)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "bad.md"), []byte("---\ntype: wiki\n---\n# B\n"), 0o644)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", dir}, &stdout, &stderr); code != exitProblems {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"lab.md:", "missing file: missing.png", "type"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("check lacks %q:\n%s", want, stderr.String())
		}
	}
}
