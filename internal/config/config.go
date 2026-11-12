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
	"path/filepath"
	"runtime"
	"strings"
)

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// Actions are the remappable single-key commands (keys.<action>).
var Actions = []string{"overview", "presenter", "draw", "timer", "mode", "fullscreen", "find", "help", "terminal", "remote", "autoplay"}

// Features are the toggleable features (feature.<name>).
var Features = []string{"draw", "timer", "presenter", "overview", "find", "terminal", "autoplay"}

// Default is an annotated si.conf showing every setting at its built-in value.
// `si --export conf` prints it. Keep it in sync with the deck.js defaults.
const Default = `# si.conf — every value shown is the built-in default.
# Keep only the lines you change; anything omitted uses the default.
# Flat key: value (same style as deck front matter), not YAML.

theme: studio          # a built-in theme (si theme list), a .css path or a URL
# fonts: heading=Fraunces:600, body=Inter:400;700   # free web fonts, inlined at build time

keys.overview:   o
keys.presenter:  p
keys.draw:       a
keys.timer:      t
keys.mode:       d
keys.fullscreen: f
keys.find:       /
keys.help:       ?
keys.terminal:   ` + "`" + `          # backtick; play terminal drawer (si play on a loopback address only)
keys.remote:     q          # si play --remote: the phone and tablet links as QR codes
keys.autoplay:   v          # auto-play: the deck reads its Notes: aloud and clicks at each [CLICK]

# si build --handout: the paper of handout PDFs
handout.paper: a4      # a4 or letter

# si play --audience: keep the room's poll counts and questions in
# .si/audience/ when the talk ends (off: keep nothing)
audience.save: on

# a talk with speakers: (Notes[Name]:): who may advance the deck from a
# teleprompter: any (default) or turn (only the speaker whose turn it is)
speakers.control: any

# auto-play (v): how long a slide with no Notes: stays, in seconds (its
# Notes: (N min) share when it has one); voice.<Name>: a speaker's own voice
autoplay.dwell: 3
# voice.Ana: kokoro:af_heart

# draw (a): hold this key and move to draw with the cursor, no click
draw.hold: alt         # alt (⌥ Option), shift, ctrl, meta (⌘ Command), or off

# si demo run and si voice say: the voice when the script and --voice set none
# voice: kokoro:af_heart
# voice.style: warm and unhurried   # a tone, for engines that take one (openai)
# API keys never go here: si voice login elevenlabs|openai keeps them in your
# own si.conf (si voice login prints where), which never goes to git
# si demo run writes <script>.mp4 alone; more with --with, with: or this
# (captions, silent, audio, subtitles, transcript, chapters, clips, report,
# readme, all)
# demo.with: subtitles, chapters

feature.draw:      on
feature.timer:     on
feature.presenter: on
feature.overview:  on
feature.find:      on
feature.terminal:  on   # si play loopback only; always off in a built deck and on --addr 0.0.0.0
feature.autoplay:  on   # the Auto-play button and key

# extra pages for find (/): it already offers index.html, the other decks, labs
# and PDFs of the build; add pages outside it (relative paths or URLs)
# links.Team wiki: https://wiki.example.com/rabbitmq
# links.Lab setup: ../setup/index.html

# break timer (t): presets, unit, a background animation layer, and the face
# timer.time: 5,10,15,20,30,45,60   # preset durations shown in the HUD
# timer.unit: minutes               # minutes (default) or seconds
# timer.face: digital               # digital (default) or analog (sport-clock dial)
# timer.animation: waves,orbit      # built-in: waves | orbit | bokeh; a comma list gives a toggle (off if unset)
# timer.animation.path: ./anim      # optional dir of custom .js/.svg animations
# timer.animation.name: waves       # the file stem(s) to run from that dir
`

// Config is a parsed si.conf. Zero value is a usable empty config.
type Config struct {
	Theme                    string            // default theme, or ""
	Fonts                    string            // default fonts: (a deck's own fonts: wins)
	FontsSubset              string            // default fonts-subset:
	Keys                     map[string]string // action -> key
	Features                 map[string]bool   // feature -> enabled
	Timer                    TimerConfig       // break-timer presets, unit and animation
	Links                    []Link            // links.<label>: <url>, searchable from find (/), in file order
	Budget                   Budget            // budget.page / budget.3d: si check fails on bigger pages
	DrawHold                 string            // draw.hold: the key that draws with the cursor ("" = alt)
	HandoutPaper             string            // handout.paper: a4 (default) or letter
	AudienceOff              bool              // audience.save: off: keep no poll counts or questions
	SpeakersTurn             bool              // speakers.control: turn: only the speaker whose turn it is advances
	Voice                    string            // voice: engine:name for si demo run and si voice say
	VoiceStyle               string            // voice.style: a tone for engines that take one (openai)
	DemoWith                 string            // demo.with: what si demo run writes besides <script>.mp4 (a comma list)
	SpeakerVoices            map[string]string // voice.<Name>: one speaker's voice (auto-play of a talk with speakers:)
	AutoplayDwell            float64           // autoplay.dwell: seconds on a slide with no Notes: (0 = 3)
	ElevenLabsKey, OpenAIKey string            // voice.<engine>.key: kept from the user's si.conf only
	Path                     string            // the project si.conf Load read ("" when none)
	Warnings                 []string          // unknown keys / bad values, for the caller to report

	secretLines []string // "line N: key" of each API key the files set
}

