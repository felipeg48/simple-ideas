// Package config reads an optional si.conf: a flat `key: value` file that sets
// project-wide defaults (theme), remaps the single-key commands, and turns
// features on or off. The format is the same key: value style as deck front
// matter, so there is no new syntax and no dependency (stdlib only).
//
// Recognised keys:
//
//	theme: crdb                  # default theme (a CLI --theme or a deck's
//	                             # `theme:` front matter still wins)
//	keys.annotate: w             # remap a command key (see Actions)
//	feature.annotate: off        # disable a feature (see Features)
//
// Blank lines and lines beginning with # are ignored.
package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Actions are the remappable single-key commands (keys.<action>).
var Actions = []string{"overview", "presenter", "draw", "timer", "mode", "fullscreen", "help"}

// Features are the toggleable features (feature.<name>).
var Features = []string{"draw", "timer", "presenter", "overview"}

// Default is an annotated si.conf showing every setting at its built-in value.
// `si --export conf` prints it. Keep it in sync with the deck.js defaults.
const Default = `# si.conf — every value shown is the built-in default.
# Keep only the lines you change; anything omitted uses the default.
# Flat key: value (same style as deck front matter), not YAML.

theme: studio          # studio | paper | mono | crdb (aliases cockroachdb, cockroachlabs)

keys.overview:   o
keys.presenter:  p
keys.draw:       a
keys.timer:      t
keys.mode:       d
keys.fullscreen: f
keys.help:       ?

feature.draw:      on
feature.timer:     on
feature.presenter: on
feature.overview:  on
`

// Config is a parsed si.conf. Zero value is a usable empty config.
type Config struct {
	Theme    string            // default theme, or ""
	Keys     map[string]string // action -> key
	Features map[string]bool   // feature -> enabled
	Warnings []string          // unknown keys / bad values, for the caller to report
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func parseBool(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case "on", "true", "yes", "1":
		return true, true
	case "off", "false", "no", "0":
		return false, true
	}
	return false, false
}

func (c *Config) setKey(action, v string, ln int) {
	if !contains(Actions, action) {
		c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: unknown action %q", ln, action))
		return
	}
	if v == "" {
		c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: keys.%s needs a key", ln, action))
		return
	}
	c.Keys[action] = v
}

func (c *Config) setFeature(name, v string, ln int) {
	if !contains(Features, name) {
		c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: unknown feature %q", ln, name))
		return
	}
	b, ok := parseBool(v)
	if !ok {
		c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: feature.%s wants on/off, got %q", ln, name, v))
		return
	}
	c.Features[name] = b
}

// Parse reads a flat `key: value` config (the same style as deck front matter,
// not YAML). Recognised: `theme:`, `keys.<action>:`, `feature.<name>:`.
func Parse(r io.Reader) *Config {
	c := &Config{Keys: map[string]string{}, Features: map[string]bool{}}
	sc := bufio.NewScanner(r)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: expected `key: value`", ln))
			continue
		}
		k = strings.TrimSpace(k)
		// strip an inline comment: a # with whitespace before it
		for i := 1; i < len(v); i++ {
			if v[i] == '#' && (v[i-1] == ' ' || v[i-1] == '\t') {
				v = v[:i]
				break
			}
		}
		v = strings.TrimSpace(v)
		switch {
		case k == "theme":
			c.Theme = v
		case strings.HasPrefix(k, "keys."):
			c.setKey(strings.TrimPrefix(k, "keys."), v, ln)
		case strings.HasPrefix(k, "feature."), strings.HasPrefix(k, "features."):
			c.setFeature(strings.TrimPrefix(strings.TrimPrefix(k, "features."), "feature."), v, ln)
		default:
			c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: unknown key %q", ln, k))
		}
	}
	return c
}

// Load returns the config from the first path that exists, or an empty config
// if none do. A read/parse of an existing file never fails hard: problems are
// returned as Warnings so a typo in si.conf cannot break a build.
func Load(paths ...string) *Config {
	for _, p := range paths {
		if p == "" {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		c := Parse(f)
		f.Close()
		return c
	}
	return &Config{Keys: map[string]string{}, Features: map[string]bool{}}
}

// Empty reports whether the config sets nothing.
func (c *Config) Empty() bool {
	return c == nil || (c.Theme == "" && len(c.Keys) == 0 && len(c.Features) == 0)
}
