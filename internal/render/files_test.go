package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Issue #8: a course (or guide) whose listed pages do not exist yet reports
// problems like a guide does, never panics.
func TestCourseMissingPages(t *testing.T) {
	const course = "---\ntitle: Repro\nlevel: beginner\n---\n\n- Module 1: One\n"
	cases := []struct {
		name  string
		files map[string]string
		want  []string // problems that must be there (substrings)
	}{
		{"only page missing", map[string]string{"_course.md": course + "  - [Missing](missing.md)\n"},
			[]string{"_course.md:7: missing page missing.md", "_course.md lists no pages"}},
		{"one real, one missing", map[string]string{"_course.md": course + "  - [Real](real.md)\n  - [Missing](missing.md)\n", "real.md": "# Real\n\nText.\n"},
			[]string{"_course.md:8: missing page missing.md"}},
		{"empty page", map[string]string{"_course.md": course + "  - [Empty](empty.md)\n", "empty.md": ""},
			nil},
		{"front matter only", map[string]string{"_course.md": course + "  - [Bare](bare.md)\n", "bare.md": "---\ntitle: Bare\n---\n"},
			nil},
		{"guide, only page missing", map[string]string{"_nav.md": "---\ntitle: G\n---\n\n- [Missing](missing.md)\n"},
			[]string{"_nav.md:5: missing page missing.md", "_nav.md lists no pages"}},
		{"guide, no listed page, one unlisted", map[string]string{"_nav.md": "---\ntitle: G\n---\n\n- [Missing](missing.md)\n", "extra.md": "---\nunlisted: true\n---\n# Extra\n"},
			[]string{"missing page missing.md"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := writeFiles(t, c.files)
			g, err := deck.LoadGuide(dir)
			if err != nil {
				t.Fatal(err)
			}
			_, probs, _ := RenderGuide(g, Options{}) // must not panic
			var all []string
			for _, p := range probs {
				s := p.String()
				all = append(all, s)
				if strings.Contains(s, "_nav.md: missing") || strings.Contains(s, "_course.md: missing") {
					t.Errorf("the message repeats the file name: %s", s)
				}
			}
			joined := strings.Join(all, "\n")
			for _, w := range c.want {
				if !strings.Contains(joined, w) {
					t.Errorf("want %q in\n%s", w, joined)
				}
			}
			if len(probs) == 0 {
				t.Error("want at least one problem")
			}
		})
	}
}

// Issue #9: a video as an image is a player, the file linked and copied
// (never inlined), with a poster from the image directive.
func TestVideoImage(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.md":  "---\ntype: article\n---\n# A\n\n<!-- image: poster=still.svg -->\n![The cluster, narrated](v.mp4)\n\nInline ![clip](clip.webm) too.\n",
		"s.md":  "# Slide\n\n![The page loads](v.mp4 \"Loading\")\n",
		"v.mp4": "x", "clip.webm": "x", "still.svg": tinySVG,
	})
	for _, mode := range []MediaMode{MediaInline, MediaCopy} {
		d, err := deck.Parse(filepath.Join(dir, "a.md"))
		if err != nil {
			t.Fatal(err)
		}
		res := Deck(d, Options{Media: mode})
		h := string(res.HTML)
		for _, want := range []string{
			`<figure class="si-image" data-video><video data-si-player controls preload="metadata" playsinline src="v.mp4" poster="`,
			`<figcaption class="si-caption">`,
			`The cluster, narrated</figcaption>`,
			`<video data-si-player controls preload="metadata" playsinline src="clip.webm"></video>`,
		} {
			if !strings.Contains(h, want) {
				t.Errorf("mode %d: article lacks %q", mode, want)
			}
		}
		if strings.Contains(h, "data:video") || strings.Contains(h, `<img src="v.mp4"`) {
			t.Errorf("mode %d: a video is never inlined nor an <img>", mode)
		}
		if !hasMedia(res, "v.mp4") || !hasMedia(res, "clip.webm") {
			t.Errorf("mode %d: the videos are copied beside the page: %v", mode, res.Media)
		}
	}
	d, err := deck.Parse(filepath.Join(dir, "s.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if h := string(res.HTML); !strings.Contains(h, `data-video><video controls preload="metadata" playsinline src="v.mp4"`) || !strings.Contains(h, ">Loading</figcaption>") || !hasMedia(res, "v.mp4") {
		t.Error("a slide plays a video image too, its title the caption, the file copied")
	}
}

