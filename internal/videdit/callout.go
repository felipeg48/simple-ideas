package videdit

import (
	_ "embed"
	"fmt"
	"html"
	"math"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

//go:embed overlay.css
var overlayCSS string

//go:embed overlay.js
var overlayJS string

// Entrance and exit lengths (seconds); the build shoots every frame inside
// them and one picture for the still part between.
const (
	enterLen = 0.35
	drawLen  = 0.5
	exitLen  = 0.3
)

// inOut are a callout's entrance and exit lengths, fitted into its d seconds.
func inOut(it *Item, d float64) (float64, float64) {
	in := enterLen
	switch it.Enter {
	case "draw":
		in = drawLen
	case "type":
		in = max(enterLen, min(0.045*float64(len([]rune(it.Str))), 2.5))
	}
	out := exitLen
	if it.Exit == "none" {
		out = 0
	}
	if in+out > d*0.8 && in+out > 0 {
		k := d * 0.8 / (in + out)
		in, out = in*k, out*k
	}
	return in, out
}

// Theme is the edit's theme as a path or a built-in name.
func (e *Edit) theme() string {
	if strings.HasSuffix(strings.ToLower(e.Theme), ".css") {
		return e.Abs(e.Theme)
	}
	return e.Theme
}

// head is the CSS every page of the edit starts with: the theme's tokens,
// the fonts, the callout CSS.
func (e *Edit) head(cfg *config.Config) (string, []string) {
	fonts, bad := "", []string(nil)
	if e.Fonts != "" || (cfg != nil && cfg.Fonts != "") {
		fonts, bad = render.FontsCSS(&deck.Deck{Path: e.Path, Meta: deck.Meta{Fonts: e.Fonts}}, cfg)
	}
	return "@layer si.theme{:root{" + webdemo.TokensOf(e.theme()) + "}}\n" + fonts + overlayCSS, bad
}

var colorTokens = map[string]string{"accent": "var(--si-accent)", "fg": "var(--si-fg)", "bg": "var(--si-bg)", "muted": "var(--si-muted)", "surface": "var(--si-surface)"}

func cssColor(c string) string {
	if v, ok := colorTokens[strings.ToLower(c)]; ok {
		return v
	}
	return strings.Map(func(r rune) rune { // a colour, never markup
		if strings.ContainsRune(`;{}<>"'\`, r) {
			return -1
		}
		return r
	}, c)
}

func pct(v float64, full int) string { return fmt.Sprintf("%.3f%%", v/float64(full)*100) }

// cq is n source pixels in container units (cqmin) of a w×h stage.
func cq(n float64, w, h int) string { return fmt.Sprintf("%.3fcqmin", n/float64(min(w, h))*100) }

// anchor places an element at p: a word sits a margin in from its edges, a
// point is the element's centre.
func anchor(p *Pos, w, h int) string {
	x, y := p.Px(w, h)
	tx, ty := -50.0, -50.0
	if p.Word != "" {
		f := words[p.Word]
		tx, ty = edge(f[0]), edge(f[1])
	}
	return fmt.Sprintf("left:%s;top:%s;transform:translate(%g%%,%g%%)", pct(x, w), pct(y, h), tx, ty)
}

func edge(f float64) float64 {
	switch {
	case f < .3:
		return 0
	case f > .7:
		return -100
	}
	return -50
}

// calloutHTML is one line's element on a w×h stage, showing from start for
// d seconds. preview adds blur and pixelate (the build does those in
// ffmpeg).
func calloutHTML(it *Item, w, h int, start, d float64, preview bool) string {
	in, out := inOut(it, d)
	esc := html.EscapeString
	data := fmt.Sprintf(` data-line="%d" data-start="%.3f" data-dur="%.3f" data-in="%.3f" data-out="%.3f" data-enter="%s" data-exit="%s"`,
		it.Line, start, d, in, out, it.Enter, it.Exit)
	if len(it.Path) > 1 { // a moving callout: its keyframes as percentages of the stage
		var ps []string
		for _, p := range it.Path {
			x, y := p.Px(w, h)
			ps = append(ps, fmt.Sprintf("%.3f,%.3f", x/float64(w)*100, y/float64(h)*100))
		}
		ease := "ease"
		if it.Flags["linear"] {
			ease = "linear"
		}
		data += fmt.Sprintf(` data-path="%s" data-pdur="%.3f" data-ease="%s"`, strings.Join(ps, ";"), pathLen(it, d), ease)
	}
	text := func() string {
		if it.Enter == "type" {
			return `<span class="ve-type" data-text="` + esc(it.Str) + `"></span>`
		}
		return esc(it.Str)
	}
	box := func(class string) string {
		x, y, rw, rh := it.Rect.Px(w, h)
		return fmt.Sprintf(`<div class="ve-c %s"%s style="left:%s;top:%s;width:%s;height:%s"><div class="ve-in"></div></div>`,
			class, data, pct(x, w), pct(y, h), pct(rw, w), pct(rh, h))
	}
	at := func(def string) string {
		p := it.At
		if p == nil {
			p = &Pos{Word: def}
		}
		return anchor(p, w, h)
	}
	switch it.Action {
	case "title":
		return fmt.Sprintf(`<div class="ve-c ve-title"%s><div class="ve-in"><h1>%s</h1></div></div>`, data, text())
	case "text":
		cls := "ve-c ve-text"
		if it.Flags["box"] {
			cls += " ve-boxed"
		}
		if it.Flags["shadow"] {
			cls += " ve-shadow"
		}
		var st []string
		if it.Font != "" {
			st = append(st, fmt.Sprintf(`font-family:"%s",var(--si-font-body,system-ui,sans-serif)`, cssColor(it.Font)))
		}
		if it.Size > 0 {
			st = append(st, "font-size:"+cq(it.Size, w, h))
		}
		if it.Color != "" {
			st = append(st, "color:"+cssColor(it.Color))
		}
		return fmt.Sprintf(`<div class="%s"%s style="%s"><div class="ve-in" style="%s">%s</div></div>`, cls, data, at("center"), strings.Join(st, ";"), text())
	case "bubble":
		p := it.Point
		if p == nil {
			p = it.At
		}
		if p == nil {
			p = &Pos{Word: "center"}
		}
		x, y := p.Px(w, h)
		cls := "ve-c ve-bubble"
		if y/float64(h) > .3 {
			cls += " ve-up"
		} else {
			cls += " ve-down"
		}
		if x/float64(w) > .5 {
			cls += " ve-l"
		} else {
			cls += " ve-r"
		}
		for _, f := range []string{"round", "think"} {
			if it.Flags[f] {
				cls += " ve-" + f
			}
		}
		return fmt.Sprintf(`<div class="%s"%s style="left:%s;top:%s"><div class="ve-in">%s</div></div>`, cls, data, pct(x, w), pct(y, h), text())
	case "arrow":
		x1, y1 := it.Tail.Px(w, h)
		x2, y2 := it.Point.Px(w, h)
		sw := float64(min(w, h)) * .009
		ang := math.Atan2(y2-y1, x2-x1)
		hl := sw * 4.2 // the head
		bx, by := x2-hl*math.Cos(ang), y2-hl*math.Sin(ang)
		px, py := -math.Sin(ang)*hl*.6, math.Cos(ang)*hl*.6
		svg := fmt.Sprintf(`<svg viewBox="0 0 %d %d" preserveAspectRatio="none"><path class="ve-path" pathLength="1" stroke-width="%.2f" d="M%.1f %.1fL%.1f %.1f"/><path class="ve-head" d="M%.1f %.1fL%.1f %.1fL%.1f %.1fZ"/></svg>`,
			w, h, sw, x1, y1, bx, by, x2, y2, bx+px, by+py, bx-px, by-py)
		label := ""
		if it.Str != "" { // at the tail, on the side away from the head
			tx, ty := 0.0, 0.0
			if x2 < x1 {
				tx = 0
			} else {
				tx = -100
			}
			if y2 < y1 {
				ty = 20
			} else {
				ty = -120
			}
			if math.Abs(x2-x1) < math.Abs(y2-y1)*.5 {
				tx = -50
			}
			switch { // kept inside the frame near its edges
			case x1/float64(w) < .25:
				tx = 0
			case x1/float64(w) > .75:
				tx = -100
			}
			switch {
			case y1/float64(h) > .85:
				ty = -120
			case y1/float64(h) < .15:
				ty = 20
			}
			label = fmt.Sprintf(`<div class="ve-label" style="left:%s;top:%s;transform:translate(%g%%,%g%%)">%s</div>`, pct(x1, w), pct(y1, h), tx, ty, esc(it.Str))
		}
		return fmt.Sprintf(`<div class="ve-c ve-arrow"%s data-tail="%.1f,%.1f" data-head="%.1f,%.1f" data-sw="%.2f"><div class="ve-in">%s%s</div></div>`, data, x1, y1, x2, y2, sw, svg, label)
	case "box", "circle", "underline":
		return box("ve-mark ve-" + it.Action)
	case "spotlight":
		return box("ve-spotlight")
	case "blur", "pixelate":
		if !preview {
			return ""
		}
		x, y, rw, rh := it.Rect.Px(w, h)
		return fmt.Sprintf(`<div class="ve-c ve-hide ve-%s" data-line="%d" data-start="%.3f" data-dur="%.3f" data-in="0" data-out="0" data-enter="none" data-exit="none" style="left:%s;top:%s;width:%s;height:%s"><div class="ve-in"></div></div>`,
			it.Action, it.Line, start, d, pct(x, w), pct(y, h), pct(rw, w), pct(rh, h))
	case "keys":
		var ks []string
		for _, k := range strings.FieldsFunc(it.Str, func(r rune) bool { return r == ' ' || r == '+' }) {
			ks = append(ks, "<kbd>"+esc(k)+"</kbd>")
		}
		return fmt.Sprintf(`<div class="ve-c ve-keys"%s style="%s"><div class="ve-in">%s</div></div>`, data, at("bottom"), strings.Join(ks, ""))
	case "badge", "speed":
		s := it.Str
		if it.Action == "speed" {
			s = it.Badge
		}
		return fmt.Sprintf(`<div class="ve-c ve-badge"%s style="%s"><div class="ve-in">%s</div></div>`, data, at("top-right"), esc(s))
	case "lower-third":
		sub := ""
		if it.Sub != "" {
			sub = "<span>" + esc(it.Sub) + "</span>"
		}
		return fmt.Sprintf(`<div class="ve-c ve-third"%s style="%s"><div class="ve-in"><b>%s</b>%s</div></div>`, data, at("bottom-left"), esc(it.Str), sub)
	}
	return ""
}

// OverlayPage is the page the build shoots: a transparent w×h stage with
// every span's callout at its output time (blur and pixelate are left to
// ffmpeg). VE.at(t) draws time t.
func (e *Edit) OverlayPage(w, h int, spans []Span, cfg *config.Config) (string, []string) {
	head, bad := e.head(cfg)
	var b strings.Builder
	b.WriteString(`<!doctype html><meta charset="utf-8"><style>` + head + `html,body{margin:0;width:100%;height:100%;background:transparent;overflow:hidden}.ve-stage{width:100vw;height:100vh}</style><div class="ve-stage">`)
	for _, s := range spans {
		if !drawn(s.It) {
			continue
		}
		b.WriteString(calloutHTML(s.It, w, h, s.From, s.To-s.From, false))
	}
	b.WriteString(`</div><script>` + overlayJS + `</script>`)
	return b.String(), bad
}

// shots are the overlay frames to draw: for each output frame, the time to
// draw it at, and frames that look the same (only still callouts, the same
// ones) share a shot. key[k] is the shot frame k shows ("" for none).
func shots(spans []Span, total float64) (key []string, at map[string]float64) {
	n := int(math.Ceil(total*webdemo.FPS - 1e-6))
	key = make([]string, n)
	at = map[string]float64{}
	for k := 0; k < n; k++ {
		t := float64(k) / webdemo.FPS
		var on []string
		moving := false
		for i, s := range spans {
			if !drawn(s.It) {
				continue
			}
			l := t - s.From
			d := s.To - s.From
			if l < 0 || l >= d {
				continue
			}
			in, out := inOut(s.It, d)
			if l < in || (out > 0 && l > d-out) || len(s.It.Path) > 1 && l <= pathLen(s.It, d) {
				moving = true
			}
			on = append(on, fmt.Sprint(i))
		}
		if len(on) == 0 {
			continue
		}
		id := "s" + strings.Join(on, "-")
		if moving {
			id = fmt.Sprintf("f%d", k)
		}
		key[k] = id
		if _, ok := at[id]; !ok {
			at[id] = t
		}
	}
	return key, at
}

// pathLen is how long a moving callout of d seconds takes to move (its
// · over, else all of d).
func pathLen(it *Item, d float64) float64 {
	if it.PathDur > 0 {
		return min(it.PathDur, d)
	}
	return d
}

// drawn reports whether the build draws the line in Chrome.
func drawn(it *Item) bool {
	return it.IsCallout() || it.Action == "speed" && it.Badge != ""
}

func rel(dir, p string) string {
	if r, err := filepath.Rel(dir, p); err == nil {
		return filepath.ToSlash(r)
	}
	return p
}
