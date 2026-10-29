package deck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write creates files (name -> content) in a temp dir and returns the dir.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		kinds    string  // slide kinds, first letter each: t s c x (closing)
		titles   string  // slide titles joined by |
		minutes  float64 // deck total
		problems []string
	}{
		{
			name:    "title, slides and notes with minutes",
			files:   map[string]string{"d.md": "# Deck\n\nSub\n\nNotes: (0.5 min)\nhello\n\n---\n\n## One\n\n- a\n\nNotes: (2 min) first line\nmore\n"},
			kinds:   "tc",
			titles:  "Deck|One",
			minutes: 2.5,
		},
		{
			name:   "separator inside a fence is code",
			files:  map[string]string{"d.md": "# Deck\n\n---\n\n## Code\n\n```yaml\na: 1\n---\nb: 2\n```\n"},
			kinds:  "tc",
			titles: "Deck|Code",
		},
		{
			name:     "missing title",
			files:    map[string]string{"d.md": "## Only an H2\n\ntext\n"},
			kinds:    "c",
			titles:   "Only an H2",
			problems: []string{"missing title"},
		},
		{
			name:     "unknown directive",
			files:    map[string]string{"d.md": "# Deck\n\n<!-- transmogrify: yes -->\n"},
			kinds:    "t",
			titles:   "Deck",
			problems: []string{`unknown directive "transmogrify"`},
		},
		{
			name:   "ordinary comments are not directives",
			files:  map[string]string{"d.md": "# Deck\n\n<!-- note to self: check this -->\n<!-- TODO -->\n"},
			kinds:  "t",
			titles: "Deck",
		},
		{
			name:     "bad layout and transition values",
			files:    map[string]string{"d.md": "# Deck\n\n---\n\n<!-- layout: diagonal -->\n<!-- transition: spin -->\n## A\n"},
			kinds:    "tc",
			titles:   "Deck|A",
			problems: []string{`unknown layout "diagonal"`, `unknown transition "spin"`},
		},
		{
			name: "include splices a module as a section",
			files: map[string]string{
				"d.md":   "# Deck\n\n---\n\n<!-- include: m.md -->\n\n---\n\n## After\n",
				"m.md":   "---\ntheme: mono\n---\n\n# Module\n\nWhy\n\n---\n\n## Inside\n",
				"x/y.md": "",
			},
			kinds:  "tscc",
			titles: "Deck|Module|Inside|After",
		},
		{
			name: "include cycle refused",
			files: map[string]string{
				"d.md": "# Deck\n\n<!-- include: a.md -->\n",
				"a.md": "# A\n\n<!-- include: b.md -->\n",
				"b.md": "# B\n\n<!-- include: a.md -->\n",
			},
			kinds:    "tsx",
			titles:   "Deck|A|B",
			problems: []string{"include cycle: d.md -> a.md -> b.md -> a.md"},
		},
		{
			name:     "missing include and missing svg",
			files:    map[string]string{"d.md": "# Deck\n\n<!-- include: nope.md -->\n\n---\n\n## S\n\n<!-- svg: assets/none.svg a b -->\n"},
			kinds:    "tc",
			titles:   "Deck|S",
			problems: []string{"include not found: nope.md", "missing file: assets/none.svg"},
		},
		{
			name:   "deck ending on an H1 closes",
			files:  map[string]string{"d.md": "# Deck\n\n---\n\n## A\n\n---\n\n# Thanks\n"},
			kinds:  "tcx",
			titles: "Deck|A|Thanks",
		},
		{
			name:     "front matter: unknown key, bad theme, bad accent",
			files:    map[string]string{"d.md": "---\ntheme: neon\naccent: not a colour!\ncolour: red\n---\n# Deck\n"},
			kinds:    "t",
			titles:   "Deck",
			problems: []string{`unknown theme "neon"`, `accent "not a colour!"`, `unknown front matter key "colour"`},
		},
		{
			name:     "background and image directives are validated",
			files:    map[string]string{"d.md": "# Deck\n\n---\n\n## A\n\n<!-- background: assets/missing.jpg opacity=2 -->\n<!-- image: width=300px -->\n"},
			kinds:    "tc",
			titles:   "Deck|A",
			problems: []string{"opacity=2", "width=300px: fixed pixel sizes"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := write(t, tc.files)
			d, err := Parse(filepath.Join(dir, "d.md"))
			if err != nil {
				t.Fatal(err)
			}
			var kinds, titles []string
			for _, s := range d.Slides {
				k := string(s.Kind[0])
				if s.Kind == KindClosing {
					k = "x"
				}
				kinds = append(kinds, k)
				titles = append(titles, s.Title)
			}
			if got := strings.Join(kinds, ""); got != tc.kinds {
				t.Errorf("kinds = %q, want %q", got, tc.kinds)
			}
			if got := strings.Join(titles, "|"); got != tc.titles {
				t.Errorf("titles = %q, want %q", got, tc.titles)
			}
			if d.Minutes() != tc.minutes {
				t.Errorf("minutes = %v, want %v", d.Minutes(), tc.minutes)
			}
			var msgs []string
			for _, p := range d.Problems {
				msgs = append(msgs, p.Msg)
			}
			all := strings.Join(msgs, "\n")
			for _, want := range tc.problems {
				if !strings.Contains(all, want) {
					t.Errorf("problems %q do not mention %q", all, want)
				}
			}
			if len(tc.problems) == 0 && len(msgs) > 0 {
				t.Errorf("unexpected problems: %q", all)
			}
		})
	}
}

