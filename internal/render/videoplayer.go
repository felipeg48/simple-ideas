package render

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/yuin/goldmark/ast"
)

// Video players: a fence whose language is `video` is drawn as a player,
// authored like a code window (options in the fence info string, any body is
// the caption):
//
//	```video src="demo.mp4" poster="demo.jpg" mode=modal title="Demo"
//	A short caption.
//	```
//
// mode= picks how it plays: inline (in place, a poster with a big play button
// first), modal (the poster opens the shared dialog, the page dimmed and
// blurred behind; every modal player on the page, or with the same group=,
// is one gallery) or facade (inline, but nothing loads until the first play).
// hud=on (si's own controls, the default), native (the browser's) or off.
// captions= and chapters= name the sidecar files; without them si looks for
// <video>.vtt (or .srt) and <video>.chapters.txt (or .vtt) beside the video,
// or a lone chapters.txt as si demo run writes it. Both are read at build and
// carried in the page (captions as a data: URI track, chapters as JSON), so a
// built page plays them from file:// too. web/videoplayer.{css,js} draw and
// drive every player; si --export player ships the same two files.

var (
	vpModes = []string{"inline", "modal", "facade"}
	vpHUDs  = []string{"on", "native", "off"}
	vpOpts  = []string{"src", "poster", "mode", "title", "width", "aspect", "ratio", "autoplay", "loop", "muted", "captions", "chapters", "group", "start", "hud", "lang"}
)

// docVideoTag starts the <video> of a ![](x.mp4) figure (media.go, videoImgRE).
const docVideoTag = `<video controls preload="metadata" playsinline src="`

// videoPlayers replaces every top-level ```video fence with its player HTML.
func (dr *deckRenderer) videoPlayers(doc ast.Node, src []byte, sc *slideCtx) {
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		if fc, ok := c.(*ast.FencedCodeBlock); ok && fenceLang(fc, src) == "video" {
			attrs := map[string]string{}
			if fc.Info != nil {
				attrs = fenceAttrs(string(fc.Info.Segment.Value(src)))
			}
			var body strings.Builder
			for i := 0; i < fc.Lines().Len(); i++ {
				seg := fc.Lines().At(i)
				body.Write(seg.Value(src))
			}
			caption := strings.TrimSpace(body.String())
			if caption != "" {
				caption = dr.inline(caption)
			}
			dr.videoplayer = true
			doc.ReplaceChild(doc, c, &rawNode{html: dr.vpFence(attrs, caption, sc), media: true})
		}
		c = next
	}
}

