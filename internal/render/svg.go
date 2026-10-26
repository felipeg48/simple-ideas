package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	xmlDeclRE   = regexp.MustCompile(`(?s)<\?xml.*?\?>\s*|<!DOCTYPE[^>]*>\s*`)
	svgRootRE   = regexp.MustCompile(`(?s)<svg\b[^>]*>`)
	idRE        = regexp.MustCompile(`\bid="([^"]+)"`)
	idRefRE     = regexp.MustCompile(`(url\(\s*#|href="#)([^)"\s]+)`)
	rootAttrRE  = regexp.MustCompile(`\s(width|height)="([\d.]+)(px)?"`)
	svgImageRE  = regexp.MustCompile(`(<image\b[^>]*?\s(?:xlink:)?href=")([^"#][^"]*)"`)
	helperRE    = regexp.MustCompile(`data-(show|hide|alert|ok|flow|packets)="([^"]*)"`)
	classAttrRE = regexp.MustCompile(`\sclass="`)
)

// canvas reads an SVG for the `svg:` directive and returns the node that
// inlines it. State words without a prefix replace the state; words that all
// start with + or - change the state the previous slide left this SVG in.
func (dr *deckRenderer) canvas(val string, sc *slideCtx) *rawNode {
	f := strings.Fields(val)
	if len(f) == 0 {
		return &rawNode{html: "<!-- svg: missing file -->"}
	}
	path, words := f[0], f[1:]
	base := resolveState(dr.lastState[path], words)
	raw, err := os.ReadFile(filepath.Join(sc.s.Dir, path))
	if err != nil {
		dr.problem(sc, "missing file: %s", path)
		return &rawNode{html: fmt.Sprintf(`<figure class="si-figure"><p>missing SVG: %s</p></figure>`, html.EscapeString(path)), media: true}
	}
	dr.svgN++
	for _, p := range lintSVG(string(raw)) {
		dr.problem(sc, "%s: %s", path, p)
	}
	svg := dr.inlineSVG(string(raw), fmt.Sprintf("c%d-", dr.svgN), strings.Join(base, " "), filepath.Dir(filepath.Join(sc.s.Dir, path)), sc)
	sc.canvases = append(sc.canvases, canvasRef{path: path, base: base})
	return &rawNode{
		html:  fmt.Sprintf(`<figure class="si-figure" data-src="%s">%s</figure>`, html.EscapeString(path), svg),
		media: true,
	}
}

// lintSVG reports authored-SVG mistakes that silently break a canvas: a
// less-than sign inside a <style> block (parsed as markup in inline SVG, RULES
// §10) and a root with no viewBox and no width+height to derive one (breaks
// fluid scaling, RULES §1). It targets hand-authored svg: files; generated
// stage SVGs are correct by construction.
func lintSVG(raw string) []string {
	var probs []string
	rest := raw
	for {
		i := strings.Index(rest, "<style")
		if i < 0 {
			break
		}
		open := strings.IndexByte(rest[i:], '>')
		end := strings.Index(rest[i:], "</style>")
		if open < 0 || end < 0 {
			break
		}
		if strings.Contains(rest[i+open+1:i+end], "<") {
			probs = append(probs, "a '<' inside <style> breaks inline SVG (RULES §10)")
		}
		rest = rest[i+end+len("</style>"):]
	}
	if root := svgRootRE.FindString(raw); root != "" && !strings.Contains(root, "viewBox") {
		if !strings.Contains(root, "width=") || !strings.Contains(root, "height=") {
			probs = append(probs, "no viewBox and no width+height to derive one (breaks fluid scaling, RULES §1)")
		}
	}
	return probs
}

// resolveState applies state words to the previous state of the same SVG.
func resolveState(prev, words []string) []string {
	relative := len(words) > 0
	for _, w := range words {
		if !strings.HasPrefix(w, "+") && !strings.HasPrefix(w, "-") {
			relative = false
		}
	}
	var state []string
	if relative {
		state = append(state, prev...)
	}
	return applyOps(state, strings.Join(words, " "))
}

// applyOps applies step ops (word, +word, -word, `.`; separated by spaces or
// commas) to a state.
func applyOps(state []string, ops string) []string {
	out := append([]string(nil), state...)
	for _, w := range strings.FieldsFunc(ops, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' }) {
		switch {
		case w == ".":
		case strings.HasPrefix(w, "-"):
			w = w[1:]
			kept := out[:0]
			for _, x := range out {
				if x != w {
					kept = append(kept, x)
				}
			}
			out = kept
		default:
			w = strings.TrimPrefix(w, "+")
			if !containsWord(out, w) {
				out = append(out, w)
			}
		}
	}
	return out
}

func containsWord(list []string, w string) bool {
	for _, x := range list {
		if x == w {
			return true
		}
	}
	return false
}

