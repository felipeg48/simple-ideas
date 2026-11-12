package webdemo

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

func TestPronounceAndVoiceFrontMatter(t *testing.T) {
	s := Parse("/x/a.md", "---\ntype: demo\napp: http://x\nvoice: openai:nova\nvoice-style: calm, friendly\npronounce: si = ess eye\npronounce: kubectl = cube control; YAML = yammel\n---\n\n## A\n\nSay: si runs kubectl.\n- go /\n")
	if len(s.Problems) > 0 {
		t.Fatal(s.Problems)
	}
	want := []Pronounce{{"si", "ess eye"}, {"kubectl", "cube control"}, {"YAML", "yammel"}}
	if len(s.Pronounce) != 3 || s.Pronounce[0] != want[0] || s.Pronounce[1] != want[1] || s.Pronounce[2] != want[2] {
		t.Errorf("pronounce: %+v", s.Pronounce)
	}
	if s.Voice != "openai:nova" || s.Style != "calm, friendly" {
		t.Errorf("voice %q style %q", s.Voice, s.Style)
	}
	if got := Spoken("Si runs si, not simple; kubectl too. YAML!", s.Pronounce); got != "ess eye runs ess eye, not simple; cube control too. yammel!" {
		t.Errorf("spoken: %q", got)
	}
	bad := Parse("/x/b.md", "---\ntype: demo\napp: http://x\nvoice: robot:x\npronounce: si\n---\n\n## A\n- go /\n")
	if all := bad.Problems[0].String() + "\n" + bad.Problems[len(bad.Problems)-1].String(); len(bad.Problems) != 2 || !strings.Contains(all, "b.md:5: pronounce") || !strings.Contains(all, "b.md:4: voice") {
		t.Errorf("problems: %v", bad.Problems)
	}
	js, err := ParseJSON("/x/c.json", []byte(`{"type":"demo","app":"http://x","voice-style":"slow","pronounce":["si = ess eye"],"features":[{"name":"A","blocks":[{"steps":["go /"]}]}]}`))
	if err != nil || len(js.Problems) > 0 || len(js.Pronounce) != 1 || js.Style != "slow" {
		t.Errorf("json: %v %+v", err, js)
	}
}

func TestParseVoice(t *testing.T) {
	for in, want := range map[string]string{"kokoro": "kokoro:af_heart", "kokoro:bf_emma": "kokoro:bf_emma", "system": "system", "System:Samantha": "system:Samantha",
		"openai": "openai:alloy", "elevenlabs:abc": "elevenlabs:abc"} {
		v, err := ParseVoice(in)
		if err != nil || v.String() != want {
			t.Errorf("%q: %v %v", in, v, err)
		}
	}
	if _, err := ParseVoice("piper:x"); err == nil {
		t.Error("piper is not an engine yet")
	}
}

func TestCacheKey(t *testing.T) {
	a := cacheKey(Voice{Engine: "kokoro", Name: "af_heart"}, "Hello.")
	if a != cacheKey(Voice{Engine: "kokoro", Name: "af_heart"}, "Hello.") || len(a) != 32 {
		t.Error("the key is not stable")
	}
	for _, v := range []Voice{{"kokoro", "af_bella", ""}, {"system", "af_heart", ""}, {"kokoro", "af_heart", "calm"}} {
		if cacheKey(v, "Hello.") == a {
			t.Errorf("%v shares a key", v)
		}
	}
	if cacheKey(Voice{Engine: "kokoro", Name: "af_heart"}, "Hello!") == a {
		t.Error("the words are not in the key")
	}
}

