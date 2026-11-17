package main

import (
	"os"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// writePlayer is si --export player: the video player of si pages as two
// files any page can load (the same CSS and JS si inlines), a README and a
// small demo page whose cards (data-video=) become players.
func writePlayer(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	files := []struct{ name, text string }{
		{"si-player.css", "/* si-player.css: the si video player (si --export player). */\n" + assets.VideoplayerCSS},
		{"si-player.js", assets.VideoplayerJS},
		{"README.md", playerReadme},
		{"demo.html", playerDemo},
		{"demo.chapters.txt", "0:00 Getting started\n0:05 Scaling out\n0:10 Watching it run\n"},
	}
	var wrote []string
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, []byte(f.text), 0o644); err != nil {
			return wrote, err
		}
		wrote = append(wrote, p)
	}
	return wrote, nil
}

const playerReadme = "# si player\n\n" +
	"The video player of si pages, for any page: a poster with a play button, its own\n" +
	"controls (scrub bar with chapters, time, volume, speed, captions, picture in\n" +
	"picture, fullscreen), keyboard and touch, and a dialog over the dimmed page that\n" +
	"steps through every video on it.\n\n" +
	"```html\n" +
	"<link rel=\"stylesheet\" href=\"si-player.css\">\n" +
	"<script src=\"si-player.js\" defer></script>\n\n" +
	"<figure class=\"video-card\" data-video=\"clips/01-intro.mp4\"\n" +
	"        data-captions=\"clips/01-intro.vtt\" data-chapters=\"clips/01-intro.chapters.txt\">\n" +
	"  <img src=\"shots/01-intro.png\" alt=\"The complete film\">\n" +
	"  <figcaption><h3>The complete film</h3><p>The whole story, end to end.</p></figcaption>\n" +
	"</figure>\n" +
	"```\n\n" +
	"Any element with `data-video` becomes a player. Its `<img>` is the poster, its\n" +
	"first heading the title and its first paragraph the caption under the video.\n\n" +
	"| Attribute | What |\n|---|---|\n" +
	"| `data-video` | the video file |\n" +
	"| `data-poster` | the still before it plays (default: the `<img>` inside) |\n" +
	"| `data-title`, `data-caption` | the title and the text under the video (default: the heading and paragraph inside) |\n" +
	"| `data-captions` | a WebVTT file; `C` or the CC button shows it |\n" +
	"| `data-chapters` | `0:00 Title` lines or a WebVTT chapters file: marks on the bar and a list |\n" +
	"| `data-mode` | `modal` (default: the dialog over the dimmed page) or `inline` (plays in place of the picture) |\n" +
	"| `data-group` | players in the same group share the dialog's ‹ › (default: the whole page) |\n" +
	"| `data-start` | start at this time (`1:05`) |\n\n" +
	"Keys while a player has focus: Space or K play and pause, ← → 5 s, J L 10 s,\n" +
	"M mute, F fullscreen, C captions, Home End, Esc. In the dialog ← → move to the\n" +
	"previous or next video while paused (Shift+← → while playing). On a phone, a tap\n" +
	"shows or hides the controls and a double tap on a side seeks 10 s.\n\n" +
	"Captions and chapters are fetched, so serve the page over http (any static\n" +
	"server); from file:// the video plays without them. Colours follow `--si-accent`\n" +
	"when the page sets it; every `--si-vp-*` token in si-player.css can be set too.\n" +
	"`window.siPlayer.init(element)` wires players added after the page loaded.\n\n" +
	"demo.html plays `demo.mp4` (make one with\n" +
	"`ffmpeg -f lavfi -i testsrc2=duration=20:size=1280x720 -pix_fmt yuv420p demo.mp4`).\n"

const playerDemo = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>si player</title>
<link rel="stylesheet" href="si-player.css">
<script src="si-player.js" defer></script>
<style>
:root{--si-accent:#e8590c;color-scheme:light dark;font-family:system-ui,sans-serif}
body{margin:0;padding:2rem clamp(1rem,4vw,3rem);background:Canvas;color:CanvasText}
h1{font-size:clamp(1.6rem,4vw,2.4rem);margin:0 0 .3em}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(min(100%,18rem),1fr));gap:1.4rem;margin-block:1.6rem}
.card{margin:0;border-radius:.8rem;overflow:hidden;background:color-mix(in srgb,CanvasText 6%,Canvas)}
.card img{display:block;width:100%;aspect-ratio:16/9;object-fit:cover}
.card figcaption{padding:.8rem 1rem 1rem}
.card h3{margin:0 0 .25em;font-size:1.05rem}
.card p{margin:0;opacity:.75}
.inline{max-width:48rem}
</style>
</head>
<body>
<h1>Three short films</h1>
<p>Each card opens the player over the page; ‹ › step through them.</p>
<div class="grid">
<figure class="card" data-video="demo.mp4" data-chapters="demo.chapters.txt">
  <img alt="Getting started" src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 9'%3E%3Crect width='16' height='9' fill='%23e8590c'/%3E%3C/svg%3E">
  <figcaption><h3>Getting started</h3><p>From nothing to a running service.</p></figcaption>
</figure>
<figure class="card" data-video="demo.mp4" data-start="0:05">
  <img alt="Scaling out" src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 9'%3E%3Crect width='16' height='9' fill='%232b8a3e'/%3E%3C/svg%3E">
  <figcaption><h3>Scaling out</h3><p>Three nodes, and one of them goes away.</p></figcaption>
</figure>
<figure class="card" data-video="demo.mp4" data-start="0:10">
  <img alt="Watching it run" src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 9'%3E%3Crect width='16' height='9' fill='%231c7ed6'/%3E%3C/svg%3E">
  <figcaption><h3>Watching it run</h3><p>The console, live.</p></figcaption>
</figure>
</div>
<h2>In place</h2>
<figure class="card inline" data-video="demo.mp4" data-mode="inline">
  <img alt="The whole tour" src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 9'%3E%3Crect width='16' height='9' fill='%235f3dc4'/%3E%3C/svg%3E">
  <figcaption><h3>The whole tour</h3><p>Plays right here, in the page.</p></figcaption>
</figure>
</body>
</html>
`
