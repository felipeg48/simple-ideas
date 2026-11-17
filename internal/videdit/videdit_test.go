package videdit

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

var update = flag.Bool("update", false, "rewrite the golden files")

const head = "---\ntype: video\nsource: demo.mp4\n---\n# T\n\n## Timeline\n\n"

func parseLines(t *testing.T, lines ...string) *Edit {
	t.Helper()
	return Parse("/x/edit.md", head+"- "+strings.Join(lines, "\n- ")+"\n")
}

func TestParseEveryAction(t *testing.T) {
	cases := []struct {
		line, action string
		check        func(*Item) bool
	}{
		{"1:36–1:46 · cut", "cut", func(i *Item) bool { return i.From.Sec == 96 && i.To.Sec == 106 }},
		{"1:36-1:46 · cut", "cut", func(i *Item) bool { return i.To.Sec == 106 }},
		{"1:36 to 1:46.5 · mute", "mute", func(i *Item) bool { return i.To.Sec == 106.5 }},
		{"0:03 · trim start", "trim-start", nil},
		{"end · trim end", "trim-end", func(i *Item) bool { return i.From.End }},
		{"1:52 · freeze 2s", "freeze", func(i *Item) bool { return i.Dur == 2 }},
		{"2:10–2:30 · speed 2x · badge \"⏩ 2×\" · mute", "speed", func(i *Item) bool { return i.Speed == 2 && i.Badge == "⏩ 2×" && i.Flags["mute"] }},
		{"1:36–1:46 · replace with fixes/curl-ok.mp4 · fade 0.4s · keep audio", "replace", func(i *Item) bool {
			return i.File == "fixes/curl-ok.mp4" && i.Trans.Kind == "fade" && i.Trans.Dur == 0.4 && i.Flags["keep-audio"]
		}},
		{"0:10 · insert clip intro.mp4 · dissolve", "insert-clip", func(i *Item) bool { return i.File == "intro.mp4" && i.Trans.Kind == "dissolve" }},
		{"0:10 · insert image shot.png · 3s · wipe", "insert-image", func(i *Item) bool { return i.Dur == 3 && i.Trans.Kind == "wipe-left" }},
		{"0:10 · insert card \"Part two\" · 2s", "card", func(i *Item) bool { return i.Str == "Part two" && i.Dur == 2 }},
		{"end · card \"Next: clustering\" · 3s · slide", "card", func(i *Item) bool { return i.Trans.Kind == "slide-left" }},
		{"0:00 · title \"Install a single node\" · 3s · fade", "title", func(i *Item) bool { return i.Str == "Install a single node" && i.Enter == "fade" }},
		{"0:05 · text \"Hello\" · at top-right · font Inter · size 48 · color accent · box · shadow · type · out none", "text", func(i *Item) bool {
			return i.At.Word == "top-right" && i.Font == "Inter" && i.Size == 48 && i.Color == "accent" && i.Flags["box"] && i.Flags["shadow"] && i.Enter == "type" && i.Exit == "none"
		}},
		{"1:34 · bubble \"Log in\" · to 820,410 · 4s · pop · round", "bubble", func(i *Item) bool { return i.Point.X.V == 820 && i.Dur == 4 && i.Enter == "pop" && i.Flags["round"] }},
		{"1:34 · bubble \"Hmm\" · to 60%,40% · think", "bubble", func(i *Item) bool { return i.Point.X.Pct && i.Point.Y.V == 40 && i.Flags["think"] }},
		{"3:05 · arrow from 200,700 to 640,420 · \"This queue\" · 3s", "arrow", func(i *Item) bool {
			return i.Tail.X.V == 200 && i.Point.Y.V == 420 && i.Str == "This queue" && i.Enter == "draw"
		}},
		{"0:05 · box 120,80 400x60", "box", func(i *Item) bool { return i.Rect.W.V == 400 && i.Dur == 3 }},
		{"0:05 · circle 10%,10% 20%x20% · slide-in", "circle", func(i *Item) bool { return i.Rect.W.Pct && i.Enter == "slide-in" }},
		{"0:05 · underline 120,80,400x8", "underline", func(i *Item) bool { return i.Rect.H.V == 8 }},
		{"0:05–0:09 · spotlight 120,80 400x300", "spotlight", nil},
		{"0:45–0:50 · blur 120,80,400x60", "blur", func(i *Item) bool { return i.Rect.X.V == 120 }},
		{"0:45 · pixelate 120,80 400x60 · 5s", "pixelate", func(i *Item) bool { return i.Dur == 5 }},
		{"3:20 · keys \"⌘ K\" · 1.5s", "keys", func(i *Item) bool { return i.Str == "⌘ K" && i.Dur == 1.5 }},
		{"0:01 · badge \"Live\" · at top-left", "badge", nil},
		{"0:02 · lower-third \"Ada Lovelace\" \"Engineer\" · 4s", "lower-third", func(i *Item) bool { return i.Str == "Ada Lovelace" && i.Sub == "Engineer" }},
		{"2:40 · zoom to 600,300 900x500 · 3s · ease", "zoom", func(i *Item) bool { return i.Rect.W.V == 900 && i.Dur == 3 && i.Move == .8 && !i.Flags["linear"] }},
		{"2:50 · pan to 100,100 900x500 · linear · move 1.2s", "pan", func(i *Item) bool { return i.Flags["linear"] && i.Move == 1.2 }},
		{"3:00 · zoom out", "zoom-out", nil},
		{"3:30 · say \"Now the queue is replicated.\" · duck 30%", "say", func(i *Item) bool { return i.Level == .3 }},
		{"0:00 · music bed.mp3 · 15%", "music", func(i *Item) bool { return i.File == "bed.mp3" && i.Level == .15 }},
		{"captions on", "captions", func(i *Item) bool { return i.NoTime }},
		{"1:36–1:46 · refilm feature \"Ops terminal\" · fade", "refilm", func(i *Item) bool { return i.Feature == "Ops terminal" && i.Trans != nil }},
		{"at chapter \"Install\" + 5s · keys \"⌘ S\"", "keys", func(i *Item) bool { return i.From.Chapter == "Install" && i.From.Off == 5 }},
		{"at said \"management UI\" - 1s · bubble \"Here\" · to center", "bubble", func(i *Item) bool { return i.From.Said == "management UI" && i.From.Off == -1 }},
		{"at said \"a\" – at said \"b\" · blur 0,0 10x10", "blur", func(i *Item) bool { return i.To.Said == "b" }},
		{"0:05 · bubble \"Follow\" · to 100,200 → 400,300 over 2s · linear", "bubble", func(i *Item) bool {
			return len(i.Path) == 2 && i.Point == i.Path[0] && i.Path[1].X.V == 400 && i.PathDur == 2 && i.Flags["linear"]
		}},
		{"0:05 · text \"Here\" · at 10,10 -> 50%,50% · over 1.5s", "text", func(i *Item) bool { return i.At == i.Path[0] && i.Path[1].X.Pct && i.PathDur == 1.5 }},
		{"0:05 · box 10,10 50x50 · path 10,10 400,300 600,250 · 4s", "box", func(i *Item) bool { return len(i.Path) == 3 && i.PathDur == 0 && i.Dur == 4 }},
		{"0:05 · arrow from 10,10 to 50,50 · path 50,50 90,90", "arrow", func(i *Item) bool { return i.Tail.X.V == 10 && i.Point == i.Path[0] }},
		{"0:05–0:09 · cut · highlight", "cut", func(i *Item) bool { return i.HL }},
		{"0:05–0:09 · blur 10,10 40x40 · to 10,10 → 200,100 100x60 → 250,100 over 2s · linear", "blur", func(i *Item) bool {
			return len(i.Path) == 3 && i.PathDur == 2 && i.Flags["linear"] && i.PathWH[0][0].V == 40 && i.PathWH[1][1].V == 60 && i.PathWH[2][0].V == 100
		}},
		{"0:05 · pixelate 10,10 40x40 · path 10,10 90,90 · 2s", "pixelate", func(i *Item) bool { return len(i.Path) == 2 && i.PathWH == nil }},
	}
	for _, c := range cases {
		e := parseLines(t, c.line)
		if len(e.Problems) > 0 || len(e.Items) != 1 {
			t.Errorf("%s: %v", c.line, e.Problems)
			continue
		}
		it := e.Items[0]
		if it.Action != c.action || it.Line != 9 {
			t.Errorf("%s: action %q line %d", c.line, it.Action, it.Line)
		}
		if c.check != nil && !c.check(it) {
			t.Errorf("%s: %+v", c.line, it)
		}
	}
	if e := parseLines(t, "captions on"); !e.Captions || e.Words {
		t.Error("captions on sets Captions")
	}
	if e := parseLines(t, "captions words"); !e.Captions || !e.Words {
		t.Error("captions words sets Captions and Words")
	}
	if e := Parse("/x/e.md", "---\ntype: video\nsource: a.mp4\ncaptions: words\n---\n"); !e.Captions || !e.Words || len(e.Problems) > 0 {
		t.Errorf("captions: words: %v", e.Problems)
	}
}

