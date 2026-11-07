// The deck page: head, styles, scripts, chrome, live reload and the scripts
// si play injects.

package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

var pageSizes = map[string]string{"16:9": "13.333in 7.5in", "16:10": "12in 7.5in", "4:3": "10in 7.5in", "1:1": "7.5in 7.5in", "9:16": "7.5in 13.333in"}

func (dr *deckRenderer) page(slides string, problems []deck.Problem, opt Options) []byte {
	d := dr.d
	m := d.Meta
	aspect := m.Aspect
	if aspect == "" {
		aspect = "16:9"
	}
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.LayoutCSS + dr.themeCSS + assets.ThemeLooksCSS)
	b.WriteString(dr.helperCSS())
	fmt.Fprintf(&b, "@page{size:%s;margin:0}\n", pageSizes[aspect])
	b.WriteString(dr.accentCSS())
	if dr.mindmap {
		b.WriteString(assets.MindmapCSS)
	}
	b.WriteString("</style>\n")
	dr.codewinCSS(&b)
	if dr.wmImg != "" || d.Meta.WatermarkOpacity != "" {
		var decls []string
		if dr.wmImg != "" {
			decls = append(decls, `--si-wm-img:url("`+dr.wmImg+`")`)
		}
		if d.Meta.WatermarkOpacity != "" {
			decls = append(decls, "--si-wm-opacity:"+d.Meta.WatermarkOpacity)
		}
		fmt.Fprintf(&b, "<style id=\"si-wm\">.si-deck{%s}</style>\n", strings.Join(decls, ";"))
	}
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body>\n")
	ratio := strings.Replace(aspect, ":", "/", 1)
	fmt.Fprintf(&b, "<main class=\"si-deck\" aria-label=\"%s\" style=\"--_aspect:%s\" data-si-tx=\"none\">\n", html.EscapeString(plain(d.Title())), ratio)
	slides, shared := shareMedia(slides)
	b.WriteString(slides)
	b.WriteString("</main>\n" + shared)
	b.WriteString(chromeHTML)
	dr.problemsBanner(&b, problems)
	plan := make([]float64, len(d.Slides))
	for i, s := range d.Slides {
		plan[i] = s.Minutes
	}
	data, _ := json.Marshal(map[string]any{
		"name": d.Name, "title": plain(d.Title()), "minutes": d.Minutes(), "plan": plan, "transition": m.Transition, "aspect": ratio,
	})
	fmt.Fprintf(&b, "<script type=\"application/json\" id=\"si-data\">%s</script>\n", data)
	keys := map[string]string{}
	feats := map[string]bool{}
	var timer any
	if opt.Config != nil {
		if opt.Config.Keys != nil {
			keys = opt.Config.Keys
		}
		if opt.Config.Features != nil {
			feats = opt.Config.Features
		}
		t := opt.Config.Timer
		if len(t.Times) > 0 || t.Unit != "" || t.Face != "" || len(dr.anims) > 0 {
			names := make([]string, 0, len(dr.anims))
			kinds := map[string]string{}
			for _, a := range dr.anims {
				names = append(names, a.name)
				kinds[a.name] = a.kind
			}
			timer = map[string]any{"time": t.Times, "unit": t.Unit, "face": t.Face, "animations": names, "animKinds": kinds}
		}
	}
	cfg, _ := json.Marshal(map[string]any{"keys": keys, "features": feats, "timer": timer, "terminal": opt.Terminal, "remote": opt.Remote, "pages": opt.Pages, "live": opt.LiveReload})
	fmt.Fprintf(&b, "<script>window.SI_CONFIG=%s;</script>\n", cfg)
	if opt.Remote {
		b.WriteString(RelayJS)
	}
	// Runbook cheat-sheet: play-only + loopback-only, like the terminal it rides
	// in. Never injected into a built deck (opt.Terminal is false for build).
	if opt.Terminal && dr.runbookJSON != "" {
		fmt.Fprintf(&b, "<script>window.SI_RUNBOOK=%s;</script>\n", strings.ReplaceAll(dr.runbookJSON, "</", `<\/`))
	}
	if opt.Terminal && dr.demosJSON != "" {
		fmt.Fprintf(&b, "<script>window.SI_DEMOS=%s;</script>\n", strings.ReplaceAll(dr.demosJSON, "</", `<\/`))
	}
	if len(dr.anims) > 0 {
		var js []string
		for _, a := range dr.anims {
			if a.kind == "js" {
				js = append(js, strconv.Quote(a.name)+":("+a.code+"\n)")
			} else {
				fmt.Fprintf(&b, "<template id=\"si-anim-svg-%s\">%s</template>\n", html.EscapeString(a.name), a.code)
			}
		}
		if len(js) > 0 {
			b.WriteString("<script>window.__siAnims={" + strings.Join(js, ",") + "};</script>\n")
		}
	}
	dr.mermaidScript(&b)
	presenter, _ := json.Marshal(assets.PresenterHTML)
	b.WriteString("<script>\nwindow.SI_PRESENTER = " + strings.ReplaceAll(string(presenter), "</", `<\/`) + ";\n")
	b.WriteString(assets.DeckJS)
	b.WriteString("</script>\n")
	if dr.mindmap {
		b.WriteString(dr.mindmapScripts())
	}
	dr.codewinJS(&b)
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

