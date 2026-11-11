package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// A deck that presents itself, built: si build --autoplay voices every part
// of every slide's Notes: (deck.Parts) at build time into
// <name>.voice/<slide>-<part>.m4a beside the page (slide from 1, part from
// 0), which the page plays by relative URL; --autoplay=inline puts them in
// the page as data URIs. si build --mp4 --voice narrates the video from the
// same parts.

// encodeM4A turns a voiced WAV into AAC (tests replace it).
var encodeM4A = webdemo.M4A

// inlineWarnBytes is the voice size past which --autoplay=inline warns.
const inlineWarnBytes = 20 << 20

// voiceForBuild voices d's parts and returns the src of each part's audio
// for the page (nil when the deck has no script).
func voiceForBuild(d *deck.Deck, cfg *config.Config, voice string, sy webdemo.Voicer, ffmpeg, out string, inline bool, stdout, stderr io.Writer) (func(i, j int) string, error) {
	parts := d.Parts()
	dv, warns, err := webdemo.ForDeck(d, cfg, voice)
	for _, w := range warns {
		fmt.Fprintf(stderr, "  warning: %s: %s\n", d.Name, w)
	}
	if err != nil {
		return nil, err
	}
	lines, err := dv.VoiceParts(sy, parts)
	if err != nil || len(lines) == 0 {
		return nil, err
	}
	rel := d.Name + ".voice"
	dir := filepath.Join(out, rel)
	if err := os.RemoveAll(dir); err != nil { // parts of an older script must not linger
		return nil, err
	}
	tmp := ""
	if inline {
		if tmp, err = os.MkdirTemp("", "si-autoplay-"); err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmp)
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	src := map[[2]int]string{}
	size := 0
	for i, ps := range parts {
		for j := range ps {
			l, ok := lines[[2]int{i, j}]
			if !ok {
				continue
			}
			name := fmt.Sprintf("%d-%d.m4a", i+1, j)
			dest := filepath.Join(dir, name)
			if inline {
				dest = filepath.Join(tmp, name)
			}
			if err := encodeM4A(ffmpeg, l.WAV, dest); err != nil {
				return nil, err
			}
			b, err := os.ReadFile(dest)
			if err != nil {
				return nil, err
			}
			size += len(b)
			if inline {
				src[[2]int{i, j}] = "data:audio/mp4;base64," + base64.StdEncoding.EncodeToString(b)
			} else {
				src[[2]int{i, j}] = rel + "/" + name
			}
		}
	}
	where := rel + "/"
	if inline {
		where = "(inline)"
		if size > inlineWarnBytes {
			fmt.Fprintf(stderr, "  warning: %s: --autoplay=inline adds %d MB of voice to the page; --autoplay keeps it in %s/ beside it\n", d.Name, size>>20, rel)
		}
	}
	fmt.Fprintf(stdout, "  %-28s (auto-play, %d parts, %s, %d KB)\n", where, len(src), dv.Default, size/1024)
	return func(i, j int) string { return src[[2]int{i, j}] }, nil
}

// voicedNarration is a narration made from the deck's own script: each view
// (slide, build step) on screen while its parts are said, then a beat; a
// view with nothing to say stays dwell seconds (or its (N min) share). It
// paces si build --mp4 --voice the way a recording paces --narration.
func voicedNarration(d *deck.Deck, steps []int, lines map[[2]int]webdemo.Line, dwell float64, voice string, wav string) (*narration, error) {
	const gap, tail = 0.35, 0.6
	n := &narration{dir: voice + " voice", audio: wav, spoken: map[[2]int]float64{}}
	var placed []webdemo.Placed
	t := 0.0
	for i, s := range d.Slides {
		last := 0
		if i < len(steps) {
			last = steps[i]
		}
		parts := s.Parts()
		view := func(k int) float64 { // the build step part j is said on
			c := 0
			for x := 1; x <= k && x < len(parts); x++ {
				if parts[x].Click {
					c++
				}
			}
			return float64(c)
		}
		for k := 0; k <= last; k++ {
			secs := 0.0
			for j := range parts {
				l, ok := lines[[2]int{i, j}]
				if !ok || int(view(j)) != k {
					continue
				}
				if secs > 0 {
					secs += gap
				}
				placed = append(placed, webdemo.Placed{WAV: l.WAV, At: t + secs})
				secs += l.Seconds
			}
			if secs > 0 {
				secs += tail
			} else if s.Minutes > 0 {
				secs = s.Minutes * 60 / float64(last+1)
			} else if dwell > 0 {
				secs = dwell
			} else {
				secs = 3
			}
			n.spoken[[2]int{i, k}] = secs
			t += secs
		}
	}
	n.seconds = t
	if len(placed) == 0 {
		return nil, fmt.Errorf("--voice: %s has no Notes: to say", d.Name)
	}
	if err := webdemo.MixWAV(wav, t, placed); err != nil {
		return nil, err
	}
	return n, nil
}

// narrateFromNotes voices the deck's script with voice (si build --mp4
// --voice) and paces the video of the page at dest by it.
func narrateFromNotes(d *deck.Deck, dest string, cfg *config.Config, voice string, dwell float64, stderr io.Writer) (*narration, error) {
	ff := tools.Find("ffmpeg")
	if ff == "" {
		return nil, fmt.Errorf("%s", tools.Need("ffmpeg", "--voice"))
	}
	dv, warns, err := webdemo.ForDeck(d, cfg, voice)
	for _, w := range warns {
		fmt.Fprintf(stderr, "  warning: %s: %s\n", d.Name, w)
	}
	if err != nil {
		return nil, err
	}
	lines, err := dv.VoiceParts(&webdemo.Synth{FFmpeg: ff, Cache: webdemo.VoiceCache(), Log: stderr}, d.Parts())
	if err != nil {
		return nil, err
	}
	page, err := os.ReadFile(dest)
	if err != nil {
		return nil, err
	}
	return voicedNarration(d, slideSteps(string(page)), lines, dwell, dv.Default.String(), filepath.Join(os.TempDir(), "si-voice-"+d.Name+".wav"))
}
