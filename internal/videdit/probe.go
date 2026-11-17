package videdit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

// FFmpeg is the ffmpeg si runs ("" when none); FFprobe the ffprobe beside it.
func FFmpeg() string { return tools.Find("ffmpeg") }

func FFprobe() string {
	if ff := FFmpeg(); ff != "" {
		name := "ffprobe"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if p := filepath.Join(filepath.Dir(ff), name); fileExists(p) {
			return p
		}
	}
	p, _ := exec.LookPath("ffprobe")
	return p
}

func fileExists(p string) bool { st, err := os.Stat(p); return err == nil && !st.IsDir() }

// Info is what ffprobe says about a video.
type Info struct {
	D        float64 // seconds
	W, H     int
	Audio    bool
	Video    bool
	Chapters []Chapter
}

// Chapter is a named start time.
type Chapter struct {
	Title string
	Start float64
}

// Probe asks ffprobe about a file.
func Probe(ffprobe, file string) (Info, error) {
	out, err := exec.Command(ffprobe, "-v", "error", "-print_format", "json", "-show_format", "-show_streams", "-show_chapters", file).Output()
	if err != nil {
		return Info{}, fmt.Errorf("ffprobe %s: %v", filepath.Base(file), err)
	}
	var j struct {
		Format  struct{ Duration string }
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int
			Height    int
			Duration  string
		}
		Chapters []struct {
			StartTime string `json:"start_time"`
			Tags      struct{ Title string }
		}
	}
	if err := json.Unmarshal(out, &j); err != nil {
		return Info{}, err
	}
	var in Info
	in.D, _ = strconv.ParseFloat(j.Format.Duration, 64)
	for _, s := range j.Streams {
		switch s.CodecType {
		case "video":
			if !in.Video {
				in.W, in.H, in.Video = s.Width, s.Height, true
			}
		case "audio":
			in.Audio = true
		}
	}
	for _, c := range j.Chapters {
		t, _ := strconv.ParseFloat(c.StartTime, 64)
		in.Chapters = append(in.Chapters, Chapter{c.Tags.Title, t})
	}
	return in, nil
}

// Keyframe is a keyframe's display and decode times.
type Keyframe struct{ PTS, DTS float64 }

// Keyframes are a video's keyframes in order.
func Keyframes(ffprobe, file string) ([]Keyframe, error) {
	out, err := exec.Command(ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "packet=pts_time,dts_time,flags", "-of", "csv=p=0", file).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe keyframes: %v", err)
	}
	var ks []Keyframe
	for _, l := range strings.Fields(string(out)) {
		f := strings.Split(l, ",")
		if len(f) < 3 || !strings.Contains(f[2], "K") {
			continue
		}
		p, err1 := strconv.ParseFloat(f[0], 64)
		d, err2 := strconv.ParseFloat(f[1], 64)
		if err1 == nil && err2 == nil {
			ks = append(ks, Keyframe{p, d})
		}
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].PTS < ks[j].PTS })
	return ks, nil
}

// Cue is a caption: its words and when.
type Cue struct {
	Text     string
	From, To float64
}

var cueTimeRE = regexp.MustCompile(`(\d+:)?(\d+):(\d+)[.,](\d+)\s*-->\s*(\d+:)?(\d+):(\d+)[.,](\d+)`)

// ReadCues reads a WebVTT or SubRip file.
func ReadCues(file string) ([]Cue, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var cs []Cue
	var cur *Cue
	for _, l := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if m := cueTimeRE.FindStringSubmatch(l); m != nil {
			cs = append(cs, Cue{From: hms(m[1], m[2], m[3], m[4]), To: hms(m[5], m[6], m[7], m[8])})
			cur = &cs[len(cs)-1]
			continue
		}
		if strings.TrimSpace(l) == "" {
			cur = nil
			continue
		}
		if cur != nil {
			if cur.Text != "" {
				cur.Text += " "
			}
			cur.Text += strings.TrimSpace(l)
		}
	}
	return cs, nil
}

func hms(h, m, s, frac string) float64 {
	hv, _ := strconv.Atoi(strings.TrimSuffix(h, ":"))
	mv, _ := strconv.Atoi(m)
	sv, _ := strconv.Atoi(s)
	fv, _ := strconv.ParseFloat("0."+frac, 64)
	return float64(hv*3600+mv*60+sv) + fv
}

// ReadChaptersTxt reads YouTube chapter lines (0:42 Install).
func ReadChaptersTxt(file string) ([]Chapter, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var cs []Chapter
	for _, l := range strings.Split(string(raw), "\n") {
		t, name, ok := strings.Cut(strings.TrimSpace(l), " ")
		if !ok {
			continue
		}
		if v, ok := clock(t); ok {
			cs = append(cs, Chapter{strings.TrimSpace(strings.TrimLeft(name, "-–— ")), v})
		}
	}
	return cs, nil
}

// cuesBeside are the captions beside a video (<stem>.vtt or .srt), nil
// when none.
func cuesBeside(video string) []Cue {
	stem := strings.TrimSuffix(video, filepath.Ext(video))
	for _, ext := range []string{".vtt", ".srt"} {
		if cs, err := ReadCues(stem + ext); err == nil {
			return cs
		}
	}
	return nil
}

// chaptersOf are a video's chapters: embedded, else chapters.txt beside it.
func chaptersOf(video string, in Info) []Chapter {
	if len(in.Chapters) > 0 {
		return in.Chapters
	}
	cs, _ := ReadChaptersTxt(filepath.Join(filepath.Dir(video), "chapters.txt"))
	return cs
}

// saidAt is when phrase starts being said in cues: the start of the first
// cue holding it (cues may split a sentence: a phrase across two cues is
// found too), plus its letter position's share of the cue.
func saidAt(cues []Cue, phrase string) (float64, bool) {
	p := normWords(phrase)
	if p == "" {
		return 0, false
	}
	for i, c := range cues {
		text := normWords(c.Text)
		if i+1 < len(cues) {
			text += " " + normWords(cues[i+1].Text)
		}
		k := strings.Index(text, p)
		own := normWords(c.Text)
		if k < 0 || k > len(own) {
			continue
		}
		frac := 0.0
		if len(own) > 0 {
			frac = float64(k) / float64(len(own))
		}
		return c.From + frac*(c.To-c.From), true
	}
	return 0, false
}

func normWords(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r == '’' || r == '\'' {
			return -1
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 {
			return r
		}
		return ' '
	}, s)
	return strings.Join(strings.Fields(s), " ")
}
