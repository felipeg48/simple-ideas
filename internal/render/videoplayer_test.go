package render

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestVideoPlayer(t *testing.T) {
	page := func(body string) string { return "---\ntype: article\n---\n# V\n\n" + body }

	// inline: a normal <video controls> with a poster, no dialog
	res, _ := renderFiles(t, map[string]string{"i.md": page("```video src=\"https://x/clip.mp4\" poster=\"https://x/p.jpg\" mode=inline title=\"Demo\"\nA caption.\n```\n")}, "i.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("inline: %s", problemsText(res))
	}
	if strings.Count(h, `<figure class="si-vp"`) != 1 || !strings.Contains(h, `data-vp-mode="inline"`) {
		t.Errorf("inline: one figure expected")
	}
	if !strings.Contains(h, `<video class="si-vp-video" controls`) {
		t.Errorf("inline: a <video controls> is expected")
	}
	if strings.Contains(h, "<dialog") {
		t.Errorf("inline: no dialog expected")
	}
	if !strings.Contains(h, `<figcaption class="si-vp-cap">A caption.</figcaption>`) {
		t.Errorf("inline: caption missing")
	}
	if !strings.Contains(h, "si video player:") {
		t.Errorf("inline: the script should be inlined")
	}

	// modal: a poster button and a hidden <video> the shared dialog plays from
	res, _ = renderFiles(t, map[string]string{"m.md": page("```video src=\"https://x/clip.mp4\" poster=\"https://x/p.jpg\" mode=modal title=\"Demo\"\n```\n")}, "m.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("modal: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-mode="modal"`) || !strings.Contains(h, `<video class="si-vp-video" hidden preload="none" playsinline poster="" src="https://x/clip.mp4" data-vp-title="Demo"`) {
		t.Errorf("modal: a hidden <video> is expected")
	}
	if !strings.Contains(h, `class="si-vp-open"`) || !strings.Contains(h, `<img class="si-vp-poster"`) {
		t.Errorf("modal: a clickable poster is expected")
	}
	if !strings.Contains(h, "si video player:") {
		t.Errorf("modal: the script should be inlined")
	}

	// facade: a poster only, the <video> built on click; no eager <video> in the output
	res, _ = renderFiles(t, map[string]string{
		"f.md":     page("```video src=\"clip.mp4\" poster=\"p.jpg\" mode=facade title=\"Demo\"\n```\n"),
		"clip.mp4": "fake mp4 bytes",
		"p.jpg":    "fake jpg bytes",
	}, "f.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("facade: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-mode="facade"`) || !strings.Contains(h, `data-vp-src="clip.mp4"`) {
		t.Errorf("facade: the src should be on a data attribute, resolved")
	}
	if strings.Contains(h, `playsinline poster="p.jpg" src=`) || !strings.Contains(h, `preload="none"`) {
		t.Errorf("facade: nothing loads before the first play")
	}

	// degrade: only a poster, no src — poster shown, play affordance disabled, no error
	res, _ = renderFiles(t, map[string]string{"d.md": page("```video poster=\"https://x/p.jpg\" mode=inline title=\"Demo\"\n```\n")}, "d.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("degrade: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-disabled`) || strings.Contains(h, `<video class="si-vp-video"`) {
		t.Errorf("degrade: a poster-only block must not emit a <video>")
	}
	if !strings.Contains(h, `<img class="si-vp-poster"`) {
		t.Errorf("degrade: the poster should still show")
	}

	// an unknown mode is a problem (and falls back to inline)
	res, _ = renderFiles(t, map[string]string{"b.md": page("```video src=\"https://x/clip.mp4\" mode=carousel\n```\n")}, "b.md")
	if !strings.Contains(problemsText(res), `video mode "carousel"`) {
		t.Errorf("unknown mode should be a problem: %s", problemsText(res))
	}

	// a plain fence stays plain; no video player scripts
	res, _ = renderFiles(t, map[string]string{"p.md": page("```go\na()\n```\n")}, "p.md")
	if h := string(res.HTML); strings.Contains(h, `class="si-vp`) || strings.Contains(h, "si video player:") {
		t.Errorf("a plain fence must not become a video player")
	}
}