// Issue #9: linked local files and the src of raw <video>/<audio>/<source>
// are copied beside the page; a missing one is a problem.
func TestLinkedFilesCopied(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.md": "---\ntype: article\n---\n# A\n\n[Slides](v.pdf), [data](sub/d.csv?x=1), [page](other.md), [site](https://x.org/a.pdf), [top](#a)\n\n" +
			"<video controls src=\"v.mp4\"><source src=\"v.webm\"></video>\n\n<audio src=\"talk.m4a\"></audio>\n",
		"v.pdf": "%PDF", "sub/d.csv": "a,b", "v.mp4": "x", "v.webm": "x", "talk.m4a": "x", "other.md": "# O\n",
		"bad.md": "---\ntype: article\n---\n# B\n\n[Gone](gone.zip)\n\n<video src=\"nope.mp4\"></video>\n",
	})
	for _, mode := range []MediaMode{MediaInline, MediaCopy} {
		d, _ := deck.Parse(filepath.Join(dir, "a.md"))
		res := Deck(d, Options{Media: mode})
		if len(res.Problems) > 0 {
			t.Fatalf("problems: %v", res.Problems)
		}
		for _, f := range []string{"v.pdf", "sub/d.csv", "v.mp4", "v.webm", "talk.m4a"} {
			if !hasMedia(res, f) {
				t.Errorf("mode %d: %s is not copied: %v", mode, f, res.Media)
			}
		}
		h := string(res.HTML)
		for _, want := range []string{`href="v.pdf"`, `href="sub/d.csv?x=1"`, `href="other.html"`, `href="https://x.org/a.pdf"`, `<source src="v.webm">`} {
			if !strings.Contains(h, want) {
				t.Errorf("mode %d: lacks %s", mode, want)
			}
		}
	}
	d, _ := deck.Parse(filepath.Join(dir, "bad.md"))
	res := Deck(d, Options{})
	var all []string
	for _, p := range res.Problems {
		all = append(all, p.String())
	}
	joined := strings.Join(all, "\n")
	for _, want := range []string{"missing file: gone.zip", "missing file: nope.mp4"} {
		if !strings.Contains(joined, want) {
			t.Errorf("want %q in %s", want, joined)
		}
	}
}

func hasMedia(res Result, rel string) bool {
	for _, m := range res.Media {
		if m.Rel == rel {
			return true
		}
	}
	return false
}

// Issue #9: a deck lesson with · kind=video shows the deck's video, its
// captions, the PDF / handout buttons and the link to the slides page.
func TestCourseDeckVideoLesson(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"_course.md":        "---\ntitle: C\n---\n\n- Module 1: One\n  - [Exchanges](03/ex.md) · kind=video · pdf · handout · slides\n  - [Plain](03/plain.md) · pdf\n",
		"03/ex.md":          "---\ntype: deck\ndescription: How a message finds its queue.\n---\n# Exchanges\n\nNotes: A producer never writes to a queue.\n",
		"03/plain.md":       "# Plain\n\nText.\n",
		"03/ex.mp4":         "x",
		"03/ex.vtt":         "WEBVTT\n",
		"03/ex.pdf":         "%PDF",
		"03/ex.handout.pdf": "%PDF",
	})
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	var gp []string
	for _, p := range g.Problems {
		gp = append(gp, p.String())
	}
	if len(gp) != 1 || !strings.Contains(gp[0], "· pdf: only a deck lesson shown as its video") {
		t.Errorf("want one problem for · pdf on a reading, got %v", gp)
	}
	it := g.Item(filepath.Join(dir, "03/ex.md"))
	if it.Kind != "video" || it.Deck == "" || !it.PDF || !it.Handout || !it.SlidesLink {
		t.Fatalf("lesson: %+v", it)
	}
	pages, _, _ := RenderGuide(g, Options{})
	byRel := map[string]GuidePage{}
	for _, p := range pages {
		byRel[p.Rel] = p
	}
	lesson, ok := byRel["03/ex.html"]
	if !ok {
		t.Fatal("no lesson page")
	}
	h := string(lesson.Result.HTML)
	for _, want := range []string{
		`<figure class="si-course-video"><video data-si-player controls preload="metadata" playsinline src="ex.mp4"><track kind="captions" src="data:text/vtt;base64,`,
		`<a class="si-course-download" href="ex.pdf" download>Download the slides (PDF)</a>`,
		`<a class="si-course-download" href="ex.handout.pdf" download>Download the handout (PDF)</a>`,
		`<a class="si-course-slides-link" href="ex.slides.html">Open the interactive slides</a>`,
		`How a message finds its queue.`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lesson lacks %s", want)
		}
	}
	for _, f := range []string{"ex.mp4", "ex.vtt", "ex.pdf", "ex.handout.pdf"} {
		if !hasMedia(lesson.Result, f) {
			t.Errorf("%s is not copied beside the lesson", f)
		}
	}
	slides, ok := byRel["03/ex.slides.html"]
	if !ok || slides.Deck == nil || slides.Deck.IsDoc() {
		t.Fatal("the interactive slides page is built")
	}
	if strings.Contains(string(slides.Result.HTML), "A producer never writes") {
		t.Error("a learner's slides carry no speaker script")
	}
	if strings.Contains(h, `href="ex.slides.html"`) && strings.Contains(string(byRel["index.html"].Result.HTML), "ex.slides.html") {
		t.Error("the slides page stays out of the menu")
	}
}
