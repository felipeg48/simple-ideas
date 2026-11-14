package render

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// The printed course (si build course/ --pdf): one page with every lesson in
// order, printed to PDF by si build. The learner workbook has the readings,
// the lab steps (solutions left out), the quiz questions without their
// answers, and a box for notes after each lesson; the instructor guide
// (--instructor --pdf) has the timings, every demo's runbook, the decks'
// script, the answers with their explanations and the lab solutions.

var (
	solutionRE = regexp.MustCompile(`(?s)<details class="si-solution">.*?</details>`)
	quizWhyRE  = regexp.MustCompile(`(?s)<p class="si-quiz-why" hidden>.*?</p>`)
	quizFootRE = regexp.MustCompile(`(?s)<div class="si-quiz-foot">.*?</div>`)
)

// CourseBookFile is the page's name: workbook.html, or instructor-guide.html.
func CourseBookFile(instructor bool) string {
	if instructor {
		return "instructor-guide.html"
	}
	return "workbook.html"
}

const courseBookCSS = `<style>
body:has(.si-cb) .si-doc-head{display:none}
.si-cb-cover{display:flex;flex-direction:column;justify-content:center}
.si-cb-kicker{font-size:.8em;letter-spacing:.08em;text-transform:uppercase;color:var(--si-muted);margin:0}
.si-cb-plan{width:100%;border-collapse:collapse}
.si-cb-plan td{padding:.3em .5em;border-bottom:1px solid var(--si-line)}
.si-cb-plan td:last-child{text-align:right;white-space:nowrap}
.si-cb-mod{break-before:page}
.si-cb-lesson{margin-top:2.5em}
.si-cb-lesson>h2{margin-top:.2em}
.si-cb-notes{margin-top:1.5em;border:1px solid var(--si-line);border-radius:.4em;min-height:12em;padding:.6em .8em;
  background:repeating-linear-gradient(transparent 0 1.9em,var(--si-line) 1.9em calc(1.9em + 1px))}
.si-cb-notes p{margin:0;color:var(--si-muted);font-size:.85em}
.si-cb-instr{border-left:.25em solid var(--si-accent);padding:.4em .8em;background:var(--si-surface)}
.si-cb-slides li{margin:.6em 0}
.si-cb-slides blockquote{margin:.3em 0 0;font-size:.95em}
.si-cb input[data-ok]+span{font-weight:700}
.si-cb input[data-ok]+span::after{content:"  (right answer)";font-weight:400;color:var(--si-muted)}
@media print{.si-cb-cover{min-height:85vh}.si-cb-lesson{break-inside:auto}.si-cb-notes{break-inside:avoid}}
</style>
`

