package studio

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestCheckAllChecksDecksAndGuidesWhole(t *testing.T) {
	f := siteFx(t)
	f.put("loose/talk.md", "# Talk\n\n<!-- layout: nope -->\n")
	_, m := f.do("GET", "checkall", nil)
	ps := fmt.Sprint(m["problems"])
	for _, want := range []string{"post.md#gone", "loose/talk.md", "nope"} {
		if !strings.Contains(ps, want) {
			t.Errorf("checkall lacks %q: %s", want, ps)
		}
	}
	if strings.Count(ps, "post.md#gone") != 1 {
		t.Errorf("a guide page is checked once, with its guide: %s", ps)
	}
	if m["checked"].(float64) != 5 {
		t.Errorf("checked %v", m["checked"])
	}
}

func TestBuildRunsTheCLI(t *testing.T) {
	f := newFx(t)
	var got []string
	f.api.run = func(args []string, out io.Writer) int { got = args; fmt.Fprint(out, "si: built"); return 0 }
	code, m := f.do("POST", "build", map[string]bool{"pdf": true})
	if code != 200 || m["output"] != "si: built" || m["code"].(float64) != 0 {
		t.Fatalf("%d %v", code, m)
	}
	if a := strings.Join(got, " "); !strings.HasPrefix(a, "build "+f.root+" -o "+f.root+"/html") || !strings.HasSuffix(a, "--pdf") {
		t.Errorf("args %q", a)
	}
}
