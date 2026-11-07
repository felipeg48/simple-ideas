package serve

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
)

// POST /_si/callout: a callout dragged on a page open in si play keeps its
// place. The page sends the callout's line and text and the new at=; the
// server finds that callout in the page's source (or, for an included file,
// in a Markdown file under the served folder) and rewrites its at= with the
// same writer the Studio uses. Loopback only (Handler, never RemoteHandler),
// same-origin only; a built page never sends it.

type calloutMove struct {
	Page string `json:"page"`
	Line int    `json:"line"`
	Text string `json:"text"`
	At   []int  `json:"at"`
}

func (s *Server) callout(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var m calloutMove
	b, _ := io.ReadAll(io.LimitReader(r.Body, 8192))
	if json.Unmarshal(b, &m) != nil || m.Line < 1 || strings.TrimSpace(m.Text) == "" || len(m.At) != 2 {
		http.Error(w, "bad callout", http.StatusBadRequest)
		return
	}
	md := s.source(strings.TrimSuffix(path.Clean("/"+m.Page), ".html"))
	files := []string{}
	if md != "" {
		files = append(files, md)
	}
	filepath.WalkDir(s.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && p != s.dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
			return filepath.SkipDir
		}
		if !d.IsDir() && p != md && (strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".adoc")) {
			files = append(files, p)
		}
		return nil
	})
	type hit struct {
		file              string
		line, step, index int
	}
	var hits []hit
	for i, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, sh := range codeshow.Shows(string(raw)) {
			for si, st := range sh.Steps {
				for ci, c := range st.Callouts {
					if c.Line == m.Line && strings.TrimSpace(c.Text) == strings.TrimSpace(m.Text) {
						hits = append(hits, hit{f, sh.Steps[0].Start + 1, si, ci})
					}
				}
			}
		}
		if i == 0 && len(hits) > 0 {
			break // the page's own file wins
		}
	}
	switch {
	case len(hits) == 0:
		http.Error(w, "no such callout", http.StatusNotFound)
		return
	case len(hits) > 1:
		http.Error(w, "more than one callout reads like that: place it in the Studio", http.StatusConflict)
		return
	}
	h := hits[0]
	raw, err := os.ReadFile(h.file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, _, err := codeshow.Apply(string(raw), h.line, codeshow.Op{Kind: "move-callout", Step: h.step, Index: h.index, At: m.At})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(h.file); err == nil {
		mode = st.Mode().Perm()
	}
	if err := os.WriteFile(h.file, []byte(out), mode); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
