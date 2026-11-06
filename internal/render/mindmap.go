package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Mind maps: one Markdown list of topics under a `# Title` per map, as a page
// (`type: mindmap`) or a ```mindmap fence in a slide or a document. Go parses
// the lists into JSON maps; web/mindmap.js lays them out, draws them, and
// handles drill-down, folding, moving, search and Play. Mind map files linked
// with `→ other.md` are inlined, so drill-down works offline in one HTML file.

type mmNode struct {
	T         string      `json:"t"`
	Kids      []*mmNode   `json:"kids,omitempty"`
	Link      string      `json:"link,omitempty"`  // another map of this page (its id): drill-down
	Href      string      `json:"href,omitempty"`  // a page, a slide or a URL
	Chip      string      `json:"chip,omitempty"`  // the label of Href: "slide 5", "page", "link"
	Img       string      `json:"img,omitempty"`   // a picture, resolved like any image
	Badge     string      `json:"badge,omitempty"` // a short tag after the text
	Sticky    bool        `json:"sticky,omitempty"`
	Code      *mmCode     `json:"code,omitempty"`
	Collapsed bool        `json:"collapsed,omitempty"`
	Pos       *[2]float64 `json:"pos,omitempty"`  // moved by hand: this topic only
	BPos      *[2]float64 `json:"bpos,omitempty"` // moved by hand: the topic and its branch
}

type mmCode struct {
	Lang  string `json:"lang"`
	HTML  string `json:"html"` // highlighted
	Text  string `json:"text"` // for search
	Lines int    `json:"lines"`
	Cols  int    `json:"cols"`
	// a window: ```go window=mac title="retry.go" under a topic
	Window string `json:"window,omitempty"`
	Title  string `json:"title,omitempty"`
}

type mmMap struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	File  string    `json:"file"`
	Kids  []*mmNode `json:"kids"`
}

// mmOpts are the look of a map: front matter of a mind map page, or the
// options of a ```mindmap fence.
type mmOpts struct {
	Layout, Lines, Nodes, Theme, WM, WMPos, View, Camera, Floors, Kind, Flow string
	Steps                                                                    string // a fence revealed one branch per build step: its number on the page
	Prefix                                                                   string // ids of the fallback list (fences: one per fence, so two never clash)
}

type mmParser struct {
	dr    *deckRenderer
	sc    *slideCtx
	maps  []*mmMap
	files map[string]string // absolute file -> its first map id
	queue []string          // linked mind map files still to inline
	slug  slugger
	own   int // maps from the file itself (the rest are inlined linked files)
}

func newMMParser(dr *deckRenderer, sc *slideCtx) *mmParser {
	return &mmParser{dr: dr, sc: sc, files: map[string]string{}, slug: newSlugger()}
}

var (
	mmAttrRE = regexp.MustCompile(`\s*\{([^{}]*)\}\s*$`)
	mmImgRE  = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	mmPosRE  = regexp.MustCompile(`^(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)$`)
)

// addSource parses one Markdown source of maps (paths relative to dir) and
// returns the id of its first map.
func (p *mmParser) addSource(src, dir, file, title string) string {
	b := []byte(src)
	doc := p.dr.md.Parser().Parse(text.NewReader(b))
	var cur *mmMap
	first := ""
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Heading:
			if n.Level != 1 {
				continue
			}
			t, id := strings.TrimSpace(textOf(n, b)), ""
			if m := customIDRE.FindStringSubmatch(t); m != nil {
				id, t = m[1], strings.TrimSpace(customIDRE.ReplaceAllString(t, ""))
			}
			cur = p.newMap(id, t, file)
		case *ast.List:
			if cur == nil {
				cur = p.newMap("", title, file)
			}
			cur.Kids = append(cur.Kids, p.items(n, b, dir)...)
		}
		if cur != nil && first == "" {
			first = cur.ID
		}
	}
	return first
}

func (p *mmParser) newMap(id, title, file string) *mmMap {
	if title == "" {
		title = "Mind map"
	}
	if id == "" {
		id = p.slug.slug(title)
	} else {
		p.slug[id] = true
	}
	rel, err := filepath.Rel(filepath.Dir(p.dr.d.Path), file)
	if err != nil {
		rel = filepath.Base(file)
	}
	m := &mmMap{ID: id, Title: title, File: filepath.ToSlash(rel)}
	p.maps = append(p.maps, m)
	return m
}

