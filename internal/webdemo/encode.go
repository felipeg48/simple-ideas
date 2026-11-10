package webdemo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The video side of si demo run: screencast frames laid out at 30 fps,
// captions as pictures over them, the voice mixed under, the features joined
// into all.mp4 with a title card and a chapter each. Every ffmpeg call gets
// -nostdin (it runs in a loop) and plain -filter_complex text (ffmpeg 9 has
// no -filter_complex_script).

const fps = 30

// cardSeconds is how long a feature's title card stays in all.mp4.
const cardSeconds = 3.0

// frame is a screencast frame and when it came, in seconds since the clip
// started.
type frame struct {
	File string
	At   float64
}

// sequence links frames into a 30 fps image sequence (seq00000.jpg …) of d
// seconds: frame k shows the last frame that came at or before k/30 s. It
// returns the ffmpeg input pattern and the number of frames.
func sequence(frames []frame, d float64, dir string) (string, int, error) {
	if len(frames) == 0 {
		return "", 0, fmt.Errorf("no frames were filmed")
	}
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].At < frames[j].At })
	n := max(1, int(d*fps+.5))
	j := 0
	for k := 0; k < n; k++ {
		t := float64(k) / fps
		for j+1 < len(frames) && frames[j+1].At <= t {
			j++
		}
		dst := filepath.Join(dir, fmt.Sprintf("seq%05d.jpg", k))
		if err := os.Link(frames[j].File, dst); err != nil {
			raw, err := os.ReadFile(frames[j].File)
			if err != nil {
				return "", 0, err
			}
			if err := os.WriteFile(dst, raw, 0o644); err != nil {
				return "", 0, err
			}
		}
	}
	return filepath.Join(dir, "seq%05d.jpg"), n, nil
}

var ffHead = []string{"-nostdin", "-y", "-hide_banner", "-loglevel", "error"}

func ff(a ...string) []string { return append(append([]string(nil), ffHead...), a...) }

const evenSize = "scale=trunc(iw/2)*2:trunc(ih/2)*2,setsar=1,format=yuv420p"

var h264 = []string{"-c:v", "libx264", "-preset", "medium", "-crf", "20", "-r", fmt.Sprint(fps), "-movflags", "+faststart"}

// caption is one caption on screen: its words and when.
type caption struct {
	Text     string
	From, To float64
	PNG      string // the picture burned in with --captions
}

// videoArgs encode the sequence into the silent clip; captions (pictures the
// size of the frame, transparent around the words) go over it when given.
func videoArgs(pattern string, caps []caption, out string) []string {
	a := ff("-framerate", fmt.Sprint(fps), "-i", pattern)
	var pics []caption
	for _, c := range caps {
		if c.PNG != "" {
			pics = append(pics, c)
		}
	}
	if len(pics) == 0 {
		return append(append(a, "-vf", evenSize), append(h264, out)...)
	}
	var g strings.Builder
	last := "0:v"
	for i, c := range pics {
		a = append(a, "-i", c.PNG)
		fmt.Fprintf(&g, "[%s][%d:v]overlay=0:0:enable='between(t,%.3f,%.3f)'[c%d];", last, i+1, c.From, c.To, i+1)
		last = fmt.Sprintf("c%d", i+1)
	}
	fmt.Fprintf(&g, "[%s]%s[v]", last, evenSize)
	return append(append(a, "-filter_complex", g.String(), "-map", "[v]"), append(h264, out)...)
}

var aac = []string{"-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-ac", "2"}

// muxArgs put the voice under the silent clip.
func muxArgs(video, wav, out string) []string {
	a := ff("-i", video, "-i", wav, "-map", "0:v", "-map", "1:a", "-c:v", "copy")
	return append(append(a, aac...), "-shortest", "-movflags", "+faststart", out)
}

// m4aArgs encode the voice alone.
func m4aArgs(wav, out string) []string {
	return append(ff("-i", wav, "-vn", "-c:a", "aac", "-b:a", "160k"), out)
}

// part is one feature in all.mp4: its title card and its clip.
type part struct {
	Name, Card, Clip string
	Seconds          float64 // the clip's length
}

