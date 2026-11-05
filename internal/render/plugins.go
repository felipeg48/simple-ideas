package render

import (
	"crypto/sha256"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
)

// Plugins: directives that work in every document (pages, articles, guides,
// books, labs, demos) and so in sites. Each is self-contained HTML plus a few
// lines in doc.js; nothing is fetched.
//
//	<!-- alert: Closed Monday 24 November · until=2026-11-25 · kind=warn -->
//	<!-- badge: Available from January 2027 · ok -->
//	<!-- accordion -->            the ### sections that follow, one open at a time
//	<!-- modal: reserve --> … <!-- end -->   any [link](#reserve) opens it
//	<!-- carousel: auto=6s -->     the image paragraphs that follow
//	<!-- event: 2026-11-20T18:00 Live tasting tonight · Book → #reserve -->
//	<!-- embed: … -->             see embed.go

// Plugins are the plugin directive names (documents only).
var Plugins = []string{"alert", "badge", "accordion", "modal", "end", "carousel", "event", "embed"}

// pluginKinds are the looks of an alert or a badge.
var pluginKinds = map[string]bool{"info": true, "ok": true, "warn": true, "crit": true}

// pluginKind returns the kind among opts and words, info by default; anything
// else is a problem (a stray "·" in the text usually).
func (dr *deckRenderer) pluginKind(sc *slideCtx, name string, opts map[string]string, words []string) string {
	kind := opts["kind"]
	if kind == "" && len(words) > 0 {
		kind = words[0]
	}
	if kind == "" {
		return "info"
	}
	if !pluginKinds[kind] {
		dr.problem(sc, "%s: %q is not one of info, ok, warn, crit (a · in the text starts an option)", name, kind)
		return "info"
	}
	return kind
}

func isPlugin(name string) bool {
	for _, p := range Plugins {
		if p == name {
			return true
		}
	}
	return false
}

// pluginParts splits "text · key=value · word" into the text and its options.
func pluginParts(val string) (string, map[string]string, []string) {
	parts := strings.Split(val, "·")
	opts, words := map[string]string{}, []string{}
	text := strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if k, v, ok := strings.Cut(p, "="); ok && !strings.Contains(k, " ") {
			opts[k] = v
		} else if p != "" {
			words = append(words, p)
		}
	}
	return text, opts, words
}

