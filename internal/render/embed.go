package render

import (
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// embedSlides renders `<!-- slides: ../decks/talk.md [23-24 | 23] -->` or
// `<!-- slides: ../decks/talk.md#live-a-first-broker -->` in a document: the
// deck (or those slides) rendered at build time, self-contained, in an iframe
// sized to the deck's aspect, with a caption naming the deck. Media are
// always inlined (the iframe document has no folder of its own).
func (dr *deckRenderer) embedSlides(val string, s *deck.Slide, sc *slideCtx) string {
	ref, rng, _ := strings.Cut(strings.TrimSpace(val), " ")
	ref, frag, _ := strings.Cut(ref, "#")
	rng = strings.TrimSpace(rng)
	if ref == "" {
		dr.problem(sc, "slides: needs a deck, e.g. <!-- slides: ../decks/talk.md 23-24 -->")
		return ""
	}
	target := filepath.Join(s.Dir, filepath.FromSlash(ref))
	sub, err := deck.Parse(target)
	if err != nil {
		dr.problem(sc, "slides: cannot read %s", ref)
		return ""
	}
	if sub.IsDoc() {
		dr.problem(sc, "slides: %s is a %s document, not a deck", ref, sub.Meta.Type)
		return ""
	}
	n := len(sub.Slides)
	from, to := 1, n
	switch {
	case frag != "":
		sl := newSlugger()
		from = 0
		for i, x := range sub.Slides {
			if sl.slug(html.UnescapeString(plain(dr.inline(x.Title)))) == frag {
				from = i + 1
				break
			}
		}
		if from == 0 {
			dr.problem(sc, "slides: %s has no slide with the heading id #%s", ref, frag)
			return ""
		}
		to = from
	case rng != "":
		a, b, ranged := strings.Cut(rng, "-")
		var e1, e2 error
		from, e1 = strconv.Atoi(strings.TrimSpace(a))
		to = from
		if ranged {
			to, e2 = strconv.Atoi(strings.TrimSpace(b))
		}
		if e1 != nil || e2 != nil || from < 1 || to > n || from > to {
			dr.problem(sc, "slides: %s has %d slides; want a range like 3 or 3-5, not %q", ref, n, rng)
			return ""
		}
	}
	part := *sub
	part.Slides = sub.Slides[from-1 : to]
	res := Deck(&part, Options{Media: MediaInline, Theme: dr.opt.Theme, Config: dr.opt.Config, UserCSS: dr.opt.UserCSS})
	for _, p := range res.Problems {
		dr.problem(sc, "slides: %s", p.Msg)
	}
	aspect := strings.ReplaceAll(sub.Meta.Aspect, ":", "/")
	if aspect == "" {
		aspect = "16/9"
	}
	which := fmt.Sprintf("slides %d–%d", from, to)
	if from == to {
		which = fmt.Sprintf("slide %d", from)
	}
	if from == 1 && to == n {
		which = fmt.Sprintf("%d slides", n)
	}
	title := plain(sub.Title())
	// the caption names the deck; it does not link to it, since the deck may not be part of the build
	return fmt.Sprintf(`<figure class="si-embed-slides" style="--_aspect:%s"><iframe srcdoc="%s" title="%s, %s" loading="lazy" allowfullscreen></iframe>`+
		`<figcaption>From <cite>%s</cite>, %s. Click the slides, then use the arrow keys.</figcaption></figure>`,
		html.EscapeString(aspect), html.EscapeString(string(res.HTML)), html.EscapeString(title), which,
		html.EscapeString(title), which)
}
