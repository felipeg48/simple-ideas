// Package videdit is si's video editor: a Markdown file with `type: video`
// lists the touches a recording needs (cuts, freezes, speed, replaced or
// inserted clips, callouts drawn in the theme, blur, zoom, narration, music,
// captions, a re-filmed demo feature) at source times; Build turns it into
// the finished video with ffmpeg and headless Chrome, Preview into the
// si play page. The spec: docs/superpowers/specs/2026-11-17-video-edit.md.
package videdit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// Edit is a parsed `type: video` file.
type Edit struct {
	Path, Dir string
	Title     string
	Sources   []Source
	Join      Trans // between two sources (Kind "" = cut)
	Out       string
	Theme     string
	Fonts     string
	Voice     string
	Script    string // the demo script for refilm (as written)
	Captions  bool
	Words     bool // captions: words: the word being said highlighted
	Items     []*Item
	Problems  []Problem

	said []Said // the say lines as voiced and placed (Build)
}

// Source is one video of the source list, trimmed to [From, To) (To 0: its
// end).
type Source struct {
	File     string
	From, To float64
	Line     int
}

// Problem is something wrong at a line of the file; Warn ones do not fail.
type Problem struct {
	Line int
	Msg  string
	Warn bool
}

// Trans is a transition: an ffmpeg xfade kind and its length.
type Trans struct {
	Kind string
	Dur  float64
}

// When is a time as written: seconds, the end, a chapter or a said phrase,
// plus an offset.
type When struct {
	Sec     float64
	End     bool
	Chapter string
	Said    string
	Off     float64
}

// Num is a length in source pixels, or a percentage of the frame.
type Num struct {
	V   float64
	Pct bool
}

// Px is n in pixels of a frame side of full pixels.
func (n Num) Px(full int) float64 {
	if n.Pct {
		return n.V * float64(full) / 100
	}
	return n.V
}

// Pos is a point: X,Y or a word (top-right …).
type Pos struct {
	X, Y Num
	Word string
}

// Px is the point in source pixels of a w×h frame (a word: its place, a
// margin in from the edges).
func (p Pos) Px(w, h int) (float64, float64) {
	if p.Word != "" {
		fx, fy := words[p.Word][0], words[p.Word][1]
		return fx * float64(w), fy * float64(h)
	}
	return p.X.Px(w), p.Y.Px(h)
}

var words = map[string][2]float64{"top-left": {.06, .08}, "top": {.5, .08}, "top-right": {.94, .08}, "left": {.06, .5}, "center": {.5, .5},
	"right": {.94, .5}, "bottom-left": {.06, .9}, "bottom": {.5, .9}, "bottom-right": {.94, .9}}

// Rect is X,Y WxH.
type Rect struct{ X, Y, W, H Num }

// Px is the rectangle in source pixels.
func (r Rect) Px(w, h int) (x, y, rw, rh float64) {
	return r.X.Px(w), r.Y.Px(h), r.W.Px(w), r.H.Px(h)
}

// Item is one timeline line.
type Item struct {
	Line   int
	Text   string // the line as written
	From   When
	To     *When // a range
	NoTime bool  // `captions on`

	Action  string // see actions
	Str     string // the quoted text (title, bubble, keys, card, say, label, a lower third's name)
	Sub     string // a lower third's role
	File    string // replace with, insert clip|image, music
	Feature string // refilm feature
	Rect    *Rect
	At      *Pos // where a callout sits
	Point   *Pos // a bubble's tail, an arrow's head
	Tail    *Pos // an arrow's tail
	Dur     float64
	HasDur  bool
	Speed   float64
	Level   float64 // duck / music, 0…1
	Trans   *Trans
	Enter   string // pop, fade, slide-in, type, draw, none
	Exit    string // fade, none
	Move    float64
	Font    string
	Size    float64 // source px
	Color   string
	Flags   map[string]bool // box, shadow, round, think, mute, keep-audio, linear
	Badge   string          // speed · badge "…"
	Path    []*Pos          // a moving callout's keyframes (· to A → B, · path A B C): its place over time
	PathDur float64         // · over 2s: how long the move takes (0: the callout's length)
	PathWH  [][2]Num        // a moving rectangle's size at each keyframe (· to X,Y WxH → X,Y WxH); nil: the rectangle's own
	HL      bool            // · highlight: in the --with highlight cut

	Start, End float64 // source seconds, once resolved
}