func TestCaptionsAndSRT(t *testing.T) {
	cs := captions("Sign in with the demo account.", 1.5, 2)
	if len(cs) != 1 || cs[0].From != 1.5 || cs[0].To != 3.5 {
		t.Fatalf("one caption: %+v", cs)
	}
	long := "Northwind keeps every order you ever placed behind one sign-in. The demo account has three of them waiting for you."
	cs = captions(long, 10, 10)
	if len(cs) != 2 || cs[0].Text != "Northwind keeps every order you ever placed behind one sign-in." || cs[1].To < 19.99 || cs[1].To > 20.01 {
		t.Fatalf("split: %+v", cs)
	}
	for _, c := range cs {
		if len(c.Text) > captionMax {
			t.Errorf("too long: %q", c.Text)
		}
		for _, l := range wrap(c.Text) {
			if len(l) > captionLine {
				t.Errorf("line too long: %q", l)
			}
		}
	}
	got := srt([]caption{{Text: "Pay sits at the bottom, under the delivery promise.", From: 9.7272, To: 3723.5}})
	want := "1\n00:00:09,727 --> 01:02:03,500\nPay sits at the bottom,\nunder the delivery promise.\n\n"
	if got != want {
		t.Errorf("srt:\n%q\nwant\n%q", got, want)
	}
}

func TestVTT(t *testing.T) {
	got := vtt([]caption{{Text: "Pay sits at the bottom, under the delivery promise.", From: 9.7272, To: 3723.5}, {Text: "Done.", From: 3724, To: 3725}})
	want := "WEBVTT\n\n00:00:09.727 --> 01:02:03.500\nPay sits at the bottom,\nunder the delivery promise.\n\n01:02:04.000 --> 01:02:05.000\nDone.\n\n"
	if got != want {
		t.Errorf("vtt:\n%q\nwant\n%q", got, want)
	}
}