func TestParseErrorsHaveLines(t *testing.T) {
	e := Parse("/x/edit.md", "---\ntype: video\nsource: demo.mp4\nwobble: 1\n---\n# T\n\nProse is fine.\n\n## Timeline\n\n- 0:01 · cut\n- 0:02 · wiggle\n- soon · title \"x\"\n- 0:03 · freeze\n- 0:04 · blur\n- 0:05 · bubble \"x\" · to 1,2 · sparkle\n- 0:01–0:02 · zoom to 0,0 10x10\n- 0:70 · title \"x\"\n- 0:06 · speed 1x\n\n## Notes\n\n- 0:01 · not parsed\n")
	want := map[int]string{4: "unknown front matter key", 12: "cut needs a range", 13: `unknown action "wiggle"`, 14: "is not a time", 15: "freeze needs a length",
		16: "blur:", 17: `unknown option "sparkle"`, 18: "takes one time", 19: "is not a time", 20: "speed 1x"}
	got := map[int]string{}
	for _, p := range e.Problems {
		got[p.Line] = p.Msg
	}
	for l, w := range want {
		if !strings.Contains(got[l], w) {
			t.Errorf("line %d: %q, want %q", l, got[l], w)
		}
	}
	if len(e.Problems) != len(want) || len(e.Items) != 0 {
		t.Errorf("problems %v items %d", e.Problems, len(e.Items))
	}
}

