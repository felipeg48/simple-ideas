package docsite

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The demo videos of si itself (docs/demos, filmed by `si demo run` in the
// demos workflow) on the site. A page asks for an area's videos with a
// <!-- si-video: decks --> line; when $SI_DEMO_VIDEOS names a folder holding
// <area>/all.mp4 (the run's output), the line becomes the whole video with
// captions and chapters, then each feature's clip, and the files are copied
// into videos/<area>/. Without the videos (a local build) the line goes away.

var videoLineRE = regexp.MustCompile(`(?m)^<!-- si-video:\s*([a-z0-9-]+)\s*-->\n?`)

// videos copies what the pages use and renders the blocks.
type videos struct {
	dir   string                       // $SI_DEMO_VIDEOS
	write func(rel, text string) error // into the site source
	done  map[string]bool              // areas copied
}

func newVideos(write func(rel, text string) error) *videos {
	return &videos{dir: os.Getenv("SI_DEMO_VIDEOS"), write: write, done: map[string]bool{}}
}

// chapter is one feature of a run: its start and its title.
type chapter struct{ at, title string }

// replace turns every <!-- si-video: area --> line of a page at rel (a site
// path like features/decks.md) into its videos.
func (v *videos) replace(md, rel string) string {
	up := strings.Repeat("../", strings.Count(rel, "/"))
	return videoLineRE.ReplaceAllStringFunc(md, func(l string) string {
		area := videoLineRE.FindStringSubmatch(l)[1]
		block, err := v.block(area, up)
		if err != nil || block == "" {
			return ""
		}
		return block
	})
}

func (v *videos) block(area, up string) (string, error) {
	src := filepath.Join(v.dir, area)
	if v.dir == "" {
		return "", nil
	}
	if _, err := os.Stat(filepath.Join(src, "all.mp4")); err != nil {
		return "", nil
	}
	chs := readChapters(filepath.Join(src, "chapters.txt"))
	files := []string{"all.mp4", "all.vtt"}
	for _, c := range chs {
		s := slugOf(c.title)
		files = append(files, s+".mp4", s+".vtt")
	}
	if !v.done[area] {
		for _, f := range files {
			raw, err := os.ReadFile(filepath.Join(src, f))
			if err != nil {
				continue // a feature without narration has no .vtt
			}
			if err := v.write("videos/"+area+"/"+f, string(raw)); err != nil {
				return "", err
			}
		}
		v.done[area] = true
	}
	base := up + "videos/" + area + "/"
	var b strings.Builder
	b.WriteString("<div class=\"demo-video\">\n")
	fmt.Fprintf(&b, "<video controls preload=\"metadata\" playsinline src=\"%sall.mp4\"><track kind=\"captions\" srclang=\"en\" label=\"English\" src=\"%sall.vtt\"></video>\n", base, base)
	if len(chs) > 0 {
		b.WriteString("<ol class=\"demo-chapters\">")
		for _, c := range chs {
			fmt.Fprintf(&b, "<li><a href=\"%s%s.mp4\">%s</a> <span>%s</span></li>", base, slugOf(c.title), html.EscapeString(c.title), html.EscapeString(c.at))
		}
		b.WriteString("</ol>\n")
	}
	b.WriteString("</div>\n" + videoCSS)
	return b.String(), nil
}

// readChapters reads `0:20 Start a talk` lines.
func readChapters(p string) []chapter {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []chapter
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if at, title, ok := strings.Cut(strings.TrimSpace(sc.Text()), " "); ok {
			out = append(out, chapter{at, title})
		}
	}
	return out
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// slugOf names a feature's files the way si demo run does.
func slugOf(s string) string {
	return strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

const videoCSS = `<style>
.demo-video{margin:1.2em 0 1.6em}
.demo-video video{display:block;width:100%;height:auto;border-radius:var(--si-radius);border:1px solid var(--si-line);background:#111318}
.demo-chapters{display:flex;flex-wrap:wrap;gap:.4em .9em;margin:.7em 0 0;padding:0;list-style:none;font-size:.88em;counter-reset:ch}
.demo-chapters li{counter-increment:ch}
.demo-chapters li::before{content:counter(ch) ". ";color:var(--si-muted)}
.demo-chapters span{color:var(--si-muted);font-family:var(--si-font-mono);font-size:.9em}
</style>
`
