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
