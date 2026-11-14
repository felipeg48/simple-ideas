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
	Acts    []Act  `json:"a,omitempty"` // presenter tools to drive after the advance, before the words
}

// Act is a presenter-tool cue in a Notes: script (si play's drawer and
// browser windows): [TERM Name: command] opens (or raises) the drawer's
// shell tab Name and runs command, [TERM Name] only opens it, [TERM close],
// [TERM min] and [TERM max] set the drawer; [BROWSER Name] opens the
// browser: link Name (or a URL) in a window over the slides, [BROWSER close]
// closes every window. Never read aloud.
type Act struct {
	Tool  string `json:"tool"`            // term or browser
	Name  string `json:"name,omitempty"`  // the tab, the browser: link name or a URL
	Cmd   string `json:"cmd,omitempty"`   // TERM: the command to run
	State string `json:"state,omitempty"` // close, min, max
}

// ActCueRE finds the presenter-tool cues in a Notes: script.
// ponytail: a cue ends at the first ], so a command holding ] cannot be cued.
var ActCueRE = regexp.MustCompile(`\[(TERM|BROWSER)(\b[^\]]*)\]`)

// ParseAct reads one cue's tool (TERM, BROWSER) and what follows it.
func ParseAct(tool, body string) (Act, error) {
	body = strings.TrimSpace(body)
	a := Act{Tool: strings.ToLower(tool)}
	if a.Tool == "browser" {
		switch {
		case body == "":
			return a, fmt.Errorf("[BROWSER] needs a browser: link name or a URL ([BROWSER RabbitMQ])")
		case body == "close":
			a.State = body
		default:
			a.Name = body
		}
		return a, nil
	}
	name, cmd, has := strings.Cut(body, ":")
	if strings.HasPrefix(cmd, "//") { // a colon of a URL is no command
		name, cmd, has = body, "", false
	}
	a.Name, a.Cmd = strings.TrimSpace(name), strings.TrimSpace(cmd)
	switch {
	case a.Name == "":
		return a, fmt.Errorf("[TERM%s] needs a tab name: [TERM Lab 1: command] or [TERM Lab 1]", map[bool]string{true: " " + body}[body != ""])
	case has && a.Cmd == "":
		return a, fmt.Errorf("[TERM %s] has an empty command after the colon", body)
	case !has && (a.Name == "close" || a.Name == "min" || a.Name == "max"):
		a.State, a.Name = a.Name, ""
	}
	return a, nil
}

// Acts are the cues of a Notes: script, with an error for each malformed one.
func Acts(notes string) ([]Act, []error) {
	var out []Act
	var errs []error
	for _, m := range ActCueRE.FindAllStringSubmatch(notes, -1) {
		a, err := ParseAct(m[1], m[2])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, a)
	}
	return out, errs
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
	s := ActCueRE.ReplaceAllString(md, "")
	s = sayImageRE.ReplaceAllString(s, "")
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
			// a [TERM …] or [BROWSER …] cue starts a new part (no advance): its
			// acts run there, then its words are said
			p := Part{Speaker: b.who, Click: i > 0}
			at := 0
			for _, m := range ActCueRE.FindAllStringSubmatchIndex(t, -1) {
				if w := Plain(t[at:m[0]]); w != "" {
					p.Text = w
					out = append(out, p)
					p = Part{Speaker: b.who}
				}
				if a, err := ParseAct(t[m[2]:m[3]], t[m[4]:m[5]]); err == nil {
					p.Acts = append(p.Acts, a)
				}
				at = m[1]
			}
			p.Text = Plain(t[at:])
			out = append(out, p)
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
