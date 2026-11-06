package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/felipeg48/simple-ideas/internal/deck"
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
