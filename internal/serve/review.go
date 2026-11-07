package serve

// Review links: `si serve --review` (or `si play --review`) opens a second,
// keyed listener on the network that serves the decks and documents read-only
// with a comment box: a reviewer picks a slide or a section and leaves a note
// (name + text). Comments are JSON next to the sources, in .si/review/ of the
// served folder; the author reads, replies and resolves them on loopback (the
// Studio's Review panel, which follows them live over SSE). Reviewers never
// reach anything that writes sources, runs a shell or moves the presenter.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Comment is one review note on a page (a deck or a document).
type Comment struct {
	ID       string  `json:"id"`
	Page     string  `json:"page"`            // the source, relative to the served folder (talk.md)
	Anchor   string  `json:"anchor"`          // slide:3, section:retry-budget, page; /b4-1a2b3c pins a block in it
	Quote    string  `json:"quote,omitempty"` // the start of the pinned block's text
	Name     string  `json:"name"`
	Text     string  `json:"text"`
	Time     string  `json:"time"` // RFC 3339
	Resolved bool    `json:"resolved,omitempty"`
	Replies  []Reply `json:"replies,omitempty"`
}

// Reply is an answer under a comment.
type Reply struct {
	Name string `json:"name"`
	Text string `json:"text"`
	Time string `json:"time"`
}

const (
	reviewNameMax = 60
	reviewTextMax = 2000
	reviewPerPage = 500
)

// anchorRE is where a comment sits: a slide, a section or the page, then
// optionally one block in it (a paragraph, a list item, code, a figure, a
// heading): /b<index>-<hash of its text>, so a pin survives small edits
// around it and falls back to the slide or section when its block changes.
var anchorRE = regexp.MustCompile(`^(slide:\d{1,4}|section:[A-Za-z0-9_.-]{1,80}|page)(/b\d{1,4}-[0-9a-f]{6})?$`)

const reviewQuoteMax = 120

type reviews struct {
	dir  string // the served folder
	mu   sync.Mutex
	subs map[chan string]bool
}

func newReviews(dir string) *reviews { return &reviews{dir: dir, subs: map[chan string]bool{}} }

// page validates a page name: an existing .md under the folder, no dot paths.
func (rv *reviews) page(p string) (string, bool) {
	p = path.Clean("/" + strings.ReplaceAll(p, "\\", "/"))[1:]
	if p == "" || !strings.HasSuffix(p, ".md") || strings.HasPrefix(p, ".") || strings.Contains(p, "/.") {
		return "", false
	}
	if st, err := os.Stat(filepath.Join(rv.dir, filepath.FromSlash(p))); err != nil || st.IsDir() {
		return "", false
	}
	return p, true
}

func (rv *reviews) file(page string) string {
	return filepath.Join(rv.dir, ".si", "review", strings.ReplaceAll(page, "/", "__")+".json")
}

func (rv *reviews) load(page string) []Comment {
	var cs []Comment
	if raw, err := os.ReadFile(rv.file(page)); err == nil {
		json.Unmarshal(raw, &cs)
	}
	return cs
}

func (rv *reviews) save(page string, cs []Comment) error {
	f := rv.file(page)
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(cs, "", "  ")
	tmp := f + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f)
}

// change runs fn on a page's comments under the lock, saves and tells every
// listener.
func (rv *reviews) change(page string, fn func([]Comment) ([]Comment, error)) error {
	rv.mu.Lock()
	cs, err := fn(rv.load(page))
	if err == nil {
		err = rv.save(page, cs)
	}
	subs := make([]chan string, 0, len(rv.subs))
	for c := range rv.subs {
		subs = append(subs, c)
	}
	rv.mu.Unlock()
	if err == nil {
		for _, c := range subs {
			select {
			case c <- page:
			default:
			}
		}
	}
	return err
}

