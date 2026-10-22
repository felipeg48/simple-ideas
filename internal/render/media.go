package render

import (
	"encoding/base64"
	"fmt"
	"html"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// MediaMode says how images, videos and background files reach the page.
type MediaMode int

const (
	// MediaInline embeds every file as a data URI: one self-contained HTML file.
	MediaInline MediaMode = iota
	// MediaLink links files inside the deck directory by relative URL and
	// inlines the rest (si play: the server serves the deck directory).
	MediaLink
	// MediaCopy links every file by relative URL; files outside the deck
	// directory go to media/. The caller copies Result.Media next to the HTML.
	MediaCopy
)

// InlineWarnBytes is the size above which an inlined file draws a warning.
const InlineWarnBytes = 2 << 20

// MediaFile is a file to copy next to the HTML (MediaCopy).
type MediaFile struct {
	Src string // absolute source path
	Rel string // slash path relative to the output directory
}

var extraTypes = map[string]string{
	".mp4": "video/mp4", ".m4v": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime", ".ogv": "video/ogg",
	".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif", ".woff2": "font/woff2", ".woff": "font/woff",
}

func cleanSrc(src string) string {
	return strings.SplitN(strings.SplitN(src, "#", 2)[0], "?", 2)[0]
}

func isExternal(src string) bool {
	return strings.HasPrefix(src, "//") || schemeRE.MatchString(src)
}

var schemeRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// dataURI reads a file and returns it as a data URI.
func dataURI(path string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(path))
	typ := extraTypes[ext]
	if typ == "" {
		typ = mime.TypeByExtension(ext)
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	if i := strings.IndexByte(typ, ';'); i > 0 {
		typ = typ[:i]
	}
	return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(raw), true
}

// media resolves a relative file reference for the page according to the
// media mode. ok is false when the file does not exist.
func (dr *deckRenderer) media(dir, src string, sc *slideCtx) (string, bool) {
	abs := filepath.Join(dir, filepath.FromSlash(cleanSrc(src)))
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return "", false
	}
	if dr.opt.Media != MediaInline {
		rel, err := filepath.Rel(filepath.Dir(dr.d.Path), abs)
		inside := err == nil && !strings.HasPrefix(rel, "..")
		if inside || dr.opt.Media == MediaCopy {
			if !inside {
				rel = filepath.Join("media", filepath.Base(abs))
			}
			rel = filepath.ToSlash(rel)
			if dr.opt.Media == MediaCopy {
				dr.files[abs] = rel
			}
			return (&url.URL{Path: rel}).String(), true
		}
	}
	if st.Size() > InlineWarnBytes {
		dr.warn(sc, "%s is %.1f MB, inlined as a data URI; build with --no-inline-media to copy it instead", src, float64(st.Size())/(1<<20))
	}
	return dataURI(abs)
}

