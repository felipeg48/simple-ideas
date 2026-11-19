package render

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// 3D charts (bar3d, donut3d, funnel3d): CSS 3D, no library. A bar is a
// cuboid of five faces (back, left, right, top, front, in that order: the
// CSS styles them by place) sized by --w, --d and --h; a 3D donut is ten
// SVG layers stacked in depth, tipped back. Everything is in em of the
// world, whose font size follows the stage (container units), so the scene
// scales with the slide. data-from is where a part's transform starts when
// the chart first appears (the camera turning, a funnel layer dropping in);
// web/chart.js tweens from it, and from one step's values to the next.

func em(f float64) string {
	if f == 0 {
		f = 0 // never "-0em"
	}
	return pct(f) + "em"
}

// cuboid writes one solid bar: its colour, size, place and the text on its front.
func cuboid(key, colour string, w, d, h float64, at, from, front string) string {
	if from != "" {
		from = ` data-from="` + from + `"`
	}
	return fmt.Sprintf(`<div class="si-c3-cub" data-key="%s" style="--w:%s;--d:%s;--h:%s;--c:%s;transform:%s"%s><i></i><i></i><i></i><i></i><i><b>%s</b></i></div>`,
		key, em(w), em(d), em(h), colour, at, from, front)
}

func (dr *deckRenderer) draw3D(c *deck.Chart, title string, s igSeries, co chartOpts) string {
	n, series := len(s.labels), len(s.names)
	hue := func(i int) string { return "var(--_s" + strconv.Itoa(co.keys[i]%6+1) + ")" }
	key := func(i int) string { return strconv.Itoa(co.keys[i]) }
	var o, legend strings.Builder
	legend.WriteString(`<ul class="si-ig-legend">`)
	for i, l := range s.labels {
		fmt.Fprintf(&legend, `<li data-key="%s" style="--_s:%s">%s`, key(i), hue(i), dr.inline(l))
		if series == 1 {
			legend.WriteString(` <b>` + html.EscapeString(s.text[i][0]) + `</b>`)
		}
		legend.WriteString(`</li>`)
	}
	legend.WriteString(`</ul>`)
	label := html.EscapeString(firstNonEmpty(plain(title), "Chart"))
	scene := func(kind string, span, tall, y float64) {
		fmt.Fprintf(&o, `<div class="si-c3" data-c3="%s" role="img" aria-label="%s" style="--_span:%s;--_tall:%s;--_y:%s%%">`, kind, label, pct(span), pct(tall), pct(y))
	}
	switch c.Type {
	case "donut3d":
		total := 0.0
		for _, v := range s.vals {
			total += v[0]
		}
		scene("donut3d", 12, 10, 50)
		o.WriteString(`<div class="si-c3-pie" style="transform:rotateX(58deg) rotateZ(-20deg)" data-from="rotateX(0deg) rotateZ(-90deg)">`)
		for layer := 9; layer >= 0; layer-- {
			shade := ""
			if layer > 0 {
				shade = ";filter:brightness(.62)"
			}
			fmt.Fprintf(&o, `<svg viewBox="0 0 100 100" aria-hidden="true" style="transform:translateZ(%s)%s">`, em(-float64(layer)*.14), shade)
			at := 0.0
			for i, v := range s.vals {
				share := IGScale(v[0], total)
				fmt.Fprintf(&o, `<circle data-key="%s" cx="50" cy="50" r="34" pathLength="100" style="stroke:%s" stroke-dasharray="%s %s" stroke-dashoffset="%s"%s/>`,
					key(i), hue(i), pct(share), pct(100-share), pct(25-at), co.lift(co.keys[i], at, share, 8))
				at += share
			}
			o.WriteString(`</svg>`)
		}
		o.WriteString(`</div></div>`)
	case "funnel3d":
		top := math.Max(s.max, 1e-9)
		tall := float64(n)*1.05 + 4
		scene("funnel3d", 11, tall, (float64(n)*1.05+1.5)/tall*100)
		o.WriteString(`<div class="si-c3-world" style="transform:rotateX(-26deg) rotateY(-32deg)">`)
		for i, v := range s.vals {
			w := 1.6 + IGScale(v[0], top)/100*5
			y := float64(i) * 1.05
			front := html.EscapeString(s.text[i][0]) // the label is in the legend: a narrow layer has no room for it
			o.WriteString(cuboid(key(i), hue(i), w, w, 1, "translate3d(0,"+em(-y)+",0)", "translate3d(0,"+em(-y-9)+",0)", front))
		}
		o.WriteString(`</div></div>`)
	default: // bar3d: one row of bars, or a row per number column (a bar city)
		const H, w, gx, gz = 6.0, 1.4, 2.4, 2.2
		top := math.Max(s.max, 1e-9)
		xs := float64(n-1)*gx + w
		zs := float64(series-1)*gz + w
		ry, rx := 28.0, 20.0
		if series > 1 {
			ry, rx = 38, 24
		}
		sin, cos := math.Sin(ry*math.Pi/180), math.Cos(ry*math.Pi/180)
		span := cos*xs + sin*zs + 3
		tall := H*math.Cos(rx*math.Pi/180) + math.Sin(rx*math.Pi/180)*(sin*xs+cos*zs) + 3
		scene("bar3d", span, tall, (H+1.2)/tall*100)
		fmt.Fprintf(&o, `<div class="si-c3-world" style="transform:rotateX(-%sdeg) rotateY(-%sdeg)" data-from="rotateX(-12deg) rotateY(-70deg)">`, pct(rx), pct(ry))
		fmt.Fprintf(&o, `<div class="si-c3-floor" style="left:%s;bottom:%s;width:%s;height:%s"></div>`, em(-xs/2-1), em(-zs/2-1), em(xs+2), em(zs+2))
		for i, v := range s.vals {
			x := (float64(i) - float64(n-1)/2) * gx
			for k, val := range v {
				z := (float64(k) - float64(series-1)/2) * gz
				id := key(i)
				if series > 1 {
					id += "-" + strconv.Itoa(k)
				}
				o.WriteString(cuboid(id, hue(i), w, w, IGScale(val, top)/100*H, "translate3d("+em(x)+",0,"+em(z)+")", "", html.EscapeString(s.text[i][k])))
			}
		}
		o.WriteString(`</div></div>`)
		if series > 1 { // the columns, back to front
			names := make([]string, series)
			for k, nm := range s.names {
				names[k] = dr.inline(nm)
			}
			o.WriteString(`<p class="si-c3-depth">` + strings.Join(names, ` <span aria-hidden="true">›</span> `) + `</p>`)
		}
	}
	return o.String() + legend.String()
}
