package webdemo_test

// si play's public presenter-tools API (window.siPlay, deck/05-playapi.js)
// and the web demo steps on it (embed-terminal, embed-browser, slide), run
// against a real si play server of a tiny deck and a local fake console.
// External test package: serve imports webdemo.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/serve"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

const fakeConsole = `<!doctype html><title>Console</title><h1>Broker console</h1>
<button onclick="document.getElementById('q').textContent='orders: 3 messages'">Queues</button><p id="q"></p>`

// playDeck serves a two-slide deck with si play (the drawer on) and a fake
// console its browser: link names; it returns the deck's URL.
func playDeck(t *testing.T) string {
	if runtime.GOOS == "windows" {
		t.Skip("the drawer's shells: unix")
	}
	if tools.Find("chrome") == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	t.Setenv("SHELL", "/bin/sh")
	console := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, fakeConsole) }))
	t.Cleanup(console.Close)
	dir := t.TempDir()
	deckFile := filepath.Join(dir, "talk.md")
	os.WriteFile(deckFile, []byte("---\nbrowser: Console="+console.URL+"/\n---\n# A tiny talk\n\n---\n\n## The broker\n\nIt routes messages.\n\n---\n\n## The queue\n\nIt fills up.\n"), 0o644)
	srv, err := serve.New(deckFile, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	srv.Terminal = true
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { srv.CloseTerminals(); hs.Close() })
	return hs.URL + "/talk.html"
}

// TestPlayAPIContract pins window.siPlay: its shape (version 1) and what each
// call does to the drawer, the browser windows and the deck.
func TestPlayAPIContract(t *testing.T) {
	page := playDeck(t)
	c, err := cdp.Start(tools.Find("chrome"), 1280, 800)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Call("Page.enable", nil, nil)
	c.Call("Page.navigate", map[string]any{"url": page + "#1"}, nil)
	eval := func(js string) string {
		t.Helper()
		v, err := c.Eval(js)
		if err != nil {
			t.Fatalf("%s: %v", js, err)
		}
		return string(v)
	}
	for i := 0; i < 100 && eval(`document.readyState==='complete'&&!!window.siPlay`) != "true"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	shape := `(function(p){var o={version:p.version};
		['terminal.open','terminal.run','terminal.type','terminal.output','terminal.names','terminal.state','browser.open','browser.close','browser.min','browser.frame','browser.names','slide','next','prev','at','act']
		.forEach(function(k){var a=k.split('.');o[k]=typeof (a[1]?p[a[0]][a[1]]:p[a[0]])});return JSON.stringify(o)})(window.siPlay)`
	var got map[string]any
	var raw string
	json.Unmarshal([]byte(eval(shape)), &raw)
	json.Unmarshal([]byte(raw), &got)
	if got["version"] != float64(1) {
		t.Errorf("siPlay.version = %v, want 1", got["version"])
	}
	for k, v := range got {
		if k != "version" && v != "function" {
			t.Errorf("siPlay.%s is %v, want a function", k, v)
		}
	}
	until := func(js, what string) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if eval(js) == "true" {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("%s: %s is still false", what, js)
	}
	// the drawer: a named tab, a command in it, its output, its state
	eval(`siPlay.terminal.open("Lab 1")`)
	eval(`siPlay.terminal.run("Lab 1", "echo si-$((40+2))")`)
	until(`siPlay.terminal.output("Lab 1").indexOf("si-42")>=0`, "terminal.run then output")
	if n := eval(`JSON.stringify(siPlay.terminal.names())`); n != `"[\"Lab 1\"]"` {
		t.Errorf("terminal.names = %s", n)
	}
	eval(`siPlay.terminal.open("Lab 2", {split: "right"})`)
	if n := eval(`siPlay.terminal.names().length`); n != "2" {
		t.Errorf("a split tab: %s tabs", n)
	}
	eval(`siPlay.terminal.type("Lab 2", "echo typed\r")`)
	until(`siPlay.terminal.output("Lab 2").indexOf("typed\n")>=0||/typed\s*\n.*typed/.test(siPlay.terminal.output("Lab 2"))`, "terminal.type")
	eval(`siPlay.terminal.state("max")`)
	eval(`siPlay.terminal.state("close")`)
	if eval(`siPlay.terminal.output("Nope")`) != `""` {
		t.Error("output of no such tab is empty")
	}
	// a browser window: by link name, placed and zoomed; its frame by name
	eval(`siPlay.browser.open("Console", {x: 40, y: 60, w: 640, h: 420, zoom: 1.25})`)
	if eval(`(function(f){return !!f&&f.matches('iframe[data-si-window="Console"]')})(siPlay.browser.frame("Console"))`) != "true" {
		t.Error("browser.frame is the window's iframe[data-si-window]")
	}
	if eval(`siPlay.browser.open("Nope").then(function(){return "opened"},function(e){return e.message})`) == `"opened"` {
		t.Error("an unknown link name is refused")
	}
	eval(`siPlay.browser.min("Console")`)
	eval(`siPlay.browser.close()`)
	if eval(`siPlay.browser.names().length`) != "0" {
		t.Error("browser.close() closes every window")
	}
	// the deck
	if at := eval(`siPlay.slide(2).then(function(a){return a.slide+"/"+a.slides})`); at != `"2/3"` {
		t.Errorf("slide(2) = %s", at)
	}
	if at := eval(`siPlay.prev().then(function(a){return a.slide})`); at != "1" {
		t.Errorf("prev = %s", at)
	}
	// the cues of a Notes: script
	eval(`siPlay.act([{tool:"term",name:"Cue",cmd:"echo from-a-cue"},{tool:"browser",name:"Console"}])`)
	until(`siPlay.terminal.output("Cue").indexOf("from-a-cue")>=0&&siPlay.browser.names().length===1`, "act")
}

// TestEmbedSteps runs a web demo on si play's drawer, a browser window over
// the slides (a click inside it, zoomed) and the deck.
func TestEmbedSteps(t *testing.T) {
	page := playDeck(t)
	dir := t.TempDir()
	src := fmt.Sprintf("---\ntype: demo\napp: %s\ntimeout: 10s\n---\n\n## In action\n\n"+
		"- slide 2\n- embed-terminal \"Lab 1\"\n- run \"echo hello-$((1+2))\"\n- expect output \"hello-3\"\n- press ctrl+c\n- type \"echo again\"\n- press Enter\n- wait for output \"again\"\n"+
		"- embed-browser \"Console\" · at 60,40 · size 700x460 · zoom 125%%\n- click \"Queues\" in embed-browser \"Console\"\n- expect \"orders: 3 messages\" visible in embed-browser \"Console\"\n"+
		"- embed-browser \"Console\" min\n- embed-terminal min\n- next\n- prev\n", page)
	path := filepath.Join(dir, "demo.md")
	os.WriteFile(path, []byte(src), 0o644)
	s, err := webdemo.ParseFile(path)
	if err != nil || len(s.Problems) > 0 {
		t.Fatal(err, s.Problems)
	}
	var log strings.Builder
	rep, err := webdemo.Run(s, webdemo.Options{Chrome: tools.Find("chrome"), Out: t.TempDir(), Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK() {
		t.Fatalf("the demo failed:\n%s", log.String())
	}
}
