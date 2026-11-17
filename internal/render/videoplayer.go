package render

import (
	"bytes"
	"fmt"
	"html"
	"path/filepath"
	"slices"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/yuin/goldmark/ast"
)

// Video players: a fence whose language is `video` is drawn as a self-contained
// player, authored like a code window (options in the fence info string, any
// body is the caption):
//
//	```video src="demo.mp4" poster="demo.jpg" mode=modal title="Demo"
//	A short caption.
//	```
//
// mode= picks how it plays: inline (a normal <video controls> with a poster),
// modal (the poster is a button that opens an overlay <dialog> lightbox) or
// facade (poster only; the <video> is built on the first click, nothing loads
// until then). web/videoplayer.{css,js} style and drive them. A block with
// only a poster (no src) degrades to the poster with the play control disabled.

var vpModes = []string{"inline", "modal", "facade"}

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

// vpFence writes one player: the figure, its poster, the video or the dialog,
// and the caption. src and poster are left as written so postprocess inlines
// or links them, except the facade's src, which is resolved here (there is no
// eager <video> to carry it).
func (dr *deckRenderer) vpFence(attrs map[string]string, caption string, sc *slideCtx) string {
	mode := firstNonEmpty(attrs["mode"], "inline")
	if !slices.Contains(vpModes, mode) {
		dr.problem(sc, "video mode %q is not one of %s", mode, strings.Join(vpModes, ", "))
		mode = "inline"
	}
	src, poster, title := attrs["src"], attrs["poster"], attrs["title"]
	_, autoplay := attrs["autoplay"]
	_, loop := attrs["loop"]

	var vars []string
	if w := attrs["width"]; w != "" {
		vars = append(vars, "--si-vp-width:"+w)
	}
	if a := firstNonEmpty(attrs["aspect"], attrs["ratio"]); a != "" {
		vars = append(vars, "--si-vp-ratio:"+a)
	}

	var b strings.Builder
	b.WriteString(`<figure class="si-vp" data-vp-mode="` + html.EscapeString(mode) + `"`)
	if loop && mode == "facade" {
		b.WriteString(` data-vp-loop`)
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

	switch {
	case src == "": // only a poster: show it, the play control disabled
		if poster == "" {
			dr.problem(sc, "a video block needs a src (or at least a poster)")
		}
		b.WriteString(`<div class="si-vp-frame" aria-label="` + html.EscapeString(label) + ` (unavailable)">` + posterImg + playSVG + `</div>`)
	case mode == "inline":
		v := ` controls preload="metadata" playsinline`
		if autoplay {
			v += ` autoplay muted`
		}
		if loop {
			v += ` loop`
		}
		p := ""
		if poster != "" {
			p = ` poster="` + html.EscapeString(poster) + `"`
		}
		fmt.Fprintf(&b, `<video class="si-vp-video"%s%s src="%s" aria-label="%s"></video>`,
			v, p, html.EscapeString(src), html.EscapeString(label))
	case mode == "modal":
		p := ""
		if poster != "" {
			p = ` poster="` + html.EscapeString(poster) + `"`
		}
		loopAttr := ""
		if loop {
			loopAttr = ` loop`
		}
		fmt.Fprintf(&b, `<button type="button" class="si-vp-open" aria-haspopup="dialog" aria-label="%s">%s%s</button>`,
			html.EscapeString("Play video: "+label), posterImg, playSVG)
		fmt.Fprintf(&b, `<dialog class="si-vp-modal" aria-label="%s"><div class="si-vp-modal-inner"><button type="button" class="si-vp-close" aria-label="Close">&times;</button><video class="si-vp-video" controls preload="none" playsinline%s%s src="%s"></video></div></dialog>`,
			html.EscapeString(label), loopAttr, p, html.EscapeString(src))
	case mode == "facade":
		resolved := src
		if !isExternal(src) {
			dir := filepath.Dir(dr.d.Path)
			if sc != nil && sc.s != nil {
				dir = sc.s.Dir
			}
			if uri, ok := dr.fileLink(dir, src, sc); ok {
				resolved = uri
			} else {
				dr.problem(sc, "missing file: %s", src)
			}
		}
		fmt.Fprintf(&b, `<button type="button" class="si-vp-open" aria-label="%s" data-si-vp-src="%s">%s%s</button>`,
			html.EscapeString("Play video: "+label), html.EscapeString(resolved), posterImg, playSVG)
	}

	if caption != "" {
		fmt.Fprintf(&b, `<figcaption class="si-vp-cap">%s</figcaption>`, caption)
	}
	b.WriteString("</figure>\n")
	return b.String()
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
