package webdemo

import (
	"os"
	"strings"
	"testing"
	"time"
)

const sample = `---
type: demo
title: Shop
app: http://localhost:3000   # where it runs
viewport: phone
timeout: 2s
setup: make seed
---

# Shop

## Pay for an order

What it does: a customer pays
and sees "Paid".

Say: Sign in with the demo account.
- go /login
- note "Username: demo" at "Username" · for 3s · position right
- type "demo" into "Username"
- type $WEBDEMO_TEST_UNSET into "Password"
- click "Sign in"

Notes: presenter text, not steps
- click "never"

Say: Open the order.
  It is waiting.
1. click role=button "Pay"
2. expect "Confirm" disabled
3. press cmd+k

` + "```text\n- click \"in a fence\"\n```\n" + `
## Empty
`

func TestParseMarkdown(t *testing.T) {
	s := Parse("/x/shop.md", sample)
	if len(s.Problems) > 0 {
		t.Fatalf("problems: %v", s.Problems)
	}
	if s.App != "http://localhost:3000" || s.W != 400 || s.H != 800 || s.Timeout != 2*time.Second || s.Setup != "make seed" {
		t.Errorf("settings: %+v", s)
	}
	if len(s.Features) != 2 {
		t.Fatalf("features: %d", len(s.Features))
	}
	f := s.Features[0]
	if f.Name != "Pay for an order" || f.What != `a customer pays and sees "Paid".` || f.Line != 12 {
		t.Errorf("feature: %+v", f)
	}
	if len(f.Blocks) != 2 || len(f.Blocks[0].Steps) != 5 || len(f.Blocks[1].Steps) != 3 {
		t.Fatalf("blocks: %+v", f.Blocks)
	}
	if f.Blocks[1].Say != "Open the order. It is waiting." {
		t.Errorf("say: %q", f.Blocks[1].Say)
	}
	n := f.Blocks[0].Steps[1]
	if n.Kind != "note" || n.Value.Text != "Username: demo" || n.Target.Name != "Username" || n.For != 3*time.Second || n.Pos != "right" || n.Line != 19 {
		t.Errorf("note: %+v", n)
	}
	if v := f.Blocks[0].Steps[3].Value; v.Var != "WEBDEMO_TEST_UNSET" || v.String() != "$WEBDEMO_TEST_UNSET" {
		t.Errorf("var: %+v", v)
	}
	if st := f.Blocks[1].Steps[0]; st.Target.Role != "button" || st.Target.Name != "Pay" {
		t.Errorf("role: %+v", st)
	}
	var warns []string
	for _, w := range s.Warnings {
		warns = append(warns, w.String())
	}
	all := strings.Join(warns, "\n")
	if !strings.Contains(all, ":21: $WEBDEMO_TEST_UNSET is not set") || !strings.Contains(all, `feature "Empty" has no steps`) {
		t.Errorf("warnings: %s", all)
	}
}

func TestParseProblemsHaveLines(t *testing.T) {
	s := Parse("/x/bad.md", "---\ntype: demo\napp: app\nviewport: huge\n---\n\n- go /\n\n## A\n\n- clik \"Pay\"\n- type \"x\" in \"Field\"\n")
	want := []string{":4: viewport \"huge\"", ":7: a step before the first feature", ":11: unknown step \"clik \\\"Pay\\\"\"", ":12: "}
	if len(s.Problems) != len(want) {
		t.Fatalf("problems: %v", s.Problems)
	}
	for i, w := range want {
		if !strings.Contains(s.Problems[i].String(), w) {
			t.Errorf("problem %d = %q, want %q", i, s.Problems[i], w)
		}
	}
	if s := Parse("/x/none.md", "---\ntype: demo\n---\n# x\n"); len(s.Problems) != 2 {
		t.Errorf("no app, no features: %v", s.Problems)
	}
}

