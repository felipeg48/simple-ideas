package webdemo

import (
	"fmt"
	"strings"
)

// The promo kit's pieces the video editor (internal/videdit, `type: video`)
// builds on: themed pages, cards, captions, the captions and vertical
// variants, music, voices and the ffmpeg settings every si video shares.

// FFHead is what every ffmpeg call starts with; H264 and AAC are si's video
// and sound encoders; FPS its frame rate.
var (
	FFHead = ffHead
	H264   = h264
	AAC    = aac
)

const FPS = fps

// TokensOf is a theme's --si-* tokens (a built-in or a .css theme file) as
// declarations for a :root rule, light mode.
func TokensOf(theme string) string { return tokensOf(theme) }

// ThemedPage is a transparent page in a theme with style and body.
func ThemedPage(theme, body, style string) string { return themedPage(theme, body, style) }

// CardHTML is a title card in the promo kit's look: a title and a line
// under it.
func CardHTML(theme, title, sub string) string {
	return brandCardHTML(theme, "", title, sub, "", false)
}

// XfadeName is ffmpeg's xfade name for a transition kind ("fade" when
// unknown; dissolve is its own).
func XfadeName(kind string) string {
	if kind == "dissolve" {
		return "dissolve"
	}
	if n := xfadeNames[kind]; n != "" {
		return n
	}
	return "fade"
}

// SRT is SubRip captions for cues, split like VTT's.
func SRT(cues []Cue) string {
	var cs []caption
	for _, c := range cues {
		cs = append(cs, captions(c.Text, c.At, c.Seconds)...)
	}
	return srt(cs)
}

// Captions splits cues into the captions shown on screen (two lines at most
// each).
func Captions(cues []Cue) []Cue {
	var out []Cue
	for _, c := range cues {
		for _, x := range captions(c.Text, c.At, c.Seconds) {
			out = append(out, Cue{x.Text, x.From, x.To - x.From})
		}
	}
	return out
}

// CaptionHTML is one caption of the captions variant (the band under the
// picture); VerticalCaptionHTML one of the vertical cut of a w×h film, drawn
// at VertW×VertH.
func CaptionHTML(text string) string                   { return captionHTML(text) }
func VerticalCaptionHTML(text string, w, h int) string { return verticalCaptionHTML(text, w, h) }

const VertW, VertH = vertW, vertH

// Pic is a frame-size picture over a video from From to To seconds.
type Pic struct {
	PNG      string
	From, To float64
}

func pics(ps []Pic) []pic {
	out := make([]pic, len(ps))
	for i, p := range ps {
		out[i] = pic(p)
	}
	return out
}

// CaptionedArgs make the captions variant of a w×h video: the picture shrunk
// into the top of the frame, caps (drawn at w×h by CaptionHTML) in the band
// under it, the sound copied. Captions that follow each other are one input
// (an ffconcat list written beside them).
func CaptionedArgs(video string, w, h int, caps []Pic, out string) []string {
	a := ff("-i", video)
	var g strings.Builder
	fmt.Fprintf(&g, "[0:v]%s[b0];", band(w, h, w, h))
	last := "b0"
	if in, ok := slideshow(pics(caps), w, h); ok {
		a = append(a, in...)
		fmt.Fprintf(&g, "[b0][1:v]overlay=0:0:eof_action=pass[b1];")
		last, caps = "b1", nil
	}
	for i, c := range caps {
		a = append(a, "-i", c.PNG)
		fmt.Fprintf(&g, "[%s][%d:v]overlay=0:0:enable='between(t,%.3f,%.3f)'[b%d];", last, i+1, c.From, c.To, i+1)
		last = fmt.Sprintf("b%d", i+1)
	}
	fmt.Fprintf(&g, "[%s]setsar=1,format=yuv420p[v]", last)
	a = append(a, "-filter_complex", g.String(), "-map", "[v]", "-map", "0:a?", "-c:a", "copy")
	return append(a, append(h264, out)...)
}

// VerticalArgs make the vertical cut (VertW×VertH) of a w×h video: fill ""
// is a blurred copy behind, "theme" the theme's background, else 0xRRGGBB.
func VerticalArgs(video string, w, h int, fill, theme string, caps []Pic, out string) []string {
	if fill == "theme" {
		fill = themeFill(theme)
	}
	return verticalArgs(video, w, h, fill, pics(caps), out)
}

// MusicArgs lay music at vol (0…1) under a d-second video, ducked while its
// own sound speaks; the picture and the chapters are copied.
func MusicArgs(video, music string, vol, d float64, out string) []string {
	return musicArgs(video, music, vol, d, out)
}

// Cut is one stretch of a video in a highlight reel.
type Cut struct {
	Clip     string
	From, To float64
}

// HighlightArgs join the cuts (a short fade in and out each) into a w×h
// highlight reel (the demo kit's, without the brand's cards).
func HighlightArgs(cuts []Cut, w, h int, out string) []string {
	cs := make([]cutOf, len(cuts))
	for i, c := range cuts {
		cs[i] = cutOf(c)
	}
	return highlightArgs(cs, nil, nil, w, h, out)
}

// FirstSentence is the share of a line its first sentence takes, by letters.
func FirstSentence(s string) float64 { return firstSentence(s) }
