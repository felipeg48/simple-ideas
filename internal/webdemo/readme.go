package webdemo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// describe says what a file of the output folder is.
func describe(name string) string {
	switch name {
	case "all.mp4":
		return "every feature in one video: a title card each, chapters embedded"
	case "all.video.mp4":
		return "every feature, the picture alone (no sound), chapters embedded"
	case "all.captions.mp4":
		return "every feature with the narration and the captions under the picture, chapters embedded"
	case "all.srt", "all.vtt":
		return "captions of all.mp4, timed"
	}
	for _, k := range []struct{ suffix, what string }{
		{".video.mp4", "the picture alone: no sound, no captions"},
		{".captions.mp4", "picture, narration and the captions in a band under the picture"},
		{".voice.m4a", "the narration alone (AAC)"},
		{".voice.wav", "the narration alone (WAV), to edit or re-voice"},
		{".srt", "captions of the narration, timed (SubRip)"},
		{".vtt", "the same captions as WebVTT, for a web page's captions track"},
	} {
		if strings.HasSuffix(name, k.suffix) {
			return k.what
		}
	}
	switch name {
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
	if strings.HasSuffix(name, ".mp4") {
		return "the feature: picture and narration (H.264 + AAC), no captions"
	}
	return ""
}

// Readme is README.md of an output folder: every file, each feature with its
// video, the script, how it was made. how is the command that made it.
func Readme(rep *Report, s *Script, dir, how string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: article\ntitle: %q\n---\n\n# %s\n\n", rep.Title, rep.Title)
	fmt.Fprintf(&b, "Made by `si demo run` from `%s`: %d feature(s), %d passed, %d failed.\n\n", rep.Script, len(rep.Features), rep.Passed, rep.Failed)
	b.WriteString("## Files\n\n| File | What |\n|---|---|\n")
	ents, _ := os.ReadDir(dir)
	var names []string
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), ".") && describe(e.Name()) != "" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		link := n
		if n == "shots" {
			link = "shots/"
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s |\n", n, link, describe(n))
	}
	b.WriteString("\n## Features\n\n")
	for _, f := range rep.Features {
		fmt.Fprintf(&b, "### %s\n\n", f.Name)
		if f.What != "" {
			fmt.Fprintf(&b, "%s\n\n", f.What)
		}
		fmt.Fprintf(&b, "%s in %.1fs.", strings.ToUpper(f.Status[:1])+f.Status[1:], f.Seconds)
		if f.Video != "" {
			stem := strings.TrimSuffix(f.Video, ".mp4")
			has := func(ext string) bool { _, err := os.Stat(filepath.Join(dir, stem+ext)); return err == nil }
			fmt.Fprintf(&b, " [%s](%s)", f.Video, f.Video)
			if has(".captions.mp4") {
				fmt.Fprintf(&b, " · [with captions](%s.captions.mp4)", stem)
			}
			if has(".video.mp4") {
				fmt.Fprintf(&b, " · [picture only](%s.video.mp4)", stem)
			}
			track := ""
			if has(".srt") {
				fmt.Fprintf(&b, " · [captions](%s.srt) · [narration](%s.voice.m4a)", stem, stem)
			}
			if has(".vtt") {
				track = fmt.Sprintf(`<track kind="captions" src="%s.vtt" srclang="en" label="Captions" default>`, stem)
			}
			fmt.Fprintf(&b, "\n\n<video controls preload=\"metadata\" src=\"%s\" style=\"width:100%%;height:auto\">%s</video>\n\n", f.Video, track)
		} else {
			b.WriteString("\n\n")
		}
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