// The time map: a wait beyond its first seconds plays faster, never under
// the narration; captions and chapters land on the video's clock.
func TestFastForward(t *testing.T) {
	// a 30 s wait from 10 s; the voice speaks 5-15 s; fast-forward after 3 s
	fs := fastForward([]span{{10, 40}}, []span{{5, 15}}, 3, 60)
	if len(fs) != 1 || fs[0].From != 15 || fs[0].To != 40 || fs[0].X != 13 {
		t.Fatalf("fast: %+v", fs)
	}
	if fastForward([]span{{10, 40}}, nil, 0, 60) != nil {
		t.Error("--no-fast-forward still fast-forwards")
	}
	if fs := fastForward([]span{{10, 14}, {20, 24.5}}, nil, 3, 60); len(fs) != 0 {
		t.Errorf("a short wait fast-forwarded: %+v", fs)
	}
	// the voice in the middle of a wait splits it in two
	if fs := fastForward([]span{{0, 40}}, []span{{12, 20}}, 2.5, 30); len(fs) != 2 || fs[0] != (fast{2.5, 12, 5}) || fs[1] != (fast{20, 30, 5}) {
		t.Errorf("split: %+v", fs)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	saved := 25 * (1 - 1.0/13)
	for _, c := range [][2]float64{{0, 0}, {15, 15}, {40, 40 - saved}, {60, 60 - saved}, {27.5, 15 + 12.5/13}} {
		if o := outT(fs, c[0]); !near(o, c[1]) {
			t.Errorf("outT(%g) = %g, want %g", c[0], o, c[1])
		}
		if i := inT(fs, c[1]); !near(i, c[0]) {
			t.Errorf("inT(%g) = %g, want %g", c[1], i, c[0])
		}
	}
	if l := outT(fs, 40) - outT(fs, 15); l > ffMost {
		t.Errorf("the fast stretch lasts %gs", l)
	}
	// the sequence: frames at 0 (prompt) and 39.5 s (the output); the video is shorter and shows the output at its end
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.jpg"), filepath.Join(dir, "b.jpg")
	os.WriteFile(a, []byte("a"), 0o644)
	os.WriteFile(b, []byte("b"), 0o644)
	_, n, err := sequence([]frame{{a, 0}, {b, 39.5}}, 41, fs, dir, nil, 0, 0)
	if err != nil || n != int(outT(fs, 41)*fps+.5) {
		t.Fatalf("sequence: %d %v", n, err)
	}
	for k, want := range map[int]string{0: "a", int(15*fps + 5): "a", n - 1: "b"} {
		if raw, _ := os.ReadFile(filepath.Join(dir, fmt.Sprintf("seq%05d.jpg", k))); string(raw) != want {
			t.Errorf("frame %d shows %q, want %q", k, raw, want)
		}
	}
	// captions after the wait move up by what it saved; the chapters follow the clips' new lengths
	caps := []caption{{"Before.", 6, 9}, {"After.", 45, 48}}
	for i := range caps {
		caps[i].From, caps[i].To = outT(fs, caps[i].From), outT(fs, caps[i].To)
	}
	if !near(caps[0].From, 6) || !near(caps[1].From, 45-saved) || !near(caps[1].To-caps[1].From, 3) {
		t.Errorf("captions: %+v", caps)
	}
	cs := chapters([]part{{Name: "A", Seconds: outT(fs, 60)}, {Name: "B", Seconds: 10}})
	all := joinCaptions(cs, [][]caption{caps, {{"B.", 1, 2}}})
	if len(all) != 3 || !near(all[1].From, cardSeconds+45-saved) || all[2].From < cs[1].Start+cardSeconds+1-1e-9 || all[2].From > cs[1].Start+cardSeconds+1+1e-9 {
		t.Errorf("all captions: %+v chapters %+v", all, cs)
	}
}

func TestChapters(t *testing.T) {
	parts := []part{{Name: "Sign in", Seconds: 6.3333}, {Name: "Pay; then #1 = done", Seconds: 57.8}, {Name: "Find", Seconds: 10}}
	cs := chapters(parts)
	if cs[1].Start < 9.33 || cs[1].Start > 9.34 || cs[2].Start < 70.13 || cs[2].Start > 70.14 {
		t.Errorf("chapters: %+v", cs)
	}
	if y := youtube(cs); y != "0:00 Sign in\n0:09 Pay; then #1 = done\n1:10 Find\n" {
		t.Errorf("chapters.txt: %q", y)
	}
	if y := youtube([]chapter{{"Late", 3725, 3800}}); y != "1:02:05 Late\n" {
		t.Errorf("an hour in: %q", y)
	}
	m := ffmetadata("Shop", cs)
	if !strings.HasPrefix(m, ";FFMETADATA1\ntitle=Shop\n") || !strings.Contains(m, "[CHAPTER]\nTIMEBASE=1/1000\nSTART=9333\nEND=70133\ntitle=Pay\\; then \\#1 \\= done\n") {
		t.Errorf("ffmetadata:\n%s", m)
	}
}

func TestFFmpegArgs(t *testing.T) {
	a := strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, nil, nil, nil, "", "/o/a.video.mp4"), " ")
	if !strings.HasPrefix(a, "-nostdin -y") || !strings.Contains(a, "-framerate 30 -i /t/seq%05d.jpg -vf scale=") || !strings.HasSuffix(a, "+faststart /o/a.video.mp4") {
		t.Errorf("video: %s", a)
	}
	badges := []pic{{"/t/ff6.png", 4, 6}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, nil, badges, nil, "", "/o/a.video.mp4"), " ")
	if !strings.Contains(a, "-i /t/ff6.png -filter_complex [0:v][1:v]overlay=0:0:enable='between(t,4.000,6.000)'[o1];[o1]scale=") || strings.Contains(a, "pad=") || strings.Contains(a, "aac") {
		t.Errorf("badges: %s", a)
	}
	caps := []pic{{"/t/c0.png", 1, 2.5}, {"/t/c1.png", 5, 6}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, nil, badges, caps, "/o/a.voice.wav", "/o/a.captions.mp4"), " ")
	for _, w := range []string{"-i /t/ff6.png -i /t/c0.png -i /t/c1.png -i /o/a.voice.wav -filter_complex",
		// the badge over the picture, the picture shrunk into the top (1222x764 at 109,18 of 1440x900), the captions in the band
		"[0:v][1:v]overlay=0:0:enable='between(t,4.000,6.000)'[o1];[o1]scale=1222:764,pad=1440:900:109:18:color=0x111318[band];" +
			"[band][2:v]overlay=0:0:enable='between(t,1.000,2.500)'[o2];[o2][3:v]overlay=0:0:enable='between(t,5.000,6.000)'[o3];[o3]scale=",
		"-map [v] -map 4:a -c:a aac", "-shortest", "/o/a.captions.mp4"} {
		if !strings.Contains(a, w) {
			t.Errorf("captions: %s\nmissing %s", a, w)
		}
	}
	if strings.Contains(a, "filter_complex_script") || strings.Contains(a, "drawtext") || strings.Contains(a, "subtitles") {
		t.Error("ffmpeg 9 has no -filter_complex_script; Homebrew's has no drawtext or subtitles")
	}
	// the band leaves the bottom 13% for the captions, whatever the frame
	for _, wh := range [][2]int{{1440, 900}, {400, 800}, {820, 1180}, {1280, 720}} {
		var sw, sh, x, y, W, H int
		fmt.Sscanf(band(wh[0], wh[1], wh[0], wh[1]), "scale=%d:%d,pad=%d:%d:%d:%d", &sw, &sh, &W, &H, &x, &y)
		if W != wh[0] || H != wh[1] || sw > W || x*2+sw != W || float64(H-y-sh) < .12*float64(H) || sh%2 != 0 || sw%2 != 0 {
			t.Errorf("band %v: %s", wh, band(wh[0], wh[1], wh[0], wh[1]))
		}
	}
	a = strings.Join(muxArgs("v.mp4", "v.wav", "o.mp4"), " ")
	if !strings.Contains(a, "-i v.mp4 -i v.wav -map 0:v -map 1:a -c:v copy -c:a aac") || !strings.Contains(a, "-shortest") {
		t.Errorf("mux: %s", a)
	}
	a = strings.Join(allArgs([]part{{Name: "A", Card: "a.png", Clip: "a.mp4", Seconds: 2}, {Name: "B", Card: "b.png", Clip: "b.mp4", Seconds: 3}}, 1440, 900, Transition{Kind: "cut"}, "m.txt", "all.mp4"), " ")
	for _, w := range []string{"-loop 1 -framerate 30 -t 3 -i a.png -f lavfi -t 3 -i anullsrc=r=48000:cl=stereo -i a.mp4",
		"-i b.mp4 -i m.txt -filter_complex", "[cv0][ca0][fv0][fa0][cv1][ca1][fv1][fa1]concat=n=4:v=1:a=1[v][a]", "-map_metadata 6 -map_chapters 6", "-c:v libx264"} {
		if !strings.Contains(a, w) {
			t.Errorf("all: %s\nmissing %s", a, w)
		}
	}
	if !strings.HasPrefix(strings.Join(normArgs("in", "out.wav"), " "), "-nostdin") {
		t.Error("normArgs without -nostdin")
	}
}

