package deck

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// A course is a folder with a _course.md: the guide format, `type: course`
// implied. Its top-level items are modules, the links under them lessons.
// A lesson's kind comes from what it links to (see lessonKind); `· kind=…`,
// `· minutes=…` and `· prerequisites=…` after an item set them by hand.
//
//	---
//	title: A first web page
//	nav: left · accordion · sticky
//	---
//	- Module 1: How the web works
//	  - [What a browser does](01/browser.md)
//	  - [A page loads](01/loading.mp4)

// CourseFile makes a folder a course.
const CourseFile = "_course.md"

// LessonKinds are the kinds of lesson, in the order the docs list them.
var LessonKinds = []string{"reading", "slides", "video", "lab", "demo", "quiz"}

// KindLabel is the short label the menu shows for a lesson kind.
var KindLabel = map[string]string{"reading": "Read", "slides": "Slides", "video": "Video", "lab": "Lab", "demo": "Demo", "quiz": "Quiz"}

// IsCourse reports whether the guide is a course (`_course.md`, or `type: course`).
func (g *Guide) IsCourse() bool { return g.Meta.Type == "course" }

var videoExts = map[string]bool{".mp4": true, ".webm": true, ".m4v": true, ".mov": true}

// IsVideoPath reports whether p names a video file a course lesson can be.
func IsVideoPath(p string) bool { return videoExts[strings.ToLower(filepath.Ext(p))] }

var itemOptRE = regexp.MustCompile(`^(kind|minutes|prerequisites)=(.*)$`)

// itemOpts cuts the `· key=value` options off a course item:
// "[Lab](lab.md) · kind=lab · minutes=20" → "[Lab](lab.md)", {kind: lab, minutes: 20}.
func itemOpts(s string) (string, map[string]string) {
	parts := strings.Split(s, " · ")
	opts := map[string]string{}
	keep := []string{parts[0]}
	for _, p := range parts[1:] {
		if m := itemOptRE.FindStringSubmatch(strings.TrimSpace(p)); m != nil {
			opts[m[1]] = strings.TrimSpace(m[2])
			continue
		}
		keep = append(keep, p)
	}
	return strings.Join(keep, " · "), opts
}

var modulePrefixRE = regexp.MustCompile(`(?i)^module\s+\d+\s*[:.\-–—]\s*`)

// ModuleTitle is a module's title without a "Module 2:" lead-in.
func ModuleTitle(t string) string { return modulePrefixRE.ReplaceAllString(t, "") }

// courseLessons checks the module / lesson shape of a course and fills each
// lesson's Kind, Minutes, Video and Prerequisites. opts holds every item's
// `· key=value` options.
func (g *Guide) courseLessons(opts map[*NavItem]map[string]string) {
	file := rel(filepath.Join(g.Dir, CourseFile))
	for _, mod := range g.Items {
		if mod.Href != "" || mod.Sub != "" {
			g.Problems = append(g.Problems, Problem{File: file, Line: mod.Line, Msg: "_course.md: a top-level item is a module (- Module 1: Title); put the lesson under one"})
			continue
		}
		mod.Prerequisites = opts[mod]["prerequisites"]
		if len(mod.Children) == 0 {
			g.Problems = append(g.Problems, Problem{File: file, Line: mod.Line, Msg: fmt.Sprintf("_course.md: module %q has no lessons", mod.Title)})
		}
		for _, it := range mod.Children {
			if it.File == "" {
				if it.Href == "" {
					g.Problems = append(g.Problems, Problem{File: file, Line: it.Line, Msg: "_course.md: a lesson is a link, [Title](page.md) or [Title](video.mp4)"})
				}
				continue
			}
			if len(it.Children) > 0 {
				g.Problems = append(g.Problems, Problem{File: file, Line: it.Line, Msg: "_course.md: a lesson has no lessons of its own (two levels: modules, then lessons)"})
			}
			g.lesson(it, opts[it])
			if strings.TrimSuffix(it.File, filepath.Ext(it.File)) == filepath.Join(g.Dir, "index") {
				g.Problems = append(g.Problems, Problem{File: file, Line: it.Line, Msg: "_course.md: index.html is the course home; name this lesson otherwise"})
			}
		}
	}
}

