package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/felipeg48/simple-ideas/docs"
	"github.com/felipeg48/simple-ideas/internal/docsite"
	"github.com/felipeg48/simple-ideas/internal/mcp"
)

// runMCP is `si mcp [dir]`: an MCP server on stdio for AI agents, with tools
// that check, explain, start, build, picture and QR-code si documents in dir
// (default: the current folder). Paths outside dir are refused.
func runMCP(rest []string, stdout, stderr io.Writer) int {
	dir := "."
	if len(rest) > 0 {
		dir = rest[0]
	}
	root, err := filepath.Abs(dir)
	if err == nil {
		_, err = os.Stat(root)
	}
	if err != nil {
		fmt.Fprintf(stderr, "si mcp: %v\n", err)
		return exitUsage
	}
	s := &mcp.Server{Name: "si", Version: version, Tools: mcpTools(root), Instructions: mcpInstructions}
	if err := s.Serve(os.Stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "si mcp: %v\n", err)
		return exitProblems
	}
	return exitOK
}

const mcpInstructions = "si turns Markdown into slide decks (with a teleprompter script in Notes:), articles, guides, books, mind maps and code shows. " +
	"Work in a loop: write the Markdown, call si_check and fix every problem, call si_screenshot to see the result, and only then report it done. " +
	"Look syntax up with si_rules instead of guessing. Notes: is what the speaker says to the room, word for word; Points: is the speaker's own list. " +
	"A theme is a .css file of --si-* tokens: si_theme_spec for the tokens, si_theme_write, si_theme_check (fix every contrast pair), si_theme_preview light and dark."

// mcpMu runs one si command at a time: the CLI keeps a few package settings.
var mcpMu sync.Mutex

