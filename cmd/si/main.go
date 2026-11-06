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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"html"
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
	"text/tabwriter"
	"time"

	docs "github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/examples"
	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/serve"
	"github.com/felipeg48/simple-ideas/internal/studio"
)

const version = "1.5.0"

// Exit codes.
const (
	exitOK       = 0
	exitProblems = 1 // deck problems or a runtime error
	exitUsage    = 2
)

const usage = `si: markdown in, animated fluid HTML slide decks out

Usage:
  si play <dir|file.md> [--addr 127.0.0.1:7070] [--open] [--css <dir|file.css>]... [--remote[=tp,draw]]
        serve the decks, rendered from markdown on every request, live reload on change
  si serve <dir|file.md> [--addr 127.0.0.1:7070] [--open] [--css <dir|file.css>]...
        serve everything for readers, live: decks, articles, guides (with their
        navigation) and labs; like play, without the terminal drawer
  si --web [dir] [--addr 127.0.0.1:7070] [--no-open]   the Studio: edit, preview and check decks in the browser (loopback only)
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
  si theme list
        list the built-in themes (name, display name, aliases) for --theme / theme:
  si --export css-rules|rules|conf|example|themes|agent-rules [--output <path>]
        css-rules: starter stylesheet with every --si-* token and .si-* hook
        rules:     RULES.md
        conf:      a starter si.conf with the default keys and features
        example:   an example to copy from (default output: ./si-example); --type picks it:
                   deck (default; also slides), article, guide (also tutorial), lab, book (also reference), demo, site, all
        themes:    one sample deck per built-in theme (default: ./si-themes)
        agent-rules: CLAUDE.md + AGENTS.md telling a coding agent how to write si decks
                   (default: the current dir; -o - prints it; never overwrites)

Flags (GNU style, before or after the path):
  -o, --output <path>   output directory (build) or file (export)
      --css <path>      user stylesheet file or directory; repeatable; loads after the theme
      --theme <name>    built-in theme (si theme list: studio, paper, mono, crdb, acemq, acemq-pro; "AceMQ Pro" works too), a .css theme file or an https:// URL; overrides theme: front matter
      --conf <path>     read si.conf (key remaps, feature toggles, default theme); default: si.conf next to the deck
      --title <text>    build, play: title for the index page — its <title> and <h1> (default: the source dir name)
      --site-title <text> alias for --title: the site or landing page title (--site <text> still works, with a warning)
      --ai <command>    migrate: pipe the migration prompt to this agent command (e.g. "claude -p")
      --addr <host:port> play address (default 127.0.0.1:7070)
      --remote[=tp,draw] play: a second, keyed listener for a tablet or phone on your network: tp = teleprompter, draw = draw pad (default: both)
      --remote-addr <host:port> play: the --remote address (default 0.0.0.0, the play port + 1)
      --offline         code: / diff: use the cache only (CI, a plane); fails on what was never fetched
      --no-instructor   build: leave the instructor demo scripts (type: demo) out of a participant build
      --open            play: open the browser
      --dev             play: hot reload — morph changed slides in place (keeps the current slide, step and overview)
      --no-inline-media build: copy images and videos next to the HTML instead of data URIs
      --pdf             build: also render each deck to pdfs/<name>.pdf and link it from the index tiles (needs headless Chrome)
      --export <what>   css-rules, rules, conf, example, themes or agent-rules
  -h, --help            this help
      --version         print the version

Example:
  si --css css/ --output html/ md/
`

// remoteFlag is --remote (every role) or --remote=tp,draw (only these).
type remoteFlag struct{ roles []string }

