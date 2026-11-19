package render

import (
	"strings"
	"testing"
)

// A document plays animate: blocks and the steps: list modes when scrolled
// into view (web/build.js); a reorder list is written in its new order.
func TestDocumentMotion(t *testing.T) {
	res := chartMD(t, "---\ntype: article\n---\n# Page\n\n<!-- animate: count -->\n1,842 declines.\n\n<!-- steps: reorder -->\n- Low 3\n- High 9\n- Mid 5\n\n<!-- animate: orbit -->\n- Order\n- Shop\n- Bank\n\n<!-- steps: deal -->\n- a\n- b\n")
	page := string(res.HTML)
	if len(res.Problems)+len(res.Warnings) != 0 {
		t.Fatalf("problems %v warnings %v", res.Problems, res.Warnings)
	}
	for _, want := range []string{`data-si-anim="count"`, `class="si-steps-reorder"`, `class="si-orbit" data-si-anim="orbit"`, `class="si-steps-deal"`, "window.siBuild", ".si-steps-deal>.si-step"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	hi, mid, lo := strings.Index(page, "High 9"), strings.Index(page, "Mid 5"), strings.Index(page, "Low 3")
	if hi > mid || mid > lo {
		t.Errorf("reorder list not written in its new order: high %d mid %d low %d", hi, mid, lo)
	}
	// a page without one carries none of it
	plain := string(chartMD(t, "---\ntype: article\n---\n# Page\n\ntext\n").HTML)
	if strings.Contains(plain, "window.siBuild") {
		t.Error("a document without animations carries build.js")
	}
}

// An infographic block takes the 3D charts of a ```chart fence.
func TestInfographic3D(t *testing.T) {
	res := chartMD(t, "---\ntype: infographic\ntitle: T\n---\n\n## Share · chart=donut3d\n| D | % |\n|---|---|\n| Phones | 64 |\n| Desktops | 36 |\n\n## Steps · chart=funnel3d\n| S | N |\n|---|---|\n| Visits | 100 |\n| Paid | 15 |\n")
	page := string(res.HTML)
	if len(res.Problems) != 0 {
		t.Fatalf("problems %v", res.Problems)
	}
	for _, want := range []string{`data-c3="donut3d"`, `data-c3="funnel3d"`, ".si-c3{"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}
