package codeshow

import (
	"strings"
	"testing"
)

const src = "# Retry\n\n" +
	"```go id=retry window=mac step=\"One attempt\"\nfunc send() error {\n\treturn post()\n}\n```\n" +
	"<!-- callout: 2 One attempt at=60,-40 -->\n\n" +
	"```go id=retry window=mac\nfunc send() error {\n\tfor try := 1; try <= 5; try++ {}\n}\n```\n\n" +
	"Text after.\n\n```sh\nls\n```\n"

func TestShows(t *testing.T) {
	shows := Shows(src)
	if len(shows) != 2 || len(shows[0].Steps) != 2 || shows[0].ID != "retry" || len(shows[1].Steps) != 1 {
		t.Fatalf("shows: %+v", shows)
	}
	c := shows[0].Steps[0].Callouts
	if len(c) != 1 || c[0].Line != 2 || c[0].Text != "One attempt" || c[0].At == nil || c[0].At[1] != -40 {
		t.Fatalf("callouts: %+v", c)
	}
	if shows[0].Steps[0].Attrs["step"] != "One attempt" {
		t.Errorf("step name: %q", shows[0].Steps[0].Attrs["step"])
	}
	if _, ok := ShowAt(src, 6); !ok {
		t.Error("line 6 is in the first show")
	}
	if _, _, err := ParseCallout("<!-- callout: here -->"); err == nil {
		t.Error("a callout without a line number is an error")
	}
}

func TestApply(t *testing.T) {
	apply := func(text string, op Op) string {
		t.Helper()
		out, _, err := Apply(text, 3, op)
		if err != nil {
			t.Fatalf("%s: %v", op.Kind, err)
		}
		return out
	}
	out := apply(src, Op{Kind: "set", Field: "window", Value: "windows"})
	if strings.Count(out, "window=windows") != 2 || !strings.Contains(out, "```sh\n") {
		t.Errorf("set: %s", out)
	}
	out = apply(src, Op{Kind: "set", Field: "numbers", Value: "true"})
	if strings.Count(out, " numbers") != 2 {
		t.Errorf("flag: %s", out)
	}
	out = apply(src, Op{Kind: "set-step", Step: 1, Field: "title", Value: "retry.go"})
	if !strings.Contains(out, "window=mac title=retry.go\n") {
		t.Errorf("set-step: %s", out)
	}
	out = apply(src, Op{Kind: "set-code", Step: 1, Text: "func send() error {\n\treturn retry(post)\n}\n"})
	if s := Shows(out); len(s[0].Steps) != 2 || s[0].Steps[1].Code != "func send() error {\n\treturn retry(post)\n}" || !strings.Contains(out, "One attempt at=60,-40") {
		t.Errorf("set-code: %s", out)
	}
	out = apply(src, Op{Kind: "move-callout", Index: 0, At: []int{10, 20}})
	if !strings.Contains(out, "<!-- callout: 2 One attempt at=10,20 -->") {
		t.Errorf("move: %s", out)
	}
	out = apply(src, Op{Kind: "add-callout", Step: 1, Line: 2, Text: "Five tries"})
	if s := Shows(out); len(s[0].Steps[1].Callouts) != 1 || s[0].Steps[1].Callouts[0].Text != "Five tries" {
		t.Errorf("add: %s", out)
	}
	out = apply(src, Op{Kind: "del-callout", Index: 0})
	if strings.Contains(out, "callout:") {
		t.Errorf("del: %s", out)
	}
	out = apply(src, Op{Kind: "add-step", Step: 1})
	if s := Shows(out); len(s[0].Steps) != 3 || s[0].Steps[2].Code != s[0].Steps[1].Code {
		t.Errorf("add-step: %s", out)
	}
	out = apply(src, Op{Kind: "move-step", Step: 1, To: 0})
	if s := Shows(out); len(s[0].Steps) != 2 || !strings.Contains(s[0].Steps[0].Code, "for try") || len(s[0].Steps[1].Callouts) != 1 {
		t.Errorf("move-step: %s", out)
	}
	out = apply(src, Op{Kind: "del-step", Step: 0})
	if s := Shows(out); len(s[0].Steps) != 1 || strings.Contains(out, "callout:") || !strings.Contains(out, "Text after.") {
		t.Errorf("del-step: %s", out)
	}
	// a plain fence gets an id when it grows a second step
	plain := "```go title=\"main.go\"\nfunc main() {}\n```\n"
	out, _, err := Apply(plain, 1, Op{Kind: "add-step"})
	if err != nil || strings.Count(out, "id=main-go") != 2 {
		t.Errorf("id: %v %s", err, out)
	}
}

