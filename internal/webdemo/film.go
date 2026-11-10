package webdemo

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

//go:embed overlay.js
var overlayJS string

// take is the filming of one feature: the screencast frames, the voice lines
// placed on its clock, the notes on screen.
type take struct {
	dir    string
	t0     time.Time
	mu     sync.Mutex
	frames []frame
	wg     sync.WaitGroup
	on     bool
	voice  []placed
	caps   []caption
	cursor *[2]float64
	notes  []*liveNote
	noteN  int
	block  time.Time // when the block now running started
	length float64   // how long its narration needs
	last   time.Time // when the last step ended
}

// liveNote is a note on screen: until (zero) the end of its block, or a time.
type liveNote struct {
	id    int
	step  *Step
	until time.Time
}

// tokens are si's studio theme (light) as custom properties.
func tokens() string {
	t := assets.Themes["studio"]
	var keys []string
	vals := map[string]string{}
	for _, m := range []map[string]string{t.Base, t.Light} {
		for k, v := range m {
			if _, ok := vals[k]; !ok {
				keys = append(keys, k)
			}
			vals[k] = v
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s:%s;", k, vals[k])
	}
	return b.String()
}

func overlayCSS() string {
	return `:host{` + tokens() + `}
*{box-sizing:border-box}
.cursor{position:fixed;left:0;top:0;width:28px;height:28px;filter:drop-shadow(0 2px 3px rgb(0 0 0/.35));will-change:transform}
.cursor[hidden]{display:none}
.ripple{position:fixed;width:44px;height:44px;margin:-22px 0 0 -22px;border-radius:50%;border:3px solid var(--si-accent,#2563eb);background:color-mix(in srgb,var(--si-accent,#2563eb) 22%,transparent);animation:rip .6s ease-out forwards}
@keyframes rip{from{transform:scale(.3);opacity:1}to{transform:scale(1.6);opacity:0}}
.note{position:fixed;max-width:min(22em,40vw);padding:.6em .85em;border-radius:var(--si-radius,.55em);background:var(--si-accent,#2563eb);color:var(--si-on-accent,#fff);
font:600 15px/1.35 var(--si-font-body,system-ui,sans-serif);box-shadow:0 .5em 1.4em rgb(0 0 0/.25);animation:pop .3s ease-out}
.note[hidden],.ring[hidden]{display:none}
.note::before{content:"";position:absolute;width:12px;height:12px;background:inherit;transform:rotate(45deg)}
.note.at-right::before{left:-6px;top:calc(50% - 6px)}.note.at-left::before{right:-6px;top:calc(50% - 6px)}
.note.at-bottom::before{top:-6px;left:calc(50% - 6px)}.note.at-top::before{bottom:-6px;left:calc(50% - 6px)}
.note.free::before{display:none}
@keyframes pop{from{opacity:0;transform:scale(.92)}to{opacity:1;transform:none}}
.ring{position:fixed;border:3px solid var(--si-accent,#2563eb);border-radius:calc(var(--si-radius,.55em) + 2px);box-shadow:0 0 0 4px color-mix(in srgb,var(--si-accent,#2563eb) 25%,transparent)}
.blur{position:fixed;backdrop-filter:blur(7px);-webkit-backdrop-filter:blur(7px);background:color-mix(in srgb,var(--si-surface,#fff) 35%,transparent);border-radius:4px}
`
}

// overlaySource is the script injected into every document of the take.
func overlaySource() string {
	css, _ := json.Marshal(overlayCSS())
	return overlayJS + "(" + string(css) + ");"
}

// startTake begins filming the page: the overlay first, then the screencast.
func (r *runner) startTake() error {
	dir, err := os.MkdirTemp("", "si-take-")
	if err != nil {
		return err
	}
	tk := &take{dir: dir}
	if err := r.c.Call("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": overlaySource()}, nil); err != nil {
		return err
	}
	r.mu.Lock()
	r.tk = tk
	r.mu.Unlock()
	return nil
}

// roll starts the screencast (the clip's 0 s).
func (r *runner) roll() error {
	tk := r.tk
	r.claim()
	r.mu.Lock()
	tk.t0, tk.on, tk.last = time.Now(), true, time.Now()
	r.mu.Unlock()
	return r.c.Call("Page.startScreencast", map[string]any{"format": "jpeg", "quality": 90, "maxWidth": r.s.W, "maxHeight": r.s.H, "everyNthFrame": 1}, nil)
}

