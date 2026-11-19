package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodeInclude(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"App.java": "class App {\n    void run() {\n        // tag::connect[]\n        var c = open();\n        c.use();\n        // end::connect[]\n    }\n}\n",
		"d.md":     "# D\n\n<!-- include: App.java tag=connect hl=2 -->\n\n<!-- include: App.java lines=1-2 lang=text title=\"Two lines\" -->\n",
		"bad.md":   "# B\n\n<!-- include: App.java tag=nope -->\n<!-- include: App.java lines=7-99 -->\n<!-- include: App.java colour=red -->\n<!-- include: gone.py -->\n",
	})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil || len(d.Problems) > 0 {
		t.Fatalf("%v %v", err, d.Problems)
	}
	body := d.Slides[0].Body
	for _, want := range []string{"```java title=\"App.java\" hl=\"2\" linenos start=4\nvar c = open();\nc.use();\n```", "```text title=\"Two lines\" linenos start=1\nclass App {\n    void run() {\n```"} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks\n%s\n---\n%s", want, body)
		}
	}
	if strings.Contains(body, "tag::") {
		t.Error("tag markers are never shown")
	}
	b, _ := Parse(filepath.Join(dir, "bad.md"))
	var msgs []string
	for _, p := range b.Problems {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{`tag "nope" not found`, "lines=7-99: the file has 8 lines", `unknown option "colour"`, "include gone.py: file not found"} {
		if !strings.Contains(all, want) {
			t.Errorf("problems lack %q:\n%s", want, all)
		}
	}
}

// TestCodeIncludeWindow: an include takes every code window option and
// writes the fence that would draw the same window (title: the file name).
func TestCodeIncludeWindow(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"start.sh": "#!/bin/sh\ndocker compose up -d\n",
		"notes.md": "# Hi\n\n```sh\nls\n```\n",
		"d.md":     "# D\n\n<!-- include: notes.md window=mac -->\n\n<!-- include: start.sh window=terminal as=cat numbers id=up step=\"Bring it up\" transition=fade theme=night backdrop=ocean hl=2 -->\n",
		"bad.md":   "# B\n\n<!-- include: start.sh window=mac colour=red -->\n",
	})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil || len(d.Problems) > 0 {
		t.Fatalf("%v %v", err, d.Problems)
	}
	want := "```bash title=\"start.sh\" hl=\"2\" linenos start=1 window=\"terminal\" theme=\"night\" backdrop=\"ocean\" id=\"up\" step=\"Bring it up\" transition=\"fade\" as=\"cat\"\n#!/bin/sh\n"
	if !strings.Contains(d.Slides[0].Body, want) {
		t.Errorf("body lacks\n%s\n---\n%s", want, d.Slides[0].Body)
	}
	if !strings.Contains(d.Slides[0].Body, "````markdown title=\"notes.md\" window=\"mac\"\n# Hi\n") {
		t.Errorf("Markdown in a window is code:\n%s", d.Slides[0].Body)
	}
	b, _ := Parse(filepath.Join(dir, "bad.md"))
	if len(b.Problems) != 1 || !strings.Contains(b.Problems[0].Msg, `unknown option "colour" (lines, tag, lang, title, hl, linenos, numbers, indent=keep, window, theme, backdrop, shadow, id, step, transition, as=cat)`) {
		t.Errorf("problems: %v", b.Problems)
	}
	code, err := IncludeCode(filepath.Join(dir, "d.md"), "<!-- include: start.sh window=mac lines=2 -->")
	if err != nil || code != "docker compose up -d" {
		t.Errorf("IncludeCode: %q %v", code, err)
	}
}

func TestAdocCodeMapping(t *testing.T) {
	md := adocToMarkdown("= T\n\ninclude::App.java[lines=5..10;20..-1]\n\ninclude::App.java[tags=a;b]\n\n.Consumer.java\n[source,java,highlight=3;5..6]\n----\nx\n----\n")
	for _, want := range []string{"<!-- include: App.java lines=5-10,20- -->", "<!-- include: App.java tag=a,b -->", "```java title=\"Consumer.java\" hl=\"3,5-6\""} {
		if !strings.Contains(md, want) {
			t.Errorf("adoc lacks %q:\n%s", want, md)
		}
	}
}

func TestCodeIncludeLanguage(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		file, body string
		opts       map[string]string
		want       string
	}{
		{write("NuGet.config", "<configuration>\n</configuration>\n"), "", map[string]string{"title": "NuGet.config"}, "```xml title=\"NuGet.config\""},
		{write("NuGet.config", "<configuration/>\n"), "", map[string]string{"lang": "text"}, "```text title=\"NuGet.config\""},
		{write("settings.unknown", "<?xml version=\"1.0\"?>\n<a/>\n"), "", nil, "```xml title=\"settings.unknown\""},
		{write("data.unknown", "{\"a\": 1}\n"), "", nil, "```json title=\"data.unknown\""},
		{write("run", "#!/usr/bin/env bash\necho hi\n"), "", nil, "```bash title=\"run\""},
		{write("Dockerfile", "FROM alpine\n"), "", nil, "```dockerfile title=\"Dockerfile\""},
		{write("App.csproj", "<Project/>\n"), "", nil, "```xml title=\"App.csproj\""},
		{write("rabbitmq.conf", "listeners.tcp.default = 5672\n"), "", nil, "```ini title=\"rabbitmq.conf\""},
		{write("app.properties", "a=1\n"), "", nil, "```ini title=\"app.properties\""},
		{write(".env", "A=1\n"), "", nil, "```ini title=\".env\""},
		{write("Containerfile", "FROM alpine\n"), "", nil, "```dockerfile title=\"Containerfile\""},
		{write("Makefile", "all:\n\tgo build\n"), "", nil, "```make title=\"Makefile\""},
		{write("compose.yml", "a: 1\n"), "", nil, "```yaml title=\"compose.yml\""},
		{write("start.zsh", "echo hi\n"), "", nil, "```bash title=\"start.zsh\""},
		{write("Cargo.toml", "[package]\n"), "", nil, "```toml title=\"Cargo.toml\""},
		{write("main.tf", "resource \"a\" \"b\" {}\n"), "", nil, "```hcl title=\"main.tf\""},
		{write("nginx.conf", "server {}\n"), "", nil, "```nginx title=\"nginx.conf\""},
		{write("README.md.txt", "# a\n"), "", nil, "```text title=\"README.md.txt\""},
		{write("q.sql", "select 1;\n"), "", nil, "```sql title=\"q.sql\""},
		{write("start.sh", "echo hi\n"), "", map[string]string{"lang": "zsh"}, "```zsh title=\"start.sh\""},
	} {
		lines, err := codeInclude(c.file, c.opts)
		if err != nil || lines[0] != c.want {
			t.Errorf("%s %v: %v %q, want %q", filepath.Base(c.file), c.opts, err, lines[0], c.want)
		}
	}
}
