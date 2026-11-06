package render

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestParseFonts(t *testing.T) {
	specs, bad := parseFonts("heading=Fraunces:600, body=bunny:Inter:700;400, mono=fontsource:JetBrains Mono, heading=fonts/Brand.woff2:600, title=X, body=Inter:450")
	if len(specs) != 4 || len(bad) != 3 {
		t.Fatalf("specs %+v bad %v", specs, bad)
	}
	if s := specs[1]; s.provider != "bunny" || s.family != "Inter" || strings.Join(s.weights, ",") != "400,700" {
		t.Errorf("bunny entry: %+v", s)
	}
	if s := specs[2]; s.provider != "fontsource" || s.family != "JetBrains Mono" || strings.Join(s.weights, ",") != "400" {
		t.Errorf("mono default weight: %+v", s)
	}
	if s := specs[3]; s.file != "fonts/Brand.woff2" || s.weights[0] != "600" {
		t.Errorf("file entry: %+v", s)
	}
}

func TestFontsInlined(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/css":
			fmt.Fprint(w, "/* cyrillic */\n@font-face {\n  font-family: 'Testa';\n  font-weight: 600;\n  src: url(/cyr.woff2) format('woff2');\n  unicode-range: U+0400-045F;\n}\n/* latin */\n@font-face {\n  font-family: 'Testa';\n  font-weight: 600;\n  src: url(files/lat.woff2) format('woff2'), url(files/lat.woff) format('woff');\n  unicode-range: U+0000-00FF;\n}\n")
		case "/files/lat.woff2":
			w.Write([]byte("LATIN"))
		case "/cyr.woff2":
			w.Write([]byte("CYRIL"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	fontProviders["test"] = func(string, []string) []string { return []string{srv.URL + "/css"} }
	defer delete(fontProviders, "test")

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "brand.woff2"), []byte("BRAND"), 0o644)
	p := filepath.Join(dir, "d.md")
	os.WriteFile(p, []byte("---\nfonts: heading=test:Testa:600, mono=brand.woff2\n---\n# D\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{
		`src:url(data:font/woff2;base64,TEFUSU4=) format("woff2");`, // LATIN
		`--si-font-heading:"Testa", `,
		`--si-heading-weight:600;`,
		`font-family:"si-mono";src:url(data:font/woff2;base64,QlJBTkQ=)`, // BRAND
		`--si-font-mono:"si-mono", `,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	if strings.Contains(h, "cyr.woff2") || strings.Contains(h, "U+0400") {
		t.Error("a non-latin subset was kept")
	}
}

func TestFontsSubset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/css":
			fmt.Fprint(w, "/* cyrillic */\n@font-face {\n  font-family: 'Sub';\n  src: url(/cyr.woff2) format('woff2');\n  unicode-range: U+0400-045F;\n}\n/* latin */\n@font-face {\n  font-family: 'Sub';\n  src: url(/lat.woff2) format('woff2');\n  unicode-range: U+0000-00FF;\n}\n")
		default:
			w.Write([]byte(r.URL.Path))
		}
	}))
	defer srv.Close()
	fontProviders["sub"] = func(string, []string) []string { return []string{srv.URL + "/css"} }
	defer delete(fontProviders, "sub")
	p := filepath.Join(t.TempDir(), "d.md")
	os.WriteFile(p, []byte("---\nfonts: body=sub:Sub\nfonts-subset: cyrillic, latin\n---\n# D\n"), 0o644)
	d, _ := deck.Parse(p)
	h := string(Deck(d, Options{}).HTML)
	if !strings.Contains(h, "U+0400-045F") || !strings.Contains(h, "U+0000-00FF") {
		t.Error("fonts-subset: cyrillic, latin keeps both")
	}
}