// items turns a list into topics: the item's first paragraph is the topic, a
// nested list its children, a fenced code block a code topic.
func (p *mmParser) items(list *ast.List, src []byte, dir string) []*mmNode {
	var out []*mmNode
	for li := list.FirstChild(); li != nil; li = li.NextSibling() {
		n := &mmNode{}
		labelled := false
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			switch b := c.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				if !labelled {
					p.label(n, strings.ReplaceAll(blockText(b, src), "\n", " "), dir)
					labelled = true
				}
			case *ast.List:
				n.Kids = append(n.Kids, p.items(b, src, dir)...)
			case *ast.FencedCodeBlock:
				lang := strings.ToLower(string(b.Language(src)))
				var code strings.Builder
				for i := 0; i < b.Lines().Len(); i++ {
					seg := b.Lines().At(i)
					code.Write(seg.Value(src))
				}
				body := strings.TrimRight(code.String(), "\n")
				cols := 0
				lines := strings.Split(body, "\n")
				for _, l := range lines {
					cols = max(cols, len([]rune(strings.ReplaceAll(l, "\t", "    "))))
				}
				name := lang
				if name == "" {
					name = "code"
				}
				c := &mmCode{Lang: lang, HTML: highlight(body, lang), Text: body, Lines: len(lines), Cols: cols}
				if b.Info != nil {
					a := fenceAttrs(string(b.Info.Segment.Value(src)))
					c.Window, c.Title = a["window"], a["title"]
				}
				n.Kids = append(n.Kids, &mmNode{T: name + " code", Code: c})
			}
		}
		if labelled || len(n.Kids) > 0 {
			out = append(out, n)
		}
	}
	return out
}

// label reads a topic line: `[!sticky] text ![](img) {attrs} → target`.
func (p *mmParser) label(n *mmNode, t, dir string) {
	t = strings.TrimSpace(t)
	if rest, ok := strings.CutPrefix(t, "[!sticky]"); ok {
		n.Sticky, t = true, strings.TrimSpace(rest)
	}
	target := ""
	if i := strings.LastIndex(t, "→"); i >= 0 {
		target, t = strings.TrimSpace(t[i+len("→"):]), t[:i]
	} else if i := strings.LastIndex(t, " -> "); i >= 0 {
		target, t = strings.TrimSpace(t[i+4:]), t[:i]
	}
	if m := mmAttrRE.FindStringSubmatch(t); m != nil {
		t = mmAttrRE.ReplaceAllString(t, "")
		p.attrs(n, m[1])
	}
	if m := mmImgRE.FindStringSubmatch(t); m != nil {
		t = strings.Replace(t, m[0], "", 1)
		src := m[2]
		if isExternal(src) {
			n.Img = src
		} else if uri, ok := p.dr.media(dir, src, p.sc); ok {
			n.Img = uri
		} else {
			p.dr.problem(p.sc, "mindmap: missing file: %s", src)
		}
		if strings.TrimSpace(t) == "" {
			t = m[1]
		}
	}
	doc := p.dr.md.Parser().Parse(text.NewReader([]byte(strings.TrimSpace(t))))
	n.T = strings.TrimSpace(markRE.ReplaceAllString(textOf(doc, []byte(strings.TrimSpace(t))), "$1"))
	if target != "" {
		p.target(n, target, dir)
	}
}

func (p *mmParser) attrs(n *mmNode, s string) {
	for _, f := range strings.Fields(s) {
		k, v, _ := strings.Cut(f, "=")
		switch k {
		case "collapsed":
			n.Collapsed = true
		case "badge":
			n.Badge = strings.ReplaceAll(v, "_", " ")
		case "pos", "branch-pos":
			m := mmPosRE.FindStringSubmatch(v)
			if m == nil {
				p.dr.problem(p.sc, "mindmap: %s=%s needs two numbers, like %s=40,-12", k, v, k)
				continue
			}
			x, _ := strconv.ParseFloat(m[1], 64)
			y, _ := strconv.ParseFloat(m[2], 64)
			if k == "pos" {
				n.Pos = &[2]float64{x, y}
			} else {
				n.BPos = &[2]float64{x, y}
			}
		default:
			p.dr.problem(p.sc, "mindmap: unknown topic option %q (use badge=, collapsed, pos=, branch-pos=)", k)
		}
	}
}