// head writes the page start up to <title> (shared by decks and labs) and
// returns the data-si-mode attribute pinned by front matter, if any.
func (dr *deckRenderer) head(b *bytes.Buffer) string {
	m := dr.d.Meta
	lang := m.Lang
	if lang == "" {
		lang = "en"
	}
	modeAttr := ""
	if m.Mode == "light" || m.Mode == "dark" {
		modeAttr = ` data-si-mode="` + m.Mode + `"`
	}
	fmt.Fprintf(b, "<!doctype html>\n<html lang=\"%s\" data-si-theme=\"%s\"%s>\n<head>\n", html.EscapeString(lang), dr.themeName, modeAttr)
	b.WriteString(`<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<meta name="generator" content="simple-ideas">
<link rel="icon" href="data:,">
`)
	title := plain(dr.d.Title())
	if g := dr.opt.Guide; g != nil && g.Meta.Title != "" && g.Meta.Title != title {
		title += " · " + g.Meta.Title
	}
	fmt.Fprintf(b, "<title>%s</title>\n", html.EscapeString(title))
	if desc := strings.TrimSpace(dr.d.Meta.Description); desc != "" {
		fmt.Fprintf(b, "<meta name=\"description\" content=\"%s\">\n", html.EscapeString(desc))
	}
	return modeAttr
}

// accentCSS is the front matter accent override and the fonts: rules, or "".
func (dr *deckRenderer) accentCSS() string {
	if a := dr.d.Meta.Accent; a != "" {
		return fmt.Sprintf("@layer si.theme{:root,:root:not([data-si-mode]),:root[data-si-mode=\"dark\"]{--si-accent:%s}}\n", a) + dr.fontsCSS
	}
	return dr.fontsCSS
}

// userAndMode writes the --css user layer and the script that restores the
// viewer's light/dark choice before first paint (a mode pinned in front
// matter wins).
func (dr *deckRenderer) userAndMode(b *bytes.Buffer, modeAttr string) {
	if css := UserCSS(dr.opt.UserCSS); css != "" {
		b.WriteString(`<style id="si-user">` + "\n" + css + "</style>\n")
	}
	if modeAttr == "" {
		b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	}
}

// problemsBanner lists the problems on the page (si play).
func (dr *deckRenderer) problemsBanner(b *bytes.Buffer, problems []deck.Problem) {
	if !dr.opt.ShowProblems || len(problems) == 0 {
		return
	}
	fmt.Fprintf(b, "<aside class=\"si-problems\"><b>%d problem(s) in this deck</b><ul>", len(problems))
	for _, p := range problems {
		fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(p.String()))
	}
	b.WriteString("</ul></aside>\n")
}

// mermaidScript loads Mermaid when the page has a mermaid fence.
func (dr *deckRenderer) mermaidScript(b *bytes.Buffer) {
	if !dr.mermaid {
		return
	}
	src := MermaidURL
	if dr.d.Meta.Mermaid != "" {
		src = dr.d.Meta.Mermaid
	}
	fmt.Fprintf(b, "<script src=\"%s\"></script>\n", html.EscapeString(src))
}

// reload adds the si play reload client (hot reload with --dev).
func (dr *deckRenderer) reload(b *bytes.Buffer) {
	if dr.opt.Dev {
		b.WriteString(DevReloadJS)
	} else if dr.opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
	if dr.opt.Dev || dr.opt.LiveReload {
		b.WriteString(PositionJS)
	}
}

// PositionJS passes a terminal teleprompter's commands (next, prev, go) to the
// deck: si play / si serve only, like the reload client, never in a built deck.
const PositionJS = `<script>window.SI_LIVE=true;(function(){if(!window.EventSource||/[?&]si-view=/.test(location.search))return;var es=new EventSource('/_si/position/events?page='+encodeURIComponent(location.pathname));es.addEventListener('cmd',function(e){try{if(window.SI&&SI.cmd)SI.cmd(JSON.parse(e.data))}catch(x){}});})();</script>` + "\n"

