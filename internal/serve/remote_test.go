package serve

import (
	"bufio"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/render"
)

func remoteServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\nNotes:\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Remote, s.Terminal = true, true
	return s, dir
}

func TestRemoteGateAndRoles(t *testing.T) {
	s, _ := remoteServer(t)
	ts := httptest.NewServer(s.RemoteHandler("sekret", []string{"tp"}))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	status := func(path string) (int, string) {
		res, err := c.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	if code, _ := status("/talk.html?role=tp"); code != http.StatusForbidden {
		t.Errorf("no key: %d, want 403", code)
	}
	if code, _ := status("/talk.html?role=tp&k=wrong"); code != http.StatusForbidden {
		t.Errorf("wrong key: %d, want 403", code)
	}
	code, body := status("/talk.html?role=tp&k=sekret")
	if code != http.StatusOK || !strings.Contains(body, `"remote":true`) || strings.Contains(body, `"terminal":true`) {
		t.Errorf("keyed tp page: %d, remote on and terminal off expected", code)
	}
	if code, _ := status("/talk.html?role=tp"); code != http.StatusOK {
		t.Errorf("the cookie should carry the key: %d", code)
	}
	if code, _ := status("/talk.html?role=draw"); code != http.StatusForbidden {
		t.Errorf("a role that is not enabled: %d, want 403", code)
	}
	if code, _ := status("/.env"); code != http.StatusNotFound {
		t.Errorf("dotfiles on the network: %d, want 404", code)
	}
	if code, _ := status("/_si/term/api/new"); code == http.StatusOK {
		t.Error("the terminal must never be reachable on the remote listener")
	}
	if code, body := status("/"); code != http.StatusOK || !strings.Contains(body, "/talk.html?role=tp") || strings.Contains(body, "role=draw") {
		t.Errorf("picker: %d, should list only the enabled roles", code)
	}
}

// The short link a QR code carries (/tp?k=…) lands on the deck in that role
// with no cookie yet; a role that is not enabled stays a 404.
func TestRemoteShortRoleLink(t *testing.T) {
	s, _ := remoteServer(t)
	ts := httptest.NewServer(s.RemoteHandler("sekret", []string{"tp", "clicker"}))
	defer ts.Close()
	res, err := http.Get(ts.URL + "/clicker?k=sekret") // follows the redirect, no cookie jar
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Request.URL.Path != "/talk.html" || res.Request.URL.Query().Get("role") != "clicker" || !strings.Contains(string(b), `"remote":true`) {
		t.Errorf("short link: %d at %s, want the deck as clicker", res.StatusCode, res.Request.URL)
	}
	if res, _ := http.Get(ts.URL + "/draw?k=sekret"); res.StatusCode != http.StatusNotFound {
		t.Errorf("short link for a role that is not enabled: %d, want 404", res.StatusCode)
	}
}

func TestRelayFansOutAndReplaysState(t *testing.T) {
	s, _ := remoteServer(t)
	ts := httptest.NewServer(s.Handler()) // loopback listener: relay without a key
	defer ts.Close()
	post := func(from, msg string) {
		res, err := http.Post(ts.URL+"/_si/remote/send?c="+from, "application/json", strings.NewReader(msg))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("send: %d", res.StatusCode)
		}
	}
	post("deck", `{"type":"state","name":"talk","i":3}`)
	res, err := http.Get(ts.URL + "/_si/remote/events?c=pad")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	lines := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if l := sc.Text(); strings.HasPrefix(l, "data: ") {
				lines <- strings.TrimPrefix(l, "data: ")
			}
		}
	}()
	next := func() string {
		select {
		case l := <-lines:
			return l
		case <-time.After(2 * time.Second):
			t.Fatal("no relay message")
			return ""
		}
	}
	if got := next(); !strings.Contains(got, `"i":3`) {
		t.Errorf("a late subscriber should get the last state first, got %s", got)
	}
	post("pad", `{"type":"cmd","cmd":"next","name":"talk"}`) // its own message: not echoed
	post("deck", `{"type":"ink","name":"talk","p":[[0.1,0.2]]}`)
	if got := next(); !strings.Contains(got, `"ink"`) {
		t.Errorf("want the deck's ink, not the pad's own echo; got %s", got)
	}
	r2, _ := http.Post(ts.URL+"/_si/remote/send", "application/json", strings.NewReader("not json"))
	if r2.StatusCode != http.StatusBadRequest {
		t.Errorf("bad body: %d", r2.StatusCode)
	}
	r2.Body.Close()
}

