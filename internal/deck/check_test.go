package deck

import (
	"strings"
	"testing"
)

func TestParseCheck(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Check
		err  string
	}{
		{`curl -fsS localhost:8080/health`, Check{Cmd: "curl -fsS localhost:8080/health"}, ""},
		{`ls · expect "index.html"`, Check{Cmd: "ls", Expect: "index.html"}, ""},
		{`grep -q x f · exit 1 · expect "ok"`, Check{Cmd: "grep -q x f", Exit: 1, Expect: "ok"}, ""},
		{`test -f a · exit=0`, Check{Cmd: "test -f a"}, ""},
		{``, Check{}, "needs a command"},
		{`ls · soon`, Check{}, `"soon" is not expect`},
	} {
		got, err := ParseCheck(tc.in)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%q: want error %q, got %v", tc.in, tc.err, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("%q: got %+v %v, want %+v", tc.in, got, err, tc.want)
		}
	}
}

// TestChecksOfAPage: the page's checks in order, fences skipped, includes in.
func TestChecksOfAPage(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"lab.md":  "---\ntype: lab\n---\n# Lab\n\n## Step 1\n\n<!-- check: echo one -->\n\n```md\n<!-- check: echo fenced -->\n```\n\n<!-- include: more.md -->\n",
		"more.md": "## Step 2\n\n<!-- check: echo two · expect \"two\" -->\n",
	})
	d, err := Parse(dir + "/lab.md")
	if err != nil {
		t.Fatal(err)
	}
	cs := d.Checks()
	if len(cs) != 2 || cs[0].Cmd != "echo one" || cs[1].Expect != "two" {
		t.Errorf("checks: %+v (problems %v)", cs, d.Problems)
	}
}

// TestCourseWebhookWarnings: si check warns about a webhook that is not
// https, unknown formats, events and values, and webhook keys in a lesson.
func TestCourseWebhookWarnings(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_course.md": "---\ntitle: C\nwebhook: http://hooks.example.com/x\nwebhook-format: pigeon\nwebhook-events: quiz.submitted, lesson.started\n" +
			"webhook-learner: anon\nwebhook-consent: maybe\nwebhook-auth: Basic x\n---\n- Module 1: M\n  - [A](a.md)\n",
		"a.md": "---\nwebhook: https://elsewhere.example.com\n---\n# A\n",
	})
	g, err := LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, w := range g.Warnings {
		msgs = append(msgs, w.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{`not https://`, `"pigeon" is not one of json, slack`, `"lesson.started" is not one of`, `"anon" is not random or ask`,
		`"maybe" is not notice or required`, "webhook-auth: only xapi"} {
		if !strings.Contains(all, want) {
			t.Errorf("want a warning %q, got:\n%s", want, all)
		}
	}
	if len(g.Problems) != 0 {
		t.Errorf("webhook settings warn, never fail: %v", g.Problems)
	}
	d, _ := ParseWith(dir+"/a.md", g.Defaults())
	if len(d.Warnings) != 1 || !strings.Contains(d.Warnings[0].Msg, "belongs in the course's _course.md") {
		t.Errorf("a lesson's own webhook: warns: %v", d.Warnings)
	}
	ok := writeTree(t, map[string]string{"_course.md": "---\ntitle: C\nwebhook: relay\nwebhook-format: xapi\nwebhook-auth: Basic x\n---\n- Module 1: M\n  - [A](a.md)\n", "a.md": "# A\n"})
	if g, _ := LoadGuide(ok); len(g.Warnings) != 0 || g.Meta.Webhook != "relay" {
		t.Errorf("relay and xapi with auth are fine: %v", g.Warnings)
	}
}
