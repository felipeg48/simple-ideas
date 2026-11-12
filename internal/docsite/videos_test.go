package docsite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVideos(t *testing.T) {
	dir := t.TempDir()
	area := filepath.Join(dir, "decks")
	os.MkdirAll(area, 0o755)
	for f, s := range map[string]string{"decks.mp4": "V", "decks.vtt": "WEBVTT", "chapters.txt": "0:00 Ask si for help\n0:20 Start a talk\n", "ask-si-for-help.mp4": "A", "start-a-talk.mp4": "B", "start-a-talk.vtt": "WEBVTT b"} {
		os.WriteFile(filepath.Join(area, f), []byte(s), 0o644)
	}
	got := map[string]string{}
	write := func(rel, text string) error { got[rel] = text; return nil }

	t.Setenv("SI_DEMO_VIDEOS", "")
	if out := newVideos(write).replace("# D\n\n<!-- si-video: decks -->\n\nText\n", "features/decks.md"); strings.Contains(out, "si-video") || strings.Contains(out, "<video") {
		t.Errorf("without videos the line should go: %q", out)
	}

	t.Setenv("SI_DEMO_VIDEOS", dir)
	out := newVideos(write).replace("# D\n\n<!-- si-video: decks -->\n<!-- si-video: nope -->\nText\n", "features/decks.md")
	for _, want := range []string{`src="../videos/decks/decks.mp4"`, `src="../videos/decks/decks.vtt"`, `href="../videos/decks/start-a-talk.mp4">Start a talk</a> <span>0:20</span>`} {
		if !strings.Contains(out, want) {
			t.Errorf("block lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "nope") {
		t.Error("an area without videos should leave nothing")
	}
	if got["videos/decks/decks.mp4"] != "V" || got["videos/decks/start-a-talk.mp4"] != "B" || got["videos/decks/start-a-talk.vtt"] != "WEBVTT b" {
		t.Errorf("files not copied: %v", got)
	}
}

// A release filmed before v2.41.0 has all.mp4: the site still shows it.
func TestVideosLegacyNames(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "decks"), 0o755)
	os.WriteFile(filepath.Join(dir, "decks", "all.mp4"), []byte("OLD"), 0o644)
	got := map[string]string{}
	t.Setenv("SI_DEMO_VIDEOS", dir)
	out := newVideos(func(rel, text string) error { got[rel] = text; return nil }).replace("<!-- si-video: decks -->\n", "features/decks.md")
	if !strings.Contains(out, `src="../videos/decks/decks.mp4"`) || got["videos/decks/decks.mp4"] != "OLD" {
		t.Errorf("legacy all.mp4 not used: %q %v", out, got)
	}
}