// Range reports whether the line names a range.
func (it *Item) Range() bool { return it.To != nil }

// IsCallout reports whether the line draws over the picture.
func (it *Item) IsCallout() bool { return calloutKinds[it.Action] }

var calloutKinds = map[string]bool{"title": true, "text": true, "bubble": true, "arrow": true, "box": true, "circle": true, "underline": true,
	"spotlight": true, "keys": true, "badge": true, "lower-third": true}

// rangeOnly and pointOnly: what takes a range, what a single time.
var (
	rangeOnly = map[string]bool{"cut": true, "speed": true, "replace": true, "mute": true, "refilm": true}
	pointOnly = map[string]bool{"trim-start": true, "trim-end": true, "freeze": true, "insert-clip": true, "insert-image": true, "card": true,
		"say": true, "music": true, "zoom": true, "pan": true, "zoom-out": true}
)

// Actions are the line kinds, as the docs and errors name them.
var Actions = []string{"cut", "trim start", "trim end", "freeze", "speed", "replace with", "insert clip", "insert image", "insert card", "card",
	"mute", "title", "text", "bubble", "arrow", "box", "circle", "underline", "spotlight", "blur", "pixelate", "keys", "badge", "lower-third",
	"zoom to", "pan to", "zoom out", "say", "music", "captions on", "captions words", "refilm feature"}

// ParseFile reads and parses a `type: video` file.
func ParseFile(path string) (*Edit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, string(raw)), nil
}

// Is reports whether the file is a video edit (front matter `type: video`).
func Is(path string) bool {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 0; sc.Scan() && n < 40; n++ {
		l := strings.TrimSpace(sc.Text())
		if n == 0 && l != "---" {
			return false
		}
		if n > 0 && l == "---" {
			return false
		}
		if k, v, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(k) == "type" {
			return strings.TrimSpace(stripComment(v)) == "video"
		}
	}
	return false
}

func stripComment(v string) string {
	if i := strings.Index(v, " #"); i >= 0 {
		return v[:i]
	}
	return v
}

// Parse parses the text of a `type: video` file at path.
func Parse(path, src string) *Edit {
	abs, _ := filepath.Abs(path)
	e := &Edit{Path: abs, Dir: filepath.Dir(abs)}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	i := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		i = e.frontMatter(lines)
	} else {
		e.problem(1, "missing front matter (---, type: video, source: …)")
	}
	if len(e.Sources) == 0 {
		e.problem(1, "no source: (the video to edit)")
	}
	type entry struct {
		line int
		text string
	}
	var all, timeline []entry
	inTimeline, sawTimeline, fence := false, false, false
	for ; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		switch {
		case strings.HasPrefix(t, "# ") && e.Title == "":
			e.Title = strings.TrimSpace(t[2:])
		case strings.HasPrefix(t, "## "):
			inTimeline = strings.EqualFold(strings.TrimSpace(t[3:]), "timeline")
			sawTimeline = sawTimeline || inTimeline
		case strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* "):
			en := entry{i + 1, strings.TrimSpace(l[2:])}
			all = append(all, en)
			if inTimeline {
				timeline = append(timeline, en)
			}
		}
	}
	if !sawTimeline {
		timeline = all
	}
	for _, en := range timeline {
		it, err := parseItem(en.text)
		if err != nil {
			e.problem(en.line, "%v", err)
			continue
		}
		it.Line = en.line
		if it.Action == "captions" {
			e.Captions = true
			e.Words = e.Words || it.Str == "words"
		}
		e.Items = append(e.Items, it)
	}
	if e.Out == "" && len(e.Sources) > 0 {
		s := e.Sources[0].File
		e.Out = strings.TrimSuffix(filepath.Base(s), filepath.Ext(s)) + ".edited.mp4"
	}
	return e
}

func (e *Edit) problem(line int, f string, a ...any) {
	e.Problems = append(e.Problems, Problem{Line: line, Msg: fmt.Sprintf(f, a...)})
}

