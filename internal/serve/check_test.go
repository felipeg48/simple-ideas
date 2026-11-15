package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/webhook"
)

func TestRunCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh commands")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "here.txt"), []byte("ok"), 0o644)
	for _, tc := range []struct {
		c        deck.Check
		pass     bool
		exit     int
		timedOut bool
	}{
		{deck.Check{Cmd: "cat here.txt", Expect: "ok"}, true, 0, false}, // the page's folder is the cwd
		{deck.Check{Cmd: "cat here.txt", Expect: "nope"}, false, 0, false},
		{deck.Check{Cmd: "exit 3"}, false, 3, false},
		{deck.Check{Cmd: "exit 3", Exit: 3}, true, 3, false},
		{deck.Check{Cmd: "sleep 5"}, false, -1, true},
	} {
		r := runCheck(dir, tc.c, 300*time.Millisecond)
		if r.Pass != tc.pass || r.Exit != tc.exit || r.TimedOut != tc.timedOut {
			t.Errorf("%+v: got %+v", tc.c, r)
		}
	}
	if r := runCheck(dir, deck.Check{Cmd: "yes | head -c 200000"}, 5*time.Second); len(r.Output) != checkOutputMax {
		t.Errorf("output capped at %d, got %d", checkOutputMax, len(r.Output))
	}
}

// TestCheckEndpoint: POST /_si/check runs a page's own check and nothing
// else, on loopback si play with the terminal on.
func TestCheckEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh commands")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lab.md"), []byte("---\ntype: lab\n---\n# Lab\n\n## Step 1\n\n<!-- check: echo hello · expect \"hello\" -->\n"), 0o644)
	s, _ := New(dir, nil, "", "")
	s.Terminal = true
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	post := func(page, cmd string) (*http.Response, checkResult) {
		b, _ := json.Marshal(map[string]string{"page": page, "cmd": cmd})
		res, err := http.Post(ts.URL+"/_si/check", "application/json", strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var r checkResult
		json.NewDecoder(res.Body).Decode(&r)
		return res, r
	}
	if res, r := post("/lab.html", "echo hello"); res.StatusCode != 200 || !r.Pass || !strings.Contains(r.Output, "hello") {
		t.Errorf("the page's check: %d %+v", res.StatusCode, r)
	}
	if res, _ := post("/lab.html", "rm -rf /tmp/x"); res.StatusCode != http.StatusForbidden {
		t.Errorf("any other command is refused, got %d", res.StatusCode)
	}
	if res, _ := post("/../../etc/passwd", "echo hello"); res.StatusCode != http.StatusNotFound {
		t.Errorf("no such page: %d", res.StatusCode)
	}
	page, _ := http.Get(ts.URL + "/lab.html")
	html, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if !regexp.MustCompile(`<body[^>]* data-si-checks`).Match(html) || !strings.Contains(string(html), `data-cmd="echo hello"`) {
		t.Error("si play's lab page turns the ✓ Check button on")
	}
	s2, _ := New(dir, nil, "", "") // si serve, or a reachable address: no checks
	ts2 := httptest.NewServer(s2.Handler())
	defer ts2.Close()
	if res, _ := http.Post(ts2.URL+"/_si/check", "application/json", strings.NewReader(`{"page":"/lab.html","cmd":"echo hello"}`)); res.StatusCode == 200 {
		t.Error("no terminal, no checks")
	}
}

// TestRelay: POST /_si/relay forwards a course page's event to the course's
// webhook, signed; only for a course page, only same-origin.
func TestRelay(t *testing.T) {
	t.Setenv(webhook.SecretEnv, "s3cret")
	var mu sync.Mutex
	var got []*http.Request
	var bodies [][]byte
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got, bodies = append(got, r), append(bodies, b)
		mu.Unlock()
	}))
	defer hook.Close()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "c", "01"), 0o755)
	os.WriteFile(filepath.Join(dir, "c", "_course.md"), []byte("---\ntitle: C\nwebhook: "+hook.URL+"\nwebhook-format: xapi\nwebhook-auth: Basic dTpw\n---\n- Module 1: M\n  - [A](01/a.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c", "01", "a.md"), []byte("# A\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "plain.md"), []byte("# P\n"), 0o644)
	s, _ := New(dir, nil, "", "")
	s.Relay = true
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	send := func(page, origin, body string) int {
		req, _ := http.NewRequest("POST", ts.URL+"/_si/relay?page="+page, strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	if c := send("/c/01/a.html", ts.URL, `{"verb":"x"}`); c != http.StatusAccepted {
		t.Fatalf("relay: %d", c)
	}
	if len(got) != 1 || string(bodies[0]) != `{"verb":"x"}` || !webhook.Verify("s3cret", got[0].Header.Get(webhook.TimestampHeader), bodies[0], got[0].Header.Get(webhook.SignatureHeader), time.Now()) ||
		got[0].Header.Get("Authorization") != "Basic dTpw" || got[0].Header.Get("X-Experience-API-Version") != "1.0.3" {
		t.Errorf("forwarded: %d %v", len(got), got)
	}
	if c := send("/c/index.html", "", `{}`); c != http.StatusAccepted {
		t.Errorf("the course home (no .md) relays too: %d", c)
	}
	if c := send("/plain.html", "", `{}`); c != http.StatusNotFound {
		t.Errorf("not a course page: %d", c)
	}
	if c := send("/c/01/a.html", "https://evil.example.com", `{}`); c != http.StatusForbidden {
		t.Errorf("another site's page: %d", c)
	}
	if c := send("/c/01/a.html", "", `not json`); c != http.StatusBadRequest {
		t.Errorf("not JSON: %d", c)
	}
	page, _ := http.Get(ts.URL + "/c/01/a.html")
	html, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if !strings.Contains(string(html), `"relay":"/_si/relay"`) || strings.Contains(string(html), hook.URL) || strings.Contains(string(html), "dTpw") {
		t.Error("a relayed course page posts to /_si/relay and holds neither the address nor the auth")
	}
	off, _ := New(dir, nil, "", "")
	ts2 := httptest.NewServer(off.Handler())
	defer ts2.Close()
	if res, _ := http.Post(ts2.URL+"/_si/relay?page=/c/01/a.html", "text/plain", strings.NewReader(`{}`)); res.StatusCode == http.StatusAccepted {
		t.Error("no --relay on a reachable address: no relay")
	}
	// si play --remote: the network listener relays too, behind the run's key
	rt := httptest.NewServer(s.RemoteHandler("sekret", []string{"tp"}))
	defer rt.Close()
	post := func(q string) int {
		res, err := http.Post(rt.URL+"/_si/relay?page=/c/01/a.html"+q, "text/plain", strings.NewReader(`{"verb":"remote"}`))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	if c := post(""); c != http.StatusForbidden {
		t.Errorf("the remote relay without the key: %d", c)
	}
	if c := post("&k=sekret"); c != http.StatusAccepted || string(bodies[len(bodies)-1]) != `{"verb":"remote"}` {
		t.Errorf("the remote relay with the key: %d", c)
	}
}
