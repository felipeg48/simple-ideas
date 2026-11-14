package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

// dumpDOM is the page's DOM after wait (--virtual-time-budget=N).
func dumpDOM(t *testing.T, url, wait string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dom, err := exec.CommandContext(ctx, findChrome(), "--headless=new", "--use-mock-keychain", "--disable-gpu",
		wait, "--dump-dom", url).Output()
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
f.addEventListener('load', function () {
  setTimeout(function () {
    var d = f.contentDocument, b = d.querySelector('.si-check-run');
    log.textContent += (b && !d.querySelector('.si-check-bar').hidden ? 'button-shown ' : 'no-button ');
    b.click();
    setTimeout(function () {
      log.textContent += 'RESULT ' + d.querySelector('.si-check-result').textContent + ' OUT ' + d.querySelector('.si-check-out').textContent;
      fetch('`+hookSrv.URL+`/got').then(function (r) { return r.text(); }).then(function (t) { log.textContent += ' GOT ' + t; });
    }, 1500);
  }, 300);
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
	dom := dumpDOM(t, srv.URL+"/harness.html", "--virtual-time-budget=8000")
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