// The q overlay's links carry the remote key: they go into the presenting
// laptop's loopback page only, never into a page the remote listener serves.
func TestRemoteQROnlyOnLoopback(t *testing.T) {
	s, _ := remoteServer(t)
	s.RemoteQR = &render.RemoteQR{Links: []render.RemoteLink{{Name: "Clicker", URL: "http://192.168.1.5:7071/clicker?k=sekret", SVG: "<svg></svg>"}}}
	get := func(h http.Handler, path string) string {
		ts := httptest.NewServer(h)
		defer ts.Close()
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return string(b)
	}
	if body := get(s.Handler(), "/talk.html"); !strings.Contains(body, "window.SI_REMOTE_QR=") || !strings.Contains(body, "clicker?k=sekret") {
		t.Error("the loopback page should carry the remote links for q")
	}
	if body := get(s.RemoteHandler("sekret", []string{"clicker"}), "/talk.html?role=clicker&k=sekret"); strings.Contains(body, "window.SI_REMOTE_QR=") || strings.Contains(body, "clicker?k=sekret") {
		t.Error("a page on the remote listener must not carry the links")
	}
}

// A session lists itself for the Mac app and unlists on the way out; the
// phone-code page is served on loopback only (it carries the remote key).
func TestSessionAndQRPage(t *testing.T) {
	t.Setenv("SI_SESSIONS_DIR", t.TempDir())
	unlist, err := WriteSession(Session{PID: 4242, Kind: "play", Src: "/x", URL: "http://127.0.0.1:7070/"})
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := SessionsDir()
	if _, err := os.Stat(filepath.Join(dir, "4242.json")); err != nil {
		t.Fatal("session file missing")
	}
	unlist()
	if _, err := os.Stat(filepath.Join(dir, "4242.json")); err == nil {
		t.Fatal("session file left behind")
	}
	s, _ := remoteServer(t)
	s.RemoteQR = &render.RemoteQR{Links: []render.RemoteLink{{Name: "Clicker", URL: "http://192.168.1.5:7071/clicker?k=sekret", SVG: "<svg></svg>"}}}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/_si/qr")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(b), "clicker?k=sekret") {
		t.Errorf("qr page: %d", res.StatusCode)
	}
	req, _ := http.NewRequest("GET", ts.URL+"/_si/qr", nil)
	req.Host = "evil.example:7070" // DNS rebinding: not a loopback host
	if res, err := http.DefaultClient.Do(req); err == nil && res.StatusCode == 200 {
		t.Error("the qr page must refuse a non-loopback Host")
	}
}

