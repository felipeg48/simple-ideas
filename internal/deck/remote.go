package deck

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Code and diffs from git, fetched at build time and cached, so a built page
// never fetches anything:
//
//	<!-- code: https://github.com/acme/app/blob/3f2a9c1/src/App.java#L5-L10 -->
//	<!-- code: path=src/App.java ref=3f2a9c1 tag=retry -->        repo: from front matter
//	<!-- code: repo=../app path=src/App.java ref=HEAD -->         a local clone (git show)
//	<!-- diff: https://github.com/acme/app/commit/3f2a9c1 path=src/App.java -->
//	<!-- diff: https://github.com/acme/app/compare/v1.1...v1.2 path=src/ -->
//	<!-- diff: repo=../app ref=v1.1..v1.2 path=src/App.java -->
//
// GitHub only for now (public and private: GITHUB_TOKEN, GH_TOKEN or a
// logged-in `gh`). A commit ref is cached for good; a branch or tag for an
// hour, and `si check` warns that it moves.

// Offline makes code: and diff: use the cache only (si build/check --offline).
var Offline bool

var (
	commitRE    = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	ghBlobRE    = regexp.MustCompile(`^https://github\.com/([\w.-]+)/([\w.-]+)/blob/([^/]+)/([^#?]+)(?:#L(\d+)(?:-L(\d+))?)?$`)
	ghCommitRE  = regexp.MustCompile(`^https://github\.com/([\w.-]+)/([\w.-]+)/commit/([0-9a-f]{7,40})/?$`)
	ghCompareRE = regexp.MustCompile(`^https://github\.com/([\w.-]+)/([\w.-]+)/compare/([^.]+)\.\.\.([^/?#]+)$`)
	ghRepoRE    = regexp.MustCompile(`^(?:https://github\.com/)?([\w.-]+)/([\w.-]+?)(?:\.git)?/?$`)
)

const fetchLimit = 5 << 20

func fetchCacheDir() string {
	if d := os.Getenv("SI_CACHE_DIR"); d != "" { // tests: a fresh cache, never the user's
		return d
	}
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "si", "fetch")
}

var (
	tokenOnce sync.Once
	token     string
)

// githubToken is GITHUB_TOKEN, GH_TOKEN, or `gh auth token`; "" if none.
func githubToken() string {
	tokenOnce.Do(func() {
		for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
			if v := strings.TrimSpace(os.Getenv(k)); v != "" {
				token = v
				return
			}
		}
		if _, err := exec.LookPath("gh"); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output(); err == nil {
				token = strings.TrimSpace(string(out))
			}
		}
	})
	return token
}

// FetchAsset GETs a versioned asset (a font stylesheet or file) through the
// cache for good; --offline uses the cache only.
func FetchAsset(url string, headers map[string]string) ([]byte, error) {
	return fetchCached("asset:"+url, url, headers, true)
}

// fetchCached GETs url (with headers) through the cache under key. An
// immutable entry never expires; others are refreshed after an hour, and a
// stale copy is used when the network fails.
func fetchCached(key, url string, headers map[string]string, immutable bool) ([]byte, error) {
	sum := sha256.Sum256([]byte(key))
	file := filepath.Join(fetchCacheDir(), fmt.Sprintf("%x", sum[:12]))
	cached, cerr := os.ReadFile(file)
	if cerr == nil {
		if fi, err := os.Stat(file); err == nil && (immutable || Offline || time.Since(fi.ModTime()) < time.Hour) {
			return cached, nil
		}
	}
	if Offline {
		return nil, fmt.Errorf("offline and not in the cache (build once online)")
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "simple-ideas")
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		if cerr == nil {
			return cached, nil // a stale copy beats a failed build
		}
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchLimit+1))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if cerr == nil {
			return cached, nil
		}
		hint := ""
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			hint = " (a private repo needs GITHUB_TOKEN or `gh auth login`; check the owner, repo, ref and path)"
		}
		return nil, fmt.Errorf("%s: HTTP %d%s", url, resp.StatusCode, hint)
	}
	if len(body) > fetchLimit {
		return nil, fmt.Errorf("%s is larger than 5 MB", url)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err == nil {
		os.WriteFile(file, body, 0o644)
	}
	return body, nil
}

