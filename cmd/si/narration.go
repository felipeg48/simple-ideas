package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// A recording made with the teleprompter's Record button (si play):
// .si/recordings/<talk>-<date-time>/ with audio.*, timeline.json and
// meta.json. It paces `si build --mp4 --narration` (each view held as long as
// it was spoken) and feeds the pace report of `si rehearse`.
type narration struct {
	dir, audio string
	seconds    float64
	spoken     map[[2]int]float64 // slide, step (0-based) -> seconds on screen
	marks      []recMark
	sha        string
}

type recMark struct {
	T     float64 `json:"t"`
	Slide int     `json:"slide"`
	Step  int     `json:"step"`
}

// findRecording is the recording of the deck at src: arg "latest" is the
// newest one for that deck in .si/recordings beside it; otherwise arg is a
// recording folder, used only when it was made from that deck ("" when none).
func findRecording(arg, src string) string {
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	if arg != "latest" {
		var meta struct{ Deck string }
		raw, err := os.ReadFile(filepath.Join(arg, "meta.json"))
		if err != nil || json.Unmarshal(raw, &meta) != nil || filepath.Base(meta.Deck) != filepath.Base(src) {
			return ""
		}
		return arg
	}
	dirs, _ := filepath.Glob(filepath.Join(filepath.Dir(src), ".si", "recordings", name+"-*"))
	var ok []string
	for _, d := range dirs { // <talk>-<date-time>: the date part sorts by time
		_, err := os.Stat(filepath.Join(d, "meta.json"))
		if rest := strings.TrimPrefix(filepath.Base(d), name+"-"); err == nil && len(rest) == len("2006-01-02-150405") {
			ok = append(ok, d)
		}
	}
	if len(ok) == 0 {
		return ""
	}
	sort.Strings(ok)
	return ok[len(ok)-1]
}

// loadNarration reads a recording and adds up how long each view was on
// screen (a view visited twice counts both times).
func loadNarration(dir string) (*narration, error) {
	var meta struct {
		Audio   string  `json:"audio"`
		Seconds float64 `json:"seconds"`
		SHA     string  `json:"sha256"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("%s: no meta.json (was the recording stopped?)", dir)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("%s/meta.json: %v", dir, err)
	}
	n := &narration{dir: dir, audio: filepath.Join(dir, filepath.Base(meta.Audio)), seconds: meta.Seconds, sha: meta.SHA, spoken: map[[2]int]float64{}}
	raw, err = os.ReadFile(filepath.Join(dir, "timeline.json"))
	if err == nil {
		err = json.Unmarshal(raw, &n.marks)
	}
	if err != nil || len(n.marks) == 0 {
		return nil, fmt.Errorf("%s: no timeline", dir)
	}
	for i, m := range n.marks {
		end := n.seconds
		if i+1 < len(n.marks) {
			end = n.marks[i+1].T
		}
		if end > m.T {
			n.spoken[[2]int{m.Slide, m.Step}] += end - m.T
		}
	}
	return n, nil
}

// changed reports whether the deck's source differs from the one recorded.
func (n *narration) changed(src string) bool {
	raw, err := os.ReadFile(src)
	sum := sha256.Sum256(raw)
	return err == nil && n.sha != "" && hex.EncodeToString(sum[:]) != n.sha
}

// hold is how long the video shows a view to match the recording: the time
// it was spoken, less lead (the transition into it, filmed before the hold),
// except for the first view. ok is false for a view the recording never
// showed.
func (n *narration) hold(slide, step int, lead float64, first bool) (float64, bool) {
	if n == nil {
		return 0, false
	}
	s, ok := n.spoken[[2]int{slide, step}]
	if !ok {
		return 0, false
	}
	if !first {
		s -= lead
	}
	return max(0.2, s), true
}

// muxAudio puts the recording's audio under the video (AAC), in place.
func muxAudio(video, audio string, stderr io.Writer) error {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("%s", ffmpegHint)
	}
	tmp := strings.TrimSuffix(video, ".mp4") + ".mux.mp4"
	out, err := exec.Command(ff, "-y", "-hide_banner", "-loglevel", "error", "-i", video, "-i", audio,
		"-map", "0:v", "-map", "1:a", "-c:v", "copy", "-c:a", "aac", "-b:a", "160k", "-movflags", "+faststart", tmp).CombinedOutput()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("ffmpeg (audio): %v %s", err, strings.TrimSpace(string(out)))
	}
	return os.Rename(tmp, video)
}

// rehearse is `si rehearse talk.md [--narration dir|latest]`: per slide, the
// time planned in Notes: (N min), the time spoken in a recording, and the
// words per minute the Notes: script implies at that pace.
func rehearse(src string, o *options, stdout, stderr io.Writer) int {
	d, err := deck.Parse(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	arg := o.narration
	if arg == "" {
		arg = "latest"
	}
	dir := findRecording(arg, src)
	if dir == "" {
		fmt.Fprintf(stderr, "si: no recording of %s: in si play, press p for the teleprompter and ● Record\n", filepath.Base(src))
		return exitProblems
	}
	n, err := loadNarration(dir)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if n.changed(src) {
		fmt.Fprintf(stderr, "si: note: %s changed since this recording\n", filepath.Base(src))
	}
	fmt.Fprint(stdout, paceReport(d, n, filepath.Base(dir)))
	return exitOK
}

var clickRE = regexp.MustCompile(`\[CLICK\]`)

// paceReport is the table of rehearse.
func paceReport(d *deck.Deck, n *narration, name string) string {
	spoken := map[int]float64{}
	for k, v := range n.spoken {
		spoken[k[0]] += v
	}
	mmss := func(s float64) string {
		if s <= 0 {
			return "-"
		}
		r := int(s + 0.5)
		return fmt.Sprintf("%d:%02d", r/60, r%60)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s, recorded %s\n\n%-6s %-34s %8s %8s %6s\n", d.Title(), name, "slide", "title", "planned", "spoken", "wpm")
	var plannedAll, spokenAll float64
	for i, s := range d.Slides {
		title := []rune(plainTitle(s.Title))
		if len(title) > 33 {
			title = append(title[:32], '…')
		}
		sp, pl := spoken[i], s.Minutes*60
		plannedAll += pl
		spokenAll += sp
		wpm := "-"
		words := len(strings.Fields(clickRE.ReplaceAllString(s.Notes, "")))
		if sp >= 5 && words > 0 {
			wpm = fmt.Sprint(int(float64(words)/(sp/60) + 0.5))
		}
		mark := ""
		switch {
		case sp == 0:
			mark = "  not shown"
		case pl > 0 && sp > pl*1.25:
			mark = "  ← long"
		case pl > 0 && sp < pl*0.75:
			mark = "  ← fast"
		}
		fmt.Fprintf(&b, "%-6d %-34s %8s %8s %6s%s\n", i+1, string(title), mmss(pl), mmss(sp), wpm, mark)
	}
	fmt.Fprintf(&b, "\n%-6s %-34s %8s %8s\n", "", "total", mmss(plannedAll), mmss(spokenAll))
	return b.String()
}

var mdMarkRE = regexp.MustCompile("[*_`=]")

func plainTitle(s string) string { return strings.TrimSpace(mdMarkRE.ReplaceAllString(s, "")) }
