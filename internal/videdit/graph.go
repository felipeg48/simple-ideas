package videdit

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// Job is one ffmpeg run: the plan's pieces joined, then the hidden places,
// the callouts, the camera and the sound lines over the result.
type Job struct {
	Src       string // the (joined) source
	W, H      int    // its size (even)
	Audio     bool   // it has sound
	Plan      *Plan
	Spans     []Span
	Overlay   string          // the callout frames (ov%05d.png), "" for none
	Cards     map[*Seg]string // a card's picture
	ClipAudio map[string]bool // clips with sound
	Says      []Said
	Meta      string // ffmetadata with the chapters, "" for none
	Out       string
}

// Said is a voiced line at an output time, the sound under it ducked to Duck.
type Said struct {
	WAV       string
	At, Len   float64
	Duck      float64
	Text      string
	Line      int
	Speakable bool
}

func f3(v float64) string { return fmt.Sprintf("%.3f", v) }

// Args are the ffmpeg arguments of the job.
func (j *Job) Args() []string {
	a := append([]string(nil), webdemo.FFHead...)
	var g strings.Builder
	in := 0
	input := func(args ...string) int {
		a = append(a, args...)
		in++
		return in - 1
	}
	W, H := j.W, j.H
	fit := fmt.Sprintf("fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=lanczos,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1,format=yuv420p,settb=1/%d",
		webdemo.FPS, W, H, W, H, webdemo.FPS)
	anorm := "aresample=48000,aformat=sample_fmts=fltp:sample_rates=48000:channel_layouts=stereo"
	silence := func(L float64) string { return fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%s", f3(L)) }
	segs := j.Plan.Segs
	var cat strings.Builder
	for i, s := range segs {
		L := s.Len
		hold := ""
		if i+1 < len(segs) && segs[i+1].Trans != nil {
			hold = fmt.Sprintf(",tpad=stop_mode=clone:stop_duration=%s", f3(segs[i+1].Trans.Dur))
		}
		vEnd := fmt.Sprintf("trim=duration=%s,setpts=PTS-STARTPTS%s[v%d];", f3(L), hold, i)
		aEnd := fmt.Sprintf("%s,apad=whole_dur=%s,atrim=duration=%s,asetpts=PTS-STARTPTS[a%d];", anorm, f3(L), f3(L), i)
		switch s.Kind {
		case "src":
			file, audio := j.Src, j.Audio
			if s.File != "" { // a source of a join
				file, audio = s.File, j.ClipAudio[s.File]
			}
			k := input("-ss", f3(s.A), "-t", f3(s.B-s.A), "-i", file)
			fmt.Fprintf(&g, "[%d:v]setpts=(PTS-STARTPTS)/%g,%s,%s", k, s.Speed, fit, vEnd)
			if audio && !s.Mute {
				fmt.Fprintf(&g, "[%d:a]asetpts=PTS-STARTPTS%s,%s", k, atempo(s.Speed), aEnd)
			} else {
				fmt.Fprintf(&g, "%s,%s", silence(L), aEnd)
			}
		case "still":
			ss := max(0, min(s.A, j.Plan.D-0.1))
			k := input("-ss", f3(ss), "-t", "1", "-i", j.Src)
			fmt.Fprintf(&g, "[%d:v]%s,trim=end_frame=1,tpad=stop_mode=clone:stop_duration=%s,%s", k, fit, f3(L), vEnd)
			fmt.Fprintf(&g, "%s,%s", silence(L), aEnd)
		case "clip":
			k := input("-i", s.File)
			fmt.Fprintf(&g, "[%d:v]setpts=PTS-STARTPTS,%s,tpad=stop_mode=clone:stop_duration=%s,%s", k, fit, f3(L), vEnd)
			switch {
			case s.Keep && j.Audio:
				o := input("-ss", f3(s.A), "-t", f3(s.B-s.A), "-i", j.Src)
				fmt.Fprintf(&g, "[%d:a]asetpts=PTS-STARTPTS,%s", o, aEnd)
			case !s.Keep && j.ClipAudio[s.File]:
				fmt.Fprintf(&g, "[%d:a]asetpts=PTS-STARTPTS,%s", k, aEnd)
			default:
				fmt.Fprintf(&g, "%s,%s", silence(L), aEnd)
			}
		case "image", "card":
			file := s.File
			if s.Kind == "card" {
				file = j.Cards[s]
			}
			k := input("-loop", "1", "-framerate", fmt.Sprint(webdemo.FPS), "-t", f3(L+1), "-i", file)
			fmt.Fprintf(&g, "[%d:v]%s,%s", k, fit, vEnd)
			fmt.Fprintf(&g, "%s,%s", silence(L), aEnd)
		}
		fmt.Fprintf(&cat, "[a%d]", i)
	}
	last := "v0"
	t := 0.0
	for i := 1; i < len(segs); i++ {
		t += segs[i-1].Len
		if tr := segs[i].Trans; tr != nil {
			fmt.Fprintf(&g, "[%s][v%d]xfade=transition=%s:duration=%s:offset=%s,settb=1/%d[x%d];", last, i, webdemo.XfadeName(tr.Kind), f3(tr.Dur), f3(t), webdemo.FPS, i)
		} else {
			fmt.Fprintf(&g, "[%s][v%d]concat=n=2:v=1:a=0,settb=1/%d[x%d];", last, i, webdemo.FPS, i)
		}
		last = fmt.Sprintf("x%d", i)
	}
	fmt.Fprintf(&g, "%sconcat=n=%d:v=0:a=1[aj];", cat.String(), len(segs))
	// the hidden places
	n := 0
	for _, s := range j.Spans {
		it := s.It
		if it.Action != "blur" && it.Action != "pixelate" {
			continue
		}
		x, y, w, h := clampRect(it.Rect, W, H)
		if w < 2 || h < 2 {
			continue
		}
		f := fmt.Sprintf("gblur=sigma=%g", math.Max(8, math.Round(float64(min(w, h))/5)))
		if it.Action == "pixelate" {
			c := max(2, min(w, h)/8, 8)
			f = fmt.Sprintf("scale=%d:%d:flags=area,scale=%d:%d:flags=neighbor", max(1, w/c), max(1, h/c), w, h)
		}
		fmt.Fprintf(&g, "[%s]split[h%dm][h%dc];[h%dc]crop=%d:%d:%d:%d,%s[h%db];[h%dm][h%db]overlay=%d:%d:enable='between(t,%s,%s)'[h%d];",
			last, n, n, n, w, h, x, y, f, n, n, n, x, y, f3(s.From), f3(s.To), n)
		last = fmt.Sprintf("h%d", n)
		n++
	}
	if j.Overlay != "" {
		k := input("-framerate", fmt.Sprint(webdemo.FPS), "-i", j.Overlay)
		fmt.Fprintf(&g, "[%s][%d:v]overlay=0:0:eof_action=pass[ov];", last, k)
		last = "ov"
	}
	if cam := cameraOf(j.Spans, W, H); len(cam) > 0 {
		rw, cx, cy := camExpr(cam, 2), camExpr(cam, 0), camExpr(cam, 1)
		z := fmt.Sprintf("(%d/(%s))", W, rw) // the zoom; crop sees its input's first size only, so x and y use it
		fmt.Fprintf(&g, "[%s]scale=w='2*trunc(%d*%s/2)':h='2*trunc(%d*%s/2)':eval=frame:flags=bicubic,crop=%d:%d:x='clip((%s)*%s-%d/2,0,trunc(%d*%s/2)*2-%d)':y='clip((%s)*%s-%d/2,0,trunc(%d*%s/2)*2-%d)'[cam];",
			last, W, z, H, z, W, H, cx, z, W, W, z, W, cy, z, H, H, z, H)
		last = "cam"
	}
	fmt.Fprintf(&g, "[%s]format=yuv420p[v];", last)
	// the sound
	alast := "aj"
	var vol []string
	for _, s := range j.Spans {
		if s.It.Action == "mute" {
			vol = append(vol, fmt.Sprintf("volume=0:enable='between(t,%s,%s)'", f3(s.From), f3(s.To)))
		}
	}
	for _, s := range j.Says {
		if s.Duck < 1 {
			vol = append(vol, fmt.Sprintf("volume=%g:enable='between(t,%s,%s)'", s.Duck, f3(s.At), f3(s.At+s.Len)))
		}
	}
	if len(vol) > 0 {
		fmt.Fprintf(&g, "[aj]%s[ad];", strings.Join(vol, ","))
		alast = "ad"
	}
	if len(j.Says) > 0 {
		mix := "[" + alast + "]"
		for i, s := range j.Says {
			k := input("-i", s.WAV)
			ms := int(s.At*1000 + .5)
			fmt.Fprintf(&g, "[%d:a]%s,adelay=%d:all=1[s%d];", k, anorm, ms, i)
			mix += fmt.Sprintf("[s%d]", i)
		}
		fmt.Fprintf(&g, "%samix=inputs=%d:duration=first:dropout_transition=0:normalize=0[am];", mix, len(j.Says)+1)
		alast = "am"
	}
	fmt.Fprintf(&g, "[%s]anull[a]", alast)
	meta := -1
	if j.Meta != "" {
		meta = input("-i", j.Meta)
	}
	a = append(a, "-filter_complex", g.String(), "-map", "[v]", "-map", "[a]")
	if meta >= 0 {
		a = append(a, "-map_metadata", fmt.Sprint(meta), "-map_chapters", fmt.Sprint(meta))
	}
	a = append(a, "-t", f3(j.Plan.Len))
	a = append(a, webdemo.H264...)
	a = append(a, webdemo.AAC...)
	return append(a, j.Out)
}

// atempo is the filter that plays sound s times faster, its pitch kept
// (atempo takes 0.5…100 per stage).
func atempo(s float64) string {
	if s == 1 {
		return ""
	}
	var parts []string
	for s < 0.5 {
		parts = append(parts, "atempo=0.5")
		s /= 0.5
	}
	parts = append(parts, fmt.Sprintf("atempo=%g", s))
	return "," + strings.Join(parts, ",")
}

// clampRect is r in whole pixels inside a W×H frame (even, for crop).
func clampRect(r *Rect, W, H int) (x, y, w, h int) {
	fx, fy, fw, fh := r.Px(W, H)
	x0, y0 := max(0, int(fx)), max(0, int(fy))
	x1, y1 := min(W, int(math.Ceil(fx+fw))), min(H, int(math.Ceil(fy+fh)))
	return x0, y0, (x1 - x0) &^ 1, (y1 - y0) &^ 1
}

// move is the camera going from one view to another: views are
// [centre x, centre y, width] in source pixels (the height follows the
// frame's shape).
type move struct {
	T0, T1   float64
	From, To [3]float64
	Linear   bool
}

// view is a rectangle as a camera view: the frame's shape around its centre,
// inside the frame.
func view(r *Rect, W, H int) [3]float64 {
	x, y, w, h := r.Px(W, H)
	cx, cy := x+w/2, y+h/2
	if w/h < float64(W)/float64(H) {
		w = h * float64(W) / float64(H)
	}
	w = math.Min(math.Max(w, float64(W)/8), float64(W))
	h = w * float64(H) / float64(W)
	cx = math.Min(math.Max(cx, w/2), float64(W)-w/2)
	cy = math.Min(math.Max(cy, h/2), float64(H)-h/2)
	return [3]float64{cx, cy, w}
}

// cameraOf are the camera's moves, from the zoom to, pan to and zoom out
// spans in time order.
func cameraOf(spans []Span, W, H int) []move {
	full := [3]float64{float64(W) / 2, float64(H) / 2, float64(W)}
	var cs []Span
	for _, s := range spans {
		if a := s.It.Action; a == "zoom" || a == "pan" || a == "zoom-out" {
			cs = append(cs, s)
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].From < cs[j].From })
	var ms []move
	cur, free := full, 0.0
	for _, s := range cs {
		it := s.It
		t0 := math.Max(s.From, free)
		to := full
		if it.Action != "zoom-out" {
			to = view(it.Rect, W, H)
		}
		ms = append(ms, move{t0, t0 + it.Move, cur, to, it.Flags["linear"]})
		cur, free = to, t0+it.Move
		if it.HasDur && it.Action != "zoom-out" {
			b := free + it.Dur
			ms = append(ms, move{b, b + it.Move, to, full, it.Flags["linear"]})
			cur, free = full, b+it.Move
		}
	}
	return ms
}

// camExpr is component c of the view over time as an ffmpeg expression of t.
func camExpr(ms []move, c int) string {
	e := fmt.Sprintf("%.2f", ms[len(ms)-1].To[c])
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		p := fmt.Sprintf("((t-%s)/%s)", f3(m.T0), f3(math.Max(m.T1-m.T0, 0.001)))
		ease := fmt.Sprintf("(%s*%s*(3-2*%s))", p, p, p)
		if m.Linear {
			ease = p
		}
		lerp := fmt.Sprintf("%.2f+(%.2f)*%s", m.From[c], m.To[c]-m.From[c], ease)
		e = fmt.Sprintf("if(lt(t,%s),%.2f,if(lt(t,%s),%s,%s))", f3(m.T0), m.From[c], f3(m.T1), lerp, e)
	}
	return e
}
