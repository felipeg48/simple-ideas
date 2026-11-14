package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// manifest is the part of a SCORM 1.2 imsmanifest.xml the tests check.
type manifest struct {
	XMLName xml.Name `xml:"manifest"`
	Ident   string   `xml:"identifier,attr"`
	Schema  string   `xml:"metadata>schema"`
	Version string   `xml:"metadata>schemaversion"`
	OrgsEl  struct {
		Default string `xml:"default,attr"`
		Orgs    []mOrg `xml:"organization"`
	} `xml:"organizations"`
	Resource []struct {
		ID    string `xml:"identifier,attr"`
		Type  string `xml:"type,attr"`
		SCORM string `xml:"http://www.adlnet.org/xsd/adlcp_rootv1p2 scormtype,attr"`
		Href  string `xml:"href,attr"`
		Files []struct {
			Href string `xml:"href,attr"`
		} `xml:"file"`
		Deps []struct {
			Ref string `xml:"identifierref,attr"`
		} `xml:"dependency"`
	} `xml:"resources>resource"`
}

type mOrg struct {
	ID    string  `xml:"identifier,attr"`
	Title string  `xml:"title"`
	Items []mItem `xml:"item"`
}

type mItem struct {
	ID      string  `xml:"identifier,attr"`
	Ref     string  `xml:"identifierref,attr"`
	Title   string  `xml:"title"`
	Mastery string  `xml:"http://www.adlnet.org/xsd/adlcp_rootv1p2 masteryscore"`
	Items   []mItem `xml:"item"`
}

// checkManifest opens a SCORM zip and checks its manifest: well-formed,
// SCORM 1.2, a default organization, every item's resource there, every
// file a resource names in the zip and every file of the zip named. It
// returns the manifest and the SCO launch files.
func checkManifest(t *testing.T, zipPath string) (manifest, []string) {
	t.Helper()
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	inZip := map[string]bool{}
	var raw []byte
	for _, f := range zr.File {
		inZip[f.Name] = true
		if f.Name == "imsmanifest.xml" {
			r, _ := f.Open()
			var b bytes.Buffer
			b.ReadFrom(r)
			r.Close()
			raw = b.Bytes()
		}
	}
	if raw == nil {
		t.Fatal("no imsmanifest.xml at the root of the zip")
	}
	var m manifest
	if err := xml.Unmarshal(raw, &m); err != nil {
		t.Fatalf("imsmanifest.xml is not well-formed: %v", err)
	}
	if m.XMLName.Space != "http://www.imsproject.org/xsd/imscp_rootv1p1p2" || m.Schema != "ADL SCORM" || m.Version != "1.2" || m.Ident == "" {
		t.Errorf("not a SCORM 1.2 manifest: %+v", m)
	}
	if len(m.OrgsEl.Orgs) != 1 || m.OrgsEl.Orgs[0].ID != m.OrgsEl.Default || m.OrgsEl.Orgs[0].Title == "" {
		t.Errorf("organizations: default %q, %+v", m.OrgsEl.Default, m.OrgsEl.Orgs)
	}
	res := map[string]bool{}
	named := map[string]bool{"imsmanifest.xml": true}
	var scos []string
	for _, r := range m.Resource {
		res[r.ID] = true
		if r.Type != "webcontent" || (r.SCORM != "sco" && r.SCORM != "asset") {
			t.Errorf("resource %s: type %q scormtype %q", r.ID, r.Type, r.SCORM)
		}
		if r.SCORM == "sco" {
			if !inZip[r.Href] {
				t.Errorf("SCO %s launches %s, not in the zip", r.ID, r.Href)
			}
			scos = append(scos, r.Href)
		}
		for _, f := range r.Files {
			named[f.Href] = true
			if !inZip[f.Href] {
				t.Errorf("resource %s names %s, not in the zip", r.ID, f.Href)
			}
		}
	}
	for _, r := range m.Resource {
		for _, d := range r.Deps {
			if !res[d.Ref] {
				t.Errorf("resource %s depends on %s, which is missing", r.ID, d.Ref)
			}
		}
	}
	var walk func([]mItem)
	walk = func(its []mItem) {
		for _, it := range its {
			if it.Ref != "" && !res[it.Ref] {
				t.Errorf("item %s points at %s, which is missing", it.ID, it.Ref)
			}
			walk(it.Items)
		}
	}
	for _, o := range m.OrgsEl.Orgs {
		walk(o.Items)
	}
	for f := range inZip {
		if !named[f] {
			t.Errorf("%s is in the zip but no resource names it", f)
		}
	}
	return m, scos
}

