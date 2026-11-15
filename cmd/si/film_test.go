//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// TestNarrateConcurrent (#20): two narrated builds of one deck at once each
// voice into a WAV of their own; the first to finish removing its file
// leaves the other's in place.
func TestNarrateConcurrent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte(autoplayDeck), 0o644)
	d, _ := deck.Parse(src)
	page := filepath.Join(dir, "talk.html")
	os.WriteFile(page, []byte(`<section class="si-slide"></section><section class="si-slide"><i data-step="1"></i><i data-step="2"></i></section><section class="si-slide"></section>`), 0o644)
	dv, _, err := webdemo.ForDeck(d, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	nars := make([]*narration, 2)
	errs := make([]error, 2)
	for i := range nars {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nars[i], errs[i] = narrateWith(d, page, dv, &halfSecond{t: t, dir: t.TempDir()}, 2)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b := nars[0].audio, nars[1].audio
	defer os.Remove(b)
	if a == b || !strings.Contains(filepath.Base(a), "si-voice-talk-") {
		t.Fatalf("one file for two builds: %s %s", a, b)
	}
	os.Remove(a)
	if st, err := os.Stat(b); err != nil || st.Size() == 0 {
		t.Errorf("the other build's narration is gone: %v", err)
	}
}

// TestNearlyUniform: a blank slide (a faint grid, a counter) is nearly
// uniform; a slide with a title is not.
func TestNearlyUniform(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			c := color.RGBA{20, 22, 30, 255}
			if x%40 == 0 || y%40 == 0 { // the grid, faint
				c = color.RGBA{30, 34, 44, 255}
			}
			img.Set(x, y, c)
		}
	}
	for y := 280; y < 288; y++ { // the counter
		for x := 370; x < 390; x++ {
			img.Set(x, y, color.White)
		}
	}
	if !nearlyUniform(img) {
		t.Error("a blank slide is not seen as blank")
	}
	for y := 100; y < 160; y++ { // a title
		for x := 60; x < 340; x += 3 {
			img.Set(x, y, color.White)
		}
	}
	if nearlyUniform(img) {
		t.Error("a slide with a title is seen as blank")
	}
}

// TestDeckPosterAnimatedTheme (#15): a theme whose slides animate in still
// gives a poster with the title on it, not an empty slide.
func TestDeckPosterAnimatedTheme(t *testing.T) {
	chrome := findChrome()
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "in.css"), []byte(`/* si theme: In · extends studio */
@keyframes si-test-in{from{opacity:0;transform:translateY(40%)}to{opacity:1;transform:none}}
.si-slide h1,.si-slide p{animation:si-test-in 6s 2s both}
`), 0o644)
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte("---\ntitle: Poster\ntheme: in.css\n---\n# A title that animates in\n\nAnd a lead line under it, animating too.\n"), 0o644)
	out := t.TempDir()
	var bo, be bytes.Buffer
	if code := run([]string{"build", src, "-o", out}, &bo, &be); code != exitOK {
		t.Fatalf("build: %s", be.String())
	}
	still := filepath.Join(out, "talk.poster.jpg")
	if err := deckPoster(chrome, filepath.Join(out, "talk.html"), filepath.Join(out, "none.mp4"), still); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(still)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if nearlyUniform(img) {
		t.Error("the poster is a blank slide: the entrance had not run")
	}
}

func lookPath(name string) string {
	p, _ := exec.LookPath(name)
	return p
}

