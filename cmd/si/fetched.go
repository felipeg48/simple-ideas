package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/pinned"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// newFetcher is the client of si's commands: GitHub's API (gh for private
// repos), ~/.si/cache, ~/.si/trusted asked at the terminal (on stderr), yes
// for the run with --trust.
func newFetcher(trust bool, stderr io.Writer) *pinned.Client {
	c := pinned.New(pinned.NewTrust(trust, stderr))
	c.Log = stderr
	if api := os.Getenv("SI_GITHUB_API"); api != "" { // tests and GitHub Enterprise
		c.API = api
	}
	return c
}

// fetchTarget turns what si demo, play, build and serve take into a local
// path: github:owner/repo@REF/path (the repo at that pin, its layout kept) or
// a pinned URL (one file) from the cache, fetched once and trusted once per
// pin. A local path comes back as it is, with name "". name is the repo (or
// the URL's file) for output folders. preview lists, from the local path, the
// files the trust question shows.
func fetchTarget(t string, c func() *pinned.Client, preview func(string) []string) (local, name string, err error) {
	src, ok, err := pinned.ParseTarget(t)
	if err != nil || !ok {
		return t, "", err
	}
	if src.Kind == "github" {
		local, err = c().Tree(src, preview)
		return local, src.Name(), err
	}
	paths, err := c().Scripts(src)
	if err != nil {
		return "", "", err
	}
	return paths[0], src.Name(), nil
}

// demoScripts are the web demo scripts of a folder (type: demo with app:,
// Markdown, AsciiDoc or JSON), in name order; a file is itself.
func demoScripts(p string) ([]string, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return []string{p}, nil
	}
	es, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range es {
		n := e.Name()
		if e.IsDir() || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") {
			continue
		}
		f := filepath.Join(p, n)
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		switch strings.ToLower(filepath.Ext(n)) {
		case ".md", ".markdown":
			if webdemo.IsWebDemo(string(raw)) {
				out = append(out, f)
			}
		case ".adoc", ".asciidoc":
			if webdemo.IsWebDemo(deck.AdocToMarkdown(string(raw))) {
				out = append(out, f)
			}
		case ".json":
			var j struct{ Type, App string }
			if json.Unmarshal(raw, &j) == nil && j.Type == "demo" && j.App != "" {
				out = append(out, f)
			}
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("%s holds no web demo scripts (type: demo with app:)", p)
	}
	return out, nil
}

// fetchSources fetches the remote sources a script's lines name (gist:,
// github:, pinned URLs), asking trust once per pin.
func fetchSources(s *webdemo.Script, c func() *pinned.Client) error {
	if len(s.Sources()) == 0 {
		return nil
	}
	return s.Resolve(c().Scripts)
}
