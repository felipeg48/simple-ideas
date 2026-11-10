package render

import (
	"fmt"
	"regexp"
	"strings"
)

// pollIDRE is a poll id: what keys its votes on the server.
var pollIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// pollHTML is <!-- poll: retry · Once · Three times · Never -->: a question
// the room answers on their phones under si play --audience. On a slide the
// options carry bars the runtime grows as votes arrive (and buttons on an
// audience phone); in a document, or a deck built without si play, it is the
// plain list of options.
func (dr *deckRenderer) pollHTML(val string, sc *slideCtx, slide bool) string {
	id, _, words := pluginParts(val)
	id = strings.TrimSpace(id)
	multiple := false
	var opts []string
	for _, w := range words {
		if w == "multiple" {
			multiple = true
			continue
		}
		opts = append(opts, w)
	}
	if !pollIDRE.MatchString(id) {
		dr.problem(sc, "poll: %q is not a poll id (lowercase letters, digits and -, e.g. <!-- poll: retry · Once · Twice -->)", id)
		return ""
	}
	if len(opts) < 2 || len(opts) > 6 {
		dr.problem(sc, "poll: %s has %d options; give 2 to 6, each after a ·", id, len(opts))
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="si-poll" data-poll="%s"`, id)
	if multiple {
		b.WriteString(` data-multiple`)
	}
	b.WriteString(`><ol class="si-poll-opts">`)
	for i, o := range opts {
		fmt.Fprintf(&b, `<li class="si-poll-opt" data-i="%d" style="--_share:0"><span class="si-poll-label">%s</span><span class="si-poll-n" aria-label="votes">0</span><span class="si-poll-bar" aria-hidden="true"></span></li>`, i, dr.inline(o))
	}
	b.WriteString(`</ol><figcaption class="si-poll-total">`)
	if slide {
		b.WriteString(`<span data-poll-total>0</span> votes`)
	} else {
		b.WriteString(`A live poll at the talk.`)
	}
	b.WriteString(`</figcaption></figure>`)
	return b.String()
}
