// Themes and break-timer animations: resolving the deck theme (built-in, a
// CSS file or a URL, cached) and the configured animations.

package render

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/themefile"
)

// resolveAnim resolves the configured break-timer animation(s) into dr.anims.
// timer.animation names one or more built-ins (waves/orbit/bokeh) or, with
// timer.animation.path set, <name>.js|.svg files in that directory (resolved
// relative to cwd, then the deck dir); a comma-separated list becomes a toggle
// in the HUD. A configured-but-missing animation is a deck problem; no
// configured animation leaves dr.anims empty (no layer).
func (dr *deckRenderer) resolveAnim() {
	if dr.opt.Config == nil {
		return
	}
	t := dr.opt.Config.Timer
	if t.Animation == "" {
		return
	}
	for _, raw := range strings.Split(t.Animation, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		dr.resolveOneAnim(name, t.AnimationPath)
	}
}

func (dr *deckRenderer) resolveOneAnim(name, dir string) {
	if dir != "" {
		for _, k := range []string{"js", "svg"} {
			p := filepath.Join(dir, name+"."+k)
			if !filepath.IsAbs(p) {
				if _, err := os.Stat(p); err != nil {
					p = filepath.Join(filepath.Dir(dr.d.Path), dir, name+"."+k)
				}
			}
			if b, err := os.ReadFile(p); err == nil {
				dr.anims = append(dr.anims, animEntry{name, k, string(b)})
				return
			}
		}
	}
	if code, kind, ok := assets.Animation(name); ok {
		dr.anims = append(dr.anims, animEntry{name, kind, code})
		return
	}
	dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("timer.animation: no animation %q (built-ins: %s)", name, strings.Join(assets.AnimationNames(), ", "))})
}

// resolveTheme sets dr.themeName and dr.themeCSS. The theme is the --theme flag,
// then the deck's theme: front matter, then si.conf, then studio. A value ending
// in .css is a user theme file (resolved relative to cwd, then the deck dir); an
// http(s):// value is fetched at build time and cached. Either way its contents
// become the si.theme layer (the built HTML stays self-contained — the fetch is
// build-time only; a file theme's relative url()s are inlined like --css files). A read/fetch error is a deck problem and falls back to studio.
func (dr *deckRenderer) resolveTheme() {
	defer dr.resolveFonts() // fonts: sets tokens on top of whichever theme won
	if dr.opt.ThemeCSS != "" {
		dr.themeName, dr.themeCSS = "custom", dr.opt.ThemeCSS
		return
	}
	theme := dr.opt.Theme
	if theme == "" {
		theme = dr.d.Meta.Theme
	}
	if theme == "" && dr.opt.Config != nil {
		theme = dr.opt.Config.Theme
	}
	var err error
	dr.themeName, dr.themeCSS, err = ThemeLayer(theme, filepath.Dir(dr.d.Path))
	if err != nil {
		dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: err.Error()})
	}
}

// ThemeLayer returns the name ("custom" for a file or URL) and the si.theme
// layer CSS of a theme: a built-in name, a .css file (relative to cwd, then
// dir) or an http(s):// URL; "" is studio. A theme file whose header says
// "si theme: Name · extends <built-in>" gets that built-in's tokens first,
// the file after (it wins); without the header nothing is underneath. On a
// read or fetch error it returns studio and the error.
func ThemeLayer(theme, dir string) (name, css string, err error) {
	if theme == "" {
		theme = "studio"
	}
	var raw string
	switch {
	case strings.HasPrefix(theme, "http://") || strings.HasPrefix(theme, "https://"):
		if raw, err = fetchThemeCSS(theme); err != nil {
			return "studio", assets.ThemeCSS("studio"), fmt.Errorf("theme: cannot fetch %s: %v", theme, err)
		}
	case strings.HasSuffix(strings.ToLower(theme), ".css"):
		p := ThemePath(theme, dir)
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return "studio", assets.ThemeCSS("studio"), fmt.Errorf("theme: cannot read %s: %v", theme, rerr)
		}
		raw = inlineURLs(filepath.Dir(p), string(b))
	default:
		name = assets.CanonicalTheme(theme)
		return name, assets.ThemeCSS(name), nil
	}
	css, err = ThemeFileLayer(raw)
	if err != nil {
		err = fmt.Errorf("theme: %s %v", theme, err)
	}
	return "custom", css, err
}

// ThemeFileLayer is the si.theme layer of a theme file's text: the built-in
// its header extends (if any), then the file.
func ThemeFileLayer(raw string) (css string, err error) {
	var base string
	if ext := themefile.Parse(raw).Extends; ext != "" {
		if _, ok := assets.Themes[assets.CanonicalTheme(ext)]; ok {
			base = assets.ThemeCSS(ext)
		} else {
			err = fmt.Errorf("extends %s, which is not a built-in theme (see si theme list)", ext)
		}
	}
	return base + "@layer si.theme{\n" + raw + "\n}\n", err
}

// InlineURLs inlines the relative url()s of css (resolved against dir) as
// data URIs, as a theme file gets at build: `si theme export` uses it.
func InlineURLs(dir, css string) string { return inlineURLs(dir, css) }

// ThemePath resolves a .css theme path: as given when it exists (or is
// absolute), else relative to dir.
func ThemePath(theme, dir string) string {
	if filepath.IsAbs(theme) {
		return theme
	}
	if _, err := os.Stat(theme); err == nil || dir == "" {
		return theme
	}
	return filepath.Join(dir, theme)
}

// fetchThemeCSS downloads a theme stylesheet at build time and caches it for an
// hour under the temp dir (so `si play` does not refetch on every request). On a
// fetch error a fresh-enough cache is reused if present, otherwise the error is
// returned. The response is capped so a huge/garbage URL cannot blow up memory.
func fetchThemeCSS(url string) (string, error) {
	sum := sha256.Sum256([]byte(url))
	cache := filepath.Join(os.TempDir(), "si-theme-cache", fmt.Sprintf("%x.css", sum[:8]))
	if fi, err := os.Stat(cache); err == nil && time.Since(fi.ModTime()) < time.Hour {
		if b, err := os.ReadFile(cache); err == nil {
			return string(b), nil
		}
	}
	readCache := func() (string, bool) {
		if b, err := os.ReadFile(cache); err == nil {
			return string(b), true
		}
		return "", false
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		if b, ok := readCache(); ok {
			return b, nil // stale cache beats a hard failure
		}
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if b, ok := readCache(); ok {
			return b, nil
		}
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB cap
	if err != nil {
		return "", err
	}
	_ = os.MkdirAll(filepath.Dir(cache), 0o755)
	_ = os.WriteFile(cache, body, 0o644)
	return string(body), nil
}
