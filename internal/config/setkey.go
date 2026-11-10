package config

import (
	"os"
	"strings"
)

// SetKey sets key: value in the si.conf at path: the first `key:` line is
// replaced, else the line is appended; the file is created when missing.
// Other lines, comments included, are kept.
func SetKey(path, key, value string) error { return editKey(path, key, value, false) }

// UnsetKey removes every `key:` line from the si.conf at path (a missing file
// is fine). Other lines, comments included, are kept.
func UnsetKey(path, key string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return editKey(path, key, "", true)
}

// editKey rewrites the file in place, so an existing file keeps its mode
// (a user si.conf with keys stays 0600).
func editKey(path, key, value string, unset bool) error {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var lines, out []string
	if len(raw) > 0 {
		lines = strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	}
	set := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if k, _, ok := strings.Cut(t, ":"); ok && strings.TrimSpace(k) == key && !strings.HasPrefix(t, "#") {
			if unset || set {
				if !unset {
					out = append(out, l)
				}
				continue
			}
			l, set = key+": "+value, true
		}
		out = append(out, l)
	}
	if !set && !unset {
		out = append(out, key+": "+value)
	}
	text := strings.Join(out, "\n")
	if text != "" {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