func clip(s string, max int) (string, bool) {
	s = strings.TrimSpace(s)
	return s, s != "" && utf8.RuneCountInString(s) <= max
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func readReq(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSONTo(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// strictOrigin refuses a POST another site's page could send: it must carry
// an Origin, and the Origin must be this listener.
func strictOrigin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o == "" || o != "http://"+r.Host {
			http.Error(w, "si review: same-origin requests only", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func (rv *reviews) list(w http.ResponseWriter, r *http.Request) {
	page, ok := rv.page(r.URL.Query().Get("page"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	cs := rv.load(page)
	if cs == nil {
		cs = []Comment{}
	}
	writeJSONTo(w, map[string]any{"page": page, "comments": cs})
}

func (rv *reviews) add(w http.ResponseWriter, r *http.Request) {
	var req struct{ Page, Anchor, Name, Text, Quote string }
	if !readReq(w, r, &req) {
		return
	}
	page, ok := rv.page(req.Page)
	name, okName := clip(req.Name, reviewNameMax)
	text, okText := clip(req.Text, reviewTextMax)
	quote := []rune(strings.TrimSpace(req.Quote))
	quote = quote[:min(len(quote), reviewQuoteMax)]
	if !ok || !anchorRE.MatchString(req.Anchor) || !okName || !okText {
		http.Error(w, fmt.Sprintf("si review: a comment needs a page, a place, a name (up to %d characters) and a text (up to %d)", reviewNameMax, reviewTextMax), http.StatusBadRequest)
		return
	}
	c := Comment{ID: newID(), Page: page, Anchor: req.Anchor, Quote: string(quote), Name: name, Text: text, Time: time.Now().UTC().Format(time.RFC3339)}
	err := rv.change(page, func(cs []Comment) ([]Comment, error) {
		if len(cs) >= reviewPerPage {
			return cs, fmt.Errorf("si review: this page has %d comments, the most it takes", reviewPerPage)
		}
		return append(cs, c), nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInsufficientStorage)
		return
	}
	writeJSONTo(w, c)
}

func (rv *reviews) reply(w http.ResponseWriter, r *http.Request) {
	var req struct{ Page, ID, Name, Text string }
	if !readReq(w, r, &req) {
		return
	}
	page, ok := rv.page(req.Page)
	name, okName := clip(req.Name, reviewNameMax)
	text, okText := clip(req.Text, reviewTextMax)
	if !ok || !okName || !okText {
		http.Error(w, "si review: a reply needs a name and a text", http.StatusBadRequest)
		return
	}
	rv.edit(w, page, req.ID, func(c *Comment) {
		c.Replies = append(c.Replies, Reply{Name: name, Text: text, Time: time.Now().UTC().Format(time.RFC3339)})
	})
}

func (rv *reviews) resolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Page, ID string
		Resolved bool
	}
	if !readReq(w, r, &req) {
		return
	}
	page, ok := rv.page(req.Page)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rv.edit(w, page, req.ID, func(c *Comment) { c.Resolved = req.Resolved })
}

func (rv *reviews) edit(w http.ResponseWriter, page, id string, fn func(*Comment)) {
	var out *Comment
	err := rv.change(page, func(cs []Comment) ([]Comment, error) {
		for i := range cs {
			if cs[i].ID == id {
				if len(cs[i].Replies) >= reviewPerPage {
					return cs, fmt.Errorf("too many replies")
				}
				fn(&cs[i])
				out = &cs[i]
				return cs, nil
			}
		}
		return cs, fmt.Errorf("no comment %q", id)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSONTo(w, out)
}

// stamps is the modification time of every page's comments file.
func (rv *reviews) stamps() map[string]time.Time {
	out := map[string]time.Time{}
	es, _ := os.ReadDir(filepath.Join(rv.dir, ".si", "review"))
	for _, e := range es {
		if strings.HasSuffix(e.Name(), ".json") {
			if fi, err := e.Info(); err == nil {
				out[e.Name()] = fi.ModTime()
			}
		}
	}
	return out
}

// events streams "review" events (the page that changed) to the author and reviewers.
func (rv *reviews) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan string, 16)
	rv.mu.Lock()
	rv.subs[ch] = true
	rv.mu.Unlock()
	defer func() {
		rv.mu.Lock()
		delete(rv.subs, ch)
		rv.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, "retry: 2000\n: review\n\n")
	fl.Flush()
	// another si (the review listener of si serve, while this is the Studio)
	// writes the same files: their changes reach this stream too
	seen := rv.stamps()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			now := rv.stamps()
			for f, t := range now {
				if !seen[f].Equal(t) {
					select {
					case ch <- strings.ReplaceAll(strings.TrimSuffix(f, ".json"), "__", "/"):
					default:
					}
				}
			}
			seen = now
		case p := <-ch:
			b, _ := json.Marshal(p)
			fmt.Fprintf(w, "event: review\ndata: %s\n\n", b)
			fl.Flush()
		}
	}
}

// authorRoutes are the loopback routes (si play / si serve / the Studio): read,
// follow, reply and resolve. A comment's own add stays on the review listener.
func (s *Server) authorRoutes(mux *http.ServeMux) {
	rv := s.reviews()
	mux.HandleFunc("GET /_si/review", localOnly(rv.list))
	mux.HandleFunc("GET /_si/review/events", localOnly(rv.events))
	mux.HandleFunc("POST /_si/review/reply", localOnly(rv.reply))
	mux.HandleFunc("POST /_si/review/resolve", localOnly(rv.resolve))
}

func (s *Server) reviews() *reviews {
	s.revOnce.Do(func() { s.rev = newReviews(s.dir) })
	return s.rev
}

// ReviewHandler is the keyed review listener: pages read-only with the
// comment box, their files (no dot paths), live reload and the review API.
func (s *Server) ReviewHandler(key string) http.Handler {
	rv := s.reviews()
	mux := http.NewServeMux()
	mux.HandleFunc("/_si/events", s.events)
	mux.HandleFunc("GET /_si/review", rv.list)
	mux.HandleFunc("GET /_si/review/events", rv.events)
	mux.HandleFunc("POST /_si/review", strictOrigin(rv.add))
	mux.HandleFunc("POST /_si/review/reply", strictOrigin(rv.reply))
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		switch {
		case strings.Contains(p, "/.") || strings.HasPrefix(p, "/_si/"):
			http.NotFound(w, r) // no dotfiles, and none of play's endpoints (position, terminal, callout)
		case p == "/":
			s.index(w)
		case strings.HasSuffix(p, ".html"):
			md := s.source(strings.TrimSuffix(p, ".html"))
			if md == "" {
				http.NotFound(w, r)
				return
			}
			rel, _ := filepath.Rel(s.dir, md)
			rec := httptest.NewRecorder()
			s.deck(rec, md, false)
			body := rec.Body.String()
			tag := `<script>window.SI_REVIEW=` + jsonString(filepath.ToSlash(rel)) + `;</script>` + "\n<script>\n" + ReviewJS + "</script>\n"
			if i := strings.LastIndex(body, "</body>"); i >= 0 {
				body = body[:i] + tag + body[i:]
			}
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			fmt.Fprint(w, body)
		default:
			static.ServeHTTP(w, r)
		}
	})
	return remoteGate(key, mux)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return strings.ReplaceAll(string(b), "</", `<\/`)
}

// ReviewJS is the comment box of a review page: a button with the count, a
// panel with the comments of the current slide (or section, or the page) and
// a form; the name is remembered in this browser.
const ReviewJS = `(function(){
var page=window.SI_REVIEW,box,list,form,btn,all=[],pin=null,pins=[];
var BLOCK='h1,h2,h3,p,li,pre,figure,blockquote,table';
function scope(){return document.querySelector('.si-slide.si-current')||document.querySelector('.si-doc-body,main')||document.body}
function blocks(){return Array.prototype.filter.call(scope().querySelectorAll(BLOCK),function(b){return !b.closest('.rv-box,.si-notes,.si-points')&&!(b.tagName==='P'&&b.closest('li'))})}
function hash(t){var h=5381;t=(t||'').replace(/\s+/g,' ').trim();for(var i=0;i<t.length;i++)h=((h*33)^t.charCodeAt(i))>>>0;return ('00000'+(h&0xffffff).toString(16)).slice(-6)}
function where(b){var bs=blocks(),i=bs.indexOf(b);return i<0?'':'/b'+i+'-'+hash(b.textContent)}
function blockOf(a){var m=/\/b(\d+)-([0-9a-f]{6})$/.exec(a);if(!m)return null;var b=blocks()[+m[1]];return b&&hash(b.textContent)===m[2]?b:null}
function base(a){return a.split('/')[0]}
function anchor0(){var m=/^#(\d+)/.exec(location.hash);if(document.querySelector('.si-deck'))return 'slide:'+(m?m[1]:'1');
var hs=document.querySelectorAll('h2[id],h3[id]'),cur='page';for(var i=0;i<hs.length;i++){if(hs[i].getBoundingClientRect().top<120)cur='section:'+hs[i].id;}return cur;}
function anchor(){return anchor0()+(pin?where(pin):'')}
function label(a){a=base(a);return a==='page'?'this page':a.indexOf('slide:')===0?'slide '+a.slice(6):'section '+a.slice(8).replace(/-/g,' ');}
function el(t,c,x){var e=document.createElement(t);if(c)e.className=c;if(x!=null)e.textContent=x;return e;}
function drawPins(){pins.forEach(function(p){p.remove()});pins=[];var a0=anchor0(),n=0;
all.forEach(function(c){if(c.resolved||base(c.anchor)!==a0)return;n++;var b=blockOf(c.anchor);if(!b)return;var r=b.getBoundingClientRect(),p=el('span','rv-pin',String(n));
p.title=c.name+': '+c.text;p.style.top=(r.top+window.scrollY)+'px';p.style.left=Math.max(4,r.left+window.scrollX-26)+'px';document.body.appendChild(p);pins.push(p)})}
function draw(){var a0=anchor0(),mine=all.filter(function(c){return base(c.anchor)===a0});btn.textContent='Comments '+all.filter(function(c){return !c.resolved}).length;
if(pin&&!document.body.contains(pin))pin=null;
box.querySelector('.rv-where').textContent='On '+(pin?'the '+({LI:'list item',PRE:'code',FIGURE:'figure',TABLE:'table',BLOCKQUOTE:'quote',H1:'heading',H2:'heading',H3:'heading'}[pin.tagName]||'paragraph')+' “'+pin.textContent.replace(/\s+/g,' ').trim().slice(0,48)+'…”':label(a0));
unpin.hidden=!pin;list.textContent='';drawPins();
if(!mine.length)list.appendChild(el('p','rv-none','No comments here yet.'));
mine.forEach(function(c){var li=el('div','rv-c'+(c.resolved?' rv-done':''));if(c.quote)li.appendChild(el('q','rv-q',c.quote));li.appendChild(el('b',null,c.name));li.appendChild(el('span','rv-t',' '+c.time.slice(0,16).replace('T',' ')));li.appendChild(el('p',null,c.text));
(c.replies||[]).forEach(function(r){var re=el('div','rv-r');re.appendChild(el('b',null,r.name));re.appendChild(el('p',null,r.text));li.appendChild(re)});list.appendChild(li)});}
function load(){fetch('/_si/review?page='+encodeURIComponent(page)).then(function(r){return r.json()}).then(function(j){all=j.comments||[];draw()}).catch(function(){})}
var css=el('style',null,'.rv-btn{position:fixed;right:1em;bottom:1em;z-index:2147483000;font:600 14px system-ui,sans-serif;padding:.6em 1em;border-radius:999px;border:0;background:#1d4ed8;color:#fff;box-shadow:0 6px 20px rgba(0,0,0,.25);cursor:pointer}'+
'.rv-box{position:fixed;right:1em;bottom:4em;z-index:2147483000;width:min(22em,calc(100vw - 2em));max-height:70vh;overflow:auto;background:#fff;color:#111;border-radius:12px;box-shadow:0 12px 40px rgba(0,0,0,.3);padding:1em;font:14px/1.45 system-ui,sans-serif}'+
'.rv-box[hidden]{display:none}.rv-box h2{font-size:15px;margin:0 0 .5em}.rv-c{border-top:1px solid #e5e7eb;padding:.5em 0}.rv-done{opacity:.55}.rv-c p,.rv-r p{margin:.2em 0}.rv-t,.rv-none{color:#555;font-size:12px}.rv-r{margin-left:1em;border-left:2px solid #e5e7eb;padding-left:.5em}'+
'.rv-pin{position:absolute;z-index:2147482999;width:20px;height:20px;border-radius:50%;background:#f59e0b;color:#111;font:700 11px/20px system-ui,sans-serif;text-align:center;box-shadow:0 2px 6px rgba(0,0,0,.3)}'+
'.rv-picked{outline:3px solid #f59e0b!important;outline-offset:3px}.rv-q{display:block;color:#555;font-size:12px;border-left:3px solid #f59e0b;padding-left:.4em;margin:.2em 0}.rv-unpin{font:inherit;font-size:12px;border:0;background:none;color:#1d4ed8;cursor:pointer;padding:0}'+
'.rv-box input,.rv-box textarea{width:100%;box-sizing:border-box;font:inherit;margin:.25em 0;padding:.4em;border:1px solid #9ca3af;border-radius:6px}.rv-box button[type=submit]{font:inherit;font-weight:600;padding:.4em .9em;border-radius:6px;border:0;background:#1d4ed8;color:#fff;cursor:pointer}.rv-err{color:#b91c1c;font-size:12px}');
document.head.appendChild(css);
btn=el('button','rv-btn','Comments');btn.type='button';btn.setAttribute('aria-expanded','false');
box=el('section','rv-box');box.hidden=true;box.setAttribute('aria-label','Review comments');box.appendChild(el('h2','rv-where'));
var unpin=el('button','rv-unpin','Comment on the whole '+(document.querySelector('.si-deck')?'slide':'section')+' instead');unpin.type='button';unpin.hidden=true;box.appendChild(unpin);
box.appendChild(el('p','rv-none','Click a paragraph, a list item, code or a figure to pin your comment to it.'));list=el('div');box.appendChild(list);
function pick(b){if(pin)pin.classList.remove('rv-picked');pin=b;if(pin)pin.classList.add('rv-picked');draw()}
unpin.addEventListener('click',function(){pick(null)});
document.addEventListener('click',function(e){if(box.hidden||e.target.closest('.rv-box,.rv-btn,a,button'))return;var b=e.target.closest(BLOCK);if(b&&scope().contains(b)){e.preventDefault();e.stopPropagation();pick(b)}},true);
form=el('form');var nm=el('input');nm.placeholder='Your name';nm.setAttribute('aria-label','Your name');nm.maxLength=60;try{nm.value=localStorage.getItem('si-review-name')||''}catch(e){}
var tx=el('textarea');tx.rows=3;tx.placeholder='Your comment';tx.setAttribute('aria-label','Your comment');tx.maxLength=2000;var sb=el('button',null,'Comment');sb.type='submit';var er=el('p','rv-err');
form.appendChild(nm);form.appendChild(tx);form.appendChild(sb);form.appendChild(er);box.appendChild(form);
form.addEventListener('submit',function(e){e.preventDefault();er.textContent='';try{localStorage.setItem('si-review-name',nm.value)}catch(x){}
fetch('/_si/review',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({page:page,anchor:anchor(),quote:pin?pin.textContent.replace(/\s+/g,' ').trim().slice(0,120):'',name:nm.value,text:tx.value})}).then(function(r){if(!r.ok)return r.text().then(function(t){throw new Error(t)});tx.value='';pick(null);load()}).catch(function(x){er.textContent=String(x.message||x)})});
btn.addEventListener('click',function(){box.hidden=!box.hidden;btn.setAttribute('aria-expanded',String(!box.hidden));if(!box.hidden){draw();tx.focus()}});
['keydown','keyup'].forEach(function(t){box.addEventListener(t,function(e){e.stopPropagation()})});
document.body.appendChild(box);document.body.appendChild(btn);
window.addEventListener('hashchange',function(){pick(null)});window.addEventListener('resize',drawPins);window.addEventListener('scroll',function(){if(!box.hidden)draw()},{passive:true});
try{var es=new EventSource('/_si/review/events');es.addEventListener('review',function(e){if(JSON.parse(e.data)===page)load()})}catch(e){}
load();})();
`
