package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillJarPublish: a first publish creates the course, a section, an HTML
// lesson, a video (an asset from its URL) and a SCORM lesson (a web package);
// a second publish patches them in place, fetches only the changed package
// again and deletes the lesson that is gone.
func TestSkillJarPublish(t *testing.T) {
	var steps []string
	bodies := map[string]map[string]any{}
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "sk-key" || p != "" {
			t.Errorf("%s %s: basic auth %q %q", r.Method, r.URL.Path, u, p)
		}
		step := r.Method + " " + r.URL.Path
		steps = append(steps, step)
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		bodies[step] = b
		if r.Method == "POST" {
			n++
			fmt.Fprintf(w, `{"id":"id%d"}`, n)
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	sj := &SkillJar{API: srv.URL, Key: "sk-key"}
	course := SJCourse{Title: "Web 101", Description: "A first page.", Duration: "1h", Lessons: []SJLesson{
		{Key: "module-1", Title: "How the web works", Type: "SECTION"},
		{Key: "01/read.md", Title: "Read", Type: "HTML", HTML: "<p>Hi</p>"},
		{Key: "01/video.mp4", Title: "Watch", Type: "ASSET", URL: "https://files.example/video.mp4", Hash: "v1"},
		{Key: "01/quiz.md", Title: "Quiz", Type: "WEB_PACKAGE", URL: "https://files.example/quiz.zip", Hash: "q1"},
	}}
	ids := &SJIDs{}
	id, err := sj.Publish(context.Background(), course, ids, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "POST /courses,POST /lessons,POST /lessons,POST /assets,POST /lessons,POST /web-packages,POST /lessons"
	if id != "id1" || strings.Join(steps, ",") != want {
		t.Fatalf("first publish: %s %v", id, steps)
	}
	if c := bodies["POST /courses"]; c["title"] != "Web 101" || c["short_description"] != "A first page." || c["manual_course_time"] != "1h" || c["enforce_sequential_navigation"] != false {
		t.Errorf("course: %v", c)
	}
	if w := bodies["POST /web-packages"]; w["content_url"] != "https://files.example/quiz.zip" {
		t.Errorf("web package: %v", w)
	}
	if l := bodies["POST /lessons"]; l["type"] != "WEB_PACKAGE" || l["content_web_package_id"] != "id6" || l["course_id"] != "id1" || l["order"] != float64(4) {
		t.Errorf("the last lesson: %v", l)
	}
	if ids.Lessons["01/video.mp4"] != "id5" || ids.Contents["01/video.mp4"] != "v1 id4" {
		t.Errorf("ids: %+v", ids)
	}
	path := filepath.Join(t.TempDir(), ".si", "publish", "skilljar.json")
	if err := ids.Save(path); err != nil {
		t.Fatal(err)
	}
	again, err := LoadSJIDs(path)
	if err != nil || again.Course != "id1" {
		t.Fatalf("reload: %v %+v", err, again)
	}

	steps = nil
	course.Lessons = []SJLesson{course.Lessons[0], course.Lessons[2], course.Lessons[3]} // the reading is gone
	course.Lessons[2].Hash = "q2"                                                        // the quiz changed
	if _, err := sj.Publish(context.Background(), course, again, nil); err != nil {
		t.Fatal(err)
	}
	want = "PATCH /courses/id1,PATCH /lessons/id2,PATCH /lessons/id5,POST /web-packages,PATCH /lessons/id7,DELETE /lessons/id3"
	if strings.Join(steps, ",") != want {
		t.Errorf("second publish:\n got %v\nwant %s", steps, want)
	}
	if _, gone := again.Lessons["01/read.md"]; gone {
		t.Error("the deleted lesson leaves the ids")
	}

	var dry bytes.Buffer // a dry run prints the requests, the key masked, and sends nothing
	steps = nil
	if _, err := sj.Publish(context.Background(), course, &SJIDs{}, &dry); err != nil {
		t.Fatal(err)
	}
	if len(steps) != 0 || !strings.Contains(dry.String(), "POST "+srv.URL+"/courses") || strings.Contains(dry.String(), "c2sta2V5") {
		t.Errorf("dry run: %v\n%s", steps, dry.String())
	}
}
