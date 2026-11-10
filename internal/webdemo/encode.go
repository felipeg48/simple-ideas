package webdemo

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The video side of si demo run: screencast frames laid out at 30 fps (long
// waits fast-forwarded), the voice mixed under, in the captions variant the
// picture shrunk over a band where the captions go, the features joined into
// all.mp4 with a title card and a chapter each. Every ffmpeg call gets
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

// sequence links frames into a 30 fps image sequence (seq00000.jpg …) of the
// d seconds of the take as the video plays them (fs: the stretches played
// faster): output frame k shows the last frame that came at or before the
// take time it maps to. It returns the ffmpeg input pattern and the number of
// frames.
func sequence(frames []frame, d float64, fs []fast, dir string) (string, int, error) {
	if len(frames) == 0 {
		return "", 0, fmt.Errorf("no frames were filmed")
	}
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].At < frames[j].At })
	n := max(1, int(outT(fs, d)*fps+.5))
	j := 0
	for k := 0; k < n; k++ {
		t := inT(fs, float64(k)/fps)
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

// span is a stretch of a take, in seconds since it started.
type span struct{ From, To float64 }

// fast is a stretch of the take the video plays X times faster.
type fast struct {
	From, To float64
	X        int
}

// ffMost is how long a fast-forwarded stretch lasts at most, once sped up.
const ffMost = 2.0

// fastForward picks the stretches of a d-second take to play faster: each
// wait beyond its first after seconds, never under the narration (the voice
// plays at its own pace over the picture at its own pace), sped up so it
// lasts ffMost seconds at most. after <= 0: none.
func fastForward(waits, spoken []span, after, d float64) []fast {
	if after <= 0 {
		return nil
	}
	var cand []span
	for _, w := range waits {
		if from, to := w.From+after, min(w.To, d); from < to {
			cand = append(cand, span{from, to})
		}
	}
	for _, v := range spoken {
		var keep []span
		for _, c := range cand {
			if v.To <= c.From || v.From >= c.To {
				keep = append(keep, c)
				continue
			}
			if c.From < v.From {
				keep = append(keep, span{c.From, v.From})
			}
			if v.To < c.To {
				keep = append(keep, span{v.To, c.To})
			}
		}
		cand = keep
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].From < cand[j].From })
	var fs []fast
	for _, c := range cand {
		if x := int(math.Ceil((c.To - c.From) / ffMost)); x >= 2 {
			fs = append(fs, fast{c.From, c.To, x})
		}
	}
	return fs
}

// outT is where take time t falls in the video.
func outT(fs []fast, t float64) float64 {
	o := t
	for _, f := range fs {
		if t > f.From {
			o -= (min(t, f.To) - f.From) * (1 - 1/float64(f.X))
		}
	}
	return o
}

// inT is the take time video time o shows (outT's inverse).
func inT(fs []fast, o float64) float64 {
	shift := 0.0
	for _, f := range fs {
		start := f.From - shift
		if o < start {
			break
		}
		l := f.To - f.From
		if o < start+l/float64(f.X) {
			return f.From + (o-start)*float64(f.X)
		}
		shift += l - l/float64(f.X)
	}
	return o + shift
}

var ffHead = []string{"-nostdin", "-y", "-hide_banner", "-loglevel", "error"}

func ff(a ...string) []string { return append(append([]string(nil), ffHead...), a...) }

const evenSize = "scale=trunc(iw/2)*2:trunc(ih/2)*2:out_range=tv,setsar=1,format=yuv420p"

var h264 = []string{"-c:v", "libx264", "-preset", "medium", "-crf", "20", "-r", fmt.Sprint(fps), "-movflags", "+faststart"}

// caption is one caption on screen: its words and when.
type caption struct {
	Text     string
	From, To float64
}

// pic is a picture the size of the frame, transparent around what it draws
// (a caption, a fast-forward badge), over the video from From to To.
type pic struct {
	PNG      string
	From, To float64
}

// The captions variant: the picture shrinks to picScale of the frame, picTop
// below its top edge, centered, on bandColor; the captions go in the band
// under it (1 - picTop - picScale of the height).
const (
	picScale  = 0.85
	picTop    = 0.02
	bandColor = "0x111318"
)

func even(f float64) int { return int(f/2) * 2 }

// band is the filter that lays a w×h picture out for the captions variant.
func band(w, h int) string {
	sh := even(float64(h) * picScale)
	sw := even(float64(w) * float64(sh) / float64(h))
	return fmt.Sprintf("scale=%d:%d,pad=%d:%d:%d:%d:color=%s", sw, sh, w, h, (w-sw)/2, even(float64(h)*picTop), bandColor)
}

// videoArgs encode the sequence of w×h frames with the badges over it. With
// caps, the picture shrinks into the top of the frame (band), the captions go
// under it and wav is the voice: the captions variant.
func videoArgs(pattern string, w, h int, badges, caps []pic, wav, out string) []string {
	a := ff("-framerate", fmt.Sprint(fps), "-i", pattern)
	if len(badges) == 0 && len(caps) == 0 && wav == "" {
		return append(append(a, "-vf", evenSize), append(h264, out)...)
	}
	var g strings.Builder
	last, in := "0:v", 1
	lay := func(ps []pic) {
		for _, p := range ps {
			a = append(a, "-i", p.PNG)
			fmt.Fprintf(&g, "[%s][%d:v]overlay=0:0:enable='between(t,%.3f,%.3f)'[o%d];", last, in, p.From, p.To, in)
			last = fmt.Sprintf("o%d", in)
			in++
		}
	}
	lay(badges)
	if len(caps) > 0 {
		fmt.Fprintf(&g, "[%s]%s[band];", last, band(w, h))
		last = "band"
		lay(caps)
	}
	fmt.Fprintf(&g, "[%s]%s[v]", last, evenSize)
	if wav != "" {
		a = append(a, "-i", wav)
	}
	a = append(a, "-filter_complex", g.String(), "-map", "[v]")
	if wav != "" {
		a = append(append(a, "-map", fmt.Sprintf("%d:a", in)), aac...)
		a = append(a, "-shortest")
	}
	return append(a, append(h264, out)...)
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

// joinCaptions puts each clip's captions on all.mp4's clock: after the
// title card of its chapter.
func joinCaptions(cs []chapter, caps [][]caption) []caption {
	var all []caption
	for i, c := range cs {
		for _, k := range caps[i] {
			all = append(all, caption{k.Text, c.Start + cardSeconds + k.From, c.Start + cardSeconds + k.To})
		}
	}
	return all
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
	var b strings.Builder
	for i, c := range cs {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, stamp(c.From, ","), stamp(c.To, ","), strings.Join(wrap(c.Text), "\n"))
	}
	return b.String()
}

// vtt is the captions as WebVTT (what a browser's <track> reads).
func vtt(cs []caption) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for _, c := range cs {
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n", stamp(c.From, "."), stamp(c.To, "."), strings.Join(wrap(c.Text), "\n"))
	}
	return b.String()
}

// stamp is t as hh:mm:ss<sep>mmm.
func stamp(t float64, sep string) string {
	ms := int(t*1000 + .5)
	return fmt.Sprintf("%02d:%02d:%02d%s%03d", ms/3600000, ms/60000%60, ms/1000%60, sep, ms%1000)
}
