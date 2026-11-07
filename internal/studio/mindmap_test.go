package studio

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/mindmap"
)

func TestMindmapEdit(t *testing.T) {
	f := newFx(t)
	page := "---\ntype: mindmap\ntitle: Plan\n---\n# Plan\n\n- One\n"
	f.put("plan.md", page)
	code, m := f.do("POST", "mindmap", mindmapReq{Path: "plan.md", Text: page, Op: &mindmap.Op{Kind: "add-child", Path: []int{0}, Text: "Two"}})
	if code != 200 {
		t.Fatalf("%d %v", code, m)
	}
	if m["text"] != "---\ntype: mindmap\ntitle: Plan\n---\n# Plan\n\n- One\n  - Two\n" {
		t.Fatalf("text: %q", m["text"])
	}
	if !strings.Contains(string(mustJSON(m["data"])), `"t":"Two"`) || m["front"].(map[string]any)["title"] != "Plan" {
		t.Fatalf("data: %v front: %v", m["data"], m["front"])
	}
	if s := m["select"].(map[string]any); s["path"].([]any)[1].(float64) != 0 {
		t.Fatalf("select: %v", s)
	}

	deckText := "# Talk\n\n```mindmap layout=right\n# Plan\n- One\n```\n"
	f.put("talk.md", deckText)
	code, m = f.do("POST", "mindmap", mindmapReq{Path: "talk.md", Text: deckText, Line: 4, Op: &mindmap.Op{Kind: "meta", Field: "lines", Value: "tapered"}})
	if code != 200 || m["text"] != "# Talk\n\n```mindmap layout=right lines=tapered\n# Plan\n- One\n```\n" || m["fenceLine"].(float64) != 3 {
		t.Fatalf("%d %q %v", code, m["text"], m["fenceLine"])
	}
	if code, _ = f.do("POST", "mindmap", mindmapReq{Path: "talk.md", Text: deckText, Line: 1}); code != 422 {
		t.Fatalf("line outside a fence: %d", code)
	}
	if code, m = f.do("POST", "mindmap", mindmapReq{Path: "../x.md", Text: ""}); code != 403 {
		t.Fatalf("outside the workspace: %d %v", code, m)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestMindmapCopy(t *testing.T) {
	f := newFx(t)
	page := "---\ntype: mindmap\ntitle: Plan\n---\n# Plan\n\n- One\n  - Two\n- Three\n"
	f.put("plan.md", page)
	code, m := f.do("POST", "mindmap", mindmapReq{Path: "plan.md", Text: page, Op: &mindmap.Op{Kind: "copy", Paths: [][]int{{0}, {1}}}})
	if code != 200 || m["copy"] != "- One\n  - Two\n- Three\n" || m["text"] != nil {
		t.Fatalf("copy: %d %v", code, m)
	}
}
