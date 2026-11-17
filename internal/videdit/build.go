package videdit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// Options are how Build runs.
type Options struct {
	Chrome, FFmpeg, FFprobe string
	OutDir                  string // where out: goes ("" = beside the edit)
	Captions, Vertical      bool   // --with captions, vertical: the promo kit's variants
	Voice                   string // --voice, over voice: and si.conf
	Config                  *config.Config
	Synth                   webdemo.Voicer // nil: the voice programs, through the cache
	Log                     io.Writer
	SI                      string // si's version (refilm reports)
}

func (o *Options) logf(f string, a ...any) {
	if o.Log != nil {
		fmt.Fprintf(o.Log, f, a...)
	}
}

// Result is what Build wrote.
type Result struct {
	Out   string
	Files []string // besides Out: captions, variants
	Len   float64
	Fast  bool // refilm spliced with stream copy
}

// Ctx is an edit with its source probed and its times resolved.
type Ctx struct {
	E     *Edit
	Src   string // the (joined) source
	Info  Info
	Cues  []Cue
	Chaps []Chapter
}

// Source joins (and trims) the sources when there are several or one is
// trimmed, into <dir>/.si/video/<name>.joined.mp4 (kept while the sources
// and the join are the same); else the one source as it is.
func (e *Edit) Source(ffmpeg, ffprobe string, log io.Writer) (string, error) {
	if len(e.Sources) == 0 {
		return "", fmt.Errorf("no source")
	}
	if len(e.Sources) == 1 && e.Sources[0].From == 0 && e.Sources[0].To == 0 {
		return e.Abs(e.Sources[0].File), nil
	}
	if ffmpeg == "" {
		return "", fmt.Errorf("joining sources needs ffmpeg")
	}
	h := sha256.New()
	fmt.Fprintf(h, "%v|", e.Join)
	for _, s := range e.Sources {
		st, err := os.Stat(e.Abs(s.File))
		if err != nil {
			return "", fmt.Errorf("source %s: %v", s.File, err)
		}
		fmt.Fprintf(h, "%s|%d|%d|%g|%g|", e.Abs(s.File), st.Size(), st.ModTime().UnixNano(), s.From, s.To)
	}
	key := hex.EncodeToString(h.Sum(nil))[:16]
	dir := filepath.Join(e.Dir, ".si", "video")
	name := strings.TrimSuffix(filepath.Base(e.Path), filepath.Ext(e.Path))
	out := filepath.Join(dir, name+".joined.mp4")
	if k, err := os.ReadFile(out + ".key"); err == nil && string(k) == key && fileExists(out) {
		return out, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	j := &Job{Plan: &Plan{}, ClipAudio: map[string]bool{}, Out: out}
	for i, s := range e.Sources {
		f := e.Abs(s.File)
		in, err := Probe(ffprobe, f)
		if err != nil {
			return "", err
		}
		if i == 0 {
			j.W, j.H = in.W&^1, in.H&^1
		}
		to := s.To
		if to == 0 || to > in.D {
			to = in.D
		}
		seg := &Seg{Kind: "src", File: f, A: s.From, B: to, Speed: 1, Len: frames(to - s.From), Out: j.Plan.Len}
		if i > 0 && e.Join.Kind != "" {
			tr := e.Join
			seg.Trans = &tr
		}
		j.ClipAudio[f] = in.Audio
		j.Plan.Segs = append(j.Plan.Segs, seg)
		j.Plan.Len += seg.Len
	}
	if log != nil {
		fmt.Fprintf(log, "  joining %d sources\n", len(e.Sources))
	}
	if err := run(ffmpeg, j.Args()); err != nil {
		return "", err
	}
	return out, os.WriteFile(out+".key", []byte(key), 0o644)
}

func frames(sec float64) float64 {
	return math.Max(1, math.Round(sec*webdemo.FPS)) / webdemo.FPS
}

func run(ffmpeg string, args []string) error {
	out, err := exec.Command(ffmpeg, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 1500 {
			msg = msg[len(msg)-1500:]
		}
		return fmt.Errorf("ffmpeg: %v\n%s", err, msg)
	}
	return nil
}

// Prepare joins and probes the source and resolves the times.
func (e *Edit) Prepare(ffmpeg, ffprobe string, log io.Writer) (*Ctx, error) {
	src, err := e.Source(ffmpeg, ffprobe, log)
	if err != nil {
		return nil, err
	}
	in, err := Probe(ffprobe, src)
	if err != nil {
		return nil, err
	}
	if !in.Video {
		return nil, fmt.Errorf("%s has no picture", filepath.Base(src))
	}
	c := &Ctx{E: e, Src: src, Info: in}
	if len(e.Sources) == 1 { // the captions and chapters beside the one source, moved by its from:
		orig := e.Abs(e.Sources[0].File)
		oi := in
		if orig != src {
			oi, _ = Probe(ffprobe, orig)
		}
		from := e.Sources[0].From
		for _, q := range cuesBeside(orig) {
			if q.To > from {
				c.Cues = append(c.Cues, Cue{q.Text, math.Max(0, q.From-from), q.To - from})
			}
		}
		for _, ch := range chaptersOf(orig, oi) {
			if ch.Start >= from {
				c.Chaps = append(c.Chaps, Chapter{ch.Title, ch.Start - from})
			}
		}
	}
	e.Resolve(in.D, c.Cues, c.Chaps)
	return c, nil
}

// Failed are the problems that stop a build.
func (e *Edit) Failed() []Problem {
	var out []Problem
	for _, p := range e.Problems {
		if !p.Warn {
			out = append(out, p)
		}
	}
	return out
}

// OutPath is where the video goes.
func (e *Edit) OutPath(dir string) string {
	if dir != "" {
		return filepath.Join(dir, filepath.Base(e.Out))
	}
	return e.Abs(e.Out)
}

// Build makes the edited video.
func Build(e *Edit, o Options) (*Result, error) {
	if o.FFmpeg == "" || o.FFprobe == "" {
		return nil, fmt.Errorf("a video edit needs ffmpeg and ffprobe")
	}
	if len(e.Failed()) > 0 {
		return nil, fmt.Errorf("%d problem(s)", len(e.Failed()))
	}
	c, err := e.Prepare(o.FFmpeg, o.FFprobe, o.Log)
	if err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp("", "si-video-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	for i, it := range e.Items { // a new take of each refilmed feature
		if it.Action == "refilm" && len(e.Failed()) == 0 {
			clip, err := e.refilm(it, o, filepath.Join(work, fmt.Sprintf("refilm%d", i)))
			if err != nil {
				return nil, fmt.Errorf("line %d: refilm %q: %v", it.Line, it.Feature, err)
			}
			it.File = clip
		}
	}
	clips := map[string]Info{}
	clipLen := func(f string) float64 {
		if in, ok := clips[f]; ok {
			return in.D
		}
		in, err := Probe(o.FFprobe, f)
		if err != nil {
			return 0
		}
		clips[f] = in
		return in.D
	}
	pl := e.Plan(c.Info.D, clipLen)
	e.quantize(pl)
	if f := e.Failed(); len(f) > 0 {
		return nil, fmt.Errorf("line %d: %s", f[0].Line, f[0].Msg)
	}
	out := e.OutPath(o.OutDir)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return nil, err
	}
	res := &Result{Out: out, Len: pl.Len}
	if e.refilmOnly() {
		ok, err := e.splice(c, pl, o, work, out)
		if err != nil {
			return nil, err
		}
		if ok {
			res.Fast = true
			return res, e.extras(c, pl, nil, o, work, res)
		}
		o.logf("  (the new take does not join the source by stream copy: re-encoding the whole video)\n")
	}
	W, H := c.Info.W&^1, c.Info.H&^1
	spans := pl.Spans(e)
	j := &Job{Src: c.Src, W: W, H: H, Audio: c.Info.Audio, Plan: pl, Spans: spans, Cards: map[*Seg]string{}, ClipAudio: map[string]bool{}, Out: out}
	for f, in := range clips {
		j.ClipAudio[f] = in.Audio
	}
	var sh *shooter
	needChrome := false
	for _, s := range spans {
		needChrome = needChrome || drawn(s.It)
	}
	for _, s := range pl.Segs {
		needChrome = needChrome || s.Kind == "card"
	}
	needChrome = needChrome || o.Captions || o.Vertical
	if needChrome {
		if o.Chrome == "" {
			return nil, fmt.Errorf("callouts and cards are drawn by Chrome: none found")
		}
		if sh, err = newShooter(o.Chrome, W, H); err != nil {
			return nil, err
		}
		defer sh.close()
	}
	for i, s := range pl.Segs {
		if s.Kind != "card" {
			continue
		}
		f := filepath.Join(work, fmt.Sprintf("card%d.png", i))
		if err := sh.page(webdemo.CardHTML(e.theme(), s.Text, ""), work, false, f); err != nil {
			return nil, err
		}
		j.Cards[s] = f
	}
	if sh != nil {
		if j.Overlay, err = e.overlay(sh, spans, pl.Len, W, H, o, work); err != nil {
			return nil, err
		}
	}
	says, err := e.says(pl, o)
	if err != nil {
		return nil, err
	}
	j.Says = says
	if meta := chapterMeta(e.Title, c.Chaps, pl); meta != "" {
		j.Meta = filepath.Join(work, "chapters.txt")
		if err := os.WriteFile(j.Meta, []byte(meta), 0o644); err != nil {
			return nil, err
		}
	}
	o.logf("  encoding %s (%s)\n", filepath.Base(out), Stamp(pl.Len))
	if err := run(o.FFmpeg, j.Args()); err != nil {
		return nil, err
	}
	return res, e.extras(c, pl, sh, o, work, res)
}