// allArgs join the parts, each clip after its card (silent), into one video
// with the chapters in meta.
func allArgs(parts []part, meta, out string) []string {
	a := ff()
	var g strings.Builder
	var cat strings.Builder
	in := 0
	for i, p := range parts {
		a = append(a, "-loop", "1", "-framerate", fmt.Sprint(fps), "-t", fmt.Sprint(cardSeconds), "-i", p.Card,
			"-f", "lavfi", "-t", fmt.Sprint(cardSeconds), "-i", "anullsrc=r=48000:cl=stereo", "-i", p.Clip)
		fmt.Fprintf(&g, "[%d:v]%s,fps=%d[cv%d];[%d:a]aformat=sample_rates=48000:channel_layouts=stereo[ca%d];", in, evenSize, fps, i, in+1, i)
		fmt.Fprintf(&g, "[%d:v]%s,fps=%d[fv%d];[%d:a]aresample=48000,aformat=sample_rates=48000:channel_layouts=stereo[fa%d];", in+2, evenSize, fps, i, in+2, i)
		fmt.Fprintf(&cat, "[cv%d][ca%d][fv%d][fa%d]", i, i, i, i)
		in += 3
	}
	fmt.Fprintf(&g, "%sconcat=n=%d:v=1:a=1[v][a]", cat.String(), 2*len(parts))
	a = append(a, "-i", meta, "-filter_complex", g.String(), "-map", "[v]", "-map", "[a]", "-map_metadata", fmt.Sprint(in), "-map_chapters", fmt.Sprint(in))
	a = append(a, h264[:6]...) // codec, preset, crf
	return append(append(a, aac...), "-r", fmt.Sprint(fps), "-movflags", "+faststart", out)
}

// chapter is one feature's place in all.mp4.
type chapter struct {
	Title      string
	Start, End float64
}

// chapters places the parts: each chapter starts at its card.
func chapters(parts []part) []chapter {
	var cs []chapter
	t := 0.0
	for _, p := range parts {
		d := cardSeconds + float64(max(1, int(p.Seconds*fps+.5)))/fps
		cs = append(cs, chapter{p.Name, t, t + d})
		t += d
	}
	return cs
}

// ffmetadata is the chapters as an FFMETADATA1 file.
func ffmetadata(title string, cs []chapter) string {
	esc := strings.NewReplacer(`\`, `\\`, "=", `\=`, ";", `\;`, "#", `\#`, "\n", `\`+"\n")
	var b strings.Builder
	b.WriteString(";FFMETADATA1\n")
	if title != "" {
		fmt.Fprintf(&b, "title=%s\n", esc.Replace(title))
	}
	for _, c := range cs {
		fmt.Fprintf(&b, "\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=%d\nEND=%d\ntitle=%s\n", int(c.Start*1000+.5), int(c.End*1000+.5), esc.Replace(c.Title))
	}
	return b.String()
}

// youtube is chapters.txt: one `0:00 Title` line per chapter.
func youtube(cs []chapter) string {
	var b strings.Builder
	for _, c := range cs {
		s := int(c.Start)
		if s >= 3600 {
			fmt.Fprintf(&b, "%d:%02d:%02d %s\n", s/3600, s/60%60, s%60, c.Title)
		} else {
			fmt.Fprintf(&b, "%d:%02d %s\n", s/60, s%60, c.Title)
		}
	}
	return b.String()
}

// captionMax is the most a caption holds: two lines of captionLine.
const (
	captionLine = 42
	captionMax  = 2 * captionLine
)

// captions splits a Say: line spoken from at for secs into captions of at
// most two lines, each on screen for its share of the words.
func captions(text string, at, secs float64) []caption {
	words := strings.Fields(text)
	var chunks []string
	cur := ""
	for i, w := range words {
		if cur != "" && len(cur)+1+len(w) > captionMax {
			chunks, cur = append(chunks, cur), ""
		}
		if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
		// a sentence end past half a caption is a good place to break
		if end := strings.HasSuffix(w, ".") || strings.HasSuffix(w, "?") || strings.HasSuffix(w, "!"); end && len(cur) > captionMax/2 && i < len(words)-1 {
			chunks, cur = append(chunks, cur), ""
		}
	}
	if cur != "" {
		chunks = append(chunks, cur)
	}
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	var out []caption
	t := at
	for _, c := range chunks {
		d := secs * float64(len(c)) / float64(total)
		out = append(out, caption{Text: c, From: t, To: t + d})
		t += d
	}
	return out
}

// wrap breaks a caption into lines of at most captionLine (two at most).
func wrap(s string) []string {
	if len(s) <= captionLine {
		return []string{s}
	}
	// the space nearest the middle that leaves both lines short enough
	best := -1
	for i := range s {
		if s[i] == ' ' && (best < 0 || abs(i-len(s)/2) < abs(best-len(s)/2)) {
			best = i
		}
	}
	if best < 0 {
		return []string{s}
	}
	return []string{s[:best], s[best+1:]}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// srt is the captions as SubRip.
func srt(cs []caption) string {
	ts := func(t float64) string {
		ms := int(t*1000 + .5)
		return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
	}
	var b strings.Builder
	for i, c := range cs {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, ts(c.From), ts(c.To), strings.Join(wrap(c.Text), "\n"))
	}
	return b.String()
}