// target resolves `→ x`: #id (a map of this page), slides: talk.md#5, a URL,
// another mind map file (inlined: drill-down), or any other page.
func (p *mmParser) target(n *mmNode, t, dir string) {
	switch {
	case strings.HasPrefix(t, "#"):
		n.Link = t[1:]
	case strings.HasPrefix(t, "slides:"):
		file, frag, _ := strings.Cut(strings.TrimSpace(strings.TrimPrefix(t, "slides:")), "#")
		n.Href, n.Chip = p.pageHref(dir, file, frag), "slides"
		if frag != "" {
			n.Chip = "slide " + frag
		}
	case isExternal(t):
		n.Href, n.Chip = t, "link"
	default:
		file, frag, _ := strings.Cut(t, "#")
		abs := filepath.Join(dir, filepath.FromSlash(file))
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			p.dr.problem(p.sc, "mindmap: missing file: %s", file)
			return
		}
		if strings.EqualFold(filepath.Ext(abs), ".md") && isMindmapFile(abs) {
			if _, seen := p.files[abs]; !seen {
				p.files[abs] = ""
				p.queue = append(p.queue, abs)
			}
			n.Link = "file:" + abs + "#" + frag
			return
		}
		n.Href, n.Chip = p.pageHref(dir, file, frag), "page"
	}
}

// pageHref is a link from the rendered page to file (relative to dir), .md
// pointing at the built .html.
func (p *mmParser) pageHref(dir, file, frag string) string {
	rel, err := filepath.Rel(filepath.Dir(p.dr.d.Path), filepath.Join(dir, filepath.FromSlash(file)))
	if err != nil {
		rel = file
	}
	rel = filepath.ToSlash(rel)
	if strings.EqualFold(filepath.Ext(rel), ".md") {
		rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html"
	}
	if frag != "" {
		rel += "#" + frag
	}
	return rel
}

func isMindmapFile(abs string) bool {
	raw, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	head := string(raw[:min(len(raw), 2048)])
	return regexp.MustCompile(`(?m)^type:\s*["']?mindmap\b`).MatchString(head)
}

// finish inlines the linked mind map files (and the ones they link) and
// resolves every drill-down link to a map id.
func (p *mmParser) finish() {
	for len(p.queue) > 0 {
		abs := p.queue[0]
		p.queue = p.queue[1:]
		d, err := deck.ParseWith(abs, deck.Meta{Type: "mindmap"})
		if err != nil {
			p.dr.problem(p.sc, "mindmap: %v", err)
			continue
		}
		first := ""
		for _, s := range d.Slides {
			if id := p.addSource(s.Body, s.Dir, abs, d.Title()); first == "" {
				first = id
			}
		}
		p.files[abs] = first
	}
	ids := map[string]bool{}
	for _, m := range p.maps {
		ids[m.ID] = true
	}
	var walk func([]*mmNode)
	walk = func(list []*mmNode) {
		for _, n := range list {
			if rest, ok := strings.CutPrefix(n.Link, "file:"); ok {
				abs, frag, _ := strings.Cut(rest, "#")
				n.Link = p.files[abs]
				if frag != "" {
					n.Link = frag
				}
			}
			if n.Link != "" && !ids[n.Link] {
				p.dr.problem(p.sc, "mindmap: no map #%s (a map is a `# Title {#id}` heading)", n.Link)
				n.Link = ""
			}
			walk(n.Kids)
		}
	}
	for _, m := range p.maps {
		walk(m.Kids)
	}
}

