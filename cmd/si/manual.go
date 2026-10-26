package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	docs "github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// runManual renders the embedded user-manual deck to --out as si-manual.html
// (always), and si-manual.pdf when --pdf is set. --slides is the default and
// just confirms HTML output. Only si-manual.* is written, so it never clobbers
// an existing index.html in the output directory.
func runManual(o *options, stdout, stderr io.Writer) int {
	out := o.output
	if out == "" {
		out = "."
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	// deck.Parse needs a path (for the deck's Dir); the manual has no external
	// assets, so a temp file is enough.
	dir, err := os.MkdirTemp("", "si-manual")
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	defer os.RemoveAll(dir)
	md := filepath.Join(dir, "si-manual.md")
	if err := os.WriteFile(md, []byte(docs.Manual), 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	d, err := deck.Parse(md)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	res := render.Deck(d, render.Options{Media: render.MediaInline})
	for _, p := range res.Problems {
		fmt.Fprintf(stderr, "  error: %s\n", p)
	}
	if len(res.Problems) > 0 {
		return exitProblems
	}
	dest := filepath.Join(out, "si-manual.html")
	if err := os.WriteFile(dest, res.HTML, 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: wrote %s\n", dest)
	if o.pdf {
		chrome := findChrome()
		if chrome == "" {
			fmt.Fprintln(stderr, "si: --pdf needs Google Chrome/Chromium; wrote HTML only")
			return exitOK
		}
		pdf := filepath.Join(out, "si-manual.pdf")
		if err := deckToPDF(chrome, dest, pdf); err != nil {
			fmt.Fprintf(stderr, "si: pdf: %v\n", err)
			return exitProblems
		}
		fmt.Fprintf(stdout, "si: wrote %s\n", pdf)
	}
	return exitOK
}
