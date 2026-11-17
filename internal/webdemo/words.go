package webdemo

import (
	"fmt"
	"html"
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
			// ponytail: one picture and one ffmpeg input per word; a long film
			// gets many inputs (fine for demo-length takes)
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
