package render

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/qr"
)

// qrHTML is <!-- qr: https://… · caption=Slides and code · style=dots ·
// logo=assets/mark.svg · size=large · ec=H -->: an inline SVG QR code that
// scales with its container, a link when the text is a web address. Its
// colours are --si-qr-fg on --si-qr-bg (dark on light in every theme: an
// inverted code reads badly).
func (dr *deckRenderer) qrHTML(val string, s *deck.Slide, sc *slideCtx) string {
	text, opts, words := pluginParts(val)
	text = strings.TrimSpace(text)
	if text == "" {
		dr.problem(sc, "qr: wants the text or address to encode, e.g. <!-- qr: https://example.com/slides · caption=The slides -->")
		return ""
	}
	level := qr.M
	if v := opts["ec"]; v != "" {
		l, err := qr.ParseLevel(v)
		if err != nil {
			dr.problem(sc, "qr: %v", err)
			return ""
		}
		level = l
	} else if opts["logo"] != "" {
		level = qr.H
	}
	c, err := qr.Encode([]byte(text), qr.Options{Level: level})
	if err != nil {
		dr.problem(sc, "qr: %v", err)
		return ""
	}
	style := opts["style"]
	for _, w := range words {
		if w == "dots" || w == "square" {
			style = w
		}
	}
	if style != "" && style != "dots" && style != "square" {
		dr.problem(sc, "qr: style=%q: use square or dots", style)
	}
	st := qr.Style{Dots: style == "dots", Class: "si-qr-code", Label: "QR code: " + text, BG: qr.Transparent} // colours from CSS
	if logo := opts["logo"]; logo != "" {
		if !deck.IsImagePath(logo) {
			dr.problem(sc, "qr: logo=%q is not an image", logo)
			return ""
		}
		href, ok := dr.media(s.Dir, logo, sc)
		if !ok {
			return ""
		}
		frac := 0.2
		if v := opts["logo-size"]; v != "" {
			p, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
			if err != nil || p <= 0 || p > 50 {
				dr.problem(sc, "qr: logo-size=%q: a percent of the width, 1 to 50", v)
				return ""
			}
			frac = p / 100
		}
		x0, y0, x1, y1, ok, _ := c.Hole(frac, 0.8)
		if !ok {
			dr.problem(sc, "qr: the logo is too big for this code (error correction %s); try a smaller logo-size or ec=H", c.Level)
			return ""
		}
		st.Logo = &qr.Logo{Href: href, X0: x0, Y0: y0, X1: x1, Y1: y1}
	}
	size := opts["size"]
	switch size {
	case "", "small", "medium", "large":
	default:
		dr.problem(sc, "qr: size=%q: use small, medium or large", size)
		size = ""
	}
	var b strings.Builder
	b.WriteString(`<figure class="si-qr"`)
	if size != "" {
		fmt.Fprintf(&b, ` data-size="%s"`, size)
	}
	b.WriteString(">")
	svg := c.SVG(st, "", "", "")
	if strings.HasPrefix(text, "https://") || strings.HasPrefix(text, "http://") {
		fmt.Fprintf(&b, `<a href="%s">%s</a>`, html.EscapeString(text), svg)
	} else {
		b.WriteString(svg)
	}
	if cap := opts["caption"]; cap != "" {
		fmt.Fprintf(&b, `<figcaption class="si-qr-cap">%s</figcaption>`, dr.inline(cap))
	}
	b.WriteString("</figure>")
	return b.String()
}
