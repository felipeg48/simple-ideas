package webdemo

import (
	"encoding/base64"
	"fmt"
	"html"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// The promo kit: the brand around a film (intro: and outro: cards, music:
// under the voice, lower thirds) and the short cuts made from the same take
// (--with vertical, --with highlight).

// Brand is what a film carries of the product: cards before and after it,
// music, the logo, the address, the theme of the cards.
type Brand struct {
	Intro, Outro *Card
	Music        string  // a sound file (absolute), looped under the whole film
	Volume       float64 // the music's level under the voice (0 to 1)
	Logo         string  // an image (absolute) on the intro and outro cards
	URL          string  // the address the outro shows
	Theme        string  // a built-in theme for every card ("" = studio)
}

// Card is an intro: or outro: card: a title and a line under it.
type Card struct{ Title, Subtitle string }

// How long the brand's cards and a lower third stay on screen.
const (
	introSeconds = 3.5
	outroSeconds = 4.5
	thirdSeconds = 4.0
)

// defaultVolume is music: under the voice when no level is given.
const defaultVolume = 0.25

// brand reads intro:, outro:, music:, logo:, url: and theme:.
func (s *Script) brand(fm map[string]string, line func(string) int) {
	b := &s.Brand
	for _, k := range []string{"intro", "outro"} {
		v := strings.TrimSpace(fm[k])
		if v == "" || v == "off" {
			continue
		}
		c := &Card{}
		if v != "on" {
			c.Title, c.Subtitle = splitThird(v)
		}
		if k == "intro" {
			b.Intro = c
		} else {
			b.Outro = c
		}
	}
	file := func(key, v string) string {
		if v = strings.Trim(strings.TrimSpace(v), `"'`); v == "" {
			return ""
		}
		p := v
		if !filepath.IsAbs(p) {
			p = filepath.Join(s.Dir(), p)
		}
		if _, err := os.Stat(p); err != nil {
			s.problem(line(key), "%s: %s is not a file next to the script", key, v)
		}
		return p
	}
	b.Logo, b.URL = file("logo", fm["logo"]), strings.TrimSpace(fm["url"])
	if v := fm["music"]; v != "" {
		cs := clauses(v)
		b.Music, b.Volume = file("music", cs[0]), defaultVolume
		if len(cs) > 1 {
			if vol, ok := level(cs[1]); ok {
				b.Volume = vol
			} else {
				s.problem(line("music"), "music %q: after ·, the music's level under the voice: 25%% or 0.25", v)
			}
		}
	}
	if t := fm["theme"]; t != "" {
		if _, ok := assets.Themes[assets.CanonicalTheme(t)]; !ok {
			s.problem(line("theme"), "theme %q: a built-in theme (%s)", t, strings.Join(assets.ThemeNames(), ", "))
		}
		b.Theme = t
	}
}

// level reads 25% or 0.25 (0 to 1).
func level(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	k := 1.0
	if strings.HasSuffix(v, "%") {
		v, k = strings.TrimSuffix(v, "%"), 100
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f / k, err == nil && f/k > 0 && f/k <= 1
}

// ParseHeaderOffset reads header-offset: auto (-1), none (0), 64 or 64px.
func ParseHeaderOffset(v string) (int, error) {
	switch v = strings.ToLower(strings.TrimSpace(v)); v {
	case "auto":
		return -1, nil
	case "none", "0", "0px":
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(v, "px"))
	if err != nil || n < 0 || n > 2000 {
		return 0, fmt.Errorf("header-offset %q: auto, none or the header's height in CSS pixels (64px)", v)
	}
	return n, nil
}

// third is a lower third on the take's clock.
type third struct {
	Name, Sub string
	From, To  float64
}

// splitThird reads "Name · Role".
func splitThird(v string) (string, string) {
	name, sub, _ := strings.Cut(v, "·")
	return strings.Trim(strings.TrimSpace(name), `"`), strings.Trim(strings.TrimSpace(sub), `"`)
}

// thirdStep reads `lower-third "Name" · "Role" · for 4s` (or "Name · Role").
func thirdStep(st *Step, rest string) (*Step, error) {
	bad := fmt.Errorf(`%q: write lower-third "Name" · "Role" (· for 4s)`, st.Text)
	parts := clauses(rest)
	q, tail, err := quoted(parts[0])
	if err != nil || q == "" || tail != "" {
		return nil, bad
	}
	st.Kind, st.Opts = "lower-third", map[string]string{}
	st.Value.Text, st.Opts["sub"] = splitThird(q)
	for _, c := range parts[1:] {
		if k, v := cutWord(c); k == "for" {
			d, ok := duration(v)
			if !ok || d <= 0 {
				return nil, bad
			}
			st.For = d
		} else if sub, t, err := quoted(c); err == nil && t == "" && st.Opts["sub"] == "" {
			st.Opts["sub"] = sub
		} else {
			return nil, bad
		}
	}
	return st, nil
}

// lowerThird puts a lower third on the take's clock (nothing in a test).
func (r *runner) lowerThird(st *Step) {
	tk := r.tk
	if tk == nil || !tk.on {
		return
	}
	d := thirdSeconds
	if st.For > 0 {
		d = st.For.Seconds()
	}
	at := time.Since(tk.t0).Seconds()
	tk.thirds = append(tk.thirds, third{st.Value.Text, st.Opts["sub"], at, at + d})
}

// firstSentence is the share of a Say: line its first sentence takes (by
// letters, as captions share the voice's time).
func firstSentence(say string) float64 {
	say = strings.TrimSpace(say)
	if say == "" {
		return 1
	}
	for i := 0; i < len(say)-1; i++ {
		if strings.ContainsRune(".!?", rune(say[i])) && say[i+1] == ' ' {
			return float64(i+1) / float64(len(say))
		}
	}
	return 1
}

// dataURI inlines an image file for a card ("" when it cannot be read).
func dataURI(file string) string {
	raw, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	t := mime.TypeByExtension(strings.ToLower(filepath.Ext(file)))
	if t == "" {
		t = "image/png"
	}
	return "data:" + t + ";base64," + base64.StdEncoding.EncodeToString(raw)
}

// brandCardHTML is the intro (or the outro, with its address large): the
// logo, a title, a line under it, the address.
func brandCardHTML(theme, logo, title, sub, url string, outro bool) string {
	var b strings.Builder
	b.WriteString(`<main>`)
	if src := dataURI(logo); logo != "" && src != "" {
		b.WriteString(`<img src="` + src + `" alt="">`)
	}
	if title != "" {
		b.WriteString(`<h1>` + cardInline(title) + `</h1>`)
	}
	if sub != "" {
		b.WriteString(`<p class="s">` + cardInline(sub) + `</p>`)
	}
	if url != "" {
		b.WriteString(`<p class="u">` + html.EscapeString(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")) + `</p>`)
	}
	b.WriteString(`</main>`)
	u := "clamp(16px,2.4vw,30px)"
	if outro {
		u = "clamp(22px,4vw,56px)"
	}
	return themedPage(theme, b.String(), `body{background:var(--si-bg);color:var(--si-fg);font-family:var(--si-font-body);display:grid;place-items:center;text-align:center}
main{width:min(84vw,64rem);display:grid;justify-items:center;gap:1.1rem}
img{max-width:min(40vw,22rem);max-height:22vh;object-fit:contain}
h1{margin:0;font-family:var(--si-font-heading);font-weight:var(--si-heading-weight,760);letter-spacing:var(--si-heading-tracking,-.02em);font-size:clamp(28px,6.4vw,92px);line-height:1.05}
.s{margin:0;color:var(--si-muted);font-size:clamp(16px,2.8vw,36px);line-height:1.35;max-width:32em}
.u{margin:.6rem 0 0;color:var(--si-accent);font-weight:700;font-size:`+u+`}
h1 code,.s code{font-family:var(--si-font-mono,ui-monospace,monospace);color:var(--si-code,var(--si-accent))}`)
}

// thirdHTML is a lower third: a name and a role at the bottom left, over
// the picture (transparent around it).
func thirdHTML(theme, name, sub string) string {
	body := `<div><b>` + html.EscapeString(name) + `</b>`
	if sub != "" {
		body += `<span>` + html.EscapeString(sub) + `</span>`
	}
	body += `</div>`
	return themedPage(theme, body, `div{position:fixed;left:4vw;bottom:9vh;display:grid;gap:.2em;padding:.65em 1.1em .7em;border-left:.32em solid var(--si-accent);
border-radius:.3em;background:color-mix(in srgb,var(--si-surface,#fff) 92%,transparent);color:var(--si-fg);box-shadow:0 .4em 1.4em rgb(0 0 0/.28);
font-family:var(--si-font-body,system-ui,sans-serif)}
b{font:700 clamp(15px,2.3vw,34px)/1.15 var(--si-font-heading,system-ui,sans-serif)}
span{color:var(--si-muted);font-size:clamp(12px,1.6vw,24px)}`)
}

// The vertical cut: 1080×1920, the film fitted across, a blurred copy of it
// behind, the captions large under the picture.
const vertW, vertH = 1080, 1920

// vertTop is where the film's picture starts in the vertical frame (it sits
// a little above the middle, the captions below).
func vertTop(w, h int) int {
	fh := even(float64(vertW) * float64(h) / float64(w))
	return even(float64(vertH)*0.44 - float64(fh)/2)
}

// verticalCaptionHTML is one caption of the vertical cut, large, under the
// picture of a w×h film.
func verticalCaptionHTML(text string, w, h int) string {
	top := vertTop(w, h) + even(float64(vertW)*float64(h)/float64(w)) + 70
	var lines []string
	for _, l := range wrapAt(text, 22) {
		lines = append(lines, html.EscapeString(l))
	}
	return pageHTML(`<div>`+strings.Join(lines, "<br>")+`</div>`, fmt.Sprintf(`div{position:fixed;left:60px;right:60px;top:%dpx;text-align:center;color:#fff;
font:800 70px/1.22 var(--si-font-body,system-ui,sans-serif);text-shadow:0 3px 14px rgb(0 0 0/.75),0 0 3px rgb(0 0 0/.9)}`, top))
}

// wrapAt breaks text into lines of about n letters.
func wrapAt(s string, n int) []string {
	var out []string
	cur := ""
	for _, w := range strings.Fields(s) {
		if cur != "" && len(cur)+1+len(w) > n {
			out, cur = append(out, cur), ""
		}
		if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// verticalArgs encode the vertical cut of a w×h film: the film across the
// middle over a blurred, darkened copy of itself, the captions over that.
func verticalArgs(video string, w, h int, caps []pic, out string) []string {
	a := ff("-i", video)
	var g strings.Builder
	fmt.Fprintf(&g, "[0:v]split=2[a][b];[a]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,gblur=sigma=40,eq=brightness=-0.16[bg];[b]scale=%d:-2:flags=lanczos[fg];[bg][fg]overlay=0:%d[v0];",
		vertW, vertH, vertW, vertH, vertW, vertTop(w, h))
	last := "v0"
	for i, c := range caps {
		a = append(a, "-i", c.PNG)
		fmt.Fprintf(&g, "[%s][%d:v]overlay=0:0:enable='between(t,%.3f,%.3f)'[v%d];", last, i+1, c.From, c.To, i+1)
		last = fmt.Sprintf("v%d", i+1)
	}
	fmt.Fprintf(&g, "[%s]setsar=1,format=yuv420p[v]", last)
	a = append(a, "-filter_complex", g.String(), "-map", "[v]", "-map", "0:a?", "-c:a", "copy")
	return append(a, append(h264, out)...)
}

// cutOf is one stretch of a clip in the highlight.
type cutOf struct {
	Clip     string
	From, To float64
}

// highlightCuts picks each clip's first sentence (or its first seconds when
// nothing is said), a breath around it, the whole kept near most seconds.
func highlightCuts(clips []*clip, dir string, most float64) []cutOf {
	var cs []cutOf
	share := most / float64(max(1, len(clips)))
	for _, c := range clips {
		from, to := 0.5, 5.5
		if c.Hi != nil {
			from, to = c.Hi.From-0.4, c.Hi.To+0.5
		}
		from = max(0, from)
		if to-from < 3 {
			to = from + 3
		}
		if to-from > share && share >= 3 {
			to = from + share
		}
		to = min(to, c.Seconds)
		if to-from < 0.5 {
			continue
		}
		cs = append(cs, cutOf{filepath.Join(dir, c.Name+".mp4"), from, to})
	}
	return cs
}

// highlightArgs join the cuts (a short fade in and out each) between the
// brand's cards (nil: none), fitted into the w×h frame.
func highlightArgs(cuts []cutOf, intro, outro *part, w, h int, out string) []string {
	w, h = w&^1, h&^1
	fit := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1,format=yuv420p,fps=%d", w, h, w, h, fps)
	audio := "aresample=48000,aformat=sample_rates=48000:channel_layouts=stereo"
	a := ff()
	var g, cat strings.Builder
	in, n := 0, 0
	card := func(p *part) {
		if p == nil {
			return
		}
		a = append(a, "-loop", "1", "-framerate", fmt.Sprint(fps), "-t", fmt.Sprint(p.Hold), "-i", p.Card,
			"-f", "lavfi", "-t", fmt.Sprint(p.Hold), "-i", "anullsrc=r=48000:cl=stereo")
		fmt.Fprintf(&g, "[%d:v]%s[s%d];[%d:a]%s[t%d];", in, fit, n, in+1, audio, n)
		fmt.Fprintf(&cat, "[s%d][t%d]", n, n)
		in, n = in+2, n+1
	}
	card(intro)
	for _, c := range cuts {
		d := c.To - c.From
		a = append(a, "-ss", fmt.Sprintf("%.3f", c.From), "-t", fmt.Sprintf("%.3f", d), "-i", c.Clip)
		f := min(0.3, d/4)
		fmt.Fprintf(&g, "[%d:v]%s,fade=t=in:st=0:d=%.2f,fade=t=out:st=%.3f:d=%.2f[s%d];[%d:a]%s,afade=t=in:st=0:d=%.2f,afade=t=out:st=%.3f:d=%.2f[t%d];",
			in, fit, f, d-f, f, n, in, audio, f, d-f, f, n)
		fmt.Fprintf(&cat, "[s%d][t%d]", n, n)
		in, n = in+1, n+1
	}
	card(outro)
	fmt.Fprintf(&g, "%sconcat=n=%d:v=1:a=1[v][a]", cat.String(), n)
	a = append(a, "-filter_complex", g.String(), "-map", "[v]", "-map", "[a]")
	a = append(a, h264[:6]...)
	return append(append(a, aac...), "-r", fmt.Sprint(fps), "-movflags", "+faststart", out)
}

// musicArgs lay music (looped, at vol) under a film of d seconds: it fades
// in and out, and ducks while the voice speaks (a sidechain compressor keyed
// by the voice). The picture and the chapters are copied.
func musicArgs(video, music string, vol, d float64, out string) []string {
	audio := "aresample=48000,aformat=sample_rates=48000:channel_layouts=stereo"
	fade := max(0.1, min(2, d/4))
	g := fmt.Sprintf("[1:a]%s,volume=%.3f,afade=t=in:st=0:d=1,afade=t=out:st=%.3f:d=%.3f[m];[0:a]%s,asplit=2[v][k];"+
		"[m][k]sidechaincompress=threshold=0.02:ratio=8:attack=20:release=450[d];[v][d]amix=inputs=2:duration=first:dropout_transition=0:normalize=0[a]",
		audio, vol, max(0, d-fade), fade, audio)
	a := ff("-i", video, "-stream_loop", "-1", "-i", music, "-filter_complex", g, "-map", "0:v", "-map", "[a]", "-c:v", "copy")
	return append(append(a, aac...), "-t", fmt.Sprintf("%.3f", d), "-movflags", "+faststart", out)
}

// brandParts renders the intro and outro cards (nil when not asked for)
// into the work folder, while a page is still open to draw them.
func (r *runner) brandParts() error {
	b := r.s.Brand
	if r.intro != nil || r.outro != nil || (b.Intro == nil && b.Outro == nil) {
		return nil
	}
	card := func(c *Card, name string, hold float64, outro bool) (*part, error) {
		if c == nil {
			return nil, nil
		}
		title := c.Title
		if title == "" && !outro {
			title = r.title()
		}
		png, err := r.picture(brandCardHTML(b.Theme, b.Logo, title, c.Subtitle, b.URL, outro), false)
		if err != nil {
			return nil, err
		}
		file := filepath.Join(r.o.Out, ".brand-"+name+".png")
		if err := os.WriteFile(file, png, 0o644); err != nil {
			return nil, err
		}
		return &part{Name: title, Card: file, Hold: hold}, nil
	}
	var err error
	if r.intro, err = card(b.Intro, "intro", introSeconds, false); err != nil {
		return err
	}
	r.outro, err = card(b.Outro, "outro", outroSeconds, true)
	return err
}

// branded puts the intro and outro around the parts.
func (r *runner) branded(parts []part) []part {
	var out []part
	if r.intro != nil {
		out = append(out, *r.intro)
	}
	out = append(out, parts...)
	if r.outro != nil {
		out = append(out, *r.outro)
	}
	return out
}

// pictureAt renders HTML at w×h CSS pixels (one device pixel each), then
// gives the page back its viewport.
func (r *runner) pictureAt(page string, w, h int) ([]byte, error) {
	if err := r.c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		return nil, err
	}
	defer r.c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": r.s.W, "height": r.s.H, "deviceScaleFactor": r.k, "mobile": false}, nil)
	return r.picture(page, true)
}

// shorts writes what With asks for from the joined film: the music under
// <script>.mp4 (and its captions variant), <script>.vertical.mp4 and
// <script>.highlight.mp4.
func (r *runner) shorts(clips []*clip, cs []chapter) error {
	w, b := r.o.With, r.s.Brand
	at := func(ext string) string { return filepath.Join(r.o.Out, r.s.Name+ext) }
	end := 0.0
	if n := len(cs); n > 0 {
		end = cs[n-1].End
	}
	withMusic := func(file string, d float64) error {
		if b.Music == "" {
			return nil
		}
		tmp := file + ".music.mp4"
		if err := r.ffmpeg(musicArgs(file, b.Music, b.Volume, d, tmp)); err != nil {
			return err
		}
		return os.Rename(tmp, file)
	}
	if w.Vertical {
		var caps []pic
		for i, c := range clips {
			for _, p := range c.VCaps {
				caps = append(caps, pic{p.PNG, cs[i].Clip + p.From, cs[i].Clip + p.To})
			}
		}
		if err := r.ffmpeg(verticalArgs(at(".mp4"), r.s.W, r.s.H, caps, at(".vertical.mp4"))); err != nil {
			return err
		}
		if err := withMusic(at(".vertical.mp4"), end); err != nil {
			return err
		}
	}
	if w.Highlight {
		cuts := highlightCuts(clips, r.o.Out, 60)
		if len(cuts) > 0 {
			if err := r.ffmpeg(highlightArgs(cuts, r.intro, r.outro, r.s.W, r.s.H, at(".highlight.mp4"))); err != nil {
				return err
			}
			d := 0.0
			for _, c := range cuts {
				d += c.To - c.From
			}
			if r.intro != nil {
				d += r.intro.Hold
			}
			if r.outro != nil {
				d += r.outro.Hold
			}
			if err := withMusic(at(".highlight.mp4"), d); err != nil {
				return err
			}
		}
	}
	for _, f := range []string{at(".mp4"), at(".captions.mp4")} {
		if _, err := os.Stat(f); err == nil {
			if err := withMusic(f, end); err != nil {
				return err
			}
		}
	}
	return nil
}
