// Command si turns markdown into animated, fluid HTML slide decks.
//
//	si play <dir|file.md> [--addr 127.0.0.1:7070] [--open] [--css path]...
//	si [build] <dir|file.md> --output <dir> [--css path]...
//	si check <dir|file.md>
//	si new <name>
//	si rules
//	si --export css-rules|rules|conf|example [--output path]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	docs "github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/examples"
	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

const version = "0.6.4"

// Exit codes.
const (
	exitOK       = 0
	exitProblems = 1 // deck problems or a runtime error
	exitUsage    = 2
)

const usage = `si: markdown in, animated fluid HTML slide decks out

Usage:
  si play <dir|file.md> [--addr 127.0.0.1:7070] [--open] [--css <dir|file.css>]...
        serve the decks, rendered from markdown on every request, live reload on change
  si [build] <dir|file.md> [--output html/] [--css <dir|file.css>]... [--no-inline-media] [--pdf]
        write one self-contained HTML file per deck plus index.html
        --pdf also renders each deck to pdfs/<name>.pdf and adds a PDF badge to the index tiles
  si check <dir|file.md>
        validate decks: title, directives, assets, includes; exit 1 on a problem
  si new <name>
        scaffold <name>/ with a deck that uses every rule
  si new-canvas <name>
        scaffold <name>.svg (a working animated canvas) + <name>.md to preview it
  si migrate <dir|file> [--output dir] [--ai "<command>"]
        print an editorial-migration prompt for an existing deck (pipe it to an
        AI agent, or run it with --ai "claude -p" / --ai "gemini", etc.)
  si manual [--slides] [--pdf] [--output dir]
        render the built-in user manual as self-contained HTML (and PDF with --pdf)
  si selfupdate
        download the latest release binary for this OS/arch (checksum-verified) and
        replace the running si; needs write access to si's install directory
  si rules
        print the markdown rules (RULES.md)
  si --export css-rules|rules|conf|example|themes [--output <path>]
        css-rules: starter stylesheet with every --si-* token and .si-* hook
        rules:     RULES.md
        conf:      a starter si.conf with the default keys and features
        example:   the showcase decks (default output: ./si-example)
        themes:    one sample deck per built-in theme (default: ./si-themes)

Flags (GNU style, before or after the path):
  -o, --output <path>   output directory (build) or file (export)
      --css <path>      user stylesheet file or directory; repeatable; loads after the theme
      --theme <name>    built-in theme: studio, paper, mono, crdb (aliases cockroachdb, cockroachlabs), or a path to a .css theme file; overrides theme: front matter
      --conf <path>     read si.conf (key remaps, feature toggles, default theme); default: si.conf next to the deck
      --title <text>    build: title for the generated index.html — its <title> and <h1> (default: the source dir name)
      --site <text>     alias for --title (reads as "the site/landing page title")
      --ai <command>    migrate: pipe the migration prompt to this agent command (e.g. "claude -p")
      --addr <host:port> play address (default 127.0.0.1:7070)
      --open            play: open the browser
      --dev             play: hot reload — morph changed slides in place (keeps the current slide, step and overview)
      --no-inline-media build: copy images and videos next to the HTML instead of data URIs
      --pdf             build: also render each deck to pdfs/<name>.pdf and link it from the index tiles (needs headless Chrome)
      --export <what>   css-rules, rules, conf, example or themes
  -h, --help            this help
      --version         print the version

Example:
  si --css css/ --output html/ md/
`

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

type options struct {
	output, addr, export, theme, conf, title, ai string
	open, help, ver                              bool
	noInline, pdf, slides, dev                   bool
	css                                          multi
	args                                         []string
}