func (e *Edit) warn(line int, f string, a ...any) {
	e.Problems = append(e.Problems, Problem{Line: line, Msg: fmt.Sprintf(f, a...), Warn: true})
}

// Abs is a path of the file resolved against the edit's folder.
func (e *Edit) Abs(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(e.Dir, filepath.FromSlash(p))
}

// frontMatter reads the --- block and returns the index after it.
func (e *Edit) frontMatter(lines []string) int {
	key := ""
	for i := 1; i < len(lines); i++ {
		raw := lines[i]
		t := strings.TrimSpace(raw)
		if t == "---" {
			return i + 1
		}
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "- ") && (raw[0] == ' ' || raw[0] == '\t' || raw[0] == '-') {
			if key != "source" {
				e.problem(i+1, "a list only goes under source:")
				continue
			}
			e.source(i+1, strings.TrimSpace(stripComment(t[2:])))
			continue
		}
		k, v, ok := strings.Cut(t, ":")
		if !ok {
			e.problem(i+1, "front matter: want key: value, found %q", t)
			continue
		}
		key = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(stripComment(v))
		switch key {
		case "type":
			if v != "video" {
				e.problem(i+1, "type: %s is not a video edit", v)
			}
		case "source":
			if v != "" {
				e.source(i+1, v)
			}
		case "out":
			e.Out = v
		case "theme":
			e.Theme = v
		case "fonts":
			e.Fonts = v
		case "voice":
			e.Voice = v
		case "script":
			e.Script = v
		case "title":
			e.Title = v
		case "captions":
			e.Captions = v == "on" || v == "true" || v == "yes" || v == "words"
			e.Words = v == "words"
		case "join":
			tr, err := parseTrans(v)
			if err != nil {
				e.problem(i+1, "join: %v", err)
			} else {
				e.Join = tr
			}
		default:
			e.problem(i+1, "unknown front matter key %q (type, source, out, theme, fonts, voice, captions, script, join, title)", key)
		}
	}
	e.problem(1, "the front matter has no closing ---")
	return len(lines)
}

// source reads `demo.mp4 · from 0:12 · to 5:40`.
func (e *Edit) source(line int, v string) {
	parts := splitDots(v)
	s := Source{File: unquote(parts[0]), Line: line}
	for _, p := range parts[1:] {
		w, rest := cutWord(p)
		t, ok := clock(rest)
		switch {
		case w == "from" && ok:
			s.From = t
		case w == "to" && ok:
			s.To = t
		default:
			e.problem(line, "source %s: %q (want · from 0:12 · to 5:40)", s.File, p)
		}
	}
	if s.To > 0 && s.To <= s.From {
		e.problem(line, "source %s: to is not after from", s.File)
	}
	e.Sources = append(e.Sources, s)
}

