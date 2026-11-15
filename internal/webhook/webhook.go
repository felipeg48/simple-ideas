// Package webhook is what si sends to a team's webhook: the event names and
// formats a course may use (the page itself sends them, web/webhook.js), the
// HMAC signature the si server adds, and the posts the server makes itself
// (the relay, si demo test --webhook). Standard library only.
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Events are the course events, in the order the docs list them.
var Events = []string{"lesson.completed", "quiz.submitted", "lab.step.checked", "demo.watched", "course.completed"}

// Formats are the payload shapes a course may send.
var Formats = []string{"json", "slack", "teams", "discord", "xapi"}

// ChatFormats are the formats a run summary (si demo test --webhook) takes.
var ChatFormats = []string{"json", "slack", "teams", "discord"}

// SignatureHeader carries the HMAC of the timestamp and body when a secret is set.
const SignatureHeader = "X-Si-Signature"

// TimestampHeader carries when the post was signed, in Unix seconds. The
// signature covers "<timestamp>.<body>", so a receiver that refuses a
// timestamp more than Window away from its clock refuses replays.
const TimestampHeader = "X-Si-Timestamp"

// Window is how far a signed post's timestamp may be from the receiver's clock.
const Window = 5 * time.Minute

// SecretEnv overrides webhook-secret in the user's si.conf.
const SecretEnv = "SI_WEBHOOK_SECRET"

// Sign is the signature header value of a post signed at ts (Unix seconds,
// as sent in TimestampHeader): "sha256=" and the hex HMAC-SHA256 of
// "<ts>.<body>".
func Sign(secret, ts string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(ts + "."))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

// Verify reports whether sig is the signature of body sent at ts under
// secret, and ts is within Window of now (for receivers written in Go, and
// the tests).
func Verify(secret, ts string, body []byte, sig string, now time.Time) bool {
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	if d := now.Sub(time.Unix(n, 0)); d > Window || d < -Window {
		return false
	}
	return hmac.Equal([]byte(Sign(secret, ts, body)), []byte(sig))
}

// Post sends body to url, signed when secret is set; auth (xAPI) becomes the
// Authorization header. A status of 300 or more is an error.
func Post(url, format string, body []byte, secret, auth string) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json") // a server posts JSON as JSON (a browser sends json and slack as text/plain, no preflight)
	req.Header.Set("User-Agent", "si-webhook")
	if format == "xapi" {
		req.Header.Set("X-Experience-API-Version", "1.0.3")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set(TimestampHeader, ts)
		req.Header.Set(SignatureHeader, Sign(secret, ts, body))
	}
	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s answered %s", url, res.Status)
	}
	return nil
}

// Secret is the signing secret: SI_WEBHOOK_SECRET, else conf (webhook-secret
// in the user's si.conf; the caller passes it).
func Secret(conf string) string {
	if s := os.Getenv(SecretEnv); s != "" {
		return s
	}
	return conf
}

// Chat wraps a sentence in a chat format's payload (slack, teams, discord).
func Chat(format, text string) []byte {
	var v any
	switch format {
	case "slack":
		v = map[string]any{"text": text}
	case "discord":
		if r := []rune(text); len(r) > 2000 { // Discord's limit
			text = string(r[:1999]) + "…"
		}
		v = map[string]any{"content": text}
	case "teams":
		v = map[string]any{"type": "message", "attachments": []any{map[string]any{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"content": map[string]any{"$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "type": "AdaptiveCard", "version": "1.4",
				"body": []any{map[string]any{"type": "TextBlock", "text": text, "wrap": true}}},
		}}}
	default:
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}
