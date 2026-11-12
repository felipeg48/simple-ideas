package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

const demoUsage = `Usage:
  si demo test <script.md|.adoc|.json> [--feature "Name"] [--junit report.xml] [--viewport V] [-o dir]
        run a web demo script (type: demo with app:) in headless Chrome as a test: one line
        per step, report.html + report.json in demo-out/<script>/ (or -o dir), exit 1 on a failure;
        --junit also writes JUnit XML (one test case per feature) for CI
  si demo run <script> [--feature "Name"] [--voice engine:name] [--viewport V] [--junit report.xml] [-o dir]
              [--with captions,subtitles,…] [--fast-forward 3s | --no-fast-forward] [--region "<css>" | --crop x,y,w,h]
        perform the same steps on film: a drawn cursor glides to each target, clicks ripple,
        note steps point at elements, fields typed from $VARs are blurred. Each Say: line is
        voiced first and sets the pace. Writes one file: <script>.mp4, every feature after its
        title card, picture and voice, chapters inside. Every step is still checked (✓/✗,
        exit 1 on a failure). More only when asked for:
        --with    a comma list (may repeat): captions (<script>.captions.mp4, the picture shrunk
                  above a band with the captions), silent (<script>.silent.mp4, no sound),
                  audio (<script>.m4a), subtitles (<script>.srt + .vtt), transcript
                  (<script>.txt, what Say: says), chapters (chapters.txt for YouTube), clips
                  (<feature>.mp4 each, plus the kinds above per feature), report (report.html +
                  report.json), readme (README.md + README.html) or all. Default: with: in
                  front matter, else demo.with: in si.conf, else none. --captions = --with captions
        --voice   kokoro[:af_heart] (the default when installed), system[:name] (say, espeak-ng,
                  Windows voices), elevenlabs:<voice id> and openai[:alloy] (a key: si voice
                  login; voice-style: sets openai's tone); else voice: in front matter or si.conf.
                  si voice install kokoro sets Kokoro up; si voice say "line" tries a voice
        --fast-forward 3s  a wait (run … · wait, wait for output, start … · wait for …) plays
                  at speed this long, then faster (a ⏩ 6× badge, at most 2s); the narration
                  never speeds up. --no-fast-forward keeps every wait at speed
        --viewport  phone (400x800), tablet (820x1180), desktop (1440x900) or WxH
        --region "<css>"  crop every clip (and the screenshot step) to that element's box;
                  --crop x,y,w,h crops to a fixed pixel box. A feature's Region: line and the
                  front matter region: override; default is the whole page
  Both run start: programs first and stop them at the end (and on Ctrl-C); terminal "Name"
  steps open a real shell, filmed as a framed window (RULES.md §27, Terminals and programs).
`

