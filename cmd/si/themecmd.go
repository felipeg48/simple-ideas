package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/themefile"
)

// si theme list | check <file.css> | export <name|file.css> | import <file|url>
func themeCmd(rest []string, o *options, stdout, stderr io.Writer) int {
	if len(rest) == 0 || rest[0] == "list" {
		if len(rest) > 1 {
			return usageErr(stderr, "theme list takes no arguments")
		}
		listThemes(stdout)
		return exitOK
	}
	if len(rest) != 2 {
		return usageErr(stderr, "theme takes list, check <file.css>, export <name|file.css>, import <file|url> or from <logo>")
	}
	switch rest[0] {
	case "check":
		return checkThemeFile(rest[1], stdout, stderr)
	case "export":
		return themeExport(rest[1], o, stdout, stderr)
	case "import":
		return themeImport(rest[1], o, stdout, stderr)
	case "from":
		return themeFrom(rest[1], o, stdout, stderr)
	}
	return usageErr(stderr, "theme takes list, check, export, import or from, not "+rest[0])
}

// checkThemeFile prints a theme file's issues (si theme check, si check x.css):
// a broken value fails, a contrast pair under its minimum is a warning.
func checkThemeFile(path string, stdout, stderr io.Writer) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	issues := themefile.Parse(string(raw)).Check(string(raw))
	failed := printThemeIssues(path, issues, stdout)
	if len(issues) == 0 {
		fmt.Fprintf(stdout, "si: %s: ok\n", path)
	}
	if failed {
		return exitProblems
	}
	return exitOK
}

func printThemeIssues(path string, issues []themefile.Issue, w io.Writer) (failed bool) {
	for _, is := range issues {
		loc := path
		if is.Line > 0 {
			loc = fmt.Sprintf("%s:%d", path, is.Line)
		}
		if is.Mode == "" {
			failed = true
			fmt.Fprintf(w, "%s: %s\n", loc, is.Msg)
			continue
		}
		fix := ""
		if is.Fix != "" {
			fix = " (try " + is.Fix + ")"
		}
		fmt.Fprintf(w, "%s: warning: %s%s\n", loc, is.Msg, fix)
	}
	return failed
}

// themeExport writes a theme as one file: a built-in in full, or a theme
// file with its url()s inlined; --format dtcg writes Design Tokens JSON.
func themeExport(src string, o *options, stdout, stderr io.Writer) int {
	var f *themefile.File
	if b, ok := themefile.Builtin(src); ok {
		f = b
	} else {
		raw, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(stderr, "si: %s is no built-in theme (si theme list) and no file: %v\n", src, err)
			return exitProblems
		}
		f = themefile.Parse(render.InlineURLs(filepath.Dir(src), string(raw)))
	}
	var out []byte
	switch o.format {
	case "", "css":
		out = []byte(f.String())
	case "dtcg", "json", "tokens":
		js, unmapped := f.DTCG()
		out = js
		if len(unmapped) > 0 {
			fmt.Fprintf(stderr, "si: not in Design Tokens (kept only in the .css): %s\n", strings.Join(unmapped, ", "))
		}
	default:
		return usageErr(stderr, "--format is css or dtcg")
	}
	if o.output == "" || o.output == "-" {
		stdout.Write(out)
		return exitOK
	}
	if err := os.WriteFile(o.output, out, 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: %s -> %s\n", src, o.output)
	return exitOK
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// themeImport reads a theme (.css, or Design Tokens .json; a file or an
// https:// URL), checks it and writes it into themes/ (or --output dir) as a
// .css theme; --use also writes theme: into si.conf.
func themeImport(src string, o *options, stdout, stderr io.Writer) int {
	var raw []byte
	var err error
	if strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
		raw, err = fetchSmall(src)
	} else {
		raw, err = os.ReadFile(src)
	}
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	var f *themefile.File
	if t := strings.TrimSpace(string(raw)); strings.HasPrefix(t, "{") {
		var unmapped []string
		if f, unmapped, err = themefile.FromDTCG(raw); err != nil {
			fmt.Fprintf(stderr, "si: %s: %v\n", src, err)
			return exitProblems
		}
		if len(unmapped) > 0 {
			fmt.Fprintf(stdout, "si: not mapped to si tokens (left out): %s\n", strings.Join(unmapped, ", "))
		}
	} else {
		f = themefile.Parse(string(raw))
	}
	low := strings.ToLower(f.Extra + f.Preamble)
	for _, bad := range []string{"@import", "expression(", "</style", "javascript:"} {
		if strings.Contains(low, bad) {
			fmt.Fprintf(stderr, "si: %s: %s is not allowed in a theme\n", src, bad)
			return exitProblems
		}
	}
	base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(src), ".json"), ".tokens")
	base = strings.TrimSuffix(strings.TrimSuffix(base, ".css"), ".si-theme")
	if f.Name == "" {
		f.Name = base
	}
	name := strings.Trim(slugRE.ReplaceAllString(strings.ToLower(f.Name), "-"), "-")
	if name == "" {
		name = "theme"
	}
	text := f.String()
	if printThemeIssues(src, f.Check(text), stdout) {
		fmt.Fprintln(stderr, "si: not imported: fix the values above")
		return exitProblems
	}
	dir := o.output
	if dir == "" {
		dir = "themes"
	}
	return writeTheme(src, filepath.Join(dir, name+".css"), text, o, stdout, stderr)
}

