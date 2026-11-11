package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// fakeVoicer writes each text into a file, as if voiced.
type fakeVoicer struct {
	dir   string
	calls []string
}

func (f *fakeVoicer) Voice(v webdemo.Voice, texts []string) (map[string]webdemo.Line, int, error) {
	out := map[string]webdemo.Line{}
	for _, t := range texts {
		f.calls = append(f.calls, v.String()+"|"+t)
		p := filepath.Join(f.dir, "w.wav")
		os.WriteFile(p, []byte("RIFF "+v.String()+" "+t), 0o644)
		out[t] = webdemo.Line{WAV: p, Seconds: 1}
	}
	return out, len(texts), nil
}

func TestVoiceEndpoint(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("---\nvoice: system:Samantha\nspeakers: Ana, Ravi\nvoices: Ravi=system:Daniel\npronounce: si = ess eye\n---\n# Talk\n\nNotes[Ana]: Hello from si. [CLICK] Second part.\n\nNotes[Ravi]: My turn.\n\n---\n\n## Quiet\n\nbody\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeVoicer{dir: t.TempDir()}
	s.Voicer = f
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	get := func(p string) (int, string) {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	for _, tc := range []struct {
		path, want string
		code       int
	}{
		{"/_si/voice/1/0.wav?page=/talk.html", "system:Samantha Hello from ess eye.", 200},
		{"/_si/voice/1/1.wav?page=/talk.html", "system:Samantha Second part.", 200},
		{"/_si/voice/1/2.wav?page=/talk.html", "system:Daniel My turn.", 200},
		{"/_si/voice/1/3.wav?page=/talk.html", "", 404},
		{"/_si/voice/2/0.wav?page=/talk.html", "", 404}, // no notes
		{"/_si/voice/1/0.wav?page=/nope.html", "", 404},
		{"/_si/voice/1/0.wav?page=/../talk.html", "system:Samantha Hello from ess eye.", 200}, // cleaned to the root
	} {
		code, body := get(tc.path)
		if code != tc.code || (tc.want != "" && !strings.Contains(body, tc.want)) {
			t.Errorf("%s = %d %q, want %d %q", tc.path, code, body, tc.code, tc.want)
		}
	}
	// a page from si play asks the server for its voice; the parts are in the page
	_, page := get("/talk.html")
	if !strings.Contains(page, `"voice":true`) || !strings.Contains(page, `"say":[[{"t":"Hello from si.","s":"Ana"}`) {
		t.Errorf("the page lacks the voice config or the parts")
	}
	if s.Autoplay = true; !strings.Contains(func() string { _, p := get("/talk.html"); return p }(), `"autoplay":true`) {
		t.Errorf("--autoplay does not prime the page")
	}
}
