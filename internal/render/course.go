package render

import (
	"fmt"
	"html"
	"math"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Courses: a folder with a _course.md (internal/deck/course.go). Every
// lesson page gets the course shell: a top bar (Menu button, the course, the
// overall progress, search, light/dark), the course navigator (modules with a
// progress ring and minutes, lessons with a kind label, minutes and a done
// tick), the lesson, and a bottom bar (on phones, and for nav: bottom). The
// home page (index.html) and the completion page (complete.html) are
// generated. course.js keeps the progress in localStorage.

// kindName is a lesson kind as the crumb names it.
var kindName = map[string]string{"reading": "Reading", "slides": "Slides", "video": "Video", "lab": "Lab", "demo": "Demo", "quiz": "Quiz"}

// courseLayout is the course navigator: nav: in _course.md, else si.conf
// nav:, else left · accordion · sticky.
func (dr *deckRenderer) courseLayout(g *deck.Guide) deck.NavLayout {
	v := g.Meta.Nav
	if v == "" && dr.opt.Config != nil {
		v = dr.opt.Config.Nav
	}
	n, _ := deck.ParseNav(v)
	if n.Pos == "" {
		n.Pos = "left"
	}
	if !n.StickySet {
		n.Sticky = true
	}
	if n.Pos == "top" {
		n.Tabs = true
	}
	if !n.AccordionSet && !n.Collapsed {
		n.Accordion = true
	}
	return n
}

// lessonMin is a lesson's minutes in the menu ("8m"), "" when unknown.
func lessonMin(m int) string {
	if m <= 0 {
		return ""
	}
	return fmt.Sprintf("%dm", m)
}

// longMin is "35 min" or "4 h 10 min".
func longMin(m int) string {
	switch {
	case m <= 0:
		return ""
	case m < 60:
		return fmt.Sprintf("%d min", m)
	case m%60 == 0:
		return fmt.Sprintf("%d h", m/60)
	}
	return fmt.Sprintf("%d h %d min", m/60, m%60)
}

// courseMinutes is the course length: duration: as written, else the sum.
func courseMinutes(g *deck.Guide) string {
	if d := strings.TrimSpace(g.Meta.Duration); d != "" {
		return d
	}
	n := 0
	for _, m := range g.Items {
		n += deck.ModuleMinutes(m)
	}
	return longMin(n)
}

// countsTowards reports whether a lesson counts in the progress (an
// instructor demo does not: learners only see a placeholder, if anything).
func countsTowards(it *deck.NavItem) bool { return it.Kind != "demo" }

func ringSVG() string {
	c := 2 * math.Pi * 7
	return fmt.Sprintf(`<svg class="si-course-ring" viewBox="0 0 20 20" aria-hidden="true"><circle class="si-course-ring-bg" cx="10" cy="10" r="7"/><circle class="si-course-ring-fg" cx="10" cy="10" r="7" stroke-dasharray="0 %.2f"/></svg>`, c)
}

// courseLessonLink is one lesson in the navigator.
func courseLessonLink(g *deck.Guide, it *deck.NavItem, page string) string {
	cur := ""
	if it.File == page {
		cur = ` aria-current="page"`
	}
	count := ""
	if countsTowards(it) {
		count = ` data-count`
	}
	return fmt.Sprintf(`<li><a href="%s" data-page="%s" data-kind="%s"%s%s><span class="si-course-kind" data-kind="%s">%s</span><span class="si-course-lt">%s</span><span class="si-course-lm">%s</span><span class="si-course-done" aria-hidden="true"></span></a></li>`,
		html.EscapeString(pageHref(page, it.File)), html.EscapeString(guideRel(g, it.File)), it.Kind, count, cur, it.Kind, deck.KindLabel[it.Kind], html.EscapeString(it.Title), lessonMin(it.Minutes))
}

// courseNav renders the modules and their lessons for page; open is the
// module shown open (0-based; -1 for none).
func (dr *deckRenderer) courseNav(g *deck.Guide, page string, open int, nl deck.NavLayout) string {
	var b strings.Builder
	b.WriteString(`<div class="si-course-mods">`)
	for i, m := range g.Items {
		isOpen := i == open || (!nl.Accordion && !nl.Collapsed && !nl.Tabs)
		cls := "si-course-mod"
		if isOpen {
			cls += " is-open"
		}
		if i == open {
			cls += " is-current"
		}
		fmt.Fprintf(&b, `<div class="%s" data-module="%d"><button type="button" class="si-course-mod-btn" aria-expanded="%t"><span class="si-course-num">%02d</span>%s<span class="si-course-mt">%s</span><span class="si-course-mm">%s</span><span class="si-course-chev" aria-hidden="true"></span></button><ul class="si-course-lessons">`,
			cls, i+1, isOpen, i+1, ringSVG(), html.EscapeString(deck.ModuleTitle(m.Title)), longMin(deck.ModuleMinutes(m)))
		for _, it := range m.Children {
			if it.File != "" {
				b.WriteString(courseLessonLink(g, it, page))
			}
		}
		b.WriteString(`</ul></div>`)
	}
	b.WriteString(`</div>`)
	if nl.Tabs && open >= 0 { // top: the open module's lessons as a row under the tabs
		b.WriteString(`<ul class="si-course-tabrow">`)
		for _, it := range g.Items[open].Children {
			if it.File != "" {
				b.WriteString(courseLessonLink(g, it, page))
			}
		}
		b.WriteString(`</ul>`)
	}
	return b.String()
}

// moduleOf is the 0-based module of a lesson file, its 0-based place in
// the module and the module's number of lessons; -1, -1, 0 when it is none.
func moduleOf(g *deck.Guide, file string) (mod, idx, n int) {
	for i, m := range g.Items {
		k, found := 0, -1
		for _, it := range m.Children {
			if it.File == "" {
				continue
			}
			if it.File == file {
				found = k
			}
			k++
		}
		if found >= 0 {
			return i, found, k
		}
	}
	return -1, -1, 0
}

// coursePage writes the body of a course page: the top bar, the navigator,
// the lesson (or the home or completion page), the bottom bar.
func (dr *deckRenderer) coursePage(b *strings.Builder, body, lede string) {
	g, d, m := dr.opt.Guide, dr.d, dr.d.Meta
	page := d.Path
	nl := dr.courseLayout(g)
	view := dr.opt.coursePage
	if view == "" {
		view = "lesson"
	}
	mod, idx, inMod := moduleOf(g, page)
	open := mod
	if open < 0 && len(g.Items) > 0 {
		open = 0
	}
	audience := "learner"
	if dr.opt.Instructor {
		audience = "instructor"
	}
	attrs := fmt.Sprintf(` data-nav="%s" data-view="%s" data-audience="%s"`, nl.Pos, view, audience)
	for _, f := range []struct {
		on   bool
		name string
	}{{nl.Sticky, "sticky"}, {nl.Accordion, "accordion"}, {nl.Collapsed, "collapsed"}, {nl.Tabs, "tabs"}} {
		if f.on {
			attrs += " data-" + f.name
		}
	}
	if m.Header != "off" {
		if h := dr.partial("_header.md"); h != "" {
			b.WriteString("<div class=\"si-doc-site\"><div class=\"si-doc-site-in\">" + h + "</div></div>\n")
		}
	}
	home := pageHref(page, filepath.Join(g.Dir, "index.md"))
	sub := []string{"Course", fmt.Sprintf("%d modules", len(g.Items))}
	if len(g.Items) == 1 {
		sub[1] = "1 module"
	}
	if t := courseMinutes(g); t != "" {
		sub = append(sub, t)
	}
	fmt.Fprintf(b, "<div class=\"si-course\"%s>\n<header class=\"si-course-top\"><button type=\"button\" class=\"si-course-menu\" aria-expanded=\"false\" aria-controls=\"si-course-nav\">Menu</button>"+
		"<a class=\"si-course-brand\" href=\"%s\">%s<small>%s</small></a>"+
		"<div class=\"si-course-overall\" title=\"Your progress\"><span class=\"si-course-pct\">0 %%</span><span class=\"si-course-bar\"><b></b></span></div>"+
		"<div class=\"si-course-tools\">%s</div></header>\n",
		attrs, html.EscapeString(home), html.EscapeString(g.Meta.Title), html.EscapeString(strings.Join(sub, " · ")), searchModeButtons)
	fmt.Fprintf(b, "<div class=\"si-course-body\">\n<nav class=\"si-course-nav\" id=\"si-course-nav\" aria-label=\"%s\"><div class=\"si-course-nav-in\">%s</div></nav>\n<div class=\"si-course-main\">\n",
		html.EscapeString(g.Meta.Title), dr.courseNav(g, page, open, nl))
	switch view {
	case "home":
		b.WriteString(dr.courseHome(g, page))
	case "complete":
		b.WriteString(dr.courseComplete(g))
	default:
		b.WriteString(dr.courseLesson(g, page, mod, idx, inMod, body, lede))
	}
	b.WriteString("</div>\n</div>\n</div>\n")
	if m.FooterPage != "off" {
		if f := dr.partial("_footer.md"); f != "" {
			b.WriteString("<div class=\"si-doc-sitefoot\"><div class=\"si-doc-site-in\">" + f + "</div></div>\n")
		}
	}
}

const searchModeButtons = `<button type="button" class="si-doc-search" data-si-action="search" title="Search (/)" aria-label="Search">` + searchIconSVG + `<span>Search</span><kbd>/</kbd></button>` +
	`<button type="button" class="si-doc-mode" data-si-action="mode" title="Light / dark (d)" aria-label="Light / dark">` + modeIconSVG + `</button>`

// courseLesson is the main column of a lesson page and the bottom bar.
func (dr *deckRenderer) courseLesson(g *deck.Guide, page string, mod, idx, inMod int, body, lede string) string {
	var b strings.Builder
	it := g.Item(page)
	kind := "reading"
	if it != nil {
		kind = it.Kind
	}
	b.WriteString("<main class=\"si-doc-body si-course-lesson\">\n")
	if mod >= 0 {
		fmt.Fprintf(&b, "<p class=\"si-course-crumb\"><span>Module %d · %s</span><span class=\"si-course-crumb-kind\">%s</span></p>\n",
			mod+1, html.EscapeString(deck.ModuleTitle(g.Items[mod].Title)), kindName[kind])
	}
	fmt.Fprintf(&b, "<h1 class=\"si-doc-title\">%s</h1>\n%s", dr.inline(dr.d.Title()), lede)
	var meta []string
	if it != nil && it.Minutes > 0 {
		meta = append(meta, longMin(it.Minutes))
	}
	if mod >= 0 {
		meta = append(meta, fmt.Sprintf("Lesson %d of %d", idx+1, inMod))
		kids := g.Items[mod].Children
		for i, k := range kids {
			if k.File == page && i+1 < len(kids) && kids[i+1].File != "" {
				nx := kids[i+1]
				then := "Then: " + kindName[nx.Kind]
				if nx.Minutes > 0 {
					then += ", " + longMin(nx.Minutes)
				}
				meta = append(meta, then)
			}
		}
	}
	if it != nil && it.Prerequisites != "" {
		meta = append(meta, "Before this: "+it.Prerequisites)
	}
	if len(meta) > 0 {
		b.WriteString("<p class=\"si-course-meta\"><span>" + strings.Join(escapeAll(meta), "</span><span>") + "</span></p>\n")
	}
	if dr.opt.Instructor {
		note := strings.TrimSpace(dr.d.Meta.Instructor)
		if kind == "demo" && note == "" {
			note = "An instructor demo: the learner build leaves it out"
			if g.Meta.InstructorDemos == "show-placeholder" {
				note = "An instructor demo: learners see a placeholder in its place"
			}
			note += ". Run it live, then go on."
		}
		if note != "" {
			b.WriteString("<aside class=\"si-course-instr\"><b>Instructor</b><p>" + dr.inline(note) + "</p></aside>\n")
		}
	}
	b.WriteString(body)
	b.WriteString("<div class=\"si-course-end\" aria-hidden=\"true\"></div>\n")
	b.WriteString(dr.coursePager(g, page))
	b.WriteString("</main>\n")
	where := ""
	if mod >= 0 {
		where = fmt.Sprintf("Module %d · %d / %d", mod+1, idx+1, inMod)
		if it != nil && it.Minutes > 0 {
			where += " · " + longMin(it.Minutes)
		}
	}
	prev, next := dr.opt.Prev, dr.opt.Next
	b.WriteString("<div class=\"si-course-dock\"><div class=\"si-course-where\"><b>" + html.EscapeString(plain(dr.inline(dr.d.Title()))) + "</b><span>" + html.EscapeString(where) + "</span></div>")
	b.WriteString(`<button type="button" class="si-course-outline" aria-controls="si-course-nav">Outline</button>`)
	if prev.Href != "" {
		fmt.Fprintf(&b, `<a class="si-course-back" href="%s">Back</a>`, html.EscapeString(prev.Href))
	}
	if next.Href != "" {
		fmt.Fprintf(&b, `<a class="si-course-go" href="%s">Next</a>`, html.EscapeString(next.Href))
	} else {
		fmt.Fprintf(&b, `<a class="si-course-go" href="%s">Finish</a>`, html.EscapeString(pageHref(page, filepath.Join(g.Dir, "complete.md"))))
	}
	b.WriteString("</div>\n")
	return b.String()
}

func escapeAll(v []string) []string {
	out := make([]string, len(v))
	for i, s := range v {
		out[i] = html.EscapeString(s)
	}
	return out
}

// coursePager is the Previous / Next pair under a lesson, each naming the
// kind and length of the lesson it leads to.
func (dr *deckRenderer) coursePager(g *deck.Guide, page string) string {
	var prev, next *deck.NavItem
	for i, p := range g.Pages {
		if p == page {
			if i > 0 {
				prev = g.Item(g.Pages[i-1])
			}
			if i < len(g.Pages)-1 {
				next = g.Item(g.Pages[i+1])
			}
		}
	}
	dr.opt.Prev, dr.opt.Next = DocLink{}, DocLink{}
	var b strings.Builder
	b.WriteString(`<nav class="si-course-next" aria-label="Lessons">`)
	for _, l := range []struct {
		it        *deck.NavItem
		rel, word string
	}{{prev, "prev", "Previous"}, {next, "next", "Next"}} {
		if l.it == nil {
			continue
		}
		href := pageHref(page, l.it.File)
		if l.rel == "prev" {
			dr.opt.Prev = DocLink{Href: href, Title: l.it.Title}
		} else {
			dr.opt.Next = DocLink{Href: href, Title: l.it.Title}
		}
		small := l.word + " · " + kindName[l.it.Kind]
		if l.it.Minutes > 0 {
			small += " · " + longMin(l.it.Minutes)
		}
		fmt.Fprintf(&b, `<a rel="%s" href="%s"><small>%s</small>%s</a>`, l.rel, html.EscapeString(href), html.EscapeString(small), html.EscapeString(l.it.Title))
	}
	if next == nil {
		fmt.Fprintf(&b, `<a rel="next" href="%s"><small>The end</small>Finish the course</a>`, html.EscapeString(pageHref(page, filepath.Join(g.Dir, "complete.md"))))
	}
	b.WriteString("</nav>\n")
	return b.String()
}

// courseHome is the course home: level, length and modules, Start /
// Continue, one card per module.
func (dr *deckRenderer) courseHome(g *deck.Guide, page string) string {
	var b strings.Builder
	b.WriteString("<main class=\"si-doc-body si-course-home\">\n")
	var lvl []string
	if l := strings.TrimSpace(g.Meta.Level); l != "" {
		lvl = append(lvl, strings.ToUpper(l[:1])+l[1:])
	}
	if t := courseMinutes(g); t != "" {
		lvl = append(lvl, t)
	}
	if len(g.Items) == 1 {
		lvl = append(lvl, "1 module")
	} else {
		lvl = append(lvl, fmt.Sprintf("%d modules", len(g.Items)))
	}
	fmt.Fprintf(&b, "<p class=\"si-course-level\">%s</p>\n<h1 class=\"si-doc-title\">%s</h1>\n", html.EscapeString(strings.Join(lvl, " · ")), dr.inline(g.Meta.Title))
	if desc := strings.TrimSpace(g.Meta.Description); desc != "" {
		b.WriteString("<p class=\"si-lede\">" + dr.inline(desc) + "</p>\n")
	}
	if p := strings.TrimSpace(g.Meta.Prerequisites); p != "" {
		b.WriteString("<p class=\"si-course-prereq\"><b>Before you start</b> " + dr.inline(p) + "</p>\n")
	}
	if ls := g.Lessons(); len(ls) > 0 {
		fmt.Fprintf(&b, "<div class=\"si-course-cta\"><a class=\"si-course-continue\" href=\"%s\" data-start=\"%s\">Start the course</a><span class=\"si-course-cta-where\">Module 1 · lesson 1 of %d</span></div>\n",
			html.EscapeString(pageHref(page, ls[0].File)), html.EscapeString(guideRel(g, ls[0].File)), len(g.Items[0].Children))
	}
	fmt.Fprintf(&b, "<p class=\"si-course-finished\" hidden><a href=\"%s\">You finished the course: see your completion page</a></p>\n", html.EscapeString(pageHref(page, filepath.Join(g.Dir, "complete.md"))))
	b.WriteString("<div class=\"si-course-cards\">\n")
	for i, m := range g.Items {
		var kinds []string
		seen := map[string]bool{}
		n := 0
		first := ""
		for _, it := range m.Children {
			if it.File == "" {
				continue
			}
			if first == "" {
				first = pageHref(page, it.File)
			}
			n++
			if !seen[it.Kind] {
				seen[it.Kind] = true
				kinds = append(kinds, "<span>"+deck.KindLabel[it.Kind]+"</span>")
			}
		}
		lessons := fmt.Sprintf("%d lessons", n)
		if n == 1 {
			lessons = "1 lesson"
		}
		pre := ""
		if m.Prerequisites != "" {
			pre = "<p class=\"si-course-card-pre\">Before: " + html.EscapeString(m.Prerequisites) + "</p>"
		}
		fmt.Fprintf(&b, "<article class=\"si-course-card\" data-module=\"%d\"><p class=\"si-course-card-row\"><span>Module %d</span><span>%s</span></p><h3><a href=\"%s\">%s</a></h3>%s<p class=\"si-course-kinds\">%s</p><p class=\"si-course-card-row\"><span>%s</span><span class=\"si-course-card-pct\">0 %%</span></p></article>\n",
			i+1, i+1, longMin(deck.ModuleMinutes(m)), html.EscapeString(first), html.EscapeString(deck.ModuleTitle(m.Title)), pre, strings.Join(kinds, ""), lessons)
	}
	b.WriteString("</div>\n</main>\n")
	return b.String()
}

// courseComplete is the completion page: how far the learner is, and with
// certificate: on, a printable certificate with the name they type.
func (dr *deckRenderer) courseComplete(g *deck.Guide) string {
	var b strings.Builder
	b.WriteString("<main class=\"si-doc-body si-course-complete\">\n")
	fmt.Fprintf(&b, "<p class=\"si-course-level\">%s</p>\n<h1 class=\"si-doc-title\">%s</h1>\n", "Your progress", dr.inline(g.Meta.Title))
	b.WriteString("<p class=\"si-course-status\" role=\"status\">Finish every lesson to complete the course.</p>\n")
	b.WriteString("<ul class=\"si-course-todo\"></ul>\n")
	if g.Meta.Certificate == "on" {
		fmt.Fprintf(&b, "<div class=\"si-course-cert-form\" hidden><label>Your name <input type=\"text\" class=\"si-course-name\" autocomplete=\"name\"></label><button type=\"button\" class=\"si-course-print\">Print the certificate</button></div>\n"+
			"<div class=\"si-course-cert\" hidden><p class=\"si-course-cert-kicker\">Certificate of completion</p><p class=\"si-course-cert-name\">Your name</p><p>has completed the course</p><p class=\"si-course-cert-title\">%s</p><p class=\"si-course-cert-date\"></p></div>\n",
			html.EscapeString(g.Meta.Title))
	}
	b.WriteString("</main>\n")
	return b.String()
}

// RenderCourse renders a course: every lesson in the course shell, the
// home page and the completion page. A learner build (opt.Instructor false)
// leaves the instructor demos out (or shows a placeholder for each); slide
// decks lose their speaker notes; lab solutions stay folded.
func renderCourse(full *deck.Guide, opt Options) (pages []GuidePage, problems, warnings []deck.Problem) {
	problems = append(problems, full.Problems...)
	warnings = append(warnings, full.Warnings...)
	g := full.ForAudience(opt.Instructor)
	opt.Guide = g
	gen := func(file, title, desc, view, body string, media []MediaFile) GuidePage {
		o := opt
		o.coursePage = view
		o.AssetPrefix = strings.Repeat("../", strings.Count(guideRel(g, file), "/"))
		d := &deck.Deck{Path: file, Name: strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), Meta: g.Defaults()}
		d.Meta.Title, d.Meta.Description = title, desc
		dr := &deckRenderer{d: d, opt: o, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
		dr.md = dr.markdown()
		dr.resolveTheme()
		res := Result{}
		dr.finish(&res)
		res.HTML, res.body, res.Media = dr.docPage(body, nil, nil), body, append(res.Media, media...)
		return GuidePage{File: file, Rel: guideRel(g, file), Deck: d, Result: res}
	}
	for _, f := range g.Pages {
		it := g.Item(f)
		o := opt
		o.AssetPrefix = strings.Repeat("../", strings.Count(guideRel(g, f), "/"))
		switch {
		case it.Video != "":
			body, media := videoBody(f, it)
			desc := ""
			if it.Script != "" {
				if d, err := deck.ParseWith(it.Script, g.Defaults()); err == nil {
					desc = d.Meta.Description
				}
			}
			pages = append(pages, gen(f, it.Title, desc, "", body, media))
		default:
			d, err := deck.ParseWith(f, g.Defaults())
			if err != nil {
				problems = append(problems, deck.Problem{File: deck.Rel(f), Msg: err.Error()})
				continue
			}
			if it.Kind == "demo" && !opt.Instructor { // show-placeholder: the learner sees that it happens live
				body := "<div class=\"si-course-live\"><p class=\"si-course-live-t\">Your instructor shows this live</p>"
				if desc := strings.TrimSpace(d.Meta.Description); desc != "" {
					body += "<p>" + html.EscapeString(desc) + "</p>"
				}
				body += "</div>\n"
				pages = append(pages, gen(f, d.Title(), "", "", body, nil))
				continue
			}
			if !d.IsDoc() { // a deck: the slides in the page, presenting themselves
				res := Deck(d, Options{Media: MediaInline, Theme: opt.Theme, Config: opt.Config, UserCSS: opt.UserCSS, NoSpeakerText: !opt.Instructor})
				problems = append(problems, res.Problems...)
				warnings = append(warnings, res.Warnings...)
				aspect := strings.ReplaceAll(d.Meta.Aspect, ":", "/")
				if aspect == "" {
					aspect = "16/9"
				}
				hint := "Click the slides, then use the arrow keys; v reads them aloud, f fills the screen."
				if opt.Instructor {
					hint = "Click the slides, then use the arrow keys; p opens the teleprompter, f fills the screen."
				}
				body := fmt.Sprintf(`<figure class="si-embed-slides si-course-slides" style="--_aspect:%s"><iframe srcdoc="%s" title="%s" allowfullscreen></iframe><figcaption>%s</figcaption></figure>`+"\n",
					html.EscapeString(aspect), html.EscapeString(string(res.HTML)), html.EscapeString(plain(d.Title())), hint)
				pages = append(pages, gen(f, d.Title(), d.Meta.Description, "", body, nil))
				continue
			}
			res := Deck(d, o)
			problems = append(problems, res.Problems...)
			warnings = append(warnings, res.Warnings...)
			pages = append(pages, GuidePage{File: f, Rel: guideRel(g, f), Deck: d, Result: res})
		}
	}
	pages = append(pages,
		gen(filepath.Join(g.Dir, "index.md"), g.Meta.Title, g.Meta.Description, "home", "", nil),
		gen(filepath.Join(g.Dir, "complete.md"), "Course complete", "", "complete", "", nil))
	listed := map[string]bool{}
	for _, f := range full.Pages {
		listed[f] = true
	}
	for _, f := range full.Unlisted() {
		if !listed[f] {
			warnings = append(warnings, deck.Problem{File: deck.Rel(f), Msg: "not listed in " + deck.CourseFile + ": it is not built"})
		}
	}
	fillGuideSearch(g, pages, nil)
	problems = append(problems, checkGuideLinks(pages, nil)...)
	return pages, problems, warnings
}

// videoBody is the player of a video lesson: the file beside the page
// (copied there by si build, never inlined).
func videoBody(page string, it *deck.NavItem) (string, []MediaFile) {
	if !fileExists(it.Video) {
		return "<div class=\"si-course-live\"><p class=\"si-course-live-t\">This video is not filmed yet</p><p>si build films it from its script.</p></div>\n", nil
	}
	r, err := filepath.Rel(filepath.Dir(page), it.Video)
	if err != nil {
		r = filepath.Base(it.Video)
	}
	r = filepath.ToSlash(r)
	return fmt.Sprintf("<figure class=\"si-course-video\"><video controls preload=\"metadata\" playsinline src=\"%s\"></video></figure>\n", html.EscapeString(r)),
		[]MediaFile{{Src: it.Video, Rel: r}}
}