// splitDots splits on · outside double quotes, trimming each part.
func splitDots(s string) []string {
	var out []string
	q := false
	start := 0
	for i, r := range s {
		switch {
		case r == '"' || r == '“' || r == '”':
			q = !q
		case r == '·' && !q:
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + len("·")
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// tokens splits on spaces, a "quoted string" one token (quotes kept).
func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	q := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '“' || r == '”':
			r = '"'
			fallthrough
		case r == '"':
			cur.WriteRune(r)
			if q {
				flush()
			}
			q = !q
		case r == ' ' && !q:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func quoted(t string) bool { return len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' }

func unquote(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(strings.TrimPrefix(t, "“"), "\"")
	t = strings.TrimSuffix(strings.TrimSuffix(t, "”"), "\"")
	return t
}

func cutWord(s string) (string, string) {
	s = strings.TrimSpace(s)
	w, rest, _ := strings.Cut(s, " ")
	return strings.ToLower(w), strings.TrimSpace(rest)
}

var (
	hmsRE = regexp.MustCompile(`^(?:(\d+):)?(\d+):(\d{1,2}(?:\.\d+)?)$`)
	secRE = regexp.MustCompile(`^(\d+(?:\.\d+)?)s?$`)
	durRE = regexp.MustCompile(`^(\d+(?:\.\d+)?)(s|ms)$`)
	pctRE = regexp.MustCompile(`^(\d+(?:\.\d+)?)%$`)
	spdRE = regexp.MustCompile(`^(\d+(?:\.\d+)?)[x×]$`)
	numRE = regexp.MustCompile(`^-?\d+(?:\.\d+)?%?$`)
)

// clock reads 1:34, 1:34.5, 1:02:03, 94s, 94.5.
func clock(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if m := hmsRE.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi("0" + m[1])
		mi, _ := strconv.Atoi(m[2])
		sec, _ := strconv.ParseFloat(m[3], 64)
		if sec >= 60 {
			return 0, false
		}
		return float64(h*3600+mi*60) + sec, true
	}
	if m := secRE.FindStringSubmatch(s); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		return v, true
	}
	return 0, false
}

// dur reads 3s, 1.5s, 400ms.
func dur(s string) (float64, bool) {
	m := durRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, false
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	if m[2] == "ms" {
		v /= 1000
	}
	return v, true
}

// Stamp is t as m:ss.s (h:mm:ss over an hour).
func Stamp(t float64) string {
	ms := int(t*10 + .5)
	s := fmt.Sprintf("%d:%02d", ms/600%60, ms/10%60)
	if ms/36000 > 0 {
		s = fmt.Sprintf("%d:%02d:%02d", ms/36000, ms/600%60, ms/10%60)
	}
	if ms%10 != 0 {
		s += fmt.Sprintf(".%d", ms%10)
	}
	return s
}

// when reads one time: a clock, end, start, at chapter "…" ± Ns, at said "…" ± Ns.
func when(s string) (When, error) {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "end":
		return When{End: true}, nil
	case "start":
		return When{}, nil
	}
	if t, ok := clock(s); ok {
		return When{Sec: t}, nil
	}
	rest, ok := strings.CutPrefix(s, "at ")
	if !ok {
		return When{}, fmt.Errorf("%q is not a time (1:34, 1:34.5, end, at chapter \"…\", at said \"…\")", s)
	}
	kind, rest := cutWord(rest)
	ts := tokens(rest)
	if (kind != "chapter" && kind != "said") || len(ts) == 0 || !quoted(ts[0]) {
		return When{}, fmt.Errorf("%q: want at chapter \"Name\" or at said \"words\" (then + 5s or - 1s)", s)
	}
	w := When{}
	if kind == "chapter" {
		w.Chapter = unquote(ts[0])
	} else {
		w.Said = unquote(ts[0])
	}
	off := strings.ReplaceAll(strings.ReplaceAll(strings.Join(ts[1:], ""), " ", ""), "−", "-")
	if off != "" {
		sign := 1.0
		switch off[0] {
		case '+':
		case '-':
			sign = -1
		default:
			return When{}, fmt.Errorf("%q: after the anchor, + 5s or - 1s", s)
		}
		d, ok := dur(strings.TrimLeft(off, "+-"))
		if !ok {
			d, ok = clock(strings.TrimLeft(off, "+-"))
		}
		if !ok {
			return When{}, fmt.Errorf("%q: the offset %q is not seconds (+ 5s)", s, off)
		}
		w.Off = sign * d
	}
	return w, nil
}

var dashRE = regexp.MustCompile(`^(\d[\d:.]*s?|start)\s*-\s*(\d[\d:.]*s?|end)$`)

// times reads the time part: one time or a range.
func times(s string) (When, *When, error) {
	split := func(a, b string) (When, *When, error) {
		f, err := when(a)
		if err != nil {
			return When{}, nil, err
		}
		t, err := when(b)
		if err != nil {
			return When{}, nil, err
		}
		return f, &t, nil
	}
	for _, d := range []string{"–", "—", " to "} {
		if i := outsideQuotes(s, d); i >= 0 {
			return split(s[:i], s[i+len(d):])
		}
	}
	if m := dashRE.FindStringSubmatch(strings.TrimSpace(s)); m != nil {
		return split(m[1], m[2])
	}
	f, err := when(s)
	return f, nil, err
}

func outsideQuotes(s, sub string) int {
	q := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			q = !q
		}
		if !q && strings.HasPrefix(s[i:], sub) {
			return i
		}
	}
	return -1
}