// writeTheme writes a new theme file (never over one) and, with --use, sets
// it as the folder's theme in si.conf.
func writeTheme(src, dest, text string, o *options, stdout, stderr io.Writer) int {
	dir := filepath.Dir(dest)
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(stderr, "si: %s exists; remove it or write with --output <dir>\n", dest)
		return exitProblems
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: %s -> %s\n", src, dest)
	if o.use {
		conf := o.conf
		if conf == "" {
			conf = "si.conf"
		}
		if err := config.SetKey(conf, "theme", filepath.ToSlash(dest)); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: %s: theme: %s\n", conf, filepath.ToSlash(dest))
	} else {
		fmt.Fprintf(stdout, "Use it: theme: %s (front matter or si.conf)\n", filepath.ToSlash(dest))
	}
	return exitOK
}

// themeFrom makes a theme from a logo (si theme from logo.png): its
// colours, light and dark, every text/background pair passing WCAG AA;
// written to themes/<name>.css, or --output (a .css file or a folder).
func themeFrom(src string, o *options, stdout, stderr io.Writer) int {
	fi, err := os.Stat(src)
	if err == nil && fi.Size() > themefile.MaxLogo {
		err = fmt.Errorf("%s is over %d MiB; a logo is smaller", src, themefile.MaxLogo>>20)
	}
	var raw []byte
	if err == nil {
		raw, err = os.ReadFile(src)
	}
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	pal, err := themefile.LogoPalette(raw)
	if err != nil {
		fmt.Fprintf(stderr, "si: %s: %v\n", src, err)
		return exitProblems
	}
	name := o.qrName
	if name == "" {
		name = themefile.LogoName(src)
	}
	f, err := themefile.FromLogo(name, pal, o.accent)
	if err != nil {
		fmt.Fprintf(stderr, "si: %s: %v\n", src, err)
		return exitProblems
	}
	var cols []string
	for _, s := range pal {
		cols = append(cols, s.Hex)
	}
	fmt.Fprintf(stdout, "si: %s: colours %s; accent %s, accent-2 %s\n", src, strings.Join(cols, " "), f.Light["--si-accent"], f.Light["--si-accent-2"])
	dest := o.output
	if !strings.HasSuffix(strings.ToLower(dest), ".css") {
		if dest == "" {
			dest = "themes"
		}
		dest = filepath.Join(dest, themefile.Slug(f.Name)+".css")
	}
	return writeTheme(src, dest, f.String(), o, stdout, stderr)
}

// fetchSmall downloads up to 2 MiB.
func fetchSmall(url string) ([]byte, error) {
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}