func TestFrontMatter(t *testing.T) {
	e := Parse("/x/edit.md", "---\ntype: video\nsource:\n  - intro.mp4\n  - demo.mp4 · from 0:12 · to 5:40\n  - outro.mp4\njoin: fade 0.5s\nout: final.mp4\ntheme: studio\ncaptions: on\nvoice: system\nscript: demo.md\n---\n- 0:01 · cut\n")
	if len(e.Problems) != 1 || !strings.Contains(e.Problems[0].Msg, "needs a range") {
		t.Fatalf("problems: %v", e.Problems)
	}
	if len(e.Sources) != 3 || e.Sources[1].From != 12 || e.Sources[1].To != 340 || e.Join != (Trans{"fade", .5}) || e.Out != "final.mp4" || !e.Captions || e.Script != "demo.md" {
		t.Errorf("%+v", e)
	}
	if e := Parse("/x/a.md", "---\ntype: video\nsource: rec.mov\n---\n"); e.Out != "rec.edited.mp4" {
		t.Errorf("default out %q", e.Out)
	}
	if Is("/nope.md") {
		t.Error("Is on a missing file")
	}
	f := filepath.Join(t.TempDir(), "e.md")
	os.WriteFile(f, []byte("---\ntitle: x\ntype: video # an edit\n---\n"), 0o644)
	if !Is(f) {
		t.Error("Is misses type: video")
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// The mapping rule: all times are source times.
func TestMapThroughEdits(t *testing.T) {
	e := parseLines(t,
		"0:10–0:20 · cut",               // -10 after 20
		"0:30 · freeze 2s",              // +2 after 30
		"0:40–0:50 · speed 2x",          // -5 after 50
		"0:55 · insert card \"x\" · 3s", // +3 from 55
		"1:00–1:10 · replace with fix.mp4",
		"1:15 · bubble \"x\" · to 1,1",
		"1:20–1:25 · blur 0,0 10x10",
		"end · card \"bye\" · 2s",
	)
	e.Resolve(100, nil, nil)
	pl := e.Plan(100, func(string) float64 { return 4 }) // fix.mp4 is 4 s: the 10 s range becomes 4
	cases := map[float64]float64{
		0:   0,
		10:  10, // in the cut: the join
		15:  10, // in the cut: the join
		20:  10, // after the cut
		25:  15,
		30:  20, // the freeze shows frame 30 first
		31:  23, // after the 2 s freeze
		40:  32,
		44:  34, // 2x: 4 s become 2
		50:  37,
		55:  45, // the card plays before frame 55
		60:  50, // the replacement starts where its range did
		62:  52, // 2 s into it
		69:  54, // clamped to its 4 s
		70:  54, // after it
		75:  59,
		100: 84, // the end card starts at the source's end
	}
	for in, want := range cases {
		if got := pl.Map(in); !near(got, want) {
			t.Errorf("Map(%g) = %g, want %g", in, got, want)
		}
	}
	if !near(pl.Len, 86) {
		t.Errorf("Len %g, want 86", pl.Len)
	}
	var kinds []string
	for _, s := range pl.Segs {
		kinds = append(kinds, s.Kind)
	}
	if strings.Join(kinds, " ") != "src src still src src src card src clip src card" {
		t.Errorf("segments %v", kinds)
	}
	spans := pl.Spans(e)
	var got []string
	for _, s := range spans {
		got = append(got, s.It.Action+" "+Stamp(s.From)+"–"+Stamp(s.To))
	}
	if strings.Join(got, ", ") != "bubble 0:59–1:02, blur 1:04–1:09" {
		t.Errorf("spans %v", got)
	}
	// overlapping ranges and a point inside a range
	e = parseLines(t, "0:10–0:20 · cut", "0:15–0:25 · speed 2x", "0:12 · freeze 1s", "0:15 · title \"x\"")
	e.Resolve(100, nil, nil)
	pl = e.Plan(100, func(string) float64 { return 0 })
	if len(e.Problems) != 2 || !strings.Contains(e.Problems[0].Msg, "speed overlaps the cut on line 9") || !e.Problems[1].Warn || !strings.Contains(e.Problems[1].Msg, "inside the cut on line 9") {
		t.Errorf("problems %v", e.Problems)
	}
	if c := pl.Cut(15, e); c == nil || c.Action != "cut" {
		t.Error("Cut(15)")
	}
}

func TestTrimAndTransitions(t *testing.T) {
	e := parseLines(t, "0:05 · trim start", "0:50 · trim end", "0:20 · insert clip a.mp4 · fade 0.5s")
	e.Resolve(60, nil, nil)
	pl := e.Plan(60, func(string) float64 { return 3 })
	if !near(pl.Len, 48) || !near(pl.Map(5), 0) || !near(pl.Map(20), 18) || !near(pl.Map(0), 0) || !near(pl.Map(55), 48) {
		t.Errorf("len %g map(20) %g", pl.Len, pl.Map(20))
	}
	if pl.Segs[1].Trans == nil || pl.Segs[2].Trans == nil || pl.Segs[0].Trans != nil {
		t.Errorf("the clip fades in and out: %+v %+v", pl.Segs[1], pl.Segs[2])
	}
}

func TestAnchors(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "demo.vtt"), []byte("WEBVTT\n\n00:00:01.000 --> 00:00:03.000\nFirst we open the\nmanagement UI.\n\n00:00:04.000 --> 00:00:06.000\nThen the queue.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "chapters.txt"), []byte("0:00 Intro\n0:42 Install\n1:30 Configure\n"), 0o644)
	cues := cuesBeside(filepath.Join(dir, "demo.mp4"))
	if len(cues) != 2 || cues[0].Text != "First we open the management UI." || cues[1].From != 4 {
		t.Fatalf("cues %+v", cues)
	}
	if at, ok := saidAt(cues, "Management UI"); !ok || at < 2 || at > 3 {
		t.Errorf("said at %g %v", at, ok)
	}
	if at, ok := saidAt(cues, "management UI then the"); !ok || at < 2 {
		t.Errorf("across two cues: %g %v", at, ok)
	}
	chaps := chaptersOf(filepath.Join(dir, "demo.mp4"), Info{})
	if len(chaps) != 3 || chaps[1] != (Chapter{"Install", 42}) {
		t.Fatalf("chapters %+v", chaps)
	}
	e := parseLines(t, "at chapter \"install\" + 5s · keys \"x\"", "at said \"the queue\" - 1s · keys \"y\"", "at chapter \"Nope\" · keys \"z\"", "at said \"never said\" · keys \"w\"", "1:50 · keys \"v\"")
	e.Resolve(100, cues, chaps)
	if !near(e.Items[0].Start, 47) || e.Items[1].Start < 3 || e.Items[1].Start > 4.5 {
		t.Errorf("anchors %g %g", e.Items[0].Start, e.Items[1].Start)
	}
	msgs := []string{}
	for _, p := range e.Problems {
		msgs = append(msgs, p.Msg)
	}
	if len(msgs) != 3 || !strings.Contains(msgs[0], `no chapter "Nope" ("Intro", "Install", "Configure")`) || !strings.Contains(msgs[1], "not said") || !strings.Contains(msgs[2], "outside the source") {
		t.Errorf("problems %v", msgs)
	}
}

func TestClockAndStamp(t *testing.T) {
	for in, want := range map[string]float64{"1:34": 94, "1:34.5": 94.5, "1:02:03": 3723, "94s": 94, "94.5": 94.5, "0:00": 0} {
		if v, ok := clock(in); !ok || v != want {
			t.Errorf("clock(%q) = %g %v", in, v, ok)
		}
	}
	for _, bad := range []string{"1:75", "x", "1:2:3:4"} {
		if _, ok := clock(bad); ok {
			t.Errorf("clock(%q) ok", bad)
		}
	}
	if Stamp(94.5) != "1:34.5" || Stamp(3723) != "1:02:03" || Stamp(5) != "0:05" {
		t.Errorf("%s %s %s", Stamp(94.5), Stamp(3723), Stamp(5))
	}
}

func TestCalloutHTML(t *testing.T) {
	e := parseLines(t,
		"0:01 · title \"A & B\"",
		"0:02 · bubble \"Here\" · to 1500,800 · think",
		"0:03 · arrow from 200,700 to 640,420 · \"Queue\"",
		"0:04 · text \"Hi\" · at 50%,10% · color #ff0;} · size 54 · box",
		"0:05 · keys \"⌘ K\"",
		"0:06 · lower-third \"Ada\" \"Engineer\"",
		"0:07–0:09 · blur 0,0 100x100",
		"0:08 · spotlight 100,100 400x200",
		"0:09 · text \"typing\" · type",
	)
	e.Resolve(60, nil, nil)
	pl := e.Plan(60, nil)
	spans := pl.Spans(e)
	page, _ := e.OverlayPage(1920, 1080, spans, nil)
	for _, w := range []string{"<h1>A &amp; B</h1>", `class="ve-c ve-bubble ve-up ve-l ve-think"`, `left:78.125%;top:74.074%`, `class="ve-label"`, ">Queue</div>",
		`class="ve-c ve-text ve-boxed"`, "color:#ff0", "font-size:5.000cqmin", "<kbd>⌘</kbd><kbd>K</kbd>", "<b>Ada</b><span>Engineer</span>",
		`class="ve-c ve-spotlight"`, `data-enter="type"`, `data-text="typing"`, "window.VE", `data-start="1.000" data-dur="3.000"`} {
		if !strings.Contains(page, w) {
			t.Errorf("overlay page lacks %q", w)
		}
	}
	if strings.Contains(page, `class="ve-c ve-hide`) || strings.Contains(page, "color:#ff0;}") {
		t.Error("blur is ffmpeg's in the build; a colour never closes the rule")
	}
	key, at := shots(spans, 12)
	// a 3 s fade-in/out title: about 10 + 9 moving frames, one still shot per set of callouts
	if key[0] != "" || key[30] == "" || key[30] == key[31] || key[45] != key[46] || len(at) > 200 {
		t.Errorf("shots: %v… %d", key[28:48], len(at))
	}
}

// A moving callout carries its keyframes (percentages of the stage) and is
// shot every frame while it moves, then once while it stands.
func TestMovingCallout(t *testing.T) {
	e := parseLines(t, "0:01 · bubble \"Follow\" · to 0,0 → 960,540 over 1s · 4s · out none")
	e.Resolve(60, nil, nil)
	spans := e.Plan(60, nil).Spans(e)
	page, _ := e.OverlayPage(1920, 1080, spans, nil)
	if !strings.Contains(page, `data-path="0.000,0.000;50.000,50.000" data-pdur="1.000" data-ease="ease"`) {
		t.Errorf("no keyframes in %s", page)
	}
	key, _ := shots(spans, 6)
	if key[30+15] == key[30+16] || key[30+40] != key[30+41] {
		t.Errorf("shots: moving %q %q, standing %q %q", key[45], key[46], key[70], key[71])
	}
}

// A moving blur is cut and laid back at a place that is an expression of
// t; a mask (geq) only when its size changes; a size in a path only after a
// place of a rectangle.
func TestMovingBlurGraph(t *testing.T) {
	for _, l := range []string{"0:01 · text \"x\" · to 1,1 10x10 → 2,2", "0:01 · box 1,1 9x9 · path 10x10 1,1 2,2", "0:01 · box 1,1 9x9 · path 1,1 5x5 6x6 2,2"} {
		if e := parseLines(t, l); len(e.Problems) != 1 || !strings.Contains(e.Problems[0].Msg, "a size follows a place") {
			t.Errorf("%s: %v", l, e.Problems)
		}
	}
	ks := [][4]float64{{0, 0, 40, 40}, {100, 0, 40, 40}, {100, 50, 40, 40}}
	if e := keyExpr(ks, 200, 100, 0, 1, 2, true, "t"); e != "(st(0,clip((t-1.000)/2.000,0,1));if(lt(ld(0),0.500000),0.00+(100.00)*(ld(0)-0.000000)/0.500000,if(lt(ld(0),1.000000),100.00+(0.00)*(ld(0)-0.500000)/0.500000,100.00)))" {
		t.Errorf("x over time: %s", e) // 50% of the frame across, then 50% down: half the time each
	}
	if e := keyExpr(ks[:1], 200, 100, 1, 0, 1, false, "t"); !strings.Contains(e, "3-2*ld(0)") {
		t.Errorf("eased: %s", e)
	}
	graph := func(line string) string {
		e := parseLines(t, line)
		e.Resolve(10, nil, nil)
		pl := e.Plan(10, nil)
		j := &Job{Src: "s.mp4", W: 640, H: 360, Plan: pl, Spans: pl.Spans(e), Out: "o.mp4"}
		a := j.Args()
		for i, s := range a {
			if s == "-filter_complex" {
				return a[i+1]
			}
		}
		return ""
	}
	g := graph("0:01–0:03 · blur 0,0 40x40 · to 0,0 → 100,50 over 1s")
	if !strings.Contains(g, "trim=start=1.000:end=3.050,crop=40:40:x='min((st(0") || !strings.Contains(g, "overlay=x='min(") || strings.Contains(g, "geq") {
		t.Errorf("moving blur: %s", g)
	}
	g = graph("0:01–0:03 · pixelate 0,0 40x40 · to 0,0 → 100,50 80x60 over 1s")
	if !strings.Contains(g, "crop=80:60:") || !strings.Contains(g, "alphamerge") || !strings.Contains(g, "geq=lum='255*gte(X,") || !strings.Contains(g, "flags=neighbor") {
		t.Errorf("moving pixelate that grows: %s", g)
	}
}

// Burned captions (a picture per word with captions words) are one ffmpeg
// input, an ffconcat list with a transparent picture in the gaps; pictures
// that overlap fall back to an overlay each.
func TestCaptionsOneInput(t *testing.T) {
	dir := t.TempDir()
	var ps []webdemo.Pic
	for i := range 300 {
		ps = append(ps, webdemo.Pic{PNG: filepath.Join(dir, fmt.Sprintf("capw%04d.png", i)), From: 1 + float64(i)*0.4, To: 1.4 + float64(i)*0.4})
	}
	ps[100].To -= .2 // a gap
	inputs := func(a []string) int { return strings.Count(strings.Join(a, " "), " -i ") }
	a := webdemo.CaptionedArgs("v.mp4", 640, 360, ps, "o.mp4")
	if n := inputs(a); n != 2 || !strings.Contains(strings.Join(a, " "), "-f concat -safe 0 -i ") {
		t.Errorf("%d inputs: %v", n, a)
	}
	list, _ := os.ReadFile(filepath.Join(dir, "pictures.ffconcat"))
	if s := string(list); !strings.HasPrefix(s, "ffconcat version 1.0\nfile '"+filepath.ToSlash(filepath.Join(dir, "blank-640x360.png"))+"'\nduration 1.000\nfile '") ||
		strings.Count(s, "blank-640x360.png") != 3 || strings.Count(s, "\nfile ") != 303 {
		t.Errorf("list:\n%.400s", s)
	}
	ps[5].From = ps[4].From // overlapping
	if n := inputs(webdemo.CaptionedArgs("v.mp4", 640, 360, ps, "o.mp4")); n != 301 {
		t.Errorf("overlapping pictures: %d inputs", n)
	}
}

func TestArgsGolden(t *testing.T) {
	e := parseLines(t,
		"0:02–0:04 · cut",
		"0:05 · freeze 1s",
		"0:06–0:08 · speed 2x · mute",
		"0:09 · insert card \"Two\" · 1s · fade 0.5s",
		"0:10–0:12 · replace with fix.mp4 · dissolve",
		"0:13–0:15 · mute",
		"0:01–0:03 · blur 10,10 100x50",
		"0:14 · pixelate 0,0 64x64 · 1s",
		"0:03 · zoom to 0,0 320x180 · 1s",
	)
	e.Resolve(20, nil, nil)
	pl := e.Plan(20, func(string) float64 { return 3 })
	e.quantize(pl)
	j := &Job{Src: "/v/src.mp4", W: 640, H: 360, Audio: true, Plan: pl, Spans: pl.Spans(e), Overlay: "/w/ov%05d.png", Cards: map[*Seg]string{},
		ClipAudio: map[string]bool{"/x/fix.mp4": true}, Says: []Said{{WAV: "/w/say.wav", At: 4, Len: 2, Duck: .3}}, Meta: "/w/chapters.txt", Out: "/o/out.mp4"}
	for _, s := range pl.Segs {
		if s.Kind == "card" {
			j.Cards[s] = "/w/card.png"
		}
	}
	got := strings.Join(j.Args(), "\n") + "\n"
	golden := filepath.Join("testdata", "args.golden")
	if *update {
		os.MkdirAll("testdata", 0o755)
		os.WriteFile(golden, []byte(got), 0o644)
	}
	want, _ := os.ReadFile(golden)
	if got != string(want) {
		t.Errorf("ffmpeg args changed (go test ./internal/videdit -update, then read the diff):\n%s", got)
	}
	if a := atempo(0.25); a != ",atempo=0.5,atempo=0.5" {
		t.Errorf("atempo(0.25) %q", a)
	}
}

// The Video view's drags write source pixels back into their line, the
// rest of the line (and the file) as written.
func TestMoveApply(t *testing.T) {
	src := head + "- 0:01 · bubble \"Here\" · to 820,410 · 4s · pop\n- 0:02 · text \"Hi\" · 3s\n" +
		"- 0:03 · arrow from 200,700 to 640,420 · \"Queue\"\n- 0:04 · box 120,80 400x60 · 2s\n- 0:05–0:06 · blur 1,2,30x40\n" +
		"- 0:07 · title \"T\"\n- 0:08 · bubble \"Go\" · to 0,0 → 10,10\n" +
		"- 0:09–0:10 · blur 1,1 9x9 · to 1,1 → 20,20 30x30→40,40 over 1s\n- 0:11 · box 1,1 9x9 · path center 5,5 · 2s\n- 0:12 · zoom to 0,0 320x180 · 1s\n"
	k1, k0 := 1, 0
	cases := []struct {
		m    Move
		want string
	}{
		{Move{Line: 9, To: &[2]int{100, 200}}, `- 0:01 · bubble "Here" · to 100,200 · 4s · pop`},
		{Move{Line: 10, At: &[2]int{5, 6}}, `- 0:02 · text "Hi" · 3s · at 5,6`},
		{Move{Line: 11, To: &[2]int{9, 9}}, `- 0:03 · arrow from 200,700 to 9,9 · "Queue"`},
		{Move{Line: 11, From: &[2]int{1, 1}}, `- 0:03 · arrow from 1,1 to 640,420 · "Queue"`},
		{Move{Line: 12, Rect: &[4]int{10, 20, 300, 0}}, `- 0:04 · box 10,20 300x1 · 2s`},
		{Move{Line: 13, Rect: &[4]int{3, 4, 50, 60}}, `- 0:05–0:06 · blur 3,4,50x60`},
		{Move{Line: 15, Key: &k1, At: &[2]int{7, 8}}, `- 0:08 · bubble "Go" · to 0,0 → 7,8`},
		{Move{Line: 16, Key: &k1, At: &[2]int{50, 60}}, `- 0:09–0:10 · blur 1,1 9x9 · to 1,1 → 50,60 30x30 → 40,40 over 1s`},
		{Move{Line: 17, Key: &k0, At: &[2]int{2, 3}}, `- 0:11 · box 1,1 9x9 · path 2,3 5,5 · 2s`},
		{Move{Line: 18, Rect: &[4]int{10, 20, 160, 90}}, `- 0:12 · zoom to 10,20 160x90 · 1s`},
	}
	for _, c := range cases {
		got, err := c.m.Apply(src)
		if err != nil {
			t.Errorf("%+v: %v", c.m, err)
			continue
		}
		if l := strings.Split(got, "\n")[c.m.Line-1]; l != c.want {
			t.Errorf("line %d: %q, want %q", c.m.Line, l, c.want)
		}
		if e := Parse("/x/e.md", got); len(e.Problems) > 0 {
			t.Errorf("%v", e.Problems)
		}
	}
	k5 := 5
	for _, m := range []Move{{Line: 14, At: &[2]int{1, 1}}, {Line: 15, To: &[2]int{1, 1}}, {Line: 16, Key: &k5, At: &[2]int{1, 1}}, {Line: 9, Key: &k0, At: &[2]int{1, 1}}, {Line: 1, At: &[2]int{1, 1}}, {Line: 99}} {
		if _, err := m.Apply(src); err == nil {
			t.Errorf("%+v: no error", m)
		}
	}
}
