package studio

import (
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/motion"
)

func TestMotionEdit(t *testing.T) {
	f := newFx(t)
	type req struct {
		Path string     `json:"path"`
		Text string     `json:"text"`
		Op   *motion.Op `json:"op"`
	}
	code, m := f.do("POST", "motion", req{Path: "anim/a.svg", Text: "", Op: &motion.Op{Kind: "blank"}})
	if code != 200 || !strings.Contains(m["text"].(string), "/* si motion: frame1 */") || m["snippet"] != "<!-- svg: anim/a.svg frame1 -->\n" {
		t.Fatalf("%d %v", code, m)
	}
	_, m = f.do("POST", "motion", req{Path: "anim/a.svg", Text: m["text"].(string), Op: &motion.Op{Kind: "add-frame"}})
	if !strings.Contains(m["snippet"].(string), "<!-- step: +frame2 -->") || len(m["model"].(map[string]any)["frames"].([]any)) != 2 {
		t.Errorf("add-frame: %v", m)
	}
	if code, _ = f.do("POST", "motion", req{Path: "anim/a.svg", Text: "x", Op: &motion.Op{Kind: "add-frame"}}); code != 422 {
		t.Errorf("not an svg: %d", code)
	}
	if code, _ = f.do("POST", "motion", req{Path: "../a.svg", Text: ""}); code != 403 {
		t.Errorf("outside the workspace: %d", code)
	}
}
