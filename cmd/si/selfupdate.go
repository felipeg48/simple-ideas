package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/felipeg48/simple-ideas/releases/latest"

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	Tag    string    `json:"tag_name"`
	Assets []ghAsset `json:"assets"`
}

// selfUpdate downloads the latest release binary for this OS/arch, verifies it
// against the release SHA256SUMS.txt, and replaces the running executable.
// Stdlib only: no auto-update framework, just the GitHub releases API.
func selfUpdate(stdout, stderr io.Writer) int {
	client := &http.Client{Timeout: 90 * time.Second}

	rel, err := latestRelease(client)
	if err != nil {
		fmt.Fprintf(stderr, "si: selfupdate: %v\n", err)
		return exitProblems
	}
	if !semverLess(version, strings.TrimPrefix(rel.Tag, "v")) {
		fmt.Fprintf(stdout, "si %s is already up to date (latest release %s)\n", version, rel.Tag)
		return exitOK
	}

	name := fmt.Sprintf("si_%s_%s_%s", rel.Tag, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var binURL, sumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case name:
			binURL = a.URL
		case "SHA256SUMS.txt":
			sumsURL = a.URL
		}
	}
	if binURL == "" {
		fmt.Fprintf(stderr, "si: selfupdate: %s has no binary for %s/%s\n", rel.Tag, runtime.GOOS, runtime.GOARCH)
		return exitProblems
	}

	data, err := download(client, binURL)
	if err != nil {
		fmt.Fprintf(stderr, "si: selfupdate: download: %v\n", err)
		return exitProblems
	}
	// Verify the checksum: a tampered or truncated binary must never be installed.
	if sumsURL != "" {
		sums, err := download(client, sumsURL)
		if err != nil {
			fmt.Fprintf(stderr, "si: selfupdate: checksums: %v\n", err)
			return exitProblems
		}
		want := sha256From(string(sums), name)
		if want == "" {
			fmt.Fprintf(stderr, "si: selfupdate: %s missing from SHA256SUMS.txt\n", name)
			return exitProblems
		}
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != want {
			fmt.Fprintf(stderr, "si: selfupdate: checksum mismatch for %s (refusing)\n", name)
			return exitProblems
		}
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "si: selfupdate: %v\n", err)
		return exitProblems
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if err := replaceExecutable(exe, data); err != nil {
		fmt.Fprintf(stderr, "si: selfupdate: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si updated %s -> %s (%s)\n", version, rel.Tag, exe)
	return exitOK
}

func latestRelease(client *http.Client) (*ghRelease, error) {
	req, _ := http.NewRequest(http.MethodGet, releaseAPI, nil)
	req.Header.Set("User-Agent", "si-selfupdate")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.Tag == "" {
		return nil, fmt.Errorf("no release found")
	}
	return &rel, nil
}

func download(client *http.Client, url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "si-selfupdate")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// semverLess reports whether version a is older than b, comparing dotted numeric
// components (v-prefix and any -prerelease suffix ignored). Guards selfupdate
// against downgrading a dev build that is ahead of the latest release.
func semverLess(a, b string) bool {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	num := func(s []string, i int) int {
		if i >= len(s) {
			return 0
		}
		v, _ := strconv.Atoi(strings.SplitN(s[i], "-", 2)[0])
		return v
	}
	for i := 0; i < n; i++ {
		if x, y := num(as, i), num(bs, i); x != y {
			return x < y
		}
	}
	return false
}

// sha256From finds the hex digest for file `name` in SHA256SUMS.txt content
// (lines of "<hex>  <name>"). Returns "" when absent.
func sha256From(sums, name string) string {
	for _, line := range strings.Split(sums, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[len(f)-1] == name {
			return f[0]
		}
	}
	return ""
}

// replaceExecutable atomically swaps the file at exe with data. It writes a temp
// file in the same directory (so os.Rename stays on one filesystem). On Windows a
// running .exe can't be renamed over, so the old one is moved aside first.
func replaceExecutable(exe string, data []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, "si-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try reinstalling, or sudo): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(exe + ".old")
		if err := os.Rename(exe, exe+".old"); err != nil {
			return err
		}
	}
	return os.Rename(tmpName, exe)
}