// onFrame keeps a screencast frame (called on the reader goroutine: it hands
// the work to another, which acks so the next frame comes).
func (r *runner) onFrame(params json.RawMessage) {
	tk := r.tk
	if tk == nil || !tk.on {
		return
	}
	at := time.Since(tk.t0).Seconds()
	session := r.session
	tk.wg.Add(1)
	go func() {
		defer tk.wg.Done()
		var p struct {
			Data      string
			SessionID int
		}
		if json.Unmarshal(params, &p) != nil {
			return
		}
		r.c.CallIn(session, "Page.screencastFrameAck", map[string]any{"sessionId": p.SessionID}, nil)
		raw, err := base64.StdEncoding.DecodeString(p.Data)
		if err != nil {
			return
		}
		tk.mu.Lock()
		name := filepath.Join(tk.dir, fmt.Sprintf("f%06d.jpg", len(tk.frames)))
		tk.frames = append(tk.frames, frame{name, at})
		tk.mu.Unlock()
		os.WriteFile(name, raw, 0o644)
	}()
}

// cut stops the screencast; it returns the clip's length.
func (r *runner) cut() float64 {
	tk := r.tk
	d := time.Since(tk.t0).Seconds()
	r.c.Call("Page.stopScreencast", nil, nil)
	r.mu.Lock()
	tk.on = false
	r.mu.Unlock()
	tk.wg.Wait()
	return d
}

func (r *runner) js(code string, a ...any) {
	if r.tk != nil {
		r.c.Eval(fmt.Sprintf(code, a...))
	}
}

// claim restores the cursor and the notes on a new document (after a
// navigation); the overlay script marks each document fresh.
func (r *runner) claim() {
	tk := r.tk
	if tk == nil {
		return
	}
	v, err := r.c.Eval(`(function(d){if(!d)return 'none';if(d.fresh){d.fresh=false;return 'new'}return 'ok'})(window.__siDemo)`)
	if err != nil || string(v) != `"new"` {
		return
	}
	if tk.cursor != nil {
		r.js(`__siDemo.move(%g,%g,0)`, tk.cursor[0], tk.cursor[1])
	}
	for _, n := range tk.notes {
		r.showNote(n, true)
	}
}

// glide moves the drawn cursor to x, y at a hand's pace and waits for it.
func (r *runner) glide(x, y float64) {
	tk := r.tk
	if tk == nil {
		return
	}
	r.claim()
	ms := 0.0
	if tk.cursor != nil {
		dist := math.Hypot(x-tk.cursor[0], y-tk.cursor[1])
		ms = max(300, min(900, 250+dist*0.6))
	}
	tk.cursor = &[2]float64{x, y}
	r.js(`__siDemo.move(%g,%g,%g)`, x, y, ms)
	time.Sleep(time.Duration(ms+120) * time.Millisecond)
}

func (r *runner) ripple(x, y float64) { r.js(`__siDemo.ripple(%g,%g)`, x, y) }

// note puts a note on screen (anchored at the element the step found).
func (r *runner) note(st *Step) {
	tk := r.tk
	if tk == nil {
		return
	}
	tk.noteN++
	n := &liveNote{id: tk.noteN, step: st}
	if st.For > 0 {
		n.until = time.Now().Add(st.For)
	}
	tk.notes = append(tk.notes, n)
	r.showNote(n, false)
}

// showNote draws n; again: on a new document, where its element must be found
// anew.
func (r *runner) showNote(n *liveNote, again bool) {
	ms := 0.0
	if !n.until.IsZero() {
		ms = time.Until(n.until).Seconds() * 1000
		if ms <= 0 {
			return
		}
	}
	anchored := n.step.Target != (Locator{})
	if anchored && again {
		if f, err := r.locate(n.step.Target, "any", false); err != nil || !f.Found {
			return
		}
	}
	text := n.step.Value.Text
	if n.step.Value.Var != "" {
		text = "$" + n.step.Value.Var
	}
	t, _ := json.Marshal(text)
	r.js(`__siDemo.note(%d,%s,%q,%g,%t)`, n.id, t, n.step.Pos, ms, anchored)
}

// expire forgets notes whose time is up (the page removes them itself).
func (r *runner) expire() {
	tk := r.tk
	keep := tk.notes[:0]
	for _, n := range tk.notes {
		if n.until.IsZero() || time.Now().Before(n.until) {
			keep = append(keep, n)
		}
	}
	tk.notes = keep
}

