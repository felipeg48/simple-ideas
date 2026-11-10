package tools

import (
	"strings"
	"testing"
)

func TestLinuxFamily(t *testing.T) {
	for in, want := range map[string]string{
		"ID=ubuntu\nID_LIKE=debian\n":                  "debian",
		"ID=debian\n":                                  "debian",
		"ID=\"fedora\"\n":                              "fedora",
		"ID=\"rocky\"\nID_LIKE=\"rhel centos fedora\"": "fedora",
		"ID=arch\n":                                    "arch",
		"ID=manjaro\nID_LIKE=arch\n":                   "arch",
		"ID=alpine\n":                                  "other",
		"":                                             "other",
	} {
		if got := linuxFamily(in); got != want {
			t.Errorf("%q: %s, want %s", in, got, want)
		}
	}
}

// Every tool says what it is for and how to install it on every platform, and
// Need puts the feature, the install line and the link together.
func TestEveryToolCanBeInstalled(t *testing.T) {
	for _, tl := range All {
		if tl.For == "" || tl.URL == "" || len(tl.Bins) == 0 {
			t.Errorf("%s: missing For, URL or Bins", tl.Name)
		}
		for _, p := range []string{"mac", "windows", "debian", "fedora", "arch", "other"} {
			if tl.Install[p] == "" {
				t.Errorf("%s: no install advice for %s", tl.Name, p)
			}
		}
		msg := Need(tl.Name, "--feature")
		if !strings.HasPrefix(msg, "--feature needs "+tl.Label) || !strings.Contains(msg, tl.How()) || !strings.Contains(msg, tl.URL) || !strings.Contains(msg, "si doctor") {
			t.Errorf("%s: Need says %q", tl.Name, msg)
		}
	}
	if Find("no-such-tool") != "" {
		t.Error("an unknown tool is never found")
	}
}
