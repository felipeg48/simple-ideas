// Package publish sends decks and articles to LinkedIn, dev.to and Hashnode
// (si publish). Every request is built the same way for a dry run and a real
// send: a dry run prints it (secrets masked) and answers with a stand-in, so
// the steps that depend on an earlier answer still print. Nothing leaves the
// machine unless the caller passes a nil dry-run writer. Stdlib only.
package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Creds are the stored logins, in the user's config dir (never the repo).
type Creds struct {
	LinkedIn struct {
		Token   string `json:"token,omitempty"`
		Expires int64  `json:"expires,omitempty"` // unix seconds; LinkedIn tokens last 60 days
		Person  string `json:"person,omitempty"`  // urn:li:person:<id>
	} `json:"linkedin"`
	DevTo struct {
		Key string `json:"key,omitempty"`
	} `json:"devto"`
	Hashnode struct {
		Token       string `json:"token,omitempty"`
		Host        string `json:"host,omitempty"`        // blog.hashnode.dev
		Publication string `json:"publication,omitempty"` // its id, looked up once
	} `json:"hashnode"`
	Ghost struct {
		Site string `json:"site,omitempty"`
		Key  string `json:"key,omitempty"` // Admin API key, id:secret
	} `json:"ghost"`
	WordPress struct {
		Site     string `json:"site,omitempty"`
		User     string `json:"user,omitempty"`
		Password string `json:"password,omitempty"` // an application password
	} `json:"wordpress"`
}

// Path is the credentials file: $SI_PUBLISH_CONFIG, else <config dir>/si/publish.json.
func Path() (string, error) {
	if p := os.Getenv("SI_PUBLISH_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "si", "publish.json"), nil
}

// Load reads the stored logins; a missing file is no logins.
func Load() (Creds, error) {
	var c Creds
	p, err := Path()
	if err != nil {
		return c, err
	}
	raw, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

// Save writes the logins readable by the user only (file 0600, dir 0700).
func Save(c Creds) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(c, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

var client = &http.Client{Timeout: 60 * time.Second}

// secretHeaders are masked in a dry run.
var secretHeaders = map[string]bool{"Authorization": true, "Api-Key": true}

func mask(v string) string {
	if strings.Contains(v, "<not logged in>") {
		return v
	}
	tok := v
	if i := strings.LastIndexByte(v, ' '); i >= 0 {
		tok = v[i+1:]
	}
	if len(tok) <= 4 {
		return strings.TrimSuffix(v, tok) + "****"
	}
	return strings.TrimSuffix(v, tok) + "****" + tok[len(tok)-4:]
}

// do sends req (with body), or, in a dry run (dry != nil), prints it and
// returns fake as the answer. A status outside 2xx is an error with the body.
func do(ctx context.Context, req *http.Request, body []byte, dry io.Writer, fake string) ([]byte, http.Header, error) {
	if dry != nil {
		fmt.Fprintf(dry, "%s %s\n", req.Method, req.URL)
		keys := make([]string, 0, len(req.Header))
		for k := range req.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := req.Header.Get(k)
			if secretHeaders[k] {
				v = mask(v)
			}
			fmt.Fprintf(dry, "  %s: %s\n", k, v)
		}
		switch ct := req.Header.Get("Content-Type"); {
		case strings.Contains(ct, "json"):
			var pretty bytes.Buffer
			if json.Indent(&pretty, body, "  ", "  ") == nil {
				fmt.Fprintf(dry, "  %s\n", pretty.String())
			}
		case len(body) > 0:
			fmt.Fprintf(dry, "  <%d bytes of %s>\n", len(body), ct)
		case strings.Contains(ct, "pdf"):
			fmt.Fprintf(dry, "  <the deck as PDF>\n")
		}
		fmt.Fprintln(dry)
		h := http.Header{}
		h.Set("X-Restli-Id", "<post id>")
		return []byte(fake), h, nil
	}
	req = req.WithContext(ctx)
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		return out, resp.Header, fmt.Errorf("%s %s: %s %s", req.Method, req.URL.Host+req.URL.Path, resp.Status, msg)
	}
	return out, resp.Header, nil
}

func jsonReq(method, url string, v any) (*http.Request, []byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // text with < > & reads as written
	if err := enc.Encode(v); err != nil {
		return nil, nil, err
	}
	body := bytes.TrimSpace(buf.Bytes())
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, body, nil
}
