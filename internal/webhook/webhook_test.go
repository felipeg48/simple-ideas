package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSign: HMAC-SHA256 of "<timestamp>.<body>" (the RFC 4231 case 2 key
// and data after a timestamp), as GitHub-style sha256=hex; Verify takes the
// right body within the 5-minute window only.
func TestSign(t *testing.T) {
	body := []byte("what do ya want for nothing?")
	got := Sign("Jefe", "1700000000", body)
	want := "sha256=1cdd0650c8be1cb0974b1788d458b1e781206cfef59b85faafc582d2e182c57e"
	if got != want {
		t.Fatalf("Sign = %s, want %s", got, want)
	}
	at := time.Unix(1700000000, 0)
	if !Verify("Jefe", "1700000000", body, want, at.Add(4*time.Minute)) || Verify("Jefe", "1700000000", []byte("other"), want, at) {
		t.Error("Verify must accept the right body only")
	}
	if Verify("Jefe", "1700000000", body, want, at.Add(-6*time.Minute)) || Verify("Jefe", "soon", body, want, at) {
		t.Error("Verify must refuse a timestamp outside the window, or not a number")
	}
}

func TestSummaryBodies(t *testing.T) {
	s := Summary{Title: "docs/demos", Passed: 5, Failed: 1, Seconds: 42, Report: "demo-out/report.html", SI: "3.18.0", Time: "2026-11-14T10:00:00Z",
		Failures: []Failure{{Script: "studio.md", Feature: "Open a file", Step: `click "Open"`}}}
	want := map[string]string{
		"json":    `{"version":1,"event":"demo.test","title":"docs/demos","passed":5,"failed":1,"skipped":0,"seconds":42,"ok":false,"report":"demo-out/report.html","failures":[{"script":"studio.md","feature":"Open a file","step":"click \"Open\""}],"si":"3.18.0","time":"2026-11-14T10:00:00Z"}`,
		"slack":   `{"text":"si demo test docs/demos: 5 passed, 1 failed in 42.0 s. Failed: studio.md › Open a file › click \"Open\". Report: demo-out/report.html"}`,
		"discord": `{"content":"si demo test docs/demos: 5 passed, 1 failed in 42.0 s. Failed: studio.md › Open a file › click \"Open\". Report: demo-out/report.html"}`,
	}
	for f, w := range want {
		b, err := s.Body(f)
		if err != nil || string(b) != w {
			t.Errorf("%s:\n got %s (%v)\nwant %s", f, b, err, w)
		}
	}
	b, _ := s.Body("teams")
	var card struct {
		Type        string
		Attachments []struct {
			ContentType string
			Content     struct {
				Type string
				Body []struct{ Type, Text string }
			}
		}
	}
	if json.Unmarshal(b, &card) != nil || card.Type != "message" || len(card.Attachments) != 1 ||
		card.Attachments[0].ContentType != "application/vnd.microsoft.card.adaptive" || card.Attachments[0].Content.Type != "AdaptiveCard" ||
		!strings.HasPrefix(card.Attachments[0].Content.Body[0].Text, "si demo test docs/demos: 5 passed") {
		t.Errorf("teams: an Adaptive Card message with the sentence, got %s", b)
	}
	if _, err := s.Body("xapi"); err == nil {
		t.Error("a run summary has no xapi form")
	}
	if b, _ := (Summary{Title: "t"}).Body("json"); !strings.Contains(string(b), `"failures":[]`) {
		t.Errorf("no failures is an empty list, not null: %s", b)
	}
}

// TestPostSigns: Post sends JSON, the signature of the exact body, and the
// xAPI headers; a refusal is an error.
func TestPostSigns(t *testing.T) {
	var got *http.Request
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, body = r, must(io.ReadAll(r.Body))
		if r.URL.Path == "/no" {
			http.Error(w, "nope", http.StatusForbidden)
		}
	}))
	defer srv.Close()
	if err := Post(srv.URL+"/ok", "xapi", []byte(`{"a":1}`), "s3cret", "Basic dTpw"); err != nil {
		t.Fatal(err)
	}
	if got.Header.Get("Content-Type") != "application/json" || got.Header.Get("X-Experience-API-Version") != "1.0.3" ||
		got.Header.Get("Authorization") != "Basic dTpw" || !Verify("s3cret", got.Header.Get(TimestampHeader), body, got.Header.Get(SignatureHeader), time.Now()) {
		t.Errorf("headers %v", got.Header)
	}
	ts := got.Header.Get(TimestampHeader)
	if Verify("s3cret", ts, body, got.Header.Get(SignatureHeader), time.Now().Add(6*time.Minute)) {
		t.Error("a replay after the 5-minute window must fail")
	}
	if Verify("s3cret", "1"+ts, body, Sign("s3cret", ts, body), time.Now()) {
		t.Error("another timestamp, the same signature: must fail")
	}
	if err := Post(srv.URL+"/no", "json", []byte(`{}`), "", ""); err == nil {
		t.Error("a 403 is an error")
	}
	if got.Header.Get(SignatureHeader) != "" || got.Header.Get(TimestampHeader) != "" {
		t.Error("no secret, no signature")
	}
}

func must(b []byte, _ error) []byte { return b }
