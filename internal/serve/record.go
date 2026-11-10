package serve

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Record your talk (si play, loopback only): the teleprompter records the
// microphone while you present and sends the audio in chunks (a crash loses
// at most one) and a mark for every slide and step change. On stop the
// recording is .si/recordings/<talk>-<date-time>/ beside the deck:
// audio.webm (or .m4a), timeline.json and meta.json. One recording at a time.

const (
	maxChunk = 4 << 20   // bytes per chunk (two seconds of Opus is far less)
	maxAudio = 600 << 20 // bytes per recording
)

// Mark is one change of view: t seconds of audio from the start, slide and
// step 0-based as the runtime counts them.
type Mark struct {
	T     float64 `json:"t"`
	Slide int     `json:"slide"`
	Step  int     `json:"step"`
}

type recorder struct {
	mu    sync.Mutex
	dir   string // the recording in progress ("" when none)
	audio string
	size  int64
	marks []Mark
	meta  map[string]any
}

var recExtRE = regexp.MustCompile(`^(webm|m4a|ogg)$`)

// recordStatus is GET /_si/record: whether this server records (the button
// shows only where it answers) and whether a recording runs.
func (s *Server) recordStatus(w http.ResponseWriter, r *http.Request) {
	s.rec.mu.Lock()
	on := s.rec.dir != ""
	s.rec.mu.Unlock()
	writeJSONTo(w, map[string]any{"ok": true, "recording": on})
}

// recordStart is POST /_si/record/start {path, ext}: path is the deck page
// (/talk.html), ext the audio container the browser records.
func (s *Server) recordStart(w http.ResponseWriter, r *http.Request) {
	var v struct{ Path, Ext string }
	if json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&v) != nil || !recExtRE.MatchString(v.Ext) {
		http.Error(w, "want {path, ext}", http.StatusBadRequest)
		return
	}
	src := s.source(strings.TrimSuffix(path.Clean("/"+v.Path), ".html"))
	if src == "" {
		http.Error(w, "no deck at "+v.Path, http.StatusNotFound)
		return
	}
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	if s.rec.dir != "" {
		http.Error(w, "already recording", http.StatusConflict)
		return
	}
	talk := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	now := time.Now()
	dir := filepath.Join(s.dir, ".si", "recordings", talk+"-"+now.Format("2006-01-02-150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, _ := os.ReadFile(src)
	sum := sha256.Sum256(raw)
	rel, _ := filepath.Rel(s.dir, src)
	s.rec.dir, s.rec.audio, s.rec.size = dir, filepath.Join(dir, "audio."+v.Ext), 0
	s.rec.marks = nil
	s.rec.meta = map[string]any{"deck": filepath.ToSlash(rel), "sha256": hex.EncodeToString(sum[:]), "started": now.Format(time.RFC3339), "audio": "audio." + v.Ext}
	writeJSONTo(w, map[string]any{"dir": filepath.ToSlash(strings.TrimPrefix(dir, s.dir+string(filepath.Separator)))})
}

// recordChunk is POST /_si/record/chunk: the next piece of audio, appended.
func (s *Server) recordChunk(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(io.LimitReader(r.Body, maxChunk+1))
	if err != nil || len(b) > maxChunk {
		http.Error(w, "chunk too large", http.StatusRequestEntityTooLarge)
		return
	}
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	if s.rec.dir == "" {
		http.Error(w, "not recording", http.StatusConflict)
		return
	}
	if s.rec.size+int64(len(b)) > maxAudio {
		http.Error(w, "the recording is too long", http.StatusRequestEntityTooLarge)
		return
	}
	f, err := os.OpenFile(s.rec.audio, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		_, err = f.Write(b)
		f.Close()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.rec.size += int64(len(b))
	w.WriteHeader(http.StatusNoContent)
}

// recordMark is POST /_si/record/mark {t, slide, step}.
func (s *Server) recordMark(w http.ResponseWriter, r *http.Request) {
	var m Mark
	if json.NewDecoder(io.LimitReader(r.Body, 512)).Decode(&m) != nil || m.T < 0 || m.Slide < 0 || m.Step < 0 {
		http.Error(w, "want {t, slide, step}", http.StatusBadRequest)
		return
	}
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	if s.rec.dir == "" {
		http.Error(w, "not recording", http.StatusConflict)
		return
	}
	if n := len(s.rec.marks); n > 0 && s.rec.marks[n-1].Slide == m.Slide && s.rec.marks[n-1].Step == m.Step {
		w.WriteHeader(http.StatusNoContent) // the same view again (a reload, a resend)
		return
	}
	if len(s.rec.marks) < 100000 {
		s.rec.marks = append(s.rec.marks, m)
	}
	s.rec.writeTimeline()
	w.WriteHeader(http.StatusNoContent)
}

// recordStop is POST /_si/record/stop {t}: the length; writes the timeline
// and meta and ends the recording.
func (s *Server) recordStop(w http.ResponseWriter, r *http.Request) {
	var v struct {
		T float64 `json:"t"`
	}
	json.NewDecoder(io.LimitReader(r.Body, 512)).Decode(&v)
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	if s.rec.dir == "" {
		http.Error(w, "not recording", http.StatusConflict)
		return
	}
	s.rec.meta["seconds"] = v.T
	s.rec.meta["ended"] = time.Now().Format(time.RFC3339)
	s.rec.writeTimeline()
	b, _ := json.MarshalIndent(s.rec.meta, "", "  ")
	os.WriteFile(filepath.Join(s.rec.dir, "meta.json"), b, 0o644)
	rel, _ := filepath.Rel(s.dir, s.rec.dir)
	s.rec.dir = ""
	writeJSONTo(w, map[string]any{"dir": filepath.ToSlash(rel), "marks": len(s.rec.marks)})
}

// writeTimeline keeps timeline.json current, so a crash keeps the marks too
// (callers hold mu).
func (rc *recorder) writeTimeline() {
	marks := rc.marks
	if marks == nil {
		marks = []Mark{}
	}
	b, _ := json.MarshalIndent(marks, "", " ")
	os.WriteFile(filepath.Join(rc.dir, "timeline.json"), b, 0o644)
}

// recordRoutes adds the recorder to the loopback server.
func (s *Server) recordRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /_si/record", localOnly(s.recordStatus))
	mux.HandleFunc("POST /_si/record/start", localOnly(s.recordStart))
	mux.HandleFunc("POST /_si/record/chunk", localOnly(s.recordChunk))
	mux.HandleFunc("POST /_si/record/mark", localOnly(s.recordMark))
	mux.HandleFunc("POST /_si/record/stop", localOnly(s.recordStop))
}
