package config

import (
	"strings"
	"testing"
)

func TestDefaultParsesClean(t *testing.T) {
	c := Parse(strings.NewReader(Default))
	if len(c.Warnings) != 0 {
		t.Fatalf("Default should parse with no warnings, got %v", c.Warnings)
	}
	if c.Theme != "studio" {
		t.Errorf("Default theme = %q, want studio", c.Theme)
	}
	for _, a := range Actions {
		if c.Keys[a] == "" {
			t.Errorf("Default is missing keys.%s", a)
		}
	}
	for _, f := range Features {
		if !c.Features[f] {
			t.Errorf("Default feature.%s should be on", f)
		}
	}
}

func TestParseFlatAndWarnings(t *testing.T) {
	src := `
theme: crdb
keys.draw: w
feature.timer: off
keys.bogus: x
feature.nope: on
feature.timer: maybe
no-colon-line
`
	c := Parse(strings.NewReader(src))
	if c.Keys["draw"] != "w" {
		t.Errorf("keys.draw = %q, want w", c.Keys["draw"])
	}
	if v, ok := c.Features["timer"]; !ok || v {
		t.Errorf("feature.timer = %v,%v, want false,true", v, ok)
	}
	// unknown action, unknown feature, bad bool, and the no-colon line → 4 warnings
	if len(c.Warnings) != 4 {
		t.Errorf("warnings = %d (%v), want 4", len(c.Warnings), c.Warnings)
	}
	if c.Empty() {
		t.Error("config should not be Empty")
	}
}

func TestLoadMissing(t *testing.T) {
	c := Load("/no/such/file.conf", "")
	if !c.Empty() {
		t.Errorf("missing files should yield an empty config, got %+v", c)
	}
}

func TestParseTimer(t *testing.T) {
	c := Parse(strings.NewReader("timer.time: 10, 15, 30\ntimer.unit: seconds\ntimer.animation: waves,orbit\ntimer.face: analog\n"))
	if len(c.Warnings) != 0 {
		t.Fatalf("warnings: %v", c.Warnings)
	}
	if len(c.Timer.Times) != 3 || c.Timer.Times[0] != 10 || c.Timer.Times[2] != 30 {
		t.Errorf("times=%v", c.Timer.Times)
	}
	if c.Timer.Unit != "seconds" || c.Timer.Animation != "waves,orbit" || c.Timer.Face != "analog" {
		t.Errorf("unit=%q anim=%q face=%q", c.Timer.Unit, c.Timer.Animation, c.Timer.Face)
	}
	// path + name variant
	c2 := Parse(strings.NewReader("timer.animation.path: ./a\ntimer.animation.name: x\ntimer.unit: hours\n"))
	if c2.Timer.AnimationPath != "./a" || c2.Timer.Animation != "x" {
		t.Errorf("path=%q name=%q", c2.Timer.AnimationPath, c2.Timer.Animation)
	}
	if len(c2.Warnings) != 1 { // bad unit "hours"
		t.Errorf("expected 1 warning for bad unit, got %v", c2.Warnings)
	}
}

func TestLinks(t *testing.T) {
	c := Parse(strings.NewReader("links.Team wiki: https://wiki.example.com/x#top\nlinks.Setup: ../setup/index.html\nlinks.: nothing\n"))
	if len(c.Links) != 2 || c.Links[0] != (Link{"Team wiki", "https://wiki.example.com/x#top"}) || c.Links[1].URL != "../setup/index.html" {
		t.Errorf("links: %+v", c.Links)
	}
	if len(c.Warnings) != 1 {
		t.Errorf("an empty label warns: %v", c.Warnings)
	}
}

func TestParseBudget(t *testing.T) {
	c := Parse(strings.NewReader("budget.page: 900 KB\nbudget.3d: 1.5 MB\n"))
	if c.Budget.Page != 900 || c.Budget.ThreeD != 1536 || !c.Budget.Set() || len(c.Warnings) != 0 {
		t.Errorf("budget: %+v %v", c.Budget, c.Warnings)
	}
	if c := Parse(strings.NewReader("budget.page: big\n")); len(c.Warnings) != 1 || c.Budget.Set() {
		t.Errorf("a bad size warns: %+v %v", c.Budget, c.Warnings)
	}
}

func TestDrawHold(t *testing.T) {
	for v, want := range map[string]string{"alt": "alt", "Option": "alt", "shift": "shift", "control": "ctrl", "cmd": "meta", "off": "off"} {
		if c := Parse(strings.NewReader("draw.hold: " + v + "\n")); c.DrawHold != want || len(c.Warnings) > 0 {
			t.Errorf("draw.hold: %s → %q %v, want %q", v, c.DrawHold, c.Warnings, want)
		}
	}
	if c := Parse(strings.NewReader("draw.hold: space\n")); c.DrawHold != "" || len(c.Warnings) != 1 {
		t.Errorf("draw.hold: space should warn and keep the default: %q %v", c.DrawHold, c.Warnings)
	}
}
func TestAudienceSave(t *testing.T) {
	if c := Parse(strings.NewReader("audience.save: off\n")); !c.AudienceOff || len(c.Warnings) > 0 {
		t.Errorf("audience.save: off → %v %v", c.AudienceOff, c.Warnings)
	}
	if c := Parse(strings.NewReader("audience.save: maybe\n")); c.AudienceOff || len(c.Warnings) != 1 {
		t.Errorf("audience.save: maybe should warn and keep saving")
	}
}

func TestSpeakersControl(t *testing.T) {
	if c := Parse(strings.NewReader("speakers.control: turn\n")); !c.SpeakersTurn || len(c.Warnings) > 0 {
		t.Errorf("speakers.control: turn → %v %v", c.SpeakersTurn, c.Warnings)
	}
	if c := Parse(strings.NewReader("speakers.control: me\n")); c.SpeakersTurn || len(c.Warnings) != 1 {
		t.Errorf("speakers.control: me should warn")
	}
}

func TestAutoplayKeys(t *testing.T) {
	c := Parse(strings.NewReader("keys.autoplay: P\nfeature.autoplay: off\nautoplay.dwell: 2.5s\nvoice.Ana: kokoro:af_heart\nvoice.style: calm\nautoplay.dwell: soon\n"))
	if c.Keys["autoplay"] != "P" || c.Features["autoplay"] || c.AutoplayDwell != 2.5 {
		t.Errorf("autoplay = key %q feature %v dwell %g", c.Keys["autoplay"], c.Features["autoplay"], c.AutoplayDwell)
	}
	if c.SpeakerVoices["Ana"] != "kokoro:af_heart" || c.VoiceStyle != "calm" || len(c.SpeakerVoices) != 1 {
		t.Errorf("voices = %v, style %q", c.SpeakerVoices, c.VoiceStyle)
	}
	if len(c.Warnings) != 1 || !strings.Contains(c.Warnings[0], "autoplay.dwell") {
		t.Errorf("warnings = %v", c.Warnings)
	}
}
