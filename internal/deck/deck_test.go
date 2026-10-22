package deck

import (
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
		"<!-- layout: two-col -->\n<!-- transition: slide -->\n<!-- class: wide dark -->\n<!-- background: #112233 -->\n## Slide\n\nbody\n\n<!-- column -->\n\nmore\n\nNotes: (1.5 min) say this\n- and this\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil || len(d.Problems) > 0 {
		t.Fatalf("parse: %v %v", err, d.Problems)
	}
	if d.Meta.Theme != "paper" || d.Meta.Accent != "#e4572e" || d.Meta.Transition != "zoom" || d.Meta.Aspect != "4:3" {
		t.Errorf("meta = %+v", d.Meta)
	}
	s := d.Slides[1]
	if s.Layout != "two-col" || s.Transition != "slide" || s.Background != "#112233" || strings.Join(s.Classes, ",") != "wide,dark" {
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
