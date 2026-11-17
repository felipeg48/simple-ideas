package serve

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func monitorServer(t *testing.T) *Server {
	t.Helper()
	s, dir := remoteServer(t)
	md := "# Talk\n\n---\n\n## Vote\n\n<!-- poll: retry · Once · Twice · hidden -->\n\n---\n\n## Words\n\n<!-- cloud: why -->\n"
	if err := os.WriteFile(filepath.Join(dir, "talk.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	s.Mon.Key, s.AudienceKey = "monkey", "audkey"
	return s
}

func req(t *testing.T, h http.Handler, method, target, origin, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Host, r.RemoteAddr = "127.0.0.1:7070", "127.0.0.1:50000"
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestMonitorKeyAndOrigin(t *testing.T) {
	s := monitorServer(t)
	h := s.Handler()
	for _, target := range []string{"/_si/monitor", "/_si/monitor?k=wrong", "/_si/monitor/data"} {
		if w := req(t, h, "GET", target, "", ""); w.Code != http.StatusForbidden {
			t.Errorf("%s without the key: %d, want 403", target, w.Code)
		}
	}
	if w := req(t, h, "POST", "/_si/monitor/poll", "", `{"poll":"retry","action":"reveal"}`); w.Code != http.StatusForbidden {
		t.Errorf("a control without the key: %d, want 403", w.Code)
	}
	w := req(t, h, "GET", "/_si/monitor?k=monkey", "", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "si play · monitor") || w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("keyed monitor page: %d", w.Code)
	}
	if w := req(t, h, "POST", "/_si/monitor/poll?k=monkey", "http://evil.example", `{"poll":"retry","action":"reveal"}`); w.Code != http.StatusForbidden {
		t.Errorf("a control from another origin: %d, want 403", w.Code)
	}
	if w := req(t, h, "POST", "/_si/monitor/poll?k=monkey", "http://127.0.0.1:7070", `{"poll":"retry","action":"reveal"}`); w.Code != http.StatusNoContent {
		t.Errorf("a control from the page: %d, want 204", w.Code)
	}
	// a DNS-rebinding page names its own host; another machine is no loopback peer
	r := httptest.NewRequest("GET", "/_si/monitor?k=monkey", nil)
	r.Host, r.RemoteAddr = "evil.example:7070", "127.0.0.1:50000"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("a foreign Host: %d, want 403", w.Code)
	}
	r = httptest.NewRequest("GET", "/_si/monitor?k=monkey", nil)
	r.Host, r.RemoteAddr = "127.0.0.1:7070", "192.168.1.20:50000"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("a peer on the network: %d, want 403", w.Code)
	}
	s.Mon.Key = "" // no key made (si serve): no page
	if w := req(t, s.Handler(), "GET", "/_si/monitor?k=", "", ""); w.Code == http.StatusOK {
		t.Errorf("no monitor key: %d", w.Code)
	}
}

