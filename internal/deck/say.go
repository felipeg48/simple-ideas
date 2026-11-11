package deck

import (
	"fmt"
	"regexp"
	"strings"
)

// Auto-play reads a slide's Notes: aloud and clicks where the script says
// [CLICK]: the script is split at each cue into parts, spoken in order, with
// one advance (a build step or the next slide) between two parts of the same
// speaker. The parts are plain text: the written words, for captions; the
// voice applies pronounce: on its own.

// Part is one spoken stretch of a slide's script.
type Part struct {
	Text    string `json:"t"`           // plain text, the written words ("" for two cues in a row)
	Speaker string `json:"s,omitempty"` // who says it, in a deck with speakers:
	Click   bool   `json:"k,omitempty"` // an advance comes before it (a [CLICK] ended the part before)
}

var (
	sayCueRE    = regexp.MustCompile(`\[(?i:click)\]`)
	sayImageRE  = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	sayLinkRE   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	sayTagRE    = regexp.MustCompile(`<[^>]*>`)
	sayListRE   = regexp.MustCompile(`(?m)^\s*(?:[-*+>]|\d+[.)])\s+`)
	sayUnderRE  = regexp.MustCompile(`(^|[^\pL\pN])_+([^_]+?)_+($|[^\pL\pN])`)
	sayMarkRE   = regexp.MustCompile("[*`]|==")
	saySpacesRE = regexp.MustCompile(`\s+`)
)

// Plain is markdown as the words a voice says: no links, emphasis, code
// marks, list markers or HTML; one line.
func Plain(md string) string {
	s := sayImageRE.ReplaceAllString(md, "")
	s = sayLinkRE.ReplaceAllString(s, "$1")
	s = sayTagRE.ReplaceAllString(s, "")
	s = sayListRE.ReplaceAllString(s, "")
	s = sayUnderRE.ReplaceAllString(s, "$1$2$3")
	s = sayMarkRE.ReplaceAllString(s, "")
	return strings.TrimSpace(saySpacesRE.ReplaceAllString(s, " "))
}

// Parts splits a slide's script at its [CLICK] cues: each speaker's Notes in
// order (a deck with speakers:), else the slide's Notes. A slide with no
// script has no parts.
func (s *Slide) Parts() []Part {
	type block struct{ who, md string }
	var blocks []block
	if len(s.Voices) > 0 {
		for _, v := range s.Voices {
			if v.Notes != "" {
				blocks = append(blocks, block{v.Speaker, v.Notes})
			}
		}
	} else if s.Notes != "" {
		blocks = append(blocks, block{"", s.Notes})
	}
	var out []Part
	for _, b := range blocks {
		for i, t := range sayCueRE.Split(b.md, -1) {
			out = append(out, Part{Text: Plain(t), Speaker: b.who, Click: i > 0})
		}
	}
	// a trailing cue's empty part stays: it is the click to the next slide
	return out
}

// Parts is every slide's parts, in slide order.
func (d *Deck) Parts() [][]Part {
	out := make([][]Part, len(d.Slides))
	for i := range d.Slides {
		out[i] = d.Slides[i].Parts()
	}
	return out
}

// VoiceMap reads `voices: Ada=af_heart, Ravi=kokoro:am_michael`: speaker
// name -> voice (a bare name is a voice of the deck's own engine).
func (m Meta) VoiceMap() map[string]string {
	out := map[string]string{}
	for _, p := range strings.Split(m.Voices, ",") {
		if n, v, ok := strings.Cut(p, "="); ok && strings.TrimSpace(n) != "" && strings.TrimSpace(v) != "" {
			out[strings.TrimSpace(n)] = strings.TrimSpace(v)
		}
	}
	return out
}

// checkVoices reads voices: as Name=voice pairs.
func checkVoices(val string) string {
	for _, p := range strings.Split(val, ",") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if n, v, ok := strings.Cut(p, "="); !ok || strings.TrimSpace(n) == "" || strings.TrimSpace(v) == "" {
			return fmt.Sprintf("voices %q: write Name=voice pairs, comma-separated (Ada=af_heart, Ravi=kokoro:am_michael)", strings.TrimSpace(p))
		}
	}
	return ""
}