// lesson fills the kind, minutes and video of one lesson.
func (g *Guide) lesson(it *NavItem, o map[string]string) {
	file := rel(filepath.Join(g.Dir, CourseFile))
	if IsVideoPath(it.File) {
		it.Kind, it.Video = "video", it.File
		it.Minutes = videoMinutes(it.File)
	} else if d, err := ParseWith(it.File, g.Defaults()); err == nil {
		it.Kind = lessonKind(d)
		if it.Kind == "video" { // a web demo script: the video si demo run films
			it.Video = strings.TrimSuffix(it.File, filepath.Ext(it.File)) + ".mp4"
			it.Script = it.File
			if !fileExists(it.Video) {
				g.Warnings = append(g.Warnings, Problem{File: file, Line: it.Line, Msg: fmt.Sprintf("%s is not filmed yet: si build films it (or si demo run %s)", Rel(it.Video), Rel(it.File))})
			} else {
				it.Minutes = videoMinutes(it.Video)
			}
		} else {
			it.Minutes = estimateMinutes(d, it.Kind)
		}
		if m, err := strconv.ParseFloat(d.Meta.Minutes, 64); err == nil && m > 0 {
			it.Minutes = int(math.Ceil(m))
		}
		it.Prerequisites = d.Meta.Prerequisites
	}
	if k := o["kind"]; k != "" {
		if KindLabel[k] == "" {
			g.Problems = append(g.Problems, Problem{File: file, Line: it.Line, Msg: fmt.Sprintf("_course.md: kind=%s is not one of %s", k, strings.Join(LessonKinds, ", "))})
		} else {
			it.Kind = k
		}
	}
	if v := o["minutes"]; v != "" {
		if m, err := strconv.ParseFloat(v, 64); err == nil && m > 0 {
			it.Minutes = int(math.Ceil(m))
		} else {
			g.Problems = append(g.Problems, Problem{File: file, Line: it.Line, Msg: fmt.Sprintf("_course.md: minutes=%s is not a number of minutes", v)})
		}
	}
	if v := o["prerequisites"]; v != "" {
		it.Prerequisites = v
	}
}

// lessonKind is the kind of a lesson page from its type.
func lessonKind(d *Deck) string {
	switch d.Meta.Type {
	case "deck":
		return "slides"
	case "lab":
		return "lab"
	case "quiz":
		return "quiz"
	case "demo":
		if d.Meta.App != "" {
			return "video"
		}
		return "demo"
	}
	return "reading"
}

var leadNumRE = regexp.MustCompile(`^\s*(\d+(?:\.\d+)?)`)

// estimateMinutes guesses a lesson's length: reading time at 230 words a
// minute, a deck's (N min) notes (else a minute a slide), five minutes a lab
// step, a minute a quiz question, a demo's duration:.
func estimateMinutes(d *Deck, kind string) int {
	var words, h2 int
	for _, s := range d.Slides {
		var ft fenceTracker
		for _, l := range strings.Split(s.Body, "\n") {
			if ft.inFence(l) {
				continue
			}
			if h2RE.MatchString(l) {
				h2++
			}
			words += len(strings.Fields(l))
		}
	}
	atLeast := func(n, min int) int {
		if n < min {
			return min
		}
		return n
	}
	switch kind {
	case "slides":
		if m := d.Minutes(); m > 0 {
			return int(math.Ceil(m))
		}
		return atLeast(len(d.Slides), 1)
	case "lab":
		return atLeast(h2*5, 5)
	case "quiz":
		return atLeast(h2, 1)
	case "demo":
		if m := leadNumRE.FindStringSubmatch(d.Meta.Duration); m != nil {
			f, _ := strconv.ParseFloat(m[1], 64)
			return atLeast(int(math.Ceil(f)), 1)
		}
		if m := d.Minutes(); m > 0 {
			return int(math.Ceil(m))
		}
		return atLeast(h2*3, 3)
	}
	return atLeast(int(math.Ceil(float64(words)/230)), 1)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// videoMinutes is an MP4's length in whole minutes (rounded up), read from
// its mvhd box; 0 when it cannot tell (a WebM, a broken file).
func videoMinutes(path string) int {
	s := VideoSeconds(path)
	if s <= 0 {
		return 0
	}
	return int(math.Ceil(s / 60))
}

// VideoSeconds is the length of an MP4 / M4V / MOV file from its movie
// header; 0 when it cannot be read.
func VideoSeconds(path string) float64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0
	}
	// walk boxes from off to end looking for name; return its payload range
	find := func(off, end int64, name string) (int64, int64, bool) {
		hdr := make([]byte, 16)
		for off+8 <= end {
			if _, err := f.ReadAt(hdr[:8], off); err != nil {
				return 0, 0, false
			}
			size, typ, head := int64(binary.BigEndian.Uint32(hdr[:4])), string(hdr[4:8]), int64(8)
			switch size {
			case 1:
				if _, err := f.ReadAt(hdr[8:16], off+8); err != nil {
					return 0, 0, false
				}
				size, head = int64(binary.BigEndian.Uint64(hdr[8:16])), 16
			case 0:
				size = end - off
			}
			if size < head {
				return 0, 0, false
			}
			if typ == name {
				return off + head, off + size, true
			}
			off += size
		}
		return 0, 0, false
	}
	ms, me, ok := find(0, st.Size(), "moov")
	if !ok {
		return 0
	}
	hs, he, ok := find(ms, me, "mvhd")
	if !ok || he-hs < 32 {
		return 0
	}
	b := make([]byte, 32)
	if _, err := f.ReadAt(b, hs); err != nil && err != io.EOF {
		return 0
	}
	var scale, dur uint64
	if b[0] == 1 { // version 1: 64-bit times
		scale, dur = uint64(binary.BigEndian.Uint32(b[20:24])), binary.BigEndian.Uint64(b[24:32])
	} else {
		scale, dur = uint64(binary.BigEndian.Uint32(b[12:16])), uint64(binary.BigEndian.Uint32(b[16:20]))
	}
	if scale == 0 {
		return 0
	}
	return float64(dur) / float64(scale)
}

