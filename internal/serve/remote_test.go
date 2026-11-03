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