// The audience key opens the audience view only: the deck without Notes: or
// Points:, a stream that carries where the talk is and nothing else, and no
// way to send, to open a presenter role, or to reach the presenter routes.
func TestAudience(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\nNotes:\nSECRET-SCRIPT\n\nPoints:\n- SECRET-POINT\n\n---\n\n## Two\n\nText.\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Remote, s.AudienceKey = true, "aud"
	ts := httptest.NewServer(s.RemoteHandler("presenter", []string{"tp", "clicker"}))
	defer ts.Close()
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	get := func(path string) (int, string) {
		res, err := noRedirect.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, res.Header.Get("Location") + string(b)
	}
	if code, loc := get("/a?k=aud"); code != http.StatusFound || !strings.Contains(loc, "/talk.html?role=audience&k=aud") {
		t.Errorf("audience link: %d %q", code, loc)
	}
	code, body := get("/talk.html?role=audience&k=aud")
	if code != 200 || strings.Contains(body, "SECRET-SCRIPT") || strings.Contains(body, "SECRET-POINT") {
		t.Errorf("audience page %d: must not carry the speaker's text", code)
	}
	if !strings.Contains(body, "Two") {
		t.Error("audience page should carry the slides")
	}
	for _, p := range []string{"/talk.html?role=tp&k=aud", "/tp?k=aud", "/_si/qr?k=aud", "/_si/term/api/new?k=aud"} {
		if code, _ := get(p); code == 200 || code == http.StatusFound {
			t.Errorf("the audience key must not open %s (%d)", p, code)
		}
	}
	if res, err := http.Post(ts.URL+"/_si/remote/send?k=aud", "application/json", strings.NewReader(`{"type":"cmd","cmd":"next"}`)); err == nil && res.StatusCode < 300 {
		t.Error("an audience page must not send to the relay")
	}
	if code, _ := get("/talk.html?role=tp&k=presenter"); code != 200 {
		t.Errorf("the presenter key still opens the teleprompter: %d", code)
	}
	state := []byte(`{"type":"state","name":"talk","i":1,"k":0,"n":2,"title":"Two","notes":"SECRET-SCRIPT","points":"SECRET-POINT","cur":"<section>…</section>"}`)
	got := string(audienceView(state))
	if strings.Contains(got, "SECRET") || strings.Contains(got, "section") || !strings.Contains(got, `"i":1`) {
		t.Errorf("audience view of a state: %s", got)
	}
	for _, m := range []string{`{"type":"ink","p":[[0,0]]}`, `{"type":"cmd","cmd":"next"}`} {
		if audienceView([]byte(m)) != nil {
			t.Errorf("the audience must not receive %s", m)
		}
	}
}

// One ballot per phone and poll; changes replace it, an empty ballot takes it
// back; bad input and fast repeats are refused; counts reach the audience.
func TestPolls(t *testing.T) {
	p := newPolls()
	now := time.Now()
	msg, code := p.vote("retry", "phoneaaaaaa1", []int{0}, now)
	if code != http.StatusNoContent || !strings.Contains(string(msg), `"counts":[1,0,0,0,0,0]`) || !strings.Contains(string(msg), `"total":1`) {
		t.Fatalf("first vote: %d %s", code, msg)
	}
	p.vote("retry", "phonebbbbbb2", []int{2}, now)
	msg, _ = p.vote("retry", "phoneaaaaaa1", []int{2}, now.Add(time.Second)) // changed mind
	if !strings.Contains(string(msg), `"counts":[0,0,2,0,0,0]`) || !strings.Contains(string(msg), `"total":2`) {
		t.Errorf("a changed vote replaces the old one: %s", msg)
	}
	if _, code := p.vote("retry", "phoneaaaaaa1", []int{1}, now.Add(time.Second+100*time.Millisecond)); code != http.StatusTooManyRequests {
		t.Errorf("a vote 100 ms after the last: %d, want 429", code)
	}
	msg, _ = p.vote("retry", "phonebbbbbb2", nil, now.Add(2*time.Second))
	if !strings.Contains(string(msg), `"total":1`) {
		t.Errorf("an empty ballot takes the vote back: %s", msg)
	}
	for _, bad := range []struct {
		poll, voter string
		c           []int
	}{{"Retry!", "phoneaaaaaa1", []int{0}}, {"retry", "x", []int{0}}, {"retry", "phonecccccc3", []int{6}}, {"retry", "phonecccccc3", []int{1, 1}}, {"retry", "phonecccccc3", []int{-1}}} {
		if _, code := p.vote(bad.poll, bad.voter, bad.c, now.Add(time.Hour)); code != http.StatusBadRequest {
			t.Errorf("%+v: %d, want 400", bad, code)
		}
	}
	if got := string(audienceView([]byte(`{"type":"poll","id":"retry","counts":[1,0],"total":1}`))); !strings.Contains(got, `"counts":[1,0]`) {
		t.Errorf("the audience gets poll counts: %s", got)
	}
}
