package webdemo

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Per-word captions (--with words; a video edit's `captions: words`): the
// word being said is highlighted. Word times are the caption's time shared
// by letters, as a cue's phrases are placed (the voice engines give no word
// timing).

// wordAt is one word of a caption and when it starts being said.
type wordAt struct {
	Word string
	At   float64
}

// wordTimes shares c's time among its words by letters (a space counts as
// one), each word starting where its letters start.
func wordTimes(c caption) []wordAt {
	ws := strings.Fields(c.Text)
	total := 0
	for _, w := range ws {
		total += len([]rune(w)) + 1
	}
	var out []wordAt
	n := 0
	for _, w := range ws {
		out = append(out, wordAt{w, c.From + (c.To-c.From)*float64(n)/float64(max(1, total))})
		n += len([]rune(w)) + 1
	}
	return out
}

// vttWords is WebVTT with a timestamp before each word after the first (the
// karaoke form players style with :past and :future).
func vttWords(cs []caption) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, c := range cs {
		ws := wordTimes(c)
		var lines []string
		k := 0
		for _, l := range wrap(c.Text) {
			var parts []string
			for range strings.Fields(l) {
				w := ws[k]
				if k == 0 {
					parts = append(parts, w.Word)
				} else {
					parts = append(parts, "<"+stamp(w.At, ".")+">"+w.Word)
				}
				k++
			}
			lines = append(lines, strings.Join(parts, " "))
		}
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n", stamp(c.From, "."), stamp(c.To, "."), strings.Join(lines, "\n"))
	}
	return b.String()
}

// vttOf is vtt, or vttWords when words.
func vttOf(cs []caption, words bool) string {
	if words {
		return vttWords(cs)
	}
	return vtt(cs)
}

// captionWordHTML is captionHTML with word k highlighted.
func captionWordHTML(text string, k int) string {
	i := 0
	var lines []string
	for _, l := range wrap(text) {
		var ws []string
		for _, w := range strings.Fields(l) {
			if i == k {
				ws = append(ws, "<mark>"+html.EscapeString(w)+"</mark>")
			} else {
				ws = append(ws, html.EscapeString(w))
			}
			i++
		}
		lines = append(lines, strings.Join(ws, " "))
	}
	page := captionHTML("")
	return strings.Replace(page, "<div></div>", "<div><span>"+strings.Join(lines, "<br>")+"</span></div>", 1) + // a span: the flex box would drop the spaces around <mark>
		`<style>mark{background:var(--si-accent,#2563eb);color:#fff;border-radius:.2em;padding:0 .12em;margin:0 -.12em}</style>`
}

// wordPics is a picture per word of each caption (pic draws the HTML into
// a file), each on screen while its word is the one being said.
func wordPics(cs []caption, draw func(page string, i int) (string, error)) ([]pic, error) {
	var out []pic
	n := 0
	for _, c := range cs {
		ws := wordTimes(c)
		for k, w := range ws {
			to := c.To
			if k+1 < len(ws) {
				to = ws[k+1].At
			}
			// a picture per word; ffmpeg plays them as one input (slideshow)
			f, err := draw(captionWordHTML(c.Text, k), n)
			if err != nil {
				return nil, err
			}
			out = append(out, pic{f, w.At, to})
			n++
		}
	}
	return out, nil
}

// slideshow is ffmpeg input arguments that play ps (w×h pictures, in time
// order, none overlapping) as one stream: an ffconcat list beside the
// pictures, each shown from its From until the next starts, a transparent
// picture in the gaps. One input and one overlay however many pictures (an
// overlay per picture made ffmpeg slow on long films, see the CHANGELOG
// for v3.33.0). ok is false when the pictures overlap or the files cannot
// be written: lay them one by one then.
func slideshow(ps []pic, w, h int) (args []string, ok bool) {
	if len(ps) < 2 {
		return nil, false
	}
	for i := 1; i < len(ps); i++ {
		if ps[i].From < ps[i-1].To-1e-3 || ps[i].From < ps[i-1].From {
			return nil, false
		}
	}
	dir := filepath.Dir(ps[0].PNG)
	blank := filepath.Join(dir, fmt.Sprintf("blank-%dx%d.png", w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		return nil, false
	}
	if err := os.WriteFile(blank, buf.Bytes(), 0o644); err != nil {
		return nil, false
	}
	var b strings.Builder
	b.WriteString("ffconcat version 1.0\n")
	ms := func(t float64) int64 { return int64(math.Round(t * 1000)) }
	at := int64(0) // where the list is, in ms: durations add up to the pictures' own times, no drift
	entry := func(file string, to int64) {
		q := strings.ReplaceAll(filepath.ToSlash(file), "'", `'\''`)
		fmt.Fprintf(&b, "file '%s'\nduration %.3f\n", q, float64(to-at)/1000)
		at = to
	}
	for _, p := range ps {
		if ms(p.From) > at {
			entry(blank, ms(p.From))
		}
		entry(p.PNG, max(ms(p.To), at+1))
	}
	entry(blank, at+1000)
	list := filepath.Join(dir, "pictures.ffconcat")
	if err := os.WriteFile(list, []byte(b.String()), 0o644); err != nil {
		return nil, false
	}
	return []string{"-f", "concat", "-safe", "0", "-i", list}, true
}

// VTTWords is VTT with word stamps (per-word captions).
func VTTWords(cues []Cue) string {
	var cs []caption
	for _, c := range cues {
		cs = append(cs, captions(c.Text, c.At, c.Seconds)...)
	}
	return vttWords(cs)
}

// WordPics are the per-word pictures of the captions variant: draw writes a
// page (CaptionHTML's look, the word highlighted) to the i-th file.
func WordPics(cues []Cue, draw func(page string, i int) (string, error)) ([]Pic, error) {
	var cs []caption
	for _, c := range cues {
		cs = append(cs, captions(c.Text, c.At, c.Seconds)...)
	}
	ps, err := wordPics(cs, draw)
	out := make([]Pic, len(ps))
	for i, p := range ps {
		out[i] = Pic(p)
	}
	return out, err
}
