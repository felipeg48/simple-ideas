package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

// TestClassroom: what a learner's browser can and cannot reach on the class
// listener. The fleet's budget is too small for any box, so nothing here
// touches Docker; a terminal tab must then fail, never open on this machine.
func TestClassroom(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lab.md"), []byte("---\ntype: lab\ntitle: L\nsandbox: docker:28-dind\n---\n# L\n\n```bash\ndocker ps\n```\n\n<!-- check: docker ps -->\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".si", "classes"), 0o700)
	os.WriteFile(filepath.Join(dir, ".si", "classes", "c.json"), []byte(`{"seats":[{"code":"SECRET22"}]}`), 0o600)
	os.WriteFile(filepath.Join(dir, "si.conf"), []byte("theme: studio\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Class = true
	c := &sandbox.Class{ID: "c", Course: "."}
	c.AddSeats(sandbox.Seat{Name: "Ada", Code: "ABCD2345"})
	fleet := &sandbox.Fleet{Spec: deck.Sandbox{Image: "docker:28-dind", Memory: 2 << 30}, Budget: 1}
	room := NewClassroom(c, fleet)
	joined := ""
	room.Joined = func(code string) { joined = code }
	h := s.ClassroomHandler(room)
	do := func(method, path, cookie string, hdr ...string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Host, r.RemoteAddr = "class.example:443", "203.0.113.9:5000"
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		for i := 0; i+1 < len(hdr); i += 2 {
			r.Header.Set(hdr[i], hdr[i+1])
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := do("GET", "/lab.html", ""); w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "seat code") {
		t.Errorf("no seat: %d", w.Code)
	}
	if w := do("POST", "/_si/term/api/new", ""); w.Code != http.StatusForbidden {
		t.Errorf("no seat, new tab: %d", w.Code)
	}
	if w := do("GET", "/go/WRONG-CODE", ""); w.Code != http.StatusNotFound {
		t.Errorf("wrong code: %d", w.Code)
	}
	w := do("GET", "/go/abcd-2345", "", "X-Forwarded-Proto", "https")
	ck := w.Result().Cookies()
	if w.Code != http.StatusFound || len(ck) != 1 || !ck[0].HttpOnly || !ck[0].Secure || ck[0].Path != "/" || joined != "ABCD2345" {
		t.Fatalf("join: %d %+v joined=%q", w.Code, ck, joined)
	}
	if c.Seats[0].Joined.IsZero() {
		t.Error("the seat's first join is not kept")
	}
	seat := "si_seat=ABCD2345"
	page := do("GET", "/lab.html", seat)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "si-doc-term") || !strings.Contains(page.Body.String(), "data-si-checks") {
		t.Errorf("the lab page: %d, drawer and checks expected", page.Code)
	}
	for _, p := range []string{"/lab.md", "/si.conf", "/.si/classes/c.json", "/_si/term/api/ls?path=/", "/_si/term/api/cat?path=/etc/hosts", "/_si/monitor", "/_si/class", "/_si/qr"} {
		if w := do("GET", p, seat, "Host", "localhost"); w.Code != http.StatusNotFound && w.Code != http.StatusForbidden && w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s reachable by a learner: %d", p, w.Code)
		}
	}
	if w := do("POST", "/_si/term/api/new", seat, "Origin", "https://evil.example"); w.Code != http.StatusForbidden {
		t.Errorf("cross-origin new tab: %d", w.Code)
	}
	if w := do("POST", "/_si/term/api/new", seat, "Origin", "https://class.example:443"); w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "budget") {
		t.Errorf("new tab without a box: %d %s (must fail, never a shell here)", w.Code, w.Body.String())
	}
	room.Update(func(c *sandbox.Class) { c.Seats = nil })
	if w := do("GET", "/lab.html", seat); w.Code != http.StatusUnauthorized {
		t.Errorf("a removed seat still gets in: %d", w.Code)
	}
	room.Update(func(c *sandbox.Class) { c.AddSeats(sandbox.Seat{Code: "ABCD2345"}); c.Expires = "2020-01-01" })
	if w := do("GET", "/go/ABCD2345", ""); w.Code != http.StatusGone {
		t.Errorf("an ended class: %d", w.Code)
	}
	for range joinFailsMax {
		do("GET", "/go/ZZZZZZZZ", "")
	}
	if w := do("GET", "/go/ABCD2345", ""); w.Code != http.StatusTooManyRequests {
		t.Errorf("guessing codes is not slowed: %d", w.Code)
	}
}

// pipeRW is a shell stand-in: what the test writes to w, the session reads.
type pipeRW struct {
	*io.PipeReader
	w *io.PipeWriter
}

func (p pipeRW) Write(b []byte) (int, error) { return len(b), nil }
func (p pipeRW) Close() error                { p.w.Close(); return p.PipeReader.Close() }

// TestTermRead: the long poll gives the replay, then each new output as soon
// as it comes, then says the shell ended.
func TestTermRead(t *testing.T) {
	h := newTermHub(t.TempDir())
	pr, pw := io.Pipe()
	s := &termSession{pty: pipeRW{pr, pw}, created: time.Now(), idle: time.Now(), exited: make(chan struct{}), wake: make(chan struct{})}
	h.tabs["1"] = s
	go s.pump()
	read := func(from string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.read(w, httptest.NewRequest("GET", "/_si/term/api/read?id=1&from="+from, nil))
		return w
	}
	pw.Write([]byte("prompt$ "))
	time.Sleep(50 * time.Millisecond)
	if w := read("-1"); w.Body.String() != "prompt$ " || w.Header().Get("X-Si-Next") != "8" {
		t.Fatalf("replay %q next %s", w.Body.String(), w.Header().Get("X-Si-Next"))
	}
	go func() { time.Sleep(100 * time.Millisecond); pw.Write([]byte("hi\n")) }()
	start := time.Now()
	if w := read("8"); w.Body.String() != "hi\n" || w.Header().Get("X-Si-Next") != "11" || time.Since(start) > 5*time.Second {
		t.Fatalf("next output %q %s", w.Body.String(), w.Header().Get("X-Si-Next"))
	}
	if w := read("3"); w.Body.String() != "mpt$ hi\n" {
		t.Errorf("from the middle %q", w.Body.String())
	}
	if w := read("-1"); w.Header().Get("X-Si-Replay") == "" {
		t.Error("a second attach is not marked as a replay")
	}
	go func() { time.Sleep(50 * time.Millisecond); pw.Close() }()
	if w := read("11"); w.Header().Get("X-Si-Ended") != "1" {
		t.Error("the end of the shell is not said")
	}
}
