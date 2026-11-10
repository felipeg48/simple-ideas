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
	for f, s := range map[string]string{"all.mp4": "V", "all.vtt": "WEBVTT", "chapters.txt": "0:00 Ask si for help\n0:20 Start a talk\n", "ask-si-for-help.mp4": "A", "start-a-talk.mp4": "B"} {
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
	for _, want := range []string{`src="../videos/decks/all.mp4"`, `src="../videos/decks/all.vtt"`, `href="../videos/decks/start-a-talk.mp4">Start a talk</a> <span>0:20</span>`} {
		if !strings.Contains(out, want) {
			t.Errorf("block lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "nope") {
		t.Error("an area without videos should leave nothing")
	}
	if got["videos/decks/all.mp4"] != "V" || got["videos/decks/start-a-talk.mp4"] != "B" {
		t.Errorf("files not copied: %v", got)
	}
}