func TestSystemVoiceArgs(t *testing.T) {
	a, _ := systemArgs("darwin", "Samantha", "t.txt", "o.aiff")
	if strings.Join(a, " ") != "say -v Samantha -o o.aiff -f t.txt" {
		t.Errorf("mac: %v", a)
	}
	a, _ = systemArgs("linux", "", "t.txt", "o.wav")
	if strings.Join(a, " ") != "espeak-ng -f t.txt -w o.wav" {
		t.Errorf("linux: %v", a)
	}
	a, env := systemArgs("windows", "Microsoft Zira Desktop", "t.txt", "o.wav")
	if a[0] != "powershell" || !strings.Contains(a[len(a)-1], "SetOutputToWaveFile") || strings.Join(env, "|") != "SI_SAY_VOICE=Microsoft Zira Desktop|SI_SAY_OUT=o.wav|SI_SAY_TEXT=t.txt" {
		t.Errorf("windows: %v %v", a, env)
	}
}

// The cloud voices, against fakes: what is sent (only the Say: text) and
// what comes back.
func TestCloudRequests(t *testing.T) {
	var got map[string]any
	var hdr http.Header
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr, path = r.Header, r.URL.String()
		b, _ := io.ReadAll(r.Body)
		got = nil
		json.Unmarshal(b, &got)
		if r.Header.Get("xi-api-key") == "bad" {
			http.Error(w, `{"detail":"invalid key"}`, 401)
			return
		}
		w.Write([]byte{1, 0, 2, 0})
	}))
	defer srv.Close()
	defer func(e, o string) { elevenURL, openaiURL = e, o }(elevenURL, openaiURL)
	elevenURL, openaiURL = srv.URL, srv.URL

	req, _ := elevenRequest(Voice{Engine: "elevenlabs", Name: "v123"}, "Pay now.", "k1")
	body, err := fetch(srv.Client(), req)
	if err != nil || len(body) != 4 || path != "/v1/text-to-speech/v123?output_format=pcm_24000" || hdr.Get("xi-api-key") != "k1" || got["text"] != "Pay now." || len(got) != 2 {
		t.Errorf("elevenlabs: %v %s %v %v", err, path, hdr, got)
	}
	req, _ = elevenRequest(Voice{Engine: "elevenlabs", Name: "v123"}, "x", "bad")
	if _, err := fetch(srv.Client(), req); err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid key") {
		t.Errorf("a refused key: %v", err)
	}
	req, _ = openaiRequest(Voice{Engine: "openai", Name: "nova", Style: "calm"}, "Pay now.", "k2")
	if _, err := fetch(srv.Client(), req); err != nil || path != "/v1/audio/speech" || hdr.Get("Authorization") != "Bearer k2" ||
		got["model"] != "gpt-4o-mini-tts" || got["voice"] != "nova" || got["input"] != "Pay now." || got["instructions"] != "calm" || got["response_format"] != "wav" {
		t.Errorf("openai: %v %v %v", err, hdr, got)
	}
	req, _ = openaiRequest(Voice{Engine: "openai", Name: "alloy"}, "x", "k")
	fetch(srv.Client(), req)
	if _, ok := got["instructions"]; ok {
		t.Error("instructions sent without a voice-style")
	}
}