// parseArgs accepts GNU-style long flags anywhere on the command line:
// `si --css css/ --output html/ md/` and `si md/ -o html/` both work.
func parseArgs(argv []string, stderr io.Writer) (*options, error) {
	o := &options{}
	fl := flag.NewFlagSet("si", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	fl.StringVar(&o.output, "output", "", "")
	fl.StringVar(&o.output, "out", "", "")
	fl.StringVar(&o.output, "o", "", "")
	fl.StringVar(&o.addr, "addr", "127.0.0.1:7070", "")
	fl.StringVar(&o.export, "export", "", "")
	fl.BoolVar(&o.open, "open", false, "")
	fl.BoolVar(&o.dev, "dev", false, "")
	fl.BoolVar(&o.noInline, "no-inline-media", false, "")
	fl.BoolVar(&o.pdf, "pdf", false, "")
	fl.BoolVar(&o.help, "help", false, "")
	fl.BoolVar(&o.help, "h", false, "")
	fl.BoolVar(&o.ver, "version", false, "")
	fl.Var(&o.css, "css", "")
	fl.StringVar(&o.theme, "theme", "", "")
	fl.StringVar(&o.conf, "conf", "", "")
	fl.StringVar(&o.title, "title", "", "")
	fl.StringVar(&o.title, "site", "", "") // alias: the index page/site title (<title> + <h1>)
	fl.StringVar(&o.ai, "ai", "", "")
	fl.BoolVar(&o.slides, "slides", false, "")
	for {
		if err := fl.Parse(argv); err != nil {
			return nil, err
		}
		argv = fl.Args()
		if len(argv) == 0 {
			break
		}
		if argv[0] == "--" {
			o.args = append(o.args, argv[1:]...)
			break
		}
		o.args = append(o.args, argv[0])
		argv = argv[1:]
	}
	return o, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	o, err := parseArgs(argv, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n\n%s", err, usage)
		return exitUsage
	}
	if o.ver {
		fmt.Fprintln(stdout, "si", version)
		return exitOK
	}
	if o.help || (len(o.args) == 0 && o.export == "") {
		fmt.Fprint(stdout, usage)
		if o.help {
			return exitOK
		}
		return exitUsage
	}
	if o.export != "" {
		return export(o, stdout, stderr)
	}
	cmd, rest := o.args[0], o.args[1:]
	switch cmd {
	case "help":
		fmt.Fprint(stdout, usage)
		return exitOK
	case "version":
		fmt.Fprintln(stdout, "si", version)
		return exitOK
	case "rules":
		fmt.Fprint(stdout, docs.Rules)
		return exitOK
	case "play":
		if len(rest) != 1 {
			return usageErr(stderr, "play needs one <dir|file.md>")
		}
		return play(rest[0], o, stdout, stderr)
	case "check":
		if len(rest) != 1 {
			return usageErr(stderr, "check needs one <dir|file.md>")
		}
		return check(rest[0], stdout, stderr)
	case "new":
		if len(rest) != 1 {
			return usageErr(stderr, "new needs a <name>")
		}
		return scaffold(rest[0], stdout, stderr)
	case "new-canvas":
		if len(rest) != 1 {
			return usageErr(stderr, "new-canvas needs a <name>")
		}
		return scaffoldCanvas(rest[0], stdout, stderr)
	case "migrate":
		if len(rest) != 1 {
			return usageErr(stderr, "migrate needs one <dir|file> to migrate")
		}
		return runMigrate(rest[0], o, stdout, stderr)
	case "manual":
		return runManual(o, stdout, stderr)
	case "selfupdate":
		return selfUpdate(stdout, stderr)
	case "build":
		if len(rest) != 1 {
			return usageErr(stderr, "build needs one <dir|file.md>")
		}
		return build(rest[0], o, stdout, stderr)
	}
	if len(o.args) != 1 {
		return usageErr(stderr, fmt.Sprintf("unknown command %q", cmd))
	}
	if _, err := os.Stat(cmd); err != nil {
		return usageErr(stderr, fmt.Sprintf("unknown command or missing path %q", cmd))
	}
	return build(cmd, o, stdout, stderr)
}

func usageErr(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "si: %s\n\n%s", msg, usage)
	return exitUsage
}

