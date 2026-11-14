package webdemo

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

// promoSite is a small marketing site for the promo kit's tests: a sticky
// header, a cookie banner (gone once accepted, by a cookie), a hero, tall
// sections, a same-origin and a cross-origin iframe (frameURL), robots.txt
// keeping crawlers out of /admin.
func promoSite(frameURL string) http.Handler {
	head := `<!doctype html><html lang="en"><meta charset="utf-8"><title>Harbor | Ship orders on time</title>
<style>body{margin:0;font:18px/1.5 system-ui}header{position:sticky;top:0;height:80px;background:#123;color:#fff;display:flex;gap:2em;align-items:center;padding:0 2em;z-index:5}
header a{color:#fff}section{padding:2em}#pricing{min-height:1400px;background:#eef}#cookie-consent{position:fixed;left:0;right:0;bottom:0;padding:1em;background:#333;color:#fff;z-index:9}
iframe{width:600px;height:220px;border:4px solid #888;padding:6px}</style>
<header><a href="/">Harbor</a><nav><a href="/pricing">Pricing</a> <a href="/about">About</a> <a href="/admin">Admin</a></nav></header>
<div id="cookie-consent" class="cookie-bar">We use cookies. <button>Accept all</button></div>
<script>if(document.cookie.indexOf('consent=yes')>=0)document.getElementById('cookie-consent').remove();
else document.querySelector('#cookie-consent button').onclick=function(){document.cookie='consent=yes;path=/';document.getElementById('cookie-consent').remove()}</script>`
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, q *http.Request) {
		io.WriteString(w, "User-agent: *\nDisallow: /admin\n")
	})
	page := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, q *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, head+"<main>"+body+"</main>")
		}
	}
	mux.HandleFunc("/{$}", page(`<section class="hero"><h1>Ship every order on time</h1><p>Harbor plans your picking, packing and carriers in one place, so nothing waits on a desk.</p><a href="/pricing" class="cta">Start free</a></section>
<section id="features"><h2>Everything in one queue</h2><p>Orders from every shop land in one queue and are packed in the order they must leave.</p></section>
<section id="pricing"><h2 id="pricing-title">Simple pricing</h2><p>One plan for every warehouse, priced by the orders you ship each month.</p><a href="/pricing">See the plans</a></section>
<section id="preview"><h2>Try it here</h2><p>The shop below runs inside the page, as your customers would see it.</p>
<iframe class="preview" title="Shop" src="/frame"></iframe><iframe class="remote" name="remote" src="`+frameURL+`"></iframe></section>`))
	mux.HandleFunc("/pricing", page(`<section><h1>Plans for every warehouse</h1><p>Start free with a hundred orders a month and grow when you need to.</p><a href="/">Back home</a></section>`))
	mux.HandleFunc("/about", page(`<section><h1>About Harbor</h1><p>We are a small team that spent ten years in warehouses before writing software.</p></section>`))
	mux.HandleFunc("/admin", page(`<section><h1>Admin</h1><p>Nobody should crawl this page, ever, not even a demo script.</p></section>`))
	mux.HandleFunc("/frame", shopFrame)
	return mux
}