// num reads 820 or 60%.
func num(s string) (Num, bool) {
	s = strings.TrimSpace(s)
	if !numRE.MatchString(s) {
		return Num{}, false
	}
	pct := strings.HasSuffix(s, "%")
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	return Num{v, pct}, err == nil
}

// pos reads X,Y or a word.
func pos(s string) (*Pos, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if _, ok := words[s]; ok {
		return &Pos{Word: s}, nil
	}
	x, y, ok := strings.Cut(s, ",")
	nx, ok1 := num(x)
	ny, ok2 := num(y)
	if !ok || !ok1 || !ok2 {
		return nil, fmt.Errorf("%q is not a position (820,410, 60%%,40%% or top-right, center, bottom …)", s)
	}
	return &Pos{X: nx, Y: ny}, nil
}

var rectRE = regexp.MustCompile(`^(-?[\d.]+%?)\s*,\s*(-?[\d.]+%?)(?:\s*,\s*|\s+)([\d.]+%?)\s*[x×]\s*([\d.]+%?)$`)

// rect reads X,Y WxH (or X,Y,WxH).
func rect(s string) (*Rect, error) {
	m := rectRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return nil, fmt.Errorf("%q is not a rectangle (X,Y WxH: 120,80 400x60)", strings.TrimSpace(s))
	}
	var n [4]Num
	for i := range n {
		n[i], _ = num(m[i+1])
	}
	if n[2].V <= 0 || n[3].V <= 0 {
		return nil, fmt.Errorf("%q: the size must be more than 0", s)
	}
	return &Rect{n[0], n[1], n[2], n[3]}, nil
}

// parseTrans reads fade [0.4s], dissolve, wipe, slide (and the demo kit's
// other kinds), or cut.
func parseTrans(s string) (Trans, error) {
	w, rest := cutWord(s)
	if w == "dissolve" {
		d := 0.4
		if rest != "" {
			v, ok := dur(rest)
			if !ok || v <= 0 || v > 5 {
				return Trans{}, fmt.Errorf("dissolve %q: a length from 0.1s to 5s", rest)
			}
			d = v
		}
		return Trans{"dissolve", d}, nil
	}
	t, err := webdemo.ParseTransition(s)
	if err != nil {
		return Trans{}, err
	}
	if t.Kind == "cut" {
		return Trans{}, nil
	}
	return Trans{t.Kind, t.Dur}, nil
}

var transWords = map[string]bool{"fade": true, "dissolve": true, "wipe": true, "slide": true, "cut": true, "pop": true, "zoom": true, "circle": true,
	"slide-left": true, "slide-right": true, "slide-up": true, "slide-down": true, "wipe-left": true, "wipe-right": true, "wipe-up": true, "wipe-down": true}

// parseItem reads one timeline line.
func parseItem(text string) (*Item, error) {
	parts := splitDots(text)
	it := &Item{Text: text, Flags: map[string]bool{}}
	head := parts[0]
	if l := strings.ToLower(head); l == "captions on" || l == "captions words" {
		it.NoTime, it.Action = true, "captions"
		if l == "captions words" {
			it.Str = "words"
		}
		return it, nil
	}
	if len(parts) < 2 {
		return nil, fmt.Errorf("%q: want TIME · action (- 1:34 · bubble \"…\" · to 820,410)", text)
	}
	from, to, err := times(head)
	if err != nil {
		return nil, err
	}
	it.From, it.To = from, to
	if err := it.action(parts[1]); err != nil {
		return nil, err
	}
	for _, o := range parts[2:] {
		if err := it.option(o); err != nil {
			return nil, err
		}
	}
	if rangeOnly[it.Action] && it.To == nil {
		return nil, fmt.Errorf("%s needs a range (1:36–1:46 · %s)", it.Action, it.Action)
	}
	if pointOnly[it.Action] && it.To != nil {
		return nil, fmt.Errorf("%s takes one time, not a range", strings.ReplaceAll(it.Action, "-", " "))
	}
	switch it.Action {
	case "freeze", "insert-image", "card":
		if !it.HasDur {
			it.Dur = 3
			if it.Action == "freeze" {
				return nil, fmt.Errorf("freeze needs a length (freeze 2s)")
			}
		}
	case "say":
		if it.Level == 0 {
			it.Level = .3
		}
	case "music":
		if it.Level == 0 {
			it.Level = .15
		}
	case "speed":
		if it.Badge != "" {
			it.Enter, it.Exit = "fade", "fade"
		}
	case "arrow":
		if it.Enter == "" {
			it.Enter = "draw"
		}
	case "zoom", "pan", "zoom-out":
		if it.Move == 0 {
			it.Move = .8
		}
	}
	if it.IsCallout() || it.Action == "blur" || it.Action == "pixelate" {
		if !it.HasDur && it.To == nil {
			it.Dur = 3
		}
		if it.Enter == "" {
			it.Enter = "fade"
		}
		if it.Exit == "" {
			it.Exit = "fade"
		}
		if (it.Action == "blur" || it.Action == "pixelate") && it.Rect == nil {
			return nil, fmt.Errorf("%s needs a rectangle (%s 120,80 400x60)", it.Action, it.Action)
		}
	}
	if len(it.Path) > 0 { // the first place is where it starts
		switch it.Action {
		case "bubble", "arrow":
			it.Point = it.Path[0]
		case "box", "circle", "underline", "spotlight":
		default:
			it.At = it.Path[0]
		}
	} else if it.PathDur > 0 {
		return nil, fmt.Errorf("over %gs: give the path (to 100,200 → 400,300 over 2s)", it.PathDur)
	}
	return it, nil
}

