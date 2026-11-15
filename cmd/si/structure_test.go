package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var updateListing = flag.Bool("update", false, "rewrite testdata/build-listing.golden")

// listing is every file under dir with its SHA-256, one per line, sorted.
func listing(t *testing.T, dir string) string {
	t.Helper()
	var lines []string
	filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		raw, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		lines = append(lines, fmt.Sprintf("%x  %s", sha256.Sum256(raw), filepath.ToSlash(rel)))
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

// TestBuildDefaultUnchanged: without a structure: block in si.conf a build
// is byte for byte what it was before project structures existed.
func TestBuildDefaultUnchanged(t *testing.T) {
	got := ""
	for _, src := range []string{"examples", "examples/guide", "examples/site"} {
		out := t.TempDir()
		var stdout, stderr bytes.Buffer
		if code := run([]string{"build", "-o", out, filepath.Join("..", "..", src)}, &stdout, &stderr); code != 0 {
			t.Fatalf("build %s exit %d: %s", src, code, stderr.String())
		}
		got += "# " + src + "\n" + listing(t, out)
	}
	golden := filepath.Join("testdata", "build-listing.golden")
	if *updateListing {
		os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("the default build changed (go test ./cmd/si -run TestBuildDefaultUnchanged -update after an intended change):\n%s", got)
	}
}

// copyTree copies the files under src into dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	filepath.WalkDir(src, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		r, _ := filepath.Rel(src, p)
		if err := copyFile(p, filepath.Join(dst, r)); err != nil {
			t.Fatal(err)
		}
		return nil
	})
}

var attrRE = regexp.MustCompile(`\s(?:href|src|poster)="([^"]*)"`)

// TestExamplesStructure builds examples/structure (the founder's map): every
// kind into its folder, deck PDFs twice, links that resolve, the zips.
func TestExamplesStructure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rabbitmq")
	ex := filepath.Join("..", "..", "examples", "structure")
	copyTree(t, filepath.Join(ex, "md"), filepath.Join(dir, "md"))
	copyFile(filepath.Join(ex, "si.conf"), filepath.Join(dir, "si.conf"))
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", dir}, &stdout, &stderr); code != 0 || strings.Contains(stderr.String(), "warning") {
		t.Fatalf("check %d: %s", code, stderr.String())
	}
	if code := run([]string{"build", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("build exit %d: %s", code, stderr.String())
	}
	want := []string{"index.html", "slides/intro.html", "labs/lab-queues.html", "html/index.html", "html/brokers.html",
		"html/images/broker.png", "html/images/exchange.png", "html/images/publish.poster.jpg", "html/animations/queue.svg", "videos/publish.mp4",
		"dist/rabbitmq-deck.zip", "dist/rabbitmq-html.zip"}
	chrome := findChrome() != ""
	if chrome {
		want = append(want, "slides/pdf/intro.pdf", "html/pdfs/intro.pdf")
	}
	var got []string
	filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		r, _ := filepath.Rel(dir, p)
		if r = filepath.ToSlash(r); err == nil && !e.IsDir() && !strings.HasPrefix(r, "md/") && r != "si.conf" {
			got = append(got, r)
		}
		return nil
	})
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("tree\n got %v\nwant %v", got, want)
	}
	// every local link of every built page resolves
	for _, f := range got {
		if !strings.HasSuffix(f, ".html") {
			continue
		}
		raw, _ := os.ReadFile(filepath.Join(dir, f))
		for _, m := range attrRE.FindAllStringSubmatch(string(raw), -1) {
			l := m[1]
			if i := strings.IndexAny(l, "#?"); i >= 0 {
				l = l[:i]
			}
			if l == "" || strings.Contains(l, ":") || strings.ContainsAny(l, "'+ ") {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, filepath.Dir(f), l)); err != nil {
				t.Errorf("%s links %s: missing", f, m[1])
			}
		}
	}
	page, _ := os.ReadFile(filepath.Join(dir, "slides", "intro.html"))
	for _, l := range []string{`src="../html/images/exchange.png"`, `src="../videos/publish.mp4"`, `poster="../html/images/publish.poster.jpg"`, `"href":"../labs/lab-queues.html"`} {
		if !bytes.Contains(page, []byte(l)) {
			t.Errorf("slides/intro.html lacks %s", l)
		}
	}
	zips := map[string]string{
		"dist/rabbitmq-deck.zip": "html/brokers.html html/images/broker.png html/images/exchange.png html/images/publish.poster.jpg html/index.html index.html labs/lab-queues.html slides/intro.html videos/publish.mp4", // with the pages the deck links to, and theirs
		"dist/rabbitmq-html.zip": "animations/queue.svg brokers.html images/broker.png images/exchange.png images/publish.poster.jpg index.html",
	}
	if chrome {
		zips["dist/rabbitmq-deck.zip"] = "html/brokers.html html/images/broker.png html/images/exchange.png html/images/publish.poster.jpg html/index.html index.html labs/lab-queues.html slides/intro.html slides/pdf/intro.pdf videos/publish.mp4"
		zips["dist/rabbitmq-html.zip"] = "animations/queue.svg brokers.html images/broker.png images/exchange.png images/publish.poster.jpg index.html pdfs/intro.pdf"
	}
	for z, w := range zips {
		r, err := zip.OpenReader(filepath.Join(dir, z))
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, f := range r.File {
			names = append(names, f.Name)
		}
		r.Close()
		sort.Strings(names)
		if strings.Join(names, " ") != w {
			t.Errorf("%s holds %v, want %s", z, names, w)
		}
		if z == "dist/rabbitmq-deck.zip" { // every local link of every zipped page resolves inside the zip
			for _, n := range names {
				if !strings.HasSuffix(n, ".html") {
					continue
				}
				for _, ref := range localRefs(dir, n) {
					if !strings.Contains(" "+w+" ", " "+ref+" ") {
						t.Errorf("%s lacks %s, linked from %s", z, ref, n)
					}
				}
			}
		}
	}
}

