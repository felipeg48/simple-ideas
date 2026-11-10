package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A recording: start, audio chunks appended, marks (a repeat dropped), stop
// writes the timeline and meta beside the deck.
func TestRecord(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\n---\n\n## Two\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	post := func(p, body string) (int, map[string]any) {
		res, err := http.Post(ts.URL+p, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var m map[string]any
		json.NewDecoder(res.Body).Decode(&m)
		return res.StatusCode, m
	}
	if code, _ := post("/_si/record/chunk", "x"); code != http.StatusConflict {
		t.Errorf("a chunk before start: %d", code)
	}
	if code, _ := post("/_si/record/start", `{"path":"/nope.html","ext":"webm"}`); code != http.StatusNotFound {
		t.Errorf("no such deck: %d", code)
	}
	if code, _ := post("/_si/record/start", `{"path":"/talk.html","ext":"exe"}`); code != http.StatusBadRequest {
		t.Errorf("a bad extension: %d", code)
	}
	code, m := post("/_si/record/start", `{"path":"/talk.html","ext":"webm"}`)
	if code != 200 || !strings.HasPrefix(m["dir"].(string), ".si/recordings/talk-") {
		t.Fatalf("start: %d %v", code, m)
	}
	if code, _ := post("/_si/record/start", `{"path":"/talk.html","ext":"webm"}`); code != http.StatusConflict {
		t.Errorf("a second recording: %d", code)
	}
	post("/_si/record/chunk", "AAA")
	post("/_si/record/chunk", "BBB")
	post("/_si/record/mark", `{"t":0,"slide":0,"step":0}`)
	post("/_si/record/mark", `{"t":4.5,"slide":1,"step":0}`)
	post("/_si/record/mark", `{"t":4.6,"slide":1,"step":0}`) // the same view: dropped
	if code, _ := post("/_si/record/stop", `{"t":9}`); code != 200 {
		t.Fatalf("stop: %d", code)
	}
	rec := filepath.Join(dir, m["dir"].(string))
	if b, _ := os.ReadFile(filepath.Join(rec, "audio.webm")); string(b) != "AAABBB" {
		t.Errorf("audio: %q", b)
	}
	var marks []Mark
	b, _ := os.ReadFile(filepath.Join(rec, "timeline.json"))
	json.Unmarshal(b, &marks)
	if len(marks) != 2 || marks[1].T != 4.5 || marks[1].Slide != 1 {
		t.Errorf("timeline: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(rec, "meta.json")); !strings.Contains(string(b), `"deck": "talk.md"`) || !strings.Contains(string(b), `"seconds": 9`) {
		t.Errorf("meta: %s", b)
	}
	// from another origin: refused
	req, _ := http.NewRequest("POST", ts.URL+"/_si/record/start", strings.NewReader(`{"path":"/talk.html","ext":"webm"}`))
	req.Header.Set("Origin", "https://evil.example")
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin start: %d", res.StatusCode)
	}
}