func build(src string, o *options, stdout, stderr io.Writer) int {
	files, err := deck.List(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if len(files) == 0 {
		fmt.Fprintf(stderr, "si: no decks (*.md) in %s\n", src)
		return exitProblems
	}
	out := o.output
	if out == "" {
		out = "html"
	}
	css, err := render.LoadCSS(o.css)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitUsage
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	dir := src
	if fi, err := os.Stat(src); err == nil && !fi.IsDir() {
		dir = filepath.Dir(src)
	}
	cfg := config.Load(o.conf, filepath.Join(dir, "si.conf"))
	for _, w := range cfg.Warnings {
		fmt.Fprintf(stderr, "si.conf: %s\n", w)
	}
	failed := false
	chrome := ""
	if o.pdf {
		if chrome = findChrome(); chrome == "" {
			fmt.Fprintln(stderr, "si: --pdf needs Google Chrome/Chromium; none found — writing HTML only")
		}
	}
	var entries []render.IndexEntry
	indexTheme := o.theme // the index page follows --theme; else the first deck's theme
	for _, f := range files {
		d, err := deck.Parse(f)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			failed = true
			continue
		}
		if indexTheme == "" && d.Meta.Theme != "" {
			indexTheme = d.Meta.Theme
		}
		mode := render.MediaInline
		if o.noInline {
			mode = render.MediaCopy
		}
		res := render.Deck(d, render.Options{UserCSS: css, Media: mode, Theme: o.theme, Config: cfg})
		for _, p := range res.Problems {
			fmt.Fprintf(stderr, "  error: %s\n", p)
			failed = true
		}
		for _, w := range res.Warnings {
			fmt.Fprintf(stderr, "  warning: %s\n", w)
		}
		for _, m := range res.Media {
			if err := copyFile(m.Src, filepath.Join(out, filepath.FromSlash(m.Rel))); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				failed = true
			}
		}
		dest := filepath.Join(out, d.Name+".html")
		if err := os.WriteFile(dest, res.HTML, 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			failed = true
			continue
		}
		entry := render.Entry(d)
		fmt.Fprintf(stdout, "  %-28s %3d slides  %5s min  %3d steps  %2d canvases  %4d KB\n",
			d.Name+".html", len(d.Slides), strconv.FormatFloat(d.Minutes(), 'f', -1, 64), res.Steps, res.Canvases, len(res.HTML)/1024)
		if chrome != "" {
			rel := filepath.Join("pdfs", d.Name+".pdf") // a pdfs/ folder beside the decks; linked from the index tiles
			pdf := filepath.Join(out, rel)
			if err := os.MkdirAll(filepath.Dir(pdf), 0o755); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				failed = true
			} else if err := deckToPDF(chrome, dest, pdf); err != nil {
				fmt.Fprintf(stderr, "si: pdf %s: %v\n", d.Name, err)
				failed = true
			} else {
				entry.PDF = filepath.ToSlash(rel)
				fmt.Fprintf(stdout, "  %-28s (pdf)\n", rel)
			}
		}
		entries = append(entries, entry)
	}
	title := filepath.Base(filepath.Clean(src))
	if st, err := os.Stat(src); err == nil && !st.IsDir() {
		title = filepath.Base(filepath.Dir(src))
	}
	if o.title != "" {
		title = o.title // --title / --site overrides the index title + heading
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), render.Index(title, entries, render.Options{UserCSS: css, Theme: indexTheme}), 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: %d deck(s) from %s -> %s\n", len(entries), src, out)
	if failed {
		fmt.Fprintln(stderr, "si: FAIL")
		return exitProblems
	}
	return exitOK
}

func check(src string, stdout, stderr io.Writer) int {
	files, err := deck.List(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	n := 0
	for _, f := range files {
		d, err := deck.Parse(f)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			n++
			continue
		}
		res := render.Deck(d, render.Options{})
		for _, p := range res.Problems {
			fmt.Fprintf(stderr, "%s\n", p)
			n++
		}
		for _, w := range res.Warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
	}
	if n > 0 {
		fmt.Fprintf(stderr, "si check: %d problem(s) in %d deck(s)\n", n, len(files))
		return exitProblems
	}
	fmt.Fprintf(stdout, "si check: %d deck(s) OK\n", len(files))
	return exitOK
}

func play(src string, o *options, stdout, stderr io.Writer) int {
	if _, err := render.LoadCSS(o.css); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitUsage
	}
	srv, err := serve.New(src, o.css, o.theme, o.conf)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	srv.Dev = o.dev
	ln, err := net.Listen("tcp", o.addr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	url := "http://" + ln.Addr().String() + "/"
	fmt.Fprintf(stdout, "si play: serving %s at %s (live reload on; Ctrl+C stops)\n", src, url)
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		if ta.IP.IsLoopback() {
			srv.Terminal = true // play terminal: loopback only, never on a reachable address
			fmt.Fprintf(stdout, "  terminal drawer: press ` (backtick) — runs local commands, loopback only\n")
			fmt.Fprintf(stdout, "  to open on a phone on your network: si play %s --addr 0.0.0.0:%d\n", src, ta.Port)
		} else if ip := lanIP(); ip != "" {
			lan := fmt.Sprintf("http://%s:%d/", ip, ta.Port)
			fmt.Fprintf(stdout, "  on your network: %s\n", lan)
			if qr, err := qrString(lan); err == nil {
				fmt.Fprint(stdout, qr)
			}
		}
	}
	stop := make(chan struct{})
	go srv.Watch(stop)
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go func() {
		<-ctx.Done()
		close(stop)
		srv.CloseTerminals()
		sctx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		hs.Shutdown(sctx)
	}()
	if o.open {
		openBrowser(url)
	}
	if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	return exitOK
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	_ = c.Start()
}

