package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

// TestClassViewHeadless: si serve course/ --class. A learner in a real
// browser types a name and sends events through the page's own webhook.js;
// two more learners post theirs; the dashboard, open the whole time, updates
// over SSE without a reload: three rows, who is stuck, the quiz question most
// got wrong. SI_CLASS_SHOTS=<dir> also writes screenshots at the four sizes
// (and dark) to look at.
func TestClassViewHeadless(t *testing.T) {
	needChrome(t)
	src := filepath.Join(t.TempDir(), "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	srv, err := serve.New(src, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	srv.Class, srv.ClassKey = true, "test-key"
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	c, err := cdp.Start(findChrome(), 1440, 900)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	eval := func(js string) string {
		t.Helper()
		v, err := c.Eval(js)
		if err != nil {
			t.Fatalf("eval %s: %v", js, err)
		}
		var s any
		json.Unmarshal(v, &s)
		b, _ := json.Marshal(s)
		return strings.Trim(string(b), `"`)
	}
	wait := func(js, want string) {
		t.Helper()
		var got string
		for i := 0; i < 60; i++ {
			if got = eval(js); got == want {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("%s: %q, want %q", js, got, want)
	}
	nav := func(u string) {
		t.Helper()
		if err := c.Call("Page.navigate", map[string]any{"url": u}, nil); err != nil {
			t.Fatal(err)
		}
		wait("document.readyState", "complete")
	}

	// learner 1, in the browser: the notice asks a name; OK; a lab check fails twice
	nav(ts.URL + "/02-html/first-page.html")
	wait("!!document.querySelector('.si-webhook-name')", "true")
	eval(`document.querySelector('.si-webhook-name').value = 'Ana'; document.querySelector('.si-webhook-ok').click();
		siWebhook.emit('lab.step.checked', {step: 1, title: 'Make the folder', pass: false, exit: 1, ms: 9});
		siWebhook.emit('lab.step.checked', {step: 1, title: 'Make the folder', pass: false, exit: 1, ms: 9}); 'ok'`)

	// the dashboard, opened before the other learners arrive
	nav(ts.URL + "/_si/class")
	wait("document.body.dataset.ready", "2")
	wait("document.querySelectorAll('.cv-grid tbody tr').length", "1")
	if got := eval("document.querySelector('.cv-grid tbody td').textContent"); !strings.HasPrefix(got, "Ana") {
		t.Errorf("the learner's typed name: %q", got)
	}
	if got := eval("document.querySelector('.cv-stuck').textContent"); !strings.Contains(got, "failed check 1") {
		t.Errorf("who's stuck: %q", got)
	}

	// learners 2 and 3 post from their own browsers
	post := func(id, ev, learner, name, lesson, data string) {
		t.Helper()
		body := `{"version":1,"event":"` + ev + `","id":"` + id + `","time":"2026-11-14T10:00:00Z","audience":"learner","course":{"id":"web101","title":"T"},` +
			`"lesson":{"id":"` + lesson + `","title":"x","kind":"quiz"},"learner":{"id":"` + learner + `","name":"` + name + `"},"data":` + data + `}`
		req, _ := http.NewRequest("POST", ts.URL+"/_si/class/event?page=/"+lesson, strings.NewReader(body))
		req.Header.Set("Origin", ts.URL)
		res, err := http.DefaultClient.Do(req)
		if err != nil || res.StatusCode != http.StatusAccepted {
			t.Fatalf("post %s: %v %v", id, err, res)
		}
		res.Body.Close()
	}
	q := `{"score":%d,"pass":%t,"passMark":70,"attempt":1,"questions":[{"id":"q1","correct":true},{"id":"q2","correct":%t},{"id":"q3","correct":true}]}`
	post("b1", "lesson.completed", "l-bbbbbbbbbbbb", "Ben", "01-web/browser.html", `{}`)
	post("b2", "lesson.completed", "l-bbbbbbbbbbbb", "Ben", "01-web/journey.html", `{}`)
	post("b3", "demo.watched", "l-bbbbbbbbbbbb", "Ben", "01-web/page-load.html", `{"percent":100,"seconds":42}`)
	post("b4", "quiz.submitted", "l-bbbbbbbbbbbb", "Ben", "02-html/quiz.html", strings.NewReplacer("%d", "100", "%t", "true").Replace(q))
	post("c1", "quiz.submitted", "l-cccccccccccc", "", "02-html/quiz.html", strings.Replace(strings.Replace(q, "%d", "67", 1), "%t", "false", 2))
	wait("document.querySelectorAll('.cv-grid tbody tr').length", "3") // live: no reload
	wait("document.body.dataset.ready", "7")
	if got := eval("document.querySelector('.cv-q').textContent"); !strings.Contains(got, "2. Where does the title element go?1/2 wrong") {
		t.Errorf("quiz questions: %q", got)
	}
	if got := eval("document.querySelector('.cv-tiles').textContent"); !strings.Contains(got, "3learners") {
		t.Errorf("tiles: %q", got)
	}
	// filter by module: Module 2 only
	eval(`var m = document.getElementById('cv-module'); m.value = m.options[2].value; m.dispatchEvent(new Event('change')); 'ok'`)
	if got := eval("document.querySelectorAll('.cv-grid thead tr:last-child th').length"); got != "5" { // learner + 4 lessons of module 2
		t.Errorf("module filter: %s columns", got)
	}
	eval(`var m = document.getElementById('cv-module'); m.value = ''; m.dispatchEvent(new Event('change')); 'ok'`)

	if dir := os.Getenv("SI_CLASS_SHOTS"); dir != "" {
		shot := func(name string, w, h int) {
			c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 1, "mobile": w < 900}, nil)
			time.Sleep(300 * time.Millisecond)
			if b, err := c.Screenshot(); err == nil {
				os.WriteFile(filepath.Join(dir, name+".png"), b, 0o644)
			}
		}
		shot("class-1440", 1440, 900)
		shot("class-phone", 400, 800)
		shot("class-portrait", 820, 1180)
		shot("class-4k", 3840, 2160)
		c.Call("Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": "dark"}}}, nil)
		shot("class-dark", 1440, 900)
	}
}