// action reads the second part: the action and its own arguments.
func (it *Item) action(p string) error {
	ts := tokens(p)
	if len(ts) == 0 {
		return fmt.Errorf("an empty action")
	}
	w := strings.ToLower(ts[0])
	args := ts[1:]
	rest := strings.TrimSpace(strings.TrimPrefix(p, ts[0]))
	str := func(name string) error {
		if len(args) == 0 || !quoted(args[0]) {
			return fmt.Errorf("%s needs its text in quotes (%s \"…\")", name, name)
		}
		it.Str = unquote(args[0])
		if len(args) > 1 {
			return fmt.Errorf("%s \"…\": put %q after a ·", name, strings.Join(args[1:], " "))
		}
		return nil
	}
	it.Action = w
	switch w {
	case "cut", "mute":
		if len(args) > 0 {
			return fmt.Errorf("%s takes nothing more (%q)", w, rest)
		}
	case "trim":
		if len(args) != 1 || (args[0] != "start" && args[0] != "end") {
			return fmt.Errorf("trim start or trim end")
		}
		it.Action = "trim-" + args[0]
	case "freeze":
		if len(args) > 0 {
			d, ok := dur(args[0])
			if !ok || d <= 0 {
				return fmt.Errorf("freeze %q: a length (freeze 2s)", rest)
			}
			it.Dur, it.HasDur = d, true
		}
	case "speed":
		m := spdRE.FindStringSubmatch(rest)
		if m == nil {
			return fmt.Errorf("speed %q: want 2x or 0.5x", rest)
		}
		it.Speed, _ = strconv.ParseFloat(m[1], 64)
		if it.Speed < .1 || it.Speed > 16 || it.Speed == 1 {
			return fmt.Errorf("speed %sx: from 0.1x to 16x (not 1x)", m[1])
		}
	case "replace":
		f, ok := strings.CutPrefix(rest, "with ")
		if !ok || strings.TrimSpace(f) == "" {
			return fmt.Errorf("replace with FILE.mp4")
		}
		it.File = unquote(f)
	case "insert":
		kind, f := cutWord(rest)
		switch kind {
		case "clip", "image":
			if f == "" {
				return fmt.Errorf("insert %s FILE", kind)
			}
			it.Action, it.File = "insert-"+kind, unquote(f)
		case "card":
			it.Action = "card"
			args = args[1:]
			return str("insert card")
		default:
			return fmt.Errorf("insert clip FILE, insert image FILE or insert card \"…\"")
		}
	case "card", "title", "text", "bubble", "keys", "badge", "say":
		return str(w)
	case "lower-third":
		if len(args) == 0 || !quoted(args[0]) || len(args) > 2 || (len(args) == 2 && !quoted(args[1])) {
			return fmt.Errorf("lower-third \"Name\" \"Role\"")
		}
		it.Str = unquote(args[0])
		if len(args) == 2 {
			it.Sub = unquote(args[1])
		}
	case "arrow":
		f, ok := strings.CutPrefix(rest, "from ")
		a, b, ok2 := strings.Cut(f, " to ")
		if !ok || !ok2 {
			return fmt.Errorf("arrow from X,Y to X,Y")
		}
		var err error
		if it.Tail, err = pos(a); err != nil {
			return err
		}
		if it.Point, err = pos(b); err != nil {
			return err
		}
	case "box", "circle", "underline", "spotlight", "blur", "pixelate":
		r, err := rect(rest)
		if err != nil {
			return fmt.Errorf("%s: %v", w, err)
		}
		it.Rect = r
	case "zoom", "pan":
		k, r := cutWord(rest)
		switch {
		case w == "zoom" && k == "out" && r == "":
			it.Action = "zoom-out"
		case k == "to":
			rr, err := rect(r)
			if err != nil {
				return fmt.Errorf("%s to: %v", w, err)
			}
			it.Rect = rr
		default:
			return fmt.Errorf("zoom to X,Y WxH, pan to X,Y WxH or zoom out")
		}
	case "music":
		if rest == "" {
			return fmt.Errorf("music FILE")
		}
		it.File = unquote(rest)
	case "captions":
		if rest != "on" && rest != "words" {
			return fmt.Errorf("captions on or captions words")
		}
		it.NoTime = true
		if rest == "words" {
			it.Str = "words"
		}
	case "refilm":
		k, r := cutWord(rest)
		if k != "feature" || !quoted(strings.TrimSpace(r)) {
			return fmt.Errorf("refilm feature \"Name\"")
		}
		it.Feature = unquote(r)
	default:
		return fmt.Errorf("unknown action %q (%s)", w, strings.Join(Actions, ", "))
	}
	return nil
}

