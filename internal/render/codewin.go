package render

import (
	"bytes"

	"encoding/json"
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/assets"
	"html"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"github.com/yuin/goldmark/ast"
)

// Code windows: a fence with window= (mac, windows, terminal, none) is drawn
// as a window on a backdrop, with <!-- callout: LINE text at=dx,dy --> notes
// written right after it; a run of fences with the same id= is a show, one
// window whose steps morph (lines that stay glide, new ones fade in). The Go
// side groups the fences and sends each step's highlighted lines as JSON;
// web/codewin.js draws and animates them. On a slide the steps are build
// steps; consecutive slides with the same id morph as the slide changes.

var (
	cwWindows     = []string{"mac", "windows", "terminal", "none"}
	cwThemes      = []string{"auto", "night", "paper", "dusk"}
	cwBackdrops   = []string{"dusk", "ocean", "mint", "graphite", "none"}
	cwTransitions = []string{"morph", "fade", "slide"}
)

type cwNote struct {
	Line int     `json:"line"`
	HTML string  `json:"html"`
	At   *[2]int `json:"at,omitempty"`
}

type cwStep struct {
	Name  string   `json:"name,omitempty"`
	Title string   `json:"title,omitempty"`
	HL    []int    `json:"hl,omitempty"`
	Lines []string `json:"lines"` // highlighted HTML, one per line
	Text  []string `json:"text"`  // the same lines as text: the morph matches them
	Notes []cwNote `json:"notes,omitempty"`
}

// cwShow is a window waiting for its first fence to be rendered.
type cwShow struct {
	opts  map[string]string
	steps []cwStep
}

// isWindow reports whether fence attributes make a code window.
func isWindow(attrs map[string]string) bool {
	_, w := attrs["window"]
	_, id := attrs["id"]
	return w || id
}

// codeWindows groups window fences, their callouts and same-id runs. The
// first fence of a group renders the whole window (cwFence); the others and
// the callout comments leave the tree.
func (dr *deckRenderer) codeWindows(doc ast.Node, src []byte, sc *slideCtx) {
	if dr.cw == nil {
		dr.cw = map[ast.Node]*cwShow{}
	}
	var cur *cwShow
	var curID string
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		switch n := c.(type) {
		case *ast.FencedCodeBlock:
			attrs := map[string]string{}
			if n.Info != nil {
				attrs = fenceAttrs(string(n.Info.Segment.Value(src)))
			}
			if !isWindow(attrs) {
				cur, curID = nil, ""
				break
			}
			lang := strings.ToLower(string(n.Language(src)))
			st := dr.cwStep(n, src, lang, attrs)
			if cur != nil && attrs["id"] != "" && attrs["id"] == curID {
				cur.steps = append(cur.steps, st)
				doc.RemoveChild(doc, c)
				break
			}
			dr.cwValidate(attrs, sc)
			cur, curID = &cwShow{opts: attrs, steps: []cwStep{st}}, attrs["id"]
			dr.cw[n] = cur
			dr.codewin = true
		case *ast.HTMLBlock:
			lines := strings.Split(blockText(n, src), "\n")
			var notes []cwNote
			all := true
			for _, l := range lines {
				co, ok, err := codeshow.ParseCallout(l)
				if !ok {
					all = false
					break
				}
				if err != nil {
					dr.problem(sc, "%v", err)
					continue
				}
				notes = append(notes, cwNote{Line: co.Line, HTML: dr.inline(co.Text), At: co.At})
			}
			if !all {
				cur, curID = nil, ""
				break
			}
			if cur == nil {
				dr.problem(sc, "a callout goes right after a code block with window= or id=")
				doc.RemoveChild(doc, c)
				break
			}
			st := &cur.steps[len(cur.steps)-1]
			for _, no := range notes {
				if no.Line < 1 || no.Line > len(st.Lines) {
					dr.problem(sc, "callout on line %d: the code block has %d lines", no.Line, len(st.Lines))
				}
			}
			st.Notes = append(st.Notes, notes...)
			doc.RemoveChild(doc, c)
		default:
			cur, curID = nil, ""
		}
		c = next
	}
}

func (dr *deckRenderer) cwStep(n *ast.FencedCodeBlock, src []byte, lang string, attrs map[string]string) cwStep {
	var code strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		code.Write(seg.Value(src))
	}
	body := strings.TrimRight(code.String(), "\n")
	st := cwStep{Name: attrs["step"], Title: attrs["title"], Text: strings.Split(body, "\n"), Lines: splitHTMLLines(highlight(body, lang))}
	for k := range lineSet(attrs["hl"]) {
		st.HL = append(st.HL, k)
	}
	sort.Ints(st.HL)
	return st
}