const chromeHTML = `<nav class="si-bar" aria-label="Deck controls">
<span class="si-controls">
<button type="button" data-si-action="prev" aria-label="Previous">&larr;</button><button type="button" data-si-action="next" aria-label="Next">&rarr;</button>
<button type="button" data-si-action="overview" title="Overview (o)">Overview</button><button type="button" data-si-action="presenter" title="Presenter teleprompter (p)">Presenter</button><button type="button" data-si-action="draw" title="Annotate (a)">Draw</button>
<button type="button" data-si-action="mode" title="Light / dark (d)">Theme</button><button type="button" data-si-action="fullscreen" title="Full screen (f)">Full screen</button><button type="button" data-si-action="help" title="Keys (?)">?</button>
</span>
<span class="si-counter" aria-live="polite"></span>
</nav>
<div class="si-progress"></div>
<div class="si-help" role="dialog" aria-label="Keyboard shortcuts"><div>
<table>
<tr><td>&rarr; &darr; Space PgDn Enter</td><td>next step, then next slide</td></tr>
<tr><td>&larr; &uarr; Shift+Space PgUp</td><td>previous step or slide</td></tr>
<tr><td>Home / End</td><td>first / last slide</td></tr>
<tr><td>o</td><td>overview grid; click a slide to open it</td></tr>
<tr><td>p</td><td>presenter teleprompter: notes, timer, next slide; drives this window</td></tr>
<tr><td>t</td><td>break timer: pick minutes, countdown to project during a break</td></tr>
<tr><td>/</td><td>find a slide: search titles and text, or type a number to jump</td></tr>
<tr><td>d</td><td>light / dark</td></tr>
<tr><td>a</td><td>draw: pen over the slide (c colour, e erase, z undo, x clear, Esc off)</td></tr>
<tr><td>f</td><td>full screen</td></tr>
<tr><td>swipe</td><td>next / previous on touch screens</td></tr>
<tr><td>#5 or #5.2</td><td>open slide 5 (at step 2)</td></tr>
<tr><td>print</td><td>one slide per page, every step shown</td></tr>
</table></div></div>
<div class="si-hud" role="dialog" aria-label="Break timer" aria-hidden="true"><div>
<div class="si-hud-time">0:00</div>
<svg class="si-hud-dial" viewBox="0 0 100 100" aria-hidden="true" fill="none" stroke="currentColor" hidden>
<circle cx="50" cy="50" r="44" stroke-width="3" opacity="0.18"></circle>
<circle cx="50" cy="50" r="44" stroke-width="3" stroke-linecap="round" transform="rotate(-90 50 50)"></circle>
<line x1="50" y1="50" x2="50" y2="9" stroke-width="2" stroke-linecap="round"></line>
<circle cx="50" cy="50" r="2.6" fill="currentColor" stroke="none"></circle>
<text x="50" y="50" fill="currentColor" stroke="none" text-anchor="middle" dominant-baseline="central">0:00</text>
</svg>
<div class="si-hud-presets"><button type="button" data-si-min="5">5</button><button type="button" data-si-min="10">10</button><button type="button" data-si-min="15">15</button><button type="button" data-si-min="20">20</button><button type="button" data-si-min="30">30</button><button type="button" data-si-min="45">45</button><button type="button" data-si-min="60">60</button></div>
<div class="si-hud-toggle"></div>
<p class="si-hud-hint">1-7 pick minutes · space pause · t / Esc close</p>
</div></div>
<div class="si-find" role="dialog" aria-label="Find slide" aria-hidden="true"><div>
<input class="si-find-input" type="text" autocomplete="off" spellcheck="false" placeholder="Search slides, or type a number to jump…" aria-label="Search slides or go to a number">
<ul class="si-find-list" role="listbox"></ul>
</div></div>
<svg class="si-draw" aria-hidden="true"></svg>
<div class="si-draw-bar" aria-label="Annotate">
<button type="button" class="si-swatch" data-si-pen="var(--si-accent)" title="Accent" style="--_c:var(--si-accent)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-crit)" title="Red" style="--_c:var(--si-crit)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-ok)" title="Green" style="--_c:var(--si-ok)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-fg)" title="Ink" style="--_c:var(--si-fg)"></button>
<span class="si-draw-sep" aria-hidden="true"></span><button type="button" class="si-brush" data-si-size="2" title="Thin" style="--_s:2px"></button><button type="button" class="si-brush" data-si-size="5" title="Medium" style="--_s:5px"></button><button type="button" class="si-brush" data-si-size="9" title="Thick" style="--_s:9px"></button>
<span class="si-draw-sep" aria-hidden="true"></span><button type="button" data-si-draw="erase" title="Eraser (e)">Erase</button><button type="button" data-si-draw="undo" title="Undo (z)">Undo</button><button type="button" data-si-draw="clear" title="Clear all (x)">Clear</button><button type="button" data-si-draw="done" title="Done (Esc)">Done</button>
</div>
`

