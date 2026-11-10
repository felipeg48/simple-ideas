package webdemo

import (
	"encoding/json"
	"io"
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
	a := strings.Join(videoArgs("/t/seq%05d.jpg", nil, "/o/a.video.mp4"), " ")
	if !strings.HasPrefix(a, "-nostdin -y") || !strings.Contains(a, "-framerate 30 -i /t/seq%05d.jpg -vf scale=") || !strings.HasSuffix(a, "+faststart /o/a.video.mp4") {
		t.Errorf("video: %s", a)
	}
	caps := []caption{{Text: "x", From: 1, To: 2.5, PNG: "/t/c0.png"}, {Text: "y", From: 3, To: 4}, {Text: "z", From: 5, To: 6, PNG: "/t/c2.png"}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", caps, "/o/a.video.mp4"), " ")
	for _, w := range []string{"-i /t/c0.png -i /t/c2.png -filter_complex",
		"[0:v][1:v]overlay=0:0:enable='between(t,1.000,2.500)'[c1];[c1][2:v]overlay=0:0:enable='between(t,5.000,6.000)'[c2];[c2]scale=",
		"-map [v]"} {
		if !strings.Contains(a, w) {
			t.Errorf("captions: %s\nmissing %s", a, w)
		}
	}
	if strings.Contains(a, "filter_complex_script") {
		t.Error("ffmpeg 9 has no -filter_complex_script")
	}
	a = strings.Join(muxArgs("v.mp4", "v.wav", "o.mp4"), " ")
	if !strings.Contains(a, "-i v.mp4 -i v.wav -map 0:v -map 1:a -c:v copy -c:a aac") || !strings.Contains(a, "-shortest") {
		t.Errorf("mux: %s", a)
	}
	a = strings.Join(allArgs([]part{{Name: "A", Card: "a.png", Clip: "a.mp4", Seconds: 2}, {Name: "B", Card: "b.png", Clip: "b.mp4", Seconds: 3}}, "m.txt", "all.mp4"), " ")
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
	for _, f := range []string{"pay.mp4", "pay.video.mp4", "pay.srt", "pay.voice.m4a", "all.mp4", "chapters.txt", "report.html", ".card-pay.png"} {
		os.WriteFile(filepath.Join(dir, f), nil, 0o644)
	}
	script := filepath.Join(dir, "shop.md")
	os.WriteFile(script, []byte("---\ntype: demo\n---\n## Pay\n"), 0o644)
	rep := &Report{Script: "shop.md", Title: "Shop", Voice: "kokoro:af_heart", Passed: 1, Features: []FeatureResult{{Name: "Pay", What: "pays", Status: "passed", Seconds: 3, Video: "pay.mp4"}}}
	md := Readme(rep, &Script{Path: script}, dir, "si demo run shop.md")
	for _, w := range []string{"type: article", "| [all.mp4](all.mp4) | every feature", "| [pay.srt](pay.srt) | captions", `<video controls preload="metadata" src="pay.mp4"`,
		"[captions](pay.srt)", "~~~~markdown\n---\ntype: demo", "- Voice: kokoro:af_heart", "`si demo run shop.md`"} {
		if !strings.Contains(md, w) {
			t.Errorf("README.md lacks %q:\n%s", w, md)
		}
	}
	if strings.Contains(md, ".card-") {
		t.Error("README lists a temporary file")
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
	rep, err := Run(s, Options{Chrome: chrome, FFmpeg: ffmpeg, Out: out, Video: true, Voice: Voice{Engine: "system"}, Cache: filepath.Join(dir, "cache")})
	if err != nil {
		t.Fatal(err)
	}
	f := rep.Features[0]
	if f.Status != "failed" || f.Video != "sign-in.mp4" || f.Steps[4].Status != "skipped" {
		t.Errorf("feature: %+v", f)
	}
	for _, name := range []string{"sign-in.mp4", "sign-in.video.mp4", "all.mp4", "chapters.txt"} {
		if st, err := os.Stat(filepath.Join(out, name)); err != nil || st.Size() == 0 {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "sign-in.srt")); err == nil {
		t.Error("an .srt without narration")
	}
	if raw, _ := os.ReadFile(filepath.Join(out, "chapters.txt")); string(raw) != "0:00 Sign in\n" {
		t.Errorf("chapters.txt: %q", raw)
	}
}
