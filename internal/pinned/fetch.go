package pinned

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/home"
)

// DefaultAPI is GitHub's REST API.
const DefaultAPI = "https://api.github.com"

// Client fetches sources into the cache, checks their pins and asks Trust
// before handing them out.
type Client struct {
	API   string       // GitHub's API (tests: an httptest server)
	HTTP  *http.Client // nil: a client with a 2-minute timeout
	Cache string       // the cache folder (default ~/.si/cache)
	Trust *Trust
	GH    string    // the gh CLI, for private repos ("" = look for it on PATH, "-" = never)
	Token string    // a GitHub token ($GITHUB_TOKEN, $GH_TOKEN): sent to the API only, never printed
	Log   io.Writer // one line per download (nil: quiet)
}

// New is a client with si's defaults: GitHub's API, ~/.si/cache, a token
// from the environment.
func New(t *Trust) *Client {
	tok := os.Getenv("GITHUB_TOKEN")
	if tok == "" {
		tok = os.Getenv("GH_TOKEN")
	}
	return &Client{API: DefaultAPI, Cache: home.Path("cache", ""), Trust: t, Token: tok}
}

// NewTrust is the trust of a command run by a person: ~/.si/trusted, asked
// at the terminal when there is one, yes for the run with --trust.
func NewTrust(yes bool, out io.Writer) *Trust {
	return &Trust{Store: home.Path("trusted", ""), Yes: yes, Ask: Terminal(), In: os.Stdin, Out: out}
}

