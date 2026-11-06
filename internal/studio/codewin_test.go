package studio

import (
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
)

func TestCodewinEdit(t *testing.T) {
	f := newFx(t)
	text := "# Code\n\n```go id=r window=mac\nfunc a() {}\n```\n<!-- callout: 1 one -->\n\n```go id=r window=mac\nfunc a() {}\nfunc b() {}\n```\n"
	f.put("c.md", text)
	code, m := f.do("POST", "codewin", codewinReq{Path: "c.md", Text: text, Line: 3})
	if code != 200 || !strings.Contains(m["html"].(string), `class="si-cw"`) || len(m["show"].(map[string]any)["steps"].([]any)) != 2 {
		t.Fatalf("%d %v", code, m["show"])
	}
	_, m = f.do("POST", "codewin", codewinReq{Path: "c.md", Text: text, Line: 3, Op: &codeshow.Op{Kind: "set", Field: "window", Value: "windows"}})
	if strings.Count(m["text"].(string), "window=windows") != 2 {
		t.Errorf("set: %v", m["text"])
	}
	_, m = f.do("POST", "codewin", codewinReq{Path: "c.md", Text: text, Line: 6, Op: &codeshow.Op{Kind: "move-callout", At: []int{40, -20}}})
	if !strings.Contains(m["text"].(string), "<!-- callout: 1 one at=40,-20 -->") {
		t.Errorf("move: %v", m["text"])
	}
	if code, _ = f.do("POST", "codewin", codewinReq{Path: "c.md", Text: text, Line: 1}); code != 422 {
		t.Errorf("a line outside a block: %d", code)
	}
	if code, _ = f.do("POST", "codewin", codewinReq{Path: "../x.md", Text: text, Line: 3}); code != 403 {
		t.Errorf("outside the workspace: %d", code)
	}
	if code, _ = f.do("POST", "codewin/export", map[string]string{"path": "c.md", "format": "gif"}); code != 501 {
		t.Errorf("export without the CLI: %d", code)
	}
}