func githubFile(owner, repo, ref, path string) ([]byte, error) {
	key := "file:" + owner + "/" + repo + "@" + ref + ":" + path
	if tok := githubToken(); tok != "" {
		return fetchCached(key, fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref),
			map[string]string{"Accept": "application/vnd.github.raw", "Authorization": "Bearer " + tok}, commitRE.MatchString(ref))
	}
	return fetchCached(key, fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path), nil, commitRE.MatchString(ref))
}

// githubDiff fetches the diff of a commit (to == "") or of from...to.
func githubDiff(owner, repo, from, to string) ([]byte, error) {
	spec, page := from, "commit/"+from
	immutable := commitRE.MatchString(from)
	if to != "" {
		spec, page = from+"..."+to, "compare/"+from+"..."+to
		immutable = immutable && commitRE.MatchString(to)
	}
	key := "diff:" + owner + "/" + repo + "@" + spec
	if tok := githubToken(); tok != "" {
		api := "commits/" + from
		if to != "" {
			api = "compare/" + spec
		}
		return fetchCached(key, fmt.Sprintf("https://api.github.com/repos/%s/%s/%s", owner, repo, api),
			map[string]string{"Accept": "application/vnd.github.diff", "Authorization": "Bearer " + tok}, immutable)
	}
	return fetchCached(key, fmt.Sprintf("https://github.com/%s/%s/%s.diff", owner, repo, page), nil, immutable)
}

func gitRun(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

// filterDiff keeps the file sections of a unified diff whose path is path or
// sits under it (path "" keeps everything).
func filterDiff(diff []byte, path string) []byte {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return diff
	}
	var out bytes.Buffer
	keep := false
	for _, l := range strings.SplitAfter(string(diff), "\n") {
		if strings.HasPrefix(l, "diff --git ") {
			f := l
			if i := strings.LastIndex(l, " b/"); i >= 0 {
				f = strings.TrimSpace(l[i+3:])
			}
			keep = f == path || strings.HasPrefix(f, path+"/")
		}
		if keep {
			out.WriteString(l)
		}
	}
	return out.Bytes()
}

// remoteRepo resolves a repo= value (or the repo: front matter): a local
// folder (a clone), else OWNER/REPO or a github.com URL.
func remoteRepo(val, base string) (local, owner, name string, err error) {
	if val == "" {
		return "", "", "", fmt.Errorf("no repo: give a GitHub permalink, repo=OWNER/REPO, repo=../local-clone, or repo: in the front matter")
	}
	p := val
	if !filepath.IsAbs(p) {
		p = filepath.Join(base, val)
	}
	if st, e := os.Stat(p); e == nil && st.IsDir() {
		return p, "", "", nil
	}
	if m := ghRepoRE.FindStringSubmatch(val); m != nil && !strings.HasPrefix(val, ".") {
		return "", m[1], m[2], nil
	}
	return "", "", "", fmt.Errorf("repo=%s is neither a local folder nor OWNER/REPO on GitHub", val)
}

