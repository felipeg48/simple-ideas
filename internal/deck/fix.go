package deck

import (
	"regexp"
	"strings"
)

// FixComponents rewrites raw-HTML components in si markdown into si syntax
// (si fix --components): <span class="hl"> to ==text==, the pill paragraph to
// <!-- badge -->, the logo lockup to <!-- logo -->, and simple card grids to
// <!-- cards --> with ### headings. Code fences are left alone, and a
// highlight inside a raw HTML block (where markdown is not parsed) too. A card
// grid holding anything but plain text, <b>, <strong>, <em>, <i> and <code> is
// left as it is.
func FixComponents(src string) (string, map[string]int) {
	n := map[string]int{}
	lines := strings.Split(src, "\n")
	var out []string
	var fence string
	htmlBlock := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		t := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			out = append(out, line)
			continue
		}
		if m := fenceOpenRE.FindString(t); m != "" {
			fence = m
			out = append(out, line)
			continue
		}
		if t == "" {
			htmlBlock = false
		}
		if m := pillRE.FindStringSubmatch(line); m != nil {
			out = append(out, "<!-- badge: "+m[1]+" -->")
			n["badge"]++
			continue
		}
		if m := lockupRE.FindStringSubmatch(line); m != nil {
			alt := ""
			if m[1] != "" {
				alt = " · alt=" + m[1]
			}
			out = append(out, "<!-- logo: × "+m[2]+alt+" -->")
			n["logo"]++
			continue
		}
		if m := cardsOpenRE.FindStringSubmatch(t); m != nil {
			if md, end, ok := fixCards(lines, i, m[1]); ok {
				out = append(out, md...)
				n["cards"]++
				i = end
				continue
			}
		}
		if strings.HasPrefix(t, "<") && !strings.HasPrefix(t, "<!--") {
			htmlBlock = true // markdown is not parsed inside a raw HTML block
		}
		if !htmlBlock {
			c := len(hlRE.FindAllString(line, -1))
			if c > 0 {
				line = hlRE.ReplaceAllString(line, "==$1==")
				n["highlight"] += c
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), n
}

var (
	fenceOpenRE = regexp.MustCompile("^(```+|~~~+)")
	hlRE        = regexp.MustCompile(`<span class="hl">([^<>=\n]+?)</span>`)
	pillRE      = regexp.MustCompile(`^\s*<p class="pill"><span class="pill-mark"></span>(.+?)</p>\s*$`)
	lockupRE    = regexp.MustCompile(`^\s*<div class="lockup"><span class="lk-logo" role="img" aria-label="([^"]*)"></span><span class="lk-x"></span><span class="lk-name">([^<]+)</span></div>\s*$`)
	cardsOpenRE = regexp.MustCompile(`^<div class="cards(?: c([234]))?">$`)
	cardRE      = regexp.MustCompile(`^<div class="card(?: (teal|orange|ok|crit|hi))?">(?:<span class="tag(?: [a-z]+)?">([^<]+)</span>)?<h4>(.+?)</h4>((?:<p>.*?</p>)+)</div>$`)
	paraRE      = regexp.MustCompile(`<p>(.*?)</p>`)
	inlineHTML  = strings.NewReplacer("<b>", "**", "</b>", "**", "<strong>", "**", "</strong>", "**",
		"<em>", "*", "</em>", "*", "<i>", "*", "</i>", "*", "<code>", "`", "</code>", "`")
	anyTagRE = regexp.MustCompile(`<[^>]+>`)
)

// fixCards converts the card grid opening at lines[i]: one card per line,
// then </div>. ok is false (and nothing changes) when a card does not fit.
func fixCards(lines []string, i int, cols string) (md []string, end int, ok bool) {
	head := "<!-- cards -->"
	if cols != "" {
		head = "<!-- cards: " + cols + " -->"
	}
	md = []string{head, ""}
	for j := i + 1; j < len(lines); j++ {
		t := strings.TrimSpace(lines[j])
		if t == "</div>" {
			if len(md) == 2 {
				return nil, 0, false
			}
			return md[:len(md)-1], j, true // drop the last blank line
		}
		m := cardRE.FindStringSubmatch(t)
		if m == nil {
			return nil, 0, false
		}
		title := m[3]
		if m[2] != "" {
			title += " · " + m[2]
		}
		if m[1] != "" {
			title += " · " + m[1]
		}
		if anyTagRE.MatchString(inlineHTML.Replace(title)) {
			return nil, 0, false
		}
		md = append(md, "### "+inlineHTML.Replace(title), "")
		for _, p := range paraRE.FindAllStringSubmatch(m[4], -1) {
			text := inlineHTML.Replace(p[1])
			if anyTagRE.MatchString(text) {
				return nil, 0, false
			}
			md = append(md, text, "")
		}
	}
	return nil, 0, false
}
