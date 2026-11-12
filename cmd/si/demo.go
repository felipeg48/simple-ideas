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
	"github.com/felipeg48/simple-ideas/internal/pinned"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

const demoUsage = `Usage:
  si demo test <script.md|.adoc|.json|folder/|source> [--feature "Name"] [--junit report.xml] [--viewport V] [-o dir] [--trust]
        run a web demo script (type: demo with app:) in headless Chrome as a test: one line
        per step, report.html + report.json in demo-out/<script>/ (or -o dir), exit 1 on a failure;
        --junit also writes JUnit XML (one test case per feature) for CI.
        A folder runs every demo script in it, in name order, each in demo-out/<script>/, with one
        combined report.html + report.json in demo-out/ and one JUnit file (a test suite per script).
        A source: github:owner/repo@REF/path/script.md or …/folder/ (the repo fetched once at that
        pin into ~/.si/cache/repos, its layout kept; output in demo-out/<repo>/<script>/), or
        "https://…/script.md · sha256 HEX" (one file, its fingerprint checked)
        --trust   run code from elsewhere (a repo, a URL, gist:/github:/URL sources of start:,
                  prepare:, setup:, stop:, cleanup:) without asking, for this run (CI). Without it,
                  si shows each new pin (where from, size, first lines) and asks once (remembered
                  in ~/.si/trusted); with no terminal to ask in it refuses and says why
  si demo run <script|folder/|source> [--feature "Name"] [--voice engine:name] [--viewport V] [--junit report.xml] [-o dir]
              [--with captions,subtitles,…] [--fast-forward 3s | --no-fast-forward] [--region "<css>" | --crop x,y,w,h]
              [--transition fade|slide|wipe|pop|zoom|circle|cut [0.4s]] [--scale 1|2]
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
                  front matter region: override; default is the whole page. "<css> · zoom" opens
                  on the whole page and zooms into the element instead
        --transition fade 0.4s  the cut at every scene change (go, browser, terminal, window)
                  and between features: cut, fade (the default), slide[-left|-right|-up|-down],
                  wipe[-…], pop, zoom, circle; else transition: in front matter. A step's own
                  · slide wins. Camera moves (zoom to, pan to, zoom out) and spotlight "…" are
                  steps (RULES.md §27, Transitions and camera)
        --scale 2 the page is filmed at 2 device pixels per CSS pixel (the default) and every
                  video scaled back down to the viewport: crisp text, sharp zoom to / pan to.
                  --scale 1 films at the viewport's size (faster, less disk); else scale: in
                  front matter
  Both run start:, prepare: and step: in the order written, then ## Before; setup: before each
  feature; ## After, stop:, every program stopped, cleanup: (always: after a failure and on
  Ctrl-C too). ## Before and ## After are never filmed. terminal "Name" steps open a real shell,
  filmed as a framed window (RULES.md §27, Terminals and programs; Before and after).
`