func TestMixWAV(t *testing.T) {
	dir := t.TempDir()
	one := filepath.Join(dir, "one.wav")
	pcm := make([]byte, 2*2400) // 0.1 s of 1000s
	for i := 0; i < len(pcm); i += 2 {
		pcm[i], pcm[i+1] = 0xe8, 0x03
	}
	os.WriteFile(one, pcmWAV(pcm, 24000), 0o644)
	if s, err := wavSeconds(one); err != nil || s != 0.1 {
		t.Fatalf("seconds: %v %v", s, err)
	}
	out := filepath.Join(dir, "mix.wav")
	if err := mixWAV(out, 1, []placed{{one, 0.5}, {one, 0.55}, {one, 0.95}}); err != nil {
		t.Fatal(err)
	}
	s, rate, err := readWAV(out)
	if err != nil || rate != 24000 || len(s) != 24000 {
		t.Fatalf("mix: %d %d %v", len(s), rate, err)
	}
	if s[0] != 0 || s[12000] != 1000 || s[13500] != 2000 || s[23999] != 1000 {
		t.Errorf("samples: %d %d %d %d", s[0], s[12000], s[13500], s[23999])
	}
}

func TestReadme(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "shop.md")
	os.WriteFile(script, []byte("---\ntype: demo\n---\n## Pay\n"), 0o644)
	files := []string{"shop.mp4", "shop.silent.mp4", "shop.captions.mp4", "shop.vtt", "shop.txt", "chapters.txt", "pay.mp4", "pay.captions.mp4", "pay.srt", "pay.vtt", "report.html", "README.md", "README.html"}
	rep := &Report{Script: "shop.md", Title: "Shop", Voice: "kokoro:af_heart", Passed: 1, Video: "shop.mp4", Files: files,
		Features: []FeatureResult{{Name: "Pay", What: "pays", Status: "passed", Seconds: 3, Video: "pay.mp4"}}}
	md := Readme(rep, &Script{Path: script, Name: "shop"}, "si demo run shop.md")
	for _, w := range []string{"type: article", "| [shop.mp4](shop.mp4) | every feature in one video", "| [pay.srt](pay.srt) | the feature's captions",
		"| [shop.silent.mp4](shop.silent.mp4) | every feature, the picture alone", "| [shop.captions.mp4](shop.captions.mp4) | every feature with the narration and the captions",
		"| [shop.txt](shop.txt) | the transcript", "| [pay.captions.mp4](pay.captions.mp4) | the feature: picture, narration and the captions in a band", "| [pay.vtt](pay.vtt) | the same captions as WebVTT",
		`<video controls preload="metadata" src="shop.mp4" style="width:100%;height:auto"><track kind="captions" src="shop.vtt" srclang="en" label="Captions" default></video>`,
		`<video controls preload="metadata" src="pay.mp4" style="width:100%;height:auto"><track kind="captions" src="pay.vtt" srclang="en" label="Captions" default></video>`,
		"[pay.mp4](pay.mp4) · [with captions](pay.captions.mp4) · [captions](pay.srt)\n", "~~~~markdown\n---\ntype: demo", "- Voice: kokoro:af_heart", "`si demo run shop.md`"} {
		if !strings.Contains(md, w) {
			t.Errorf("README.md lacks %q:\n%s", w, md)
		}
	}
	// only what was made: no picture-only clip, no narration file, no report.json
	for _, w := range []string{"pay.silent.mp4", "pay.m4a", "report.json", "shop.m4a"} {
		if strings.Contains(md, w) {
			t.Errorf("README links %s, which was not made", w)
		}
	}
}

