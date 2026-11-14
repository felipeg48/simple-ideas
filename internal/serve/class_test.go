package serve

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func classEv(id, event, learner, lesson, data string) string {
	l := ""
	if lesson != "" {
		l = `"lesson":{"id":"` + lesson + `","title":"x","kind":"lab"},`
	}
	return `{"version":1,"event":"` + event + `","id":"` + id + `","time":"2026-11-14T10:00:00Z","audience":"learner",` +
		`"course":{"id":"c","title":"C"},` + l + `"learner":{"id":"` + learner + `"},"data":` + data + `}`
}

func TestParseClassEvent(t *testing.T) {
	now := time.Now()
	ev, err := parseClassEvent([]byte(`{"version":1,"event":"quiz.submitted","id":"e-1","audience":"instructor",
		"lesson":{"id":"/02/../02/quiz.html"},"learner":{"id":"l-0123456789ab","name":"  Ana\u0007 "},
		"data":{"score":180.4,"pass":true,"attempt":2,"questions":[{"id":"q1","correct":true},{"id":"<b>","correct":false}],"answers":["secret"]}}`), "web101", now)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Course != "web101" || ev.Lesson != "02/quiz.html" || ev.Name != "Ana" || !ev.Instructor || ev.Data.Score != 100 ||
		ev.Data.Attempt != 2 || len(ev.Data.Questions) != 1 || !ev.At.Equal(now) {
		t.Errorf("parsed: %+v", ev)
	}
	line, _ := json.Marshal(ev)
	if strings.Contains(string(line), "secret") {
		t.Error("only the fields si's events carry are kept")
	}
	for name, body := range map[string]string{
		"version":  `{"version":2,"event":"lesson.completed","id":"e","learner":{"id":"l-0123456789ab"}}`,
		"event":    `{"version":1,"event":"rm -rf","id":"e","learner":{"id":"l-0123456789ab"}}`,
		"learner":  `{"version":1,"event":"lesson.completed","id":"e","learner":{"id":"bob"}}`,
		"id":       `{"version":1,"event":"lesson.completed","id":"a b","learner":{"id":"l-0123456789ab"}}`,
		"lesson":   `{"version":1,"event":"lesson.completed","id":"e","lesson":{"id":"x.md"},"learner":{"id":"l-0123456789ab"}}`,
		"not json": `nope`,
	} {
		if _, err := parseClassEvent([]byte(body), "c", now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func mustEv(t *testing.T, body string, at time.Time) classEvent {
	t.Helper()
	ev, err := parseClassEvent([]byte(body), "c", at)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

// TestClassAggregate: three learners; the grid, who is stuck, the lesson
// and quiz-question summaries.
func TestClassAggregate(t *testing.T) {
	course := classCourse{ID: "c", Title: "C", Lessons: []classLesson{
		{ID: "01/read.html", Title: "Read", Kind: "reading", Module: "M1"},
		{ID: "01/lab.html", Title: "Lab", Kind: "lab", Module: "M1"},
		{ID: "02/quiz.html", Title: "Quiz", Kind: "quiz", Module: "M2"},
	}}
	t0 := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	ana, ben, cy := "l-aaaaaaaaaaaa", "l-bbbbbbbbbbbb", "l-cccccccccccc"
	at := func(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }
	evs := []classEvent{
		mustEv(t, strings.Replace(classEv("1", "lesson.completed", ana, "01/read.html", `{}`), `"id":"`+ana+`"`, `"id":"`+ana+`","name":"Ana"`, 1), at(0)),
		mustEv(t, classEv("2", "lab.step.checked", ana, "01/lab.html", `{"step":1,"title":"Start","pass":true}`), at(1)),
		mustEv(t, classEv("3", "lesson.completed", ana, "01/lab.html", `{}`), at(2)),
		mustEv(t, classEv("4", "quiz.submitted", ana, "02/quiz.html", `{"score":50,"pass":false,"attempt":1,"questions":[{"id":"q1","correct":true},{"id":"q2","correct":false}]}`), at(3)),
		mustEv(t, classEv("5", "quiz.submitted", ana, "02/quiz.html", `{"score":100,"pass":true,"attempt":2,"questions":[{"id":"q1","correct":true},{"id":"q2","correct":true}]}`), at(4)),
		mustEv(t, classEv("6", "lesson.completed", ana, "02/quiz.html", `{}`), at(4)),
		mustEv(t, classEv("7", "course.completed", ana, "", `{"lessons":3}`), at(4)),
		// Ben fails the same check twice
		mustEv(t, classEv("8", "lab.step.checked", ben, "01/lab.html", `{"step":2,"title":"Serve","pass":false,"exit":1}`), at(20)),
		mustEv(t, classEv("9", "lab.step.checked", ben, "01/lab.html", `{"step":2,"title":"Serve","pass":false,"exit":1}`), at(21)),
		// Cy took the quiz and got q2 wrong, a while ago
		mustEv(t, classEv("10", "quiz.submitted", cy, "02/quiz.html", `{"score":50,"pass":false,"attempt":1,"questions":[{"id":"q1","correct":true},{"id":"q2","correct":false}]}`), at(0)),
		// another course's event is not this one's
		{ID: "11", Event: "lesson.completed", Course: "other", Learner: cy, Lesson: "01/read.html", At: at(0)},
	}
	v := classAggregate(course, evs, at(22), 10*time.Minute)
	if len(v.Learners) != 3 || v.Events != 10 {
		t.Fatalf("learners %d events %d", len(v.Learners), v.Events)
	}
	if v.Learners[0].Label != "Ana" || v.Learners[1].Label != "Learner bbbb" || v.Learners[0].Done != 3 || !v.Learners[0].Completed {
		t.Errorf("learners: %+v %+v", v.Learners[0], v.Learners[1])
	}
	q := v.Learners[0].Cells["02/quiz.html"].Quiz
	if q.Score != 100 || q.Best != 100 || q.Attempts != 2 || !q.Pass {
		t.Errorf("quiz cell: %+v", q)
	}
	k := v.Learners[1].Cells["01/lab.html"].Checks[0]
	if k.Step != 2 || k.Fails != 2 || k.Pass || k.Tries != 2 {
		t.Errorf("check cell: %+v", k)
	}
	var reasons []string
	for _, s := range v.Stuck {
		reasons = append(reasons, s.Label+": "+s.Reason)
	}
	want := []string{"Learner cccc: no progress for 22 min", "Learner bbbb: failed check 2 (“Serve”) 2 times in a row"}
	if fmt.Sprint(reasons) != fmt.Sprint(want) {
		t.Errorf("stuck: %q, want %q", reasons, want)
	}
	if v2 := classAggregate(course, evs, at(22), 30*time.Minute); len(v2.Stuck) != 1 {
		t.Errorf("stuck after 30 min: %+v", v2.Stuck)
	}
	lab, quiz := v.Lessons[1], v.Lessons[2]
	if lab.Done != 1 || lab.Started != 2 || lab.Failing != 1 || quiz.QuizTakers != 2 || quiz.QuizPassed != 1 || quiz.QuizAvg != 75 {
		t.Errorf("lessons: %+v %+v", lab, quiz)
	}
	if len(v.Quizzes) != 1 || fmt.Sprint(v.Quizzes[0].Questions) != "[{q1 0 2} {q2 1 2}]" {
		t.Errorf("quiz questions (each learner's last attempt): %+v", v.Quizzes)
	}

	rows, err := csv.NewReader(strings.NewReader(string(classCSV(v)))).ReadAll()
	if err != nil || len(rows) != 1+3*3 || rows[0][0] != "course" {
		t.Fatalf("csv: %v %d", err, len(rows))
	}
	if r := rows[3]; r[2] != "Ana" || r[4] != "Quiz" || r[6] != "yes" || r[7] != "100" || r[9] != "2" || r[10] != "true" {
		t.Errorf("Ana's quiz row: %q", r)
	}
	if r := rows[5]; r[11] != "0" || r[12] != "1" {
		t.Errorf("Ben's lab row: %q", r)
	}
	if csvCell("=HYPERLINK(1)") != "'=HYPERLINK(1)" || csvCell("Ana") != "Ana" {
		t.Error("a typed name never runs as a formula")
	}
}

// TestClassRoom: events are kept once, in .si/class/<date>.jsonl, and a new
// server starts from today's file.
func TestClassRoom(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	c := newClassRoom(dir, now)
	ev := mustEv(t, classEv("e-1", "lesson.completed", "l-aaaaaaaaaaaa", "01/a.html", `{}`), now)
	ch, done := c.subscribe()
	defer done()
	if ok, err := c.add(ev); !ok || err != nil {
		t.Fatal(ok, err)
	}
	select {
	case <-ch:
	default:
		t.Error("a dashboard hears of a new event")
	}
	if ok, _ := c.add(ev); ok {
		t.Error("a duplicate id is dropped")
	}
	b, _ := os.ReadFile(filepath.Join(dir, now.Format("2006-01-02")+".jsonl"))
	if strings.Count(string(b), "\n") != 1 || !strings.Contains(string(b), `"learner":"l-aaaaaaaaaaaa"`) {
		t.Errorf("file: %s", b)
	}
	if again := newClassRoom(dir, now); len(again.list()) != 1 || !again.seen["e-1"] {
		t.Error("a restarted server keeps today's class")
	}
	c.per["l-aaaaaaaaaaaa"] = classMaxPerLearner
	ev.ID = "e-2"
	if _, err := c.add(ev); err != errClassFull {
		t.Errorf("a learner over the cap: %v", err)
	}
}

func classServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "c", "01"), 0o755)
	os.WriteFile(filepath.Join(dir, "c", "_course.md"), []byte("---\ntitle: Web 101\n---\n- Module 1: Basics\n  - [A](01/a.md)\n  - [B](01/b.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c", "01", "a.md"), []byte("# A\n\nText.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c", "01", "b.md"), []byte("# B\n\nText.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "plain.md"), []byte("# P\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Class, s.ClassKey = true, "k3y"
	return s, dir
}

// TestClassHTTP: learners' pages post events; the dashboard, its data, the
// stream and the CSV open on the loopback or with the key only.
func TestClassHTTP(t *testing.T) {
	s, dir := classServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	page, _ := http.Get(ts.URL + "/c/01/a.html")
	html, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if !strings.Contains(string(html), `"class":"/_si/class/event"`) || !strings.Contains(string(html), `"learner":"ask"`) {
		t.Error("a course page served with --class sends to the class view and asks the learner's name, even without a webhook")
	}

	// the stream, before the event
	sreq, _ := http.NewRequest("GET", ts.URL+"/_si/class/events", nil)
	sres, err := http.DefaultClient.Do(sreq)
	if err != nil || sres.StatusCode != 200 {
		t.Fatalf("events: %v %v", err, sres)
	}
	defer sres.Body.Close()
	lines := bufio.NewReader(sres.Body)
	lines.ReadString('\n') // retry
	lines.ReadString('\n') // : connected
	lines.ReadString('\n') // the blank line

	post := func(page, origin, body string) int {
		req, _ := http.NewRequest("POST", ts.URL+"/_si/class/event?page="+page, strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	good := classEv("e-1", "lesson.completed", "l-aaaaaaaaaaaa", "01/a.html", `{}`)
	if c := post("/c/01/a.html", ts.URL, good); c != http.StatusAccepted {
		t.Fatalf("event: %d", c)
	}
	if c := post("/c/01/a.html", ts.URL, good); c != http.StatusNoContent {
		t.Errorf("the same event again: %d", c)
	}
	if c := post("/c/01/a.html", "https://evil.example.com", good); c != http.StatusForbidden {
		t.Errorf("another site: %d", c)
	}
	if c := post("/plain.html", "", good); c != http.StatusNotFound {
		t.Errorf("not a course page: %d", c)
	}
	if c := post("/c/01/a.html", "", `{"version":1}`); c != http.StatusBadRequest {
		t.Errorf("a bad event: %d", c)
	}
	done := make(chan string, 1)
	go func() { l, _ := lines.ReadString('\n'); done <- l }()
	select {
	case l := <-done:
		if l != "event: changed\n" {
			t.Errorf("stream: %q", l)
		}
	case <-time.After(3 * time.Second):
		t.Error("the dashboard's stream hears nothing")
	}

	res, _ := http.Get(ts.URL + "/_si/class/data")
	var v classView
	json.NewDecoder(res.Body).Decode(&v)
	res.Body.Close()
	if v.Course.ID != "c" || len(v.Course.Lessons) != 2 || len(v.Learners) != 1 || !v.Learners[0].Cells["01/a.html"].Done {
		t.Errorf("data: %+v", v)
	}
	if _, err := os.Stat(filepath.Join(dir, ".si", "class", time.Now().Format("2006-01-02")+".jsonl")); err != nil {
		t.Error("the event is kept beside the course")
	}
	for _, p := range []string{"/_si/class", "/_si/class/csv"} {
		res, _ := http.Get(ts.URL + p)
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || (p == "/_si/class" && !strings.Contains(string(b), "Web 101")) || (p == "/_si/class/csv" && !strings.HasPrefix(string(b), "course,learner")) {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
	}

	// another machine: the key, never the Host header alone
	call := func(remote, host, q string) int {
		req := httptest.NewRequest("GET", "/_si/class/data"+q, nil)
		req.RemoteAddr, req.Host = remote, host
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	if c := call("192.168.1.9:5000", "127.0.0.1:8000", ""); c != http.StatusForbidden {
		t.Errorf("a LAN request naming the loopback: %d", c)
	}
	if c := call("192.168.1.9:5000", "192.168.1.2:8000", "?k=wrong"); c != http.StatusForbidden {
		t.Errorf("a wrong key: %d", c)
	}
	if c := call("192.168.1.9:5000", "192.168.1.2:8000", "?k=k3y"); c != http.StatusOK {
		t.Errorf("the key: %d", c)
	}
	if c := call("127.0.0.1:5000", "evil.example.com", ""); c != http.StatusForbidden {
		t.Errorf("a rebound name on the loopback: %d", c)
	}

	off, _ := New(dir, nil, "", "")
	ts2 := httptest.NewServer(off.Handler())
	defer ts2.Close()
	if res, _ := http.Get(ts2.URL + "/_si/class"); res.StatusCode == http.StatusOK {
		t.Error("no --class: no class view")
	}
	page, _ = http.Get(ts2.URL + "/c/01/a.html")
	html, _ = io.ReadAll(page.Body)
	page.Body.Close()
	if strings.Contains(string(html), "/_si/class/event") || strings.Contains(string(html), "SI_WEBHOOK") {
		t.Error("without --class or a webhook a course sends nothing")
	}
}