func TestSlideDirectivesAndNotes(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntheme: paper\naccent: \"#e4572e\"\ntransition: zoom\naspect: 4:3\n---\n# Deck\n\n---\n\n" +
		"<!-- layout: two-col -->\n<!-- transition: slide -->\n<!-- class: wide dark -->\n<!-- eyebrow: Context · Q3 -->\n<!-- background: #112233 -->\n## Slide\n\nbody\n\n<!-- column -->\n\nmore\n\nNotes: (1.5 min) say this\n- and this\n\nPoints: talk about X\n- and Y\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil || len(d.Problems) > 0 {
		t.Fatalf("parse: %v %v", err, d.Problems)
	}
	if d.Meta.Theme != "paper" || d.Meta.Accent != "#e4572e" || d.Meta.Transition != "zoom" || d.Meta.Aspect != "4:3" {
		t.Errorf("meta = %+v", d.Meta)
	}
	s := d.Slides[1]
	if s.Layout != "two-col" || s.Transition != "slide" || s.Background != "#112233" || strings.Join(s.Classes, ",") != "wide,dark" || s.Eyebrow != "Context · Q3" {
		t.Errorf("directives = %+v", s)
	}
	if s.Title != "Slide" {
		t.Errorf("title = %q (a heading after slide directives)", s.Title)
	}
	if !strings.Contains(s.Body, "<!-- column -->") || strings.Contains(s.Body, "layout") {
		t.Errorf("body keeps body directives only: %q", s.Body)
	}
	if !s.HasMinutes || s.Minutes != 1.5 || s.Notes != "say this\n- and this" {
		t.Errorf("notes = %q (%v min)", s.Notes, s.Minutes)
	}
	if s.Points != "talk about X\n- and Y" {
		t.Errorf("points = %q (Notes and Points must not swallow each other)", s.Points)
	}
}

func TestParseDirective(t *testing.T) {
	tests := []struct {
		line, name, value string
		ok                bool
	}{
		{"<!-- steps -->", "steps", "", true},
		{"<!-- step: paid -declined -->", "step", "paid -declined", true},
		{"  <!--svg: assets/a.svg x y-->  ", "svg", "assets/a.svg x y", true},
		{"<!-- background: linear-gradient(90deg, #000, #fff) -->", "background", "linear-gradient(90deg, #000, #fff)", true},
		{"<!-- note to self -->", "", "", false},
		{"<!-- TODO -->", "", "", false},
		{"text <!-- step -->", "", "", false},
	}
	for _, tc := range tests {
		name, value, ok := ParseDirective(tc.line)
		if name != tc.name || value != tc.value || ok != tc.ok {
			t.Errorf("ParseDirective(%q) = %q, %q, %v", tc.line, name, value, ok)
		}
	}
}

