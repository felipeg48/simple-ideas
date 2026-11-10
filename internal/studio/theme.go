package studio

import (
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
	rel := "themes/" + slug + ".css"
	abs, ok := a.resolve(w, rel)
	if !ok {
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	text := themefile.New(strings.TrimSpace(strings.NewReplacer("*/", "", "\n", " ").Replace(req.Name)), assets.CanonicalTheme(from)).String()
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
