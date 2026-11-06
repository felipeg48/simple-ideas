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
	res := renderDoc(t, "<!-- modal: x -->\n\nText.\n\n<!-- end -->\n\n<!-- end -->\n\n<!-- accordion -->\n\nNo questions.\n\n<!-- event: soon Party -->\n\n<!-- badge: a · kind=nope -->\n")
	got := fmt.Sprint(res.Problems)
	for _, want := range []string{"end: without", "accordion: must be followed", "event: wants a date", `badge: kind="nope" is not one of`} {
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
	write("post.md", "---\ntype: article\ntitle: Retries\ndescription: Back off.\n---\n# Retries\n\n## Full jitter {#jitter}\n\nSpread them out.\n\n## Budgets\n\nCap them.\n")
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
	if strings.Contains(h, "{#jitter}") {
		t.Error("a custom id is not part of the heading text")
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

func TestBadgeKeepsItsDotAndWorksOnSlides(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "logo.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 4"><rect width="10" height="4"/></svg>`), 0o644)
	p := filepath.Join(dir, "d.md")
	os.WriteFile(p, []byte("# Cover\n\n<!-- logo: × CACI · alt=AceMQ -->\n\n<!-- badge: Prepared for CACI · Session 1 of 5 -->\n\n---\n\n## Two\n\n<!-- logo: logo.svg × CACI -->\n\n<!-- badge: Pipeline healthy · ok -->\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{
		`<p class="si-pill pill" data-kind="info"><span class="si-pill-mark pill-mark"></span>Prepared for CACI · Session 1 of 5</p>`,
		`<p class="si-pill pill" data-kind="ok"><span class="si-pill-mark pill-mark"></span>Pipeline healthy</p>`,
		`<div class="si-lockup lockup"><span class="si-lk-logo lk-logo" role="img" aria-label="AceMQ"></span><span class="si-lk-x lk-x"></span><span class="si-lk-name lk-name">CACI</span></div>`,
		`<img class="si-lk-img" src="data:image/svg+xml`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
}

func TestCardsOnSlidesAndInDocuments(t *testing.T) {
	body := "<!-- cards: 2 -->\n\n### High availability · In seconds · teal\n\nOne cluster.\n\n### Disaster recovery · orange\n\nTwo sites.\n\n## After\n"
	h := string(renderDoc(t, body).HTML)
	for _, want := range []string{
		`<div class="si-cards cards c2">`,
		`<div class="si-cell card teal">`, `<span class="si-card-tag tag teal">In seconds</span><h4>High availability</h4>`,
		`<div class="si-cell card orange">`, `<h4>Disaster recovery</h4>`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("document lacks %q", want)
		}
	}
	p := filepath.Join(t.TempDir(), "d.md")
	os.WriteFile(p, []byte("# T\n\n---\n\n## Two\n\n<!-- step -->\n<!-- cards -->\n\n### A\n\nx\n\n### B · hi\n\ny\n"), 0o644)
	d, _ := deck.Parse(p)
	res := Deck(d, Options{})
	if len(res.Problems) > 0 {
		t.Fatal(res.Problems)
	}
	if hs := string(res.HTML); !strings.Contains(hs, `<div class="si-cards cards si-step"`) && !strings.Contains(hs, `si-step`) || !strings.Contains(hs, `<div class="si-cell card hi">`) {
		t.Errorf("slide cards: %s", hs[strings.Index(hs, "si-cards")-80:strings.Index(hs, "si-cards")+200])
	}
}
