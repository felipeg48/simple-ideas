// Package home is si's own folder, ~/.si ($SI_HOME overrides it): the user's
// si.conf, logins, si's certificate authority, running sessions, Kokoro and
// the caches. Before v2.43.0 these lived in the system's config and cache
// folders; Path moves an old copy into ~/.si the first time it is asked for.
package home

import (
	"os"
	"path/filepath"
)

// Dir is ~/.si, or $SI_HOME.
func Dir() string {
	if d := os.Getenv("SI_HOME"); d != "" {
		return d
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "si")
	}
	return filepath.Join(h, ".si")
}

// Where an item lived before v2.43.0.
const (
	Config = "config" // os.UserConfigDir()/si
	Cache  = "cache"  // os.UserCacheDir()/si
)

// Path is rel inside Dir. When it is not there yet and the old place (was:
// Config or Cache, with the same rel, or old if given) has it, it is moved
// over first; if the move fails (another disk), the old place is used.
func Path(rel, was string, old ...string) string {
	p := filepath.Join(Dir(), filepath.FromSlash(rel))
	if os.Getenv("SI_HOME") != "" { // tests and explicit homes never touch the old places
		return p
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	from := legacy(was, rel)
	if len(old) > 0 {
		from = legacy(was, old[0])
	}
	if from == "" {
		return p
	}
	if _, err := os.Stat(from); err != nil {
		return p
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return from
	}
	if err := os.Rename(from, p); err != nil {
		return from
	}
	return p
}

func legacy(was, rel string) string {
	var d string
	var err error
	switch was {
	case Config:
		d, err = os.UserConfigDir()
	case Cache:
		d, err = os.UserCacheDir()
	default:
		return ""
	}
	if err != nil {
		return ""
	}
	return filepath.Join(d, "si", filepath.FromSlash(rel))
}