func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// TestFilmLive (#17, #16): filming a deck with cues runs its setup: and
// start: first, as si play does, so a cue reads the port its start: serves;
// stop runs stop:, the program's end and cleanup:. --no-live runs none. A
// closed drawer gives the keys back to the deck: → moves the slides.
func TestFilmLive(t *testing.T) {
	chrome, py := findChrome(), lookPath("python3")
	if chrome == "" || py == "" || testing.Short() {
		t.Skip("needs Chrome and python3")
	}
	t.Setenv("SI_HOME", t.TempDir())
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	os.WriteFile(filepath.Join(dir, "rec"), []byte("#!/bin/sh\necho \"$@\" >> \""+log+"\"\n"), 0o755)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	port := freePort(t)
	src := filepath.Join(dir, "live.md")
	os.WriteFile(src, []byte(fmt.Sprintf("---\ntitle: Live\nbrowser: Page=http://127.0.0.1:%d/\nsetup: ./rec setup\nstart: %s -m http.server %d --bind 127.0.0.1 as web · wait for port %d\nstop: ./rec stop\ncleanup: ./rec clean\n---\n# Live\n\nNotes: Let's see it. [TERM Lab: curl -s -o /dev/null -w 'code=%%{http_code}\\n' http://127.0.0.1:%d/] There. [TERM close] [CLICK]\n\n---\n\n## Two\n\nNotes: The second slide.\n\n---\n\n## Three\n\nNotes: The third.\n", port, py, port, port, port)), 0o644)

	var so syncBuf
	url, stop, err := playForFilm(src, &options{noLive: true}, &so)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if readFile(log) != "" {
		t.Errorf("--no-live ran the lifecycle: %q", readFile(log))
	}

	url, stop, err = playForFilm(src, &options{}, &so)
	if err != nil {
		t.Fatalf("%v\n%s", err, so.String())
	}
	defer stop()
	if got := strings.TrimSpace(readFile(log)); got != "setup" {
		t.Errorf("before filming: %q", got)
	}
	c, err := cdp.Start(chrome, 1280, 800)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	eval := func(js string) string {
		t.Helper()
		v, err := c.Eval(js)
		if err != nil {
			t.Fatalf("eval %s: %v", js, err)
		}
		var s any
		json.Unmarshal(v, &s)
		b, _ := json.Marshal(s)
		return strings.Trim(string(b), `"`)
	}
	wait := func(js, want string) {
		t.Helper()
		var got string
		for i := 0; i < 150; i++ {
			if got = eval(js); strings.Contains(got, want) {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("%s: %q, want %q", js, got, want)
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url + "#1"}, nil); err != nil {
		t.Fatal(err)
	}
	wait("document.readyState", "complete")
	wait("!!window.siPlay", "true")
	d, _ := deck.Parse(src)
	acts := d.Slides[0].Parts()[1].Acts
	j, _ := json.Marshal(acts[:1])
	eval(`siPlay.act(` + string(j) + `).then(function(){return 1})`)
	wait("siPlay.terminal.output('Lab')", "code=200")
	eval(`siPlay.act([{tool:'term',state:'close'}]).then(function(){return 1})`)
	// the drawer closed: → is the deck's again (si play, live)
	if err := c.Key("ArrowRight", 39); err != nil {
		t.Fatal(err)
	}
	wait("siPlay.at().slide", "2")
	// filming advances through siPlay.next(), whatever has the focus
	eval(`siPlay.terminal.open('Lab').then(function(){return 1})`)
	if err := advance(c); err != nil {
		t.Fatal(err)
	}
	wait("siPlay.at().slide", "3")
	// #19: a cue window placed, sized and zoomed; Name: URL moves that window
	for _, cue := range []string{"Page · at 100,50 · size 600x400 · zoom 125%", "Page: /sub/"} {
		a, err := deck.ParseAct("BROWSER", cue)
		if err != nil {
			t.Fatal(err)
		}
		j, _ := json.Marshal([]deck.Act{a})
		eval(`siPlay.act(` + string(j) + `).then(function(){return 1})`)
	}
	if got := eval(`(function(){var f=siPlay.browser.frame('Page'),w=f.closest('.si-web');return [siPlay.browser.names().length,w.style.left,w.style.top,w.style.width,w.style.height,w.querySelector('.si-web-url').value].join(' ')})()`); !strings.HasPrefix(got, "1 100px 50px 600px 400px ") || !strings.HasSuffix(got, "/sub/") {
		t.Errorf("the window: %s", got)
	}
	if got := eval(`document.querySelector('.si-web-zoom').textContent`); got != "125%" {
		t.Errorf("zoom %s", got)
	}
	stop()
	if got := strings.Join(strings.Fields(readFile(log)), " "); got != "setup stop clean" {
		t.Errorf("after filming: %q", got)
	}
	if ln, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
		ln.Close()
		t.Error("the start: program still serves after the film")
	}
}

// TestFilmEndsOnLastSlide (#16): the issue's deck filmed: after a [TERM …]
// cue and [TERM close] the video still moves on, and ends on slide 2 (red).
func TestFilmEndsOnLastSlide(t *testing.T) {
	chrome, ff := findChrome(), tools.Find("ffmpeg")
	if chrome == "" || ff == "" || testing.Short() {
		t.Skip("needs Chrome and ffmpeg")
	}
	t.Setenv("SI_HOME", t.TempDir())
	dir := t.TempDir()
	src := filepath.Join(dir, "stuck.md")
	os.WriteFile(src, []byte("---\ntitle: Stuck\n---\n\n# One\n\nNotes: (0.05 min)\nLet's see it. [TERM Lab: echo hello] There. [TERM close] [CLICK]\n\n---\n\n<!-- background: #d01010 -->\n\n## Two\n\nNotes: (0.05 min)\nThis slide is shown in the video.\n"), 0o644)
	out := t.TempDir()
	var bo, be bytes.Buffer
	if code := run([]string{"build", src, "--mp4", "-o", out}, &bo, &be); code != exitOK || strings.Contains(be.String(), "falling back") {
		t.Fatalf("build: %d %s", code, be.String())
	}
	last := filepath.Join(out, "last.png")
	if msg, err := exec.Command(ff, "-y", "-loglevel", "error", "-sseof", "-0.5", "-i", filepath.Join(out, "stuck.mp4"), "-frames:v", "1", last).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v %s", err, msg)
	}
	img, err := readPNG(last)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(img.Bounds().Dx()/20, img.Bounds().Dy()/2).RGBA()
	if r>>8 < 150 || g>>8 > 80 || b>>8 > 80 {
		t.Errorf("the video ends on %d,%d,%d, not on slide 2 (red)", r>>8, g>>8, b>>8)
	}
}