// remoteBlock turns a code: or diff: directive into fence lines. warn is a
// non-fatal note (a moving ref).
func (d *Deck) remoteBlock(kind, val, file string) (lines []string, warn string, err error) {
	val = strings.TrimSpace(val)
	var url string
	if strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "http://") {
		url, val, _ = strings.Cut(val, " ")
	}
	_, opts := splitIncludeValue("x " + val)
	pop := func(k string) string { v := opts[k]; delete(opts, k); return v }
	repoVal, ref, path := pop("repo"), pop("ref"), pop("path")
	if repoVal == "" {
		repoVal = d.Meta.Repo
	}
	base := filepath.Dir(file)
	moving := func(r string) string {
		if !commitRE.MatchString(r) {
			return fmt.Sprintf("%s: ref %q can move (a branch or tag): pin a commit so the shown code cannot drift", kind, r)
		}
		return ""
	}
	switch kind {
	case "code":
		var raw []byte
		src := url
		switch {
		case url != "":
			m := ghBlobRE.FindStringSubmatch(url)
			if m == nil {
				return nil, "", fmt.Errorf("want a GitHub file permalink https://github.com/OWNER/REPO/blob/COMMIT/PATH[#L5-L10]")
			}
			path = m[4]
			if m[5] != "" && opts["lines"] == "" && opts["tag"] == "" {
				opts["lines"] = m[5]
				if m[6] != "" {
					opts["lines"] += "-" + m[6]
				}
			}
			warn = moving(m[3])
			raw, err = githubFile(m[1], m[2], m[3], m[4])
		default:
			if path == "" {
				return nil, "", fmt.Errorf("path= is required (or give a GitHub permalink)")
			}
			local, owner, name, rerr := remoteRepo(repoVal, base)
			if rerr != nil {
				return nil, "", rerr
			}
			if local != "" {
				if ref == "" {
					ref = "HEAD"
				}
				raw, err = gitRun(local, "show", ref+":"+path)
				break
			}
			if ref == "" {
				return nil, "", fmt.Errorf("ref= is required for a GitHub repo (a commit, so the code cannot drift)")
			}
			warn = moving(ref)
			src = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, name, ref, path)
			raw, err = githubFile(owner, name, ref, path)
		}
		if err != nil {
			return nil, warn, err
		}
		lines, err = codeBlock(raw, path, opts, src)
		return lines, warn, err
	case "diff":
		var raw []byte
		src, title := url, path
		switch {
		case url != "":
			if m := ghCommitRE.FindStringSubmatch(url); m != nil {
				raw, err = githubDiff(m[1], m[2], m[3], "")
				if title == "" {
					title = "commit " + m[3][:7]
				}
			} else if m := ghCompareRE.FindStringSubmatch(url); m != nil {
				warn = moving(m[3])
				if warn == "" {
					warn = moving(m[4])
				}
				raw, err = githubDiff(m[1], m[2], m[3], m[4])
				if title == "" {
					title = m[3] + "…" + m[4]
				}
			} else {
				return nil, "", fmt.Errorf("want a GitHub commit or compare URL (…/commit/SHA, …/compare/A...B)")
			}
		default:
			local, owner, name, rerr := remoteRepo(repoVal, base)
			if rerr != nil {
				return nil, "", rerr
			}
			if ref == "" {
				return nil, "", fmt.Errorf("ref= is required: a commit, or A..B")
			}
			from, to, rng := strings.Cut(ref, "..")
			to = strings.TrimPrefix(to, ".")
			if local != "" {
				args := []string{"show", "--format=", from}
				if rng {
					args = []string{"diff", from + ".." + to}
				}
				if path != "" {
					args = append(args, "--", path)
				}
				raw, err = gitRun(local, args...)
			} else {
				warn = moving(from)
				raw, err = githubDiff(owner, name, from, to)
				src = fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, name, from)
				if rng {
					src = fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", owner, name, from, to)
				}
			}
			if title == "" {
				title = ref
			}
		}
		if err != nil {
			return nil, "", err
		}
		raw = filterDiff(raw, path)
		if len(bytes.TrimSpace(raw)) == 0 {
			return nil, "", fmt.Errorf("the diff has no changes under %q", path)
		}
		for k := range opts {
			if k != "title" {
				return nil, "", fmt.Errorf("unknown option %q (path, ref, repo, title)", k)
			}
		}
		if t := opts["title"]; t != "" {
			title = t
		}
		fence := "````"
		info := fmt.Sprintf("%sdiff title=%q", fence, title)
		if src != "" {
			info += fmt.Sprintf(" src=%q", src)
		}
		lines = []string{info}
		lines = append(lines, strings.Split(strings.TrimRight(string(raw), "\n"), "\n")...)
		return append(lines, fence), warn, nil
	}
	return nil, "", fmt.Errorf("unknown kind %s", kind)
}