// beginBlock places the block's voice on the clip's clock: the block lasts
// as long as its narration, or its steps when they need longer.
func (r *runner) beginBlock(b *Block) {
	tk := r.tk
	tk.block, tk.length = time.Now(), 0
	if l, ok := r.lines[b]; ok {
		at := time.Since(tk.t0).Seconds()
		tk.voice = append(tk.voice, placed{l.WAV, at})
		tk.caps = append(tk.caps, captions(b.Say, at, l.Seconds)...)
		tk.length = l.Seconds + 0.4
	}
}

// pace holds step i of n until its share of the narration, and a breath
// after the step before.
func (r *runner) pace(i, n int) {
	tk := r.tk
	next := tk.last.Add(350 * time.Millisecond)
	if tk.length > 0 {
		if t := tk.block.Add(time.Duration(tk.length * float64(i) / float64(n) * float64(time.Second))); t.After(next) {
			next = t
		}
	}
	time.Sleep(time.Until(next))
}

// endBlock waits for the narration to finish, then takes the block's notes
// down.
func (r *runner) endBlock() {
	tk := r.tk
	time.Sleep(time.Until(tk.block.Add(time.Duration(tk.length * float64(time.Second)))))
	keep := tk.notes[:0]
	for _, n := range tk.notes {
		if n.until.IsZero() {
			r.js(`__siDemo.unnote(%d)`, n.id)
		} else {
			keep = append(keep, n)
		}
	}
	tk.notes = keep
	r.expire()
}

// typeSlowly types text at a person's pace.
func (r *runner) typeSlowly(text string) error {
	for _, ch := range text {
		if err := r.c.Call("Input.insertText", map[string]any{"text": string(ch)}, nil); err != nil {
			return err
		}
		time.Sleep(55 * time.Millisecond)
	}
	return nil
}

// picture renders HTML at the viewport size: opaque (a title card) or
// transparent around what it draws (a caption).
func (r *runner) picture(page string, transparent bool) ([]byte, error) {
	color := map[string]any{"r": 255, "g": 255, "b": 255, "a": 1}
	if transparent {
		color["a"] = 0
	}
	if err := r.c.Call("Emulation.setDefaultBackgroundColorOverride", map[string]any{"color": color}, nil); err != nil {
		return nil, err
	}
	if err := r.navigate("data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(page))); err != nil {
		return nil, err
	}
	r.c.Eval(`document.fonts.ready.then(function(){return 1})`)
	return r.c.Screenshot()
}

func pageHTML(body, style string) string {
	return `<!doctype html><meta charset="utf-8"><style>:root{` + tokens() + `}html,body{margin:0;height:100%;background:transparent}` + style + `</style>` + body
}