// TestIncludeSteps: includes (and <!-- code: -->) in a window are steps
// whose code is a file; options edit in place, the code does not.
func TestIncludeSteps(t *testing.T) {
	text := "# T\n\n<!-- include: v1.sh numbers id=up window=terminal -->\n<!-- callout: 1 once -->\n\n<!-- include: ../v2.sh id=up window=terminal -->\n\n<!-- include: plain.go -->\n\n<!-- code: https://github.com/a/b/blob/0123abc/x.yaml window=mac -->\n"
	shows := Shows(text)
	if len(shows) != 2 || len(shows[0].Steps) != 2 || shows[0].Steps[0].File != "v1.sh" || shows[0].Steps[0].Lang != "bash" || len(shows[0].Steps[0].Callouts) != 1 {
		t.Fatalf("shows: %+v", shows)
	}
	if _, n := shows[0].Steps[0].Attrs["numbers"]; !n || shows[1].Steps[0].File != "https://github.com/a/b/blob/0123abc/x.yaml" {
		t.Errorf("a first bare option, the code source: %+v", shows)
	}
	out, _, err := Apply(text, 3, Op{Kind: "set", Field: "theme", Value: "night"})
	if err != nil || !strings.Contains(out, "<!-- include: v1.sh numbers id=up window=terminal theme=night -->") || !strings.Contains(out, "<!-- include: ../v2.sh id=up window=terminal theme=night -->") {
		t.Fatalf("set: %v\n%s", err, out)
	}
	out, _, err = Apply(text, 3, Op{Kind: "add-step", Step: 1})
	if err != nil || strings.Count(out, "../v2.sh") != 2 {
		t.Errorf("add-step: %v\n%s", err, out)
	}
	if _, _, err := Apply(text, 3, Op{Kind: "set-code", Text: "x"}); err == nil {
		t.Error("an include's code is edited in its file")
	}
	if src, n := Source(text, nil, ""); n != 2 || !strings.Contains(src, "<!-- include: ../v2.sh") || strings.Contains(src, "plain.go") {
		t.Errorf("source: %d\n%s", n, src)
	}
}

func TestSource(t *testing.T) {
	text := "# T\n\n---\n\n## A\n\n```go id=r window=mac\na()\n```\n<!-- callout: 1 one -->\n\n---\n\n## B\n\n```go id=r window=mac\na()\nb()\n```\n\n```sh window=terminal\nls\n```\n\n```go\nplain()\n```\n"
	src, n := Source(text, map[string]string{"theme": "paper"}, "")
	if n != 2 || strings.Count(src, "id=r") != 2 || !strings.Contains(src, "callout: 1 one") || strings.Contains(src, "plain()") || !strings.Contains(src, "theme: paper") {
		t.Fatalf("%d\n%s", n, src)
	}
	if _, n := Source(text, nil, "r"); n != 1 {
		t.Errorf("only r: %d shows", n)
	}
}

func TestAttrsForms(t *testing.T) {
	for _, c := range []struct{ info, lang, title string }{
		{`xml title="config.csproj"`, "xml", "config.csproj"},
		{` title="config.csproj"`, "xml", "config.csproj"},
		{`title=main.go`, "go", "main.go"},
		{`{.xml title="config.csproj"}`, "xml", "config.csproj"},
		{`xml {title="config.csproj"}`, "xml", "config.csproj"},
		{`go`, "go", ""},
		{`title="notes"`, "", "notes"},
		{`title="Dockerfile"`, "dockerfile", "Dockerfile"},
	} {
		lang, vals := Info(c.info)
		if lang != c.lang || vals["title"] != c.title {
			t.Errorf("%q: lang %q title %q, want %q %q", c.info, lang, vals["title"], c.lang, c.title)
		}
	}
}
