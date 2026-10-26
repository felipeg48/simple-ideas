package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdocToMarkdown(t *testing.T) {
	src := `= My Talk
:author: Ada
:theme: crdb

== Intro

Some *bold* and a https://example.com[link].

* one
* two

[source,go]
----
x := 1
----

NOTE: heads up.

// a comment to drop
=== Sub`
	md := adocToMarkdown(src)
	for _, want := range []string{
		"---\ntitle: My Talk",
		"author: Ada",
		"theme: crdb",
		"# My Talk",
		"\n---\n",       // slide break before the section
		"## Intro",      // == -> ##
		"Some **bold**", // inline bold
		"[link](https://example.com)",
		"- one", // list
		"```go", // source block fence
		"x := 1",
		"> **Note:** heads up.",
		"### Sub", // === -> ###
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
	if strings.Contains(md, "a comment to drop") {
		t.Error("// comment not dropped")
	}
}

func TestAdocDeepFeatures(t *testing.T) {
	src := `== Features

A quote:

[quote]
____
Simplicity is prerequisite for reliability.
____

Options::
Term:: a *bold* definition

[cols="1,1"]
|===
| Name | Role

| Ada | Engineer
| Bob | Designer
|===

include::appendix.adoc[]
`
	md := adocToMarkdown(src)
	for _, want := range []string{
		"> Simplicity is prerequisite",     // quote block -> blockquote
		"**Term** — a **bold** definition", // description list + inline bold
		"| Name | Role |",                  // table header
		"| --- | --- |",                    // table separator
		"| Ada | Engineer |",               // table data row
		"<!-- include: appendix.adoc -->",  // include::
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
}

func TestAdocCallouts(t *testing.T) {
	src := "[source,go]\n----\nfmt.Println(\"hi\") // <1>\n----\n<1> prints a greeting\n<2> and returns\n"
	md := adocToMarkdown(src)
	for _, want := range []string{
		"fmt.Println(\"hi\") // <1>", // marker kept verbatim in the code
		"1. prints a greeting",       // callout list -> numbered item
		"2. and returns",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in:\n%s", want, md)
		}
	}
}

func TestColsCount(t *testing.T) {
	cases := map[string]int{`"1,1,2"`: 3, `3*`: 3, `4`: 4, `"a,b"`: 2, `~`: 0}
	for spec, want := range cases {
		if got := colsCount(spec); got != want {
			t.Errorf("colsCount(%q)=%d, want %d", spec, got, want)
		}
	}
}

func TestParseAdocDeck(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "talk.adoc")
	os.WriteFile(p, []byte("= T\n:theme: crdb\n\n== One\n\n* a\n\n== Two\n\nbody\n"), 0o644)
	d, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "talk" {
		t.Errorf("name=%q", d.Name)
	}
	if d.Meta.Theme != "crdb" {
		t.Errorf("theme=%q (attrs not mapped to front matter)", d.Meta.Theme)
	}
	if len(d.Slides) != 3 { // title + One + Two
		t.Errorf("slides=%d, want 3", len(d.Slides))
	}
}

func TestListAcceptsAdoc(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.md", "b.adoc", "c.asciidoc", "_mod.adoc", "README.md"} {
		os.WriteFile(filepath.Join(dir, n), []byte("= x\n"), 0o644)
	}
	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 { // a.md, b.adoc, c.asciidoc; _mod and README excluded
		t.Errorf("List=%v, want 3 decks", got)
	}
}
