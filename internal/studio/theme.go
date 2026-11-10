package studio

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/themefile"
)

// The Theme view: a .css theme file as token controls beside a preview.
// POST theme applies one op to the editor's text through internal/themefile
// and answers with the new text, the file's tokens, the values a page sees
// (the extended built-in underneath), the checks and the specimen deck drawn
// with it; the UI puts the text in the editor as one undoable change and
// saves it the usual way. theme/new creates themes/<name>.css; theme/use
// writes theme: into the folder's si.conf.

type themeOp struct {
	Token string `json:"token"`
	Mode  string `json:"mode"` // light (or "") or dark
	Value string `json:"value"`
	Field string `json:"field"` // "name" or "extends" instead of a token
}

type themeInfo struct {
	Name  string            `json:"name"`
	Label string            `json:"label"`
	About string            `json:"about"`
	Light map[string]string `json:"light"`
	Dark  map[string]string `json:"dark"`
}

// themeSpec: the token groups, every theme token with its kind, and the
// listed built-ins' values (for Start and the placeholders).
func (a *api) themeSpec(w http.ResponseWriter, r *http.Request) {
	var toks []assets.TokenMeta
	for _, m := range assets.AllMeta() {
		if m.Group != "Element" {
			toks = append(toks, m)
		}
	}
	var themes []themeInfo
	for _, n := range assets.ThemeNames() {
		if assets.UnlistedThemes[n] {
			continue
		}
		l, d, _ := assets.ThemeTokens(n)
		themes = append(themes, themeInfo{n, assets.Themes[n].Label, assets.Themes[n].About, l, d})
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": assets.TokenGroups[:len(assets.TokenGroups)-1], "tokens": toks, "themes": themes})
}

func (a *api) themeEdit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string   `json:"path"`
		Text string   `json:"text"`
		Op   *themeOp `json:"op"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	f := themefile.Parse(req.Text)
	text := req.Text
	if op := req.Op; op != nil {
		var err error
		switch op.Field {
		case "name":
			f.Name = strings.TrimSpace(strings.NewReplacer("*/", "", "\n", " ").Replace(op.Value))
		case "extends":
			if _, _, known := assets.ThemeTokens(op.Value); op.Value != "" && !known {
				err = errors.New("no built-in theme " + op.Value)
			}
			f.Extends = op.Value
		case "":
			err = f.Set(op.Token, op.Mode, op.Value)
		default:
			err = errors.New("field is name or extends")
		}
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		text = f.String()
	}
	light, dark := f.Effective()
	issues := f.Check(text)
	if issues == nil {
		issues = []themefile.Issue{}
	}
	html := ""
	css, err := render.ThemeFileLayer(render.InlineURLs(filepath.Dir(abs), text))
	if err == nil {
		if d, perr := deck.ParseSource(filepath.Join(a.ws.Root, ".si-specimen.md"), assets.Specimen(), deck.Meta{Type: "deck"}); perr == nil {
			html = string(render.Deck(d, render.Options{Media: render.MediaInline, ThemeCSS: css}).HTML)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "name": f.Name, "extends": f.Extends, "light": f.Light, "dark": f.Dark,
		"effLight": light, "effDark": dark, "issues": issues, "extra": f.Extra != "", "html": html})
}

var themeSlugRE = regexp.MustCompile(`[^a-z0-9]+`)

// themeNew creates themes/<name>.css extending a built-in (studio when from
// is empty); it answers with the new path.
func (a *api) themeNew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		From string `json:"from"`
	}
	if !decode(w, r, &req) {
		return
	}
	slug := strings.Trim(themeSlugRE.ReplaceAllString(strings.ToLower(req.Name), "-"), "-")
	if slug == "" {
		fail(w, http.StatusBadRequest, "a theme needs a name")
		return
	}
	from := req.From
	if from == "" {
		from = "studio"
	}
	if _, _, ok := assets.ThemeTokens(from); !ok {
		fail(w, http.StatusBadRequest, "no built-in theme "+from)
		return
	}
	a.createTheme(w, "themes/"+slug+".css", themefile.New(strings.TrimSpace(strings.NewReplacer("*/", "", "\n", " ").Replace(req.Name)), assets.CanonicalTheme(from)).String())
}

// themeLogo makes a theme from a logo: an image in the workspace (path) or
// one sent from the browser (data, base64), at most themefile.MaxLogo. It
// answers with the logo's colours and the theme's light and dark values;
// with create it also writes themes/<name>.css and answers with its path.
func (a *api) themeLogo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path   string `json:"path"`
		Data   string `json:"data"`
		Name   string `json:"name"`
		Accent string `json:"accent"`
		Create bool   `json:"create"`
	}
	if !decode(w, r, &req) {
		return
	}
	var raw []byte
	switch {
	case req.Data != "":
		if base64.StdEncoding.DecodedLen(len(req.Data)) > themefile.MaxLogo+3 {
			fail(w, http.StatusRequestEntityTooLarge, "a logo is at most 8 MiB")
			return
		}
		var err error
		if raw, err = base64.StdEncoding.DecodeString(req.Data); err != nil {
			fail(w, http.StatusBadRequest, "data is not base64")
			return
		}
	case req.Path != "":
		abs, ok := a.resolve(w, req.Path)
		if !ok {
			return
		}
		fi, err := os.Stat(abs)
		if err != nil || fi.IsDir() {
			fail(w, http.StatusNotFound, "not found")
			return
		}
		if fi.Size() > themefile.MaxLogo {
			fail(w, http.StatusRequestEntityTooLarge, "a logo is at most 8 MiB")
			return
		}
		if raw, err = os.ReadFile(abs); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		if req.Name == "" {
			req.Name = themefile.LogoName(req.Path)
		}
	default:
		fail(w, http.StatusBadRequest, "path or data required")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Brand"
	}
	pal, err := themefile.LogoPalette(raw)
	if err == nil {
		var f *themefile.File
		if f, err = themefile.FromLogo(req.Name, pal, req.Accent); err == nil {
			rel := "themes/" + themefile.Slug(f.Name) + ".css"
			if req.Create {
				a.createTheme(w, rel, f.String())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"palette": pal, "name": f.Name, "path": rel, "light": f.Light, "dark": f.Dark})
			return
		}
	}
	fail(w, http.StatusUnprocessableEntity, err.Error())
}

// createTheme writes a new theme file (never over one) and answers 201 with its path.
func (a *api) createTheme(w http.ResponseWriter, rel, text string) {
	abs, ok := a.resolve(w, rel)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	fh, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		fail(w, http.StatusConflict, rel+" exists")
		return
	}
	if err == nil {
		_, err = fh.WriteString(text)
		if cerr := fh.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": rel})
}

// themeUse writes theme: <path> into the workspace's si.conf.
func (a *api) themeUse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	if !strings.HasSuffix(strings.ToLower(abs), ".css") {
		fail(w, http.StatusBadRequest, "a theme is a .css file")
		return
	}
	rel := filepath.ToSlash(a.ws.Rel(abs))
	if err := config.SetKey(filepath.Join(a.ws.Root, "si.conf"), "theme", rel); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conf": "si.conf", "theme": rel})
}
