package studio

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// The Class view: Start runs the course with --class on its own port (this
// machine), the dashboard and a learner's event go through it, a second
// Start attaches to the same class, Stop ends it; a folder that is not a
// course is refused.
func TestClassView(t *testing.T) {
	f := newFx(t)
	if _, _, _, err := WriteStarter("course", filepath.Join(f.root, "web101"), "Web for beginners", f.root); err != nil {
		t.Fatal(err)
	}
	defer f.api.cls.closeAll()
	if code, m := f.do("GET", "class?course=web101", nil); code != 200 || m["running"] != false || m["course"] != "web101/_course.md" {
		t.Fatalf("before: %d %v", code, m)
	}
	code, m := f.do("POST", "class", map[string]any{"course": "web101/_course.md"})
	dash, _ := m["dashboard"].(string)
	if code != 200 || m["running"] != true || !strings.HasPrefix(dash, "http://127.0.0.1:") || !strings.Contains(m["qr"].(string), "<svg") {
		t.Fatalf("start: %d %v", code, m)
	}
	learners := m["learners"].(string)
	g, err := deck.LoadGuide(filepath.Join(f.root, "web101"))
	if err != nil || len(g.Lessons()) == 0 {
		t.Fatal(err)
	}
	rel, _ := filepath.Rel(g.Dir, g.Lessons()[0].File)
	page := filepath.ToSlash(strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html")
	ev := `{"version":1,"event":"lesson.completed","id":"e-1","audience":"learner","lesson":{"id":"` + page + `"},"learner":{"id":"l-0123456789ab","name":"Ana"},"data":{}}`
	req, _ := http.NewRequest("POST", learners+"_si/class/event?page=/"+page, strings.NewReader(ev))
	req.Header.Set("Origin", strings.TrimSuffix(learners, "/"))
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != http.StatusAccepted {
		t.Fatalf("event: %v %v", res, err)
	}
	res, err := http.Get(dash + "/data")
	if err != nil {
		t.Fatal(err)
	}
	var v struct{ Learners []struct{ Label string } }
	json.NewDecoder(res.Body).Decode(&v)
	res.Body.Close()
	if len(v.Learners) != 1 || v.Learners[0].Label != "Ana" {
		t.Errorf("dashboard data: %+v", v)
	}
	if _, m2 := f.do("POST", "class", map[string]any{"course": "web101"}); m2["dashboard"] != dash {
		t.Errorf("a second Start attaches: %v", m2["dashboard"])
	}
	if _, m := f.do("POST", "class", map[string]any{"course": "web101", "stop": true}); m["running"] != false {
		t.Errorf("stop: %v", m)
	}
	if _, err := http.Get(dash); err == nil {
		t.Error("the class still answers after Stop")
	}
	f.put("notes/a.md", "# A\n")
	if code, _ := f.do("GET", "class?course=notes", nil); code != 404 {
		t.Errorf("not a course: %d", code)
	}
}

// New… → Course writes the course si new --type course writes, with the
// name as its title, and opens its _course.md.
func TestStarterCourse(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "web101")
	created, open, _, err := WriteStarter("course", abs, "Web for beginners", root)
	if err != nil {
		t.Fatal(err)
	}
	if open != filepath.Join(abs, deck.CourseFile) || len(created) < 5 {
		t.Fatalf("open %s, created %d", open, len(created))
	}
	g, err := deck.LoadGuide(abs)
	if err != nil || !g.IsCourse() || g.Meta.Title != "Web for beginners" || len(g.Problems) != 0 {
		t.Fatalf("course: %v %+v", err, g)
	}
}
