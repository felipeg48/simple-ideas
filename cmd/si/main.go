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

	simpleideas "github.com/felipeg48/simple-ideas"
	"github.com/felipeg48/simple-ideas/examples"
	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

const version = "0.1.0"

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
  si [build] <dir|file.md> [--output html/] [--css <dir|file.css>]... [--no-inline-media]
        write one self-contained HTML file per deck plus index.html
  si check <dir|file.md>
        validate decks: title, directives, assets, includes; exit 1 on a problem
  si new <name>
        scaffold <name>/ with a deck that uses every rule
  si new-canvas <name>
        scaffold <name>.svg (a working animated canvas) + <name>.md to preview it
  si rules
        print the markdown rules (RULES.md)
  si --export css-rules|rules|conf|example [--output <path>]
        css-rules: starter stylesheet with every --si-* token and .si-* hook
        rules:     RULES.md
        conf:      a starter si.conf with the default keys and features
        example:   the showcase decks (default output: ./si-example)

Flags (GNU style, before or after the path):
  -o, --output <path>   output directory (build) or file (export)
      --css <path>      user stylesheet file or directory; repeatable; loads after the theme
      --theme <name>    built-in theme: studio, paper, mono, crdb (aliases cockroachdb, cockroachlabs); overrides theme: front matter
      --conf <path>     read si.conf (key remaps, feature toggles, default theme); default: si.conf next to the deck
      --addr <host:port> play address (default 127.0.0.1:7070)
      --open            play: open the browser
      --no-inline-media build: copy images and videos next to the HTML instead of data URIs
      --export <what>   css-rules, rules, conf or example
  -h, --help            this help
      --version         print the version

Example:
  si --css css/ --output html/ md/
`

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

type options struct {
	output, addr, export, theme, conf string
	open, help, ver                   bool
	noInline                          bool
	css                               multi
	args                              []string
}

// parseArgs accepts GNU-style long flags anywhere on the command line:
// `si --css css/ --output html/ md/` and `si md/ -o html/` both work.
func parseArgs(argv []string, stderr io.Writer) (*options, error) {
	o := &options{}
	fl := flag.NewFlagSet("si", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	fl.StringVar(&o.output, "output", "", "")
	fl.StringVar(&o.output, "o", "", "")
	fl.StringVar(&o.addr, "addr", "127.0.0.1:7070", "")
	fl.StringVar(&o.export, "export", "", "")
	fl.BoolVar(&o.open, "open", false, "")
	fl.BoolVar(&o.noInline, "no-inline-media", false, "")
	fl.BoolVar(&o.help, "help", false, "")
	fl.BoolVar(&o.help, "h", false, "")
	fl.BoolVar(&o.ver, "version", false, "")
	fl.Var(&o.css, "css", "")
	fl.StringVar(&o.theme, "theme", "", "")
	fl.StringVar(&o.conf, "conf", "", "")
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
		fmt.Fprint(stdout, simpleideas.Rules)
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
	var entries []render.IndexEntry
	for _, f := range files {
		d, err := deck.Parse(f)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			failed = true
			continue
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
		entries = append(entries, render.Entry(d))
		fmt.Fprintf(stdout, "  %-28s %3d slides  %5s min  %3d steps  %2d canvases  %4d KB\n",
			d.Name+".html", len(d.Slides), strconv.FormatFloat(d.Minutes(), 'f', -1, 64), res.Steps, res.Canvases, len(res.HTML)/1024)
	}
	title := filepath.Base(filepath.Clean(src))
	if st, err := os.Stat(src); err == nil && !st.IsDir() {
		title = filepath.Base(filepath.Dir(src))
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), render.Index(title, entries, render.Options{UserCSS: css}), 0o644); err != nil {
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
	ln, err := net.Listen("tcp", o.addr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	url := "http://" + ln.Addr().String() + "/"
	fmt.Fprintf(stdout, "si play: serving %s at %s (live reload on; Ctrl+C stops)\n", src, url)
	stop := make(chan struct{})
	go srv.Watch(stop)
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go func() {
		<-ctx.Done()
		close(stop)
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
		return write(simpleideas.Rules)
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
	}
	return usageErr(stderr, fmt.Sprintf("unknown --export %q (css-rules, rules, conf, example)", o.export))
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