// inlineSVG prepares an SVG file for inlining: ids prefixed (the same file
// appears on many slides), the root marked as a canvas with its state, a
// viewBox ensured so it scales, and <image> files embedded.
func (dr *deckRenderer) inlineSVG(svg, prefix, state, dir string, sc *slideCtx) string {
	svg = strings.TrimSpace(xmlDeclRE.ReplaceAllString(svg, ""))
	ids := map[string]bool{}
	for _, m := range idRE.FindAllStringSubmatch(svg, -1) {
		ids[m[1]] = true
	}
	if len(ids) > 0 {
		svg = idRE.ReplaceAllStringFunc(svg, func(s string) string {
			return `id="` + prefix + idRE.FindStringSubmatch(s)[1] + `"`
		})
		svg = idRefRE.ReplaceAllStringFunc(svg, func(s string) string {
			m := idRefRE.FindStringSubmatch(s)
			if ids[m[2]] {
				return m[1] + prefix + m[2]
			}
			return s
		})
	}
	svg = svgImageRE.ReplaceAllStringFunc(svg, func(s string) string {
		m := svgImageRE.FindStringSubmatch(s)
		uri, ok := dataURI(filepath.Join(dir, filepath.FromSlash(cleanSrc(m[2]))))
		if !ok {
			if !isExternal(m[2]) {
				dr.problem(sc, "missing file: %s (image inside SVG)", m[2])
			}
			return s
		}
		return m[1] + uri + `"`
	})
	for _, m := range helperRE.FindAllStringSubmatch(svg, -1) {
		for _, w := range strings.Fields(m[2]) {
			dr.words[m[1]+" "+w] = true
		}
	}
	loc := svgRootRE.FindStringIndex(svg)
	if loc == nil {
		dr.problem(sc, "not an SVG file")
		return ""
	}
	tag := svg[loc[0]:loc[1]]
	if !strings.Contains(tag, "viewBox") {
		var w, h string
		for _, m := range rootAttrRE.FindAllStringSubmatch(tag, -1) {
			if m[1] == "width" {
				w = m[2]
			} else {
				h = m[2]
			}
		}
		if w != "" && h != "" {
			tag = strings.Replace(tag, "<svg", fmt.Sprintf(`<svg viewBox="0 0 %s %s"`, w, h), 1)
		}
	}
	tag = rootAttrRE.ReplaceAllString(tag, "")
	if classAttrRE.MatchString(tag) {
		tag = classAttrRE.ReplaceAllString(tag, ` class="si-canvas `)
	} else {
		tag = strings.Replace(tag, "<svg", `<svg class="si-canvas"`, 1)
	}
	esc := html.EscapeString(state)
	tag = strings.Replace(tag, "<svg", fmt.Sprintf(`<svg data-state="%s" data-base="%s"`, esc, esc), 1)
	if !strings.Contains(tag, "role=") {
		tag = strings.Replace(tag, "<svg", `<svg role="img"`, 1)
	}
	return svg[:loc[0]] + tag + svg[loc[1]:]
}

// helperCSS returns the generated state rules for the canvas helper
// attributes used by this deck's SVGs.
func (dr *deckRenderer) helperCSS() string {
	// hide rules come last so data-hide wins over data-show on one element
	rank := map[string]int{"show": 0, "ok": 1, "alert": 2, "flow": 3, "packets": 4, "hide": 5}
	keys := make([]string, 0, len(dr.words))
	for k := range dr.words {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ai, aj := keys[i][:strings.IndexByte(keys[i], ' ')], keys[j][:strings.IndexByte(keys[j], ' ')]
		if rank[ai] != rank[aj] {
			return rank[ai] < rank[aj]
		}
		return keys[i] < keys[j]
	})
	var b strings.Builder
	for _, k := range keys {
		attr, w := k[:strings.IndexByte(k, ' ')], k[strings.IndexByte(k, ' ')+1:]
		on := fmt.Sprintf(`.si-canvas[data-state~="%s"]`, w)
		switch attr {
		case "show":
			fmt.Fprintf(&b, `%s [data-show~="%s"]{opacity:1}`, on, w)
		case "hide":
			fmt.Fprintf(&b, `%s [data-hide~="%s"]{opacity:0}`, on, w)
		case "alert":
			fmt.Fprintf(&b, `%[1]s [data-alert~="%[2]s"]:is(rect,circle,ellipse,path,line,polygon,polyline),%[1]s [data-alert~="%[2]s"] :is(rect,circle,ellipse,path,line,polygon,polyline){stroke:var(--si-crit)}`, on, w)
			fmt.Fprintf(&b, `%s text[data-alert~="%s"]{fill:var(--si-crit)}`, on, w)
			fmt.Fprintf(&b, `%[1]s [data-alert~="%[2]s"]:is(rect,circle,ellipse,path,line,polygon,polyline),%[1]s [data-alert~="%[2]s"] :is(rect,circle,ellipse,path,line,polygon,polyline){animation:si-pulse 1.3s ease-in-out infinite}`, on, w)
		case "ok":
			fmt.Fprintf(&b, `%[1]s [data-ok~="%[2]s"]:is(rect,circle,ellipse,path,line,polygon,polyline),%[1]s [data-ok~="%[2]s"] :is(rect,circle,ellipse,path,line,polygon,polyline){stroke:var(--si-ok)}`, on, w)
			fmt.Fprintf(&b, `%s text[data-ok~="%s"]{fill:var(--si-ok)}`, on, w)
		case "flow":
			fmt.Fprintf(&b, `%s [data-flow~="%s"]{stroke-dasharray:14 10;animation:si-flow 1.1s linear infinite}`, on, w)
		case "packets":
			fmt.Fprintf(&b, `%s .si-packets[data-for~="%s"]{opacity:1}`, on, w)
		}
		b.WriteByte('\n')
	}
	if len(keys) > 0 {
		b.WriteString(".si-canvas .si-packet{fill:var(--si-accent-2);stroke:var(--si-bg);stroke-width:2}\n")
	}
	return b.String()
}
