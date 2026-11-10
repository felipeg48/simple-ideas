package config

import (
	"os"
	"strings"
)

// SetKey sets key: value in the si.conf at path: the first `key:` line is
// replaced, else the line is appended; the file is created when missing.
// Other lines, comments included, are kept.
func SetKey(path, key, value string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var lines []string
	if len(raw) > 0 {
		lines = strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	}
	set := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if k, _, ok := strings.Cut(t, ":"); ok && strings.TrimSpace(k) == key && !strings.HasPrefix(t, "#") {
			lines[i], set = key+": "+value, true
			break
		}
	}
	if !set {
		lines = append(lines, key+": "+value)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
