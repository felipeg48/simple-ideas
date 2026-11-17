package videdit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// ScriptPath is the demo script a refilm runs: script:, else the source's
// sibling <stem>.md (or .adoc, .json) with type: demo, else the one demo
// script beside the source.
func (e *Edit) ScriptPath() (string, error) {
	if e.Script != "" {
		p := e.Abs(e.Script)
		if !fileExists(p) {
			return "", fmt.Errorf("script: %s not found", e.Script)
		}
		return p, nil
	}
	if len(e.Sources) == 0 {
		return "", fmt.Errorf("no source")
	}
	src := e.Abs(e.Sources[0].File)
	stem := strings.TrimSuffix(src, filepath.Ext(src))
	isDemo := func(p string) bool {
		if p == e.Path || !fileExists(p) {
			return false
		}
		s, err := webdemo.ParseFile(p)
		return err == nil && s != nil && len(s.Features) > 0
	}
	for _, ext := range []string{".md", ".adoc", ".json"} {
		if isDemo(stem + ext) {
			return stem + ext, nil
		}
	}
	var found []string
	ms, _ := filepath.Glob(filepath.Join(filepath.Dir(src), "*.md"))
	for _, m := range ms {
		if raw, err := os.ReadFile(m); err == nil && strings.Contains(string(raw), "type: demo") && isDemo(m) {
			found = append(found, m)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if len(found) > 1 {
		return "", fmt.Errorf("several demo scripts beside the source: name one with script:")
	}
	return "", fmt.Errorf("no demo script for the source (name it %s.md, or set script:)", filepath.Base(stem))
}

// refilm runs one feature of the demo script into dir and returns its clip.
func (e *Edit) refilm(it *Item, o Options, dir string) (string, error) {
	sp, err := e.ScriptPath()
	if err != nil {
		return "", err
	}
	s, err := webdemo.ParseFile(sp)
	if err != nil {
		return "", err
	}
	if len(s.Problems) > 0 {
		return "", fmt.Errorf("%s: %s", filepath.Base(sp), s.Problems[0].Msg)
	}
	if o.Chrome == "" {
		return "", fmt.Errorf("refilm needs Chrome")
	}
	v := webdemo.DefaultVoice()
	for _, x := range []string{o.Voice, s.Voice} {
		if x != "" {
			if v, err = webdemo.ParseVoice(x); err != nil {
				return "", err
			}
			break
		}
	}
	if s.Voice == "" && o.Voice == "" && o.Config != nil && o.Config.Voice != "" {
		if v, err = webdemo.ParseVoice(o.Config.Voice); err != nil {
			return "", err
		}
	}
	o.logf("  refilming %q from %s\n", it.Feature, filepath.Base(sp))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	rep, err := webdemo.Run(s, webdemo.Options{Chrome: o.Chrome, Feature: it.Feature, Out: dir, Log: o.Log, SI: o.SI, Video: true,
		Voice: v, FFmpeg: o.FFmpeg, With: webdemo.With{Clips: true}})
	if err != nil {
		return "", err
	}
	if rep.Failed > 0 {
		return "", fmt.Errorf("the feature failed: see %s", dir)
	}
	ms, _ := filepath.Glob(filepath.Join(dir, "*.mp4"))
	for _, m := range ms {
		b := filepath.Base(m)
		if b != s.Name+".mp4" && !strings.HasPrefix(b, s.Name+".") {
			return m, nil
		}
	}
	for _, m := range ms {
		if filepath.Base(m) == s.Name+".mp4" {
			return m, nil
		}
	}
	return "", fmt.Errorf("the run made no clip")
}

// refilmOnly reports whether every line is a refilm (or captions on): the
// rest of the video can then be kept as it is.
func (e *Edit) refilmOnly() bool {
	n := 0
	for _, it := range e.Items {
		switch it.Action {
		case "refilm":
			n++
		case "captions":
		default:
			return false
		}
	}
	return n == 1 && len(e.Sources) == 1 && e.Sources[0].From == 0 && e.Sources[0].To == 0
}

// splice puts the new take over its range keeping the rest of the source
// byte for byte: stream copy up to the last keyframe before the range (less
// the fade) and from the first keyframe after it, the stretch between
// re-encoded; the parts joined by the concat demuxer. false (no error) when
// the result does not decode cleanly: the caller re-encodes.
func (e *Edit) splice(c *Ctx, pl *Plan, o Options, work, out string) (bool, error) {
	var seg *Seg
	for _, s := range pl.Segs {
		if s.Item != nil && s.Item.Action == "refilm" {
			seg = s
		}
	}
	if seg == nil {
		return false, nil
	}
	ks, err := Keyframes(o.FFprobe, c.Src)
	if err != nil || len(ks) == 0 {
		return false, nil
	}
	fade := 0.3
	if seg.Item.Trans != nil {
		fade = seg.Item.Trans.Dur
	}
	D := c.Info.D
	var k1 Keyframe
	k2 := Keyframe{D, D}
	for _, k := range ks {
		if k.PTS <= seg.A-fade-0.05 {
			k1 = k
		}
	}
	for i := len(ks) - 1; i >= 0; i-- {
		if ks[i].PTS >= seg.B+fade+0.05 {
			k2 = ks[i]
		}
	}
	tr := &Trans{"fade", fade}
	if seg.Item.Trans != nil {
		tr = seg.Item.Trans
	}
	mid := &Plan{D: D}
	add := func(s *Seg) {
		if s.Len <= 0 {
			return
		}
		s.Out = mid.Len
		mid.Segs = append(mid.Segs, s)
		mid.Len += s.Len
	}
	add(&Seg{Kind: "src", A: k1.PTS, B: seg.A, Speed: 1, Len: (seg.A - k1.PTS)})
	clip := &Seg{Kind: "clip", A: seg.A, B: seg.B, File: seg.File, Len: frames(seg.Len), Trans: tr}
	add(clip)
	after := &Seg{Kind: "src", A: seg.B, B: k2.PTS, Speed: 1, Len: (k2.PTS - seg.B), Trans: tr}
	add(after)
	if len(mid.Segs) > 0 && mid.Segs[0] != clip {
		mid.Segs[0].Len = frames(mid.Segs[0].Len)
	}
	mid.Len = 0
	for _, s := range mid.Segs {
		s.Out = mid.Len
		mid.Len += s.Len
	}
	mid.Segs[0].Trans = nil
	in, _ := Probe(o.FFprobe, seg.File)
	middle := filepath.Join(work, "middle.mp4")
	j := &Job{Src: c.Src, W: c.Info.W &^ 1, H: c.Info.H &^ 1, Audio: c.Info.Audio, Plan: mid, ClipAudio: map[string]bool{seg.File: in.Audio}, Out: middle}
	if err := run(o.FFmpeg, j.Args()); err != nil {
		return false, err
	}
	// the source's own packets around the new stretch: up to the keyframe
	// before it (cut at its decode time, so no frame of the next group comes
	// along; duration: its display time) and from the keyframe after it
	src := strings.ReplaceAll(filepath.ToSlash(c.Src), "'", `'\''`)
	var list strings.Builder
	if k1.PTS > 0 {
		fmt.Fprintf(&list, "file '%s'\noutpoint %s\nduration %s\n", src, f3(k1.DTS), f3(k1.PTS))
	}
	fmt.Fprintf(&list, "file '%s'\nduration %s\n", filepath.ToSlash(middle), f3(mid.Len))
	if k2.PTS < D-0.01 {
		fmt.Fprintf(&list, "file '%s'\ninpoint %s\n", src, f3(k2.PTS))
	}
	lf := filepath.Join(work, "parts.txt")
	if err := os.WriteFile(lf, []byte(list.String()), 0o644); err != nil {
		return false, err
	}
	joined := filepath.Join(work, "joined.mp4")
	a := append(append([]string(nil), webdemo.FFHead...), "-f", "concat", "-safe", "0", "-i", lf, "-c", "copy", "-movflags", "+faststart", joined)
	if err := run(o.FFmpeg, a); err != nil {
		o.logf("  (splice: %v)\n", err)
		return false, nil
	}
	chk, _ := exec.Command(o.FFmpeg, "-v", "error", "-i", joined, "-f", "null", "-").CombinedOutput()
	if msg := strings.TrimSpace(string(chk)); msg != "" {
		o.logf("  (splice: %s)\n", firstLine(msg))
		return false, nil
	}
	pl.Len = k1.PTS + mid.Len + (D - k2.PTS)
	return true, moveFile(joined, out)
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}