// extras: the music pass, the captions files, the variants.
func (e *Edit) extras(c *Ctx, pl *Plan, sh *shooter, o Options, work string, res *Result) error {
	out := res.Out
	for _, it := range e.Items {
		if it.Action != "music" {
			continue
		}
		tmp := filepath.Join(work, "music.mp4")
		if err := run(o.FFmpeg, webdemo.MusicArgs(out, e.Abs(it.File), it.Level, pl.Len, tmp)); err != nil {
			return err
		}
		if err := moveFile(tmp, out); err != nil {
			return err
		}
	}
	cues := e.cues(c, pl)
	stem := strings.TrimSuffix(out, filepath.Ext(out))
	if e.Captions && len(cues) > 0 {
		for ext, text := range map[string]string{".vtt": webdemo.VTT(cues), ".srt": webdemo.SRT(cues)} {
			if err := os.WriteFile(stem+ext, []byte(text), 0o644); err != nil {
				return err
			}
			res.Files = append(res.Files, stem+ext)
		}
	}
	if !o.Captions && !o.Vertical {
		sort.Strings(res.Files)
		return nil
	}
	W, H := c.Info.W&^1, c.Info.H&^1
	if sh == nil {
		var err error
		if sh, err = newShooter(o.Chrome, W, H); err != nil {
			return err
		}
		defer sh.close()
	}
	lines := webdemo.Captions(cues)
	if o.Captions {
		var ps []webdemo.Pic
		for i, q := range lines {
			f := filepath.Join(work, fmt.Sprintf("cap%03d.png", i))
			if err := sh.pageAt(webdemo.CaptionHTML(q.Text), W, H, work, f); err != nil {
				return err
			}
			ps = append(ps, webdemo.Pic{PNG: f, From: q.At, To: q.At + q.Seconds})
		}
		if err := run(o.FFmpeg, webdemo.CaptionedArgs(out, W, H, ps, stem+".captions.mp4")); err != nil {
			return err
		}
		res.Files = append(res.Files, stem+".captions.mp4")
	}
	if o.Vertical {
		var ps []webdemo.Pic
		for i, q := range lines {
			f := filepath.Join(work, fmt.Sprintf("vcap%03d.png", i))
			if err := sh.pageAt(webdemo.VerticalCaptionHTML(q.Text, W, H), webdemo.VertW, webdemo.VertH, work, f); err != nil {
				return err
			}
			ps = append(ps, webdemo.Pic{PNG: f, From: q.At, To: q.At + q.Seconds})
		}
		if err := run(o.FFmpeg, webdemo.VerticalArgs(out, W, H, "", e.theme(), ps, stem+".vertical.mp4")); err != nil {
			return err
		}
		res.Files = append(res.Files, stem+".vertical.mp4")
	}
	sort.Strings(res.Files)
	return nil
}