func TestFooterAndWatermarkMeta(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntitle: T\nfooter: Training Team\nfooter-position: bottom-left\nwatermark: Draft\nwatermark-opacity: 0.08\n---\n\n# T\n\n<!-- footer: off -->\n\nhi\n\n---\n\n## S\n\nbody\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", d.Problems)
	}
	if d.Meta.Footer != "Training Team" || d.Meta.FooterPos != "bottom-left" || d.Meta.Watermark != "Draft" || d.Meta.WatermarkOpacity != "0.08" {
		t.Errorf("meta = %+v", d.Meta)
	}
	if d.Slides[0].Footer != "off" { // per-slide override hides the footer on the title slide
		t.Errorf("title slide footer = %q, want off", d.Slides[0].Footer)
	}
}

func TestFooterPositionInvalid(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntitle: T\nfooter-position: rbc\n---\n\n# T\n\nhi\n"})
	d, _ := Parse(filepath.Join(dir, "d.md"))
	found := false
	for _, p := range d.Problems {
		if strings.Contains(p.Msg, "footer-position") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a footer-position problem, got %v", d.Problems)
	}
}

func TestLabsDirective(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntitle: T\nlabs-label: Hands on\n---\n\n# T\n\nhi\n\n---\n\n<!-- labs: [Create a Cluster](https://a.example/lab), [Scale Out](https://b.example/lab) -->\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", d.Problems)
	}
	if d.Meta.LabsLabel != "Hands on" {
		t.Errorf("labs-label = %q", d.Meta.LabsLabel)
	}
	last := d.Slides[len(d.Slides)-1]
	if len(last.Labs) != 2 || last.Labs[0].Text != "Create a Cluster" || last.Labs[0].URL != "https://a.example/lab" {
		t.Errorf("labs = %+v", last.Labs)
	}
}

func TestLabsBareURL(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntitle: T\n---\n\n# T\n\n<!-- labs: https://only.example/lab -->\n"})
	d, _ := Parse(filepath.Join(dir, "d.md"))
	last := d.Slides[len(d.Slides)-1]
	if len(last.Labs) != 1 || last.Labs[0].URL != "https://only.example/lab" || last.Labs[0].Text != "" {
		t.Errorf("labs = %+v, problems %v", last.Labs, d.Problems)
	}
}

func TestLabsRejectsNonURL(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\ntitle: T\n---\n\n# T\n\n<!-- labs: not-a-url -->\n"})
	d, _ := Parse(filepath.Join(dir, "d.md"))
	found := false
	for _, p := range d.Problems {
		if strings.Contains(p.Msg, "labs") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a labs problem, got %v", d.Problems)
	}
}

func TestParseBackground(t *testing.T) {
	tests := []struct {
		in, kind, value, err string
		check                func(Background) bool
	}{
		{in: "#123456", kind: "colour", value: "#123456"},
		{in: "linear-gradient(135deg, #1d2b64 0%, #3657d6 100%)", kind: "colour", value: "linear-gradient(135deg, #1d2b64 0%, #3657d6 100%)"},
		{in: "none", kind: "none", value: "none"},
		{in: "assets/city.jpg fit=contain position=left-top opacity=0.35 blur=4 overlay=light", kind: "image", value: "assets/city.jpg",
			check: func(b Background) bool {
				return b.Fit == "contain" && b.Position == "left top" && b.Opacity == "0.35" && b.Blur == "4" && b.Overlay == "light"
			}},
		{in: "assets/loop.mp4 poster=assets/p.jpg", kind: "video", value: "assets/loop.mp4", check: func(b Background) bool { return b.Poster == "assets/p.jpg" }},
		{in: "assets/a.jpg opacity=1.5", err: "opacity"},
		{in: "assets/a.jpg overlay=purple", err: "overlay"},
		{in: "#fff blur=3", err: "images and videos only"},
		{in: "fit=cover", err: "needs a value"},
	}
	for _, tc := range tests {
		b, err := ParseBackground(tc.in)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("ParseBackground(%q) err = %v, want %q", tc.in, err, tc.err)
			}
			continue
		}
		if err != nil || b.Kind != tc.kind || b.Value != tc.value || (tc.check != nil && !tc.check(b)) {
			t.Errorf("ParseBackground(%q) = %+v, %v", tc.in, b, err)
		}
	}
}

