package serve

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeRenderAndReload(t *testing.T) {
	dir := t.TempDir()
	deck := filepath.Join(dir, "talk.md")
	if err := os.WriteFile(deck, []byte("# Talk\n\nfirst version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Interval = 20 * time.Millisecond
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	stop := make(chan struct{})
	defer close(stop)
	go s.Watch(stop)

	get := func(path string) string {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return string(b)
	}
	if idx := get("/"); !strings.Contains(idx, `href="talk.html"`) || !strings.Contains(idx, "/_si/events") {
		t.Fatalf("index lacks the deck or the reload client")
	}
	if page := get("/talk.html"); !strings.Contains(page, "first version") {
		t.Fatal("deck not rendered")
	}

	res, err := http.Get(ts.URL + "/_si/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	events := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			events <- sc.Text()
		}
	}()
	time.Sleep(100 * time.Millisecond) // let the watcher take its first signature
	if err := os.WriteFile(deck, []byte("# Talk\n\nsecond version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case line := <-events:
			if line == "event: reload" {
				if page := get("/talk.html"); !strings.Contains(page, "second version") {
					t.Fatal("deck not re-rendered from markdown")
				}
				return
			}
		case <-deadline:
			t.Fatal("no reload event after a file change")
		}
	}
}

func TestSingleFileRedirects(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "one.md")
	os.WriteFile(p, []byte("# One\n"), 0o644)
	s, err := New(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := c.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/one.html" {
		t.Errorf("got %d %q", res.StatusCode, res.Header.Get("Location"))
	}
}
