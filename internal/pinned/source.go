// Package pinned brings scripts and documents from elsewhere, safely: a GitHub
// gist, a file, folder or whole tree of a GitHub repo, or a plain URL. Every
// source is pinned (a gist revision, a tag or commit, a sha256), asked about
// once per pin before it runs (remembered in ~/.si/trusted, see Trust) and
// cached by pin (~/.si/cache/scripts, ~/.si/cache/repos). si demo uses it for
// start:/prepare:/setup:/stop:/cleanup: sources and for scripts from repos and
// URLs; si play, build and serve for decks from repos and URLs. Spec:
// docs/superpowers/specs/2026-11-12-demo-lifecycle.md.
package pinned

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Source is one place to fetch from.
type Source struct {
	Kind string // gist, github or url
	Text string // as written

	Owner, Repo, Ref, Path string // github:owner/repo@ref/path (a Path ending in / is a folder; "" the whole repo)
	ID, Rev, File          string // gist:id@rev/file
	URL, Sum               string // a URL and the sha256 it must have (hex)
}

var (
	gistRE = regexp.MustCompile(`^gist:([0-9A-Za-z]+)(?:@([0-9A-Za-z]+))?(?:/([^/\s]+))?$`)
	ghRE   = regexp.MustCompile(`^github:([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)(?:@([^/\s]+))?(/\S*)?$`)
	sumRE  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// IsSource reports whether w (a command's first word) names a remote source:
// gist:…, github:…, https://… or http://….
func IsSource(w string) bool {
	for _, p := range []string{"gist:", "github:", "https://", "http://"} {
		if strings.HasPrefix(w, p) {
			return true
		}
	}
	return false
}

// Parse reads a source; sum is the sha256 a URL must have (its · sha256
// clause). A source without its pin is an error that says how to pin it.
func Parse(w, sum string) (*Source, error) {
	w = strings.TrimSpace(w)
	s := &Source{Text: w}
	switch {
	case strings.HasPrefix(w, "gist:"):
		m := gistRE.FindStringSubmatch(w)
		if m == nil {
			return nil, fmt.Errorf("%s: write gist:ID@REVISION (or gist:ID@REVISION/file.sh)", w)
		}
		if m[2] == "" {
			return nil, fmt.Errorf("%s is not pinned: add the gist's revision, gist:%s@<revision> (the gist's Revisions tab)", w, m[1])
		}
		s.Kind, s.ID, s.Rev, s.File = "gist", m[1], m[2], m[3]
	case strings.HasPrefix(w, "github:"):
		m := ghRE.FindStringSubmatch(w)
		if m == nil {
			return nil, fmt.Errorf("%s: write github:owner/repo@REF/path (a tag or a commit)", w)
		}
		if m[3] == "" {
			return nil, fmt.Errorf("%s is not pinned: add a tag or a commit, github:%s/%s@v1.2%s", w, m[1], m[2], m[4])
		}
		p := strings.TrimPrefix(m[4], "/")
		if strings.Contains("/"+p+"/", "/../") || strings.Contains("/"+p+"/", "/./") {
			return nil, fmt.Errorf("%s: the path may not hold . or .. parts", w)
		}
		s.Kind, s.Owner, s.Repo, s.Ref, s.Path = "github", m[1], m[2], m[3], p
	case strings.HasPrefix(w, "https://") || strings.HasPrefix(w, "http://"):
		u, err := url.Parse(w)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("%s: not a URL", w)
		}
		sum = strings.ToLower(strings.TrimSpace(sum))
		if sum == "" {
			return nil, fmt.Errorf("%s is not pinned: add its fingerprint, %s · sha256 <hex> (shasum -a 256 the file)", w, w)
		}
		if !sumRE.MatchString(sum) {
			return nil, fmt.Errorf("%s: · sha256 takes the 64 hex digits of the file's SHA-256", w)
		}
		s.Kind, s.URL, s.Sum = "url", w, sum
	default:
		return nil, fmt.Errorf("%s: not a source (gist:, github: or a URL)", w)
	}
	return s, nil
}

// Folder reports whether the source is a folder of a repo (every script in
// it, or a tree of documents).
func (s *Source) Folder() bool {
	return s.Kind == "github" && (s.Path == "" || strings.HasSuffix(s.Path, "/"))
}

// Key is the pin: what ~/.si/trusted remembers and the cache is named by.
func (s *Source) Key() string {
	switch s.Kind {
	case "gist":
		k := "gist:" + s.ID + "@" + s.Rev
		if s.File != "" {
			k += "/" + s.File
		}
		return k
	case "github":
		return "github:" + s.Owner + "/" + s.Repo + "@" + s.Ref + "/" + s.Path
	}
	return s.URL + " · sha256 " + s.Sum
}

// RepoKey is the pin of the whole repo (github), else Key.
func (s *Source) RepoKey() string {
	if s.Kind == "github" {
		return "github:" + s.Owner + "/" + s.Repo + "@" + s.Ref
	}
	return s.Key()
}

func (s *Source) String() string { return s.Key() }

// Name is a short name for output folders: the repo, the gist, or the URL's
// file without its extension.
func (s *Source) Name() string {
	switch s.Kind {
	case "github":
		return s.Repo
	case "gist":
		return "gist-" + s.ID
	}
	u, _ := url.Parse(s.URL)
	b := path.Base(u.Path)
	if b == "." || b == "/" || b == "" {
		return u.Host
	}
	return strings.TrimSuffix(b, path.Ext(b))
}

// ParseTarget reads what si demo test, si play, si build and si serve take
// besides a local path: github:owner/repo@REF/path, or a URL followed by
// "· sha256 HEX". ok is false when t is a local path.
func ParseTarget(t string) (*Source, bool, error) {
	t = strings.TrimSpace(t)
	head, rest, _ := strings.Cut(t, "·")
	head = strings.TrimSpace(head)
	if !strings.HasPrefix(head, "github:") && !strings.HasPrefix(head, "https://") && !strings.HasPrefix(head, "http://") {
		if strings.HasPrefix(head, "gist:") {
			return nil, true, fmt.Errorf("%s: a gist holds scripts for start:, prepare:, setup:, stop: and cleanup:; a demo or a deck comes from github: or a URL", head)
		}
		return nil, false, nil
	}
	sum := ""
	if rest = strings.TrimSpace(rest); rest != "" {
		k, v, _ := strings.Cut(rest, " ")
		if k != "sha256" {
			return nil, true, fmt.Errorf("%s: after · only sha256 <hex>", t)
		}
		sum = strings.TrimSpace(v)
	}
	s, err := Parse(head, sum)
	return s, true, err
}

// JoinTarget puts a URL written as separate arguments back together:
// `https://…/x.md · sha256 HEX` given unquoted is four arguments.
func JoinTarget(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if (strings.HasPrefix(a, "https://") || strings.HasPrefix(a, "http://")) && i+3 < len(args) && args[i+1] == "·" && args[i+2] == "sha256" {
			out = append(out, a+" · sha256 "+args[i+3])
			i += 3
			continue
		}
		out = append(out, a)
	}
	return out
}