func TestParseImageAttrs(t *testing.T) {
	tests := []struct {
		in   string
		want ImageAttrs
		err  string
	}{
		{in: "width=40% align=right", want: ImageAttrs{Width: "40%", Align: "right"}},
		{in: "ratio=16:9 fit=cover position=center-top", want: ImageAttrs{Ratio: "16/9", Fit: "cover", Position: "center top"}},
		{in: "width=24em", want: ImageAttrs{Width: "24em"}},
		{in: "width=300px", err: "fixed pixel"},
		{in: "align=up", err: "align"},
		{in: "ratio=wide", err: "ratio"},
		{in: "size=big", err: "image option"},
	}
	for _, tc := range tests {
		a, err := ParseImageAttrs(tc.in)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("ParseImageAttrs(%q) err = %v, want %q", tc.in, err, tc.err)
			}
			continue
		}
		if err != nil || a != tc.want {
			t.Errorf("ParseImageAttrs(%q) = %+v, %v; want %+v", tc.in, a, err, tc.want)
		}
	}
}

func TestList(t *testing.T) {
	dir := write(t, map[string]string{"b.md": "# B", "a.md": "# A", "_module.md": "# M", "README.md": "# R", "notes.txt": ""})
	files, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	if got := strings.Join(names, ","); got != "a.md,b.md" {
		t.Errorf("List = %s", got)
	}
}

