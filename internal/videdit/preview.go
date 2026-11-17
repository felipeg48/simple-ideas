package videdit

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
)

//go:embed preview.js
var previewJS string

//go:embed preview.css
var previewCSS string

// Preview is the si play page of a video edit: the source playing with
// every callout drawn live over it (the build's own HTML, CSS and runtime),
// blur and the camera in CSS, cuts skipped, speeds and freezes played; a
// strip with a marker per line. liveReload is the page's reload script.
func Preview(path string, cfg *config.Config, liveReload string) []byte {
	e, err := ParseFile(path)
	if err != nil {
		return []byte("<!doctype html><title>si</title><pre>" + html.EscapeString(err.Error()) + "</pre>")
	}
	note := e.Check(FFprobe())
	src, w, h, d := "", 16, 9, 0.0
	if s, err := e.Source(FFmpeg(), FFprobe(), nil); err == nil {
		if in, err := Probe(FFprobe(), s); err == nil {
			src, w, h, d = s, in.W, in.H, in.D
		}
	}
	head, _ := e.head(cfg)
	var b strings.Builder
	title := e.Title
	if title == "" {
		title = e.Path
	}
	esc := html.EscapeString
	fmt.Fprintf(&b, `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>%s%s</style>`, esc(title), head, previewCSS)
	fmt.Fprintf(&b, `<body><header><h1>%s</h1><p><code>si build %s</code> writes <code>%s</code>`, esc(title), esc(rel(e.Dir, e.Path)), esc(e.Out))
	if d > 0 {
		fmt.Fprintf(&b, ` · source %s, %d×%d`, Stamp(d), w, h)
	}
	b.WriteString(`</p></header>`)
	if len(e.Problems) > 0 || note != "" {
		b.WriteString(`<aside class="vp-problems"><ul>`)
		for _, p := range e.Problems {
			kind := "error"
			if p.Warn {
				kind = "warning"
			}
			fmt.Fprintf(&b, `<li class="%s">line %d: %s</li>`, kind, p.Line, esc(p.Msg))
		}
		if note != "" {
			fmt.Fprintf(&b, `<li class="warning">%s</li>`, esc(note))
		}
		b.WriteString(`</ul></aside>`)
	}
	if src == "" {
		b.WriteString(`<p class="vp-none">The source could not be read (ffmpeg and ffprobe play and build video edits).</p></body></html>`)
		return []byte(b.String())
	}
	fmt.Fprintf(&b, `<main><div class="vp-frame" style="aspect-ratio:%d/%d;width:min(100%%,calc(72vh*%d/%d))"><div class="vp-cam"><video src="%s" preload="auto" playsinline></video><div class="ve-stage">`, w, h, w, h, esc(rel(e.Dir, src)))
	type mark struct {
		Line   int     `json:"line"`
		Kind   string  `json:"kind"`
		From   float64 `json:"from"`
		To     float64 `json:"to"`
		Label  string  `json:"label"`
		Range  bool    `json:"range"`
		Dur    float64 `json:"dur,omitempty"`
		Speed  float64 `json:"speed,omitempty"`
		Approx bool    `json:"approx,omitempty"`
		Rect   []float64
		Move   float64 `json:"move,omitempty"`
		Hold   bool    `json:"hold,omitempty"`
		Linear bool    `json:"linear,omitempty"`
	}
	var marks []mark
	var spans []Span
	for _, it := range e.Items {
		if it.NoTime {
			continue
		}
		to := it.Start + it.Dur
		if it.Range() {
			to = it.End
		}
		m := mark{Line: it.Line, Kind: it.Action, From: it.Start, To: to, Label: it.Text, Range: it.Range(), Dur: it.Dur, Speed: it.Speed, Move: it.Move, Hold: it.HasDur, Linear: it.Flags["linear"]}
		switch it.Action {
		case "replace", "refilm", "insert-clip", "insert-image", "card", "say", "music":
			m.Approx = true
		}
		if it.Action == "zoom" || it.Action == "pan" {
			v := view(it.Rect, w, h)
			m.Rect = v[:]
		}
		marks = append(marks, m)
		if drawn(it) || it.Action == "blur" || it.Action == "pixelate" {
			spans = append(spans, Span{it, it.Start, to})
		}
	}
	for _, s := range spans {
		b.WriteString(calloutHTML(s.It, w, h, s.From, s.To-s.From, true))
	}
	b.WriteString(`</div></div></div>`)
	b.WriteString(`<div class="vp-bar"><button type="button" class="vp-play" aria-label="Play">▶</button><span class="vp-time">0:00</span><span class="vp-say" aria-live="polite"></span></div>`)
	b.WriteString(`<div class="vp-strip" role="slider" aria-label="Source time" tabindex="0"><div class="vp-head"></div></div><ol class="vp-lines"></ol>`)
	b.WriteString(`<p class="vp-note">The preview plays the source: cuts are skipped, speeds and freezes played, callouts, blur and the camera drawn live; inserted, replaced and re-filmed clips, cards and sound are markers (∗). The build is exact.</p></main>`)
	data, _ := json.Marshal(map[string]any{"d": d, "w": w, "h": h, "marks": marks})
	fmt.Fprintf(&b, `<script>window.VP=%s</script><script>%s</script><script>%s</script>%s</body></html>`, data, overlayJS, previewJS, liveReload)
	return []byte(b.String())
}