// cardHTML is a feature's title card: the script's title, the feature, what
// it does.
func cardHTML(title, name, what string) string {
	if r := []rune(what); len(r) > 0 {
		what = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	return pageHTML(`<main><p class="k">`+html.EscapeString(title)+`</p><h1>`+html.EscapeString(name)+`</h1>`+
		map[bool]string{true: `<p class="w">` + html.EscapeString(what) + `</p>`, false: ""}[what != ""]+`</main>`,
		`body{background:var(--si-bg);color:var(--si-fg);font-family:var(--si-font-body);display:grid;place-items:center;container-type:inline-size}
main{width:min(80vw,60rem);padding:0 1rem}
.k{margin:0 0 1.2rem;color:var(--si-muted);font-size:clamp(14px,2.2vw,26px);letter-spacing:.04em}
h1{margin:0;font-family:var(--si-font-heading);font-weight:var(--si-heading-weight,760);letter-spacing:var(--si-heading-tracking,-.02em);font-size:clamp(28px,6vw,84px);line-height:1.05}
h1::after{content:"";display:block;width:2.4em;height:.12em;margin-top:.35em;background:var(--si-accent);border-radius:1em}
.w{margin:1.4rem 0 0;color:var(--si-fg);opacity:.86;font-size:clamp(16px,2.6vw,32px);line-height:1.4;max-width:34em}`)
}

// captionHTML is one caption at the bottom of the frame, two lines at most.
func captionHTML(text string) string {
	var lines []string
	for _, l := range wrap(text) {
		lines = append(lines, html.EscapeString(l))
	}
	return pageHTML(`<div><span>`+strings.Join(lines, "<br>")+`</span></div>`,
		`div{position:fixed;left:0;right:0;bottom:6vh;text-align:center;padding:0 5vw}
span{display:inline-block;padding:.35em .8em;border-radius:var(--si-radius,.55em);background:rgb(15 17 21/.82);color:#fff;
font:600 clamp(16px,3.1vw,40px)/1.35 var(--si-font-body,system-ui,sans-serif);box-shadow:0 .3em 1em rgb(0 0 0/.25)}`)
}

// clip is what a feature's take wrote.
type clip struct {
	Name    string // file stem: <feature slug>
	title   string // the feature's name
	Seconds float64
	Card    string // the title card (temporary)
	Voiced  bool
}

// finishTake renders the feature's card and captions, then writes its files:
// .video.mp4, .voice.wav/.m4a, .srt, .mp4.
func (r *runner) finishTake(f *Feature, d float64) (*clip, error) {
	tk := r.tk
	defer func() {
		r.mu.Lock()
		r.tk = nil
		r.mu.Unlock()
	}()
	stem := slug(f.Name)
	if stem == "" {
		stem = "feature"
	}
	out := func(ext string) string { return filepath.Join(r.o.Out, stem+ext) }
	cl := &clip{Name: stem, title: f.Name, Seconds: d, Voiced: len(tk.voice) > 0}
	for i := range tk.caps { // a caption never outlives the clip
		tk.caps[i].To = min(tk.caps[i].To, d)
	}
	var caps []caption
	for _, c := range tk.caps {
		if c.From < d {
			caps = append(caps, c)
		}
	}
	if r.o.Captions {
		for i := range caps {
			png, err := r.picture(captionHTML(caps[i].Text), true)
			if err != nil {
				return nil, err
			}
			caps[i].PNG = filepath.Join(tk.dir, fmt.Sprintf("cap%03d.png", i))
			if err := os.WriteFile(caps[i].PNG, png, 0o644); err != nil {
				return nil, err
			}
		}
	}
	card, err := r.picture(cardHTML(r.title(), f.Name, f.What), false)
	if err != nil {
		return nil, err
	}
	cl.Card = filepath.Join(r.o.Out, ".card-"+stem+".png")
	if err := os.WriteFile(cl.Card, card, 0o644); err != nil {
		return nil, err
	}
	seq, err := os.MkdirTemp(tk.dir, "seq")
	if err != nil {
		return nil, err
	}
	pattern, n, err := sequence(tk.frames, d, seq)
	if err != nil {
		return nil, err
	}
	cl.Seconds = float64(n) / fps
	if err := r.ffmpeg(videoArgs(pattern, caps, out(".video.mp4"))); err != nil {
		return nil, err
	}
	wav := out(".voice.wav")
	if err := mixWAV(wav, cl.Seconds, tk.voice); err != nil {
		return nil, err
	}
	if err := r.ffmpeg(muxArgs(out(".video.mp4"), wav, out(".mp4"))); err != nil {
		return nil, err
	}
	if cl.Voiced {
		if err := r.ffmpeg(m4aArgs(wav, out(".voice.m4a"))); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out(".srt"), []byte(srt(caps)), 0o644); err != nil {
			return nil, err
		}
	} else {
		os.Remove(wav)
	}
	os.RemoveAll(tk.dir)
	return cl, nil
}

func (r *runner) title() string {
	if r.s.Title != "" {
		return r.s.Title
	}
	return r.s.Name
}

func (r *runner) ffmpeg(args []string) error {
	if out, err := exec.Command(r.o.FFmpeg, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %v %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// joinAll writes all.mp4 (every clip after its title card, chapters inside)
// and chapters.txt.
func (r *runner) joinAll(clips []*clip) error {
	defer func() {
		for _, c := range clips {
			os.Remove(c.Card)
		}
	}()
	var parts []part
	for _, c := range clips {
		parts = append(parts, part{Name: c.title, Card: c.Card, Clip: filepath.Join(r.o.Out, c.Name+".mp4"), Seconds: c.Seconds})
	}
	cs := chapters(parts)
	meta := filepath.Join(r.o.Out, ".chapters.ffmeta")
	if err := os.WriteFile(meta, []byte(ffmetadata(r.title(), cs)), 0o644); err != nil {
		return err
	}
	defer os.Remove(meta)
	if err := r.ffmpeg(allArgs(parts, meta, filepath.Join(r.o.Out, "all.mp4"))); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.o.Out, "chapters.txt"), []byte(youtube(cs)), 0o644)
}
