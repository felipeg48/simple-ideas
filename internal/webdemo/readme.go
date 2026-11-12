package webdemo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// describe says what a file of a run's output folder is (script: the
// script's name, the stem of the whole run's files).
func describe(name, script string) string {
	switch name {
	case script + ".mp4":
		return "every feature in one video: picture and narration, a title card each, chapters embedded"
	case script + ".silent.mp4":
		return "every feature, the picture alone (no sound), chapters embedded"
	case script + ".captions.mp4":
		return "every feature with the narration and the captions under the picture, chapters embedded"
	case script + ".m4a":
		return "the narration of " + script + ".mp4 alone (AAC), on its clock"
	case script + ".srt", script + ".vtt":
		return "captions of " + script + ".mp4, timed"
	case script + ".txt":
		return "the transcript: what each feature says, as prose"
	case "chapters.txt":
		return "chapter lines for a YouTube description"
	case "report.html":
		return "the run: each step, its time and a picture; what failed and why"
	case "report.json":
		return "the same run, for tools"
	case "README.md", "README.html":
		return "this page"
	case "shots":
		return "the pictures of the report (screenshot steps, failures)"
	}
	for _, k := range []struct{ suffix, what string }{
		{".silent.mp4", "the feature, the picture alone: no sound, no captions"},
		{".captions.mp4", "the feature: picture, narration and the captions in a band under the picture"},
		{".m4a", "the feature's narration alone (AAC)"},
		{".srt", "the feature's captions, timed (SubRip)"},
		{".vtt", "the same captions as WebVTT, for a web page's captions track"},
		{".log", "a start: program's output"},
		{".mp4", "the feature: picture and narration (H.264 + AAC), no captions"},
	} {
		if strings.HasSuffix(name, k.suffix) {
			return k.what
		}
	}
	return ""
}

// Readme is README.md of an output folder: the files the run made
// (rep.Files), the whole run's video, each feature with its clip when there
// are clips, the script, how it was made. how is the command that made it.
func Readme(rep *Report, s *Script, how string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: article\ntitle: %q\n---\n\n# %s\n\n", rep.Title, rep.Title)
	fmt.Fprintf(&b, "Made by `si demo run` from `%s`: %d feature(s), %d passed, %d failed.\n\n", rep.Script, len(rep.Features), rep.Passed, rep.Failed)
	has := map[string]bool{}
	for _, f := range rep.Files {
		has[f] = true
	}
	if rep.Video != "" {
		b.WriteString(video(rep.Video, has))
	}
	b.WriteString("## Files\n\n| File | What |\n|---|---|\n")
	for _, n := range rep.Files {
		link := n
		if n == "shots" {
			link = "shots/"
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s |\n", n, link, describe(n, s.Name))
	}
	b.WriteString("\n## Features\n\n")
	for _, f := range rep.Features {
		fmt.Fprintf(&b, "### %s\n\n", f.Name)
		if f.What != "" {
			fmt.Fprintf(&b, "%s\n\n", f.What)
		}
		fmt.Fprintf(&b, "%s in %.1fs.", strings.ToUpper(f.Status[:1])+f.Status[1:], f.Seconds)
		if f.Video == "" {
			b.WriteString("\n\n")
			continue
		}
		stem := strings.TrimSuffix(f.Video, ".mp4")
		fmt.Fprintf(&b, " [%s](%s)", f.Video, f.Video)
		for _, k := range []struct{ ext, label string }{{".captions.mp4", "with captions"}, {".silent.mp4", "picture only"}, {".srt", "captions"}, {".m4a", "narration"}} {
			if has[stem+k.ext] {
				fmt.Fprintf(&b, " · [%s](%s%s)", k.label, stem, k.ext)
			}
		}
		b.WriteString("\n\n" + video(f.Video, has))
	}
	if raw, err := os.ReadFile(s.Path); err == nil {
		lang := strings.TrimPrefix(filepath.Ext(s.Path), ".")
		if lang == "md" {
			lang = "markdown"
		}
		fmt.Fprintf(&b, "## The script\n\n`%s`:\n\n~~~~%s\n%s\n~~~~\n\n", rep.Script, lang, strings.TrimRight(string(raw), "\n"))
	}
	b.WriteString("## How it was made\n\n")
	fmt.Fprintf(&b, "- Command: `%s`\n", how)
	if rep.Voice != "" {
		fmt.Fprintf(&b, "- Voice: %s\n", rep.Voice)
	}
	fmt.Fprintf(&b, "- App: %s, viewport %s\n- Browser: %s (headless, over the DevTools protocol)\n- Video: ffmpeg, 30 fps H.264, AAC\n- si %s, %s\n", rep.App, rep.Viewport, rep.Chrome, rep.SI, rep.Date)
	return b.String()
}

// video is a <video> of name, with its .vtt as the captions track when it
// was made.
func video(name string, has map[string]bool) string {
	track := ""
	if vtt := strings.TrimSuffix(name, ".mp4") + ".vtt"; has[vtt] {
		track = fmt.Sprintf(`<track kind="captions" src="%s" srclang="en" label="Captions" default>`, vtt)
	}
	return fmt.Sprintf("<video controls preload=\"metadata\" src=\"%s\" style=\"width:100%%;height:auto\">%s</video>\n\n", name, track)
}