// Budget is the largest a built page may be, in KB (0: not set).
type Budget struct{ Page, ThreeD int }

// Set reports whether si.conf sets any budget.
func (b Budget) Set() bool { return b.Page > 0 || b.ThreeD > 0 }

// sizeKB reads "900", "900 KB", "1.5 MB" as kilobytes.
func sizeKB(v string) (int, bool) {
	v = strings.ToUpper(strings.TrimSpace(v))
	mul := 1.0
	switch {
	case strings.HasSuffix(v, "MB"):
		v, mul = strings.TrimSpace(strings.TrimSuffix(v, "MB")), 1024
	case strings.HasSuffix(v, "KB"):
		v = strings.TrimSpace(strings.TrimSuffix(v, "KB"))
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%g", &f); err != nil || f <= 0 {
		return 0, false
	}
	return int(f*mul + 0.5), true
}

// Link is one links.<label>: <url> line: a page outside the build that find (/) offers.
type Link struct{ Label, URL string }

// TimerConfig holds the break-timer settings (timer.* keys).
type TimerConfig struct {
	Times         []int  // preset durations; empty = built-in 5,10,15,20,30,45,60
	Unit          string // "minutes" (default) or "seconds"
	Animation     string // animation name(s); comma-separated gives a HUD toggle
	AnimationPath string // optional directory of custom animation files
	Face          string // "digital" (default) or "analog" (sport-clock dial)
}

func (c *Config) setTimerTimes(v string, ln int) {
	var out []int
	for _, f := range strings.Split(v, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		n := 0
		if _, err := fmt.Sscanf(f, "%d", &n); err != nil || n <= 0 {
			c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: timer.time wants positive numbers, got %q", ln, f))
			return
		}
		out = append(out, n)
	}
	c.Timer.Times = out
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
	c := empty()
	c.parse(r)
	return c
}

func empty() *Config { return &Config{Keys: map[string]string{}, Features: map[string]bool{}} }

// parse reads r into c: what r sets wins over what c already holds.
func (c *Config) parse(r io.Reader) {
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
		case k == "fonts":
			c.Fonts = v
		case k == "fonts-subset":
			c.FontsSubset = v
		case k == "timer.time":
			c.setTimerTimes(v, ln)
		case k == "timer.unit":
			if v != "minutes" && v != "seconds" {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: timer.unit wants minutes/seconds, got %q", ln, v))
			} else {
				c.Timer.Unit = v
			}
		case k == "timer.animation.path":
			c.Timer.AnimationPath = v
		case k == "timer.animation.name", k == "timer.animation":
			c.Timer.Animation = v
		case k == "timer.face":
			if v != "digital" && v != "analog" {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: timer.face wants digital/analog, got %q", ln, v))
			} else {
				c.Timer.Face = v
			}
		case strings.HasPrefix(k, "links."):
			if label := strings.TrimSpace(strings.TrimPrefix(k, "links.")); label == "" || v == "" {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: links.<label>: <url> needs a label and a URL", ln))
			} else {
				c.Links = append(c.Links, Link{Label: label, URL: v})
			}
		case k == "budget.page", k == "budget.3d":
			kb, ok := sizeKB(v)
			if !ok {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: %s wants a size like 900 KB or 1.5 MB, got %q", ln, k, v))
			} else if k == "budget.page" {
				c.Budget.Page = kb
			} else {
				c.Budget.ThreeD = kb
			}
		case k == "handout.paper":
			if p := strings.ToLower(v); p == "a4" || p == "letter" {
				c.HandoutPaper = p
			} else {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: handout.paper wants a4 or letter, got %q", ln, v))
			}
		case k == "audience.save":
			if on, ok := parseBool(v); ok {
				c.AudienceOff = !on
			} else {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: audience.save wants on or off, got %q", ln, v))
			}
		case k == "speakers.control":
			switch strings.ToLower(v) {
			case "turn":
				c.SpeakersTurn = true
			case "any":
				c.SpeakersTurn = false
			default:
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: speakers.control wants any or turn, got %q", ln, v))
			}
		case k == "voice":
			c.Voice = v
		case k == "voice.style":
			c.VoiceStyle = v
		case k == "demo.with":
			c.DemoWith = v
		case k == "voice.elevenlabs.key":
			c.ElevenLabsKey = v
			c.secretLines = append(c.secretLines, fmt.Sprintf("line %d: %s", ln, k))
		case k == "voice.openai.key":
			c.OpenAIKey = v
			c.secretLines = append(c.secretLines, fmt.Sprintf("line %d: %s", ln, k))
		case k == "autoplay.dwell":
			var f float64
			if _, err := fmt.Sscanf(strings.TrimSuffix(v, "s"), "%g", &f); err != nil || f <= 0 {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: autoplay.dwell wants seconds (3 or 2.5s), got %q", ln, v))
			} else {
				c.AutoplayDwell = f
			}
		case strings.HasPrefix(k, "voice.") && !strings.HasSuffix(k, ".key") && strings.Count(k, ".") == 1:
			if c.SpeakerVoices == nil {
				c.SpeakerVoices = map[string]string{}
			}
			c.SpeakerVoices[strings.TrimPrefix(k, "voice.")] = v
		case k == "draw.hold":
			hold, ok := map[string]string{"alt": "alt", "option": "alt", "shift": "shift", "ctrl": "ctrl", "control": "ctrl",
				"meta": "meta", "cmd": "meta", "command": "meta", "off": "off", "none": "off"}[strings.ToLower(v)]
			if !ok {
				c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: draw.hold wants alt, shift, ctrl, meta or off, got %q", ln, v))
			} else {
				c.DrawHold = hold
			}
		case strings.HasPrefix(k, "keys."):
			c.setKey(strings.TrimPrefix(k, "keys."), v, ln)
		case strings.HasPrefix(k, "feature."), strings.HasPrefix(k, "features."):
			c.setFeature(strings.TrimPrefix(strings.TrimPrefix(k, "features."), "feature."), v, ln)
		default:
			c.Warnings = append(c.Warnings, fmt.Sprintf("line %d: unknown key %q", ln, k))
		}
	}
}

