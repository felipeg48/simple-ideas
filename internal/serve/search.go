package serve

// The play terminal's quick open (⌘P) and search in files (⌘⇧F): a walk of
// one folder on the presenter's machine, loopback-only like the rest of the
// Viewer. Both skip .git, node_modules, vendor, dist, build and dot-folders.

import (
	"bufio"
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	findMaxEntries = 20000 // files and folders a walk looks at
	findMaxResults = 50
	grepMaxHits    = 1000
	grepMaxLine    = 240 // characters of a line shown around its first hit
)

var skipDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true}

// walkFiles calls fn with each file under dir (path relative to dir, with /),
// skipping the usual heavy folders, and dot-folders unless dots; fn returns
// false to stop. It looks at no more than findMaxEntries entries.
func walkFiles(r *http.Request, dir string, dots bool, fn func(rel, abs string, d os.DirEntry) bool) {
	n := 0
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if n++; n > findMaxEntries || r.Context().Err() != nil {
			return filepath.SkipAll
		}
		if p == dir {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if skipDirs[name] || (!dots && strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(p, dir+string(filepath.Separator)))
		if !fn(rel, p, d) {
			return filepath.SkipAll
		}
		return nil
	})
}

// fuzzyScore scores q as a subsequence of path (case-insensitive): 0 when it
// is not one. Runs of letters in a row, letters at the start of a name part
// (after / . _ - or a lower-to-upper step) and hits in the file name score
// more; shorter paths win ties. The file name alone is tried too, so "chk"
// lights checkout.go, not the c of src/.
func fuzzyScore(q, path string) int {
	if q == "" {
		return 1
	}
	base := strings.LastIndexByte(path, '/') + 1
	s := fuzzyRun([]rune(strings.ToLower(q)), path, base)
	if base > 0 {
		s = max(s, fuzzyRun([]rune(strings.ToLower(q)), path[base:], 0))
	}
	if s == 0 {
		return 0
	}
	return s*10 + 1000/(utf8.RuneCountInString(path)+10)
}

// fuzzyRun is the greedy left-to-right match of qr in s; base is where the
// file name starts in s.
func fuzzyRun(qr []rune, s string, base int) int {
	score, qi, run := 0, 0, 0
	prev := rune('/')
	for off, c := range s {
		if qi < len(qr) && unicode.ToLower(c) == qr[qi] {
			n := 1 + run*3
			if strings.ContainsRune("/._- ", prev) || (unicode.IsUpper(c) && unicode.IsLower(prev)) {
				n += 6
			}
			if off >= base {
				n += 2
			}
			score += n
			run++
			qi++
		} else {
			run = 0
		}
		prev = c
	}
	if qi < len(qr) {
		return 0
	}
	return score
}

type findHit struct {
	Path  string `json:"path"`
	Rel   string `json:"rel"`
	score int
}

// find: GET ?dir=&q= -> {dir, files: [{path, rel}], more}: the 50 files under
// dir whose path best matches q as a subsequence.
func (h *termHub) find(w http.ResponseWriter, r *http.Request) {
	dir := h.absPath(r, "dir")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		http.Error(w, "not a folder: "+dir, http.StatusNotFound)
		return
	}
	q := strings.ReplaceAll(r.URL.Query().Get("q"), " ", "")
	var hits []findHit
	walkFiles(r, dir, strings.HasPrefix(q, "."), func(rel, abs string, _ os.DirEntry) bool {
		if s := fuzzyScore(q, rel); s > 0 {
			hits = append(hits, findHit{abs, rel, s})
		}
		return true
	})
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].Rel < hits[j].Rel
	})
	more := len(hits) > findMaxResults
	if more {
		hits = hits[:findMaxResults]
	}
	writeJSON(w, map[string]any{"dir": dir, "files": append([]findHit{}, hits...), "more": more})
}

type grepLine struct {
	Path string   `json:"path"`
	Rel  string   `json:"rel"`
	Line int      `json:"line"`
	Text string   `json:"text"`
	Hits [][2]int `json:"hits"` // UTF-16 offsets into text, for the page
}

// u16 is the length of s in UTF-16 units (a JavaScript string's length).
func u16(s string) int { return len(utf16.Encode([]rune(s))) }

// grep: GET ?dir=&q=&case=1&re=1 -> {dir, lines: [{path, rel, line, text,
// hits}], hits, more}: lines of text files (≤ 1 MB, not binary) under dir
// that contain q; plain text and case-insensitive unless case/re say
// otherwise; at most 1000 hits.
func (h *termHub) grep(w http.ResponseWriter, r *http.Request) {
	dir := h.absPath(r, "dir")
	qv := r.URL.Query()
	q := qv.Get("q")
	if q == "" {
		http.Error(w, "nothing to search for", http.StatusBadRequest)
		return
	}
	pat := q
	if qv.Get("re") != "1" {
		pat = regexp.QuoteMeta(q)
	}
	if qv.Get("case") != "1" {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		http.Error(w, "bad pattern: "+err.Error(), http.StatusBadRequest)
		return
	}
	lines, total, more := []grepLine{}, 0, false
	walkFiles(r, dir, strings.HasPrefix(q, "."), func(rel, abs string, d os.DirEntry) bool {
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > filesMaxBytes {
			return true
		}
		b, err := os.ReadFile(abs)
		if err != nil || bytes.IndexByte(b[:min(len(b), 8192)], 0) >= 0 {
			return true
		}
		sc := bufio.NewScanner(bytes.NewReader(b))
		sc.Buffer(make([]byte, 0, 64<<10), filesMaxBytes+1)
		for n := 1; sc.Scan(); n++ {
			text := strings.ToValidUTF8(sc.Text(), "�")
			locs := re.FindAllStringIndex(text, -1)
			if len(locs) == 0 {
				continue
			}
			// a long line is cut to a window around its first hit
			from := 0
			if len(text) > grepMaxLine {
				from = max(0, locs[0][0]-grepMaxLine/4)
				for from > 0 && !utf8.RuneStart(text[from]) {
					from--
				}
			}
			to := len(text)
			if to-from > grepMaxLine {
				to = from + grepMaxLine
				for to < len(text) && !utf8.RuneStart(text[to]) {
					to++
				}
			}
			gl := grepLine{Path: abs, Rel: rel, Line: n, Text: text[from:to]}
			for _, l := range locs {
				if l[0] < from || l[1] > to || l[0] == l[1] {
					continue
				}
				gl.Hits = append(gl.Hits, [2]int{u16(text[from:l[0]]), u16(text[from:l[1]])})
			}
			if len(gl.Hits) == 0 {
				continue
			}
			lines = append(lines, gl)
			if total += len(locs); total >= grepMaxHits {
				more = true
				return false
			}
		}
		return true
	})
	writeJSON(w, map[string]any{"dir": dir, "lines": lines, "hits": total, "more": more})
}
