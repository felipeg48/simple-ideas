package serve

import (
	"encoding/json"
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/home"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Session is one running si play / si serve, as a small file the Mac app's
// menu-bar icon reads (<config dir>/si/sessions/<pid>.json): what it serves,
// where, and the loopback page with the phone codes when --remote is on.
type Session struct {
	PID     int       `json:"pid"`
	Kind    string    `json:"kind"` // play or serve
	Src     string    `json:"src"`  // the folder or deck, absolute
	URL     string    `json:"url"`  // the index (loopback)
	Deck    string    `json:"deck,omitempty"`
	QR      string    `json:"qr,omitempty"` // /_si/qr: the phone links as codes
	Started time.Time `json:"started"`
}

// SessionsDir is where sessions are listed: $SI_SESSIONS_DIR, else
// ~/.si/sessions.
func SessionsDir() (string, error) {
	if d := os.Getenv("SI_SESSIONS_DIR"); d != "" {
		return d, nil
	}
	return home.Path("sessions", home.Config), nil
}

// WriteSession lists this process as a session; the returned func removes
// it (a session killed without it is skipped by readers: its pid is gone).
func WriteSession(s Session) (func(), error) {
	dir, err := SessionsDir()
	if err != nil {
		return func() {}, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return func() {}, err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	p := filepath.Join(dir, strconv.Itoa(s.PID)+".json")
	if err := os.WriteFile(p, b, 0o600); err != nil { // carries the QR page's address only, but keep it to the user
		return func() {}, err
	}
	return func() { os.Remove(p) }, nil
}

// qrPage is GET /_si/qr on the loopback listener: the --remote links as QR
// codes, for the presenter's own screen (the Mac app's menu opens it).
func (s *Server) qrPage(w http.ResponseWriter, r *http.Request) {
	rq := s.RemoteQR
	if rq == nil || len(rq.Links) == 0 {
		http.Error(w, "no --remote links in this si play", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>si — phone links</title>
<style>body{margin:0;min-height:100vh;display:flex;flex-wrap:wrap;align-content:center;justify-content:center;gap:clamp(1rem,4vmin,3rem);padding:4vmin;box-sizing:border-box;font:16px/1.4 system-ui,sans-serif;background:#f6f5f2;color:#1a1a1a}
figure{margin:0;display:flex;flex-direction:column;align-items:center;gap:.6em;width:min(30vmin,80vw);min-width:12rem}figure div{width:100%;background:#fff;border-radius:.8em;padding:.4em;box-sizing:border-box}
figure svg{display:block;width:100%;height:auto}figcaption{text-align:center;font-weight:600}small{display:block;font:400 .75em/1.3 ui-monospace,monospace;color:#666;word-break:break-all}
p{flex-basis:100%;text-align:center;color:#666;margin:0}@media (prefers-color-scheme:dark){body{background:#16171b;color:#eee}small,p{color:#aaa}}</style></head><body>`)
	for _, l := range rq.Links {
		fmt.Fprintf(w, `<figure><div>%s</div><figcaption>%s<small>%s</small></figcaption></figure>`, l.SVG, html.EscapeString(l.Name), html.EscapeString(l.URL))
	}
	if len(rq.Also) > 0 {
		fmt.Fprintf(w, `<p>Not on this network? The same links work with %s.</p>`, html.EscapeString(fmt.Sprint(rq.Also)))
	}
	fmt.Fprint(w, `</body></html>`)
}
