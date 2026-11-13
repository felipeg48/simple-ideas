package webdemo

import (
	"image"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCSSSelectorWithSpaces: a CSS selector keeps its spaces and ends at the
// words a step writes after its element, or at ` · `.
func TestCSSSelectorWithSpaces(t *testing.T) {
	for in, want := range map[string]string{
		`click .hm-eng.big .hm-open`:             ".hm-eng.big .hm-open",
		`click .big>.hm-open`:                    ".big>.hm-open",
		`hover nav > a.active`:                   "nav > a.active",
		`click button.pay`:                       "button.pay",
		`click [data-testid="pay now"] span`:     `[data-testid="pay now"] span`,
		`click li:not(.done .x) a`:               "li:not(.done .x) a",
		`expect .cart .total visible`:            ".cart .total",
		`type "Ada" into form .name input`:       "",
		`type "Ada" into #signup .name input`:    "#signup .name input",
		`wait for .toast .msg · within 10s`:      ".toast .msg",
		`zoom to .chart .bar · 1.2s`:             ".chart .bar",
		`spotlight .card .price · circle`:        ".card .price",
		`note "Here" at .cart .total · for 2s`:   ".cart .total",
		`drag from .lane .card to #done .column`: ".lane .card",
	} {
		st, err := ParseStep(in)
		if want == "" { // form is a word, not CSS: still an error, as before
			if err == nil {
				t.Errorf("%s: no error", in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		got := st.Target.CSS
		if st.Drag != nil {
			got = st.Drag.From.CSS
			if st.Drag.To.CSS != "#done .column" {
				t.Errorf("%s: to %q", in, st.Drag.To.CSS)
			}
		}
		if got != want {
			t.Errorf("%s: css %q, want %q", in, got, want)
		}
		if in == `expect .cart .total visible` && st.State != "visible" {
			t.Errorf("%s: state %q", in, st.State)
		}
	}
	for _, in := range []string{`click "Pay now"`, `click role=button "Pay"`} {
		if st, err := ParseStep(in); err != nil || st.Target.CSS != "" {
			t.Errorf("%s: %+v %v", in, st, err)
		}
	}
}

func TestParseSi3Steps(t *testing.T) {
	for _, c := range []struct {
		in    string
		check func(*Step) bool
	}{
		{`go /orders · slide`, func(s *Step) bool { return s.Value.Text == "/orders" && *s.Trans == Transition{"slide-left", 0.4} }},
		{`go /orders`, func(s *Step) bool { return s.Trans == nil }},
		{`browser · pop 0.6s`, func(s *Step) bool { return s.Kind == "browser" && *s.Trans == Transition{"pop", 0.6} }},
		{`terminal "Client" · window=mac · wipe-up`, func(s *Step) bool {
			return s.Name == "Client" && s.Opts["window"] == "mac" && *s.Trans == Transition{"wipe-up", 0.4}
		}},
		{`terminal "Client" · cut`, func(s *Step) bool { return *s.Trans == Transition{"cut", 0} }},
		{`window "si-presenter"`, func(s *Step) bool { return s.Kind == "window" && s.Value.Text == "si-presenter" }},
		{`window 2 · circle`, func(s *Step) bool { return s.Code == 2 && s.Trans.Kind == "circle" }},
		{`window main`, func(s *Step) bool { return s.Name == "main" }},
		{`drag from "Shop" to "Doorstep"`, func(s *Step) bool { return s.Drag.From.Name == "Shop" && s.Drag.To.Name == "Doorstep" }},
		{`drag from 400,300 to 900,420`, func(s *Step) bool {
			return *s.Drag.FromXY == [2]float64{400, 300} && *s.Drag.ToXY == [2]float64{900, 420}
		}},
		{`drag from #a to 10,-5`, func(s *Step) bool { return s.Drag.From.CSS == "#a" && *s.Drag.ToXY == [2]float64{10, -5} }},
		{`drag by 300,80`, func(s *Step) bool { return *s.Drag.By == [2]float64{300, 80} }},
		{`zoom to "Agents panel"`, func(s *Step) bool { return s.Kind == "zoom" && s.Target.Name == "Agents panel" && s.Dur == 0 }},
		{`zoom to #total · 1.2s`, func(s *Step) bool {
			return s.Kind == "zoom" && s.Target.CSS == "#total" && s.Dur == 1200*time.Millisecond
		}},
		{`pan to "Footer"`, func(s *Step) bool { return s.Kind == "pan" && s.Target.Name == "Footer" }},
		{`zoom out · 0.5s`, func(s *Step) bool { return s.Kind == "zoom-out" && s.Dur == 500*time.Millisecond }},
		{`spotlight "Pay"`, func(s *Step) bool { return s.Kind == "spotlight" && s.Target.Name == "Pay" && len(s.Opts) == 0 }},
		{`focus "Pay" · circle · blur`, func(s *Step) bool {
			return s.Kind == "spotlight" && s.Opts["shape"] == "circle" && s.Opts["look"] == "blur"
		}},
		{`spotlight "Agents panel" · label "Focus" · for 3s`, func(s *Step) bool { return s.Value.Text == "Focus" && s.For == 3*time.Second }},
		{`spotlight off`, func(s *Step) bool { return s.Opts["off"] == "1" }},
	} {
		st, err := ParseStep(c.in)
		if err != nil || !c.check(st) {
			t.Errorf("%s: %+v %v", c.in, st, err)
		}
	}
	for _, bad := range []string{`go /x · swirl`, `browser · fade · pop`, `terminal "A" · spin`, `window`, `window "a" b`, `drag "A"`,
		`drag from "A" "B"`, `drag by 3`, `zoom "A"`, `zoom to`, `pan out`, `zoom to "A" · fast`, `spotlight`, `spotlight "A" · glow`,
		`spotlight "A" · label`, `go /x · fade 9s`} {
		if _, err := ParseStep(bad); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
}

func TestTransitionSettings(t *testing.T) {
	s := Parse("/x/d.md", "---\ntype: demo\napp: none\ntransition: wipe 0.6s\nregion: \".agents\" · zoom\n---\n\n## A\n\n- terminal \"T\" · swirl\n")
	if s.Trans == nil || *s.Trans != (Transition{"wipe-left", 0.6}) || s.Region != ".agents" || !s.Zoom {
		t.Errorf("settings: %+v %+v", s.Trans, s)
	}
	if len(s.Problems) != 1 || !strings.Contains(s.Problems[0].Msg, `unknown transition "swirl"`) {
		t.Errorf("problems: %v", s.Problems)
	}
	s = Parse("/x/d.md", "---\ntype: demo\napp: none\ntransition: dissolve\n---\n\n## A\n\nRegion: #x · zoom\n\n- terminal \"T\"\n")
	if len(s.Problems) != 1 || !strings.Contains(s.Problems[0].Msg, "dissolve") || s.Features[0].Region != "#x" || !s.Features[0].Zoom {
		t.Errorf("bad front matter: %v %+v", s.Problems, s.Features[0])
	}
	r := &runner{s: &Script{Trans: &Transition{"pop", 0.4}}}
	if r.transition(nil).Kind != "pop" || r.transition(&Step{Trans: &Transition{"cut", 0}}).Kind != "cut" {
		t.Error("script and step transitions")
	}
	r.o.Transition = &Transition{"zoom", 1}
	if r.transition(nil).Kind != "zoom" || (&runner{s: &Script{}}).transition(nil) != DefaultTransition {
		t.Error("--transition and the default")
	}
}

func TestBlendMath(t *testing.T) {
	if ease(0) != 0 || ease(1) != 1 || ease(.5) != .5 || ease(2) != 1 {
		t.Error("ease")
	}
	// slide-left: the new picture comes in from the right edge, the old leaves left
	if a, _, b, _ := slideShift("slide-left", 0, 100, 50); a != 0 || b != 100 {
		t.Errorf("slide start %v %v", a, b)
	}
	if a, _, b, _ := slideShift("slide-left", 1, 100, 50); a != -100 || b != 0 {
		t.Errorf("slide end %v %v", a, b)
	}
	if _, a, _, b := slideShift("slide-down", .5, 100, 50); a != 25 || b != -25 {
		t.Errorf("slide-down half %v %v", a, b)
	}
	if r := circleRadius(1, 0, 0, 30, 40); r != 50 {
		t.Errorf("circle from a corner: %v", r)
	}
	if r := circleRadius(.5, 15, 20, 30, 40); r != 12.5 {
		t.Errorf("circle half from the centre: %v", r)
	}
	// fade mixes evenly; a wipe-left at half shows the new picture on the right half
	_, _, mix := blend("fade", .5, 100, 50, 0, 0)
	if mix(10, 10) != .5 {
		t.Error("fade mix")
	}
	_, _, mix = blend("wipe-left", .5, 100, 50, 0, 0)
	if mix(90, 10) != 1 || mix(10, 10) != 0 {
		t.Error("wipe mix")
	}
	// pop: the new picture is 92 % at the start, outside it the old one shows
	_, b, mix := blend("pop", 0, 100, 100, 0, 0)
	if b.S != .92 || mix(1, 1) != 0 || mix(50, 50) != 0 {
		t.Errorf("pop start %+v", b)
	}
	if _, b, mix := blend("pop", 1, 100, 100, 0, 0); b.S != 1 || mix(1, 1) != 1 {
		t.Errorf("pop end %+v", b)
	}
}

func TestComposePictures(t *testing.T) {
	pic := func(y byte) *image.YCbCr {
		m := image.NewYCbCr(image.Rect(0, 0, 64, 32), image.YCbCrSubsampleRatio420)
		for i := range m.Y {
			m.Y[i] = y
		}
		for i := range m.Cb {
			m.Cb[i], m.Cr[i] = 128, 128
		}
		return m
	}
	a, b := pic(0), pic(200)
	if m := compose(a, b, "fade", .5, 0, 0); m.Y[0] != 100 || m.Y[len(m.Y)-1] != 100 {
		t.Errorf("fade half: %d", m.Y[0])
	}
	m := compose(a, b, "slide-left", .5, 0, 0)
	if m.Y[2] != 0 || m.Y[60] != 200 {
		t.Errorf("slide half: left %d right %d", m.Y[2], m.Y[60])
	}
	m = compose(a, b, "circle", .3, 32, 16)
	if m.Y[16*m.YStride+32] != 200 || m.Y[0] != 0 {
		t.Error("circle opens from its centre")
	}
	// the camera on the left half shows the left half's pixels over the frame
	half := pic(0)
	for y := 0; y < 32; y++ {
		for x := 32; x < 64; x++ {
			half.Y[y*half.YStride+x] = 255
		}
	}
	z := camera(half, rect{0, 0, 64, 32}, rectF{0, 0, 32, 16})
	if z.Y[31*z.YStride+62] != 0 {
		t.Error("zoomed into the dark half, the right edge is dark")
	}
	if camera(half, rect{0, 0, 64, 32}, rectF{0, 0, 64, 32}) != half {
		t.Error("no camera: the same picture")
	}
}

func TestCameraMath(t *testing.T) {
	view := rectF{0, 0, 1440, 900}
	// a small box: grown by its margin to the frame's aspect, never past 3×
	r := zoomRect(rectF{700, 400, 40, 20}, view, 24)
	if math.Abs(r.W/r.H-1.6) > 1e-9 || r.W != 480 || r.X != 720-240 || r.Y != 410-150 {
		t.Errorf("small: %+v", r)
	}
	// a wide box near the edge: kept inside the view
	r = zoomRect(rectF{1000, 800, 800, 80}, view, 24)
	if r.X+r.W > 1440 || r.Y+r.H > 900 || math.Abs(r.W/r.H-1.6) > 1e-9 || r.W != 848 {
		t.Errorf("edge: %+v", r)
	}
	// a region view (crop): the camera keeps the region's aspect and stays in it
	reg := rectF{100, 100, 600, 300}
	r = zoomRect(rectF{150, 120, 50, 50}, reg, 10)
	if r.X < 100 || r.Y < 100 || math.Abs(r.W/r.H-2) > 1e-9 {
		t.Errorf("region: %+v", r)
	}
	// keys: the move eases from the whole view to the box and stays; scale 2
	// (a frame twice the viewport) doubles the box
	keys := resolveMoves([]move{{At: 1, Dur: 1, Kind: "zoom", B: box{X: 350, Y: 200, W: 20, H: 10}}, {At: 4, Dur: 0.5, Kind: "out"}}, nil, view, 2)
	if got := camAt(keys, 0.5, view); got != view {
		t.Errorf("before: %+v", got)
	}
	mid, end := camAt(keys, 1.5, view), camAt(keys, 3, view)
	if end != zoomRect(rectF{700, 400, 40, 20}, view, 48) || mid.W >= view.W || mid.W <= end.W {
		t.Errorf("mid %+v end %+v", mid, end)
	}
	if got := camAt(keys, 5, view); got != view {
		t.Errorf("zoom out: %+v", got)
	}
	// pan keeps the zoom's size
	keys = resolveMoves([]move{{At: 0, Dur: 0, Kind: "zoom", B: box{X: 700, Y: 400, W: 40, H: 20}}, {At: 1, Dur: 0, Kind: "pan", B: box{X: 100, Y: 100, W: 10, H: 10}}}, nil, view, 1)
	if p := camAt(keys, 2, view); p.W != keys[0].To.W || p.X != 0 {
		t.Errorf("pan: %+v", p)
	}
}

func TestEffectsOnTheVideoClock(t *testing.T) {
	// a 4-second wait from 2 s played 2× faster: a move at 8 s of the take
	// starts at 6 s of the video and keeps its length
	fs := []fast{{From: 2, To: 6, X: 2}}
	view := rectF{0, 0, 100, 100}
	keys := resolveMoves([]move{{At: 8, Dur: 1, Kind: "zoom", B: box{X: 40, Y: 40, W: 10, H: 10}}}, fs, view, 1)
	if keys[0].Start != 6 || keys[0].Dur != 1 {
		t.Errorf("key: %+v", keys[0])
	}
	// a cut at 8–8.5 s of the take: the old frame held from 6 s to 6.5 s of
	// the video, then 0.4 s of fade
	frames := []frame{{"a", 0}, {"a2", 7.9}, {"b", 8.6}, {"b2", 9}}
	fx := &effects{cuts: []cutAt{{From: 8, To: 8.5, Tr: Transition{"fade", 0.4}, X: -1}}, view: rect{0, 0, 100, 100}, scale: 1}
	shots := plan(frames, 10, fs, fx, 100, 100)
	if len(shots) != 240 {
		t.Fatalf("shots: %d", len(shots))
	}
	at := func(o float64) shot { return shots[int(o*fps+.5)] }
	if s := at(6.2); s.A != 1 || s.B >= 0 {
		t.Errorf("held: %+v", s)
	}
	if s := at(6.7); s.A != 1 || s.B != 2 || math.Abs(s.P-0.5) > .1 || s.Kind != "fade" || s.CX != 50 {
		t.Errorf("blend: %+v", s)
	}
	if s := at(7.5); s.B >= 0 || s.A != 3 {
		t.Errorf("after: %+v", s)
	}
	// cut: the old behaviour, frame by frame
	fx.cuts[0].Tr = Transition{"cut", 0}
	if !fx.none() {
		t.Error("cuts only: nothing to draw")
	}
}

func TestSequenceDrawsTransitions(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, y byte) string {
		m := image.NewYCbCr(image.Rect(0, 0, 32, 16), image.YCbCrSubsampleRatio420)
		for i := range m.Y {
			m.Y[i] = y
		}
		for i := range m.Cb {
			m.Cb[i], m.Cr[i] = 128, 128
		}
		f, _ := os.Create(filepath.Join(dir, name))
		defer f.Close()
		if err := jpeg.Encode(f, m, nil); err != nil {
			t.Fatal(err)
		}
		return f.Name()
	}
	a, b := write("a.jpg", 20), write("b.jpg", 220)
	seq := t.TempDir()
	fx := &effects{cuts: []cutAt{{From: 0.5, To: 0.5, Tr: Transition{"fade", 0.4}, X: -1}}, view: rect{0, 0, 32, 16}, scale: 1}
	_, n, err := sequence([]frame{{a, 0}, {b, 0.55}}, 1.5, nil, seq, fx, 32, 16)
	if err != nil || n != 45 {
		t.Fatal(n, err)
	}
	mid, err := decodeJPEG(filepath.Join(seq, "seq00021.jpg")) // 0.7 s: half way
	if err != nil {
		t.Fatal(err)
	}
	if y := mid.Y[8*mid.YStride+16]; y < 90 || y > 150 {
		t.Errorf("mid-fade luma %d", y)
	}
}