func (dr *deckRenderer) cwValidate(attrs map[string]string, sc *slideCtx) {
	for _, o := range []struct {
		key  string
		vals []string
	}{{"window", cwWindows}, {"theme", cwThemes}, {"backdrop", cwBackdrops}, {"transition", cwTransitions}} {
		if v, ok := attrs[o.key]; ok && v != "" {
			if msg := oneOfMsg("code window "+o.key, v, o.vals); msg != "" {
				dr.problem(sc, "%s", msg)
			}
		}
	}
}

// cwFence writes a window: the figure, its chrome, the data for codewin.js,
// every step as a plain code block (what readers without JavaScript see),
// and on a slide one build step per extra step.
func (dr *deckRenderer) cwFence(sh *cwShow) string {
	o := sh.opts
	win := firstNonEmpty(o["window"], "mac")
	theme := firstNonEmpty(o["theme"], "auto")
	backdrop := firstNonEmpty(o["backdrop"], "none")
	if _, ok := o["window"]; !ok {
		win = "none"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="si-cw" data-window="%s" data-cw-theme="%s" data-backdrop="%s" data-transition="%s"`,
		html.EscapeString(win), html.EscapeString(theme), html.EscapeString(backdrop), html.EscapeString(firstNonEmpty(o["transition"], "morph")))
	if id := o["id"]; id != "" {
		fmt.Fprintf(&b, ` data-cw-id="%s"`, html.EscapeString(id))
	}
	if _, ok := o["numbers"]; ok {
		b.WriteString(` data-numbers`)
	} else if _, ok := o["linenos"]; ok {
		b.WriteString(` data-numbers`)
	}
	if dr.doc || dr.cur == nil {
		b.WriteString(` data-cw-nav`)
	}
	title := firstNonEmpty(sh.steps[0].Title, o["title"])
	fmt.Fprintf(&b, ` aria-label="%s">`, html.EscapeString(firstNonEmpty(title, "Code")))
	b.WriteString(`<div class="si-cw-stage"><div class="si-cw-win">`)
	b.WriteString(cwChrome(win, title))
	b.WriteString(`<div class="si-cw-body"><div class="si-cw-code"></div></div></div></div>`)
	data, _ := json.Marshal(map[string]any{"steps": sh.steps})
	fmt.Fprintf(&b, `<script type="application/json" class="si-cw-data">%s</script>`, data)
	b.WriteString(`<div class="si-cw-steps">`)
	for i, st := range sh.steps {
		fmt.Fprintf(&b, `<div class="si-cw-plain" data-i="%d">`, i)
		if st.Name != "" {
			fmt.Fprintf(&b, `<p class="si-cw-name">%s</p>`, html.EscapeString(st.Name))
		}
		b.WriteString(`<pre class="si-code"><code>` + strings.Join(st.Lines, "\n") + `</code></pre>`)
		if len(st.Notes) > 0 {
			b.WriteString(`<ol class="si-cw-notes">`)
			for _, n := range st.Notes {
				fmt.Fprintf(&b, `<li data-line="%d">%s</li>`, n.Line, n.HTML)
			}
			b.WriteString(`</ol>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	if sc := dr.cur; sc != nil && !dr.doc && len(sh.steps) > 1 {
		for i := 1; i < len(sh.steps); i++ {
			fmt.Fprintf(&b, `<span class="si-step si-cw-mark" data-step="%d" hidden></span>`, sc.steps+i)
			sc.ops = append(sc.ops, "")
		}
		sc.steps += len(sh.steps) - 1
	}
	b.WriteString("</figure>\n")
	return b.String()
}

func cwChrome(win, title string) string {
	t := html.EscapeString(title)
	switch win {
	case "mac":
		return `<div class="si-cw-bar"><span class="si-cw-lights" aria-hidden="true"><i></i><i></i><i></i></span><span class="si-cw-title">` + t + `</span><span class="si-cw-pad"></span></div>`
	case "windows":
		return `<div class="si-cw-bar"><span class="si-cw-title">` + t + `</span><span class="si-cw-ctl" aria-hidden="true"><svg viewBox="0 0 10 10"><path d="M1 5h8"/></svg><svg viewBox="0 0 10 10"><rect x="1" y="1" width="8" height="8"/></svg><svg viewBox="0 0 10 10"><path d="M1 1l8 8M9 1l-8 8"/></svg></span></div>`
	case "terminal":
		return `<div class="si-cw-tabs"><span class="si-cw-tab">` + firstNonEmpty(t, "zsh") + `</span></div><div class="si-cw-prompt" aria-hidden="true">$ <b>cat ` + firstNonEmpty(t, "file") + `</b></div>`
	}
	return ""
}

func (dr *deckRenderer) codewinCSS(b *bytes.Buffer) {
	if dr.codewin {
		b.WriteString(`<style id="si-cw-css">` + "\n" + assets.CodewinCSS + "</style>\n")
	}
}

func (dr *deckRenderer) codewinJS(b *bytes.Buffer) {
	if dr.codewin {
		b.WriteString("<script>\n" + assets.CodewinJS + "</script>\n")
	}
}
