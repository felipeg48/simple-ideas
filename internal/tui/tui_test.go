package tui

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

var escRE = regexp.MustCompile("\x1b\\[[0-9;]*m|\x1b\\]8;[^\x07\x1b]*(\x07|\x1b\\\\)")

func plain(s string) string { return escRE.ReplaceAllString(s, "") }

func testDeck(t *testing.T, md string) *deck.Deck {
	t.Helper()
	p := filepath.Join(t.TempDir(), "talk.md")
	if err := os.WriteFile(p, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

const talk = `---
title: The Thursday Promise
author: Ada
footer: Northwind
---

# The Thursday Promise

<!-- badge: Operations review · ok -->

How one order arrives on time.

---

## One promise, ==five hand-offs==

<!-- steps -->
- The shop promises **Thursday**
- Three teams own one hand-off each

Notes: Look at that last line. [CLICK] It still arrives late. [CLICK]

---

## The retry rule

` + "```go title=\"retry.go\" hl=\"2\"\nkey := \"order\"\nreturn charge(key) // <1>\n```" + `

| Hand-off | Before | After |
|---|---:|---:|
| Paid to picked | 9 h | 3 h |
`

func TestSlideFillsTheScreenAndCountsBuilds(t *testing.T) {
	d := testDeck(t, talk)
	pal := PaletteFor("studio", "")
	for _, size := range [][2]int{{100, 30}, {80, 24}, {40, 16}, {200, 60}} {
		for i := range d.Slides {
			s, _ := Slide(d, i, 99, size[0], size[1], pal)
			if n := len(strings.Split(s, "\n")); n != size[1] {
				t.Errorf("slide %d at %v: %d lines", i+1, size, n)
			}
		}
	}
	_, steps := Slide(d, 1, 0, 100, 30, pal)
	if steps != 2 {
		t.Errorf("steps: %d", steps)
	}
	at0, _ := Slide(d, 1, 0, 100, 30, pal)
	at1, _ := Slide(d, 1, 1, 100, 30, pal)
	if strings.Contains(plain(at0), "Thursday") || !strings.Contains(plain(at1), "The shop promises Thursday") || strings.Contains(plain(at1), "Three teams") {
		t.Errorf("builds reveal one bullet at a time:\n%s", plain(at1))
	}
}

func TestCoverCodeAndTable(t *testing.T) {
	d := testDeck(t, talk)
	pal := PaletteFor("studio", "")
	cover, _ := Slide(d, 0, 0, 100, 30, pal)
	p := plain(cover)
	if !strings.Contains(p, "█") || !strings.Contains(p, "● Operations review") || !strings.Contains(p, "Ada") {
		t.Errorf("cover: block title, badge and byline:\n%s", p)
	}
	if strings.Index(p, "● Operations") > strings.Index(p, "█") {
		t.Error("a cover's badge sits above its title")
	}
	code, _ := Slide(d, 2, 0, 100, 30, pal)
	p = plain(code)
	for _, want := range []string{"retry.go", "return charge(key)", "❶", "┌", "Paid to picked", "9 h"} {
		if !strings.Contains(p, want) {
			t.Errorf("code / table slide lacks %q:\n%s", want, p)
		}
	}
	two, _ := Slide(d, 1, 2, 100, 30, pal)
	if !strings.Contains(two, "48;2;") { // the ==highlight== band is a background colour
		t.Error("highlight band missing")
	}
}

func TestBigFont(t *testing.T) {
	if rows := big("HI"); len(rows) != 3 || rows[0] != "█   █ ▀▀█▀▀" {
		t.Errorf("big: %q", rows)
	}
	if _, ok := bigLines("Ünïcode", 80); ok {
		t.Error("letters without a glyph fall back to plain text")
	}
	if lines, ok := bigLines("THE THURSDAY PROMISE", 70); !ok || len(lines) != 7 {
		t.Errorf("wraps by word into two block lines: %d %v", len(lines), ok)
	}
}

func TestPaletteFromTheme(t *testing.T) {
	p := PaletteFor("acemq-pro", "")
	if hex(p.Accent) == hex(PaletteFor("studio", "").Accent) {
		t.Error("a theme brings its own accent")
	}
	if hex(PaletteFor("no-such-theme", "").BG) != hex(PaletteFor("studio", "").BG) {
		t.Error("an unknown theme falls back to studio")
	}
}

func TestPresenterStepsThroughBuilds(t *testing.T) {
	d := testDeck(t, talk)
	m := &presenter{d: d, pal: PaletteFor("studio", ""), w: 100, h: 30, start: time.Now(), slideStart: time.Now()}
	key := func(k string) { m.Update(tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}) }
	key("n")
	if m.i != 1 || m.k != 0 {
		t.Fatalf("next slide: %d.%d", m.i, m.k)
	}
	key("n")
	key("n")
	if m.i != 1 || m.k != 2 {
		t.Fatalf("two builds: %d.%d", m.i, m.k)
	}
	key("n")
	if m.i != 2 {
		t.Fatalf("then the next slide: %d.%d", m.i, m.k)
	}
	key("p")
	if m.i != 1 || m.k != 2 {
		t.Errorf("back shows the slide whole: %d.%d", m.i, m.k)
	}
}

func TestTeleprompterCues(t *testing.T) {
	d := testDeck(t, talk)
	m := &teleprompter{d: d, pal: PaletteFor("studio", ""), w: 120, h: 30, i: 1, k: 1, start: time.Now(), slideStart: time.Now(), f: &follower{}}
	p := plain(m.render())
	if !strings.Contains(p, "✓ click") || !strings.Contains(p, "▶ CLICK") || !strings.Contains(p, "Look at that last line") {
		t.Errorf("cues: done, then the next one highlighted:\n%s", p)
	}
	if !strings.Contains(p, "NOW 2/3") || !strings.Contains(p, "NEXT") {
		t.Errorf("previews:\n%s", p)
	}
}

func TestTeleprompterSpeaker(t *testing.T) {
	d := testDeck(t, "---\nspeakers: Ana, Felipe\n---\n# Talk\n\n---\n\n## Pay\n\nNotes[Ana]: (1 min) The token reaches payments and turns red. [CLICK]\n\nNotes[Felipe]: (0.5 min) Here is the fix we shipped: one retry, thirty seconds later. [CLICK]\n")
	m := &teleprompter{d: d, pal: PaletteFor("studio", ""), w: 140, h: 30, i: 1, k: 1, start: time.Now(), slideStart: time.Now(), f: &follower{}, opt: Options{Speaker: "Felipe"}}
	p := plain(m.render())
	if !strings.Contains(p, "Teleprompter · Felipe") || !strings.Contains(p, "Ana 01:00") || !strings.Contains(p, "one retry") || strings.Count(p, "CLICK") != 1 {
		t.Errorf("Felipe's script, Ana as a cue line:\n%s", p)
	}
	if !strings.Contains(p, "▶ CLICK") {
		t.Errorf("Felipe's cue is step 2, the next one:\n%s", p)
	}
}

func TestSocketSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	if len(path) > 100 {
		path = filepath.Join(os.TempDir(), "si-test.sock")
	}
	cmds := make(chan command, 4)
	h, err := listen(path, func(c command) { cmds <- c })
	if err != nil {
		t.Fatal(err)
	}
	defer h.close()
	got := make(chan state, 4)
	f := follow(path, func(s state, live bool) {
		if live {
			got <- s
		}
	})
	if c := <-cmds; c.Cmd != "hello" {
		t.Fatalf("first command: %v", c)
	}
	h.send(state{I: 3, K: 1, N: 9})
	select {
	case s := <-got:
		if s.I != 3 || s.K != 1 {
			t.Errorf("state %v", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no state")
	}
	f.send(command{Cmd: "next"})
	if c := <-cmds; c.Cmd != "next" {
		t.Errorf("command %v", c)
	}
}

func TestTeleprompterFollowsABrowserDeck(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte(talk), 0o644)
	srv, err := serve.New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer func() { ts.CloseClientConnections(); ts.Close() }() // the follower's stream never ends on its own
	addr := strings.TrimPrefix(ts.URL, "http://")
	got := make(chan state, 8)
	f := &follower{onState: func(s state, live bool) {
		if live {
			got <- s
		}
	}}
	f.followBrowser(addr, filepath.Join(dir, "talk.md"))
	time.Sleep(200 * time.Millisecond)
	// the deck in the browser reports where it is
	http.Post(ts.URL+"/_si/position", "application/json", strings.NewReader(`{"page":"/talk.html","i":1,"k":1,"steps":2,"n":3}`))
	select {
	case s := <-got:
		if s.I != 1 || s.K != 1 {
			t.Errorf("state %+v", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the teleprompter never heard the browser deck")
	}
	// and the teleprompter's next reaches the deck's stream as a cmd event
	resp, err := http.Get(ts.URL + "/_si/position/events?page=/talk.html")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !f.send(command{Cmd: "next"}) {
		t.Fatal("send failed")
	}
	sc := bufio.NewScanner(resp.Body)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && sc.Scan() {
		if strings.Contains(sc.Text(), `"cmd":"next"`) {
			return
		}
	}
	t.Error("the deck never got next")
}

func TestCodeShowStepsAreBuilds(t *testing.T) {
	d := testDeck(t, "# T\n\n---\n\n## Retry\n\n```go id=r window=mac title=retry.go step=\"One try\"\nsend()\n```\n<!-- callout: 1 Only once -->\n\n```go id=r window=mac title=retry.go step=\"Five tries\"\nfor i := 0; i < 5; i++ {\n\tsend()\n}\n```\n<!-- callout: 1 The loop -->\n\n```go id=r window=mac title=retry.go\nretry(send)\n```\n")
	for k, want := range []string{"1/3 One try", "2/3 Five tries", "3/3"} {
		s, steps := Slide(d, 1, k, 80, 30, Palette{})
		p := plain(s)
		if steps != 2 || !strings.Contains(p, "retry.go · "+want) {
			t.Errorf("k=%d: steps=%d, want title %q:\n%s", k, steps, want, p)
		}
		if k == 0 && (!strings.Contains(p, "Only once") || strings.Contains(p, "The loop") || strings.Contains(p, "retry(send)")) {
			t.Errorf("step 1 shows only its own code and callouts:\n%s", p)
		}
		if k == 1 && (!strings.Contains(p, "The loop") || strings.Contains(p, "Only once")) {
			t.Errorf("step 2 shows its callouts:\n%s", p)
		}
	}
}