// CourseBook renders the printed course for the audience of opt (opt.Instructor).
func CourseBook(full *deck.Guide, opt Options) (GuidePage, []deck.Problem) {
	pages, probs, _ := renderCourse(full, opt)
	g := full.ForAudience(opt.Instructor)
	file := filepath.Join(g.Dir, strings.TrimSuffix(CourseBookFile(opt.Instructor), ".html")+".md")
	d := &deck.Deck{Path: file, Name: strings.TrimSuffix(filepath.Base(file), ".md"), Meta: g.Defaults()}
	o := opt
	o.Guide = nil
	dr := &deckRenderer{d: d, opt: o, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	byFile := map[string]GuidePage{}
	for _, p := range pages {
		byFile[p.File] = p
	}
	e := html.EscapeString
	what := "Workbook"
	if opt.Instructor {
		what = "Instructor guide"
	}
	var b strings.Builder
	b.WriteString(courseBookCSS + `<div class="si-cb">`)
	b.WriteString(`<section class="si-cb-cover"><p class="si-cb-kicker">` + what + `</p><h1 class="si-doc-title">` + e(g.Meta.Title) + `</h1>`)
	if d := strings.TrimSpace(g.Meta.Description); d != "" {
		b.WriteString(`<p class="si-lede">` + e(d) + `</p>`)
	}
	var meta []string
	if l := strings.TrimSpace(g.Meta.Level); l != "" {
		meta = append(meta, strings.ToUpper(l[:1])+l[1:])
	}
	if t := courseMinutes(g); t != "" {
		meta = append(meta, t)
	}
	if len(meta) > 0 {
		b.WriteString(`<p class="si-doc-meta"><span>` + strings.Join(escapeAll(meta), "</span><span>") + `</span></p>`)
	}
	if p := strings.TrimSpace(g.Meta.Prerequisites); p != "" {
		b.WriteString(`<p><b>Before you start</b> ` + e(p) + `</p>`)
	}
	b.WriteString(`<table class="si-cb-plan"><tbody>`)
	for i, m := range g.Items {
		fmt.Fprintf(&b, `<tr><td>Module %d · %s</td><td>%s</td></tr>`, i+1, e(deck.ModuleTitle(m.Title)), longMin(deck.ModuleMinutes(m)))
	}
	b.WriteString(`</tbody></table></section>`)
	for i, m := range g.Items {
		fmt.Fprintf(&b, `<section class="si-cb-mod"><p class="si-cb-kicker">Module %d · %s</p><h1>%s</h1>`, i+1, longMin(deck.ModuleMinutes(m)), e(deck.ModuleTitle(m.Title)))
		if m.Prerequisites != "" {
			b.WriteString(`<p><b>Before this module</b> ` + e(m.Prerequisites) + `</p>`)
		}
		for _, it := range m.Children {
			if it.File == "" {
				continue
			}
			kicker := kindName[it.Kind]
			if it.Minutes > 0 {
				kicker += " · " + longMin(it.Minutes)
			}
			fmt.Fprintf(&b, `<section class="si-cb-lesson"><p class="si-cb-kicker">%s</p><h2>%s</h2>`, e(kicker), e(it.Title))
			b.WriteString(dr.courseBookLesson(g, it, byFile[it.File]))
			if !opt.Instructor {
				b.WriteString(`<div class="si-cb-notes"><p>Notes</p></div>`)
			}
			b.WriteString(`</section>`)
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`</div>`)
	d.Meta.Title, d.Meta.Type = g.Meta.Title+" · "+what, "article"
	res := Result{}
	dr.finish(&res)
	body := b.String()
	res.HTML, res.body = dr.docPage(body, nil, nil), body
	return GuidePage{File: file, Rel: CourseBookFile(opt.Instructor), Deck: d, Result: res}, probs
}

// courseBookLesson is one lesson's printed body.
func (dr *deckRenderer) courseBookLesson(g *deck.Guide, it *deck.NavItem, p GuidePage) string {
	instr := dr.opt.Instructor
	var b strings.Builder
	var d *deck.Deck
	if it.Video == "" {
		d, _ = deck.ParseWith(it.File, g.Defaults())
	}
	if instr && d != nil {
		if n := strings.TrimSpace(d.Meta.Instructor); n != "" {
			b.WriteString(`<aside class="si-cb-instr"><b>Instructor</b> ` + dr.inline(n) + `</aside>`)
		}
	}
	switch {
	case it.Video != "":
		b.WriteString(`<p>Watch the video in the course (` + html.EscapeString(filepath.Base(it.Video)) + `).</p>`)
	case it.Kind == "slides" && d != nil:
		b.WriteString(`<ol class="si-cb-slides">`)
		for _, s := range d.Slides {
			t := strings.TrimSpace(s.Title)
			if t == "" && !(instr && strings.TrimSpace(s.Notes) != "") {
				continue
			}
			b.WriteString(`<li>` + dr.inline(t))
			if instr && strings.TrimSpace(s.Notes) != "" {
				b.WriteString(`<blockquote>` + dr.toHTML(deck.ActCueRE.ReplaceAllString(strings.ReplaceAll(s.Notes, "[CLICK]", ""), ""), nil) + `</blockquote>`)
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ol>`)
	default:
		body := p.Result.body
		if instr {
			body = strings.ReplaceAll(body, `<p class="si-quiz-why" hidden>`, `<p class="si-quiz-why">`)
			body = strings.ReplaceAll(body, ` data-ok>`, ` data-ok checked>`)
		} else {
			body = quizWhyRE.ReplaceAllString(solutionRE.ReplaceAllString(body, ""), "")
		}
		b.WriteString(quizFootRE.ReplaceAllString(body, ""))
	}
	return b.String()
}