// vpFence writes one player: the figure and its <video> (with its captions
// track and chapters), or for modal the poster button and a hidden <video>
// the dialog reads its clip from. src and poster are left as written so
// postprocess links or inlines them, except the facade's src, which is
// resolved here (it rides in data-vp-src until the first play).
func (dr *deckRenderer) vpFence(attrs map[string]string, caption string, sc *slideCtx) string {
	for k := range attrs {
		if !slices.Contains(vpOpts, k) {
			dr.problem(sc, "video option %q is not one of %s", k, strings.Join(vpOpts, ", "))
		}
	}
	mode := firstNonEmpty(attrs["mode"], "inline")
	if !slices.Contains(vpModes, mode) {
		dr.problem(sc, "video mode %q is not one of %s", mode, strings.Join(vpModes, ", "))
		mode = "inline"
	}
	hud := firstNonEmpty(attrs["hud"], "on")
	if !slices.Contains(vpHUDs, hud) {
		dr.problem(sc, "video hud %q is not one of %s", hud, strings.Join(vpHUDs, ", "))
		hud = "on"
	}
	src, poster, title := attrs["src"], attrs["poster"], attrs["title"]
	_, autoplay := attrs["autoplay"]
	_, loop := attrs["loop"]
	_, muted := attrs["muted"]
	start := 0.0
	if s := attrs["start"]; s != "" {
		var ok bool
		if start, ok = vpClock(s); !ok {
			dr.problem(sc, "video start %q is not a time (12, 1:05 or 1:02:03)", s)
		}
	}

	dir := filepath.Dir(dr.d.Path)
	if sc != nil && sc.s != nil {
		dir = sc.s.Dir
	}
	local := src != "" && !isExternal(src)
	abs := ""
	if local {
		abs = filepath.Join(dir, filepath.FromSlash(cleanSrc(src)))
	}
	if poster == "" && local { // the first frame, <video>.poster.jpg
		if still := autoPoster(abs); still != "" {
			if r, err := filepath.Rel(dir, still); err == nil {
				poster = filepath.ToSlash(r)
			}
		}
	}
	// captions and chapters: the files named, else the ones beside the video
	at := func(f string) string {
		if f == "" || isExternal(f) {
			return f
		}
		return filepath.Join(dir, filepath.FromSlash(f))
	}
	for _, k := range []string{"captions", "chapters"} {
		if f := attrs[k]; f != "" && !isExternal(f) && !fileExists(at(f)) {
			dr.problem(sc, "missing file: %s", f)
		}
	}
	capFile, chapFile := at(attrs["captions"]), at(attrs["chapters"])
	if abs != "" {
		capFile, chapFile = vpSidecars(abs, capFile, chapFile)
	}
	lang := firstNonEmpty(attrs["lang"], dr.d.Meta.Lang)
	tracks := vpTrack(capFile, lang, false)
	chapters := vpChapters(chapFile)
	if attrs["chapters"] != "" && chapters == "" && fileExists(chapFile) {
		dr.problem(sc, "chapters %s: no chapter lines (0:00 Title, or WebVTT cues)", attrs["chapters"])
	}

	var vars []string
	if w := attrs["width"]; w != "" {
		vars = append(vars, "--si-vp-width:"+w)
	}
	if a := firstNonEmpty(attrs["aspect"], attrs["ratio"]); a != "" {
		vars = append(vars, "--si-vp-ratio:"+a)
	}

	var b strings.Builder
	b.WriteString(`<figure class="si-vp" data-vp-mode="` + mode + `"`)
	if hud != "on" {
		b.WriteString(` data-vp-hud="` + hud + `"`)
	}
	if g := attrs["group"]; g != "" && mode == "modal" {
		b.WriteString(` data-vp-group="` + html.EscapeString(g) + `"`)
	}
	if src == "" {
		b.WriteString(` data-vp-disabled`)
	}
	if len(vars) > 0 {
		fmt.Fprintf(&b, ` style="%s"`, html.EscapeString(strings.Join(vars, ";")))
	}
	b.WriteString(`>`)

	label := firstNonEmpty(title, "Video")
	posterImg := ""
	if poster != "" {
		posterImg = fmt.Sprintf(`<img class="si-vp-poster" src="%s" alt="%s">`, html.EscapeString(poster), html.EscapeString(title))
	}
	const playSVG = `<span class="si-vp-play" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></span>`

	// the <video>: every mode has one (the dialog copies the modal's)
	video := func(preload string, hidden bool) string {
		var v strings.Builder
		v.WriteString(`<video class="si-vp-video"`)
		if hidden {
			v.WriteString(` hidden`)
		}
		if hud != "off" && !hidden {
			v.WriteString(` controls`) // the HUD turns it off; without JS the browser's controls stay
		}
		v.WriteString(` preload="` + preload + `" playsinline`)
		if autoplay && !hidden {
			v.WriteString(` autoplay muted`)
		} else if muted {
			v.WriteString(` muted`)
		}
		if loop {
			v.WriteString(` loop`)
		}
		if hidden || poster == "" {
			v.WriteString(` poster=""`) // postprocess adds no still of its own
		} else {
			v.WriteString(` poster="` + html.EscapeString(poster) + `"`)
		}
		if mode == "facade" {
			resolved := src
			if local {
				if uri, ok := dr.fileLink(dir, src, sc); ok {
					resolved = uri
				} else {
					dr.problem(sc, "missing file: %s", src)
				}
			}
			v.WriteString(` data-vp-src="` + html.EscapeString(resolved) + `"`)
		} else {
			v.WriteString(` src="` + html.EscapeString(src) + `"`)
		}
		if title != "" {
			v.WriteString(` data-vp-title="` + html.EscapeString(title) + `"`)
		}
		if start > 0 {
			v.WriteString(` data-vp-start="` + strconv.FormatFloat(start, 'f', -1, 64) + `"`)
		}
		v.WriteString(chapters)
		v.WriteString(` aria-label="` + html.EscapeString(label) + `">` + tracks + `</video>`)
		return v.String()
	}

	switch {
	case src == "": // only a poster: show it, the play control disabled
		if poster == "" {
			dr.problem(sc, "a video block needs a src (or at least a poster)")
		}
		b.WriteString(`<div class="si-vp-frame" aria-label="` + html.EscapeString(label) + ` (unavailable)">` + posterImg + playSVG + `</div>`)
	case mode == "modal":
		fmt.Fprintf(&b, `<button type="button" class="si-vp-open" aria-haspopup="dialog" aria-label="%s">%s%s</button>`,
			html.EscapeString("Play video: "+label), posterImg, playSVG)
		b.WriteString(video("none", true))
	case mode == "facade":
		b.WriteString(video("none", false))
	default:
		b.WriteString(video("metadata", false))
	}

	if caption != "" {
		fmt.Fprintf(&b, `<figcaption class="si-vp-cap">%s</figcaption>`, caption)
	}
	b.WriteString("</figure>\n")
	return b.String()
}

