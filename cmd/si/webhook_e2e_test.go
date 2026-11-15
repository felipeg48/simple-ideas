package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/serve"
	"github.com/felipeg48/simple-ideas/internal/webhook"
)

// collector is a webhook receiver: POST keeps the event, GET /got lists
// them (with CORS, so a page on another origin can read them).
type collector struct {
	mu     sync.Mutex
	events []map[string]any
	sigs   []string
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodPost {
		b, _ := io.ReadAll(r.Body)
		var ev map[string]any
		json.Unmarshal(b, &ev)
		c.mu.Lock()
		c.events, c.sigs = append(c.events, ev), append(c.sigs, r.Header.Get(webhook.SignatureHeader))
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	json.NewEncoder(w).Encode(c.events)
}

func (c *collector) kinds() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var k []string
	for _, e := range c.events {
		k = append(k, e["event"].(string))
	}
	return strings.Join(k, " ")
}

// pollJS is poll(get, then): a harness waits for a condition, never a fixed
// time: get (a value or a promise) is tried every 100 ms until it is truthy,
// at most 200 times (20 s of virtual time), then then(value) runs.
const pollJS = `function poll(get, then, n) {
  n = n || 0;
  Promise.resolve().then(get).catch(function () { return null; }).then(function (v) {
    if (v || n >= 200) then(v); else setTimeout(function () { poll(get, then, n + 1); }, 100);
  });
}`

// dumpDOM is the page's DOM after wait (--virtual-time-budget=N) and any
// other flags.
func dumpDOM(t *testing.T, url, wait string, flags ...string) string {
	t.Helper()
	dom, err := cdp.Output(chromeWait, findChrome(), append(append([]string{"--headless=new", "--use-mock-keychain", "--disable-gpu",
		wait}, flags...), "--dump-dom", url)...)
	if err != nil {
		t.Fatalf("chrome: %v", err)
	}
	return string(dom)
}