// moving are the callouts a path moves: a bubble's tail, an arrow's head, a
// text's or key caps' place, a rectangle's top-left corner.
var moving = map[string]bool{"bubble": true, "arrow": true, "text": true, "keys": true, "badge": true, "lower-third": true,
	"box": true, "circle": true, "underline": true, "spotlight": true, "blur": true, "pixelate": true}

var sizeRE = regexp.MustCompile(`^([\d.]+%?)[x×]([\d.]+%?)$`)

// path reads the keyframes of `to A → B → C over 2s` or `path A B C`.
func (it *Item) path(s string) error {
	f := strings.Fields(s)
	if n := len(f); n >= 2 && f[n-2] == "over" {
		d, ok := dur(f[n-1])
		if !ok || d <= 0 {
			return fmt.Errorf("over %q: a length (over 2s)", f[n-1])
		}
		it.PathDur, f = d, f[:n-2]
	}
	it.Path, it.PathWH = nil, nil
	var wh []*[2]Num
	for _, t := range f {
		if m := sizeRE.FindStringSubmatch(t); m != nil { // a rectangle's size at the place before it
			if it.Rect == nil || len(it.Path) == 0 || wh[len(wh)-1] != nil {
				return fmt.Errorf("%q: a size follows a place, for a rectangle (to 10,10 200x80 → 400,300 300x120)", t)
			}
			w, _ := num(m[1])
			h, _ := num(m[2])
			if w.V <= 0 || h.V <= 0 {
				return fmt.Errorf("%q: the size must be more than 0", t)
			}
			wh[len(wh)-1] = &[2]Num{w, h}
			continue
		}
		p, err := pos(t)
		if err != nil {
			return err
		}
		it.Path = append(it.Path, p)
		wh = append(wh, nil)
	}
	if len(it.Path) < 2 {
		return fmt.Errorf("a path needs two places or more (to 100,200 → 400,300 over 2s)")
	}
	sized := false
	for _, s := range wh {
		sized = sized || s != nil
	}
	if sized { // a place without a size keeps the one before it (the first: the rectangle's)
		cur := [2]Num{it.Rect.W, it.Rect.H}
		for _, s := range wh {
			if s != nil {
				cur = *s
			}
			it.PathWH = append(it.PathWH, cur)
		}
	}
	return nil
}

