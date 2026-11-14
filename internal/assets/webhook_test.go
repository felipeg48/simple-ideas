package assets

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var updateWebhook = flag.Bool("update", false, "rewrite testdata/webhook golden files")

// webhookEvents are one event of each kind, as webhook.js makes them.
const webhookEvents = `[
 {"version":1,"event":"lesson.completed","id":"5f0c9a6b-2d41-4e8f-9a3c-0d1e2f3a4b5c","time":"2026-11-14T10:02:11.418Z","si":"3.18.0","audience":"learner",
  "course":{"id":"web101","title":"A first web page","url":"https://learn.example.com/web101/index.html"},
  "lesson":{"id":"02-html/tags.html","title":"Tags, elements and attributes","kind":"reading","url":"https://learn.example.com/web101/02-html/tags.html"},
  "learner":{"id":"l-3f9a0c71b2e4"},"data":{}},
 {"version":1,"event":"quiz.submitted","id":"6a1d0b7c-3e52-4f90-8b4d-1e2f3a4b5c6d","time":"2026-11-14T10:05:00.000Z","si":"3.18.0","audience":"learner",
  "course":{"id":"web101","title":"A first web page","url":"https://learn.example.com/web101/index.html"},
  "lesson":{"id":"02-html/quiz.html","title":"Check yourself","kind":"quiz","url":"https://learn.example.com/web101/02-html/quiz.html"},
  "learner":{"id":"l-3f9a0c71b2e4","name":"Ana"},
  "data":{"score":80,"pass":true,"passMark":70,"attempt":2,"questions":[{"id":"q1","correct":true},{"id":"q2","correct":false}]},"token":"c42"},
 {"version":1,"event":"lab.step.checked","id":"7b2e1c8d-4f63-4a01-9c5e-2f3a4b5c6d7e","time":"2026-11-14T10:09:30.000Z","si":"3.18.0","audience":"instructor",
  "course":{"id":"web101","title":"A first web page","url":"https://learn.example.com/web101/index.html"},
  "lesson":{"id":"02-html/first-page.html","title":"Build your first page","kind":"lab","url":"https://learn.example.com/web101/02-html/first-page.html"},
  "learner":{"id":"l-3f9a0c71b2e4"},"data":{"step":2,"title":"Step 2: Add the skeleton","pass":false,"exit":1,"ms":412}},
 {"version":1,"event":"demo.watched","id":"8c3f2d9e-5074-4b12-8d6f-3a4b5c6d7e8f","time":"2026-11-14T10:12:00.000Z","si":"3.18.0","audience":"learner",
  "course":{"id":"web101","title":"A first web page","url":"https://learn.example.com/web101/index.html"},
  "lesson":{"id":"01-web/page-load.html","title":"A page loads, step by step","kind":"video","url":"https://learn.example.com/web101/01-web/page-load.html"},
  "learner":{"id":"l-3f9a0c71b2e4"},"data":{"percent":100,"seconds":95}},
 {"version":1,"event":"course.completed","id":"9d4a3e0f-6185-4c23-9e7a-4b5c6d7e8f90","time":"2026-11-14T11:00:00.000Z","si":"3.18.0","audience":"learner",
  "course":{"id":"web101","title":"A first web page","url":"https://learn.example.com/web101/index.html"},
  "learner":{"id":"l-3f9a0c71b2e4","name":"Ana"},"data":{"lessons":10}}
]`

