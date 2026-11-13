package render

import "strings"

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
