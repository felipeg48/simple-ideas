package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// browserLink is one entry of the front matter `browser: Name=URL, …`: a page
// the presenter opens in a floating window over the slides (`si play`, key w).
type browserLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// browserLinks parses `Name=URL, Name=URL` (a bare URL is its own name).
func browserLinks(spec string) []browserLink {
	links := []browserLink{}
	for _, p := range strings.Split(spec, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, url := p, p
		if i := strings.Index(p, "="); i > 0 && !strings.Contains(p[:i], "://") {
			name, url = strings.TrimSpace(p[:i]), strings.TrimSpace(p[i+1:])
		}
		links = append(links, browserLink{Name: name, URL: url})
	}
	return links
}

// toolCues takes the [TERM …] and [BROWSER …] cues out of a Notes: script
// before it becomes HTML (a command's * or _ is no emphasis): each becomes a
// private-use mark the returned replacer turns into a teleprompter badge.
// A malformed cue, or a [BROWSER Name] with no such browser: link (and not
// a URL), is a problem.
func (dr *deckRenderer) toolCues(sc *slideCtx, md string) (string, *strings.Replacer) {
	var pairs []string
	n := 0
	out := deck.ActCueRE.ReplaceAllStringFunc(md, func(c string) string {
		m := deck.ActCueRE.FindStringSubmatch(c)
		a, err := deck.ParseAct(m[1], m[2])
		if err != nil {
			dr.problem(sc, "notes: %v", err)
			return c
		}
		if a.Tool == "browser" && a.Name != "" && !strings.ContainsAny(a.Name, "/.:") {
			known := false
			for _, l := range browserLinks(dr.d.Meta.Browser) {
				known = known || l.Name == a.Name
			}
			if !known {
				dr.problem(sc, "notes: [BROWSER %s]: no browser: link is named %s (front matter browser: %s=http://…), and it is not a URL", a.Name, a.Name, a.Name)
			}
		}
		n++
		mark := fmt.Sprintf("\uE010%d\uE011", n)
		pairs = append(pairs, mark, `<span class="si-cue" data-cue="`+a.Tool+`">`+html.EscapeString(strings.Trim(c, "[]"))+`</span>`)
		return mark
	})
	return out, strings.NewReplacer(pairs...)
}