// background renders a slide background: colours and gradients through the
// --si-slide-bg token, images and videos as a .si-bg layer with an overlay.
// It returns the extra classes, the inline style and the layer HTML.
func (dr *deckRenderer) background(s *deck.Slide, sc *slideCtx) (classes []string, style, layer string) {
	val, dir := s.Background, s.Dir
	if val == "" {
		val, dir = dr.d.Meta.Background, filepath.Dir(dr.d.Path)
	}
	if val == "" {
		return nil, "", ""
	}
	b, err := deck.ParseBackground(val)
	if err != nil || b.Kind == "none" {
		return nil, "", "" // the parser reported the error
	}
	classes = append(classes, "si-has-bg")
	if b.Kind == "colour" {
		style = "--si-slide-bg:" + b.Value
		if b.Overlay == "dark" || b.Overlay == "light" {
			classes = append(classes, "si-bg-"+b.Overlay)
		} else if tone := toneOf(b.Value); tone != "" && b.Overlay != "none" {
			classes = append(classes, "si-bg-"+tone)
		}
		return classes, style, ""
	}
	src, ok := dr.media(dir, b.Value, sc)
	if !ok {
		dr.problem(sc, "missing file: %s", b.Value)
		return nil, "", ""
	}
	var vars []string
	if b.Fit != "" {
		vars = append(vars, "--si-bg-fit:"+b.Fit)
	}
	if b.Position != "" {
		vars = append(vars, "--si-bg-position:"+b.Position)
	}
	if b.Opacity != "" {
		vars = append(vars, "--si-bg-opacity:"+b.Opacity)
	}
	if b.Blur != "" {
		vars = append(vars, "--si-bg-blur:calc("+b.Blur+" * .1cqi)")
	}
	overlay := b.Overlay
	if overlay == "" {
		overlay = "auto"
	}
	if overlay != "none" {
		tone := overlay
		if tone == "auto" {
			tone = "dark" // refined by the runtime from the image's brightness
		}
		classes = append(classes, "si-bg-"+tone)
	}
	styleAttr := ""
	if len(vars) > 0 {
		styleAttr = ` style="` + html.EscapeString(strings.Join(vars, ";")) + `"`
	}
	var mediaHTML string
	if b.Kind == "video" {
		poster := ""
		if b.Poster != "" {
			if p, ok := dr.media(dir, b.Poster, sc); ok {
				poster = ` poster="` + html.EscapeString(p) + `"`
			} else {
				dr.problem(sc, "missing file: %s", b.Poster)
			}
		}
		mediaHTML = fmt.Sprintf(`<video class="si-bg-media" src="%s"%s muted loop playsinline preload="auto"%s></video>`, html.EscapeString(src), poster, styleAttr)
	} else {
		mediaHTML = fmt.Sprintf(`<img class="si-bg-media" src="%s" alt=""%s>`, html.EscapeString(src), styleAttr)
	}
	layer = fmt.Sprintf(`<div class="si-bg" aria-hidden="true">%s<div class="si-bg-overlay" data-overlay="%s"></div></div>`+"\n", mediaHTML, overlay)
	return classes, "", layer
}

var hexRE = regexp.MustCompile(`#([0-9a-fA-F]{6}|[0-9a-fA-F]{3})\b`)

// toneOf guesses whether a colour or gradient is dark or light from its first
// hex colour, so text on it picks a readable palette.
func toneOf(v string) string {
	m := hexRE.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	h := m[1]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	n, _ := strconv.ParseUint(h, 16, 32)
	r, g, b := float64(n>>16&255), float64(n>>8&255), float64(n&255)
	if (0.2126*r+0.7152*g+0.0722*b)/255 < 0.5 {
		return "dark"
	}
	return "light"
}

// figure renders a paragraph that holds a single image as a figure: the
// image title becomes the caption, the image directive sizes and places it.
func figure(alt, dest, title string, a *deck.ImageAttrs) string {
	var b strings.Builder
	b.WriteString(`<figure class="si-image"`)
	if a != nil {
		if a.Align != "" {
			fmt.Fprintf(&b, ` data-align="%s"`, a.Align)
		}
		var vars []string
		if a.Width != "" {
			vars = append(vars, "--si-img-width:"+a.Width)
		}
		if a.Ratio != "" {
			vars = append(vars, "--si-img-ratio:"+a.Ratio)
		}
		if a.Fit != "" {
			vars = append(vars, "--si-img-fit:"+a.Fit)
		}
		if a.Position != "" {
			vars = append(vars, "--si-img-position:"+a.Position)
		}
		if a.Ratio != "" {
			b.WriteString(` data-ratio`)
		}
		if len(vars) > 0 {
			fmt.Fprintf(&b, ` style="%s"`, html.EscapeString(strings.Join(vars, ";")))
		}
	}
	fmt.Fprintf(&b, `><img src="%s" alt="%s">`, html.EscapeString(dest), html.EscapeString(alt))
	if title != "" {
		fmt.Fprintf(&b, `<figcaption class="si-caption">%s</figcaption>`, html.EscapeString(title))
	}
	b.WriteString(`</figure>`)
	return b.String()
}