// option reads one · part after the action.
func (it *Item) option(o string) error {
	if o == "" {
		return fmt.Errorf("an empty · part")
	}
	if d, ok := dur(o); ok {
		it.Dur, it.HasDur = d, true
		return nil
	}
	w, rest := cutWord(o)
	if w == "highlight" && rest == "" {
		it.HL = true
		return nil
	}
	if moving[it.Action] {
		switch {
		case (w == "to" || w == "at") && (strings.Contains(rest, "→") || strings.Contains(rest, "->")):
			return it.path(strings.NewReplacer("→", " ", "->", " ").Replace(rest))
		case w == "path":
			return it.path(rest)
		case w == "over":
			d, ok := dur(rest)
			if !ok || d <= 0 {
				return fmt.Errorf("over %q: a length (over 2s)", rest)
			}
			it.PathDur = d
			return nil
		case (w == "ease" || w == "linear") && rest == "":
			it.Flags["linear"] = w == "linear"
			return nil
		}
	}
	if m := pctRE.FindStringSubmatch(o); m != nil && (it.Action == "music" || it.Action == "say") {
		v, _ := strconv.ParseFloat(m[1], 64)
		it.Level = v / 100
		return nil
	}
	edit := it.Action == "replace" || it.Action == "insert-clip" || it.Action == "insert-image" || it.Action == "card" || it.Action == "refilm"
	switch {
	case edit && transWords[w]:
		t, err := parseTrans(o)
		if err != nil {
			return err
		}
		it.Trans = &t
		return nil
	case w == "keep" && rest == "audio" && it.Action == "replace":
		it.Flags["keep-audio"] = true
		return nil
	case w == "mute" && rest == "" && it.Action == "speed":
		it.Flags["mute"] = true
		return nil
	case w == "badge" && it.Action == "speed":
		if !quoted(rest) {
			return fmt.Errorf("badge \"…\"")
		}
		it.Badge = unquote(rest)
		return nil
	case w == "duck" && it.Action == "say":
		m := pctRE.FindStringSubmatch(rest)
		if m == nil {
			return fmt.Errorf("duck %q: a percentage (duck 30%%)", rest)
		}
		v, _ := strconv.ParseFloat(m[1], 64)
		it.Level = v / 100
		return nil
	case (w == "ease" || w == "linear") && rest == "" && (it.Action == "zoom" || it.Action == "pan" || it.Action == "zoom-out"):
		it.Flags["linear"] = w == "linear"
		return nil
	case w == "move" && (it.Action == "zoom" || it.Action == "pan" || it.Action == "zoom-out"):
		d, ok := dur(rest)
		if !ok || d <= 0 {
			return fmt.Errorf("move %q: a length (move 1.2s)", rest)
		}
		it.Move = d
		return nil
	}
	if !it.IsCallout() && it.Action != "blur" && it.Action != "pixelate" {
		return fmt.Errorf("%s: unknown option %q", strings.ReplaceAll(it.Action, "-", " "), o)
	}
	switch w {
	case "pop", "fade", "slide-in", "type", "draw":
		if rest != "" {
			return fmt.Errorf("%q: the entrance takes no value", o)
		}
		it.Enter = w
	case "out":
		if rest != "fade" && rest != "none" {
			return fmt.Errorf("out fade or out none")
		}
		it.Exit = rest
	case "at":
		p, err := pos(rest)
		if err != nil {
			return err
		}
		it.At = p
	case "to":
		p, err := pos(rest)
		if err != nil {
			return err
		}
		it.Point = p
	case "font":
		it.Font = unquote(rest)
	case "size":
		v, err := strconv.ParseFloat(strings.TrimSuffix(rest, "px"), 64)
		if err != nil || v <= 0 {
			return fmt.Errorf("size %q: source pixels (size 48)", rest)
		}
		it.Size = v
	case "color":
		if rest == "" {
			return fmt.Errorf("color #fff or a theme colour (accent, fg, bg, muted, surface)")
		}
		it.Color = rest
	case "box", "shadow", "round", "think":
		it.Flags[w] = true
	default:
		if quoted(o) && it.Action == "arrow" {
			it.Str = unquote(o)
			return nil
		}
		return fmt.Errorf("%s: unknown option %q", it.Action, o)
	}
	return nil
}
