package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublishCourseSkillJarDryRun: a dry run writes the files Skilljar
// fetches (one SCORM package per slides, lab and quiz lesson, the video),
// prints the requests, sends nothing, saves no ids and never publishes an
// instructor demo; --yes without --base-url is refused.
func TestPublishCourseSkillJarDryRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SI_PUBLISH_CONFIG", filepath.Join(tmp, "publish.json"))
	src := filepath.Join(tmp, "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	stdout.Reset()
	if code := run([]string{"publish", src, "--to", "skilljar", "--base-url", "https://files.example/web101/"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry run: %d\n%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"Dry run", "POST https://api.skilljar.com/v1/courses", `"type": "SECTION"`, `"type": "HTML"`,
		`"content_url": "https://files.example/web101/02-html-quiz-scorm12.zip"`, `"content_url": "https://files.example/web101/01-web-page-load.mp4"`} {
		if !strings.Contains(out, want) {
			t.Errorf("the dry run lacks %q", want)
		}
	}
	if strings.Contains(out, "Inspect a page live") || strings.Contains(out, "inspect") {
		t.Error("an instructor demo is never published")
	}
	if _, err := os.Stat(sjIDs(src)); err == nil {
		t.Error("a dry run saves no ids")
	}
	checkManifest(t, filepath.Join(sjFiles(src), "02-html-quiz-scorm12.zip"))
	if code := run([]string{"publish", "login", "skilljar", "--token", "sk-secret"}, &stdout, &stderr); code != 0 {
		t.Fatalf("login: %s", stderr.String())
	}
	if fi, err := os.Stat(filepath.Join(tmp, "publish.json")); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the login file is 0600: %v %v", fi, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"publish", src, "--to", "skilljar", "--yes"}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "--base-url") {
		t.Errorf("--yes without --base-url: %d %s", code, stderr.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "sk-secret") {
		t.Error("the key is never printed")
	}
}
