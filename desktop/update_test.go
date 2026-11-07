package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v2.1.1", "2.1.0", true}, {"2.1.0", "2.1.0", false}, {"2.10.0", "2.9.9", true}, {"2.1", "2.1.0", false},
		{"2.0.9", "2.1.0", false}, {"v3.0.0-rc1", "2.9.0", true}, {"", "2.0.0", false}, {"2.0.0", "", true}, {"x", "1.0.0", false},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestPickSi(t *testing.T) {
	ver := map[string]string{"/app/si": "2.1.0", "/home/.bin/si": "2.1.1", "/usr/local/bin/si": "1.9.0", "/broken/si": ""}
	v := func(p string) string { return ver[p] }
	if p, version, note := pickSi("/app/si", []string{"/usr/local/bin/si", "/home/.bin/si", "/broken/si"}, v); p != "/home/.bin/si" || version != "2.1.1" || note == "" {
		t.Errorf("a newer installed si wins: %q %q %q", p, version, note)
	}
	if p, _, note := pickSi("/app/si", []string{"/usr/local/bin/si"}, v); p != "/app/si" || note != "" {
		t.Errorf("an older installed si loses: %q %q", p, note)
	}
	if p, _, _ := pickSi("", []string{"/usr/local/bin/si"}, v); p != "/usr/local/bin/si" {
		t.Errorf("no bundled si: use the installed one: %q", p)
	}
	if p, _, _ := pickSi("", nil, v); p != "" {
		t.Errorf("no si at all: %q", p)
	}
}

func TestLatestReleaseIsCachedForADay(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "r.json")
	calls := 0
	fetch := func() (release, error) { calls++; return release{Tag: "v2.2.0", URL: "https://example.org/r"}, nil }
	now := time.Unix(1_800_000_000, 0)
	if r := latestRelease(cache, now, fetch); r.Tag != "v2.2.0" || calls != 1 {
		t.Fatalf("first check: %+v calls=%d", r, calls)
	}
	if r := latestRelease(cache, now.Add(3*time.Hour), fetch); r.Tag != "v2.2.0" || calls != 1 {
		t.Errorf("within a day the cache answers: %+v calls=%d", r, calls)
	}
	latestRelease(cache, now.Add(25*time.Hour), fetch)
	if calls != 2 {
		t.Errorf("after a day GitHub is asked again: calls=%d", calls)
	}
	fail := func() (release, error) { return release{}, errors.New("offline") }
	if r := latestRelease(cache, now.Add(60*time.Hour), fail); r.Tag != "v2.2.0" {
		t.Errorf("offline keeps the last answer: %+v", r)
	}
}