func (c *Client) logf(format string, a ...any) {
	if c.Log != nil {
		fmt.Fprintf(c.Log, format, a...)
	}
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

func (c *Client) api() string { return strings.TrimRight(firstOf(c.API, DefaultAPI), "/") }

func firstOf(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

// get fetches u; the token goes only to the API's host.
func (c *Client) get(u, accept string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, 0, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	req.Header.Set("User-Agent", "si")
	if c.Token != "" && strings.HasPrefix(u, c.api()+"/") {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	return b, res.StatusCode, err
}

// github asks GitHub's API for p (repos/…, gists/…); a private repo it does
// not show is asked again through gh when gh is there.
func (c *Client) github(p, accept string) ([]byte, error) {
	b, code, err := c.get(c.api()+"/"+p, accept)
	if err == nil && code == 200 {
		return b, nil
	}
	if gh := c.gh(); gh != "" && (err != nil || code == 401 || code == 403 || code == 404) {
		out, gerr := exec.Command(gh, "api", "-H", "Accept: "+accept, p).Output()
		if gerr == nil {
			return out, nil
		}
	}
	if err != nil {
		return nil, err
	}
	msg := strings.TrimSpace(string(b))
	var e struct{ Message string }
	if json.Unmarshal(b, &e) == nil && e.Message != "" {
		msg = e.Message
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return nil, fmt.Errorf("GitHub: %s: %d %s", p, code, strings.TrimPrefix(msg, fmt.Sprint(code)+": "))
}

func (c *Client) gh() string {
	if c.GH == "-" || c.api() != DefaultAPI {
		return ""
	}
	if c.GH != "" {
		return c.GH
	}
	p, _ := exec.LookPath("gh")
	return p
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// file is one fetched file.
type file struct {
	name string
	data []byte
}

// filesSum is the sha256 of a set of files: names and contents.
func filesSum(fs []file) string {
	h := sha256.New()
	for _, f := range fs {
		fmt.Fprintf(h, "%s\x00%s\n", f.name, sum(f.data))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// scriptExt are the files a folder source runs (and files without one).
var scriptExt = map[string]bool{".sh": true, ".bash": true, ".zsh": true, ".py": true, ".rb": true, ".pl": true, ".js": true, ".mjs": true, ".ts": true, ".ps1": true, ".cmd": true, ".bat": true}

func isScript(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return false
	}
	e := strings.ToLower(path.Ext(name))
	return e == "" || scriptExt[e]
}

// download fetches a script source: its files, in name order.
func (c *Client) download(s *Source) ([]file, error) {
	c.logf("fetch: %s\n", s.Key())
	switch s.Kind {
	case "url":
		b, code, err := c.get(s.URL, "")
		if err != nil {
			return nil, err
		}
		if code != 200 {
			return nil, fmt.Errorf("%s: %d", s.URL, code)
		}
		if got := sum(b); got != s.Sum {
			return nil, fmt.Errorf("%s: the pin no longer matches: · sha256 %s… was written, the file is %s…; it does not run", s.URL, short(s.Sum), short(got))
		}
		name := s.Name() + path.Ext(urlPath(s.URL))
		return []file{{name, b}}, nil
	case "gist":
		b, err := c.github("gists/"+s.ID+"/"+s.Rev, "application/vnd.github+json")
		if err != nil {
			return nil, err
		}
		var g struct {
			Files map[string]struct {
				Filename, Content, RawURL string
				Truncated                 bool
			} `json:"files"`
		}
		if err := json.Unmarshal(b, &g); err != nil {
			return nil, fmt.Errorf("gist %s: %v", s.ID, err)
		}
		var names []string
		for n := range g.Files {
			names = append(names, n)
		}
		sort.Strings(names)
		pick := s.File
		if pick == "" {
			if len(names) != 1 {
				return nil, fmt.Errorf("gist:%s@%s has %d files (%s): name one, gist:%s@%s/<file>", s.ID, s.Rev, len(names), strings.Join(names, ", "), s.ID, s.Rev)
			}
			pick = names[0]
		}
		f, ok := g.Files[pick]
		if !ok {
			return nil, fmt.Errorf("gist:%s@%s has no file %s (it has %s)", s.ID, s.Rev, pick, strings.Join(names, ", "))
		}
		data := []byte(f.Content)
		if f.Truncated {
			b, code, err := c.get(f.RawURL, "")
			if err != nil || code != 200 {
				return nil, fmt.Errorf("gist %s: %s: %v %d", s.ID, pick, err, code)
			}
			data = b
		}
		return []file{{pick, data}}, nil
	case "github":
		if !s.Folder() {
			b, err := c.github(contents(s, s.Path), "application/vnd.github.raw")
			if err != nil {
				return nil, err
			}
			return []file{{path.Base(s.Path), b}}, nil
		}
		b, err := c.github(contents(s, strings.TrimSuffix(s.Path, "/")), "application/vnd.github+json")
		if err != nil {
			return nil, err
		}
		var list []struct{ Name, Path, Type string }
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, fmt.Errorf("%s is not a folder", s.Key())
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		var out []file
		for _, e := range list {
			if e.Type != "file" || !isScript(e.Name) {
				continue
			}
			d, err := c.github(contents(s, e.Path), "application/vnd.github.raw")
			if err != nil {
				return nil, err
			}
			out = append(out, file{e.Name, d})
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("%s holds no scripts", s.Key())
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s: not a script source", s.Text)
}

func urlPath(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return p.Path
}

func contents(s *Source, p string) string {
	return "repos/" + s.Owner + "/" + s.Repo + "/contents/" + escapePath(p) + "?ref=" + url.QueryEscape(s.Ref)
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, x := range parts {
		parts[i] = url.PathEscape(x)
	}
	return strings.Join(parts, "/")
}

func keyDir(k string) string { return sum([]byte(k))[:24] }

// Scripts gives the local files of a script source (a gist, a repo's file or
// folder, a pinned URL): from the cache, else downloaded; checked against the
// pin and the trust store, and asked about when new. A folder gives every
// script in it, in name order. Each file may be run.
func (c *Client) Scripts(s *Source) ([]string, error) {
	if s.Kind == "github" && s.Path == "" {
		return nil, fmt.Errorf("%s: name a file or a folder/ of the repo", s.Text)
	}
	dir := filepath.Join(c.Cache, "scripts", keyDir(s.Key()))
	fs, err := readCached(dir)
	if err != nil {
		if fs, err = c.download(s); err != nil {
			return nil, err
		}
		if err := writeCached(dir, fs); err != nil {
			return nil, err
		}
	}
	if s.Kind == "url" && sum(fs[0].data) != s.Sum {
		return nil, fmt.Errorf("%s: the pin no longer matches: the cached file is not sha256 %s…", s.URL, short(s.Sum))
	}
	err = c.Trust.Check(s.Key(), filesSum(fs), func(w io.Writer) {
		fmt.Fprintf(w, "  from %s\n", where(s))
		for _, f := range fs {
			Preview(w, f.name, f.data, 10)
		}
	})
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, f := range fs {
		paths = append(paths, filepath.Join(dir, f.name))
	}
	return paths, nil
}

// where says in words where a source comes from.
func where(s *Source) string {
	switch s.Kind {
	case "gist":
		return fmt.Sprintf("GitHub gist %s at revision %s", s.ID, s.Rev)
	case "github":
		return fmt.Sprintf("GitHub repo %s/%s at %s, %s", s.Owner, s.Repo, s.Ref, firstOf(s.Path, "the whole repo"))
	}
	return s.URL
}

// readCached reads a cache folder written whole by writeCached.
func readCached(dir string) ([]file, error) {
	b, err := os.ReadFile(filepath.Join(dir, ".si-files"))
	if err != nil {
		return nil, err
	}
	var fs []file
	for _, n := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		d, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		fs = append(fs, file{n, d})
	}
	return fs, nil
}

func writeCached(dir string, fs []file) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".tmp-")
	if err != nil {
		return err
	}
	var names []string
	for _, f := range fs {
		if f.name != filepath.Base(f.name) || f.name == "." || f.name == ".." {
			os.RemoveAll(tmp)
			return fmt.Errorf("a file named %q", f.name)
		}
		if err := os.WriteFile(filepath.Join(tmp, f.name), f.data, 0o755); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		names = append(names, f.name)
	}
	os.WriteFile(filepath.Join(tmp, ".si-files"), []byte(strings.Join(names, "\n")+"\n"), 0o644)
	os.RemoveAll(dir)
	return os.Rename(tmp, dir)
}

// Tree gives a repo at a pin, its layout kept (a demo's ../../examples/…
// still resolves): downloaded once into ~/.si/cache/repos/<owner>/<repo>@<ref>
// and asked about once per pin. It returns the local path of s.Path inside
// it (the repo itself when s.Path is ""). preview names, from that path, the
// documents the trust question shows (nil: none).
func (c *Client) Tree(s *Source, preview func(local string) []string) (string, error) {
	if s.Kind != "github" {
		return "", fmt.Errorf("%s: not a repo", s.Text)
	}
	root := filepath.Join(c.Cache, "repos", s.Owner, s.Repo+"@"+safeRef(s.Ref))
	tsum, err := os.ReadFile(root + ".sum")
	if _, serr := os.Stat(root); err != nil || serr != nil {
		c.logf("fetch: %s (the repo, once)\n", s.RepoKey())
		b, err := c.github("repos/"+s.Owner+"/"+s.Repo+"/tarball/"+url.PathEscape(s.Ref), "application/vnd.github+json")
		if err != nil {
			return "", err
		}
		got, err := extract(b, root)
		if err != nil {
			return "", fmt.Errorf("%s: %v", s.RepoKey(), err)
		}
		tsum = []byte(got)
		if err := os.WriteFile(root+".sum", tsum, 0o644); err != nil {
			return "", err
		}
	}
	local := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(s.Path, "/")))
	if _, err := os.Stat(local); err != nil {
		return "", fmt.Errorf("%s: %s is not in the repo at %s", s.Text, firstOf(s.Path, "/"), s.Ref)
	}
	var shown []string
	if preview != nil {
		shown = preview(local)
	}
	err = c.Trust.Check(s.RepoKey(), strings.TrimSpace(string(tsum)), func(w io.Writer) {
		n, size := treeSize(root)
		fmt.Fprintf(w, "  from %s\n  the repo: %d file(s), %d KB; what runs:\n", where(s), n, size/1024)
		for i, p := range shown {
			if i == 3 {
				fmt.Fprintf(w, "  … and %d more\n", len(shown)-3)
				break
			}
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			Preview(w, filepath.ToSlash(rel), b, 12)
		}
	})
	if err != nil {
		return "", err
	}
	return local, nil
}

func safeRef(r string) string { return strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(r) }

func treeSize(root string) (n int, size int64) {
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if fi, err := d.Info(); err == nil {
				n++
				size += fi.Size()
			}
		}
		return nil
	})
	return
}

// extract unpacks a GitHub tarball (one top folder, dropped) into root and
// returns the tree's sha256 (every file's path and contents, in path order).
// Only folders and regular files come out; a path that would leave root is
// an error.
func extract(tgz []byte, root string) (string, error) {
	zr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return "", fmt.Errorf("not a tarball: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(root), ".tmp-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	tr := tar.NewReader(zr)
	var all []file
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok || rel == "" || h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		rel = path.Clean(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
			return "", fmt.Errorf("%s leaves the repo", h.Name)
		}
		dst := filepath.Join(tmp, filepath.FromSlash(rel))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return "", err
			}
			mode := os.FileMode(0o644)
			if h.Mode&0o111 != 0 {
				mode = 0o755
			}
			if err := os.WriteFile(dst, data, mode); err != nil {
				return "", err
			}
			all = append(all, file{rel, data})
		}
	}
	if len(all) == 0 {
		return "", fmt.Errorf("the tarball is empty")
	}
	sort.Slice(all, func(i, j int) bool { return all[i].name < all[j].name })
	os.RemoveAll(root)
	if err := os.Rename(tmp, root); err != nil {
		return "", err
	}
	return filesSum(all), nil
}