// vpSidecars finds a local video's captions and chapters: the files given,
// else <stem>.vtt (or .srt) and <stem>.chapters.txt (or .chapters.vtt), else a
// chapters.txt beside a video that is alone in its folder (si demo run).
func vpSidecars(video, captions, chapters string) (string, string) {
	stem := strings.TrimSuffix(video, filepath.Ext(video))
	if captions == "" {
		for _, ext := range []string{".vtt", ".srt"} {
			if fileExists(stem + ext) {
				captions = stem + ext
				break
			}
		}
	}
	if chapters == "" {
		for _, f := range []string{stem + ".chapters.txt", stem + ".chapters.vtt"} {
			if fileExists(f) {
				return captions, f
			}
		}
		if f := filepath.Join(filepath.Dir(video), "chapters.txt"); fileExists(f) {
			n := 0
			for _, ext := range []string{"*.mp4", "*.webm", "*.mov", "*.m4v"} {
				m, _ := filepath.Glob(filepath.Join(filepath.Dir(video), ext))
				n += len(m)
			}
			if n == 1 {
				chapters = f
			}
		}
	}
	return captions, chapters
}

var srtTimeRE = regexp.MustCompile(`(\d\d:\d\d:\d\d),(\d\d\d)`)

// vpTrack is a captions <track> carrying the file as a data: URI (a .srt is
// turned into WebVTT), "" when there is no file.
func vpTrack(file, lang string, def bool) string {
	if file == "" {
		return ""
	}
	src := file
	if !isExternal(file) {
		raw, err := os.ReadFile(file)
		if err != nil {
			return ""
		}
		text := strings.ReplaceAll(string(raw), "\r", "")
		if !strings.HasPrefix(strings.TrimPrefix(text, "\ufeff"), "WEBVTT") {
			text = "WEBVTT\n\n" + srtTimeRE.ReplaceAllString(text, "$1.$2")
		}
		src = "data:text/vtt;base64," + base64.StdEncoding.EncodeToString([]byte(text))
	}
	d := ""
	if def {
		d = " default"
	}
	return fmt.Sprintf(`<track kind="captions" src="%s" srclang="%s" label="Captions"%s>`, html.EscapeString(src), html.EscapeString(firstNonEmpty(lang, "en")), d)
}

// vpChapters is the data-vp-chapters attribute of a chapters file, "" when
// there is none or it has no chapters.
func vpChapters(file string) string {
	if file == "" || isExternal(file) {
		if file != "" {
			return ` data-vp-chapters-url="` + html.EscapeString(file) + `"`
		}
		return ""
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	cs := parseChapters(string(raw))
	if len(cs) == 0 {
		return ""
	}
	j, _ := json.Marshal(cs)
	return ` data-vp-chapters="` + html.EscapeString(string(j)) + `"`
}

var vttCueRE = regexp.MustCompile(`^\s*([\d:.,]+)\s+-->\s+`)

// parseChapters reads chapter lines ("0:00 Intro", as YouTube and si demo run
// write them) or a WebVTT file whose cues are the chapters: [[seconds, title]].
func parseChapters(text string) [][2]any {
	var out [][2]any
	lines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimPrefix(lines[0], "\ufeff"), "WEBVTT") {
		for i, l := range lines {
			if m := vttCueRE.FindStringSubmatch(l); m != nil && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
				if t, ok := vpClock(m[1]); ok {
					out = append(out, [2]any{t, strings.TrimSpace(lines[i+1])})
				}
			}
		}
		return out
	}
	for _, l := range lines {
		t, name, ok := strings.Cut(strings.TrimSpace(l), " ")
		if !ok {
			continue
		}
		if v, ok := vpClock(strings.Trim(t, "()")); ok {
			if name = strings.TrimSpace(strings.TrimLeft(name, "-–— ")); name != "" {
				out = append(out, [2]any{v, name})
			}
		}
	}
	return out
}

// vpClock reads 12, 12s, 1:05, 1:02:03 or 00:01:02.500 as seconds.
func vpClock(s string) (float64, bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "s")
	if s == "" {
		return 0, false
	}
	t := 0.0
	for _, p := range strings.Split(s, ":") {
		n, err := strconv.ParseFloat(strings.Replace(p, ",", ".", 1), 64)
		if err != nil || n < 0 {
			return 0, false
		}
		t = t*60 + n
	}
	return t, true
}

func (dr *deckRenderer) videoplayerCSS(b *bytes.Buffer) {
	if dr.videoplayer {
		b.WriteString(`<style id="si-vp-css">` + "\n" + assets.VideoplayerCSS + "</style>\n")
	}
}

func (dr *deckRenderer) videoplayerJS(b *bytes.Buffer) {
	if dr.videoplayer {
		b.WriteString("<script>\n" + assets.VideoplayerJS + "</script>\n")
	}
}