// A take end to end: needs Chrome and ffmpeg (skipped without them). No Say:
// lines, so no voice engine is needed.
func TestRunFilmsAFeature(t *testing.T) {
	chrome, ffmpeg := tools.Find("chrome"), tools.Find("ffmpeg")
	if chrome == "" || ffmpeg == "" || testing.Short() {
		t.Skip("needs Chrome and ffmpeg")
	}
	app, _ := filepath.Abs("../../examples/webdemo/app")
	dir := t.TempDir()
	script := filepath.Join(dir, "film.md")
	os.WriteFile(script, []byte("---\ntype: demo\ntitle: Film\napp: "+app+"\nviewport: 640x480\n---\n\n## Sign in\n\nWhat it does: signs in.\n\n- note \"Use demo\" at \"Username\" · for 1s\n- type \"demo\" into \"Username\"\n- click \"Sign in\"\n- expect text \"nothing like this\"\n- click \"never reached\"\n"), 0o644)
	s, err := ParseFile(script)
	if err != nil || len(s.Problems) > 0 {
		t.Fatal(err, s.Problems)
	}
	out := filepath.Join(dir, "out")
	os.MkdirAll(out, 0o755)
	rep, err := Run(s, Options{Chrome: chrome, FFmpeg: ffmpeg, Out: out, Video: true, Voice: Voice{Engine: "system"}, Cache: filepath.Join(dir, "cache"),
		With: With{Clips: true, Chapters: true, Subtitles: true, Report: true}})
	if err != nil {
		t.Fatal(err)
	}
	f := rep.Features[0]
	if f.Status != "failed" || f.Video != "sign-in.mp4" || f.Silent != "" || f.Steps[4].Status != "skipped" {
		t.Errorf("feature: %+v", f)
	}
	// no narration: no subtitles; the work folder is gone
	want := []string{"chapters.txt", "film.mp4", "report.html", "report.json", "shots", "sign-in.mp4"}
	ents, _ := os.ReadDir(out)
	var got []string
	for _, e := range ents {
		got = append(got, e.Name())
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("out: %v, want %v", got, want)
	}
	if raw, _ := os.ReadFile(filepath.Join(out, "chapters.txt")); string(raw) != "0:00 Sign in\n" {
		t.Errorf("chapters.txt: %q", raw)
	}
}
