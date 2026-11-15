package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
	"github.com/felipeg48/simple-ideas/internal/webhook"
)

// TestDemoTestWebhook: si demo test --webhook posts the run's summary
// (counts, report, the failed steps), signed with SI_WEBHOOK_SECRET; a
// refused post is a warning.
func TestDemoTestWebhook(t *testing.T) {
	t.Setenv(webhook.SecretEnv, "ci-secret")
	t.Setenv("SI_REPORT_URL", "https://ci.example.com/job/7/report.html")
	var got []byte
	var sig, ts string
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		sig, ts = r.Header.Get(webhook.SignatureHeader), r.Header.Get(webhook.TimestampHeader)
		if strings.HasSuffix(r.URL.Path, "/down") {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer hook.Close()
	suite := &webdemo.Suite{Title: "demos"}
	suite.Add(&webdemo.Report{Script: "studio.md", Passed: 2, Failed: 1, Seconds: 3.5, Features: []webdemo.FeatureResult{
		{Name: "Open a file", Status: "failed", Steps: []webdemo.StepResult{{Text: "go /", Status: "passed"}, {Text: `click "Open"`, Status: "failed"}}},
	}}, "studio")
	var stdout, stderr bytes.Buffer
	postDemoSummary(hook.URL, "json", demoSummary(suite, "demo-out/report.html"), &stdout, &stderr)
	var s webhook.Summary
	if err := json.Unmarshal(got, &s); err != nil {
		t.Fatalf("%v: %s", err, got)
	}
	if s.Event != "demo.test" || s.Version != 1 || s.Passed != 2 || s.Failed != 1 || s.OK || s.Report != "https://ci.example.com/job/7/report.html" ||
		len(s.Failures) != 1 || s.Failures[0].Step != `click "Open"` || s.Failures[0].Feature != "Open a file" {
		t.Errorf("summary: %+v", s)
	}
	if !webhook.Verify("ci-secret", ts, got, sig, time.Now()) {
		t.Error("signed with SI_WEBHOOK_SECRET")
	}
	stderr.Reset()
	postDemoSummary(hook.URL+"/down", "slack", demoSummary(suite, "r.html"), &stdout, &stderr)
	if !strings.Contains(stderr.String(), "warning: --webhook") {
		t.Errorf("a refused post warns: %q", stderr.String())
	}
	if code := run([]string{"demo", "test", "x.md", "--webhook", hook.URL, "--webhook-format", "xapi"}, &stdout, &stderr); code != exitUsage {
		t.Errorf("--webhook-format xapi is a usage error, got %d", code)
	}
}