var eventAtRE = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2})?)\s+(.+)$`)

// plugins runs before a document part is rendered: it turns plugin
// directives (and the blocks they own) into HTML.
func (dr *deckRenderer) plugins(doc ast.Node, src []byte, s *deck.Slide, sc *slideCtx) {
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		hb, ok := c.(*ast.HTMLBlock)
		if !ok {
			c = next
			continue
		}
		name, val, isDir := deck.ParseDirective(blockText(hb, src))
		if !isDir || !isPlugin(name) {
			c = next
			continue
		}
		switch name {
		case "alert":
			text, opts, words := pluginParts(val)
			kind := dr.pluginKind(sc, name, opts, words)
			sum := sha256.Sum256([]byte(text))
			doc.ReplaceChild(doc, c, &rawNode{html: fmt.Sprintf(`<div class="si-alert" role="status" data-kind="%s" data-until="%s" data-id="%x"><p>%s</p><button type="button" class="si-alert-x" aria-label="Dismiss">×</button></div>`,
				html.EscapeString(kind), html.EscapeString(opts["until"]), sum[:6], dr.inline(text))})
		case "badge":
			text, opts, words := pluginParts(val)
			kind := dr.pluginKind(sc, name, opts, words)
			doc.ReplaceChild(doc, c, &rawNode{html: fmt.Sprintf(`<p class="si-pill-row"><span class="si-pill" data-kind="%s">%s</span></p>`, html.EscapeString(kind), dr.inline(text))})
		case "event":
			text, opts, _ := pluginParts(val)
			m := eventAtRE.FindStringSubmatch(text)
			if m == nil {
				dr.problem(sc, "event: wants a date and a text, e.g. <!-- event: 2026-11-20T18:00 Live tasting · Book → #reserve -->")
				doc.RemoveChild(doc, c)
				break
			}
			label, href := "", ""
			if _, rest, ok := strings.Cut(val, "·"); ok {
				l, h, ok := strings.Cut(rest, "→")
				if !ok {
					l, h, _ = strings.Cut(rest, "->")
				}
				label, href = strings.TrimSpace(l), strings.TrimSpace(h)
			}
			_ = opts
			doc.ReplaceChild(doc, c, &rawNode{html: fmt.Sprintf(`<template class="si-event" data-at="%s" data-text="%s" data-label="%s" data-href="%s"></template>`,
				html.EscapeString(strings.Replace(m[1], " ", "T", 1)), html.EscapeString(m[2]), html.EscapeString(label), html.EscapeString(href))})
		case "accordion":
			acc := &divNode{class: "si-accordion"}
			group := fmt.Sprintf("acc-%d", dr.stageN+len(dr.problems)+int(hb.Lines().At(0).Start))
			var cur *divNode
			k := next
			for k != nil {
				kn := k.NextSibling()
				if h, ok := k.(*ast.Heading); ok && h.Level <= 2 {
					break
				}
				if _, ok := k.(*ast.ThematicBreak); ok {
					break
				}
				if kh, ok := k.(*ast.HTMLBlock); ok { // the next directive ends the accordion
					if _, _, ok := deck.ParseDirective(blockText(kh, src)); ok {
						break
					}
				}
				if h, ok := k.(*ast.Heading); ok && h.Level == 3 {
					cur = &divNode{tag: "details", class: "si-acc", attrs: [][2]string{{"name", group}},
						open: "<summary>" + dr.inline(textOf(h, src)) + "</summary>\n"}
					acc.AppendChild(acc, cur)
					doc.RemoveChild(doc, k)
				} else if cur != nil {
					doc.RemoveChild(doc, k)
					cur.AppendChild(cur, k)
				} else {
					break
				}
				k = kn
			}
			if acc.ChildCount() == 0 {
				dr.problem(sc, "accordion: must be followed by ### questions")
				doc.RemoveChild(doc, c)
				break
			}
			doc.ReplaceChild(doc, c, acc)
			next = acc.NextSibling()
		case "modal":
			id := strings.TrimSpace(val)
			if id == "" {
				dr.problem(sc, "modal: needs an id, e.g. <!-- modal: reserve -->")
				doc.RemoveChild(doc, c)
				break
			}
			dlg := &divNode{tag: "dialog", class: "si-modal", attrs: [][2]string{{"id", id}},
				open: `<form method="dialog" class="si-modal-close"><button aria-label="Close">×</button></form>` + "\n"}
			closed := false
			k := next
			for k != nil {
				kn := k.NextSibling()
				if kh, ok := k.(*ast.HTMLBlock); ok {
					if n, _, ok := deck.ParseDirective(blockText(kh, src)); ok && n == "end" {
						doc.RemoveChild(doc, k)
						closed = true
						k = kn
						break
					}
				}
				doc.RemoveChild(doc, k)
				dlg.AppendChild(dlg, k)
				k = kn
			}
			if !closed {
				dr.problem(sc, "modal: %s has no <!-- end -->", id)
			}
			doc.ReplaceChild(doc, c, dlg)
			next = k
		case "end":
			dr.problem(sc, "end: without a <!-- modal: … --> before it")
			doc.RemoveChild(doc, c)
		case "carousel":
			_, opts, _ := pluginParts("·" + strings.ReplaceAll(val, " ", "·"))
			car := &divNode{class: "si-carousel", attrs: [][2]string{{"data-auto", opts["auto"]}, {"role", "region"}, {"aria-label", "Images"}}}
			track := &divNode{class: "si-carousel-track"}
			car.AppendChild(car, track)
			k := next
			for k != nil {
				kn := k.NextSibling()
				fig := imageFigure(k, src, nil)
				if fig == nil {
					break
				}
				doc.RemoveChild(doc, k)
				track.AppendChild(track, fig)
				k = kn
			}
			if track.ChildCount() == 0 {
				dr.problem(sc, "carousel: must be followed by images, one per paragraph")
				doc.RemoveChild(doc, c)
				break
			}
			doc.ReplaceChild(doc, c, car)
			next = k
		case "embed":
			doc.ReplaceChild(doc, c, &rawNode{html: dr.embed(val, s, sc)})
		}
		c = next
	}
}