func moveFile(from, to string) error {
	if err := os.Rename(from, to); err == nil {
		return nil
	}
	raw, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, raw, 0o644)
}

// quantize makes every piece a whole number of frames (the pieces are cut
// at frames; the map and the joins must agree with them).
func (e *Edit) quantize(pl *Plan) {
	pl.Len = 0
	for _, s := range pl.Segs {
		s.Len = frames(s.Len)
		s.Out = pl.Len
		pl.Len += s.Len
	}
}

// cues are the captions of the output: the source's moved by the edits,
// the said lines added.
func (e *Edit) cues(c *Ctx, pl *Plan) []webdemo.Cue {
	var out []webdemo.Cue
	for _, q := range c.Cues {
		a, b := pl.Map(q.From), pl.Map(q.To)
		if b-a > 0.05 {
			out = append(out, webdemo.Cue{Text: q.Text, At: a, Seconds: b - a})
		}
	}
	for _, s := range e.said {
		out = append(out, webdemo.Cue{Text: s.Text, At: s.At, Seconds: s.Len})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

// chapterMeta is ffmetadata with the source's chapters at their output
// times ("" when none).
func chapterMeta(title string, cs []Chapter, pl *Plan) string {
	if len(cs) == 0 {
		return ""
	}
	type ch struct {
		t     string
		start float64
	}
	var ms []ch
	for _, c := range cs {
		t := pl.Map(c.Start)
		if t < pl.Len && (len(ms) == 0 || t > ms[len(ms)-1].start+0.5) {
			ms = append(ms, ch{c.Title, t})
		}
	}
	var b strings.Builder
	b.WriteString(";FFMETADATA1\n")
	if title != "" {
		b.WriteString("title=" + metaEsc(title) + "\n")
	}
	for i, m := range ms {
		end := pl.Len
		if i+1 < len(ms) {
			end = ms[i+1].start
		}
		fmt.Fprintf(&b, "[CHAPTER]\nTIMEBASE=1/1000\nSTART=%d\nEND=%d\ntitle=%s\n", int(m.start*1000), int(end*1000), metaEsc(m.t))
	}
	return b.String()
}

func metaEsc(s string) string {
	r := strings.NewReplacer("=", `\=`, ";", `\;`, "#", `\#`, `\`, `\\`, "\n", " ")
	return r.Replace(s)
}

// says voices the say lines; each at its output time.
func (e *Edit) says(pl *Plan, o Options) ([]Said, error) {
	e.said = nil
	var texts []string
	var its []*Item
	for _, it := range e.Items {
		if it.Action == "say" {
			its = append(its, it)
			texts = append(texts, webdemo.Spoken(it.Str, nil))
		}
	}
	if len(its) == 0 {
		return nil, nil
	}
	v, warn, err := e.voice(o)
	if err != nil {
		return nil, err
	}
	if warn != "" {
		o.logf("  warning: %s\n", warn)
	}
	sy := o.Synth
	if sy == nil {
		sy = &webdemo.Synth{FFmpeg: o.FFmpeg, Cache: webdemo.VoiceCache(), Log: o.Log}
	}
	lines, _, err := sy.Voice(v, texts)
	if err != nil {
		return nil, err
	}
	for i, it := range its {
		l := lines[texts[i]]
		at := pl.Map(it.Start)
		e.said = append(e.said, Said{WAV: l.WAV, At: at, Len: l.Seconds, Duck: it.Level, Text: it.Str, Line: it.Line})
	}
	return e.said, nil
}

// voice is the voice for say: --voice, voice:, si.conf voice:, else si's
// default; one this computer cannot use gives way to the system voice.
func (e *Edit) voice(o Options) (webdemo.Voice, string, error) {
	s := o.Voice
	if s == "" {
		s = e.Voice
	}
	if s == "" && o.Config != nil {
		s = o.Config.Voice
	}
	v := webdemo.DefaultVoice()
	if s != "" {
		var err error
		if v, err = webdemo.ParseVoice(s); err != nil {
			return v, "", err
		}
	}
	if o.Config != nil {
		v.Style = o.Config.VoiceStyle
	}
	if o.Synth != nil {
		return v, "", nil
	}
	if why := v.Ready(); why != "" {
		sys := webdemo.Voice{Engine: "system"}
		if v.Engine == "system" || sys.Ready() != "" {
			return v, "", fmt.Errorf("%s", why)
		}
		first, _, _ := strings.Cut(why, "\n")
		return sys, strings.TrimSuffix(first, ".") + "; speaking with the system voice", nil
	}
	return v, "", nil
}

// overlay shoots the callouts and lays them out as a 30 fps sequence the
// length of the output; "" when nothing is drawn.
func (e *Edit) overlay(sh *shooter, spans []Span, total float64, W, H int, o Options, work string) (string, error) {
	key, at := shots(spans, total)
	if len(at) == 0 {
		return "", nil
	}
	page, bad := e.OverlayPage(W, H, spans, o.Config)
	for _, b := range bad {
		o.logf("  warning: fonts: %s\n", b)
	}
	pf := filepath.Join(work, "overlay.html")
	if err := os.WriteFile(pf, []byte(page), 0o644); err != nil {
		return "", err
	}
	if err := sh.open(pf); err != nil {
		return "", err
	}
	ids := make([]string, 0, len(at))
	for id := range at {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return at[ids[i]] < at[ids[j]] })
	o.logf("  drawing %d callout picture(s)\n", len(ids))
	dir := filepath.Join(work, "ov")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	files := map[string]string{}
	for _, id := range ids {
		raw, err := sh.shoot(fmt.Sprintf("VE.at(%.4f)", at[id]))
		if err != nil {
			return "", err
		}
		f := filepath.Join(dir, "shot-"+id+".png")
		if err := os.WriteFile(f, raw, 0o644); err != nil {
			return "", err
		}
		files[id] = f
	}
	blank := filepath.Join(dir, "blank.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, W, H))); err != nil {
		return "", err
	}
	if err := os.WriteFile(blank, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	for k, id := range key {
		src := blank
		if id != "" {
			src = files[id]
		}
		dst := filepath.Join(dir, fmt.Sprintf("ov%05d.png", k))
		if err := os.Link(src, dst); err != nil {
			raw, err := os.ReadFile(src)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(dst, raw, 0o644); err != nil {
				return "", err
			}
		}
	}
	return filepath.Join(dir, "ov%05d.png"), nil
}

// shooter is headless Chrome drawing pages into PNGs.
type shooter struct {
	c    *cdp.Client
	w, h int
}

func newShooter(chrome string, w, h int) (*shooter, error) {
	c, err := cdp.Start(chrome, w, h)
	if err != nil {
		return nil, fmt.Errorf("chrome: %v", err)
	}
	if err := c.Call("Emulation.setDefaultBackgroundColorOverride", map[string]any{"color": map[string]any{"r": 0, "g": 0, "b": 0, "a": 0}}, nil); err != nil {
		c.Close()
		return nil, err
	}
	return &shooter{c: c, w: w, h: h}, nil
}

func (s *shooter) close() { s.c.Close() }

func fileURL(p string) string {
	abs, _ := filepath.Abs(p)
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	return (&url.URL{Scheme: "file", Path: abs}).String()
}

// open loads a page file and waits for it and its fonts.
func (s *shooter) open(file string) error {
	var nav struct{ ErrorText string }
	if err := s.c.Call("Page.navigate", map[string]any{"url": fileURL(file)}, &nav); err != nil {
		return err
	}
	if nav.ErrorText != "" {
		return fmt.Errorf("chrome: %s", nav.ErrorText)
	}
	end := time.Now().Add(30 * time.Second)
	for {
		v, err := s.c.Eval("document.readyState")
		if err == nil && string(v) == `"complete"` {
			break
		}
		if time.Now().After(end) {
			return fmt.Errorf("chrome: the page did not load")
		}
		time.Sleep(30 * time.Millisecond)
	}
	_, err := s.c.Eval(`document.fonts.ready.then(function(){return 1})`)
	return err
}

func (s *shooter) shoot(js string) ([]byte, error) {
	if js != "" {
		if _, err := s.c.Eval(js); err != nil {
			return nil, err
		}
	}
	return s.c.Screenshot()
}

// page draws an HTML page at the shooter's size into a PNG file.
func (s *shooter) page(html, work string, _ bool, dst string) error {
	return s.pageAt(html, s.w, s.h, work, dst)
}

// pageAt draws a page at w×h, then gives the viewport back.
func (s *shooter) pageAt(html string, w, h int, work, dst string) error {
	if w != s.w || h != s.h {
		if err := s.c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
			return err
		}
		defer s.c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": s.w, "height": s.h, "deviceScaleFactor": 1, "mobile": false}, nil)
	}
	f := filepath.Join(work, "page.html")
	if err := os.WriteFile(f, []byte(html), 0o644); err != nil {
		return err
	}
	if err := s.open(f); err != nil {
		return err
	}
	raw, err := s.shoot("")
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}
