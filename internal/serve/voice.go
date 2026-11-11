package serve

import (
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// Auto-play in si play: GET /_si/voice/<slide>/<part>.wav?page=/talk.html
// voices one part of a slide's script (slide from 1, part from 0, as split
// by deck.Parts) with the deck's voice, through the voice cache of si voice
// and si demo run: the first play voices it, every later one is instant, and
// an edited part is voiced again. Loopback only (it runs the voice
// programs); the page falls back to the browser's voice on any error.
func (s *Server) voice(w http.ResponseWriter, r *http.Request) {
	slide, err1 := strconv.Atoi(r.PathValue("slide"))
	part, err2 := strconv.Atoi(strings.TrimSuffix(r.PathValue("part"), ".wav"))
	md := s.file
	if page := r.URL.Query().Get("page"); page != "" {
		md = s.source(strings.TrimSuffix(path.Clean("/"+page), ".html"))
	}
	if err1 != nil || err2 != nil || md == "" {
		http.NotFound(w, r)
		return
	}
	d, err := deck.Parse(md)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	parts := d.Parts()
	if slide < 1 || slide > len(parts) || part < 0 || part >= len(parts[slide-1]) || parts[slide-1][part].Text == "" {
		http.NotFound(w, r)
		return
	}
	p := parts[slide-1][part]
	dv, warns, err := webdemo.ForDeck(d, config.Load(s.Conf, filepath.Join(s.dir, "si.conf")), "")
	for _, m := range warns {
		s.warnOnce(m)
	}
	if err != nil {
		s.warnOnce(err.Error())
		http.NotFound(w, r)
		return
	}
	text := webdemo.Spoken(p.Text, dv.Pronounce)
	s.voiceMu.Lock() // ponytail: one voicing at a time; the page asks for one part ahead
	lines, _, err := s.voicer().Voice(dv.For(p.Speaker), []string{text})
	s.voiceMu.Unlock()
	if err != nil {
		s.warnOnce("auto-play: " + err.Error() + " (the page speaks with the browser's voice)")
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, lines[text].WAV)
}

func (s *Server) voicer() webdemo.Voicer {
	if s.Voicer != nil {
		return s.Voicer
	}
	ff := tools.Find("ffmpeg")
	if ff == "" {
		ff = "ffmpeg" // the error names it; tools.Need says how to install it
		s.warnOnce(tools.Need("ffmpeg", "auto-play voices"))
	}
	return &webdemo.Synth{FFmpeg: ff, Cache: webdemo.VoiceCache()}
}

func (s *Server) warnOnce(m string) {
	if _, seen := s.warned.LoadOrStore(m, true); !seen {
		log.Printf("si play: %s", m)
	}
}
