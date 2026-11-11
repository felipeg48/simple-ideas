package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// halfSecond voices every text as half a second of silence.
type halfSecond struct {
	t    *testing.T
	dir  string
	said []string
}

func (h *halfSecond) Voice(v webdemo.Voice, texts []string) (map[string]webdemo.Line, int, error) {
	out := map[string]webdemo.Line{}
	for i, s := range texts {
		h.said = append(h.said, v.String()+"|"+s)
		p := filepath.Join(h.dir, strings.ReplaceAll(v.String(), ":", "_")+"-"+string(rune('a'+len(h.said)))+string(rune('a'+i))+".wav")
		silentWAV(h.t, p)
		out[s] = webdemo.Line{WAV: p, Seconds: 0.5}
	}
	return out, len(texts), nil
}

const autoplayDeck = "---\nvoice: system:Samantha\nspeakers: Ana, Ravi\nvoices: Ravi=Daniel\n---\n# Talk\n\nNotes[Ana]: Hello. [CLICK] Then.\n\nNotes[Ravi]: Mine.\n\n---\n\n## Steps\n\n<!-- steps -->\n- a\n- b\n\nNotes[Ana]: One. [CLICK] Two. [CLICK]\n\n---\n\n## Quiet\n\nbody\n"

func TestBuildAutoplayFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte(autoplayDeck), 0o644)
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	defer func(f func(string, string, string) error) { encodeM4A = f }(encodeM4A)
	encodeM4A = func(_, wav, out string) error { b, _ := os.ReadFile(wav); return os.WriteFile(out, b, 0o644) }
	out := filepath.Join(dir, "html")
	os.MkdirAll(filepath.Join(out, "talk.voice"), 0o755)
	os.WriteFile(filepath.Join(out, "talk.voice", "9-9.m4a"), []byte("old"), 0o644)
	h := &halfSecond{t: t, dir: t.TempDir()}
	audio, err := voiceForBuild(d, &config.Config{}, "", h, "ffmpeg", out, false, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	ents, _ := os.ReadDir(filepath.Join(out, "talk.voice"))
	for _, e := range ents {
		files = append(files, e.Name())
	}
	if got := strings.Join(files, " "); got != "1-0.m4a 1-1.m4a 1-2.m4a 2-0.m4a 2-1.m4a" {
		t.Errorf("files = %q (an old part must go, an empty one has none)", got)
	}
	if !strings.Contains(strings.Join(h.said, ","), "system:Daniel|Mine.") || !strings.Contains(strings.Join(h.said, ","), "system:Samantha|Hello.") {
		t.Errorf("voices: %v", h.said)
	}
	page := string(render.Deck(d, render.Options{VoiceAudio: audio}).HTML)
	if !strings.Contains(page, `{"t":"Mine.","s":"Ravi","a":"talk.voice/1-2.m4a"}`) || !strings.Contains(page, `{"t":"","s":"Ana","k":true}`) {
		t.Errorf("the page does not point at the files")
	}
	inl, err := voiceForBuild(d, &config.Config{}, "", h, "ffmpeg", out, true, io.Discard, io.Discard)
	if err != nil || !strings.HasPrefix(inl(0, 0), "data:audio/mp4;base64,UklGR") {
		t.Errorf("inline = %.40q, %v", inl(0, 0), err)
	}
	if _, err := os.Stat(filepath.Join(out, "talk.voice")); err == nil {
		t.Errorf("inline leaves no folder beside the page")
	}
}

func TestVoicedNarration(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte(autoplayDeck), 0o644)
	d, _ := deck.Parse(src)
	dv, _, err := webdemo.ForDeck(d, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	lines, err := dv.VoiceParts(&halfSecond{t: t, dir: t.TempDir()}, d.Parts())
	if err != nil {
		t.Fatal(err)
	}
	n, err := voicedNarration(d, []int{0, 2, 0}, lines, 2, "system", filepath.Join(dir, "n.wav"))
	if err != nil {
		t.Fatal(err)
	}
	// slide 1: Hello (step 0), Then + Mine (step 1 does not exist: the slide has no steps)
	for k, want := range map[[2]int]float64{{0, 0}: 0.5 + 0.6, {1, 0}: 1.1, {1, 1}: 1.1, {1, 2}: 2, {2, 0}: 2} {
		if got := n.spoken[k]; got < want-1e-9 || got > want+1e-9 {
			t.Errorf("view %v = %g s, want %g", k, got, want)
		}
	}
	if st, err := os.Stat(n.audio); err != nil || st.Size() < int64(24000*2*n.seconds) {
		t.Errorf("audio %v %v for %g s", st, err, n.seconds)
	}
}