// component is the map's HTML: the data for mindmap.js and, for readers
// without JavaScript (and screen readers), the maps as nested lists.
func (p *mmParser) component(o mmOpts, start string, full bool) string {
	data := p.data(start)
	var b strings.Builder
	title := ""
	for _, m := range p.maps {
		if m.ID == start {
			title = m.Title
		}
	}
	fmt.Fprintf(&b, `<div class="si-mindmap" data-layout="%s" data-lines="%s" data-nodes="%s" data-mm-theme="%s"`,
		firstNonEmpty(o.Layout, "map"), firstNonEmpty(o.Lines, "curved"), firstNonEmpty(o.Nodes, "box"), firstNonEmpty(o.Theme, "auto"))
	if full {
		b.WriteString(` data-full`)
	}
	if o.WM != "" {
		fmt.Fprintf(&b, ` data-wm="%s" data-wm-pos="%s"`, html.EscapeString(o.WM), firstNonEmpty(o.WMPos, "corner"))
	}
	if o.View == "3d" { // only set when not the default, so 2D markup stays as it was
		fmt.Fprintf(&b, ` data-view="3d"`)
	}
	if o.Camera != "" {
		fmt.Fprintf(&b, ` data-camera="%s"`, html.EscapeString(o.Camera))
	}
	if o.Kind != "" {
		fmt.Fprintf(&b, ` data-kind="%s"`, html.EscapeString(o.Kind))
	}
	if o.Floors != "" {
		fmt.Fprintf(&b, ` data-floors="%s"`, html.EscapeString(o.Floors))
	}
	if o.Flow != "" {
		fmt.Fprintf(&b, ` data-flow="%s"`, html.EscapeString(o.Flow))
	}
	if o.Steps != "" {
		fmt.Fprintf(&b, ` data-steps="%s"`, o.Steps)
	}
	fmt.Fprintf(&b, ` role="group" aria-label="%s">`+"\n", html.EscapeString(title))
	fmt.Fprintf(&b, `<script type="application/json" class="si-mm-data">%s</script>`+"\n", data)
	b.WriteString(`<div class="si-mm-list">`)
	for _, m := range p.maps {
		fmt.Fprintf(&b, `<section id="%s"><h2>%s</h2>`, html.EscapeString(o.Prefix+m.ID), html.EscapeString(m.Title))
		listHTML(&b, m.Kids, o.Prefix)
		b.WriteString("</section>")
	}
	b.WriteString("</div>\n</div>\n")
	return b.String()
}

// data is the JSON the runtime reads: the maps (this file's first, own of
// them, then the inlined linked files) and the one to start on.
func (p *mmParser) data(start string) []byte {
	data, _ := json.Marshal(map[string]any{"start": start, "own": p.own, "maps": p.maps})
	return data
}

