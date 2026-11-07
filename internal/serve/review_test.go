package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewListener(t *testing.T) {
	s, dir := remoteServer(t)
	ts := httptest.NewServer(s.ReviewHandler("rkey"))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	do := func(method, path, origin, body string) (int, string) {
		req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	if code, _ := do("GET", "/talk.html", "", ""); code != http.StatusForbidden {
		t.Errorf("no key: %d", code)
	}
	code, page := do("GET", "/talk.html?k=rkey", "", "")
	if code != 200 || !strings.Contains(page, `window.SI_REVIEW="talk.md"`) || strings.Contains(page, `"terminal":true`) {
		t.Fatalf("review page: %d", code)
	}
	for _, p := range []string{"/.env", "/_si/position/cmd", "/_si/term/api/new", "/_si/callout"} {
		if code, _ := do("GET", p, "", ""); code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
			t.Errorf("%s on the review listener: %d", p, code)
		}
	}
	comment := `{"page":"talk.md","anchor":"slide:1","name":"Ana","text":"Say <b>why</b> first."}`
	if code, _ := do("POST", "/_si/review", "", comment); code != http.StatusForbidden {
		t.Errorf("a POST without Origin: %d", code)
	}
	if code, _ := do("POST", "/_si/review", "http://evil.example", comment); code != http.StatusForbidden {
		t.Errorf("another site's POST: %d", code)
	}
	code, body := do("POST", "/_si/review", ts.URL, comment)
	if code != 200 {
		t.Fatalf("comment: %d %s", code, body)
	}
	var c1 Comment
	json.Unmarshal([]byte(body), &c1)
	pinned := `{"page":"talk.md","anchor":"slide:1/b2-a1b2c3","quote":"` + strings.Repeat("q", 200) + `","name":"Ana","text":"This bullet."}`
	code2, body2 := do("POST", "/_si/review", ts.URL, pinned)
	var cp Comment
	if code2 != 200 || json.Unmarshal([]byte(body2), &cp) != nil || cp.Anchor != "slide:1/b2-a1b2c3" || len(cp.Quote) != 120 {
		t.Fatalf("a comment pinned to a block: %d %s", code2, body2)
	}
	for _, bad := range []string{
		`{"page":"../x.md","anchor":"page","name":"A","text":"t"}`,
		`{"page":"talk.md","anchor":"slide:1;x","name":"A","text":"t"}`,
		`{"page":"talk.md","anchor":"slide:1/b2-zzzzzz","name":"A","text":"t"}`,
		`{"page":"talk.md","anchor":"page","name":"","text":"t"}`,
		`{"page":"talk.md","anchor":"page","name":"A","text":"` + strings.Repeat("x", 2001) + `"}`,
	} {
		if code, _ := do("POST", "/_si/review", ts.URL, bad); code != http.StatusBadRequest {
			t.Errorf("refused %s: %d", bad[:40], code)
		}
	}
	if code, _ := do("POST", "/_si/review/resolve", ts.URL, `{"page":"talk.md","id":"`+c1.ID+`","resolved":true}`); code == 200 {
		t.Error("reviewers cannot resolve")
	}
	if _, err := os.Stat(filepath.Join(dir, ".si", "review", "talk.md.json")); err != nil {
		t.Fatalf("stored next to the sources: %v", err)
	}
	if cs := s.reviews().load("talk.md"); len(cs) != 2 || cs[0].Text != "Say <b>why</b> first." {
		t.Fatalf("stored as written (the page shows it with textContent): %+v", cs)
	}

	// the author, on loopback: list, reply, resolve; another host is refused
	h := s.Handler()
	call := func(method, path, host, body string) (int, string) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Host = host
		r.Header.Set("Origin", "http://"+host)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code, w.Body.String()
	}
	if code, body := call("GET", "/_si/review?page=talk.md", "127.0.0.1:7070", ""); code != 200 || !strings.Contains(body, c1.ID) {
		t.Fatalf("author list: %d %s", code, body)
	}
	if code, _ := call("POST", "/_si/review/reply", "127.0.0.1:7070", `{"page":"talk.md","id":"`+c1.ID+`","name":"Me","text":"Done."}`); code != 200 {
		t.Errorf("reply: %d", code)
	}
	if code, _ := call("POST", "/_si/review/resolve", "127.0.0.1:7070", `{"page":"talk.md","id":"`+c1.ID+`","resolved":true}`); code != 200 {
		t.Errorf("resolve: %d", code)
	}
	if code, _ := call("POST", "/_si/review/resolve", "evil.example", `{"page":"talk.md","id":"`+c1.ID+`","resolved":false}`); code != http.StatusForbidden {
		t.Errorf("a rebinding host: %d", code)
	}
	if cs := s.reviews().load("talk.md"); len(cs) != 2 || !cs[0].Resolved || len(cs[0].Replies) != 1 {
		t.Errorf("stored: %+v", cs)
	}
}
