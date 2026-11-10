package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/mcp"
	"github.com/felipeg48/simple-ideas/internal/themefile"
)

// The theme tools of si mcp: an agent designs a theme the way it writes a
// deck: write it, check it, look at it light and dark, fix, look again.
func mcpThemeTools(in func(string) (string, error), si func(...string) (string, int), relTo func(string) string) []mcp.Tool {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	tokens := func(desc string) map[string]any {
		return map[string]any{"type": "object", "description": desc, "additionalProperties": map[string]any{"type": "string"}}
	}
	obj := func(req []string, props map[string]any) map[string]any {
		o := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			o["required"] = req
		}
		return o
	}
	report := func(f *themefile.File, text string) map[string]any {
		issues := f.Check(text)
		if issues == nil {
			issues = []themefile.Issue{}
		}
		ok := true
		for _, is := range issues {
			ok = ok && is.Mode != ""
		}
		return map[string]any{"ok": ok, "passes": len(issues) == 0, "issues": issues}
	}
	return []mcp.Tool{
		{Name: "si_theme_spec", Title: "List theme tokens", ReadOnly: true,
			Description: "The tokens a theme sets (--si-*), by group, with each one's kind (color, length, font, number, enum, duration, shadow), its allowed units or values, and whether it takes a light and a dark value; and the built-in themes to extend. theme: one built-in's full values.",
			Input:       obj(nil, map[string]any{"theme": str("a built-in theme name, for its values")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Theme string }
				json.Unmarshal(raw, &a)
				if a.Theme != "" {
					l, d, ok := assets.ThemeTokens(a.Theme)
					if !ok {
						return mcp.Result{}, fmt.Errorf("no built-in theme %q", a.Theme)
					}
					return mcp.Result{Text: fmt.Sprintf("%s: %d tokens, %d with a dark value", a.Theme, len(l), len(d)), Structured: map[string]any{"light": l, "dark": d}}, nil
				}
				var toks []assets.TokenMeta
				for _, m := range assets.AllMeta() {
					if m.Group != "Element" {
						toks = append(toks, m)
					}
				}
				var themes []map[string]string
				for _, n := range assets.ThemeNames() {
					if !assets.UnlistedThemes[n] {
						themes = append(themes, map[string]string{"name": n, "label": assets.Themes[n].Label, "about": assets.Themes[n].About})
					}
				}
				return mcp.Result{Text: fmt.Sprintf("%d theme tokens in %d groups; %d built-in themes to extend", len(toks), len(assets.TokenGroups)-1, len(themes)),
					Structured: map[string]any{"groups": assets.TokenGroups[:len(assets.TokenGroups)-1], "tokens": toks, "themes": themes}}, nil
			}},
		{Name: "si_theme_write", Title: "Write a theme",
			Description: "Write a .css theme (default themes/<name>.css): the built-in it extends, and the tokens it changes, light and dark (\"--si-accent\": \"#0b3d91\"; an empty value removes one). An existing file keeps its other tokens and rules unless replace is true. Only known tokens with values of their kind are written (lengths in em, rem, cqi, cqb, %, ch; never px). Returns the file and its checks (WCAG contrast in light and dark, with a passing colour for each failing pair).",
			Input: obj([]string{"name"}, map[string]any{"name": str("the theme's name (Acme)"), "path": str("the .css file (default themes/<name>.css)"),
				"extends": str("a built-in theme to start from (studio, paper, …); empty: nothing underneath"),
				"light":   tokens("token: value, both modes unless dark sets it (fonts, sizes, light colours)"),
				"dark":    tokens("token: value in dark mode (colours, shadows)"),
				"replace": map[string]any{"type": "boolean", "description": "start the file over"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Name, Path, Extends string
					Light, Dark         map[string]string
					Replace             bool
				}
				json.Unmarshal(raw, &a)
				if a.Path == "" {
					a.Path = "themes/" + strings.Trim(slugRE.ReplaceAllString(strings.ToLower(a.Name), "-"), "-") + ".css"
				}
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				if !strings.HasSuffix(p, ".css") {
					return mcp.Result{}, fmt.Errorf("a theme is a .css file")
				}
				f := themefile.New("", "")
				if b, err := os.ReadFile(p); err == nil && !a.Replace {
					f = themefile.Parse(string(b))
				}
				if a.Name != "" {
					f.Name = strings.NewReplacer("*/", "", "\n", " ").Replace(a.Name)
				}
				if a.Extends != "" {
					if _, _, ok := assets.ThemeTokens(a.Extends); !ok {
						return mcp.Result{}, fmt.Errorf("no built-in theme %q (si_theme_spec lists them)", a.Extends)
					}
					f.Extends = assets.CanonicalTheme(a.Extends)
				}
				var errs []string
				for mode, m := range map[string]map[string]string{"light": a.Light, "dark": a.Dark} {
					for k, v := range m {
						if !strings.HasPrefix(k, "--") {
							k = "--si-" + k
						}
						if err := f.Set(k, mode, v); err != nil {
							errs = append(errs, err.Error())
						}
					}
				}
				if len(errs) > 0 {
					return mcp.Result{}, fmt.Errorf("not written: %s", strings.Join(errs, "; "))
				}
				text := f.String()
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return mcp.Result{}, err
				}
				if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
					return mcp.Result{}, err
				}
				st := report(f, text)
				st["path"] = relTo(p)
				return mcp.Result{Text: relTo(p) + "\n\n" + text, Structured: st}, nil
			}},
		{Name: "si_theme_check", Title: "Check a theme", ReadOnly: true,
			Description: "Check a .css theme like `si theme check`: unknown tokens, values of the wrong kind, and every text/background pair under its WCAG minimum in light or dark (4.5:1 for text, 3:1 for accents and status colours), each with the nearest passing colour (fix).",
			Input:       obj([]string{"path"}, map[string]any{"path": str("the .css theme")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Path string }
				json.Unmarshal(raw, &a)
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return mcp.Result{}, err
				}
				var out strings.Builder
				printThemeIssues(relTo(p), themefile.Parse(string(b)).Check(string(b)), &out)
				st := report(themefile.Parse(string(b)), string(b))
				if out.Len() == 0 {
					out.WriteString(relTo(p) + ": ok")
				}
				return mcp.Result{Text: out.String(), Structured: st}, nil
			}},
		{Name: "si_theme_preview", Title: "See a theme", ReadOnly: true,
			Description: "A picture of a theme in use: the specimen deck (headings, lists, columns, a table, a callout, a code window, a big number, a quote, a mind map, a QR code: slides 1 to 9), or a deck of yours, drawn with the theme, light or dark. Look at both modes.",
			Input: obj([]string{"path"}, map[string]any{"path": str("the .css theme"), "deck": str("a deck to show instead of the specimen"),
				"slide": str("slide number (default 2)"), "dark": map[string]any{"type": "boolean"},
				"width": map[string]any{"type": "integer", "description": "default 1280"}, "height": map[string]any{"type": "integer", "description": "default 720"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Path, Deck, Slide string
					Dark              bool
					Width, Height     int
				}
				json.Unmarshal(raw, &a)
				p, err := in(a.Path)
				if err != nil {
					return mcp.Result{}, err
				}
				tmp, err := os.MkdirTemp("", "si-mcp-theme-")
				if err != nil {
					return mcp.Result{}, err
				}
				defer os.RemoveAll(tmp)
				src := filepath.Join(tmp, "specimen.md")
				if a.Deck != "" {
					if src, err = in(a.Deck); err != nil {
						return mcp.Result{}, err
					}
				} else if err := os.WriteFile(src, []byte(assets.Specimen()), 0o644); err != nil {
					return mcp.Result{}, err
				}
				out := filepath.Join(tmp, "out")
				if o, code := si("build", src, "-o", out, "--theme", p); code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", o)
				}
				if a.Slide == "" {
					a.Slide = "2"
				}
				w, h := max(a.Width, 0), max(a.Height, 0)
				if w == 0 {
					w = 1280
				}
				if h == 0 {
					h = 720
				}
				img, err := chromeShot(filepath.Join(out, strings.TrimSuffix(filepath.Base(src), ".md")+".html"), a.Slide, w, h, a.Dark)
				if err != nil {
					return mcp.Result{}, err
				}
				mode := map[bool]string{true: "dark", false: "light"}[a.Dark]
				return mcp.Result{Text: fmt.Sprintf("%s, slide %s, %s, %dx%d", relTo(p), a.Slide, mode, w, h),
					Images: []mcp.Image{{MIME: "image/png", Data: base64.StdEncoding.EncodeToString(img)}}}, nil
			}},
		{Name: "si_theme_export", Title: "Export a theme", ReadOnly: true,
			Description: "A theme as one file to share: a built-in in full or a .css theme with its url()s inlined (format css), or W3C Design Tokens JSON for Figma (Tokens Studio), Style Dictionary and others (format dtcg; tokens with no Design Tokens type are listed, not guessed). output: also write it there.",
			Input:       obj([]string{"theme"}, map[string]any{"theme": str("a built-in name or a .css theme"), "format": str("css (default) or dtcg"), "output": str("a file to write")}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct{ Theme, Format, Output string }
				json.Unmarshal(raw, &a)
				src := a.Theme
				if _, ok := themefile.Builtin(src); !ok {
					p, err := in(src)
					if err != nil {
						return mcp.Result{}, err
					}
					src = p
				}
				args := []string{"theme", "export", src}
				if a.Format != "" {
					args = append(args, "--format", a.Format)
				}
				if a.Output != "" {
					p, err := in(a.Output)
					if err != nil {
						return mcp.Result{}, err
					}
					args = append(args, "-o", p)
				}
				out, code := si(args...)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
		{Name: "si_theme_import", Title: "Import a theme",
			Description: "Bring a theme in: a .css theme or a Design Tokens .tokens.json (a file in the folder or an https:// URL). It is checked (a broken value refuses it), written to themes/<name>.css, and with use: true set as the folder's theme in si.conf.",
			Input:       obj([]string{"source"}, map[string]any{"source": str("a file or an https:// URL"), "use": map[string]any{"type": "boolean", "description": "also write theme: into si.conf"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Source string
					Use    bool
				}
				json.Unmarshal(raw, &a)
				src := a.Source
				if !strings.HasPrefix(src, "https://") && !strings.HasPrefix(src, "http://") {
					p, err := in(src)
					if err != nil {
						return mcp.Result{}, err
					}
					src = p
				}
				root, _ := in(".")
				args := []string{"theme", "import", src, "-o", filepath.Join(root, "themes")}
				if a.Use {
					args = append(args, "--use", "--conf", filepath.Join(root, "si.conf"))
				}
				out, code := si(args...)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
		{Name: "si_theme_from_logo", Title: "Make a theme from a logo",
			Description: "A theme from a logo's colours (a PNG, JPEG, GIF or SVG in the folder, up to 8 MiB): the brand colour becomes the accent, a second one accent-2, backgrounds and text are neutrals tinted with the accent's hue, light and dark, and every text/background pair passes WCAG AA (colours moved for contrast are noted in the file's header). Written to themes/<name>.css (or path); use: true also sets it in si.conf. Look at it with si_theme_preview, then pick fonts.",
			Input: obj([]string{"logo"}, map[string]any{"logo": str("the logo file"), "name": str("the theme's name (default: from the file name)"),
				"accent": str("a hex colour to keep as the accent; the logo gives the rest"), "path": str("the .css file (default themes/<name>.css)"),
				"use": map[string]any{"type": "boolean", "description": "also write theme: into si.conf"}}),
			Call: func(raw json.RawMessage) (mcp.Result, error) {
				var a struct {
					Logo, Name, Accent, Path string
					Use                      bool
				}
				json.Unmarshal(raw, &a)
				logo, err := in(a.Logo)
				if err != nil {
					return mcp.Result{}, err
				}
				root, _ := in(".")
				dest := filepath.Join(root, "themes")
				if a.Path != "" {
					if dest, err = in(a.Path); err != nil {
						return mcp.Result{}, err
					}
				}
				args := []string{"theme", "from", logo, "-o", dest}
				if a.Name != "" {
					args = append(args, "--name", a.Name)
				}
				if a.Accent != "" {
					args = append(args, "--accent", a.Accent)
				}
				if a.Use {
					args = append(args, "--use", "--conf", filepath.Join(root, "si.conf"))
				}
				out, code := si(args...)
				if code != exitOK {
					return mcp.Result{}, fmt.Errorf("%s", out)
				}
				return mcp.Result{Text: out}, nil
			}},
	}
}