// runDemo is `si demo test|run <script>`. Its flags are its own: --feature
// takes a name here, unlike si report's --feature.
func runDemo(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si demo", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	var feature, junit, out, viewport, voice, region, crop string
	var help, captions, noCaptions, noFF bool
	var with multi
	fastFwd := 3 * time.Second
	fl.StringVar(&feature, "feature", "", "")
	fl.StringVar(&junit, "junit", "", "")
	fl.StringVar(&out, "output", "", "")
	fl.StringVar(&out, "o", "", "")
	fl.StringVar(&viewport, "viewport", "", "")
	fl.StringVar(&voice, "voice", "", "")
	fl.Var(&with, "with", "")
	fl.BoolVar(&captions, "captions", false, "")      // = --with captions
	fl.BoolVar(&noCaptions, "no-captions", false, "") // gone in v2.41.0: an error that says why
	fl.StringVar(&region, "region", "", "")
	fl.StringVar(&crop, "crop", "", "")
	fl.DurationVar(&fastFwd, "fast-forward", fastFwd, "")
	fl.BoolVar(&noFF, "no-fast-forward", false, "")
	fl.BoolVar(&help, "help", false, "")
	fl.BoolVar(&help, "h", false, "")
	var args []string
	for {
		if err := fl.Parse(argv); err != nil {
			fmt.Fprintf(stderr, "si demo: %v\n\n%s", err, demoUsage)
			return exitUsage
		}
		argv = fl.Args()
		if len(argv) == 0 {
			break
		}
		args, argv = append(args, argv[0]), argv[1:]
	}
	if help {
		fmt.Fprint(stdout, demoUsage)
		return exitOK
	}
	if noCaptions {
		fmt.Fprintf(stderr, "si demo: --no-captions is gone: si demo run makes no captions unless asked (--with captions)\n")
		return exitUsage
	}
	if captions {
		with = append(with, "captions")
	}
	if len(args) != 2 || (args[0] != "test" && args[0] != "run") {
		fmt.Fprintf(stderr, "si demo: test or run, and one script\n\n%s", demoUsage)
		return exitUsage
	}
	film := args[0] == "run"
	if !film && len(with) > 0 {
		fmt.Fprintf(stderr, "si demo: --with and --captions are for si demo run (si demo test always writes the report)\n")
		return exitUsage
	}
	if _, _, err := demoWith(with, "", ""); err != nil {
		fmt.Fprintf(stderr, "si demo: --%v\n", err)
		return exitUsage
	}
	s, err := webdemo.ParseFile(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	for _, w := range s.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	if len(s.Problems) > 0 {
		for _, p := range s.Problems {
			fmt.Fprintln(stderr, p)
		}
		return exitProblems
	}
	if err := s.SetViewport(viewport); err != nil {
		fmt.Fprintf(stderr, "si demo: --%v\n", err)
		return exitUsage
	}
	if region != "" && crop != "" {
		fmt.Fprintf(stderr, "si demo: use --region or --crop, not both\n")
		return exitUsage
	}
	if crop != "" {
		if _, err := webdemo.ParseCrop(crop); err != nil {
			fmt.Fprintf(stderr, "si demo: --%v\n", err)
			return exitUsage
		}
	}
	o := webdemo.Options{Feature: feature, Log: stdout, SI: version, Region: region, Crop: crop}
	if film {
		// --voice > front matter > si.conf beside the script > the user's si.conf > default
		cfg := config.Load("", filepath.Join(filepath.Dir(args[1]), "si.conf"))
		for _, w := range cfg.Warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
		v, err := chooseVoice(voice, s.Voice, cfg.Voice)
		if err != nil {
			fmt.Fprintf(stderr, "si demo: %v\n", err)
			return exitUsage
		}
		if s.Style == "" {
			s.Style = cfg.VoiceStyle
		}
		w, warn, _ := demoWith(with, s.With, cfg.DemoWith) // --with was checked above, with: by the parse
		if warn != "" {
			fmt.Fprintf(stderr, "warning: %s\n", warn)
		}
		o.Video, o.Voice, o.With, o.FastForward = true, v, w, fastFwd
		if noFF || fastFwd < 0 {
			o.FastForward = 0
		}
		if o.FFmpeg = tools.Find("ffmpeg"); o.FFmpeg == "" {
			fmt.Fprintf(stderr, "si: %s\n", tools.Need("ffmpeg", "si demo run"))
			return exitProblems
		}
	}
	if o.Chrome = findChrome(); o.Chrome == "" {
		fmt.Fprintf(stderr, "si: %s\n", tools.Need("chrome", "si demo"))
		return exitProblems
	}
	if out == "" {
		out = filepath.Join("demo-out", s.Name)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	o.Out = out
	rep, err := webdemo.Run(s, o)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if !film { // a run writes its report itself, when asked for (--with report)
		if err := rep.Write(out); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	if junit != "" {
		b, err := rep.JUnit()
		if err == nil {
			err = os.WriteFile(junit, b, 0o644)
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	if film && o.With.Readme {
		if err := demoReadme(rep, s, out, "si demo "+strings.Join(os.Args[2:], " ")); err != nil {
			fmt.Fprintf(stderr, "si: README: %v\n", err)
			return exitProblems
		}
	}
	fmt.Fprintf(stdout, "si demo: %d passed, %d failed in %.1fs", rep.Passed, rep.Failed, rep.Seconds)
	switch {
	case !film:
		fmt.Fprintf(stdout, "; report: %s\n", filepath.Join(out, "report.html"))
	case len(rep.Files) == 0:
		fmt.Fprintln(stdout, "; nothing filmed")
	default:
		fmt.Fprintf(stdout, "; in %s: %s\n", out, strings.Join(rep.Files, ", "))
	}
	if !rep.OK() {
		return exitProblems
	}
	return exitOK
}

// demoWith is what si demo run writes besides <script>.mp4: --with (every
// list given) > with: in front matter > demo.with: in si.conf (the project's
// over the user's, as config.Load merges them) > nothing. A bad --with or
// with: is an error; a bad si.conf value is a warning and counts as none.
func demoWith(flags []string, front, conf string) (webdemo.With, string, error) {
	var w webdemo.With
	switch {
	case len(flags) > 0:
		for _, f := range flags {
			if err := w.Add(f); err != nil {
				return webdemo.With{}, "", err
			}
		}
	case front != "":
		if err := w.Add(front); err != nil {
			return webdemo.With{}, "", err
		}
	default:
		if err := w.Add(conf); err != nil {
			return webdemo.With{}, "si.conf: demo." + err.Error(), nil
		}
	}
	return w, "", nil
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

// demoReadme writes README.md and, rendered by si as an article with the
// videos linked (not inlined), README.html.
func demoReadme(rep *webdemo.Report, s *webdemo.Script, out, how string) error {
	md := filepath.Join(out, "README.md")
	if err := os.WriteFile(md, []byte(webdemo.Readme(rep, s, how)), 0o644); err != nil {
		return err
	}
	d, err := deck.Parse(md)
	if err != nil {
		return err
	}
	res := render.Deck(d, render.Options{Media: render.MediaLink})
	if len(res.Problems) > 0 {
		return fmt.Errorf("%s", res.Problems[0])
	}
	return os.WriteFile(filepath.Join(out, "README.html"), res.HTML, 0o644)
}

// webDemoProblems prints a web demo script's problems (a bad step, with its
// line) and warnings (an unset $VAR) and returns the number of problems.
func webDemoProblems(path string, stderr io.Writer) int {
	s, err := webdemo.ParseFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return 1
	}
	for _, p := range s.Problems {
		fmt.Fprintln(stderr, p)
	}
	for _, w := range s.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	return len(s.Problems)
}

// checkWebDemo is si check on a JSON web demo script.
func checkWebDemo(path string, stdout, stderr io.Writer) int {
	if n := webDemoProblems(path, stderr); n > 0 {
		fmt.Fprintf(stderr, "si check: %d problem(s) in %s\n", n, path)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si check: %s OK\n", path)
	return exitOK
}
