package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestPublishDryRun(t *testing.T) {
	t.Setenv("SI_PUBLISH_CONFIG", filepath.Join(t.TempDir(), "publish.json"))
	ex := filepath.Join("..", "..", "examples")
	var out, errb bytes.Buffer
	if code := run([]string{"publish", filepath.Join(ex, "showcase.md"), "--to", "linkedin"}, &out, &errb); code != 0 {
		t.Fatalf("deck dry run: %d %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{"Dry run", "POST https://api.linkedin.com/rest/documents?action=initializeUpload", "POST https://api.linkedin.com/rest/posts", `"title": "`, "urn:li:person:<you>"} {
		if !strings.Contains(s, want) {
			t.Errorf("deck dry run lacks %q", want)
		}
	}
	out.Reset()
	if code := run([]string{"publish", filepath.Join(ex, "article", "retry.md"), "--to", "devto", "--base-url", "https://blog.example.com/retry"}, &out, &errb); code != 0 {
		t.Fatalf("article dry run: %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "POST https://dev.to/api/articles") || !strings.Contains(out.String(), `"published": false`) {
		t.Errorf("devto dry run:\n%s", out.String())
	}
	out.Reset()
	if code := run([]string{"publish", filepath.Join(ex, "article", "retry.md"), "--to", "linkedin"}, &out, &errb); code != 0 || !strings.Contains(out.String(), `"source": "https://northwind.example/engineering/one-retry"`) ||
		!strings.Contains(out.String(), "/rest/images?action=initializeUpload") || !strings.Contains(out.String(), `"thumbnail": "urn:li:image:from-initializeUpload"`) {
		t.Errorf("an article with canonical: goes to LinkedIn as a link post: %d %s", code, out.String())
	}
	bare := filepath.Join(t.TempDir(), "a.md")
	os.WriteFile(bare, []byte("---\ntype: article\ntitle: A\n---\n# A\n\nText.\n"), 0o644)
	errb.Reset()
	if code := run([]string{"publish", bare, "--to", "linkedin"}, &out, &errb); code != exitUsage || !strings.Contains(errb.String(), "--url") {
		t.Errorf("an article to LinkedIn without --url: %d %s", code, errb.String())
	}
	errb.Reset()
	if code := run([]string{"publish", filepath.Join(ex, "article", "retry.md"), "--to", "devto", "--yes"}, &out, &errb); code != exitProblems || !strings.Contains(errb.String(), "si publish login devto") {
		t.Errorf("--yes without a login: %d %s", code, errb.String())
	}
}

func TestArticleFor(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("---\ntype: article\ntitle: Retry\ndescription: Why\ntags: go, sql\n---\n# Retry\n\nSee ![van](img/van.png) and [next](next.md) or [web](https://x.dev).\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	a, warns := articleFor(d, "https://me.dev/retry/", "https://me.dev/retry")
	if strings.Contains(a.Body, "# Retry") || !strings.Contains(a.Body, "![van](https://me.dev/retry/img/van.png)") || !strings.Contains(a.Body, "[next](https://me.dev/retry/next.html)") || !strings.Contains(a.Body, "(https://x.dev)") {
		t.Errorf("body:\n%s", a.Body)
	}
	if len(warns) != 0 || strings.Join(a.Tags, "|") != "go|sql" || a.Description != "Why" || a.Canonical != "https://me.dev/retry" {
		t.Errorf("%v %+v", warns, a)
	}
	if _, warns := articleFor(d, "", ""); len(warns) != 1 || !strings.Contains(warns[0], "img/van.png") {
		t.Errorf("relative image without --base-url should warn: %v", warns)
	}
}