func TestVideoPlayerOptions(t *testing.T) {
	page := func(body string) string { return "---\ntype: article\nlang: es\n---\n# V\n\n" + body }
	vtt := "WEBVTT\n\n00:00.000 --> 00:02.000\nHola\n"
	srt := "1\n00:00:00,000 --> 00:00:02,000\nHola\n"
	chapters := "0:00 Intro\n0:12 - The cluster\n1:05 The end\n"

	// sidecars found beside the video: <video>.vtt and <video>.chapters.txt
	res, _ := renderFiles(t, map[string]string{
		"a.md":              page("```video src=\"clip.mp4\" group=tour start=0:05 hud=on\n```\n"),
		"clip.mp4":          "x",
		"clip.vtt":          vtt,
		"clip.chapters.txt": chapters,
	}, "a.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("sidecars: %s", problemsText(res))
	}
	for _, want := range []string{
		`<track kind="captions" src="data:text/vtt;base64,`,
		`srclang="es" label="Captions">`,
		`data-vp-chapters="[[0,&#34;Intro&#34;],[12,&#34;The cluster&#34;],[65,&#34;The end&#34;]]"`,
		`data-vp-start="5"`,
		`<video class="si-vp-video" controls preload="metadata"`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("sidecars: page lacks %s", want)
		}
	}
	if strings.Contains(h, `data-vp-group`) {
		t.Error("group= is for modal players only")
	}

	// named files win; a .srt becomes WebVTT; WebVTT chapters; group and hud on the figure
	res, _ = renderFiles(t, map[string]string{
		"b.md":        page("```video src=\"clip.mp4\" mode=modal group=tour hud=native captions=subs/es.srt chapters=ch.vtt\n```\n"),
		"clip.mp4":    "x",
		"subs/es.srt": srt,
		"ch.vtt":      "WEBVTT\n\n00:00:00.000 --> 00:00:30.000\nOne\n\n00:00:30.000 --> 00:01:00.000\nTwo\n",
	}, "b.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("named: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-group="tour"`) || !strings.Contains(h, `data-vp-hud="native"`) {
		t.Error("named: group= and hud= go on the figure")
	}
	if !strings.Contains(h, `data-vp-chapters="[[0,&#34;One&#34;],[30,&#34;Two&#34;]]"`) {
		t.Error("named: WebVTT chapters")
	}
	if !strings.Contains(h, "data:text/vtt;base64,"+base64.StdEncoding.EncodeToString([]byte("WEBVTT\n\n1\n00:00:00.000 --> 00:00:02.000\nHola\n"))) {
		t.Error("named: a .srt is carried as WebVTT")
	}

	// si demo run's chapters.txt beside a lone video; hud=off has no controls
	res, _ = renderFiles(t, map[string]string{
		"c.md":              page("```video src=\"film/tour.mp4\" hud=off loop autoplay\n```\n"),
		"film/tour.mp4":     "x",
		"film/chapters.txt": chapters,
	}, "c.md")
	h = string(res.HTML)
	if !strings.Contains(h, `data-vp-chapters="[[0,`) || !strings.Contains(h, `data-vp-hud="off"`) || strings.Contains(h, `<video class="si-vp-video" controls`) {
		t.Error("chapters.txt beside a lone video, hud=off")
	}

	// problems: an unknown option, a bad hud, a bad start, a missing sidecar
	res, _ = renderFiles(t, map[string]string{
		"d.md":     page("```video src=\"clip.mp4\" colour=red hud=fancy start=soon captions=nope.vtt\n```\n"),
		"clip.mp4": "x",
	}, "d.md")
	p := problemsText(res)
	for _, want := range []string{`video option "colour"`, `video hud "fancy"`, `video start "soon"`, `missing file: nope.vtt`} {
		if !strings.Contains(p, want) {
			t.Errorf("problems lack %s: %s", want, p)
		}
	}

	// ![](x.mp4) in a document plays in the same player, with its sidecars
	res, _ = renderFiles(t, map[string]string{
		"e.md":     page("![The tour](clip.mp4)\n"),
		"clip.mp4": "x",
		"clip.vtt": vtt,
	}, "e.md")
	h = string(res.HTML)
	if !strings.Contains(h, `<video data-si-player controls preload="metadata" playsinline src="clip.mp4"`) || !strings.Contains(h, `<track kind="captions" src="data:text/vtt;base64,`) || !strings.Contains(h, "si video player:") {
		t.Error("a document video is an si player with its captions")
	}
}

func TestParseChapters(t *testing.T) {
	got := parseChapters("(0:00) Intro\n1:02:03 Long — part\nnot a line\n12.5 Half\n")
	if len(got) != 3 || got[1][0] != 3723.0 || got[1][1] != "Long — part" || got[2][0] != 12.5 {
		t.Errorf("chapters: %v", got)
	}
}
