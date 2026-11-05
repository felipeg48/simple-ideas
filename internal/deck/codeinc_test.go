package deck

import (
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

func TestAdocCodeMapping(t *testing.T) {
	md := adocToMarkdown("= T\n\ninclude::App.java[lines=5..10;20..-1]\n\ninclude::App.java[tags=a;b]\n\n.Consumer.java\n[source,java,highlight=3;5..6]\n----\nx\n----\n")
	for _, want := range []string{"<!-- include: App.java lines=5-10,20- -->", "<!-- include: App.java tag=a,b -->", "```java title=\"Consumer.java\" hl=\"3,5-6\""} {
		if !strings.Contains(md, want) {
			t.Errorf("adoc lacks %q:\n%s", want, md)
		}
	}
}