func TestBuildCourseSCORM(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	out := filepath.Join(tmp, "html")
	if code := run([]string{"build", src, "-o", out, "--scorm", "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build --scorm: %d\n%s", code, stderr.String())
	}
	m, scos := checkManifest(t, filepath.Join(tmp, "web101-scorm12.zip"))
	g, _ := deck.LoadGuide(src)
	learner := g.ForAudience(false)
	want := 0
	for _, it := range learner.Lessons() {
		if it.Kind != "demo" {
			want++
		}
	}
	if len(scos) != want {
		t.Errorf("one SCO per lesson that counts: %d SCOs, want %d (%v)", len(scos), want, scos)
	}
	if !strings.Contains(fmtItems(m.OrgsEl.Orgs[0].Items), "masteryscore") {
		t.Error("a quiz with pass: gives its mastery score")
	}

	single := filepath.Join(tmp, "one")
	if code := run([]string{"build", src, "-o", single, "--scorm=single", "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build --scorm=single: %d\n%s", code, stderr.String())
	}
	os.Rename(filepath.Join(tmp, "web101-scorm12.zip"), filepath.Join(tmp, "one.zip"))
	if _, scos := checkManifest(t, filepath.Join(tmp, "one.zip")); len(scos) != 1 || scos[0] != "scorm.html" {
		t.Errorf("--scorm=single: one SCO launching scorm.html, got %v", scos)
	}
	if code := run([]string{"build", src, "-o", filepath.Join(tmp, "x"), "--scorm", "--instructor", "--no-film"}, &stdout, &stderr); code == 0 {
		t.Error("--scorm --instructor is refused (an LMS gets the learner build)")
	}
	if code := run([]string{"build", src, "-o", filepath.Join(tmp, "y"), "--scorm=many"}, &stdout, &stderr); code == 0 {
		t.Error("--scorm=many is a usage error")
	}
}

func fmtItems(its []mItem) string {
	var b strings.Builder
	for _, it := range its {
		if it.Mastery != "" {
			b.WriteString("masteryscore ")
		}
		b.WriteString(fmtItems(it.Items))
	}
	return b.String()
}

// TestSCORMRuntime: a quiz lesson of a --scorm build in a fake LMS (a page
// with window.API, the lesson in a frame, served over HTTP): course.js opens
// the session, the right answers (checked by doc.js against the hashes) pass
// the quiz, and the score, status, suspend data, commit and finish reach the
// LMS.
func TestSCORMRuntime(t *testing.T) {
	needChrome(t)
	tmp := t.TempDir()
	src := filepath.Join(tmp, "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	out := filepath.Join(tmp, "html")
	if code := run([]string{"build", src, "-o", out, "--scorm", "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build: %s", stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(src, "02-html", "quiz.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, qs, _ := deck.ParseQuiz(string(raw))
	var answers [][]string
	for _, q := range qs {
		var ids []string
		for j, o := range q.Options {
			if o.OK {
				ids = append(ids, string(rune('0'+j)))
			}
		}
		answers = append(answers, ids)
	}
	aj, _ := json.Marshal(answers)
	lms := `<!doctype html><meta charset="utf-8"><pre id="log"></pre><iframe id="f" src="02-html/quiz.html"></iframe><script>
var log = [], st = {'cmi.core.lesson_status': 'not attempted', 'cmi.suspend_data': ''};
function say(s) { log.push(s); document.getElementById('log').textContent = log.join('\n'); }
window.API = {
  LMSInitialize: function () { say('LMSInitialize'); return 'true'; },
  LMSGetValue: function (k) { return st[k] || ''; },
  LMSSetValue: function (k, v) { st[k] = v; say(k + '=' + v); return 'true'; },
  LMSCommit: function () { say('LMSCommit'); return 'true'; },
  LMSFinish: function () { say('LMSFinish'); return 'true'; },
  LMSGetLastError: function () { return '0'; }
};
var answers = ` + string(aj) + `;
var f = document.getElementById('f');
f.addEventListener('load', function once() {
  f.removeEventListener('load', once);
  setTimeout(function () {
    var d = f.contentDocument, form = d.querySelector('form.si-quiz');
    d.querySelectorAll('.si-quiz-q').forEach(function (q, i) {
      q.querySelectorAll('input').forEach(function (inp) { inp.checked = answers[i].indexOf(inp.value) >= 0; });
    });
    form.requestSubmit();
    say('form=' + form.className);
    setTimeout(function () { f.src = 'about:blank'; }, 200);
  }, 300);
});
</script>`
	os.WriteFile(filepath.Join(out, "lms.html"), []byte(lms), 0o644)
	srv := httptest.NewServer(http.FileServer(http.Dir(out)))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dom, err := exec.CommandContext(ctx, findChrome(), "--headless=new", "--use-mock-keychain", "--disable-gpu",
		"--virtual-time-budget=5000", "--dump-dom", srv.URL+"/lms.html").Output()
	if err != nil {
		t.Fatalf("chrome: %v", err)
	}
	got := string(dom)
	if strings.Contains(got, "lesson_status=failed") {
		t.Error("a passed quiz never reports failed on the way")
	}
	for _, want := range []string{"LMSInitialize", "cmi.core.lesson_status=incomplete", "is-passed", "cmi.core.score.raw=100",
		"cmi.core.lesson_status=passed", `cmi.suspend_data={"d":["02-html/quiz.html"]`, "LMSCommit", "LMSFinish"} {
		if !strings.Contains(got, strings.ReplaceAll(want, `"`, "&quot;")) && !strings.Contains(got, want) {
			t.Errorf("the LMS log lacks %q:\n%s", want, got)
		}
	}
}