// formatAll runs webhook.js under node: every event in every format.
func formatAll(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed: webhook.js formats untested")
	}
	script := "global.window = {};\n" + WebhookJS + `
const evs = JSON.parse(require('fs').readFileSync(0, 'utf8')), out = {};
for (const f of ['json', 'slack', 'teams', 'discord', 'xapi']) out[f] = evs.map(e => JSON.parse(window.siWebhook.format(e, f)));
process.stdout.write(JSON.stringify(out));`
	cmd := exec.Command(node, "-e", script)
	cmd.Stdin = strings.NewReader(webhookEvents)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, stderr.String())
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestWebhookFormatsGolden: the payload of each event in each format, as
// webhook.js builds it, against testdata/webhook/<format>.json.
func TestWebhookFormatsGolden(t *testing.T) {
	for f, raw := range formatAll(t) {
		var pretty bytes.Buffer
		json.Indent(&pretty, raw, "", "  ")
		pretty.WriteString("\n")
		golden := filepath.Join("testdata", "webhook", f+".json")
		if *updateWebhook {
			os.MkdirAll(filepath.Dir(golden), 0o755)
			os.WriteFile(golden, pretty.Bytes(), 0o644)
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if pretty.String() != string(want) {
			t.Errorf("%s payloads changed (go test ./internal/assets -run TestWebhookFormatsGolden -update after an intended change):\n%s", f, pretty.String())
		}
	}
}

// TestWebhookJSONNeverHasAnswers: the json events carry scores and which
// questions were right, never what the learner picked.
func TestWebhookJSONNeverHasAnswers(t *testing.T) {
	out := formatAll(t)
	s := string(out["json"])
	for _, bad := range []string{"answer", "picked", "output"} {
		if strings.Contains(s, bad) {
			t.Errorf("json events mention %q: %s", bad, s)
		}
	}
	if !strings.Contains(string(out["slack"]), `"Ana passed the quiz “Check yourself” in A first web page: 80 % (attempt 2)."`) {
		t.Errorf("slack sentence: %s", out["slack"])
	}
	if !strings.Contains(string(out["slack"]), "(instructor build)") {
		t.Error("an instructor build's event says so in chat")
	}
}

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestWebhookXAPIStatements: each xapi payload is a valid xAPI 1.0.3
// statement: a UUID id, an Agent actor identified by an account, an ADL
// verb with a display, an Activity object with an IRI id, a timestamp, and a
// scaled score in [-1, 1] when there is one.
func TestWebhookXAPIStatements(t *testing.T) {
	var sts []map[string]any
	if err := json.Unmarshal(formatAll(t)["xapi"], &sts); err != nil {
		t.Fatal(err)
	}
	verbs := []string{"completed", "passed", "answered", "experienced", "completed"}
	for i, st := range sts {
		if id, _ := st["id"].(string); !uuidRE.MatchString(id) {
			t.Errorf("%d: id %q is not a UUID", i, id)
		}
		actor, _ := st["actor"].(map[string]any)
		acc, _ := actor["account"].(map[string]any)
		if actor["objectType"] != "Agent" || acc["homePage"] == nil || acc["name"] == nil {
			t.Errorf("%d: actor %v: want an Agent with account {homePage, name}", i, actor)
		}
		if _, ok := actor["mbox"]; ok {
			t.Errorf("%d: one identifier only (account)", i)
		}
		verb, _ := st["verb"].(map[string]any)
		if verb["id"] != "http://adlnet.gov/expapi/verbs/"+verbs[i] || verb["display"] == nil {
			t.Errorf("%d: verb %v, want ADL %s with a display", i, verb, verbs[i])
		}
		obj, _ := st["object"].(map[string]any)
		if id, _ := obj["id"].(string); obj["objectType"] != "Activity" || !strings.HasPrefix(id, "https://") {
			t.Errorf("%d: object %v: want an Activity with an IRI id", i, obj)
		}
		if ts, _ := st["timestamp"].(string); !strings.HasSuffix(ts, "Z") {
			t.Errorf("%d: timestamp %q", i, ts)
		}
		if res, ok := st["result"].(map[string]any); ok {
			if sc, ok := res["score"].(map[string]any); ok {
				if s, _ := sc["scaled"].(float64); s < -1 || s > 1 {
					t.Errorf("%d: score.scaled %v out of [-1, 1]", i, s)
				}
			}
		}
	}
}
