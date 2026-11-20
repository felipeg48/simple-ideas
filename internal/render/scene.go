package render

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Scenes: a ```scene fence (deck.ParseScene) is one inline SVG, every object
// drawn where it is before the first step (on a slide) or after the last (a
// document, or a page without the script). The figure carries every step's
// state and how to play it as JSON (data-scene); web/scene.js plays one step
// to the next and sets any other step at once. On a slide each step is a
// build step (.si-scene-mark), as a chart's are.

// the order objects are drawn in: lanes and routes under the blocks, small
// things over them, pins on top
var sceneLayer = map[string]int{"queue": 0, "route": 1, "trail": 2, "box": 3, "tile": 4, "piece": 4, "parcel": 5, "token": 5, "message": 5, "pin": 6}

func sceneObjSVG(b *strings.Builder, i int, o deck.SceneObj, s deck.SceneState) {
	cls := "si-scene-o si-scene-" + o.Kind
	if s.On {
		cls += " is-on"
	}
	fmt.Fprintf(b, `<g class="%s" data-o="%d"`, cls, i)
	if o.Kind != "trail" && o.Kind != "route" {
		fmt.Fprintf(b, ` transform="translate(%g %g)"`, s.X, s.Y)
	}
	if !s.V {
		b.WriteString(` opacity="0"`)
	}
	b.WriteString(">")
	rect := func(w, h, r float64) string {
		return fmt.Sprintf(`<rect x="%g" y="%g" width="%g" height="%g" rx="%g"/>`, -w/2, -h/2, w, h, r)
	}
	switch o.Kind {
	case "box":
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" rx="%g" transform="rotate(%g)"/><text y="%g">%s</text>`, -s.W/2, -s.H/2, s.W, s.H, s.R, s.A, s.LY, html.EscapeString(s.L))
	case "queue":
		b.WriteString(rect(s.W, s.H, s.R))
		fmt.Fprintf(b, `<text y="%g" font-size="7.5">%s</text>`, -s.H/2-8, html.EscapeString(s.L))
	case "parcel":
		b.WriteString(`<rect x="-9" y="-7" width="18" height="14" rx="2.5"/><path d="M-9 -1H9M0 -7V7"/>`)
	case "token":
		b.WriteString(`<circle r="7"/>`)
	case "message":
		b.WriteString(`<rect x="-10" y="-7" width="20" height="14" rx="2"/><path d="M-10 -7L0 1L10 -7"/>`)
	case "tile", "piece":
		b.WriteString(rect(s.W, s.H, s.R))
	case "trail", "route":
		fmt.Fprintf(b, `<path d="%s"/>`, o.D)
	case "pin":
		b.WriteString(`<path d="M0 0C-6 -8 -6 -14 0 -14C6 -14 6 -8 0 0Z"/>`)
	}
	b.WriteString("</g>")
}

// sceneFence writes a scene: the SVG, its states and steps, and on a slide
// one build step per line that moves something.
func (dr *deckRenderer) sceneFence(attrs map[string]string, body string, problem func(string)) string {
	sc := dr.cur
	dr.scene = true
	s, probs := deck.ParseScene(attrs, body)
	for _, p := range probs {
		problem(p)
	}
	on := 0
	if dr.doc || sc == nil {
		on = len(s.States) - 1 // a page without the script, or printed, shows where the scene ends
	}
	order := make([]int, len(s.Objs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return sceneLayer[s.Objs[order[a]].Kind] < sceneLayer[s.Objs[order[b]].Kind] })
	data, _ := json.Marshal(map[string]any{"s": s.States, "p": s.Steps})
	label := firstNonEmpty(plain(s.Title), "Scene")
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="si-figure si-scene" aria-label="%s" data-on="%d" data-scene="%s"`, html.EscapeString(label), on, html.EscapeString(string(data)))
	if (dr.doc || sc == nil) && len(s.Steps) > 0 {
		b.WriteString(` data-scene-nav`)
	}
	fmt.Fprintf(&b, `><svg class="si-scene-svg" viewBox="0 0 %g %g" font-size="9" role="img" aria-label="%s">`, s.W, s.H, html.EscapeString(label))
	if s.Title != "" {
		fmt.Fprintf(&b, `<text class="si-scene-title" x="16" y="18" font-size="7">%s</text>`, html.EscapeString(strings.ToUpper(plain(s.Title))))
	}
	for _, i := range order {
		sceneObjSVG(&b, i, s.Objs[i], s.States[on][i])
	}
	b.WriteString(`<g class="si-scene-fx"></g></svg>`)
	if sc != nil && !dr.doc {
		for i := range s.Steps {
			fmt.Fprintf(&b, `<span class="si-step si-scene-mark" data-step="%d" hidden></span>`, sc.steps+i+1)
			sc.ops = append(sc.ops, "")
		}
		sc.steps += len(s.Steps)
	}
	b.WriteString("</figure>\n")
	return b.String()
}
