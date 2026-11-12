package webdemo

import (
	"fmt"
	"strings"
)

// With is what si demo run writes besides <script>.mp4, the whole run with
// its voice, title cards and chapters: --with, the script's with: or
// demo.with in si.conf, a comma list of WithNames.
type With struct {
	Captions   bool // <script>.captions.mp4: the captions in a band under the picture
	Silent     bool // <script>.silent.mp4: the picture alone
	Audio      bool // <script>.m4a: the narration alone
	Subtitles  bool // <script>.srt + <script>.vtt
	Transcript bool // <script>.txt: what Say: says, as prose
	Chapters   bool // chapters.txt: YouTube chapter lines
	Clips      bool // the same kinds per feature: <feature>.mp4, …
	Report     bool // report.html + report.json
	Readme     bool // README.md + README.html
}

// WithNames are the names --with takes.
var WithNames = []string{"captions", "silent", "audio", "subtitles", "transcript", "chapters", "clips", "report", "readme", "all"}

// Add turns on the names of a comma list ("" adds nothing); an unknown name
// is an error that lists the known ones.
func (w *With) Add(list string) error {
	for _, n := range strings.Split(list, ",") {
		n = strings.ToLower(strings.TrimSpace(n))
		switch n {
		case "":
		case "captions":
			w.Captions = true
		case "silent":
			w.Silent = true
		case "audio":
			w.Audio = true
		case "subtitles":
			w.Subtitles = true
		case "transcript":
			w.Transcript = true
		case "chapters":
			w.Chapters = true
		case "clips":
			w.Clips = true
		case "report":
			w.Report = true
		case "readme":
			w.Readme = true
		case "all":
			*w = With{true, true, true, true, true, true, true, true, true}
		default:
			return fmt.Errorf("with %q: use %s", n, strings.Join(WithNames, ", "))
		}
	}
	return nil
}

// Planned is a filmed feature as the plan sees it: its file stem and whether
// it has narration.
type Planned struct {
	Stem   string
	Voiced bool
}

// Files is what a run with w writes, in order, for the script named script
// and the features filmed (nothing of the videos when none was). Captions,
// audio, subtitles and the transcript need narration.
func (w With) Files(script string, fs []Planned) []string {
	var out []string
	add := func(ok bool, names ...string) {
		if ok {
			out = append(out, names...)
		}
	}
	voiced := false
	for _, f := range fs {
		voiced = voiced || f.Voiced
	}
	if len(fs) > 0 {
		add(true, script+".mp4")
		add(w.Captions && voiced, script+".captions.mp4")
		add(w.Silent, script+".silent.mp4")
		add(w.Audio && voiced, script+".m4a")
		add(w.Subtitles && voiced, script+".srt", script+".vtt")
		add(w.Transcript && voiced, script+".txt")
		add(w.Chapters, "chapters.txt")
	}
	if w.Clips {
		for _, f := range fs {
			add(true, f.Stem+".mp4")
			add(w.Captions && f.Voiced, f.Stem+".captions.mp4")
			add(w.Silent, f.Stem+".silent.mp4")
			add(w.Audio && f.Voiced, f.Stem+".m4a")
			add(w.Subtitles && f.Voiced, f.Stem+".srt", f.Stem+".vtt")
		}
	}
	add(w.Report, "report.html", "report.json", "shots")
	add(w.Readme, "README.md", "README.html")
	return out
}

// transcript is <script>.txt: each feature's name on a line of its own, then
// what its Say: lines say as one paragraph, the words as written (not as
// pronounce: makes the voice say them).
func transcript(fs []*Feature) string {
	var b strings.Builder
	for _, f := range fs {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.Name + "\n")
		var says []string
		for _, bl := range f.Blocks {
			if s := strings.TrimSpace(bl.Say); s != "" {
				says = append(says, s)
			}
		}
		if len(says) > 0 {
			b.WriteString("\n" + strings.Join(says, " ") + "\n")
		}
	}
	return b.String()
}