// lanIP returns the host's first private non-loopback IPv4, for opening the deck
// from another device on the same network. Empty if none is found.
func lanIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok {
			if ip := ipn.IP.To4(); ip != nil && !ip.IsLoopback() && ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return ""
}

// findChrome returns a path to a headless-capable Chromium browser, or "".
func findChrome() string {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	case "windows":
		candidates = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		}
	default:
		candidates = []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"}
	}
	for _, c := range candidates {
		if filepath.IsAbs(c) {
			if _, err := os.Stat(c); err == nil {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

// deckToPDF renders a built deck HTML to a self-contained PDF with headless
// Chrome. The deck is loaded at #print so deck.js shows every slide in its
// final state (all steps revealed, canvases at their last state).
func deckToPDF(chrome, htmlPath, pdfPath string) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	url := "file://" + filepath.ToSlash(abs) + "#print"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome,
		"--headless=new", "--disable-gpu", "--no-pdf-header-footer",
		"--virtual-time-budget=2000",
		"--print-to-pdf="+pdfPath, url)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	if fi, err := os.Stat(pdfPath); err != nil || fi.Size() == 0 {
		return fmt.Errorf("no PDF produced")
	}
	return nil
}

func export(o *options, stdout, stderr io.Writer) int {
	write := func(text string) int {
		if o.output == "" {
			fmt.Fprint(stdout, text)
			return exitOK
		}
		if err := os.WriteFile(o.output, []byte(text), 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: wrote %s\n", o.output)
		return exitOK
	}
	switch o.export {
	case "css-rules":
		return write(assets.StarterCSS())
	case "rules":
		return write(docs.Rules)
	case "conf":
		return write(config.Default)
	case "example":
		out := o.output
		if out == "" {
			out = "si-example"
		}
		if err := writeExamples(out, ""); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: wrote the showcase decks to %s; try: si play %s\n", out, out)
		return exitOK
	case "themes":
		out := o.output
		if out == "" {
			out = "si-themes"
		}
		if err := writeThemeGallery(out); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: wrote %d theme sample decks to %s; try: si play %s\n", len(assets.ThemeNames()), out, out)
		return exitOK
	}
	return usageErr(stderr, fmt.Sprintf("unknown --export %q (css-rules, rules, conf, example, themes)", o.export))
}

func scaffold(name string, stdout, stderr io.Writer) int {
	base := filepath.Base(name)
	if err := writeExamples(name, base+".md"); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: created %s/%s.md (every rule in use)\n  si play %s --open\n", name, base, name)
	return exitOK
}

// writeExamples copies the embedded examples into dir; mainName renames the
// showcase deck. dir must not exist or be empty.
func writeExamples(dir, mainName string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s exists and is not empty", dir)
	}
	return fs.WalkDir(examples.FS, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, ".go") {
			return nil
		}
		dest := filepath.Join(dir, filepath.FromSlash(p))
		if p == examples.Main && mainName != "" {
			dest = filepath.Join(dir, mainName)
		}
		if e.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		raw, err := examples.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, raw, 0o644)
	})
}

// writeThemeGallery writes one sample deck per built-in theme into dir, so that
// `si play dir` (or `si build dir`) shows a gallery: the same content rendered in
// every theme, linked from index.html. dir must not exist or be empty.
func writeThemeGallery(dir string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s exists and is not empty", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range assets.ThemeNames() {
		md := strings.ReplaceAll(themeSample, "THEME", name)
		if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(md), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// themeSample is one deck's worth of content exercising a theme's tokens and
// looks (headings, body, accent, code, a flow). THEME is the theme name.
const themeSample = "---\ntitle: THEME theme\ntheme: THEME\n---\n\n" + `# The THEME theme

The same sample deck, rendered in every built-in theme.

---

## Headings, text, and accent

- A bullet, and a [link](https://example.com)
- **Bold**, *italic*, and ` + "`inline code`" + `
- The accent colours bullets, links and the rule under headings

Notes: Press d to flip light/dark and see both palettes.

---

## A flow, themed

` + "```stage\ndir: LR\nDraft --> Review --> Ship\ndot Draft..Ship 2s\n```" + `

- Boxes, arrows and the moving dot pick up the theme tokens

---

## Code

` + "```go\nfunc main() {\n\tfmt.Println(\"the THEME theme\")\n}\n```" + `

---

<!-- layout: big-number -->

**THEME**

one of the built-in themes.
`

func copyFile(src, dest string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, raw, 0o644)
}
