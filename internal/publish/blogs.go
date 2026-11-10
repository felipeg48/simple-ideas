package publish

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Ghost posts to a Ghost site through its Admin API with an Admin API key
// (id:secret, from Settings › Integrations › a custom integration).
type Ghost struct{ Site, Key string }

// ghostToken is the short-lived JWT the Admin API wants: HS256 with the key's
// hex secret, kid = its id, audience /admin/, five minutes.
func ghostToken(key string, now time.Time) (string, error) {
	id, secret, ok := strings.Cut(key, ":")
	raw, err := hex.DecodeString(secret)
	if !ok || err != nil || id == "" {
		return "", fmt.Errorf("ghost: the Admin API key is id:secret (Settings › Integrations)")
	}
	enc := base64.RawURLEncoding
	head, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": id})
	body, _ := json.Marshal(map[string]any{"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "aud": "/admin/"})
	signed := enc.EncodeToString(head) + "." + enc.EncodeToString(body)
	mac := hmac.New(sha256.New, raw)
	mac.Write([]byte(signed))
	return signed + "." + enc.EncodeToString(mac.Sum(nil)), nil
}

// CheckGhostKey reports a key that is not id:secret.
func CheckGhostKey(key string) error { _, err := ghostToken(key, time.Now()); return err }

// Publish creates the post from the article's HTML (a draft unless a.Public)
// and returns its URL.
func (g *Ghost) Publish(ctx context.Context, a Article, dry io.Writer) (string, error) {
	post := map[string]any{"title": a.Title, "html": a.HTML, "status": "draft"}
	if a.Public {
		post["status"] = "published"
	}
	if a.Description != "" {
		post["custom_excerpt"] = a.Description
	}
	if a.Canonical != "" {
		post["canonical_url"] = a.Canonical
	}
	if len(a.Tags) > 0 {
		post["tags"] = a.Tags
	}
	req, raw, err := jsonReq("POST", strings.TrimRight(g.Site, "/")+"/ghost/api/admin/posts/?source=html", map[string]any{"posts": []any{post}})
	if err != nil {
		return "", err
	}
	tok := "<not logged in>"
	if g.Key != "<not logged in>" {
		if tok, err = ghostToken(g.Key, time.Now()); err != nil {
			return "", err
		}
	}
	req.Header.Set("Authorization", "Ghost "+tok)
	req.Header.Set("Accept-Version", "v5.0")
	out, _, err := do(ctx, req, raw, dry, `{"posts":[{"url":"https://<your site>/<the new post>/"}]}`)
	if err != nil {
		return "", err
	}
	var r struct {
		Posts []struct {
			URL string `json:"url"`
		} `json:"posts"`
	}
	json.Unmarshal(out, &r)
	if len(r.Posts) == 0 {
		return "", fmt.Errorf("ghost: no post in the answer")
	}
	return r.Posts[0].URL, nil
}

// WordPress posts to a WordPress site through its REST API with a user name
// and an application password (Users › Profile › Application Passwords).
type WordPress struct{ Site, User, Password string }

// Publish creates the post from the article's HTML (a draft unless a.Public)
// and returns its URL. Tags are left out: WordPress wants tag ids.
func (w *WordPress) Publish(ctx context.Context, a Article, dry io.Writer) (string, error) {
	post := map[string]any{"title": a.Title, "content": a.HTML, "status": "draft"}
	if a.Public {
		post["status"] = "publish"
	}
	if a.Description != "" {
		post["excerpt"] = a.Description
	}
	req, raw, err := jsonReq("POST", strings.TrimRight(w.Site, "/")+"/wp-json/wp/v2/posts", post)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(w.User+":"+w.Password)))
	out, _, err := do(ctx, req, raw, dry, `{"link":"https://<your site>/?p=<the new post>"}`)
	if err != nil {
		return "", err
	}
	var r struct {
		Link string `json:"link"`
	}
	json.Unmarshal(out, &r)
	return r.Link, nil
}
