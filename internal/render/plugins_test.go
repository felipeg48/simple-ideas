package render

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestPlugins(t *testing.T) {
	res := renderDoc(t, `<!-- alert: Closed **Monday** · until=2026-11-25 · warn -->

<!-- badge: Open from January · ok -->

<!-- accordion -->

### Do you take bookings?

Yes, up to six people.

### Is there parking?

On the street.

<!-- modal: reserve -->

### Reserve

Call us.

<!-- end -->

[Book](#reserve)

<!-- carousel: auto=6s -->

![Room](data:image/gif;base64,R0lGODlhAQABAAAAACw=)

![Bar](data:image/gif;base64,R0lGODlhAQABAAAAACw=)

<!-- event: 2026-11-20T18:00 Live tasting · Book → #reserve -->
`)
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{
		`<div class="si-alert" role="status" data-kind="warn" data-until="2026-11-25"`,
		`<strong>Monday</strong>`,
		`<span class="si-pill" data-kind="ok">Open from January</span>`,
		`<details class="si-acc"`,
		`<summary>Is there parking?</summary>`,
		`<dialog class="si-modal" id="reserve"`,
		`<div class="si-carousel" data-auto="6s"`,
		`<template class="si-event" data-at="2026-11-20T18:00" data-text="Live tasting" data-label="Book" data-href="#reserve">`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	if strings.Count(h, `class="si-acc"`) != 2 {
		t.Errorf("want 2 accordion items")
	}
	if strings.Count(h, `<figure`) < 2 {
		t.Errorf("carousel lost its images")
	}
}

func TestPluginProblems(t *testing.T) {
	res := renderDoc(t, "<!-- modal: x -->\n\nText.\n\n<!-- end -->\n\n<!-- end -->\n\n<!-- accordion -->\n\nNo questions.\n\n<!-- event: soon Party -->\n\n<!-- badge: a · b -->\n")
	got := fmt.Sprint(res.Problems)
	for _, want := range []string{"end: without", "accordion: must be followed", "event: wants a date", `badge: "b" is not one of`} {
		if !strings.Contains(got, want) {
			t.Errorf("problems lack %q: %s", want, got)
		}
	}
}

func TestEmbed(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("post.md", "---\ntype: article\ntitle: Retries\ndescription: Back off.\n---\n# Retries\n\n## Jitter\n\nSpread them out.\n\n## Budgets\n\nCap them.\n")
	write("talk.md", "---\ntitle: Talk\n---\n# Talk\n\n---\n\n## Two\n")
	write("d.md", "---\ntype: article\n---\n# D\n\n<!-- embed: post.md -->\n\n<!-- embed: post.md#jitter -->\n\n<!-- embed: talk.md -->\n")
	d, err := deck.Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{`class="si-embed-card"`, `Back off.`, `class="si-embed-doc"`, `Spread them out.`, `srcdoc=`} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	if strings.Contains(h[strings.Index(h, `si-embed-doc`):], "Cap them.") {
		t.Errorf("section embed ran past its section")
	}
}

func TestPluginsOnSlideAreProblems(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.md")
	os.WriteFile(p, []byte("# S\n\n<!-- alert: hi -->\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if res := Deck(d, Options{}); !strings.Contains(fmt.Sprint(res.Problems), "not on a slide") {
		t.Errorf("problems: %v", res.Problems)
	}
}
