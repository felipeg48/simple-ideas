package render

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
)

// What other front ends (the terminal renderer in internal/tui) share with
// the HTML renderer: the same markdown dialect and the same highlighter.

// KindHighlight is the AST node kind of a ==highlight==.
var KindHighlight = kindHighlight

// NewParser is si's markdown dialect (GFM plus ==highlight==) for walking the
// AST; its HTML renderer is not si's.
func NewParser() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM, highlightExt{}))
}

// Highlight is si's code highlighter: HTML with <span class="si-tk-KIND"> tokens
// (c comment, s string, k keyword, n number, f function, v variable, m meta, y type).
func Highlight(code, lang string) string { return highlight(code, lang) }

// TextOf is the plain text under a node.
func TextOf(n ast.Node, src []byte) string { return textOf(n, src) }

// DemoFiles expands a deck's demos: value (files and folders, relative to
// base) into its markdown files, and the entries it could not read.
func DemoFiles(base, spec string) ([]string, []string) { return demoFiles(base, spec) }

// RunLang reports whether a fence language is a command (Copy / Run in the
// Demos panel, a runnable entry in the terminal Lab).
func RunLang(lang string) bool { return runLangs[lang] }

// ResolveState applies an svg: directive's words to the state the previous
// slide left the same canvas in (relative +w / -w, else absolute).
func ResolveState(prev, words []string) []string { return resolveState(prev, words) }

// ApplyOps applies one build step's state ops (w, +w, -w, `.`) to a state.
func ApplyOps(state []string, ops string) []string { return applyOps(state, ops) }