// TestStructureRoute: where each kind of staged file goes.
func TestStructureRoute(t *testing.T) {
	src, stage := t.TempDir(), t.TempDir()
	conf := "structure:\n  md/ markdown\n  slides/ deck\n  more/ deck\n  pdf/ deck:pdf\n  labs/ lab\n  web/ site\n    img/ image\n"
	files := map[string]string{
		"si.conf":           conf,
		"md/talk.md":        "# Talk\n",
		"md/lab.md":         "---\ntype: lab\n---\n\n# Lab\n\n## One\n",
		"md/web/_site.md":   "---\ntitle: Web\n---\n\n- [Home](index.md)\n",
		"md/web/index.md":   "# Home\n",
		"md/timeline.md":    "---\ntype: timeline\n---\n\n# T\n",
		"md/assets/x.png":   "png",
		"md/notes-talk.txt": "",
	}
	for f, c := range files {
		p := filepath.Join(src, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(c), 0o644)
	}
	staged := []string{"index.html", "talk.html", "pdfs/talk.pdf", "handouts/talk.pdf", "talk.mp4", "lab.html",
		"web/index.html", "web/images/x.png", "_si/three.min.js", "timeline.html", "timeline.png", "other.txt"}
	for _, f := range staged {
		p := filepath.Join(stage, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		body := "x"
		if f == "talk.html" {
			body = `<script src="_si/three.min.js"></script><a href="lab.html">`
		}
		os.WriteFile(p, []byte(body), 0o644)
	}
	st := projectStructure(src, &options{})
	col := &collected{files: map[string]bool{filepath.Join(stage, "web", "images", "x.png"): true}}
	items, err := route(stage, src, st, &options{mp4: true}, col)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, it := range items {
		got[it.rel] = strings.Join(it.dests, " ")
	}
	want := map[string]string{
		"index.html":        "index.html",
		"talk.html":         "slides/talk.html more/talk.html",
		"pdfs/talk.pdf":     "pdf/talk.pdf",
		"handouts/talk.pdf": "",         // not in the map, not asked for: dropped
		"talk.mp4":          "talk.mp4", // asked for with --mp4, no folder: today's place
		"lab.html":          "labs/lab.html",
		"web/index.html":    "web/index.html", // the one site fills its folder
		"web/images/x.png":  "web/img/x.png",
		"_si/three.min.js":  "more/_si/three.min.js slides/_si/three.min.js",
		"timeline.html":     "timeline.html", // no timeline folder: the root
		"timeline.png":      "",
		"other.txt":         "other.txt",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s -> %q, want %q", k, got[k], w)
		}
	}
	ws, _ := structureWarnings(src, st)
	warns := strings.Join(ws, "\n")
	for _, w := range []string{"md/timeline.md: a timeline, and the structure has no timeline folder"} {
		if !strings.Contains(warns, w) {
			t.Errorf("warnings lack %q:\n%s", w, warns)
		}
	}
}