// UserPath is the user's own si.conf, <config dir>/si/si.conf: settings that
// follow the user from project to project, API keys included.
func UserPath() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "si", "si.conf")
}

// readFile parses the file at p into c (its values win) and reports whether
// it exists. Its warnings carry its path; a .css theme it sets is relative
// to it (when the file is there).
func (c *Config) readFile(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	n, theme := len(c.Warnings), c.Theme
	c.parse(f)
	for i := n; i < len(c.Warnings); i++ {
		c.Warnings[i] = p + ": " + c.Warnings[i]
	}
	if t := c.Theme; t != theme && strings.HasSuffix(strings.ToLower(t), ".css") && !filepath.IsAbs(t) && !strings.Contains(t, "://") {
		if q := filepath.Join(filepath.Dir(p), t); fileExists(q) {
			c.Theme = q
		}
	}
	return true
}

// Load returns the user's si.conf (UserPath) overlaid with the first of
// paths that exists: the project's values win. Reading never fails hard:
// problems come back as Warnings, each with its file, so a typo in si.conf
// cannot break a build. API keys count only in the user's file: a project
// si.conf goes to git, so a key there is ignored with a warning.
func Load(paths ...string) *Config {
	c := LoadUser()
	for _, p := range paths {
		if p == "" {
			continue
		}
		before, eleven, openai := len(c.secretLines), c.ElevenLabsKey, c.OpenAIKey
		if !c.readFile(p) {
			continue
		}
		for _, l := range c.secretLines[before:] {
			c.Warnings = append(c.Warnings, p+": "+l+" is ignored: a project si.conf goes to git, so API keys go only in your own "+UserPath()+" (si voice login)")
		}
		c.secretLines, c.ElevenLabsKey, c.OpenAIKey, c.Path = c.secretLines[:before], eleven, openai, p
		break
	}
	return c
}

// LoadUser reads the user's si.conf alone. On unix it warns when the file
// holds an API key and others may read it.
func LoadUser() *Config {
	c := empty()
	p := UserPath()
	if p == "" || !c.readFile(p) {
		return c
	}
	if st, err := os.Stat(p); err == nil && len(c.secretLines) > 0 && runtime.GOOS != "windows" && st.Mode().Perm()&0o077 != 0 {
		c.Warnings = append(c.Warnings, fmt.Sprintf("%s holds an API key and others may read it (%v): chmod 600 %s", p, st.Mode().Perm(), p))
	}
	return c
}

// KeyEnv is the environment variable for each voice engine's API key.
var KeyEnv = map[string]string{"elevenlabs": "ELEVENLABS_API_KEY", "openai": "OPENAI_API_KEY"}

// APIKey is the key for a voice engine (elevenlabs, openai): its environment
// variable first, then voice.<engine>.key in the user's si.conf. from names
// where it came from ("" when there is none).
func APIKey(engine string) (key, from string) {
	env := KeyEnv[engine]
	if env == "" {
		return "", ""
	}
	if k := os.Getenv(env); k != "" {
		return k, env
	}
	u := LoadUser()
	if k := map[string]string{"elevenlabs": u.ElevenLabsKey, "openai": u.OpenAIKey}[engine]; k != "" {
		return k, UserPath()
	}
	return "", ""
}

// Mask shows a secret by its last four characters only.
func Mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

// Empty reports whether the config sets nothing.
func (c *Config) Empty() bool {
	return c == nil || (c.Theme == "" && len(c.Keys) == 0 && len(c.Features) == 0 &&
		len(c.Timer.Times) == 0 && c.Timer.Unit == "" && c.Timer.Animation == "" && c.Timer.AnimationPath == "" && c.Timer.Face == "")
}
