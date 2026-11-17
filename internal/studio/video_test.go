package studio

import (
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/videdit"
)

// The Video view: the preview drawn from the editor's text with the edit
// hooks and a base for its relative files; a drag applied to its line; the
// edit checked as a video edit, not as a deck.
func TestVideoView(t *testing.T) {
	f := newFx(t)
	text := "---\ntype: video\nsource: clip.mp4\nout: final.mp4\n---\n# Clip\n\n- 0:01 · box 10,10 50x50\n"
	f.put("vid/edit.md", text)
	code, m := f.do("POST", "video", videoReq{Path: "vid/edit.md", Text: text})
	if code != 200 || m["out"] != "vid/final.mp4" || !strings.Contains(m["html"].(string), `<base href="/vid/">`) || !strings.Contains(m["html"].(string), ".ve-grip") {
		t.Fatalf("%d %v", code, m["out"])
	}
	_, m = f.do("POST", "video", videoReq{Path: "vid/edit.md", Text: text, Move: &videdit.Move{Line: 8, Rect: &[4]int{1, 2, 3, 4}}})
	if !strings.Contains(m["text"].(string), "- 0:01 · box 1,2 3x4\n") {
		t.Errorf("move: %v", m["text"])
	}
	if code, _ = f.do("POST", "video", videoReq{Path: "vid/edit.md", Text: text, Move: &videdit.Move{Line: 6, At: &[2]int{1, 1}}}); code != 422 {
		t.Errorf("a move on a heading: %d", code)
	}
	code, m = f.do("GET", "check?deck=vid/edit.md", nil)
	if code != 200 || m["kind"] != "video" || !strings.Contains(m["problems"].([]any)[0].(map[string]any)["msg"].(string), "missing source clip.mp4") {
		t.Errorf("check: %d %v", code, m)
	}
	if code, _ = f.do("POST", "video/build", map[string]string{"path": "vid/edit.md"}); code != 501 {
		t.Errorf("build without the CLI: %d", code)
	}
}