func TestParseStep(t *testing.T) {
	good := map[string]string{
		`go /orders/4711`:                     "go",
		`go https://example.com/x`:            "go",
		`click #pay`:                          "click",
		`click [data-testid=pay]`:             "click",
		`hover "Help"`:                        "hover",
		`check "Gift wrap"`:                   "check",
		`uncheck "Gift wrap"`:                 "uncheck",
		`scroll to "Footer"`:                  "scroll",
		`type "a \"b\"" into "Note"`:          "type",
		`select "Express" in "Shipping"`:      "select",
		`press Enter`:                         "press",
		`press ctrl+shift+p`:                  "press",
		`wait 1s`:                             "wait",
		`wait 250ms`:                          "wait",
		`wait for "Saved"`:                    "waitfor",
		`note "Hi"`:                           "note",
		`expect text "Paid"`:                  "expect-text",
		`expect url /orders/4711/receipt`:     "expect-url",
		`expect "Pay" disabled`:               "expect-state",
		`expect role=link "Home" visible`:     "expect-state",
		`expect no errors`:                    "expect-errors",
		`screenshot receipt`:                  "screenshot",
		`type $SHOP_PASSWORD into "Password"`: "type",
	}
	for in, kind := range good {
		st, err := ParseStep(in)
		if err != nil || st.Kind != kind {
			t.Errorf("%s: %v %+v", in, err, st)
		}
	}
	if st, _ := ParseStep(`type "a \"b\"" into "Note"`); st.Value.Text != `a "b"` {
		t.Errorf("escaped quote: %q", st.Value.Text)
	}
	bad := []string{`go`, `click Pay`, `type "x" "Field"`, `press hyper+k`, `press Banana`, `wait soon`, `expect "Pay" shiny`,
		`expect text Paid`, `scroll "Footer"`, `note "x" at "y" · colour red`, `screenshot`, `dance`, `expect no problems`, `click "unclosed`}
	for _, in := range bad {
		if _, err := ParseStep(in); err == nil {
			t.Errorf("%s: no error", in)
		}
	}
}

func TestParseJSON(t *testing.T) {
	src := `{
  "type": "demo",
  "title": "Shop",
  "app": "app",
  "viewport": "1280x720",
  "features": [
    {"name": "Sign in", "what": "a customer signs in",
     "blocks": [
       {"say": "Sign in.", "steps": [
         "go /",
         "click \"Sign in\"",
         "clack \"Sign in\""
       ]}
     ]}
  ]
}`
	s, err := ParseJSON("/x/shop.json", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if s.W != 1280 || s.H != 720 || len(s.Features) != 1 || len(s.Features[0].Blocks[0].Steps) != 2 {
		t.Fatalf("parsed: %+v", s)
	}
	if st := s.Features[0].Blocks[0].Steps[1]; st.Kind != "click" || st.Line != 11 {
		t.Errorf("step: %+v", st)
	}
	if len(s.Problems) != 1 || s.Problems[0].Line != 12 {
		t.Errorf("problems: %v", s.Problems)
	}
	if _, err := ParseJSON("/x/bad.json", []byte("{")); err == nil {
		t.Error("bad JSON: no error")
	}
}

func TestIsWebDemo(t *testing.T) {
	if !IsWebDemo("---\ntype: demo\napp: http://x\n---\n") || IsWebDemo("---\ntype: demo\n---\n") {
		t.Error("IsWebDemo")
	}
}

func TestURLMatches(t *testing.T) {
	cases := []struct {
		href, want string
		ok         bool
	}{
		{"http://h/orders/4711/receipt", "/orders/4711/receipt", true},
		{"http://h/orders/4711/receipt?x=1", "/orders/4711/receipt", true},
		{"http://h/receipt.html?order=4711", "/receipt.html?order=4711", true},
		{"http://h/orders/47111", "/orders/4711", false},
		{"http://h/a#b", "/a#b", true},
		{"https://e.com/x", "https://e.com/x", true},
		{"https://e.com/xy", "https://e.com/x", false},
	}
	for _, c := range cases {
		if urlMatches(c.href, c.want) != c.ok {
			t.Errorf("urlMatches(%q, %q) != %v", c.href, c.want, c.ok)
		}
	}
}

func TestParseAsciiDoc(t *testing.T) {
	p := t.TempDir() + "/shop.adoc"
	if err := os.WriteFile(p, []byte("= Shop\n:type: demo\n:app: http://localhost:1\n:viewport: tablet\n\n== Sign in\n\n* go /\n* click \"Sign in\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(p)
	if err != nil || len(s.Problems) > 0 {
		t.Fatalf("%v %v", err, s.Problems)
	}
	if s.App != "http://localhost:1" || s.W != 820 || len(s.Features) != 1 || len(s.Features[0].Blocks[0].Steps) != 2 {
		t.Errorf("adoc: %+v", s)
	}
}