func TestMonitorRemoteRole(t *testing.T) {
	s := monitorServer(t)
	get := func(h http.Handler, target string) int {
		r := httptest.NewRequest("GET", target, nil)
		r.Host = "192.168.1.5:7071"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	on := s.RemoteHandler("remkey", []string{"tp", "monitor"})
	if c := get(on, "/_si/monitor"); c != http.StatusForbidden {
		t.Errorf("no key: %d, want 403", c)
	}
	if c := get(on, "/_si/monitor?k=audkey"); c == http.StatusOK {
		t.Errorf("the audience key opened the monitor: %d", c)
	}
	if c := get(on, "/_si/monitor?k=monkey"); c != http.StatusForbidden {
		t.Errorf("the loopback key on the network: %d, want 403", c)
	}
	if c := get(on, "/_si/monitor?k=remkey"); c != http.StatusOK {
		t.Errorf("the presenter key: %d, want 200", c)
	}
	if c := get(on, "/monitor?k=remkey"); c != http.StatusFound {
		t.Errorf("the short link: %d, want a redirect", c)
	}
	if c := get(on, "/talk.html?role=monitor&k=remkey"); c != http.StatusForbidden {
		t.Errorf("a deck page in the monitor role: %d, want 403", c)
	}
	off := s.RemoteHandler("remkey", []string{"tp"})
	if c := get(off, "/_si/monitor?k=remkey"); c == http.StatusOK {
		t.Errorf("the monitor without its role: %d", c)
	}
}

func TestMonitorControlsAndRoles(t *testing.T) {
	s := monitorServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	aud := httptest.NewServer(s.audienceHandler())
	defer aud.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	open := func(u string) {
		r, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		go func() { io.Copy(io.Discard, res.Body); res.Body.Close() }()
	}
	open(ts.URL + "/_si/events?role=deck")
	open(ts.URL + "/_si/events?role=teleprompter")
	open(ts.URL + "/_si/events?role=bogus")
	open(aud.URL + "/_si/events?role=deck") // the audience listener says who it is
	open(aud.URL + "/_si/events")
	var l Live
	for i := 0; i < 50; i++ {
		if l = s.Live(); l.Pages == 5 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if l.Roles["deck"] != 1 || l.Roles["teleprompter"] != 1 || l.Roles["audience"] != 2 || l.Roles["page"] != 1 || l.Pages != 5 {
		t.Errorf("roles: %v (%d pages)", l.Roles, l.Pages)
	}

	post := func(path, body string) int {
		w := req(t, s.Handler(), "POST", path+"?k=monkey", "", body)
		return w.Code
	}
	if post("/_si/monitor/poll", `{"poll":"retry","action":"close"}`) != http.StatusNoContent {
		t.Fatal("close")
	}
	if _, code := s.polls.vote("retry", "voter12345", []int{0}, time.Now()); code != http.StatusConflict {
		t.Errorf("a vote on a closed poll: %d, want 409", code)
	}
	post("/_si/monitor/poll", `{"poll":"retry","action":"open"}`)
	if _, code := s.polls.vote("retry", "voter12345", []int{1}, time.Now()); code != http.StatusNoContent {
		t.Errorf("a vote on an open poll: %d", code)
	}
	post("/_si/monitor/poll", `{"poll":"retry","action":"reveal"}`)
	if post("/_si/monitor/poll", `{"poll":"retry","action":"explode"}`) != http.StatusBadRequest {
		t.Error("an unknown action is refused")
	}
	var snap struct {
		Polls []monPoll
	}
	json.Unmarshal(s.monitorSnapshot(), &snap)
	if len(snap.Polls) != 1 || p0(snap.Polls).ID != "retry" || !p0(snap.Polls).Hidden || !p0(snap.Polls).Revealed || p0(snap.Polls).Total != 1 || strings.Join(p0(snap.Polls).Options, ",") != "Once,Twice" {
		t.Errorf("the poll on the monitor: %+v", snap.Polls)
	}
	// every page hears it, the audience too (with the state, not just counts)
	if v := audienceView(s.relay.last["poll:retry"]); !strings.Contains(string(v), `"revealed":true`) {
		t.Errorf("the audience's poll message: %s", v)
	}
	post("/_si/monitor/poll", `{"poll":"retry","action":"reset"}`)
	json.Unmarshal(s.monitorSnapshot(), &snap)
	if p0(snap.Polls).Total != 0 || !strings.Contains(string(s.relay.last["poll:retry"]), `"round":1`) {
		t.Errorf("reset: %+v %s", snap.Polls, s.relay.last["poll:retry"])
	}

	s.qa.ask("Why?", "Ana", "voter12345", time.Now())
	if post("/_si/monitor/qa", `{"id":1,"state":"approved"}`) != http.StatusNoContent {
		t.Error("approve")
	}
	s.clouds.add("why", "voter12345", "speed", time.Now())
	if post("/_si/monitor/cloud", `{"cloud":"why","clear":true}`) != http.StatusNoContent {
		t.Error("clear")
	}
	if sh, _, _ := s.clouds.count("why"); len(sh) != 0 {
		t.Errorf("cleared cloud: %v", sh)
	}
	// the moderation is kept on exit
	p, err := s.SaveAudience("talk", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"state": "approved"`) {
		t.Errorf("saved: %s", b)
	}
}

func p0(ps []monPoll) monPoll {
	if len(ps) == 0 {
		return monPoll{}
	}
	return ps[0]
}
