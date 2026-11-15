package pinned

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Need is one file a document from a pinned URL lists beside it: its path
// relative to the document's URL and the sha256 it must have.
type Need struct{ Path, Sum string }

var filesRE = regexp.MustCompile(`^:?files:\s*(.*?)\s*$`)

// ParseFiles reads a `files:` value: `path · sha256 HEX`, comma-separated.
// A path is relative, with no . or .. parts.
func ParseFiles(v string) ([]Need, error) {
	var out []Need
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, pin, _ := strings.Cut(part, "·")
		p, pin = strings.TrimSpace(p), strings.TrimSpace(pin)
		k, h, _ := strings.Cut(pin, " ")
		h = strings.ToLower(strings.TrimSpace(h))
		if k != "sha256" || !sumRE.MatchString(h) {
			return nil, fmt.Errorf("files: %q needs its own pin, %s · sha256 <hex> (shasum -a 256 the file)", p, p)
		}
		if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, `\`) || strings.Contains(p, "://") ||
			strings.Contains("/"+p+"/", "/../") || strings.Contains("/"+p+"/", "/./") || strings.Contains(p, "//") {
			return nil, fmt.Errorf("files: %q: a path beside the document, with no . or .. parts", p)
		}
		out = append(out, Need{p, h})
	}
	return out, nil
}

// filesKey finds the `files:` key of a document's front matter (Markdown
// between --- lines), header (AsciiDoc :files: before the first blank line)
// or top-level "files" (a JSON script: a string or a list of entries).
func filesKey(doc []byte) string {
	if t := strings.TrimSpace(strings.TrimPrefix(string(doc), "\uFEFF")); strings.HasPrefix(t, "{") {
		var j struct {
			Files json.RawMessage `json:"files"`
		}
		if json.Unmarshal([]byte(t), &j) != nil || len(j.Files) == 0 {
			return ""
		}
		var one string
		if json.Unmarshal(j.Files, &one) == nil {
			return one
		}
		var list []string
		json.Unmarshal(j.Files, &list)
		return strings.Join(list, ", ")
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimPrefix(string(doc), "\uFEFF"), "\r\n", "\n"), "\n")
	md := len(lines) > 1 && strings.TrimSpace(lines[0]) == "---"
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if md && i > 0 && t == "---" || !md && t == "" && i > 0 {
			break
		}
		if m := filesRE.FindStringSubmatch(t); m != nil && (md || strings.HasPrefix(t, ":")) {
			return strings.Trim(m[1], `"'`)
		}
	}
	return ""
}

// Document gives a document (a web demo script, a deck) from a pinned URL
// or a gist, with the files beside it, and its local path:
//
//   - a pinned URL: the document, and each file its front matter lists in
//     `files: path · sha256 HEX, …`, fetched relative to the URL and checked
//     against its own pin;
//   - gist:ID@REV/file: every file of the gist at that revision (a gist is
//     flat and pinned whole), the named one is the document.
//
// Everything is cached by pin (~/.si/cache/docs) with its folders kept, and
// asked about once, all files together. A github: document is Tree's (the
// repo at its pin).
func (c *Client) Document(s *Source) (string, error) {
	var fs []file
	var main string
	dir := filepath.Join(c.Cache, "docs", keyDir(s.Key()))
	cached, err := readTree(dir)
	fresh := err != nil
	switch s.Kind {
	case "url":
		main = s.Name() + path.Ext(urlPath(s.URL))
		if !fresh {
			fs = cached
			break
		}
		doc, err := c.download(s)
		if err != nil {
			return "", err
		}
		fs = doc
		needs, err := ParseFiles(filesKey(doc[0].data))
		if err != nil {
			return "", fmt.Errorf("%s: %v", s.URL, err)
		}
		base, _ := url.Parse(s.URL)
		for _, n := range needs {
			ref, _ := url.Parse(n.Path)
			u := base.ResolveReference(ref).String()
			c.logf("fetch: %s\n", u)
			b, code, err := c.get(u, "")
			if err != nil {
				return "", err
			}
			if code != 200 {
				return "", fmt.Errorf("%s (files: of %s): %d", u, s.URL, code)
			}
			if got := sum(b); got != n.Sum {
				return "", fmt.Errorf("%s: the pin no longer matches: files: %s · sha256 %s… was written, the file is %s…; nothing runs", u, n.Path, short(n.Sum), short(got))
			}
			fs = append(fs, file{n.Path, b})
		}
	case "gist":
		if s.File == "" {
			return "", fmt.Errorf("%s: name the document, gist:%s@%s/<file>", s.Text, s.ID, s.Rev)
		}
		main = s.File
		if !fresh {
			fs = cached
			break
		}
		if fs, err = c.gist(s); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("%s: not a document source (a pinned URL or a gist)", s.Text)
	}
	if fresh { // fetched now: cache it
		if err := writeTree(dir, fs); err != nil {
			return "", err
		}
	}
	found := false
	for _, f := range fs {
		if f.name == main {
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("%s: no file %s", s.Text, main)
	}
	// the cache is checked again: a pin is what was written, not what is on disk
	if s.Kind == "url" && sum(fs[0].data) != s.Sum {
		return "", fmt.Errorf("%s: the pin no longer matches: the cached file is not sha256 %s…", s.URL, short(s.Sum))
	}
	err = c.Trust.Check(s.Key(), filesSum(fs), func(w io.Writer) {
		fmt.Fprintf(w, "  from %s\n", where(s))
		for i, f := range fs {
			if i == 4 {
				fmt.Fprintf(w, "  … and %d more file(s)\n", len(fs)-4)
				break
			}
			Preview(w, f.name, f.data, 10)
		}
	})
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.FromSlash(main)), nil
}

// readTree reads a cache folder written whole by writeTree, in its order.
func readTree(dir string) ([]file, error) {
	b, err := os.ReadFile(filepath.Join(dir, ".si-files"))
	if err != nil {
		return nil, err
	}
	var fs []file
	for _, n := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		d, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(n)))
		if err != nil {
			return nil, err
		}
		fs = append(fs, file{n, d})
	}
	return fs, nil
}

// writeTree writes fs (slash paths, checked by ParseFiles or flat) into dir
// at once: a temp folder renamed into place.
func writeTree(dir string, fs []file) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".tmp-")
	if err != nil {
		return err
	}
	var names []string
	for _, f := range fs {
		p := filepath.Join(tmp, filepath.FromSlash(f.name))
		if rel, err := filepath.Rel(tmp, p); err != nil || rel == "." || strings.HasPrefix(rel, "..") || strings.HasPrefix(f.name, ".si-files") {
			os.RemoveAll(tmp)
			return fmt.Errorf("a file named %q", f.name)
		}
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, f.data, 0o755); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		names = append(names, f.name)
	}
	os.WriteFile(filepath.Join(tmp, ".si-files"), []byte(strings.Join(names, "\n")+"\n"), 0o644)
	os.RemoveAll(dir)
	return os.Rename(tmp, dir)
}