func listHTML(b *strings.Builder, list []*mmNode, prefix string) {
	if len(list) == 0 {
		return
	}
	b.WriteString("<ul>")
	for _, n := range list {
		b.WriteString("<li>")
		switch {
		case n.Code != nil:
			fmt.Fprintf(b, `<pre class="si-code"><code>%s</code></pre>`, n.Code.HTML)
		case n.Link != "":
			fmt.Fprintf(b, `<a href="#%s">%s</a>`, html.EscapeString(prefix+n.Link), html.EscapeString(n.T))
		case n.Href != "":
			fmt.Fprintf(b, `<a href="%s">%s</a>`, html.EscapeString(n.Href), html.EscapeString(n.T))
		default:
			b.WriteString(html.EscapeString(n.T))
		}
		if n.Badge != "" {
			fmt.Fprintf(b, " (%s)", html.EscapeString(n.Badge))
		}
		listHTML(b, n.Kids, prefix)
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
}

// mindmapFence renders a ```mindmap fence: options are layout=, lines=,
// nodes=, theme= on the fence line.
func (dr *deckRenderer) mindmapFence(body string, attrs map[string]string, dir string) string {
	o := mmOpts{Layout: attrs["layout"], Lines: attrs["lines"], Nodes: attrs["nodes"], Theme: attrs["theme"], View: attrs["view"], Camera: attrs["camera"], Floors: attrs["floors"], Kind: attrs["kind"], Flow: attrs["flow"]}
	dr.mmN++
	o.Prefix = fmt.Sprintf("mm%d-", dr.mmN)
	_, steps := attrs["steps"]
	if attrs["reveal"] == "branches" {
		steps = true
	}
	for k, v := range attrs {
		if k == "steps" && v == "" {
			continue
		}
		allowed := map[string][]string{"layout": deck.MapLayouts, "lines": deck.MapLines, "nodes": deck.MapNodes, "theme": deck.MapThemes,
			"view": deck.MapViews, "camera": deck.MapCameras, "floors": deck.OnOff, "kind": deck.MapKinds, "flow": deck.OnOff, "reveal": {"branches"}}[k]
		if allowed == nil {
			dr.problem(dr.cur, "mindmap: unknown fence option %q (use layout=, lines=, nodes=, theme=, view=, camera=, floors=, kind=, flow=, steps)", k)
		} else if msg := oneOfMsg("mindmap "+k, v, allowed); msg != "" {
			dr.problem(dr.cur, "%s", msg)
		}
	}
	p := newMMParser(dr, dr.cur)
	start := p.addSource(body, dir, dr.d.Path, "Mind map")
	p.own = len(p.maps)
	p.finish()
	if start == "" {
		dr.problem(dr.cur, "mindmap: no topics (write a list, one topic per item)")
		return ""
	}
	dr.mindmap = true
	if o.View == "3d" {
		dr.mindmap3d = true
	}
	var marks strings.Builder
	if steps {
		sc := dr.cur
		if sc == nil {
			dr.problem(nil, "mindmap: steps works on a slide (a document shows every branch)")
		} else {
			o.Steps = strconv.Itoa(dr.mmN)
			for i := range p.maps[0].Kids {
				fmt.Fprintf(&marks, `<span class="si-mm-step" data-mm="%d" data-step="%d" hidden></span>`+"\n", dr.mmN, sc.steps+1+i)
			}
			sc.steps += len(p.maps[0].Kids)
		}
	}
	return `<figure class="si-figure si-mm-figure">` + p.component(o, start, false) + "</figure>\n" + marks.String()
}

func oneOfMsg(what, val string, allowed []string) string {
	for _, a := range allowed {
		if val == a {
			return ""
		}
	}
	return fmt.Sprintf("unknown %s %q (use %s)", what, val, strings.Join(allowed, ", "))
}

// Mindmap renders a mind map page (`type: mindmap`): every map of the file,
// and of the mind map files it links, in one full-viewport page.
func Mindmap(d *deck.Deck, opt Options) Result {
	dr := &deckRenderer{d: d, opt: opt, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	p := newMMParser(dr, nil)
	p.files[d.Path] = ""
	start := ""
	for _, s := range d.Slides {
		if id := p.addSource(s.Body, s.Dir, d.Path, d.Title()); start == "" {
			start = id
		}
	}
	p.files[d.Path] = start
	p.own = len(p.maps)
	p.finish()
	if start == "" {
		dr.problem(nil, "mindmap: no topics (write a list, one topic per item)")
	}
	m := d.Meta
	wmLayer := dr.watermark(nil) // text goes into the map (and its exports); a picture stays a page layer
	wm := ""
	if dr.wmKind == "text" {
		wm, wmLayer = dr.wmText, ""
	}
	comp := p.component(mmOpts{Layout: m.MapLayout, Lines: m.MapLines, Nodes: m.MapNodes, Theme: m.MapTheme, WM: wm, WMPos: m.WatermarkPos,
		View: m.MapView, Camera: m.MapCamera, Floors: m.MapFloors, Kind: m.MapKind, Flow: m.MapFlow}, start, true)
	dr.mindmap3d = m.MapView == "3d"
	res := Result{}
	dr.finish(&res)
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + assets.MindmapCSS + dr.themeCSS + dr.accentCSS() + "</style>\n")
	if wmLayer != "" {
		decls := `--si-wm-img:url("` + dr.wmImg + `")`
		if m.WatermarkOpacity != "" {
			decls += ";--si-wm-opacity:" + m.WatermarkOpacity
		}
		b.WriteString(`<style id="si-wm">.si-doc{` + decls + "}</style>\n")
	}
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body class=\"si-doc si-mm-page\" data-si-doc=\"mindmap\">\n" + strings.Replace(wmLayer, `class="si-watermark `, `class="si-watermark si-doc-wm `, 1) + "<main class=\"si-mm-main\">\n")
	b.WriteString(comp)
	b.WriteString("</main>\n")
	dr.problemsBanner(&b, res.Problems)
	b.WriteString(dr.mindmapScripts())
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	res.HTML = b.Bytes()
	return res
}

