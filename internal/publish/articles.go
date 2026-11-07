package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Article is a Markdown article for dev.to or Hashnode.
type Article struct {
	Title       string
	Body        string // Markdown, without front matter and title
	Description string
	Canonical   string
	Tags        []string
	Public      bool // dev.to: publish now (otherwise a draft)
}

// DevTo posts to dev.to (Forem API) with an API key.
type DevTo struct{ API, Key string }

// NewDevTo is a client for the real dev.to.
func NewDevTo(key string) *DevTo { return &DevTo{API: "https://dev.to/api", Key: key} }

// Publish creates the article (a draft unless a.Public) and returns its URL.
func (d *DevTo) Publish(ctx context.Context, a Article, dry io.Writer) (string, error) {
	tags := a.Tags
	if len(tags) > 4 { // dev.to takes up to 4
		tags = tags[:4]
	}
	art := map[string]any{"title": a.Title, "body_markdown": a.Body, "published": a.Public, "tags": strings.Join(tags, ",")}
	if a.Description != "" {
		art["description"] = a.Description
	}
	if a.Canonical != "" {
		art["canonical_url"] = a.Canonical
	}
	req, raw, err := jsonReq("POST", d.API+"/articles", map[string]any{"article": art})
	if err != nil {
		return "", err
	}
	req.Header.Set("Api-Key", d.Key)
	req.Header.Set("Accept", "application/vnd.forem.api-v1+json")
	out, _, err := do(ctx, req, raw, dry, `{"url":"https://dev.to/<you>/<the new article>"}`)
	if err != nil {
		return "", err
	}
	var r struct {
		URL string `json:"url"`
	}
	json.Unmarshal(out, &r)
	return r.URL, nil
}

// Hashnode posts to a Hashnode blog through its GraphQL API (a Pro plan on
// the publication is required since 2026) with a personal access token.
type Hashnode struct{ API, Token string }

// NewHashnode is a client for the real Hashnode.
func NewHashnode(token string) *Hashnode {
	return &Hashnode{API: "https://gql.hashnode.com", Token: token}
}

func (h *Hashnode) query(ctx context.Context, q string, vars map[string]any, dry io.Writer, fake string, out any) error {
	req, raw, err := jsonReq("POST", h.API, map[string]any{"query": q, "variables": vars})
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.Token)
	body, _, err := do(ctx, req, raw, dry, fake)
	if err != nil {
		return err
	}
	var r struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("hashnode: %v", err)
	}
	if len(r.Errors) > 0 {
		return fmt.Errorf("hashnode: %s", r.Errors[0].Message)
	}
	return json.Unmarshal(r.Data, out)
}

// Publication finds the publication id of a blog host (blog.hashnode.dev).
func (h *Hashnode) Publication(ctx context.Context, host string, dry io.Writer) (string, error) {
	var r struct {
		Publication *struct {
			ID string `json:"id"`
		} `json:"publication"`
	}
	err := h.query(ctx, `query Publication($host: String!) { publication(host: $host) { id } }`,
		map[string]any{"host": host}, dry, `{"data":{"publication":{"id":"<publication id>"}}}`, &r)
	if err != nil {
		return "", err
	}
	if r.Publication == nil {
		return "", fmt.Errorf("hashnode: no blog at %s", host)
	}
	return r.Publication.ID, nil
}

// Publish publishes the article to the publication and returns its URL.
func (h *Hashnode) Publish(ctx context.Context, publication string, a Article, dry io.Writer) (string, error) {
	tags := []map[string]string{}
	for _, t := range a.Tags {
		tags = append(tags, map[string]string{"slug": slug(t), "name": t})
	}
	in := map[string]any{"title": a.Title, "contentMarkdown": a.Body, "publicationId": publication, "tags": tags}
	if a.Description != "" {
		in["subtitle"] = a.Description
	}
	if a.Canonical != "" {
		in["originalArticleURL"] = a.Canonical
	}
	var r struct {
		PublishPost struct {
			Post struct {
				URL string `json:"url"`
			} `json:"post"`
		} `json:"publishPost"`
	}
	err := h.query(ctx, `mutation PublishPost($input: PublishPostInput!) { publishPost(input: $input) { post { id url } } }`,
		map[string]any{"input": in}, dry, `{"data":{"publishPost":{"post":{"url":"https://<your blog>/<the new post>"}}}}`, &r)
	return r.PublishPost.Post.URL, err
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
