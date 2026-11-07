package studio

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStartersPassTheCheck(t *testing.T) {
	f := newFx(t)
	for kind, path := range map[string]string{
		"deck": "talk.md", "article": "post.md", "lab": "lab.md", "demo": "demo.md", "page": "page.md",
		"guide": "guide", "book": "book", "site": "site", "mindmap3d": "map3d.md", "codeshow": "show.md",
	} {
		code, m := f.do("POST", "starter", map[string]string{"kind": kind, "path": path})
		if code != 201 {
			t.Fatalf("%s: %d %v", kind, code, m)
		}
		open := m["open"].(string)
		if strings.Contains(f.read(open), "{{") {
			t.Errorf("%s: placeholders left in %s", kind, open)
		}
		_, c := f.do("GET", "check?deck="+open, nil)
		if ps := fmt.Sprint(c["problems"]); ps != "[]" {
			t.Errorf("%s starter has problems: %s", kind, ps)
		}
		if want := map[string]string{"page": "page", "site": "page", "book": "book"}[kind]; want != "" && c["kind"] != want {
			t.Errorf("%s: kind %v", kind, c["kind"]) // what the Studio shows it as
		}
	}
	if !strings.Contains(f.read("post.md"), "title: Post") {
		t.Error("the title comes from the file name")
	}
	if code, _ := f.do("POST", "starter", map[string]string{"kind": "site", "path": "site"}); code != 409 {
		t.Errorf("a starter never overwrites: %d", code)
	}
	if code, _ := f.do("POST", "starter", map[string]string{"kind": "novel", "path": "x.md"}); code != 400 {
		t.Errorf("unknown kind: %d", code)
	}
}

func TestStarterJoinsTheSiteMenu(t *testing.T) {
	f := siteFx(t)
	_, m := f.do("POST", "starter", map[string]string{"kind": "article", "path": "blog/queues.md", "title": "Queue depth"})
	if m["linked"] != "_site.md" || !strings.Contains(f.read("_site.md"), "- [Queue depth](blog/queues.md)\n") {
		t.Fatalf("not linked: %v\n%s", m, f.read("_site.md"))
	}
	f.do("POST", "starter", map[string]string{"kind": "page", "path": "docs/more.md", "title": "More"})
	if !strings.Contains(f.read("docs/_nav.md"), "- [More](more.md)") || strings.Contains(f.read("_site.md"), "more.md") {
		t.Errorf("a page in a sub-guide joins that guide only")
	}
	f.do("POST", "starter", map[string]string{"kind": "guide", "path": "handbook", "title": "Handbook"})
	if !strings.Contains(f.read("_site.md"), "- [Handbook](handbook/)") {
		t.Errorf("a new guide folder joins the site menu: %s", f.read("_site.md"))
	}
	_, c := f.do("GET", "checkall", nil)
	if ps := fmt.Sprint(c["problems"]); strings.Contains(ps, "queues") || strings.Contains(ps, "handbook") {
		t.Errorf("new pages check clean: %s", ps)
	}
}

// The start gallery's thumbnails: every starter renders as a page; nothing
// lands in the workspace.
func TestGalleryRendersEveryStarter(t *testing.T) {
	f := newFx(t)
	for _, kind := range StarterKinds {
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, httptest.NewRequest("GET", "/_si/studio/api/gallery/"+kind, nil))
		titled := starterFolder(kind) || strings.Contains(rec.Body.String(), "Your title here") // a folder shows its first page
		if rec.Code != 200 || !titled || !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			t.Errorf("%s: %d %.120s", kind, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest("GET", "/_si/studio/api/gallery/nope", nil))
	if rec.Code != 404 {
		t.Errorf("unknown kind: %d", rec.Code)
	}
	if entries, _ := os.ReadDir(f.root); len(entries) != 0 {
		t.Errorf("the gallery wrote into the workspace: %v", entries)
	}
}