func (r *remoteFlag) IsBoolFlag() bool { return true }
func (r *remoteFlag) String() string   { return strings.Join(r.roles, ",") }
func (r *remoteFlag) Set(v string) error {
	if v == "true" {
		r.roles = append([]string(nil), serve.RemoteRoles...)
		return nil
	}
	r.roles = nil
	if v == "false" {
		return nil
	}
	for _, w := range strings.Split(v, ",") {
		w = strings.TrimSpace(w)
		if _, ok := serve.RemoteRoleNames[w]; !ok {
			return fmt.Errorf("unknown --remote role %q (use %s)", w, strings.Join(serve.RemoteRoles, ", "))
		}
		r.roles = append(r.roles, w)
	}
	return nil
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

type options struct {
	output, addr, export, theme, conf, title, ai string
	kind                                         string // --type of --export example
	remoteAddr                                   string
	siteOld                                      string // the old --site <text>, now --site-title
	remote                                       remoteFlag
	open, help, ver                              bool
	noInline, pdf, slides, dev, web, noOpen      bool
	serveMode                                    bool // si serve: play without the terminal (everything, for readers)
	offline                                      bool // code: / diff: from the cache only
	noInstructor                                 bool // si build: leave type: demo scripts out (a participant build)
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
	fl.Var(&o.remote, "remote", "")
	fl.StringVar(&o.remoteAddr, "remote-addr", "", "")
	fl.StringVar(&o.export, "export", "", "")
	fl.StringVar(&o.kind, "type", "", "")
	fl.BoolVar(&o.offline, "offline", false, "")
	fl.BoolVar(&o.noInstructor, "no-instructor", false, "")
	fl.BoolVar(&o.open, "open", false, "")
	fl.BoolVar(&o.dev, "dev", false, "")
	fl.BoolVar(&o.web, "web", false, "")
	fl.BoolVar(&o.noOpen, "no-open", false, "")
	fl.BoolVar(&o.noInline, "no-inline-media", false, "")
	fl.BoolVar(&o.pdf, "pdf", false, "")
	fl.BoolVar(&o.help, "help", false, "")
	fl.BoolVar(&o.help, "h", false, "")
	fl.BoolVar(&o.ver, "version", false, "")
	fl.Var(&o.css, "css", "")
	fl.StringVar(&o.theme, "theme", "", "")
	fl.StringVar(&o.conf, "conf", "", "")
	fl.StringVar(&o.title, "title", "", "")
	fl.StringVar(&o.title, "site-title", "", "")
	fl.StringVar(&o.siteOld, "site", "", "") // deprecated alias of --site-title; --site is kept for site features
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
	if o.siteOld != "" {
		fmt.Fprintln(stderr, "si: warning: --site <text> is now --site-title <text>")
		if o.title == "" {
			o.title = o.siteOld
		}
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
	deck.Offline = o.offline
	if o.theme != "" {
		lower := strings.ToLower(o.theme)
		if !strings.HasSuffix(lower, ".css") && !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			name := assets.CanonicalTheme(o.theme)
			if _, ok := assets.Themes[name]; !ok {
				return usageErr(stderr, fmt.Sprintf("unknown theme %q (see: si theme list)", o.theme))
			}
			o.theme = name
		}
	}
	if o.web && !o.help {
		if len(o.args) > 1 {
			return usageErr(stderr, "--web takes at most one [dir]")
		}
		dir := "."
		if len(o.args) == 1 {
			dir = o.args[0]
		}
		return web(dir, o, stdout, stderr)
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
	case "theme", "themes":
		if len(rest) > 1 || (len(rest) == 1 && rest[0] != "list") {
			return usageErr(stderr, "theme takes one subcommand: list")
		}
		listThemes(stdout)
		return exitOK
	case "play":
		if len(rest) != 1 {
			if len(o.remote.roles) > 0 {
				return usageErr(stderr, "play needs one <dir|file.md> (pick remote roles with --remote=tp,draw, no space)")
			}
			return usageErr(stderr, "play needs one <dir|file.md>")
		}
		return play(rest[0], o, stdout, stderr)
	case "serve":
		if len(rest) != 1 {
			return usageErr(stderr, "serve needs one <dir|file.md>")
		}
		if len(o.remote.roles) > 0 {
			return usageErr(stderr, "--remote is for si play (the presenter); si serve has no teleprompter remote")
		}
		o.serveMode = true
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
	if g, ok := guideAt(src); ok {
		out := o.output
		if out == "" {
			out = "html"
		}
		css, err := render.LoadCSS(o.css)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitUsage
		}
		cfg := config.Load(o.conf, filepath.Join(src, "si.conf"))
		if _, ok := buildGuide(g, out, o, css, cfg, stdout, stderr); !ok {
			fmt.Fprintln(stderr, "si: FAIL")
			return exitProblems
		}
		return exitOK
	}
	files, err := deck.List(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	guides := subGuides(src)
	if len(files) == 0 && len(guides) == 0 {
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
	decks := make([]*deck.Deck, len(files))
	var labs []int // labs in name order, linked prev/next
	for i, f := range files {
		d, err := deck.Parse(f)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			failed = true
			continue
		}
		if o.noInstructor && d.IsDemo() {
			fmt.Fprintf(stdout, "  %-28s skipped (instructor demo; --no-instructor)\n", d.Name+".html")
			continue
		}
		decks[i] = d
		if d.IsLab() {
			labs = append(labs, i)
		}
	}
	// find (/) offers the other pages of this build, plus .html already in out
	pages := render.Pages(decks, true, chrome != "", out, cfg)
	for i, d := range decks {
		if d == nil {
			continue
		}
		if indexTheme == "" && d.Meta.Theme != "" {
			indexTheme = d.Meta.Theme
		}
		mode := render.MediaInline
		if o.noInline {
			mode = render.MediaCopy
		}
		ro := render.Options{UserCSS: css, Media: mode, Theme: o.theme, Config: cfg, Pages: pages}
		for k, j := range labs {
			if j == i {
				if k > 0 {
					ro.Prev = render.DocLink{Href: decks[labs[k-1]].Name + ".html", Title: decks[labs[k-1]].Title()}
				}
				if k < len(labs)-1 {
					ro.Next = render.DocLink{Href: decks[labs[k+1]].Name + ".html", Title: decks[labs[k+1]].Title()}
				}
			}
		}
		res := render.Deck(d, ro)
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
		if d.IsDoc() {
			fmt.Fprintf(stdout, "  %-28s %s  %d sections  %4d KB\n", d.Name+".html", d.Meta.Type, entry.Sections, len(res.HTML)/1024)
		} else {
			fmt.Fprintf(stdout, "  %-28s %3d slides  %5s min  %3d steps  %2d canvases  %4d KB\n",
				d.Name+".html", len(d.Slides), strconv.FormatFloat(d.Minutes(), 'f', -1, 64), res.Steps, res.Canvases, len(res.HTML)/1024)
		}
		if chrome != "" && !d.IsDoc() { // the slides PDF does not apply to a document (print it from the browser)
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
	for _, g := range guides { // a guide folder becomes one tile, built into out/<folder>/
		sub := filepath.Base(g.Dir)
		home, ok := buildGuide(g, filepath.Join(out, sub), o, css, cfg, stdout, stderr)
		if !ok {
			failed = true
		}
		if home == "" {
			continue
		}
		if indexTheme == "" && g.Meta.Theme != "" {
			indexTheme = g.Meta.Theme
		}
		entries = append(entries, render.IndexEntry{Href: sub + "/" + home, Title: g.Meta.Title, Subtitle: g.Meta.Description,
			Lab: true, Kind: guideKindOf(g), Sections: len(g.Pages), Version: firstOf(g.Meta.Edition, g.Meta.Version), Status: g.Meta.Status})
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
	if g, ok := guideAt(src); ok {
		return checkGuides([]*deck.Guide{g}, 0, stdout, stderr)
	}
	files, err := deck.List(src)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	guides := subGuides(src)
	if len(guides) > 0 && len(files) == 0 {
		return checkGuides(guides, 0, stdout, stderr)
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
	if len(guides) > 0 {
		fmt.Fprintf(stdout, "si check: %d deck(s), %d problem(s)\n", len(files), n)
		return checkGuides(guides, n, stdout, stderr)
	}
	if n > 0 {
		fmt.Fprintf(stderr, "si check: %d problem(s) in %d deck(s)\n", n, len(files))
		return exitProblems
	}
	fmt.Fprintf(stdout, "si check: %d deck(s) OK\n", len(files))
	return exitOK
}

// guideAt reports whether path is a guide folder (it holds a _nav.md).
func guideAt(path string) (*deck.Guide, bool) {
	if !deck.IsGuideDir(path) {
		return nil, false
	}
	g, err := deck.LoadGuide(path)
	return g, err == nil
}

// subGuides are the guides in the immediate subfolders of dir.
func subGuides(dir string) []*deck.Guide {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []*deck.Guide
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(e.Name(), "_") {
			if g, ok := guideAt(filepath.Join(dir, e.Name())); ok {
				out = append(out, g)
			}
		}
	}
	return out
}

// checkGuides renders every page of each guide and reports problems
// (including broken links and anchors between pages) and warnings. prior is
// the number of problems already found (by the decks of the same check).
func checkGuides(guides []*deck.Guide, prior int, stdout, stderr io.Writer) int {
	n, pages := prior, 0
	for i := 0; i < len(guides); i++ { // a site's menu may link sub-guides and books: check those too
		guides = append(guides, subGuidesOf(guides[i].Items)...)
	}
	for _, g := range guides {
		ps, probs, warns := render.RenderGuide(g, render.Options{})
		pages += len(ps)
		for _, p := range probs {
			fmt.Fprintf(stderr, "%s\n", p)
		}
		for _, w := range warns {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
		n += len(probs)
	}
	if n > 0 {
		fmt.Fprintf(stderr, "si check: %d problem(s); %d guide(s), %d page(s)\n", n, len(guides), pages)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si check: %d guide(s), %d page(s) OK\n", len(guides), pages)
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
	srv.Title = o.title
	srv.Remote = len(o.remote.roles) > 0
	ln, err := net.Listen("tcp", o.addr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	var remote *http.Server
	if srv.Remote {
		rs, code := startRemote(srv, ln.Addr(), o, stdout, stderr)
		if rs == nil {
			ln.Close()
			return code
		}
		remote = rs
	}
	url := "http://" + ln.Addr().String() + "/"
	cmdName := "si play"
	if o.serveMode {
		cmdName = "si serve"
	}
	fmt.Fprintf(stdout, "%s: serving %s at %s (live reload on; Ctrl+C stops)\n", cmdName, src, url)
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		if o.serveMode {
			// readers: decks, articles, guides and labs, no terminal drawer
		} else if ta.IP.IsLoopback() {
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
		if remote != nil {
			remote.Shutdown(sctx)
		}
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

// startRemote opens the --remote listener (default: every interface, the play
// port + 1), prints one URL + QR code per role and serves it in the
// background. It returns nil and an exit code on failure.
func startRemote(srv *serve.Server, main net.Addr, o *options, stdout, stderr io.Writer) (*http.Server, int) {
	addr := o.remoteAddr
	if addr == "" {
		port := 0
		if ta, ok := main.(*net.TCPAddr); ok && ta.Port > 0 {
			port = ta.Port + 1
		}
		addr = fmt.Sprintf("0.0.0.0:%d", port)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(stderr, "si: --remote: %v (pick another with --remote-addr)\n", err)
		return nil, exitProblems
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		fmt.Fprintf(stderr, "si: --remote: %v\n", err)
		return nil, exitProblems
	}
	key := hex.EncodeToString(raw[:])
	host := lanIP()
	ta, _ := ln.Addr().(*net.TCPAddr)
	if ta != nil && !ta.IP.IsUnspecified() {
		host = ta.IP.String()
	}
	if host == "" {
		host = "127.0.0.1"
	}
	base := fmt.Sprintf("http://%s/", net.JoinHostPort(host, strconv.Itoa(ta.Port)))
	fmt.Fprintf(stdout, "  remote (same network, keyed link; plain HTTP, so use a trusted network):\n")
	decks := srv.RemoteDecks()
	if len(decks) == 1 {
		for _, r := range o.remote.roles {
			u := fmt.Sprintf("%s%s?role=%s&k=%s", base, strings.TrimPrefix(decks[0], "/"), r, key)
			fmt.Fprintf(stdout, "  %s: %s\n", serve.RemoteRoleNames[r], u)
			if qr, err := qrString(u); err == nil {
				fmt.Fprint(stdout, qr)
			}
		}
	} else {
		u := base + "?k=" + key
		fmt.Fprintf(stdout, "  pick a deck and a role (%s): %s\n", strings.Join(o.remote.roles, ", "), u)
		if qr, err := qrString(u); err == nil {
			fmt.Fprint(stdout, qr)
		}
	}
	rs := &http.Server{Handler: srv.RemoteHandler(key, o.remote.roles), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := rs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "si: --remote: %v\n", err)
		}
	}()
	return rs, exitOK
}

func web(dir string, o *options, stdout, stderr io.Writer) int {
	if _, err := render.LoadCSS(o.css); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitUsage
	}
	ln, err := net.Listen("tcp", o.addr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	defer ln.Close()
	if ta, ok := ln.Addr().(*net.TCPAddr); !ok || !ta.IP.IsLoopback() {
		fmt.Fprintln(stderr, "si: si --web is loopback-only in this version")
		return exitUsage
	}
	tok := studio.NewToken()
	st, err := studio.New(dir, studio.Options{CSS: o.css, Theme: o.theme, Conf: o.conf, Title: o.title, Token: tok, Addr: ln.Addr().String(),
		Run: func(args []string, out io.Writer) int { return run(args, out, out) }})
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	url := "http://" + ln.Addr().String() + "/?token=" + tok
	fmt.Fprintf(stdout, "si --web: Studio for %s at %s (Ctrl+C stops)\n", dir, url)
	stop := make(chan struct{})
	go st.Watch(stop)
	hs := &http.Server{Handler: st.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go func() {
		<-ctx.Done()
		close(stop)
		st.Close()
		sctx, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		hs.Shutdown(sctx)
	}()
	if !o.noOpen {
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
		kind, ok := exampleKinds[strings.ToLower(o.kind)]
		if !ok {
			return usageErr(stderr, fmt.Sprintf("unknown --type %q (deck, article, guide, lab, book, demo, site, all)", o.kind))
		}
		if err := writeExampleKind(out, kind); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		cmd := "si play"
		if kind != "deck" {
			cmd = "si serve"
		}
		fmt.Fprintf(stdout, "si: wrote the %s example to %s; try: %s %s\n", kind, out, cmd, out)
		return exitOK
	case "agent-rules":
		if o.output == "-" {
			fmt.Fprint(stdout, agentRules())
			return exitOK
		}
		dir := o.output
		if dir == "" {
			dir = "."
		}
		wrote, err := writeAgentRules(dir)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: wrote %s (for Claude Code and other coding agents)\n", strings.Join(wrote, " and "))
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
	return usageErr(stderr, fmt.Sprintf("unknown --export %q (css-rules, rules, conf, example, themes, agent-rules)", o.export))
}

// agentRules is the agent guide, stamped with the si version that wrote it.
func agentRules() string {
	return "<!-- written by si " + version + " (si --export agent-rules); regenerate after upgrading si -->\n\n" + docs.AgentRules
}

// writeAgentRules writes CLAUDE.md (Claude Code) and AGENTS.md (Codex, Cursor
// and other agents) into dir. It refuses to overwrite either: a project's
// existing agent file is the author's, so they merge by hand (or use -o -).
func writeAgentRules(dir string) ([]string, error) {
	names := []string{"CLAUDE.md", "AGENTS.md"}
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return nil, fmt.Errorf("%s already exists; merge by hand from `si --export agent-rules -o -`, or pick another --output dir", filepath.Join(dir, n))
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var wrote []string
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte(agentRules()), 0o644); err != nil {
			return wrote, err
		}
		wrote = append(wrote, p)
	}
	return wrote, nil
}

// buildGuide writes every page of g into out (mirroring its folders) and an
// index.html (the first page, or a redirect to it). It returns the first
// page's path inside out, and false when a page or link had a problem.
func buildGuide(g *deck.Guide, out string, o *options, css []render.Stylesheet, cfg *config.Config, stdout, stderr io.Writer) (string, bool) {
	mode := render.MediaInline
	if o.noInline {
		mode = render.MediaCopy
	}
	ro := render.Options{UserCSS: css, Media: mode, Theme: o.theme, Config: cfg}
	chrome := ""
	if o.pdf && g.IsBook() {
		if chrome = findChrome(); chrome == "" {
			fmt.Fprintln(stderr, "si: --pdf needs Google Chrome/Chromium; none found — writing HTML only")
		} else {
			ro.BookPDF = "book.pdf"
		}
	}
	pages, probs, warns := render.RenderGuide(g, ro)
	for _, w := range warns {
		fmt.Fprintf(stderr, "  warning: %s\n", w)
	}
	for _, p := range probs {
		fmt.Fprintf(stderr, "  error: %s\n", p)
	}
	ok := len(probs) == 0
	for _, p := range pages {
		dest := filepath.Join(out, filepath.FromSlash(p.Rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return "", false
		}
		for _, m := range p.Result.Media {
			if err := copyFile(m.Src, filepath.Join(filepath.Dir(dest), filepath.FromSlash(m.Rel))); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				ok = false
			}
		}
		if err := os.WriteFile(dest, p.Result.HTML, 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return "", false
		}
		fmt.Fprintf(stdout, "  %-40s %s  %4d KB\n", filepath.ToSlash(filepath.Join(filepath.Base(out), p.Rel)), p.Deck.Meta.Type, len(p.Result.HTML)/1024)
	}
	if len(pages) == 0 {
		return "", false
	}
	if chrome != "" { // a book: the one-file edition printed to PDF
		if err := deckToPDF(chrome, filepath.Join(out, "book.html"), filepath.Join(out, "book.pdf")); err != nil {
			fmt.Fprintf(stderr, "si: book pdf: %v\n", err)
			ok = false
		} else {
			fmt.Fprintf(stdout, "  %-40s (pdf)\n", filepath.ToSlash(filepath.Join(filepath.Base(out), "book.pdf")))
		}
	}
	home := pages[0].Rel
	if g.IsBook() {
		home = "index.html" // the cover
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		redirect := fmt.Sprintf("<!doctype html><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=%s\"><title>%s</title><a href=\"%s\">%s</a>\n",
			home, htmlEscape(g.Meta.Title), home, htmlEscape(g.Meta.Title))
		if err := os.WriteFile(filepath.Join(out, "index.html"), []byte(redirect), 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return "", false
		}
	}
	for _, sub := range subGuidesOf(g.Items) { // the guides, books and sites its menu links to
		rel, _ := filepath.Rel(g.Dir, sub.Dir)
		if _, subOK := buildGuide(sub, filepath.Join(out, rel), o, css, cfg, stdout, stderr); !subOK {
			ok = false
		}
	}
	kind := "guide"
	if g.IsBook() {
		kind = "book"
	} else if g.IsSite() {
		kind = "site"
	}
	fmt.Fprintf(stdout, "si: %s %q, %d page(s) -> %s\n", kind, g.Meta.Title, len(pages), out)
	return home, ok
}

// subGuidesOf loads the sub-guides (books, sites) a menu links to.
func subGuidesOf(items []*deck.NavItem) []*deck.Guide {
	var out []*deck.Guide
	var walk func([]*deck.NavItem)
	walk = func(its []*deck.NavItem) {
		for _, it := range its {
			if it.Sub != "" {
				if g, err := deck.LoadGuide(it.Sub); err == nil {
					out = append(out, g)
				}
			}
			walk(it.Children)
		}
	}
	walk(items)
	return out
}

func htmlEscape(s string) string { return html.EscapeString(s) }

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
// exampleKinds maps --type (and its aliases) to an example set.
var exampleKinds = map[string]string{"": "deck", "deck": "deck", "slides": "deck", "article": "article",
	"guide": "guide", "guides": "guide", "tutorial": "guide", "tutorials": "guide", "lab": "lab",
	"book": "book", "reference": "book", "demo": "demo", "site": "site", "all": "all"}

// writeExampleKind writes one example set into dir (which must not exist or
// be empty): deck = the showcase decks (today's export), article = the article
// alone, guide = the guide folder's contents, lab = the lab and its figure,
// all = the decks with the article beside them and the guide in guide/.
func writeExampleKind(dir, kind string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s exists and is not empty", dir)
	}
	keep := func(p string) (string, bool) { // embedded path -> output path
		top, _, _ := strings.Cut(p, "/")
		switch kind {
		case "deck":
			return p, top != "article" && top != "guide" && top != "book" && top != "site"
		case "book":
			r, ok := strings.CutPrefix(p, "book/")
			return r, ok
		case "demo":
			r, ok := strings.CutPrefix(p, "demos/")
			return r, ok
		case "article":
			r, ok := strings.CutPrefix(p, "article/")
			return r, ok
		case "guide":
			r, ok := strings.CutPrefix(p, "guide/")
			return r, ok
		case "site":
			r, ok := strings.CutPrefix(p, "site/")
			return r, ok
		case "lab":
			return p, p == "order-lab.md" || p == "assets/floor.svg"
		default: // all
			if r, ok := strings.CutPrefix(p, "article/"); ok {
				return r, true
			}
			return p, true
		}
	}
	return fs.WalkDir(examples.FS, ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || p == "embed.go" { // the package's own source, not an example file
			return err
		}
		out, ok := keep(p)
		if !ok {
			return nil
		}
		dest := filepath.Join(dir, filepath.FromSlash(out))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		raw, err := examples.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, raw, 0o644)
	})
}

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
		if top, _, _ := strings.Cut(p, "/"); top == "article" || top == "guide" || top == "book" {
			if e.IsDir() {
				return fs.SkipDir // si new scaffolds a deck; the document examples are --export example --type
			}
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

// listThemes prints the built-in themes: the name to pass to --theme or
// theme:, its display name (also accepted, quoted), aliases and a summary.
func listThemes(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tLABEL\tALIASES\tABOUT")
	for _, n := range assets.ThemeNames() {
		th := assets.Themes[n]
		al := strings.Join(assets.ThemeAliases(n), ", ")
		if al == "" {
			al = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", n, th.Label, al, th.About)
	}
	tw.Flush()
	fmt.Fprintln(w, "\nUse: si build --theme acemq-pro decks/   (or --theme \"AceMQ Pro\", or theme: in front matter)")
	fmt.Fprintln(w, "A path to a .css file or an https:// URL also works as a theme.")
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
		md := assets.ThemeSample(name)
		if md == "" {
			md = strings.ReplaceAll(themeSample, "THEME", name)
		}
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

Points: Try it
- Press d to flip light/dark and see both palettes

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

func firstOf(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func guideKindOf(g *deck.Guide) string {
	switch {
	case g.IsBook():
		return "book"
	case g.IsSite():
		return "site"
	}
	return "guide"
}
