package render

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Fonts: free web fonts fetched at build time and inlined as @font-face data
// URIs, so a deck or page stays one self-contained file.
//
//	fonts: heading=Fraunces:600, body=Inter:400;700, mono=JetBrains Mono
//	fonts: body=bunny:Inter, heading=fontshare:Satoshi:700, mono=fontsource:Fira Code
//	fonts: heading=fonts/Brand.woff2
//
// A role (heading, body, mono) takes a family from Google Fonts (the default),
// Bunny Fonts, Fontshare or Fontsource, or a local font file. Stylesheets and
// files are cached for good (so --offline works after one online build); only
// the latin and latin-ext subsets of a split family are kept.

// fontRoles maps a role to the tokens it sets.
var fontRoles = map[string][]string{
	"heading": {"--si-font-heading"},
	"body":    {"--si-font-body", "--si-font-prose"},
	"mono":    {"--si-font-mono"},
}

var fontFallback = map[string]string{
	"heading": "ui-sans-serif, system-ui, sans-serif",
	"body":    "ui-sans-serif, system-ui, sans-serif",
	"mono":    `ui-monospace, "SF Mono", Menlo, Consolas, monospace`,
}

var fontDefaultWeights = map[string]string{"heading": "700", "body": "400;700", "mono": "400"}

// fontProviders turn a family and its weights into stylesheet URLs.
var fontProviders = map[string]func(family string, weights []string) []string{
	"google": func(f string, w []string) []string {
		return []string{"https://fonts.googleapis.com/css2?family=" + strings.ReplaceAll(url.QueryEscape(f), "%20", "+") + ":wght@" + strings.Join(w, ";") + "&display=swap"}
	},
	"bunny": func(f string, w []string) []string {
		return []string{"https://fonts.bunny.net/css?family=" + fontID(f) + ":" + strings.Join(w, ",") + "&display=swap"}
	},
	"fontshare": func(f string, w []string) []string {
		return []string{"https://api.fontshare.com/v2/css?f[]=" + fontID(f) + "@" + strings.Join(w, ",") + "&display=swap"}
	},
	"fontsource": func(f string, w []string) []string {
		var out []string
		for _, x := range w {
			out = append(out, "https://cdn.jsdelivr.net/npm/@fontsource/"+fontID(f)+"@5/"+x+".css")
		}
		return out
	},
}

// fontUA asks for woff2 (Google serves older formats to unknown agents).
var fontUA = map[string]string{"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"}

