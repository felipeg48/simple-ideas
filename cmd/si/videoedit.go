package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/videdit"
)

// si build / si check of a video edit (type: video, RULES.md §32): the
// edited video from the source and the timeline's lines.

func videoProblems(e *videdit.Edit, stderr io.Writer) int {
	n := 0
	name := relPath(e.Path)
	for _, p := range e.Problems {
		if p.Warn {
			fmt.Fprintf(stderr, "warning: %s:%d: %s\n", name, p.Line, p.Msg)
			continue
		}
		fmt.Fprintf(stderr, "%s:%d: %s\n", name, p.Line, p.Msg)
		n++
	}
	return n
}

func relPath(p string) string {
	wd, _ := os.Getwd()
	if r, err := filepath.Rel(wd, p); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return p
}

// checkVideo checks a video edit and returns its problem count.
func checkVideo(f string, stdout, stderr io.Writer) int {
	e, err := videdit.ParseFile(f)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return 1
	}
	if note := e.Check(videdit.FFprobe()); note != "" {
		fmt.Fprintf(stdout, "  note: %s: %s\n", relPath(e.Path), note)
	}
	return videoProblems(e, stderr)
}

// buildVideo writes a video edit's out: (into -o when given).
func buildVideo(src string, o *options, stdout, stderr io.Writer) int {
	e, err := videdit.ParseFile(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	ff, fp := videdit.FFmpeg(), videdit.FFprobe()
	if ff == "" || fp == "" {
		fmt.Fprintf(stderr, "si: %s\n", tools.Need("ffmpeg", "video edits (type: video)"))
		return exitProblems
	}
	e.Check(fp)
	if videoProblems(e, stderr) > 0 {
		fmt.Fprintln(stderr, "si: FAIL")
		return exitProblems
	}
	vo := videdit.Options{Chrome: findChrome(), FFmpeg: ff, FFprobe: fp, OutDir: o.output, Voice: o.voice, Log: stdout, SI: version,
		Config: config.Load(o.conf, filepath.Join(e.Dir, "si.conf"))}
	for _, w := range strings.Split(o.with, ",") {
		switch w = strings.TrimSpace(strings.ToLower(w)); w {
		case "":
		case "captions":
			vo.Captions = true
		case "vertical":
			vo.Vertical = true
		case "highlight":
			vo.Highlight = true
		default:
			return usageErr(stderr, fmt.Sprintf("--with %q: a video edit takes captions, vertical, highlight", w))
		}
	}
	// fresh: the parse again (Check resolved the times without joining)
	if e, err = videdit.ParseFile(src); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	res, err := videdit.Build(e, vo)
	videoProblems(&videdit.Edit{Path: e.Path, Problems: warningsOnly(e.Problems)}, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %s: %v\n", relPath(e.Path), err)
		return exitProblems
	}
	how := ""
	if res.Fast {
		how = "  (the new take spliced in; the rest copied as it was)"
	}
	fmt.Fprintf(stdout, "  %-28s %s%s\n", relPath(res.Out), videdit.Stamp(res.Len), how)
	for _, f := range res.Files {
		fmt.Fprintf(stdout, "  %s\n", relPath(f))
	}
	return exitOK
}

func warningsOnly(ps []videdit.Problem) []videdit.Problem {
	var out []videdit.Problem
	for _, p := range ps {
		if p.Warn {
			out = append(out, p)
		}
	}
	return out
}