// Lessons are the course's lessons in order, each with its module.
func (g *Guide) Lessons() []*NavItem {
	var out []*NavItem
	for _, m := range g.Items {
		for _, it := range m.Children {
			if it.File != "" {
				out = append(out, it)
			}
		}
	}
	return out
}

// ModuleMinutes is the sum of a module's lesson minutes.
func ModuleMinutes(mod *NavItem) int {
	n := 0
	for _, it := range mod.Children {
		n += it.Minutes
	}
	return n
}

// ForAudience is the course a build shows: the instructor's has everything;
// a learner's leaves the instructor demos out, unless `instructor-demos:
// show-placeholder` keeps them as placeholders. g itself is not changed.
func (g *Guide) ForAudience(instructor bool) *Guide {
	if instructor || g.Meta.InstructorDemos == "show-placeholder" {
		return g
	}
	c := *g
	c.Items, c.Pages = nil, nil
	for _, m := range g.Items {
		mc := *m
		mc.Children = nil
		for _, it := range m.Children {
			if it.Kind == "demo" {
				continue
			}
			ic := *it
			ic.Parent = &mc
			mc.Children = append(mc.Children, &ic)
			if ic.File != "" {
				c.Pages = append(c.Pages, ic.File)
			}
		}
		c.Items = append(c.Items, &mc)
	}
	return &c
}

// NavLayout is a navigator's `nav:` value.
type NavLayout struct {
	Pos                     string // left, right, top or bottom ("" when not said)
	Sticky, StickySet       bool   // sticky: stays while the page scrolls (StickySet: said either way)
	Accordion, AccordionSet bool   // accordion: one module or section open at a time
	Tabs, Collapsed         bool   // tabs: modules as tabs (top); collapsed: all closed but the current one
	Numbered                bool   // numbered: modules or sections carry numbers
}

// NavPositions are the places a navigator can sit.
var NavPositions = []string{"left", "right", "top", "bottom"}

// ParseNav reads `left · accordion · sticky` (words split by ·, commas or
// spaces): a position and any of sticky (or not-sticky), accordion, tabs,
// collapsed, numbered.
func ParseNav(v string) (NavLayout, error) {
	var n NavLayout
	for _, w := range strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return r == '·' || r == ',' || r == ' ' || r == '\t' }) {
		switch w {
		case "left", "right", "top", "bottom":
			if n.Pos != "" && n.Pos != w {
				return n, fmt.Errorf("nav: one position, not %s and %s", n.Pos, w)
			}
			n.Pos = w
		case "sticky":
			n.Sticky, n.StickySet = true, true
		case "not-sticky", "static", "scrolls":
			n.Sticky, n.StickySet = false, true
		case "accordion":
			n.Accordion, n.AccordionSet = true, true
		case "tabs":
			n.Tabs = true
		case "collapsed":
			n.Collapsed = true
		case "open", "expanded":
			n.Accordion, n.AccordionSet = false, true
		case "numbered":
			n.Numbered = true
		default:
			return n, fmt.Errorf("nav: %q is not a position (left, right, top, bottom) or one of sticky, not-sticky, accordion, tabs, collapsed, open, numbered", w)
		}
	}
	return n, nil
}

var passRE = regexp.MustCompile(`^(\d{1,3})\s*%?$`)

// ParsePass reads a pass mark, "70%" or "70".
func ParsePass(v string) (int, bool) {
	m := passRE.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return n, n <= 100
}