// shopFrame is the framed shop: a quantity, a Buy button, a result.
func shopFrame(w http.ResponseWriter, q *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html><meta charset="utf-8"><body style="margin:0;font:16px system-ui">
<label>Qty <input id="qty" value="1"></label><button style="margin-top:80px;display:block">Buy</button><p id="out"></p>
<script>document.querySelector('button').onclick=function(){document.getElementById('out').textContent='Bought '+document.getElementById('qty').value}</script>`)
}

// sites starts the promo site on 127.0.0.1 and the cross-origin frame on
// localhost (another site: Chrome puts it in a process of its own).
func sites(t *testing.T) string {
	fr := httptest.NewServer(http.HandlerFunc(shopFrame))
	t.Cleanup(fr.Close)
	_, port, _ := net.SplitHostPort(fr.Listener.Addr().String())
	main := httptest.NewServer(promoSite("http://localhost:" + port + "/frame"))
	t.Cleanup(main.Close)
	return main.URL
}

func needChrome(t *testing.T) string {
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	return chrome
}

func runText(t *testing.T, chrome, text string) (*Report, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "promo.md")
	os.WriteFile(p, []byte(text), 0o644)
	s, err := ParseFile(p)
	if err != nil || len(s.Problems) > 0 {
		t.Fatalf("%v %v", err, s.Problems)
	}
	var log strings.Builder
	rep, err := Run(s, Options{Chrome: chrome, Log: &log, Out: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return rep, log.String()
}

// TestSessionKeep: with session: keep, the cookie banner accepted in
// ## Before stays accepted in every feature; without it each feature starts
// in a fresh profile and sees the banner again.
func TestSessionKeep(t *testing.T) {
	chrome := needChrome(t)
	app := sites(t)
	src := func(session string) string {
		return "---\ntype: demo\napp: " + app + "\n" + session + "---\n\n## Before\n\n- go /\n- click \"Accept all\"\n\n" +
			"## One\n\n- go /\n- expect #cookie-consent hidden\n\n## Two\n\n- go /pricing\n- expect #cookie-consent hidden\n"
	}
	if rep, log := runText(t, chrome, src("session: keep\n")); !rep.OK() || rep.Passed != 2 {
		t.Errorf("session: keep: %d passed\n%s", rep.Passed, log)
	}
	if rep, _ := runText(t, chrome, src("")); rep.OK() || rep.Failed != 2 {
		t.Errorf("a fresh profile per feature: %d failed", rep.Failed)
	}
}

// TestFrames: steps reach into a same-origin and a cross-origin iframe.
func TestFrames(t *testing.T) {
	chrome := needChrome(t)
	app := sites(t)
	text := "---\ntype: demo\napp: " + app + "\n---\n\n## Buy in the preview\n\n- go /\n" +
		"- type \"3\" into \"Qty\" in frame .preview\n- click \"Buy\" in frame .preview\n- expect #out visible in frame .preview\n- expect \"Bought 3\" visible in frame .preview\n" +
		"- spotlight \"Buy\" in frame .preview\n\n" +
		"## Buy across origins\n\n- go /\n- type \"7\" into #qty in frame \"remote\"\n- click role=button \"Buy\" in frame \"remote\"\n- expect \"Bought 7\" visible in frame iframe.remote\n"
	rep, log := runText(t, chrome, text)
	if !rep.OK() || rep.Passed != 2 {
		t.Errorf("frames: %d passed, %d failed\n%s", rep.Passed, rep.Failed, log)
	}
}

// TestHeaderOffset: an element scrolled to lands below the sticky header,
// a tall one with its top just under it; one already in view is not moved;
// header-offset: none puts a tall one at the very top.
func TestHeaderOffset(t *testing.T) {
	chrome := needChrome(t)
	app := sites(t)
	c, err := cdp.Start(chrome, 1440, 900)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Call("Page.navigate", map[string]any{"url": app + "/"}, nil)
	for i := 0; i < 50; i++ {
		if v, _ := c.Eval("document.readyState"); string(v) == `"complete"` {
			break
		}
		c.Eval("new Promise(function(r){setTimeout(r,100)})")
	}
	at := func(css string, header any) (top float64, scrolled float64) {
		arg, _ := json.Marshal(map[string]any{"loc": Locator{CSS: css}, "want": "any", "scroll": true, "header": header})
		c.Eval(locateJS + "(" + string(arg) + ")")
		v, _ := c.Eval(fmt.Sprintf(`[document.querySelector(%q).getBoundingClientRect().top, scrollY]`, css))
		var r [2]float64
		json.Unmarshal(v, &r)
		return r[0], r[1]
	}
	if top, _ := at("#pricing", "auto"); top < 80 || top > 110 {
		t.Errorf("a tall section: its top at %g, want just under the 80px header", top)
	}
	if top, _ := at("#preview h2", "auto"); top < 80 {
		t.Errorf("a heading: at %g, under the header", top)
	}
	_, y0 := at("#preview h2", "auto")
	if _, y1 := at("#preview p", "auto"); y1 != y0 {
		t.Errorf("an element in view moved the page: %g → %g", y0, y1)
	}
	if top, _ := at("#pricing", 0); top > 20 {
		t.Errorf("header-offset: none: a tall section at %g, want the top", top)
	}
	if top, _ := at("#features", 200); top < 200 {
		t.Errorf("header-offset: 200: at %g", top)
	}
}

// TestDraft: si demo draft reads the local site (not /admin, which
// robots.txt forbids) and writes a script that passes si demo test.
func TestDraft(t *testing.T) {
	chrome := needChrome(t)
	app := sites(t)
	var log strings.Builder
	md, err := Draft(app+"/", DraftOptions{Chrome: chrome, Pages: 4, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type: demo", "app: " + app, "session: keep", "title: Harbor", "## Before", "## Ship every order on time",
		"Say: Ship every order on time. Harbor plans your picking", "<!-- draft: polish this line -->", "## Simple pricing", "- go /pricing", "## Start free", "outro: Start free"} {
		if !strings.Contains(md, want) {
			t.Errorf("draft lacks %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, "/admin") || strings.Contains(log.String(), "/admin") {
		t.Errorf("robots.txt: /admin was read\n%s", log.String())
	}
	rep, out := runText(t, chrome, md)
	if !rep.OK() {
		t.Errorf("the draft fails si demo test:\n%s\n%s", out, md)
	}
}

func TestRobots(t *testing.T) {
	rb := parseRobots("User-agent: Googlebot\nDisallow: /\n\nUser-agent: *\nDisallow: /admin\nDisallow: /*.pdf$\nAllow: /admin/public\nCrawl-delay: 2\n")
	for p, want := range map[string]bool{"/": true, "/pricing": true, "/admin": false, "/admin/x": false, "/admin/public/a": true, "/a.pdf": false, "/a.pdfx": true} {
		if got := rb.allowed(p); got != want {
			t.Errorf("%s: allowed %v, want %v", p, got, want)
		}
	}
	if rb.delay.Seconds() != 2 {
		t.Errorf("crawl-delay: %v", rb.delay)
	}
	if !parseRobots("User-agent: *\nDisallow:\n").allowed("/x") {
		t.Error("an empty Disallow allows everything")
	}
	if parseRobots("User-agent: si\nDisallow: /\nUser-agent: *\nDisallow:\n").allowed("/") {
		t.Error("si's own group wins")
	}
}

func TestParsePromo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "logo.png"), []byte("png"), 0o644)
	os.WriteFile(filepath.Join(dir, "bed.mp3"), []byte("mp3"), 0o644)
	p := filepath.Join(dir, "p.md")
	s := Parse(p, "---\ntype: demo\napp: http://localhost:1\nsession: keep\nheader-offset: 72px\nintro: Harbor · Ship on time\noutro: Start free · harbor.example\n"+
		"music: bed.mp3 · 20%\nlogo: logo.png\nurl: https://harbor.example\ntheme: forest\n---\n\n## A\n\nLower-third: Ana Ruiz · Head of Support\n\n"+
		"- go /\n- lower-third \"Ana Ruiz\" · \"Ops\" · for 3s\n- click \"Buy\" in frame .preview\n- type \"2\" into #qty in frame \"shop\"\n")
	if len(s.Problems) > 0 {
		t.Fatal(s.Problems)
	}
	b := s.Brand
	if s.Session != "keep" || s.Header != 72 || b.Intro == nil || b.Intro.Title != "Harbor" || b.Intro.Subtitle != "Ship on time" ||
		b.Outro.Title != "Start free" || b.Volume != 0.2 || b.Music != filepath.Join(dir, "bed.mp3") || b.Logo != filepath.Join(dir, "logo.png") || b.Theme != "forest" {
		t.Errorf("front matter: %+v %+v", s, b)
	}
	f := s.Features[0]
	st := f.Blocks[0].Steps
	if f.Third != "Ana Ruiz · Head of Support" || st[1].Kind != "lower-third" || st[1].Value.Text != "Ana Ruiz" || st[1].Opts["sub"] != "Ops" || st[1].For.Seconds() != 3 {
		t.Errorf("lower thirds: %q %+v", f.Third, st[1])
	}
	if st[2].Target.Name != "Buy" || st[2].Target.Frame != ".preview" || st[3].Target.CSS != "#qty" || !strings.Contains(st[3].Target.Frame, `iframe[name="shop"]`) {
		t.Errorf("frames: %+v %+v", st[2].Target, st[3].Target)
	}
	if got := st[2].Target.String(); got != `"Buy" in frame .preview` {
		t.Errorf("locator string: %s", got)
	}
	s = Parse(p, "---\ntype: demo\napp: http://localhost:1\nsession: sometimes\nheader-offset: tall\nmusic: none.mp3 · loud\ntheme: nope\n---\n\n## A\n\n- go /\n")
	if len(s.Problems) != 5 {
		t.Errorf("bad values: %v", s.Problems)
	}
	// a theme .css file for the cards: the built-in it extends, under its own tokens
	os.WriteFile(filepath.Join(dir, "harbor.css"), []byte("/* si theme: Harbor · extends forest */\n:root {\n  --si-accent: #c2410c;\n  --si-bg: #0b1020;\n}\n"), 0o644)
	s = Parse(p, "---\ntype: demo\napp: http://localhost:1\ntheme: harbor.css\nvertical-fill: theme\n---\n\n## A\n\n- go /\n")
	if len(s.Problems) > 0 || s.Brand.Theme != filepath.Join(dir, "harbor.css") || s.Brand.VFill != "theme" {
		t.Fatalf("theme file: %v %+v", s.Problems, s.Brand)
	}
	if tk := tokensOf(s.Brand.Theme); !strings.Contains(tk, "--si-accent:#c2410c;") || strings.Count(tk, "--si-fg:") != 1 {
		t.Errorf("theme file tokens: %s", tk)
	}
	if f := themeFill(s.Brand.Theme); f != "0x0b1020" {
		t.Errorf("the theme's fill: %s", f)
	}
	s = Parse(p, "---\ntype: demo\napp: http://localhost:1\ntheme: gone.css\nvertical-fill: stripes\n---\n\n## A\n\n- go /\n")
	if len(s.Problems) != 2 {
		t.Errorf("a missing theme file and a bad fill: %v", s.Problems)
	}
	if h, _ := ParseHeaderOffset("auto"); h != -1 {
		t.Error("auto")
	}
	if h, _ := ParseHeaderOffset("none"); h != 0 {
		t.Error("none")
	}
}

func TestPromoMath(t *testing.T) {
	if f := firstSentence("Ship on time. Every time."); f < 0.5 || f > 0.6 {
		t.Errorf("first sentence share: %g", f)
	}
	if firstSentence("No full stop") != 1 {
		t.Error("one sentence")
	}
	// the intro belongs to the first chapter (0:00), the outro to the last
	parts := []part{{Name: "Intro", Hold: 3.5}, {Name: "A", Seconds: 10}, {Name: "B", Seconds: 5}, {Name: "Outro", Hold: 4.5}}
	cs := chapters(parts)
	if len(cs) != 2 || cs[0].Start != 0 || cs[0].Clip != 3.5+cardSeconds || cs[1].End != 3.5+2*cardSeconds+15+4.5 {
		t.Errorf("chapters: %+v", cs)
	}
	a := strings.Join(allArgs(parts, 1440, 900, Transition{Kind: "cut"}, "m", "o.mp4"), " ")
	if !strings.Contains(a, "concat=n=6:v=1:a=1") || !strings.Contains(a, "-t 3.5 -i ") {
		t.Errorf("all with cards: %s", a)
	}
	a = strings.Join(xfadeArgs(parts, 1440, 900, DefaultTransition, "m", "o.mp4"), " ")
	if !strings.Contains(a, "concat=n=6:v=0:a=1") || strings.Count(a, "xfade=") != 5 {
		t.Errorf("xfade with cards: %s", a)
	}
	a = strings.Join(musicArgs("v.mp4", "m.mp3", 0.25, 30, "o.mp4"), " ")
	for _, w := range []string{"-stream_loop -1 -i m.mp3", "volume=0.250", "sidechaincompress", "amix=inputs=2:duration=first", "-c:v copy", "-t 30.000"} {
		if !strings.Contains(a, w) {
			t.Errorf("music: %s lacks %s", a, w)
		}
	}
	a = strings.Join(verticalArgs("all.mp4", 1440, 900, "", []pic{{"c.png", 1, 2}}, "v.mp4"), " ")
	for _, w := range []string{"crop=1080:1920", "gblur", "scale=1080:-2", "overlay=0:", "[v0][1:v]overlay=0:0:enable='between(t,1.000,2.000)'"} {
		if !strings.Contains(a, w) {
			t.Errorf("vertical: %s lacks %s", a, w)
		}
	}
	a = strings.Join(verticalArgs("all.mp4", 1440, 900, "0x0b1020", nil, "v.mp4"), " ")
	if !strings.Contains(a, "color=c=0x0b1020:s=1080x1920") || strings.Contains(a, "gblur") || !strings.Contains(a, "shortest=1") {
		t.Errorf("vertical over a colour: %s", a)
	}
	for in, want := range map[string]string{"blur": "", "": "", "theme": "theme", "color #0B1020": "0x0b1020", "color #fff": "0xffffff"} {
		if got, err := ParseVerticalFill(in); err != nil || got != want {
			t.Errorf("vertical-fill %q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"red", "color red", "color #12345"} {
		if _, err := ParseVerticalFill(bad); err == nil {
			t.Errorf("vertical-fill %q: no error", bad)
		}
	}
	if f := themeFill(""); !strings.HasPrefix(f, "0x") || len(f) != 8 {
		t.Errorf("studio's background: %q", f)
	}
	cuts := highlightCuts([]*clip{{Name: "a", Seconds: 20, Hi: &span{2, 6}}, {Name: "b", Seconds: 4}}, "/w", 60)
	if len(cuts) != 2 || cuts[0].From != 1.6 || cuts[0].To != 6.5 || cuts[1].From != 0.5 || cuts[1].To != 4 || cuts[0].Clip != "/w/a.mp4" {
		t.Errorf("highlight cuts: %+v", cuts)
	}
	if cuts := highlightCuts([]*clip{{Name: "a", Seconds: 90, Hi: &span{1, 80}}}, "/w", 60); cuts[0].To-cuts[0].From > 60 {
		t.Errorf("the highlight is kept near a minute: %+v", cuts)
	}
	a = strings.Join(highlightArgs(cuts, &part{Card: "i.png", Hold: 3.5}, nil, 1440, 900, "h.mp4"), " ")
	if !strings.Contains(a, "-ss 1.600") || !strings.Contains(a, "concat=n=3:v=1:a=1") || !strings.Contains(a, "fade=t=in") {
		t.Errorf("highlight: %s", a)
	}
	if h := thirdHTML("", "Ana <Ruiz>", "Ops"); !strings.Contains(h, "Ana &lt;Ruiz&gt;") || !strings.Contains(h, "<span>Ops</span>") {
		t.Errorf("lower third: %s", h)
	}
	if h := brandCardHTML("forest", "", "Harbor", "Ship", "https://harbor.example", true); !strings.Contains(h, ">harbor.example<") || !strings.Contains(h, "<h1>Harbor</h1>") {
		t.Errorf("outro card: %s", h)
	}
}
