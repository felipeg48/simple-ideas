// Slide chrome: the watermark, the footer and its position, lab slides, the copyright line.

package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// watermark returns the per-slide watermark element. The deck-level value is
// resolved once (text vs inlined image); every slide reuses it so the image is
// embedded a single time (as a --si-wm-img var set on .si-deck in page()).
func (dr *deckRenderer) watermark(sc *slideCtx) string {
	if !dr.wmDone {
		dr.wmDone = true
		if val := dr.d.Meta.Watermark; val != "" {
			if deck.IsImagePath(val) {
				dir := filepath.Dir(dr.d.Path)
				if g := dr.opt.Guide; g != nil && !filepath.IsAbs(val) { // a guide page may inherit it from _nav.md
					if _, err := os.Stat(filepath.Join(dir, val)); err != nil {
						dir = g.Dir
					}
				}
				if uri, ok := dr.media(dir, val, sc); ok {
					dr.wmKind, dr.wmImg = "img", uri
				} else {
					dr.problem(sc, "missing file: %s", val)
				}
			} else {
				dr.wmKind, dr.wmText = "text", val
			}
		}
	}
	switch dr.wmKind {
	case "text":
		return `<div class="si-watermark si-wm-text" aria-hidden="true">` + html.EscapeString(dr.wmText) + "</div>\n"
	case "img":
		return `<div class="si-watermark si-wm-img" aria-hidden="true"></div>` + "\n"
	}
	return ""
}

// footer returns the per-slide footer element. A slide's own `footer:` wins over
// the deck default; `off` (or empty on both) hides it.
func (dr *deckRenderer) footer(s *deck.Slide) string {
	text := dr.d.Meta.Footer
	if s.Footer != "" {
		text = s.Footer
	}
	if text == "" || text == "off" {
		return ""
	}
	return fmt.Sprintf(`<div class="si-footer" data-pos="%s">%s</div>`+"\n", footerPos(dr.d.Meta.FooterPos), html.EscapeString(text))
}

// footerPos normalises a footer-position to its long form; empty is bottom-right.
func footerPos(p string) string {
	switch p {
	case "bl", "bottom-left":
		return "bottom-left"
	case "bc", "bottom-center":
		return "bottom-center"
	case "tr", "top-right":
		return "top-right"
	case "tl", "top-left":
		return "top-left"
	case "tc", "top-center":
		return "top-center"
	}
	return "bottom-right"
}

// labIconSVG is the beaker icon drawn on a lab slide.
const labIconSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 3h6"/><path d="M10 3v6L4.5 18.2A2 2 0 0 0 6.2 21h11.6a2 2 0 0 0 1.7-2.8L14 9V3"/><path d="M7 14.5h10"/></svg>`

// labSlide renders a lab slide: a big centred beaker icon, the lab label as a
// heading (like the course title), and one button per lab URL.
func (dr *deckRenderer) labSlide(s *deck.Slide) string {
	label := dr.d.Meta.LabsLabel
	if label == "" {
		label = "Now, let's do a Lab!"
	}
	var b strings.Builder
	b.WriteString(`<div class="si-lab-icon" aria-hidden="true">` + labIconSVG + "</div>\n")
	fmt.Fprintf(&b, `<h1 class="si-heading si-lab-title">%s</h1>`+"\n", html.EscapeString(label))
	b.WriteString(`<div class="si-lab-links">` + "\n")
	for i, u := range s.Labs {
		name := u.Text
		if name == "" {
			if len(s.Labs) == 1 {
				name = "Open the Lab"
			} else {
				name = fmt.Sprintf("Lab %d", i+1)
			}
		}
		fmt.Fprintf(&b, `<a class="si-lab-link" href="%s" target="_blank" rel="noopener noreferrer">%s</a>`+"\n", html.EscapeString(u.URL), html.EscapeString(name))
	}
	b.WriteString("</div>\n")
	return b.String()
}

// copyrightLine builds the title-slide colophon from the year and copyright
// holder: "© 2026 CockroachLabs", "© 2026" or "© CockroachLabs". Empty when
// neither is set.
func copyrightLine(year, holder string) string {
	year, holder = strings.TrimSpace(year), strings.TrimSpace(holder)
	if year == "" && holder == "" {
		return ""
	}
	return strings.TrimSpace("© " + strings.TrimSpace(year+" "+holder))
}