// runDemo is `si demo test|run <script|folder|source>`. Its flags are its
// own: --feature takes a name here, unlike si report's --feature.
func runDemo(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si demo", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	var feature, junit, out, viewport, voice, region, crop, trans, scale string
	var help, captions, noCaptions, noFF, trust bool
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
	fl.StringVar(&trans, "transition", "", "")
	fl.StringVar(&scale, "scale", "", "")
	fl.DurationVar(&fastFwd, "fast-forward", fastFwd, "")
	fl.BoolVar(&noFF, "no-fast-forward", false, "")
	fl.BoolVar(&trust, "trust", false, "")
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
	args = pinned.JoinTarget(args)
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
		fmt.Fprintf(stderr, "si demo: test or run, and one script, folder or source\n\n%s", demoUsage)
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
	base := webdemo.Options{Feature: feature, Log: stdout, SI: version, Region: region, Crop: crop}
	if trans != "" {
		t, err := webdemo.ParseTransition(trans)
		if err != nil {
			fmt.Fprintf(stderr, "si demo: --transition: %v\n", err)
			return exitUsage
		}
		base.Transition = &t
	}
	if scale != "" {
		k, err := webdemo.ParseScale(scale)
		if err != nil {
			fmt.Fprintf(stderr, "si demo: --%v\n", err)
			return exitUsage
		}
		if !film {
			fmt.Fprintf(stderr, "si demo: --scale is for si demo run (si demo test films nothing)\n")
			return exitUsage
		}
		base.Scale = k
	}
	// the scripts: a file, a folder (every demo script in it), or the same
	// from a repo or a URL, fetched and trusted once per pin
	var client *pinned.Client
	fetcher := func() *pinned.Client {
		if client == nil {
			client = newFetcher(trust, stderr)
		}
		return client
	}
	local, repo, err := fetchTarget(args[1], fetcher, func(p string) []string { ps, _ := demoScripts(p); return ps })
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	paths, err := demoScripts(local)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	folder := len(paths) > 1 || isDir(local)
	var scripts []*webdemo.Script
	bad := 0
	for _, p := range paths {
		s, err := webdemo.ParseFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		for _, w := range s.Warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
		for _, p := range s.Problems {
			fmt.Fprintln(stderr, p)
		}
		bad += len(s.Problems)
		if err := s.SetViewport(viewport); err != nil {
			fmt.Fprintf(stderr, "si demo: --%v\n", err)
			return exitUsage
		}
		scripts = append(scripts, s)
	}
	if bad > 0 {
		return exitProblems
	}
	for _, s := range scripts { // every source fetched and trusted before anything runs
		if err := fetchSources(s, fetcher); err != nil {
			fmt.Fprintf(stderr, "si: %s: %v\n", filepath.Base(s.Path), err)
			return exitProblems
		}
	}
	if film {
		if base.FFmpeg = tools.Find("ffmpeg"); base.FFmpeg == "" {
			fmt.Fprintf(stderr, "si: %s\n", tools.Need("ffmpeg", "si demo run"))
			return exitProblems
		}
	}
	if base.Chrome = findChrome(); base.Chrome == "" {
		fmt.Fprintf(stderr, "si: %s\n", tools.Need("chrome", "si demo"))
		return exitProblems
	}
	// demo-out/<script>/; a folder: demo-out/<script>/ each and the combined
	// report in demo-out/; from a repo or a URL: demo-out/<repo>/<script>/
	root := out
	if root == "" {
		root = filepath.Join("demo-out", repo)
	}
	suite := &webdemo.Suite{Title: filepath.Base(strings.TrimSuffix(local, string(filepath.Separator))), SI: version, Date: time.Now().Format(time.RFC3339)}
	var last *webdemo.Report
	code := exitOK
	for _, s := range scripts {
		dir := root
		if folder || out == "" {
			dir = filepath.Join(root, s.Name)
		}
		if folder {
			fmt.Fprintf(stdout, "── %s\n", filepath.Base(s.Path))
		}
		rep, c := demoOne(s, base, dir, film, with, voice, fastFwd, noFF, stdout, stderr)
		if rep == nil {
			if !folder {
				return c
			}
			code = c
			rep = &webdemo.Report{Script: filepath.Base(s.Path), Title: firstNonEmpty(s.Title, s.Name), Failed: 1,
				Features: []webdemo.FeatureResult{{Name: s.Name, Status: "failed", Steps: []webdemo.StepResult{{Text: "run the script", Status: "failed", Found: "it could not start (see the output)"}}}}}
		}
		if !rep.OK() {
			code = exitProblems
		}
		rel, _ := filepath.Rel(root, dir)
		suite.Add(rep, rel)
		last = rep
	}
	if folder {
		if err := os.MkdirAll(root, 0o755); err == nil {
			err = suite.Write(root)
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si demo: %d script(s): %d passed, %d failed in %.1fs; report: %s\n", len(suite.Reports), suite.Passed, suite.Failed, suite.Seconds, filepath.Join(root, "report.html"))
	}
	if junit != "" {
		var b []byte
		var err error
		if folder {
			b, err = suite.JUnit()
		} else {
			b, err = last.JUnit()
		}
		if err == nil {
			err = os.WriteFile(junit, b, 0o644)
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	return code
}

// demoOne runs one script with its output in out; a nil report means it
// could not start (the code says why it ended).
func demoOne(s *webdemo.Script, base webdemo.Options, out string, film bool, with multi, voice string, fastFwd time.Duration, noFF bool, stdout, stderr io.Writer) (*webdemo.Report, int) {
	o := base
	if film {
		// --voice > front matter > si.conf beside the script > the user's si.conf > default
		cfg := config.Load("", filepath.Join(s.Dir(), "si.conf"))
		for _, w := range cfg.Warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
		v, err := chooseVoice(voice, s.Voice, cfg.Voice)
		if err != nil {
			fmt.Fprintf(stderr, "si demo: %v\n", err)
			return nil, exitUsage
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
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return nil, exitProblems
	}
	o.Out = out
	rep, err := webdemo.Run(s, o)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return nil, exitProblems
	}
	if !film { // a run writes its report itself, when asked for (--with report)
		if err := rep.Write(out); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return nil, exitProblems
		}
	}
	if film && o.With.Readme {
		if err := demoReadme(rep, s, out, "si demo "+strings.Join(os.Args[2:], " ")); err != nil {
			fmt.Fprintf(stderr, "si: README: %v\n", err)
			return nil, exitProblems
		}
	}
	fmt.Fprintf(stdout, "si demo: %d passed, %d failed", rep.Passed, rep.Failed)
	if rep.Skipped > 0 {
		fmt.Fprintf(stdout, ", %d skipped", rep.Skipped)
	}
	if !rep.OK() && rep.Failed == 0 {
		fmt.Fprint(stdout, " (a step around the features failed)")
	}
	fmt.Fprintf(stdout, " in %.1fs", rep.Seconds)
	switch {
	case !film:
		fmt.Fprintf(stdout, "; report: %s\n", filepath.Join(out, "report.html"))
	case len(rep.Files) == 0:
		fmt.Fprintln(stdout, "; nothing filmed")
	default:
		fmt.Fprintf(stdout, "; in %s: %s\n", out, strings.Join(rep.Files, ", "))
	}
	if !rep.OK() {
		return rep, exitProblems
	}
	return rep, exitOK
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