var (
	fontFaceRE  = regexp.MustCompile(`(?s)(?:/\*([^*]*)\*/\s*)?@font-face\s*\{(.*?)\}`)
	fontSrcRE   = regexp.MustCompile(`(?s)src:[^;]*;?`)
	fontURLRE   = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)\s*format\(\s*['"]([\w-]+)['"]\s*\)`)
	fontFamRE   = regexp.MustCompile(`font-family:\s*['"]?([^;'"]+)`)
	fontMIME    = map[string]string{"woff2": "font/woff2", "woff": "font/woff", "truetype": "font/ttf", "opentype": "font/otf"}
	fontFileExt = map[string]string{".woff2": "woff2", ".woff": "woff", ".ttf": "truetype", ".otf": "opentype"}
)

func fontID(family string) string { return strings.ToLower(strings.Join(strings.Fields(family), "-")) }

// fontSpec is one role=… entry.
type fontSpec struct {
	role, provider, family, file string
	weights                      []string
}

// parseFonts splits a fonts: value into its entries.
func parseFonts(val string) ([]fontSpec, []string) {
	var out []fontSpec
	var bad []string
	for _, part := range strings.Split(val, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		role, rest, ok := strings.Cut(part, "=")
		role, rest = strings.ToLower(strings.TrimSpace(role)), strings.TrimSpace(rest)
		if !ok || fontRoles[role] == nil || rest == "" {
			bad = append(bad, fmt.Sprintf("%q: want role=Family[:weights] with a role of heading, body or mono", part))
			continue
		}
		sp := fontSpec{role: role, provider: "google"}
		file, w, _ := strings.Cut(rest, ":") // fonts/Brand.woff2 or fonts/Brand.woff2:600
		if fontFileExt[strings.ToLower(filepath.Ext(file))] != "" {
			sp.file = file
			if w != "" {
				sp.weights = []string{strings.TrimSpace(w)}
			}
			out = append(out, sp)
			continue
		}
		segs := strings.Split(rest, ":")
		if p := strings.ToLower(strings.TrimSpace(segs[0])); fontProviders[p] != nil && len(segs) > 1 {
			sp.provider, segs = p, segs[1:]
		}
		sp.family = strings.TrimSpace(segs[0])
		w = fontDefaultWeights[role]
		if len(segs) > 1 {
			w = segs[1]
		}
		for _, x := range strings.FieldsFunc(w, func(r rune) bool { return r == ';' || r == ' ' || r == '+' }) {
			if n, err := strconv.Atoi(x); err != nil || n < 100 || n > 900 || n%100 != 0 {
				bad = append(bad, fmt.Sprintf("%s: weight %q is not 100…900", sp.family, x))
				continue
			}
			sp.weights = append(sp.weights, x)
		}
		sort.Slice(sp.weights, func(i, j int) bool {
			a, _ := strconv.Atoi(sp.weights[i])
			b, _ := strconv.Atoi(sp.weights[j])
			return a < b
		})
		if sp.family == "" || len(sp.weights) == 0 {
			bad = append(bad, fmt.Sprintf("%q: needs a family and a weight", part))
			continue
		}
		out = append(out, sp)
	}
	return out, bad
}

// resolveFonts builds dr.fontsCSS from the deck's fonts: (or si.conf's).
func (dr *deckRenderer) resolveFonts() {
	val := dr.d.Meta.Fonts
	if val == "" && dr.opt.Config != nil {
		val = dr.opt.Config.Fonts
	}
	if strings.TrimSpace(val) == "" || val == "off" {
		return
	}
	specs, bad := parseFonts(val)
	for _, m := range bad {
		dr.problem(nil, "fonts: %s", m)
	}
	var faces, decls strings.Builder
	for _, sp := range specs {
		family, css, err := dr.fontFaces(sp)
		if err != nil {
			dr.problem(nil, "fonts: %v", err)
			continue
		}
		faces.WriteString(css)
		for _, tok := range fontRoles[sp.role] {
			fmt.Fprintf(&decls, "%s:\"%s\", %s;", tok, family, fontFallback[sp.role])
		}
		if sp.role == "heading" && len(sp.weights) == 1 {
			fmt.Fprintf(&decls, "--si-heading-weight:%s;", sp.weights[0])
		}
	}
	if decls.Len() > 0 {
		dr.fontsCSS = "@layer si.theme{\n" + faces.String() + ":root,:root:not([data-si-mode]),:root[data-si-mode=\"dark\"]{" + decls.String() + "}\n}\n"
	}
}

// fontFaces returns the family name to use and its @font-face rules, every
// font file inlined.
func (dr *deckRenderer) fontFaces(sp fontSpec) (string, string, error) {
	if sp.file != "" {
		p := sp.file
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(dr.d.Path), p)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", "", fmt.Errorf("cannot read %s", sp.file)
		}
		format := fontFileExt[strings.ToLower(filepath.Ext(p))]
		family, weight := "si-"+sp.role, "100 900" // one file serves every weight unless it names one
		if len(sp.weights) == 1 {
			weight = sp.weights[0]
		}
		return family, fmt.Sprintf("@font-face{font-family:\"%s\";src:url(data:%s;base64,%s) format(\"%s\");font-weight:%s;font-display:swap}\n",
			family, fontMIME[format], base64.StdEncoding.EncodeToString(raw), format, weight), nil
	}
	var b strings.Builder
	family := sp.family
	for _, sheet := range fontProviders[sp.provider](sp.family, sp.weights) {
		css, err := deck.FetchAsset(sheet, fontUA)
		if err != nil {
			if strings.Contains(err.Error(), "HTTP 4") {
				return "", "", fmt.Errorf("%s has no family %q with weights %s (check the spelling and the weights)", sp.provider, sp.family, strings.Join(sp.weights, ";"))
			}
			return "", "", fmt.Errorf("cannot fetch %s from %s: %v", sp.family, sp.provider, err)
		}
		base, _ := url.Parse(sheet)
		n := 0
		for _, m := range fontFaceRE.FindAllStringSubmatch(string(css), -1) {
			comment, body := strings.ToLower(m[1]), m[2]
			if strings.Contains(body, "unicode-range") && !strings.Contains(comment, "latin") {
				continue // a non-latin subset of a split family
			}
			src := fontURLRE.FindAllStringSubmatch(body, -1)
			if len(src) == 0 {
				continue
			}
			pick := src[0]
			for _, s := range src {
				if s[2] == "woff2" {
					pick = s
				}
			}
			ref, err := url.Parse(pick[1])
			if err != nil {
				continue
			}
			file := base.ResolveReference(ref).String()
			raw, err := deck.FetchAsset(file, fontUA)
			if err != nil {
				return "", "", fmt.Errorf("cannot fetch a file of %s: %v", sp.family, err)
			}
			if f := fontFamRE.FindStringSubmatch(body); f != nil {
				family = strings.TrimSpace(f[1])
			}
			data := fmt.Sprintf(`src:url(data:%s;base64,%s) format("%s");`, fontMIME[pick[2]], base64.StdEncoding.EncodeToString(raw), pick[2])
			b.WriteString("@font-face{" + strings.TrimSpace(fontSrcRE.ReplaceAllLiteralString(body, data)) + "}\n")
			n++
		}
		if n == 0 {
			return "", "", fmt.Errorf("%s has no font %s on %s (check the name and weights)", sheet, sp.family, sp.provider)
		}
	}
	return family, b.String(), nil
}