// TestStructureFromInside: si build of a page or a guide folder inside a
// project with a structure map builds the project with its map; a page in a
// folder the map does not name, or outside the project, builds as before.
func TestStructureFromInside(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "p")
	os.MkdirAll(filepath.Join(dir, "md", "guide"), 0o755)
	os.MkdirAll(filepath.Join(dir, "notes"), 0o755)
	os.WriteFile(filepath.Join(dir, "si.conf"), []byte("structure:\n  md/ markdown\n  slides/ deck\n  docs/ guide\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "md", "talk.md"), []byte("# Talk\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "md", "guide", "_nav.md"), []byte("---\ntitle: G\n---\n\n- [One](one.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "md", "guide", "one.md"), []byte("---\ntype: guide\n---\n# One\n\nText.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes", "n.md"), []byte("# Note\n"), 0o644)
	for _, src := range []string{filepath.Join(dir, "md", "talk.md"), filepath.Join(dir, "md", "guide")} {
		os.RemoveAll(filepath.Join(dir, "slides"))
		os.RemoveAll(filepath.Join(dir, "docs"))
		var stdout, stderr bytes.Buffer
		if code := run([]string{"build", src}, &stdout, &stderr); code != 0 {
			t.Fatalf("build %s: %d %s", src, code, stderr.String())
		}
		for _, f := range []string{"slides/talk.html", "docs/one.html"} {
			if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
				t.Errorf("si build %s: the map puts %s", src, f)
			}
		}
	}
	if enclosingProject(filepath.Join(dir, "notes", "n.md"), &options{}) != "" {
		t.Error("notes/ is not in the map: not a source of the project")
	}
	if enclosingProject(dir, &options{}) != "" {
		t.Error("nothing above the project")
	}
}

func TestStructureCheckWarnings(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "si.conf"), []byte("structure:\n  md/ markdown\n  slides/ deck\n  labs/ lab\n  dist/ labs/:zip, course:scorm\n"), 0o644)
	os.MkdirAll(filepath.Join(src, "slides"), 0o755)
	os.MkdirAll(filepath.Join(src, "md"), 0o755)
	os.WriteFile(filepath.Join(src, "md", "talk.md"), []byte("# Talk\n"), 0o644)
	os.WriteFile(filepath.Join(src, "slides", "lab.md"), []byte("---\ntype: lab\n---\n\n# Lab\n\n## One\n"), 0o644)
	var stdout, stderr bytes.Buffer
	run([]string{"check", src}, &stdout, &stderr)
	for _, w := range []string{
		"slides/lab.md: a lab source in slides/, the folder for deck; sources go in md/",
		"note: si.conf line 5: structure: course:scorm: nothing to bundle yet, the project has no course",
	} {
		if !strings.Contains(stderr.String(), w) {
			t.Errorf("check lacks %q:\n%s", w, stderr.String())
		}
	}
	os.Remove(filepath.Join(src, "slides", "lab.md"))
	stderr.Reset()
	run([]string{"check", src}, &stdout, &stderr)
	if !strings.Contains(stderr.String(), "labs/:zip would be empty: no source builds into labs/") {
		t.Errorf("empty folder bundle: %s", stderr.String())
	}
}

func TestNewStructure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rabbitmq")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", dir, "--structure"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	for _, f := range []string{"si.conf", "md/rabbitmq.md", "slides/pdf", "html/images", "html/animations", "html/pdfs", "videos", "dist", "labs"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("new --structure lacks %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "md", "si.conf")); err == nil {
		t.Error("one si.conf, at the project root")
	}
	if code := run([]string{"check", dir}, &stdout, &stderr); code != 0 || strings.Contains(stderr.String(), "warning") {
		t.Errorf("a new structured project checks without warnings: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "note: si.conf line") || !strings.Contains(stderr.String(), "course:scorm: nothing to bundle yet") {
		t.Errorf("an empty kind bundle is a note: %s", stderr.String())
	}
	// plain si new into a folder whose si.conf has a map makes its folders too
	other := filepath.Join(t.TempDir(), "kafka")
	os.MkdirAll(other, 0o755)
	os.WriteFile(filepath.Join(other, "si.conf"), []byte("structure:\n  src/ markdown\n  out/ deck\n"), 0o644)
	if code := run([]string{"new", other}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(other, "src", "kafka.md")); err != nil {
		t.Error("the deck goes into the markdown folder")
	}
}