// RelayJS defines window.SI_RELAY(), the `si play --remote` transport: a
// BroadcastChannel-shaped object ({postMessage, onmessage}) over the relay
// (POST /_si/remote/send, SSE /_si/remote/events). Play only, like
// LiveReloadJS; deck.js copies the tag into the remote teleprompter page.
const RelayJS = `<script id="si-relay">window.SI_RELAY=function(){var id=Math.random().toString(36).slice(2),c={onmessage:null};` +
	`var es=new EventSource('/_si/remote/events?c='+id);es.onmessage=function(e){try{if(c.onmessage)c.onmessage({data:JSON.parse(e.data)})}catch(x){}};` +
	`c.postMessage=function(m){fetch('/_si/remote/send?c='+id,{method:'POST',body:JSON.stringify(m),keepalive:true}).catch(function(){})};return c};</script>` + "\n"

// LiveReloadJS reloads the page when si play reports a change; the URL hash
// keeps the slide and step.
const LiveReloadJS = `<script>(function(){if(!window.EventSource||/[?&]si-view=thumb/.test(location.search))return;var es=new EventSource('/_si/events');es.addEventListener('reload',function(){location.reload()});})();</script>` + "\n"

// DevReloadJS is the si play --dev hot-reload client: on a change it fetches the
// re-rendered deck and morphs the slides in place (when the slide count is
// unchanged), so the current slide, step and overview are preserved with no
// full-page flash. It falls back to a full reload when the structure changes,
// and shows an overlay when the deck fails to render.
const DevReloadJS = `<script>(function(){
  if(!window.EventSource||/[?&]si-view=thumb/.test(location.search))return;
  var es=new EventSource('/_si/events');
  es.addEventListener('reload',hot);
  function overlay(msg){
    var o=document.getElementById('si-dev-error');
    if(!msg){if(o)o.remove();return;}
    if(!o){o=document.createElement('div');o.id='si-dev-error';document.body.appendChild(o);}
    o.textContent=msg;
  }
  var NAV=/\bsi-(current|past|future|sel)\b/g;
  function hot(){
    fetch(location.href,{cache:'no-store'}).then(function(r){
      return r.text().then(function(t){return {ok:r.ok,status:r.status,t:t};});
    }).then(function(res){
      if(!res.ok){overlay('si play '+res.status+'\n'+res.t.replace(/<[^>]*>/g,' ').slice(0,1500).trim());return;}
      overlay(null);
      var doc=new DOMParser().parseFromString(res.t,'text/html');
      var nd=doc.querySelector('.si-deck'),od=document.querySelector('.si-deck');
      if(!nd||!od){location.reload();return;}
      var ns=nd.querySelectorAll('.si-slide'),os=od.querySelectorAll('.si-slide');
      if(ns.length!==os.length){location.reload();return;}
      ['si-css','si-user','si-wm','si-data'].forEach(function(id){
        var a=document.getElementById(id),b=doc.getElementById(id);
        if(b&&a)a.textContent=b.textContent;else if(b&&!a)document.head.appendChild(b.cloneNode(true));else if(a&&!b)a.remove();
      });
      for(var i=0;i<os.length;i++){
        var a=os[i],b=ns[i];
        if(a.innerHTML!==b.innerHTML)a.innerHTML=b.innerHTML;
        var nav=(a.className.match(NAV)||[]).join(' ');
        a.className=b.className+(nav?' '+nav:'');
        ['data-title','data-layout','data-topic','data-transition','data-minutes','style'].forEach(function(at){
          var v=b.getAttribute(at);if(v==null)a.removeAttribute(at);else a.setAttribute(at,v);
        });
      }
      if(window.SI&&window.SI.refresh)window.SI.refresh();else location.reload();
    }).catch(function(e){overlay('si play: '+((e&&e.message)||'reload failed'));});
  }
})();</script>
<style>#si-dev-error{position:fixed;left:1rem;right:1rem;bottom:1rem;z-index:9999;max-height:60vh;overflow:auto;white-space:pre-wrap;background:#7f1d1d;color:#fff;padding:1rem 1.2rem;border-radius:.5rem;font:13px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;box-shadow:0 10px 40px rgba(0,0,0,.45)}</style>
` + "\n"
