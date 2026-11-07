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
	theme := dr.opt.Theme
	if theme == "" {
		theme = dr.d.Meta.Theme
	}
	if theme == "" && dr.opt.Config != nil {
		theme = dr.opt.Config.Theme
	}
	if theme == "" {
		theme = "studio"
	}
	if strings.HasPrefix(theme, "http://") || strings.HasPrefix(theme, "https://") {
		css, err := fetchThemeCSS(theme)
		if err != nil {
			dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("theme: cannot fetch %s: %v", theme, err)})
			dr.themeName, dr.themeCSS = "studio", assets.ThemeCSS("studio")
			return
		}
		dr.themeName, dr.themeCSS = "custom", "@layer si.theme{\n"+css+"\n}\n"
		return
	}
	if strings.HasSuffix(strings.ToLower(theme), ".css") {
		p := theme
		if !filepath.IsAbs(p) {
			if _, err := os.Stat(p); err != nil {
				p = filepath.Join(filepath.Dir(dr.d.Path), theme)
			}
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("theme: cannot read %s: %v", theme, err)})
			dr.themeName, dr.themeCSS = "studio", assets.ThemeCSS("studio")
			return
		}
		dr.themeName = "custom"
		dr.themeCSS = "@layer si.theme{\n" + inlineURLs(filepath.Dir(p), string(raw)) + "\n}\n"
		return
	}
	dr.themeName = assets.CanonicalTheme(theme)
	dr.themeCSS = assets.ThemeCSS(dr.themeName)
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