func mcpTools(root string) []mcp.Tool {
	// in resolves a path the agent gave, inside root only
	in := func(p string) (string, error) {
		if p == "" {
			p = "."
		}
		abs := p
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, p)
		}
		abs = filepath.Clean(abs)
		if rel, err := filepath.Rel(root, abs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%s is outside the folder si mcp serves (%s)", p, root)
		}
		return abs, nil
	}
	si := func(args ...string) (string, int) {
		mcpMu.Lock()
		defer mcpMu.Unlock()
		var out bytes.Buffer
		code := run(args, &out, &out)
		return strings.TrimSpace(out.String()), code
	}
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	obj := func(req []string, props map[string]any) map[string]any {
		o := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 { // "required" must be an array or absent, never null
			o["required"] = req
		}
		return o
	}
	relTo := func(p string) string {
		if r, err := filepath.Rel(root, p); err == nil {
			return filepath.ToSlash(r)
		}
		return p
	}
	problemRE := regexp.MustCompile(`^(.+?):(\d+): (.+)$`)
	return append([]mcp.Tool{
		{Name: "si_check", Title: "Check si documents", ReadOnly: true,
			Description: "Validate a deck, document or folder like `si check`: front matter, directives, layouts, files, includes, links, Notes: [CLICK] cues. Returns every problem with its file and line; fix them all before you finish.",
			Input:       obj([]string{"path"}, map[string]any{"path": str("a .md file or a folder, relative to the served folder")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Path string }
				json.Unmarshal(raw, &a)
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				out, code := si("check", p)
				problems := []map[string]any{}
				for _, l := range strings.Split(out, "\n") {
					if m := problemRE.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
						n, _ := strconv.Atoi(m[2])
						problems = append(problems, map[string]any{"file": relTo(m[1]), "line": n, "message": m[3]})
					}
				}
				return mcp.Result{Text: out, Structured: map[string]any{"ok": code == exitOK, "problems": problems}}, nil
			}},
		{Name: "si_rules", Title: "Look up the si format", ReadOnly: true,
			Description: "The si Markdown format, one section at a time. No arguments: the list of sections. section: a number (\"13\") or a word from its title. query: the sections that mention it (e.g. \"qr\", \"two-col\", \"Notes:\").",
			Input:       obj(nil, map[string]any{"section": str("section number or title word"), "query": str("text to search the rules for")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Section, Query string }
				json.Unmarshal(raw, &a)
				_, secs := docsite.Split(docs.Rules)
				if a.Section == "" && a.Query == "" {
					var b strings.Builder
					for _, s := range secs {
						fmt.Fprintf(&b, "%s. %s\n", s.N, s.Title)
					}
					return mcp.Result{Text: b.String()}, nil
				}
				var b strings.Builder
				n := 0
				for _, s := range secs {
					hit := a.Section != "" && (s.N == a.Section || strings.Contains(strings.ToLower(s.Title), strings.ToLower(a.Section)))
					hit = hit || a.Query != "" && strings.Contains(strings.ToLower(s.Body), strings.ToLower(a.Query))
					if hit && n < 3 {
						fmt.Fprintf(&b, "# %s. %s\n\n%s\n\n", s.N, s.Title, strings.TrimSpace(s.Body))
						n++
					}
				}
				if n == 0 {
					return mcp.Result{}, fmt.Errorf("no section matches; call si_rules with no arguments for the list")
				}
				return mcp.Result{Text: b.String()}, nil
			}},
		{Name: "si_new", Title: "Start a document",
			Description: "Create a small starter of a kind, to edit: deck, article, guide, book, site, page, lab, demo, mindmap, mindmap3d or codeshow. A guide, book or site is a folder; the rest a .md file.",
			Input:       obj([]string{"name", "type"}, map[string]any{"name": str("file or folder name, relative to the served folder"), "type": str("deck, article, guide, book, site, page, lab, demo, mindmap, mindmap3d or codeshow")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Name, Type string }
				json.Unmarshal(raw, &a)
				p, err := in(a.Name)
				if err != nil {
					return mcp.Result{}, err
				}
				out, code := si("new", p, "--type", a.Type)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
		{Name: "si_build", Title: "Build HTML",
			Description: "Build self-contained HTML (one file per deck or page, plus an index) like `si build`; pdf also renders each deck to a PDF (needs Chrome).",
			Input: obj([]string{"path"}, map[string]any{"path": str("a .md file or a folder"), "output": str("output folder (default: html)"),
				"pdf": map[string]any{"type": "boolean", "description": "also write PDFs"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Path, Output string
					PDF          bool
				}
				json.Unmarshal(raw, &a)
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				if a.Output == "" {
					a.Output = "html"
				}
				o, err := in(a.Output)
				if err != nil {
					return mcp.Result{}, err
				}
				args := []string{"build", p, "-o", o}
				if a.PDF {
					args = append(args, "--pdf")
				}
				out, code := si(args...)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
		{Name: "si_qr", Title: "Make a QR code file",
			Description: "Write a QR code for a link or text as png, jpg, svg and/or pdf (like `si qr`). On a slide, write <!-- qr: https://… · caption=… --> instead.",
			Input: obj([]string{"text", "output"}, map[string]any{"text": str("the link or text"), "output": str("file name; its extension picks the format"),
				"formats": str("comma list: png,jpg,svg,pdf"), "logo": str("a PNG/JPEG (or SVG for svg output) for the middle"), "style": str("square or dots")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Text, Output, Formats, Logo, Style string }
				json.Unmarshal(raw, &a)
				o, err := in(a.Output)
				if err != nil {
					return mcp.Result{}, err
				}
				args := []string{"qr", a.Text, "-o", o}
				if a.Formats != "" {
					args = append(args, "--type", a.Formats)
				}
				if a.Logo != "" {
					l, err := in(a.Logo)
					if err != nil {
						return mcp.Result{}, err
					}
					args = append(args, "--logo", l)
				}
				if a.Style != "" {
					args = append(args, "--style", a.Style)
				}
				out, code := si(args...)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
		{Name: "si_screenshot", Title: "See a slide or page", ReadOnly: true,
			Description: "Build a deck or document and return a picture of it (headless Chrome): a slide by number (\"5\", or \"5.2\" for its second build step), at a screen size, light or dark. Look at it: a deck that passes si_check can still look wrong.",
			Input: obj([]string{"path"}, map[string]any{"path": str("the .md file"), "slide": str("slide number, e.g. 3 or 3.2 (decks)"),
				"width": map[string]any{"type": "integer", "description": "default 1280"}, "height": map[string]any{"type": "integer", "description": "default 720"},
				"dark": map[string]any{"type": "boolean"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Path, Slide   string
					Width, Height int
					Dark          bool
				}
				json.Unmarshal(raw, &a)
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				tmp, err := os.MkdirTemp("", "si-mcp-shot-")
				if err != nil {
					return mcp.Result{}, err
				}
				defer os.RemoveAll(tmp)
				if out, code := si("build", p, "-o", tmp); code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				page := filepath.Join(tmp, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+".html")
				w, h := max(a.Width, 0), max(a.Height, 0)
				if w == 0 {
					w = 1280
				}
				if h == 0 {
					h = 720
				}
				img, err := chromeShot(page, a.Slide, w, h, a.Dark)
				if err != nil {
					return mcp.Result{}, err
				}
				return mcp.Result{Text: fmt.Sprintf("%s%s at %dx%d", relTo(p), map[bool]string{true: " #" + a.Slide, false: ""}[a.Slide != ""], w, h),
					Images: []mcp.Image{{MIME: "image/png", Data: base64.StdEncoding.EncodeToString(img)}}}, nil
			}},
	}, mcpThemeTools(in, si, relTo)...)
}

// chromeShot is a PNG of a built page (a slide by number for a deck) in
// headless Chrome, light or dark.
func chromeShot(page, slide string, w, h int, dark bool) ([]byte, error) {
	chrome := findChrome()
	if chrome == "" {
		return nil, fmt.Errorf("no Chrome or Chromium found for screenshots")
	}
	if _, err := os.Stat(page); err != nil {
		return nil, fmt.Errorf("built no %s", filepath.Base(page))
	}
	url := "file://" + filepath.ToSlash(page)
	if slide != "" {
		url += "#" + slide
	}
	png := filepath.Join(filepath.Dir(page), "shot.png")
	args := []string{"--headless=new", "--disable-gpu", "--hide-scrollbars", "--screenshot=" + png, fmt.Sprintf("--window-size=%d,%d", w, h), "--virtual-time-budget=3000"}
	if dark {
		args = append(args, "--force-dark-mode", "--blink-settings=preferredColorScheme=0")
	}
	if b, err := exec.Command(chrome, append(args, url)...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("chrome: %v: %s", err, b)
	}
	return os.ReadFile(png)
}
