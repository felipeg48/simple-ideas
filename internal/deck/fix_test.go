package deck

import (
	"strings"
	"testing"
)

func TestFixComponents(t *testing.T) {
	src := "# Cover\n\n<div class=\"lockup\"><span class=\"lk-logo\" role=\"img\" aria-label=\"AceMQ\"></span><span class=\"lk-x\"></span><span class=\"lk-name\">CACI</span></div>\n" +
		"<p class=\"pill\"><span class=\"pill-mark\"></span>Prepared for CACI · Session 1 of 5</p>\n\n" +
		"## HA is <span class=\"hl\">not DR</span>\n\n" +
		"<div class=\"cards c2\">\n" +
		"<div class=\"card teal\"><span class=\"tag teal\">In seconds</span><h4>High availability</h4><p>One cluster. <b>Recovers by itself.</b></p></div>\n" +
		"<div class=\"card\"><span class=\"tag\">RPO</span><h4>How much can we lose?</h4><p>Per flow.</p><p><b>CACI value: ___</b></p></div>\n" +
		"</div>\n\n" +
		"<div class=\"x\"><span class=\"hl\">raw stays</span></div>\n\n" +
		"```html\n<span class=\"hl\">code stays</span>\n```\n"
	got, n := FixComponents(src)
	for _, want := range []string{
		"<!-- logo: × CACI · alt=AceMQ -->\n<!-- badge: Prepared for CACI · Session 1 of 5 -->",
		"## HA is ==not DR==",
		"<!-- cards: 2 -->\n\n### High availability · In seconds · teal\n\nOne cluster. **Recovers by itself.**\n\n### How much can we lose? · RPO\n\nPer flow.\n\n**CACI value: ___**\n\n<div class",
		`<span class="hl">raw stays</span>`,
		`<span class="hl">code stays</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q in\n%s", want, got)
		}
	}
	if n["highlight"] != 1 || n["badge"] != 1 || n["logo"] != 1 || n["cards"] != 1 {
		t.Errorf("counts %v", n)
	}
	if again, m := FixComponents(got); again != got || len(m) != 0 {
		t.Errorf("a second run changes nothing: %v", m)
	}
}

func TestFixLeavesRichCardsAlone(t *testing.T) {
	src := "<div class=\"cards\">\n<div class=\"card\"><h4>A</h4><p>See <a href=\"x\">this</a>.</p></div>\n</div>\n"
	if got, n := FixComponents(src); got != src || n["cards"] != 0 {
		t.Errorf("a card with a link stays HTML:\n%s", got)
	}
}
