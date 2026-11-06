package main

import (
	"bytes"
	"io"
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

func TestThemeList(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"theme", "list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("theme list: %d %s", code, stderr.String())
	}
	for _, want := range []string{"acemq-pro", "AceMQ Pro", "cockroachlabs", "studio"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("theme list lacks %q", want)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--theme", "nope", "build", "."}, &stdout, &stderr); code != exitUsage || !strings.Contains(stderr.String(), "si theme list") {
		t.Errorf("unknown --theme: code %d, %s", code, stderr.String())
	}
	dir, out := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.md"), []byte("# D\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"build", "--theme", "AceMQ Pro", "-o", out, dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("build --theme \"AceMQ Pro\": %d %s", code, stderr.String())
	}
	if html, _ := os.ReadFile(filepath.Join(out, "d.html")); !strings.Contains(string(html), `data-si-theme="acemq-pro"`) {
		t.Error("--theme \"AceMQ Pro\" does not render acemq-pro")
	}
}

func TestExportAgentRules(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--export", "agent-rules", "-o", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("export: %d %s", code, stderr.String())
	}
	for _, n := range []string{"CLAUDE.md", "AGENTS.md"} {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil || !strings.Contains(string(b), "Teleprompter vs Speaker Notes") || !strings.Contains(string(b), "si "+version) {
			t.Errorf("%s: missing or without the teleprompter rule (%v)", n, err)
		}
	}
	if code := run([]string{"--export", "agent-rules", "-o", dir}, &stdout, &stderr); code != exitProblems {
		t.Error("an existing CLAUDE.md must not be overwritten")
	}
	stdout.Reset()
	if code := run([]string{"--export", "agent-rules", "-o", "-"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Points:") {
		t.Error("-o - prints the guide")
	}
}

func TestSiteTitleFlag(t *testing.T) {
	o, err := parseArgs([]string{"build", "--site-title", "Harbor", "d"}, io.Discard)
	if err != nil || o.title != "Harbor" {
		t.Fatalf("--site-title: %v %q", err, o.title)
	}
	var stderr bytes.Buffer
	if o, _ := parseArgs([]string{"build", "--site", "Old", "d"}, &stderr); o.title != "Old" || !strings.Contains(stderr.String(), "--site-title") {
		t.Errorf("--site <text> still sets the title, with a warning: %q %q", o.title, stderr.String())
	}
}

func TestRemoteFlag(t *testing.T) {
	o, err := parseArgs([]string{"play", "--remote", "d.md"}, io.Discard)
	if err != nil || strings.Join(o.remote.roles, ",") != "tp,draw" || len(o.args) != 2 {
		t.Fatalf("--remote: %v %v %v", err, o.remote.roles, o.args)
	}
	o, err = parseArgs([]string{"play", "d.md", "--remote=draw", "--remote-addr", "127.0.0.1:0"}, io.Discard)
	if err != nil || strings.Join(o.remote.roles, ",") != "draw" || o.remoteAddr != "127.0.0.1:0" {
		t.Fatalf("--remote=draw: %v %v", err, o.remote.roles)
	}
	if _, err := parseArgs([]string{"play", "d.md", "--remote=nope"}, io.Discard); err == nil {
		t.Error("an unknown role should fail")
	}
}

func TestExportExampleTypesAndBuildGuide(t *testing.T) {
	for _, kind := range []string{"", "slides", "article", "guide", "tutorial", "lab", "book", "reference", "site", "all"} {
		dir := filepath.Join(t.TempDir(), "ex")
		var stdout, stderr bytes.Buffer
		if code := run([]string{"--export", "example", "--type", kind, "-o", dir}, &stdout, &stderr); code != 0 {
			t.Fatalf("--type %q: %d %s", kind, code, stderr.String())
		}
		if code := run([]string{"check", dir}, &stdout, &stderr); code != 0 {
			t.Errorf("--type %q: si check failed: %s", kind, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--export", "example", "--type", "slideshow", "-o", t.TempDir()}, &stdout, &stderr); code != exitUsage {
		t.Error("an unknown --type is a usage error")
	}
	src := filepath.Join(t.TempDir(), "all")
	run([]string{"--export", "example", "--type", "all", "-o", src}, &stdout, &stderr)
	out := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"build", "-o", out, src}, &stdout, &stderr); code != 0 {
		t.Fatalf("build all: %d %s", code, stderr.String())
	}
	for _, f := range []string{"index.html", "retry.html", "showcase.html", "guide/index.html", "guide/happy/pay.html", "guide/reset.html", "site/index.html", "site/about.html", "site/docs/break.html", "site/talks/journey.html"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("build all lacks %s", f)
		}
	}
	idx, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(idx), `href="guide/index.html"`) || !strings.Contains(string(idx), ">GUIDE<") || !strings.Contains(string(idx), ">ARTICLE<") {
		t.Error("the index has a guide tile and an article tile")
	}
}

func TestNewWithTypeAndFix(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	var out, errb bytes.Buffer
	if code := run([]string{"new", "harbor", "--type", "site"}, &out, &errb); code != 0 {
		t.Fatalf("new site: %d %s", code, errb.String())
	}
	if code := run([]string{"new", "harbor/queues", "--type", "article"}, &out, &errb); code != 0 {
		t.Fatalf("new article: %d %s", code, errb.String())
	}
	if nav, _ := os.ReadFile(filepath.Join(dir, "harbor", "_site.md")); !strings.Contains(string(nav), "- [Queues](queues.md)") {
		t.Errorf("the article joins the site menu:\n%s", nav)
	}
	if code := run([]string{"check", "harbor"}, &out, &errb); code != 0 {
		t.Errorf("starters check clean: %s", errb.String())
	}
	if code := run([]string{"new", "harbor/queues", "--type", "article"}, &out, &errb); code == 0 {
		t.Error("si new never overwrites")
	}

	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\n## A <span class=\"hl\">phrase</span>\n"), 0o644)
	out.Reset()
	run([]string{"fix", "talk.md"}, &out, &errb)
	if raw, _ := os.ReadFile(filepath.Join(dir, "talk.md")); strings.Contains(string(raw), "==") || !strings.Contains(out.String(), "1 highlight") {
		t.Errorf("a dry run reports and writes nothing: %s", out.String())
	}
	run([]string{"fix", ".", "--write"}, &out, &errb)
	if raw, _ := os.ReadFile(filepath.Join(dir, "talk.md")); !strings.Contains(string(raw), "## A ==phrase==") {
		t.Errorf("--write applies it: %s", raw)
	}
}
