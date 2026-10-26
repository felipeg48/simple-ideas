package main

import "testing"

func TestSHA256From(t *testing.T) {
	sums := "abc123  si_v0.2.0_linux_amd64\n" +
		"def456  si_v0.2.0_darwin_arm64\n" +
		"999aaa  si_v0.2.0_windows_amd64.exe\n"
	cases := map[string]string{
		"si_v0.2.0_linux_amd64":       "abc123",
		"si_v0.2.0_darwin_arm64":      "def456",
		"si_v0.2.0_windows_amd64.exe": "999aaa",
		"si_v0.2.0_linux_arm64":       "", // absent
	}
	for name, want := range cases {
		if got := sha256From(sums, name); got != want {
			t.Errorf("sha256From(%q)=%q, want %q", name, got, want)
		}
	}
}

func TestSemverLess(t *testing.T) {
	less := [][2]string{{"0.2.0", "0.3.0"}, {"v0.2.0", "0.10.0"}, {"0.3.0", "1.0.0"}, {"0.3.0", "0.3.1"}}
	for _, c := range less {
		if !semverLess(c[0], c[1]) {
			t.Errorf("semverLess(%q,%q)=false, want true", c[0], c[1])
		}
	}
	notLess := [][2]string{{"0.3.0", "0.3.0"}, {"0.3.0", "v0.2.0"}, {"1.0.0", "0.9.9"}} // equal or newer -> no downgrade
	for _, c := range notLess {
		if semverLess(c[0], c[1]) {
			t.Errorf("semverLess(%q,%q)=true, want false", c[0], c[1])
		}
	}
}