func TestLabType(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		lab      bool
		typ      string
		title    string
		body     []string // substrings the single document body must contain
		absent   []string // substrings it must not contain
		problems []string
		warnings []string
		noWarn   []string
	}{
		{name: "no type is a deck", files: map[string]string{"d.md": "# D\n\n---\n\n## A\n"}, typ: "deck"},
		{name: "type deck", files: map[string]string{"d.md": "---\ntype: deck\n---\n# D\n"}, typ: "deck"},
		{name: "bad type", files: map[string]string{"d.md": "---\ntype: book\n---\n# D\n"}, typ: "book", problems: []string{`unknown type "book"`}},
		{name: "lab keeps hr and fenced ---", lab: true, typ: "lab", title: "Lab",
			files: map[string]string{"d.md": "---\ntype: lab\n---\n# Lab\n\n## One\n\ntext\n\n---\n\n```yaml\n---\na: 1\n```\n"},
			body:  []string{"# Lab", "\n---\n", "```yaml\n---\na: 1"}},
		{name: "include expands", lab: true, typ: "lab", title: "Lab",
			files:  map[string]string{"d.md": "---\ntype: lab\n---\n# Lab\n\nintro\n<!-- include: p.md -->\nafter\n", "p.md": "## Part\n\nfrom part\n"},
			body:   []string{"intro\n\n## Part", "from part\n\nafter"},
			absent: []string{"---"}}, // a slide break would be an <hr> or a setext heading
		{name: "front matter title wins", lab: true, typ: "lab", title: "Given",
			files: map[string]string{"d.md": "---\ntype: lab\ntitle: Given\n---\n# Heading\n"}},
		{name: "slide directives warn", lab: true, typ: "lab", title: "Lab",
			files:    map[string]string{"d.md": "---\ntype: lab\n---\n<!-- layout: two-col -->\n# Lab\n\n<!-- steps -->\n- a\n\n```\n<!-- class: x -->\n```\n"},
			warnings: []string{`"layout" has no effect in a lab document`, `"steps" has no effect in a lab document`}},
		{name: "unknown directive warns in a lab", lab: true, typ: "lab", title: "Lab",
			files:    map[string]string{"d.md": "---\ntype: lab\n---\n# Lab\n\n<!-- columns -->\ntext\n"},
			warnings: []string{`unknown directive "columns"`}},
		{name: "deck-only front matter warns in a lab", lab: true, typ: "lab", title: "Lab",
			files: map[string]string{"d.md": "---\ntype: lab\naspect: 4:3\ntransition: zoom\nwatermark: X\nwatermark-opacity: 0.1\nlabs-label: L\nfooter: F\nfooter-position: tl\nauthor: A\n---\n# Lab\n"},
			warnings: []string{`front matter "aspect" has no effect`, `front matter "transition" has no effect`, `front matter "watermark" has no effect`,
				`front matter "watermark-opacity" has no effect`, `front matter "labs-label" has no effect`, `front matter "footer-position" has no effect`},
			noWarn: []string{`"footer"`, `"author"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(filepath.Join(write(t, tc.files), "d.md"))
			if err != nil {
				t.Fatal(err)
			}
			if d.IsLab() != tc.lab || d.Meta.Type != tc.typ {
				t.Errorf("IsLab=%v type=%q", d.IsLab(), d.Meta.Type)
			}
			var msgs, warns []string
			for _, p := range d.Problems {
				msgs = append(msgs, p.Msg)
			}
			for _, p := range d.Warnings {
				warns = append(warns, p.Msg)
			}
			all, allW := strings.Join(msgs, "\n"), strings.Join(warns, "\n")
			for _, w := range tc.problems {
				if !strings.Contains(all, w) {
					t.Errorf("problems %q lack %q", all, w)
				}
			}
			for _, w := range tc.warnings {
				if !strings.Contains(allW, w) {
					t.Errorf("warnings %q lack %q", allW, w)
				}
			}
			for _, w := range tc.noWarn {
				if strings.Contains(allW, w) {
					t.Errorf("warnings %q contain %q", allW, w)
				}
			}
			if len(tc.problems) == 0 && len(msgs) > 0 {
				t.Errorf("unexpected problems %q", all)
			}
			if !tc.lab {
				return
			}
			if len(d.Slides) == 0 || d.Slides[0].Title != tc.title {
				t.Fatalf("slides = %+v", d.Slides)
			}
			var bodies []string
			for _, s := range d.Slides {
				if s.Kind != KindDocument {
					t.Errorf("part kind %q", s.Kind)
				}
				bodies = append(bodies, s.Body)
			}
			body := strings.Join(bodies, "\n")
			for _, w := range tc.body {
				if !strings.Contains(body, w) {
					t.Errorf("body %q lacks %q", body, w)
				}
			}
			for _, w := range tc.absent {
				if strings.Contains(body, w) {
					t.Errorf("body %q contains %q", body, w)
				}
			}
			if strings.Contains(allW, "class") {
				t.Errorf("fenced directive warned: %q", all)
			}
		})
	}
}

// TestLabParts: a lab is split into one part per run of lines of one file, so
// every part resolves paths against its own file and keeps real line numbers.
func TestLabParts(t *testing.T) {
	dir := write(t, map[string]string{
		"d.md":             "---\ntype: lab\n---\n# Lab\n\nintro\n<!-- include: sections/01-a.md -->\nafter\n",
		"sections/01-a.md": "## A\n\n![pic](../images/x.svg)\n",
	})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	type part struct {
		file string
		line int
		dir  string
	}
	var got []part
	for _, s := range d.Slides {
		got = append(got, part{filepath.Base(s.File), s.Line, filepath.Base(s.Dir)})
	}
	want := []part{{"d.md", 4, filepath.Base(dir)}, {"01-a.md", 1, "sections"}, {"d.md", 7, filepath.Base(dir)}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("parts = %v, want %v", got, want)
	}
	if d.Slides[0].Title != "Lab" || d.Slides[1].Title != "" {
		t.Errorf("titles = %q, %q", d.Slides[0].Title, d.Slides[1].Title)
	}
	if !strings.HasPrefix(d.Slides[2].Body, "\nafter") {
		t.Errorf("third part body %q", d.Slides[2].Body)
	}
}

// Studio's generator writes `runbook:` lines; a BOM before the front matter
// and a quoted value (paths with spaces or #) must both parse.
func TestFrontMatterBOMAndQuotedRunbook(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "\uFEFF--- \ntitle: T\n  runbook :  \"labs/x #1.json\"\n---\n\n# T\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta.Title != "T" || d.Meta.Runbook != "labs/x #1.json" {
		t.Errorf("meta = %+v", d.Meta)
	}
}