// mindmapScripts is the mind map runtime, with three.js and the 3D renderer
// before it when a map on the page starts in 3D (2D-only pages stay small).
func (dr *deckRenderer) mindmapScripts() string {
	js := ""
	if dr.mindmap3d {
		js = "<script>\n" + assets.ThreeJS + "\n" + assets.ThreeOrbitJS + "\n" + assets.Mindmap3DJS + "</script>\n"
	}
	return js + "<script>\n" + assets.MindmapJS + "</script>\n"
}

// mmBuild parses one mind map for the Studio's map editor: the page itself,
// or (info != "") a ```mindmap fence of it with that opening line and body.
func mmBuild(d *deck.Deck, body, info string, opt Options) (*deckRenderer, *mmParser, string, mmOpts) {
	dr := &deckRenderer{d: d, opt: opt, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	p := newMMParser(dr, nil)
	p.files[d.Path] = ""
	start := ""
	var o mmOpts
	if info != "" {
		a := fenceAttrs(strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(info), "`~")))
		o = mmOpts{Layout: a["layout"], Lines: a["lines"], Nodes: a["nodes"], Theme: a["theme"], Flow: a["flow"]}
		start = p.addSource(body, filepath.Dir(d.Path), d.Path, "Mind map")
	} else {
		for _, s := range d.Slides {
			if id := p.addSource(s.Body, s.Dir, d.Path, d.Title()); start == "" {
				start = id
			}
		}
		m := d.Meta
		dr.watermark(nil)
		wm := ""
		if dr.wmKind == "text" {
			wm = dr.wmText
		}
		o = mmOpts{Layout: m.MapLayout, Lines: m.MapLines, Nodes: m.MapNodes, Theme: m.MapTheme, WM: wm, WMPos: m.WatermarkPos, Flow: m.MapFlow}
	}
	p.files[d.Path] = start
	p.own = len(p.maps)
	p.finish()
	return dr, p, start, o
}

// MindmapView is what the map editor redraws from: the maps, the options
// that change the drawing, and the problems.
type MindmapView struct {
	Data     json.RawMessage   `json:"data"`
	Opts     map[string]string `json:"opts"`
	Problems []deck.Problem    `json:"problems"`
}

// MindmapEdit renders one mind map (see mmBuild) as the editor's data.
func MindmapEdit(d *deck.Deck, body, info string, opt Options) MindmapView {
	dr, p, start, o := mmBuild(d, body, info, opt)
	res := Result{}
	dr.finish(&res)
	return MindmapView{Data: p.data(start), Problems: res.Problems, Opts: map[string]string{
		"layout": firstNonEmpty(o.Layout, "map"), "lines": firstNonEmpty(o.Lines, "curved"), "nodes": firstNonEmpty(o.Nodes, "box"),
		"mm-theme": firstNonEmpty(o.Theme, "auto"), "wm": o.WM, "wm-pos": firstNonEmpty(o.WMPos, "corner"), "flow": o.Flow}}
}

// MindmapCanvas is the map editor's canvas: one mind map, full size, with the
// editing script that sends every change to the Studio as an op.
func MindmapCanvas(d *deck.Deck, body, info string, opt Options) []byte {
	dr, p, start, o := mmBuild(d, body, info, opt)
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + assets.MindmapCSS + dr.themeCSS + dr.accentCSS() + "</style>\n")
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body class=\"si-doc si-mm-page\" data-si-doc=\"mindmap\">\n<main class=\"si-mm-main\">\n")
	b.WriteString(strings.Replace(p.component(o, start, true), `<div class="si-mindmap"`, `<div class="si-mindmap" data-edit`, 1))
	b.WriteString("</main>\n<script>\n" + assets.MindmapJS + "</script>\n<script>\n" + assets.MindmapEditJS + "</script>\n</body>\n</html>\n")
	return b.Bytes()
}