// TestCourseWebhookHeadless: a built course with webhook: sends its events
// from a real browser. The quiz lesson shows the notice; Check my answers
// with the right picks sends quiz.submitted (score, attempt, per-question
// correctness, no answers) and lesson.completed, by sendBeacon to another
// origin, with the course, lesson and a random learner id.
func TestCourseWebhookHeadless(t *testing.T) {
	needChrome(t)
	tmp := t.TempDir()
	out := filepath.Join(tmp, "html")
	hooks := &collector{}
	mux := http.NewServeMux()
	mux.Handle("/hook", hooks)
	mux.Handle("/", http.FileServer(http.Dir(out)))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	hookSrv := httptest.NewServer(hooks) // another origin: what a real webhook is
	defer hookSrv.Close()

	src := filepath.Join(tmp, "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	cf := filepath.Join(src, "_course.md")
	raw, _ := os.ReadFile(cf)
	os.WriteFile(cf, []byte(strings.Replace(string(raw), "---\n", "---\nwebhook: "+hookSrv.URL+"/hook\nwebhook-token: class-7\n", 1)), 0o644)
	if code := run([]string{"build", src, "-o", out, "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "is not https://") {
		t.Errorf("si build warns about a plain http webhook: %s", stderr.String())
	}
	quiz, _ := os.ReadFile(filepath.Join(src, "02-html", "quiz.md"))
	_, qs, _ := deck.ParseQuiz(string(quiz))
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
	// the harness answers the quiz in a frame, then runs then(d) with the
	// frame's document; got() counts what the webhook received
	harness := func(then string) string {
		return `<!doctype html><meta charset="utf-8"><pre id="log"></pre><iframe id="f" src="02-html/quiz.html"></iframe><script>
var answers = ` + string(aj) + `, f = document.getElementById('f'), log = document.getElementById('log');
var got = function () { return fetch('` + hookSrv.URL + `/got').then(function (r) { return r.json(); }).then(function (a) { return (a || []).length; }); };
f.addEventListener('load', function () {
  setTimeout(function () {
    var d = f.contentDocument;
    log.textContent += (d.querySelector('.si-webhook-notice') ? 'notice-shown ' : 'no-notice ');
    d.querySelectorAll('.si-quiz-q').forEach(function (q, i) {
      q.querySelectorAll('input').forEach(function (inp) { inp.checked = answers[i].indexOf(inp.value) >= 0; });
    });
    d.querySelector('form.si-quiz').requestSubmit();
    setTimeout(function () { (` + then + `)(d); }, 1000);
  }, 300);
});
</script>`
	}
	os.WriteFile(filepath.Join(out, "harness.html"), []byte(harness(`function () { got().then(function (k) { log.textContent += 'GOT ' + k; }); }`)), 0o644)
	dom := dumpDOM(t, srv.URL+"/harness.html", "--virtual-time-budget=8000")
	if !strings.Contains(dom, "notice-shown") {
		t.Errorf("the results notice shows: %s", dom)
	}
	if !strings.Contains(hooks.kinds(), "quiz.submitted") || !strings.Contains(hooks.kinds(), "lesson.completed") {
		t.Fatalf("events received: %q\n%s", hooks.kinds(), dom)
	}
	for _, ev := range hooks.events {
		if ev["event"] != "quiz.submitted" {
			continue
		}
		d := ev["data"].(map[string]any)
		qs := d["questions"].([]any)
		lesson := ev["lesson"].(map[string]any)
		learner := ev["learner"].(map[string]any)
		if d["score"] != 100.0 || d["pass"] != true || d["attempt"] != 1.0 || len(qs) != len(answers) || ev["token"] != "class-7" ||
			lesson["id"] != "02-html/quiz.html" || lesson["kind"] != "quiz" || ev["course"].(map[string]any)["id"] != "web101" ||
			!strings.HasPrefix(learner["id"].(string), "l-") || ev["version"] != 1.0 {
			t.Errorf("quiz.submitted: %v", ev)
		}
		b, _ := json.Marshal(ev)
		if strings.Contains(string(b), "value") || strings.Contains(string(b), "answer") {
			t.Errorf("an event never holds the picks: %s", b)
		}
	}

	// webhook-consent: required: nothing leaves before OK; then the queue does
	os.WriteFile(cf, []byte(strings.Replace(string(raw), "---\n", "---\nwebhook: "+hookSrv.URL+"/hook\nwebhook-consent: required\n", 1)), 0o644)
	if code := run([]string{"build", src, "-o", out, "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build: %s", stderr.String())
	}
	hooks.mu.Lock()
	hooks.events = nil
	hooks.mu.Unlock()
	os.WriteFile(filepath.Join(out, "harness.html"), []byte(harness(`function (d) {
  got().then(function (k) {
    log.textContent += 'BEFORE ' + k + ' ';
    d.querySelector('.si-webhook-ok').click();
    setTimeout(function () { got().then(function (k) { log.textContent += 'AFTER ' + k; }); }, 1000);
  });
}`)), 0o644)
	dom = dumpDOM(t, srv.URL+"/harness.html", "--virtual-time-budget=8000")
	if !strings.Contains(dom, "BEFORE 0 ") || strings.Contains(dom, "AFTER 0") || !strings.Contains(dom, "AFTER ") {
		t.Errorf("consent required: nothing before OK, the queue after: %s", dom)
	}
}

// TestLabCheckHeadless: in si play a lab's ✓ Check runs the step's command
// on the server, shows Passed with the output, and the lab.step.checked event
// goes through /_si/relay to the course's webhook, signed.
func TestLabCheckHeadless(t *testing.T) {
	needChrome(t)
	t.Setenv(webhook.SecretEnv, "relay-secret")
	hooks := &collector{}
	hookSrv := httptest.NewServer(hooks)
	defer hookSrv.Close()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "c", "01"), 0o755)
	os.WriteFile(filepath.Join(dir, "c", "_course.md"), []byte("---\ntitle: Checks\nwebhook: "+hookSrv.URL+"/hook\n---\n- Module 1: M\n  - [Lab](01/lab.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c", "01", "lab.md"), []byte("---\ntype: lab\n---\n# Lab\n\n## Step 1: Say hello\n\n<!-- check: echo hello · expect \"hello\" -->\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "harness.html"), []byte(`<!doctype html><meta charset="utf-8"><pre id="log"></pre><iframe id="f" src="c/01/lab.html"></iframe><script>
var f = document.getElementById('f'), log = document.getElementById('log');
`+pollJS+`
f.addEventListener('load', function () {
  poll(function () { return f.contentDocument.querySelector('.si-check-run'); }, function () {
    var d = f.contentDocument, b = d.querySelector('.si-check-run');
    log.textContent += (b && !d.querySelector('.si-check-bar').hidden ? 'button-shown ' : 'no-button ');
    b.click();
    poll(function () { return /Passed|Failed/.test(d.querySelector('.si-check-result').textContent); }, function () {
      log.textContent += 'RESULT ' + d.querySelector('.si-check-result').textContent + ' OUT ' + d.querySelector('.si-check-out').textContent;
      // the relay posts to the webhook before it answers: wait until the webhook has it
      poll(function () { return fetch('`+hookSrv.URL+`/got').then(function (r) { return r.text(); }).then(function (t) { return t.indexOf('lab.step.checked') >= 0 && t; }); },
        function (t) { log.textContent += ' GOT ' + t; });
    });
  });
});
</script>`), 0o644)
	s, err := serve.New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Terminal, s.Relay = true, true
	h := s.Handler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") { // no live streams: 204 stops an EventSource, so virtual time can run out
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()
	dom := dumpDOM(t, srv.URL+"/harness.html", "--virtual-time-budget=30000")
	if !strings.Contains(dom, "button-shown") || !strings.Contains(dom, "RESULT Passed · exit 0") || !strings.Contains(dom, "OUT hello") {
		t.Errorf("✓ Check: %s", dom)
	}
	if !strings.Contains(hooks.kinds(), "lab.step.checked") {
		t.Fatalf("events: %q\n%s", hooks.kinds(), dom)
	}
	for i, ev := range hooks.events {
		if ev["event"] != "lab.step.checked" {
			continue
		}
		d := ev["data"].(map[string]any)
		if d["pass"] != true || d["step"] != 1.0 || d["title"] != "Step 1: Say hello" || d["output"] != nil {
			t.Errorf("lab.step.checked: %v", ev)
		}
		if hooks.sigs[i] == "" || !strings.HasPrefix(hooks.sigs[i], "sha256=") {
			t.Errorf("the relay signs what it forwards: %q", hooks.sigs[i])
		}
	}
}

// TestWebhookRetryAndIDs: a course opened from file:// names its pages
// urn:si:course:<id>:<page> in xAPI, or under webhook-base: when set; a send
// the webhook refuses (503) is tried again on the same page.
func TestWebhookRetryAndIDs(t *testing.T) {
	needChrome(t)
	var mu sync.Mutex
	var fails int
	var got []map[string]any
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodOptions:
		case http.MethodPost:
			if fails == 0 {
				fails++
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			var st map[string]any
			json.NewDecoder(r.Body).Decode(&st)
			got = append(got, st)
		default:
			fmt.Fprint(w, len(got))
		}
	}))
	defer hookSrv.Close()
	tmp := t.TempDir()
	src, out := filepath.Join(tmp, "web101"), filepath.Join(tmp, "html")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new: %s", stderr.String())
	}
	cf := filepath.Join(src, "_course.md")
	raw, _ := os.ReadFile(cf)
	quiz, _ := os.ReadFile(filepath.Join(src, "02-html", "quiz.md"))
	_, qs, _ := deck.ParseQuiz(string(quiz))
	var answers [][]string
	for _, q := range qs {
		var ids []string
		for j, o := range q.Options {
			if o.OK {
				ids = append(ids, fmt.Sprint(j))
			}
		}
		answers = append(answers, ids)
	}
	aj, _ := json.Marshal(answers)
	for _, c := range []struct{ base, lesson, course string }{
		{"", "urn:si:course:web101:02-html/quiz.html", "urn:si:course:web101"},
		{"https://learn.example.com/web101", "https://learn.example.com/web101/02-html/quiz.html", "https://learn.example.com/web101/"},
	} {
		head := "---\nwebhook: " + hookSrv.URL + "/hook\nwebhook-format: xapi\nwebhook-events: quiz.submitted\n"
		if c.base != "" {
			head += "webhook-base: " + c.base + "\n"
		}
		os.WriteFile(cf, []byte(strings.Replace(string(raw), "---\n", head, 1)), 0o644)
		if code := run([]string{"build", src, "-o", out, "--no-film"}, &stdout, &stderr); code != 0 {
			t.Fatalf("build: %s", stderr.String())
		}
		mu.Lock()
		fails, got = 0, nil
		mu.Unlock()
		os.WriteFile(filepath.Join(out, "harness.html"), []byte(`<!doctype html><meta charset="utf-8"><pre id="log"></pre><iframe id="f" src="02-html/quiz.html"></iframe><script>
var answers = `+string(aj)+`, f = document.getElementById('f'), log = document.getElementById('log');
`+pollJS+`
f.addEventListener('load', function () {
  poll(function () { return f.contentDocument.querySelector('form.si-quiz'); }, function () {
    var d = f.contentDocument;
    d.querySelectorAll('.si-quiz-q').forEach(function (q, i) {
      q.querySelectorAll('input').forEach(function (inp) { inp.checked = answers[i].indexOf(inp.value) >= 0; });
    });
    d.querySelector('form.si-quiz').requestSubmit();
    poll(function () { return fetch('`+hookSrv.URL+`/got').then(function (r) { return r.text(); }).then(function (n) { return +n > 0 && n; }); },
      function (n) { log.textContent += 'GOT ' + n; });
  });
});
</script>`), 0o644)
		dom := dumpDOM(t, "file://"+filepath.ToSlash(filepath.Join(out, "harness.html")), "--virtual-time-budget=30000", "--allow-file-access-from-files")
		mu.Lock()
		if fails != 1 || len(got) != 1 {
			t.Errorf("base %q: a refused send is tried again on the page: %d refused, %d got\n%s", c.base, fails, len(got), dom)
		} else {
			obj := got[0]["object"].(map[string]any)
			parent := got[0]["context"].(map[string]any)["contextActivities"].(map[string]any)["parent"].([]any)[0].(map[string]any)
			if obj["id"] != c.lesson || parent["id"] != c.course {
				t.Errorf("base %q: activity ids %v, %v; want %s, %s", c.base, obj["id"], parent["id"], c.lesson, c.course)
			}
		}
		mu.Unlock()
	}
}
