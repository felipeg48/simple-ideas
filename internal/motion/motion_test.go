package motion

import (
	"strings"
	"testing"
)

// A hand-written canvas survives open, edit one frame, save: its own CSS and
// shapes stay byte for byte, and the frames read back as written.
func TestRoundTripKeepsHandWrittenSVG(t *testing.T) {
	hand := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 560" class="j">
<style>
.j .card{fill:var(--si-surface,#fff)}
.j[data-state~="paid"] .token{transform:translateX(235px)}
</style>
<rect class="card" x="10" y="10" width="100" height="60"/>
<g class="token"><rect x="0" y="0" width="40" height="20"/><text x="5" y="15">#4711</text></g>
</svg>
`
	m := Read(hand)
	if m.ViewBox != "0 0 1200 560" || len(m.Frames) != 1 || len(m.Parts) != 4 || m.Parts[3].Label != "#4711" {
		t.Fatalf("read: %+v", m)
	}
	svg, err := Apply(hand, Op{Kind: "name", Index: 1, Name: "token"})
	must(t, err)
	svg, err = Apply(svg, Op{Kind: "add-frame", Frame: 0})
	must(t, err)
	svg, err = Apply(svg, Op{Kind: "look", Frame: 1, Part: "token", Look: &Look{X: 310, Y: -12.5, Scale: 1.2, Fill: "var(--si-ok)"}})
	must(t, err)
	svg, err = Apply(svg, Op{Kind: "timing", Frame: 1, Dur: 0.8, Ease: "spring"})
	must(t, err)
	svg, err = Apply(svg, Op{Kind: "add-frame", Frame: 1})
	must(t, err)
	svg, err = Apply(svg, Op{Kind: "look", Frame: 2, Part: "token", Look: &Look{Hidden: true}})
	must(t, err)
	for _, keep := range []string{".j .card{fill:var(--si-surface,#fff)}", `.j[data-state~="paid"] .token{transform:translateX(235px)}`,
		`<rect class="card" x="10" y="10" width="100" height="60"/>`, `<g data-part="token" class="token">`} {
		if !strings.Contains(svg, keep) {
			t.Errorf("lost %q:\n%s", keep, svg)
		}
	}
	for _, want := range []string{`[data-state~="frame2"] [data-part="token"]{transform:translate(310px,-12.5px) scale(1.2);fill:var(--si-ok)}`,
		`[data-state~="frame2"] *{transition-duration:0.8s;transition-timing-function:cubic-bezier(.3,1.5,.5,1)}`,
		`[data-state~="frame3"] [data-part="token"]{opacity:0}`} {
		if !strings.Contains(svg, want) {
			t.Errorf("missing %q:\n%s", want, svg)
		}
	}
	m = Read(svg)
	if len(m.Frames) != 3 || m.Frames[1].Parts["token"] != (Look{X: 310, Y: -12.5, Scale: 1.2, Fill: "var(--si-ok)"}) ||
		m.Frames[1].Dur != 0.8 || m.Frames[1].Ease != "spring" || !m.Frames[2].Parts["token"].Hidden {
		t.Errorf("read back: %+v", m.Frames)
	}
	// writing the same model again changes nothing
	if again := write(svg, Read(svg)); again != svg {
		t.Errorf("not stable:\n%s\n---\n%s", svg, again)
	}
	// deleting the middle frame renames the rest in order
	svg, err = Apply(svg, Op{Kind: "del-frame", Frame: 1})
	must(t, err)
	if m = Read(svg); len(m.Frames) != 2 || m.Frames[1].Name != "frame2" || !m.Frames[1].Parts["token"].Hidden {
		t.Errorf("after delete: %+v", m.Frames)
	}
	if got := Snippet("anim/j.svg", Read(svg)); got != "<!-- svg: anim/j.svg frame1 -->\n<!-- step: +frame2 -->\n" {
		t.Errorf("snippet: %q", got)
	}
}

func TestBlankShapesAndErrors(t *testing.T) {
	svg, err := Apply("", Op{Kind: "blank"})
	must(t, err)
	for _, s := range []string{"rect", "circle", "text", "line"} {
		svg, err = Apply(svg, Op{Kind: "add-shape", Shape: s})
		must(t, err)
	}
	m := Read(svg)
	if len(m.Parts) != 4 || m.Parts[0].Name != "rect1" || m.Parts[2].Label != "Text" {
		t.Fatalf("parts: %+v", m.Parts)
	}
	svg, err = Apply(svg, Op{Kind: "del-shape", Index: 2})
	must(t, err)
	if strings.Contains(svg, "<text") || len(Read(svg).Parts) != 3 {
		t.Errorf("del-shape:\n%s", svg)
	}
	svg, _ = Apply(svg, Op{Kind: "add-frame"})
	snip := Snippet("a.svg", Read(svg))
	svg, _ = Apply(svg, Op{Kind: "add-frame", Frame: 1})
	if snip = Snippet("a.svg", Read(svg)); !strings.Contains(snip, "<!-- step: -frame2 +frame3 -->") {
		t.Errorf("snippet: %s", snip)
	}
	for _, bad := range []Op{{Kind: "blank"}, {Kind: "look", Part: "nope", Look: &Look{}}, {Kind: "look", Part: "rect1", Look: &Look{Fill: "red;}"}},
		{Kind: "name", Index: 0, Name: "circle1"}, {Kind: "name", Index: 0, Name: "1x"}, {Kind: "timing", Ease: "wobble"}, {Kind: "del-frame", Frame: 9}, {Kind: "zap"}} {
		if _, err := Apply(svg, bad); err == nil {
			t.Errorf("%+v: no error", bad)
		}
	}
	if _, err := Apply("not svg", Op{Kind: "add-frame"}); err == nil {
		t.Error("not an svg: no error")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
