package main

// Keeping the app current: the si it runs (the bundled one, unless a newer
// one is installed) and a daily check for a newer release on GitHub.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const releasesAPI = "https://api.github.com/repos/felipeg48/simple-ideas/releases/latest"

// versionOf runs `bin --version` ("si 2.1.0") and returns "2.1.0", or "".
func versionOf(bin string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return ""
	}
	f := strings.Fields(string(out))
	if len(f) < 2 || f[0] != "si" {
		return ""
	}
	return f[1]
}

// newer reports whether version a is newer than b: dotted numbers, a leading v
// ignored, anything after a "-" (a pre-release) left out; "" is never newer.
func newer(a, b string) bool {
	pa, pb := parts(a), parts(b)
	if pa == nil {
		return false
	}
	if pb == nil {
		return true
	}
	for i := 0; i < max(len(pa), len(pb)); i++ {
		x, y := at(pa, i), at(pb, i)
		if x != y {
			return x > y
		}
	}
	return false
}

func parts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-")
	if v == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(v, ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func at(p []int, i int) int {
	if i < len(p) {
		return p[i]
	}
	return 0
}

// pickSi chooses the si to run: the bundled one, unless an installed one is
// newer. note says why, for the About panel ("" when the bundled one runs).
func pickSi(bundled string, installed []string, ver func(string) string) (path, version, note string) {
	path, version = bundled, ""
	if bundled != "" {
		version = ver(bundled)
	}
	for _, p := range installed {
		if v := ver(p); v != "" && (path == "" || newer(v, version)) {
			path, version = p, v
		}
	}
	if path != bundled && bundled != "" {
		note = "using si " + version + " from " + path + ", newer than the one in the app"
	}
	return path, version, note
}

// installedSi lists the si binaries a user may have installed, in the order
// they are preferred when versions tie (a variable, so tests can stub it).
var installedSi = func() []string {
	var out []string
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".bin", "si"), filepath.Join(home, "bin", "si"), filepath.Join(home, "go", "bin", "si"))
	}
	out = append(out, "/opt/homebrew/bin/si", "/usr/local/bin/si")
	if p, err := exec.LookPath("si"); err == nil {
		out = append(out, p)
	}
	var seen = map[string]bool{}
	var real []string
	for _, p := range out {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && !seen[p] {
			seen[p] = true
			real = append(real, p)
		}
	}
	return real
}

type release struct {
	Checked int64  `json:"checked"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

// latestRelease returns the newest release, asking GitHub at most once a day
// (the answer is cached in cache); any failure is silent and gives "".
func latestRelease(cache string, now time.Time, fetch func() (release, error)) release {
	var r release
	if b, err := os.ReadFile(cache); err == nil && json.Unmarshal(b, &r) == nil && now.Sub(time.Unix(r.Checked, 0)) < 24*time.Hour {
		return r
	}
	got, err := fetch()
	if err != nil {
		return r // the stale answer, if any
	}
	got.Checked = now.Unix()
	if b, err := json.Marshal(got); err == nil {
		os.MkdirAll(filepath.Dir(cache), 0o755)
		os.WriteFile(cache, b, 0o644)
	}
	return got
}

func fetchRelease() (release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, errors.New(resp.Status)
	}
	var v struct {
		Tag string `json:"tag_name"`
		URL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return release{}, err
	}
	return release{Tag: v.Tag, URL: v.URL}, nil
}

func releaseCache() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "si-studio", "release.json")
}
