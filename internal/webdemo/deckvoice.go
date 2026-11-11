package webdemo

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// A deck that presents itself (auto-play, si build --autoplay, si build --mp4
// --voice): each part of a slide's Notes: is voiced like a demo's Say: line,
// through the same cache.

// Voicer voices texts into WAVs; *Synth is one (tests inject fakes).
type Voicer interface {
	Voice(v Voice, texts []string) (map[string]Line, int, error)
}

// DeckVoice is who reads a deck: one voice, or one per speaker.
type DeckVoice struct {
	Default   Voice
	Speakers  map[string]Voice // speaker name -> voice (voices:, si.conf voice.<Name>:)
	Pronounce []Pronounce
}

// ForDeck picks the deck's voices: override (--voice), else the front matter
// voice:, else si.conf voice:, else DefaultVoice; per speaker, voices: in the
// front matter, else si.conf voice.<Name>:. A bare name (af_heart) is a voice
// of the default engine. A voice that cannot speak on this computer gives way
// to the system voice, with a warning.
func ForDeck(d *deck.Deck, cfg *config.Config, override string) (DeckVoice, []string, error) {
	var warn []string
	if cfg == nil {
		cfg = &config.Config{}
	}
	style := firstSet(d.Meta.VoiceStyle, cfg.VoiceStyle)
	ready := func(v Voice) Voice {
		v.Style = style
		if why := v.Ready(); why != "" && v.Engine != "system" && (Voice{Engine: "system"}).Ready() == "" {
			first, _, _ := strings.Cut(why, "\n")
			warn = append(warn, fmt.Sprintf("%s; speaking with the system voice", strings.TrimSuffix(first, ".")))
			return Voice{Engine: "system"}
		}
		return v
	}
	dv := DeckVoice{Default: DefaultVoice(), Speakers: map[string]Voice{}, Pronounce: DeckPronounce(d.Path)}
	if s := firstSet(override, d.Meta.Voice, cfg.Voice); s != "" {
		v, err := ParseVoice(s)
		if err != nil {
			return dv, warn, err
		}
		dv.Default = v
	}
	parse := func(s string) (Voice, error) {
		if !strings.Contains(s, ":") && !isEngine(s) {
			if names, _ := KokoroVoices(""); dv.Default.Engine != "kokoro" && slices.Contains(names, s) {
				return Voice{Engine: "kokoro", Name: s}, nil // a Kokoro name says which engine it means
			}
			return Voice{Engine: dv.Default.Engine, Name: s}, nil
		}
		return ParseVoice(s)
	}
	named := map[string]string{}
	for n, s := range cfg.SpeakerVoices {
		named[strings.ToLower(n)] = s
	}
	for n, s := range d.Meta.VoiceMap() {
		named[strings.ToLower(n)] = s
	}
	for _, n := range d.Meta.SpeakerList() {
		if s := named[strings.ToLower(n)]; s != "" {
			v, err := parse(s)
			if err != nil {
				return dv, warn, fmt.Errorf("voices: %s: %v", n, err)
			}
			dv.Speakers[n] = ready(v)
		}
	}
	dv.Default = ready(dv.Default)
	return dv, warn, nil
}

func isEngine(s string) bool {
	for _, e := range Engines {
		if e == strings.ToLower(strings.TrimSpace(s)) {
			return true
		}
	}
	return false
}

func firstSet(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// For is the voice of a speaker ("" or unnamed: the deck's voice).
func (dv DeckVoice) For(speaker string) Voice {
	if v, ok := dv.Speakers[speaker]; ok {
		return v
	}
	return dv.Default
}

// VoiceParts voices every part with text, one batch per voice, and returns
// each one's line by [slide, part].
func (dv DeckVoice) VoiceParts(sy Voicer, parts [][]deck.Part) (map[[2]int]Line, error) {
	texts := map[Voice][]string{}
	for _, ps := range parts {
		for _, p := range ps {
			if p.Text != "" {
				v := dv.For(p.Speaker)
				texts[v] = append(texts[v], Spoken(p.Text, dv.Pronounce))
			}
		}
	}
	lines := map[Voice]map[string]Line{}
	for v, ts := range texts {
		l, _, err := sy.Voice(v, ts)
		if err != nil {
			return nil, err
		}
		lines[v] = l
	}
	out := map[[2]int]Line{}
	for i, ps := range parts {
		for j, p := range ps {
			if p.Text != "" {
				out[[2]int{i, j}] = lines[dv.For(p.Speaker)][Spoken(p.Text, dv.Pronounce)]
			}
		}
	}
	return out, nil
}

// DeckPronounce is every pronounce: line of a file's front matter.
func DeckPronounce(path string) []Pronounce {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s := newScript(path)
	lines := splitLines(string(src))
	_, start := frontMatter(lines)
	for i := 1; i < start; i++ {
		if m := metaRE.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil && m[1] == "pronounce" {
			s.pronounce(m[2], i+1)
		}
	}
	return s.Pronounce
}

// Placed is a WAV at a time in seconds, for MixWAV.
type Placed = placed

// MixWAV writes total seconds of 24 kHz mono audio with each line at its time.
func MixWAV(out string, total float64, lines []Placed) error { return mixWAV(out, total, lines) }

// M4A encodes a WAV as AAC in an .m4a file.
func M4A(ffmpeg, wav, out string) error {
	if b, err := exec.Command(ffmpeg, m4aArgs(wav, out)...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg (m4a): %v %s", err, strings.TrimSpace(string(b)))
	}
	return nil
}
